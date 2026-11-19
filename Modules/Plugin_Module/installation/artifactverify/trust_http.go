package artifactverify

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const maxTrustConfigurationBytes = 1 << 20

var (
	ErrInvalidTrustEndpoint = errors.New("invalid trust configuration endpoint")
	ErrTrustTransport       = errors.New("trust configuration transport failed")
)

// HTTPTrustConfigurationSource retrieves a bounded trust document from a
// distribution-configured HTTPS endpoint. It deliberately does not define a
// Marketplace URL or authentication protocol.
type HTTPTrustConfigurationSource struct {
	endpoint *url.URL
	client   *http.Client
}

// HTTPTrustConfigurationSourceConfig is supplied by the PRIDE distribution.
// EndpointURL must be an HTTPS URL and its hostname must appear exactly in
// AllowedHosts. Wildcards and IP-literal hosts are not supported.
type HTTPTrustConfigurationSourceConfig struct {
	EndpointURL string
	AllowedHosts []string
	Timeout time.Duration
}

// NewHTTPTrustConfigurationSource constructs a fail-closed HTTPS source.
// Redirects and environment proxies are disabled; DNS is resolved at dial time
// and every candidate address must be globally routable.
func NewHTTPTrustConfigurationSource(config HTTPTrustConfigurationSourceConfig) (*HTTPTrustConfigurationSource, error) {
	endpoint, err := url.Parse(config.EndpointURL)
	if err != nil || endpoint.Scheme != "https" || endpoint.Hostname() == "" ||
		endpoint.User != nil || endpoint.Fragment != "" ||
		(endpoint.Port() != "" && endpoint.Port() != "443") {
		return nil, ErrInvalidTrustEndpoint
	}
	host := normalizeTrustHost(endpoint.Hostname())
	if net.ParseIP(host) != nil {
		return nil, ErrInvalidTrustEndpoint
	}
	allowed := false
	for _, candidate := range config.AllowedHosts {
		normalized := normalizeTrustHost(candidate)
		if normalized == "" || strings.ContainsAny(normalized, "*/:@?#") || net.ParseIP(normalized) != nil {
			return nil, ErrInvalidTrustEndpoint
		}
		if normalized == host {
			allowed = true
		}
	}
	if !allowed {
		return nil, ErrInvalidTrustEndpoint
	}
	if config.Timeout == 0 {
		config.Timeout = 15 * time.Second
	}
	if config.Timeout < time.Second || config.Timeout > time.Minute {
		return nil, ErrInvalidTrustEndpoint
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy: nil,
		TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		DisableCompression: true,
		MaxResponseHeaderBytes: 64 << 10,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialHost, port, err := net.SplitHostPort(address)
			if err != nil || port != "443" || normalizeTrustHost(dialHost) != host {
				return nil, ErrInvalidTrustEndpoint
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("%w: DNS lookup: %v", ErrTrustTransport, err)
			}
			for _, candidate := range ips {
				if !isPublicTrustAddress(candidate.IP) {
					continue
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			}
			return nil, ErrInvalidTrustEndpoint
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout: config.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &HTTPTrustConfigurationSource{endpoint: endpoint, client: client}, nil
}

// ReadTrustConfiguration fetches one complete JSON document. The response is
// still untrusted until TrustResolver verifies its bootstrap signature.
func (s *HTTPTrustConfigurationSource) ReadTrustConfiguration(ctx context.Context) ([]byte, error) {
	if s == nil || s.endpoint == nil || s.client == nil {
		return nil, ErrInvalidTrustEndpoint
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, s.endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: create request: %v", ErrTrustTransport, err)
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "identity")
	response, err := s.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: request: %v", ErrTrustTransport, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: unexpected HTTP status %d", ErrTrustTransport, response.StatusCode)
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return nil, ErrTrustTransport
	}
	contentType := strings.ToLower(strings.TrimSpace(strings.Split(response.Header.Get("Content-Type"), ";")[0]))
	if contentType != "application/json" {
		return nil, ErrTrustTransport
	}
	if response.ContentLength > maxTrustConfigurationBytes {
		return nil, ErrTrustTransport
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxTrustConfigurationBytes+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read response: %v", ErrTrustTransport, err)
	}
	if len(body) == 0 || len(body) > maxTrustConfigurationBytes {
		return nil, ErrTrustTransport
	}
	return body, nil
}

func normalizeTrustHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func isPublicTrustAddress(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 ||
			v4[0] == 192 && v4[1] == 0 && (v4[2] == 0 || v4[2] == 2) ||
			v4[0] == 192 && v4[1] == 88 && v4[2] == 99 ||
			v4[0] == 198 && (v4[1] == 18 || v4[1] == 19 || v4[1] == 51 && v4[2] == 100) ||
			v4[0] == 203 && v4[1] == 0 && v4[2] == 113 || v4[0] >= 240 {
			return false
		}
	} else if strings.HasPrefix(strings.ToLower(ip.String()), "2001:db8:") {
		return false
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() && !ip.IsLoopback() &&
		!ip.IsLinkLocalUnicast() && !ip.IsLinkLocalMulticast() &&
		!ip.IsMulticast() && !ip.IsUnspecified()
}
