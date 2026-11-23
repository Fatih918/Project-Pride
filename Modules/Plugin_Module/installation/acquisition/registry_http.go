package acquisition

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

)

const (
	maxRegistryDescriptorBytes = 1 << 20
	defaultRegistryTimeout     = 15 * time.Second
	maxRegistryRetryAfter      = time.Hour
)

var (
	ErrRegistryConfiguration = errors.New("invalid Plugin Registry configuration")
	ErrRegistryAuthentication = errors.New("Plugin Registry authentication failed")
	ErrRegistryAuthorization  = errors.New("Plugin Registry authorization denied")
	ErrRegistryNotFound       = errors.New("Plugin Registry release not found")
	ErrRegistryConflict       = errors.New("Plugin Registry release conflict")
	ErrRegistryRateLimited    = errors.New("Plugin Registry rate limited")
	ErrRegistryUnavailable    = errors.New("Plugin Registry temporarily unavailable")
	ErrRegistryResponse       = errors.New("invalid Plugin Registry response")
	ErrRegistryIdentity       = errors.New("Plugin Registry descriptor identity mismatch")
)

// BearerToken is a short-lived token issued by the host for one Registry
// namespace and audience. The Module never refreshes or persists this token.
type BearerToken struct {
	Value     string
	ExpiresAt time.Time
}

// RegistryTokenProvider is implemented by the PRIDE host. Audience is the
// exact configured Registry API origin, not an artifact download origin.
type RegistryTokenProvider interface {
	Token(context.Context, string, string) (BearerToken, error)
}

// DescriptorResolver resolves a typed immutable release request into the
// exact signed descriptor bytes returned by the Registry.
type DescriptorResolver interface {
	ResolveDescriptor(context.Context, DescriptorResolutionRequest) ([]byte, error)
}

// RegistryResolverConfig is trusted host configuration. Every namespace maps
// to one HTTPS API origin; callers cannot select or override that origin.
type RegistryResolverConfig struct {
	Origins map[string]string
	Timeout time.Duration
}

// RegistryHTTPError contains only a status code, a bounded retry hint, and a
// safe request ID. Response bodies and authentication material are discarded.
type RegistryHTTPError struct {
	StatusCode int
	RetryAfter time.Duration
	RequestID  string
}

func (e *RegistryHTTPError) Error() string {
	switch e.StatusCode {
	case http.StatusBadRequest:
		return ErrRegistryResponse.Error()
	case http.StatusUnauthorized:
		return ErrRegistryAuthentication.Error()
	case http.StatusForbidden:
		return ErrRegistryAuthorization.Error()
	case http.StatusNotFound:
		return ErrRegistryNotFound.Error()
	case http.StatusConflict:
		return ErrRegistryConflict.Error()
	case http.StatusTooManyRequests:
		return ErrRegistryRateLimited.Error()
	default:
		if e.StatusCode >= 500 {
			return ErrRegistryUnavailable.Error()
		}
		return ErrRegistryResponse.Error()
	}
}

func (e *RegistryHTTPError) Is(target error) bool {
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return target == ErrRegistryAuthentication
	case http.StatusForbidden:
		return target == ErrRegistryAuthorization
	case http.StatusNotFound:
		return target == ErrRegistryNotFound
	case http.StatusConflict:
		return target == ErrRegistryConflict
	case http.StatusTooManyRequests:
		return target == ErrRegistryRateLimited
	default:
		if e.StatusCode >= 500 {
			return target == ErrRegistryUnavailable
		}
		return target == ErrRegistryResponse
	}
}

type HTTPRegistryDescriptorResolver struct {
	origins map[string]*url.URL
	tokens  RegistryTokenProvider
	client  *http.Client
	now     func() time.Time
}

// NewHTTPRegistryDescriptorResolver creates a hardened Registry API client.
// It disables redirects and proxies, pins requests to configured origins,
// validates every DNS answer, and bounds time, headers, and response bytes.
func NewHTTPRegistryDescriptorResolver(config RegistryResolverConfig, tokens RegistryTokenProvider) (*HTTPRegistryDescriptorResolver, error) {
	if tokens == nil || len(config.Origins) == 0 {
		return nil, ErrRegistryConfiguration
	}
	if config.Timeout == 0 {
		config.Timeout = defaultRegistryTimeout
	}
	if config.Timeout < time.Second || config.Timeout > time.Minute {
		return nil, ErrRegistryConfiguration
	}

	origins := make(map[string]*url.URL, len(config.Origins))
	for namespace, raw := range config.Origins {
		if !validNamespace(namespace) {
			return nil, ErrRegistryConfiguration
		}
		origin, err := parseRegistryOrigin(raw)
		if err != nil {
			return nil, ErrRegistryConfiguration
		}
		origins[namespace] = origin
	}

	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                 nil,
		DisableCompression:    true,
		MaxResponseHeaderBytes: 64 << 10,
		TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil || port != "443" {
				return nil, ErrRegistryConfiguration
			}
			allowed := false
			for _, origin := range origins {
				if normalizeHost(host) == normalizeHost(origin.Hostname()) {
					allowed = true
					break
				}
			}
			if !allowed || net.ParseIP(host) != nil {
				return nil, ErrRegistryConfiguration
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil || len(ips) == 0 {
				return nil, ErrRegistryUnavailable
			}
			// Reject the entire answer set if any address is non-public. This
			// prevents selecting a public answer from a mixed public/private set.
			for _, candidate := range ips {
				if !isPublicUnicast(candidate.IP) {
					return nil, ErrRegistryConfiguration
				}
			}
			return dialer.DialContext(ctx, network, net.JoinHostPort(ips[0].IP.String(), port))
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &HTTPRegistryDescriptorResolver{origins: origins, tokens: tokens, client: client, now: func() time.Time { return time.Now().UTC() }}, nil
}

func parseRegistryOrigin(raw string) (*url.URL, error) {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() == "" ||
		parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" ||
		(parsed.Path != "" && parsed.Path != "/") ||
		(parsed.Port() != "" && parsed.Port() != "443") {
		return nil, ErrRegistryConfiguration
	}
	host := normalizeHost(parsed.Hostname())
	if host == "" || net.ParseIP(host) != nil {
		return nil, ErrRegistryConfiguration
	}
	return &url.URL{Scheme: "https", Host: host}, nil
}

func (r *HTTPRegistryDescriptorResolver) ResolveDescriptor(ctx context.Context, resolution DescriptorResolutionRequest) ([]byte, error) {
	if r == nil || r.client == nil || r.tokens == nil ||
		!validNamespace(resolution.Reference.Namespace) ||
		!validPluginID(string(resolution.Reference.PluginID)) ||
		!validReleaseID(resolution.Reference.ReleaseID) ||
		resolution.ExpectedPluginID != resolution.Reference.PluginID ||
		!validVersion(resolution.ExpectedVersion) {
		return nil, ErrRegistryConfiguration
	}
	origin, ok := r.origins[resolution.Reference.Namespace]
	if !ok {
		return nil, ErrRegistryConfiguration
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	token, err := r.tokens.Token(ctx, resolution.Reference.Namespace, origin.String())
	if err != nil {
		return nil, ErrRegistryAuthentication
	}
	now := r.now()
	if !validBearerToken(token.Value) || !token.ExpiresAt.After(now) {
		return nil, ErrRegistryAuthentication
	}

	endpoint := *origin
	endpoint.Path = "/v1/plugins/" + url.PathEscape(string(resolution.Reference.PluginID)) +
		"/releases/" + url.PathEscape(resolution.Reference.ReleaseID) + "/descriptor"
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, ErrRegistryConfiguration
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-Encoding", "identity")
	request.Header.Set("Authorization", "Bearer "+token.Value)

	response, err := r.client.Do(request)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrRegistryUnavailable
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, registryStatusError(response)
	}
	if response.ContentLength > maxRegistryDescriptorBytes ||
		!identityEncoding(response.Header.Get("Content-Encoding")) ||
		!isJSONContentType(response.Header.Get("Content-Type")) {
		return nil, ErrRegistryResponse
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, maxRegistryDescriptorBytes+1))
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrRegistryUnavailable
	}
	if len(body) == 0 || len(body) > maxRegistryDescriptorBytes {
		return nil, ErrRegistryResponse
	}
	if err := rejectDuplicateJSONKeys(body); err != nil {
		return nil, ErrRegistryResponse
	}

	var descriptor struct {
		Schema        string `json:"$schema"`
		SchemaVersion string `json:"schema_version"`
		PluginID      string `json:"plugin_id"`
		Version       string `json:"version"`
		ReleaseID     string `json:"release_id"`
	}
	if err := json.Unmarshal(body, &descriptor); err != nil ||
		descriptor.Schema != "urn:pride:registry-artifact-descriptor:1" ||
		descriptor.SchemaVersion != "1.0.0" ||
		descriptor.PluginID != string(resolution.ExpectedPluginID) ||
		descriptor.Version != resolution.ExpectedVersion ||
		descriptor.ReleaseID != resolution.Reference.ReleaseID {
		return nil, ErrRegistryIdentity
	}
	return append([]byte(nil), body...), nil
}

func registryStatusError(response *http.Response) error {
	err := &RegistryHTTPError{StatusCode: response.StatusCode}
	if response.StatusCode == http.StatusTooManyRequests {
		err.RetryAfter = boundedRetryAfter(response.Header.Get("Retry-After"), time.Now())
	}
	requestID := response.Header.Get("X-Request-ID")
	if requestID == "" {
		requestID = response.Header.Get("Request-ID")
	}
	if len(requestID) <= 128 && isSafeRequestID(requestID) {
		err.RequestID = requestID
	}
	return err
}

func boundedRetryAfter(value string, now time.Time) time.Duration {
	value = strings.TrimSpace(value)
	if seconds, err := strconv.Atoi(value); err == nil && seconds >= 0 {
		if time.Duration(seconds) > maxRegistryRetryAfter/time.Second {
			return maxRegistryRetryAfter
		}
		return time.Duration(seconds) * time.Second
	}
	if at, err := http.ParseTime(value); err == nil && at.After(now) {
		delay := at.Sub(now)
		if delay > maxRegistryRetryAfter {
			return maxRegistryRetryAfter
		}
		return delay
	}
	return 0
}

func validBearerToken(value string) bool {
	if value == "" || len(value) > 8192 {
		return false
	}
	for _, r := range value {
		if r <= 0x20 || r >= 0x7f {
			return false
		}
	}
	return true
}

func identityEncoding(value string) bool {
	return value == "" || strings.EqualFold(strings.TrimSpace(value), "identity")
}

func isJSONContentType(value string) bool {
	mediaType := strings.ToLower(strings.TrimSpace(strings.Split(value, ";")[0]))
	return mediaType == "application/json"
}

func isSafeRequestID(value string) bool {
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' ||
			r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.') {
			return false
		}
	}
	return true
}

