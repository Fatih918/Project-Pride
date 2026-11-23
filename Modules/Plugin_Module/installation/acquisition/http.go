package acquisition

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

const (
	maxDescriptorBytes = 1 << 20
	maxArchiveBytes    = 100 << 20
	defaultTimeout     = 30 * time.Second
)

var (
	ErrInvalidDescriptor = errors.New("invalid Registry artifact descriptor for acquisition")
	ErrUnsafeArtifactURL = errors.New("artifact URL is not permitted by acquisition policy")
	ErrArtifactTooLarge  = errors.New("artifact exceeds configured compressed-size limit")
	ErrArtifactSize      = errors.New("downloaded artifact size does not match descriptor")
	ErrUnexpectedStatus  = errors.New("artifact host returned an unexpected HTTP status")
)

// ArtifactFileStore exposes only the confined artifact-file operations needed
// by acquisition. Implementations must create exclusively below stagingPath.
type ArtifactFileStore interface {
	CreateArtifactFile(context.Context, string) (*os.File, error)
	RemoveArtifactFile(context.Context, string) error
}

type Config struct {
	// AllowedHosts is an exact hostname allowlist. Wildcards and IP literals
	// are intentionally unsupported.
	AllowedHosts []string
	// MaxArchiveBytes must be positive and cannot exceed PZIP-0001's 100 MiB.
	MaxArchiveBytes int64
	Timeout         time.Duration
}

type HTTPArtifactAcquirer struct {
	resolver DescriptorResolver
	files    ArtifactFileStore
	client   *http.Client
	hosts    map[string]struct{}
	maxBytes int64
}

type artifactDescriptor struct {
	Schema        string `json:"$schema"`
	SchemaVersion string `json:"schema_version"`
	PluginID      string `json:"plugin_id"`
	Version       string `json:"version"`
	ReleaseID     string `json:"release_id"`
	DownloadURI   string `json:"download_uri"`
	MediaType     string `json:"media_type"`
	SizeBytes     uint64 `json:"size_bytes"`
}

// NewHTTPArtifactAcquirer creates a production HTTP downloader. It disables
// proxies and redirects, requires exact allowed hostnames, resolves DNS at
// connection time, rejects non-public IP addresses, and enforces TLS.
func NewHTTPArtifactAcquirer(resolver DescriptorResolver, files ArtifactFileStore, config Config) (*HTTPArtifactAcquirer, error) {
	if resolver == nil || files == nil {
		return nil, errors.New("descriptor resolver and artifact file store are required")
	}
	if config.MaxArchiveBytes == 0 {
		config.MaxArchiveBytes = maxArchiveBytes
	}
	if config.MaxArchiveBytes < 1 || config.MaxArchiveBytes > maxArchiveBytes {
		return nil, fmt.Errorf("MaxArchiveBytes must be between 1 and %d", maxArchiveBytes)
	}
	if config.Timeout == 0 {
		config.Timeout = defaultTimeout
	}
	if config.Timeout < time.Second || config.Timeout > 2*time.Minute {
		return nil, errors.New("HTTP acquisition timeout must be between 1 second and 2 minutes")
	}
	hosts, err := normalizeAllowedHosts(config.AllowedHosts)
	if err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: 10 * time.Second, KeepAlive: 30 * time.Second}
	transport := &http.Transport{
		Proxy:                  nil,
		TLSClientConfig:        &tls.Config{MinVersion: tls.VersionTLS12},
		DisableCompression:     true,
		MaxResponseHeaderBytes: 1 << 20,
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil || port != "443" {
				return nil, ErrUnsafeArtifactURL
			}
			host = normalizeHost(host)
			if _, ok := hosts[host]; !ok || net.ParseIP(host) != nil {
				return nil, ErrUnsafeArtifactURL
			}
			ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
			if err != nil {
				return nil, fmt.Errorf("resolve artifact host: %w", err)
			}
			for _, candidate := range ips {
				if !isPublicUnicast(candidate.IP) {
					continue
				}
				return dialer.DialContext(ctx, network, net.JoinHostPort(candidate.IP.String(), port))
			}
			return nil, ErrUnsafeArtifactURL
		},
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   config.Timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return newHTTPArtifactAcquirer(resolver, files, hosts, config.MaxArchiveBytes, client), nil
}

func newHTTPArtifactAcquirer(resolver DescriptorResolver, files ArtifactFileStore, hosts map[string]struct{}, maxBytes int64, client *http.Client) *HTTPArtifactAcquirer {
	return &HTTPArtifactAcquirer{resolver: resolver, files: files, hosts: hosts, maxBytes: maxBytes, client: client}
}

// Acquire resolves the source reference, downloads exactly the descriptor's
// declared ZIP size, and stores both descriptor and bytes in the operation
// workspace. Signature and digest verification remain the verifier's job.
func (a *HTTPArtifactAcquirer) Acquire(ctx context.Context, sourceReference string, expectedPluginID domain.PluginID, expectedVersion, stagingPath string) (installation.StagedArtifact, error) {
	if a == nil || a.resolver == nil || a.files == nil || a.client == nil ||
		strings.TrimSpace(sourceReference) == "" || strings.TrimSpace(string(expectedPluginID)) == "" || strings.TrimSpace(expectedVersion) == "" || strings.TrimSpace(stagingPath) == "" {
		return installation.StagedArtifact{}, ErrInvalidDescriptor
	}
	if err := ctx.Err(); err != nil {
		return installation.StagedArtifact{}, err
	}
	reference, err := ParseRegistryReleaseReference(sourceReference)
	if err != nil {
		return installation.StagedArtifact{}, err
	}
	resolution, err := NewDescriptorResolutionRequest(reference, expectedPluginID, expectedVersion)
	if err != nil {
		return installation.StagedArtifact{}, err
	}
	descriptorJSON, err := a.resolver.ResolveDescriptor(ctx, resolution)
	if err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("resolve Marketplace artifact descriptor: %w", err)
	}
	if len(descriptorJSON) == 0 || len(descriptorJSON) > maxDescriptorBytes {
		return installation.StagedArtifact{}, ErrInvalidDescriptor
	}
	if err := rejectDuplicateJSONKeys(descriptorJSON); err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	var descriptor artifactDescriptor
	if err := json.Unmarshal(descriptorJSON, &descriptor); err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	if descriptor.PluginID != string(expectedPluginID) || descriptor.Version != expectedVersion || descriptor.ReleaseID != reference.ReleaseID {
		return installation.StagedArtifact{}, ErrRegistryIdentity
	}
	artifactURL, err := a.validateDescriptor(descriptor)
	if err != nil {
		return installation.StagedArtifact{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, artifactURL.String(), nil)
	if err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("create artifact request: %w", err)
	}
	request.Header.Set("Accept", "application/vnd.pride.plugin+zip")
	request.Header.Set("Accept-Encoding", "identity")
	response, err := a.client.Do(request)
	if err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("download Plugin artifact: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return installation.StagedArtifact{}, fmt.Errorf("%w: %d", ErrUnexpectedStatus, response.StatusCode)
	}
	if encoding := response.Header.Get("Content-Encoding"); encoding != "" && encoding != "identity" {
		return installation.StagedArtifact{}, ErrInvalidDescriptor
	}
	if response.ContentLength >= 0 && response.ContentLength != int64(descriptor.SizeBytes) {
		return installation.StagedArtifact{}, ErrArtifactSize
	}

	file, err := a.files.CreateArtifactFile(ctx, stagingPath)
	if err != nil {
		return installation.StagedArtifact{}, fmt.Errorf("create staged artifact file: %w", err)
	}
	removePartial := func(cause error) (installation.StagedArtifact, error) {
		closeErr := file.Close()
		removeErr := a.files.RemoveArtifactFile(context.WithoutCancel(ctx), stagingPath)
		return installation.StagedArtifact{}, errors.Join(cause, closeErr, removeErr)
	}

	written, copyErr := io.Copy(file, io.LimitReader(&contextReader{ctx: ctx, reader: response.Body}, a.maxBytes+1))
	if copyErr != nil {
		return removePartial(fmt.Errorf("write staged artifact: %w", copyErr))
	}
	if written > a.maxBytes {
		return removePartial(ErrArtifactTooLarge)
	}
	if uint64(written) != descriptor.SizeBytes {
		return removePartial(ErrArtifactSize)
	}
	if err := ctx.Err(); err != nil {
		return removePartial(err)
	}
	if err := file.Sync(); err != nil {
		return removePartial(fmt.Errorf("sync staged artifact: %w", err))
	}
	if err := file.Close(); err != nil {
		_ = a.files.RemoveArtifactFile(context.WithoutCancel(ctx), stagingPath)
		return installation.StagedArtifact{}, fmt.Errorf("close staged artifact: %w", err)
	}
	return installation.StagedArtifact{
		StagingPath:    stagingPath,
		DescriptorJSON: append([]byte(nil), descriptorJSON...),
	}, nil
}

func (a *HTTPArtifactAcquirer) validateDescriptor(descriptor artifactDescriptor) (*url.URL, error) {
	if descriptor.Schema != "urn:pride:registry-artifact-descriptor:1" ||
		descriptor.SchemaVersion != "1.0.0" ||
		strings.TrimSpace(descriptor.ReleaseID) == "" ||
		descriptor.MediaType != "application/vnd.pride.plugin+zip" ||
		descriptor.SizeBytes == 0 || descriptor.SizeBytes > uint64(a.maxBytes) {
		if descriptor.SizeBytes > uint64(a.maxBytes) {
			return nil, ErrArtifactTooLarge
		}
		return nil, ErrInvalidDescriptor
	}
	parsed, err := url.Parse(descriptor.DownloadURI)
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" ||
		parsed.Fragment != "" || (parsed.Port() != "" && parsed.Port() != "443") {
		return nil, ErrUnsafeArtifactURL
	}
	host := normalizeHost(parsed.Hostname())
	if net.ParseIP(host) != nil {
		return nil, ErrUnsafeArtifactURL
	}
	if _, ok := a.hosts[host]; !ok {
		return nil, ErrUnsafeArtifactURL
	}
	return parsed, nil
}

func normalizeAllowedHosts(values []string) (map[string]struct{}, error) {
	if len(values) == 0 {
		return nil, errors.New("at least one exact artifact hostname must be allowlisted")
	}
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		host := normalizeHost(value)
		if host == "" || strings.ContainsAny(host, "*/:@?#") || net.ParseIP(host) != nil {
			return nil, fmt.Errorf("invalid artifact hostname allowlist entry %q", value)
		}
		if _, err := url.Parse("https://" + host); err != nil {
			return nil, fmt.Errorf("invalid artifact hostname allowlist entry %q", value)
		}
		result[host] = struct{}{}
	}
	return result, nil
}

func normalizeHost(host string) string {
	return strings.TrimSuffix(strings.ToLower(strings.TrimSpace(host)), ".")
}

func isPublicUnicast(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if v4 := ip.To4(); v4 != nil {
		ip = v4
		// Reject shared address space and special-purpose/documentation ranges
		// in addition to RFC 1918, loopback, and link-local addresses.
		if v4[0] == 100 && v4[1] >= 64 && v4[1] <= 127 ||
			v4[0] == 192 && v4[1] == 0 && (v4[2] == 0 || v4[2] == 2) ||
			v4[0] == 192 && v4[1] == 88 && v4[2] == 99 ||
			v4[0] == 198 && (v4[1] == 18 || v4[1] == 19 || v4[1] == 51 && v4[2] == 100) ||
			v4[0] == 203 && v4[1] == 0 && v4[2] == 113 ||
			v4[0] >= 240 {
			return false
		}
	} else if strings.HasPrefix(strings.ToLower(ip.String()), "2001:db8:") {
		return false
	}
	return ip.IsGlobalUnicast() && !ip.IsPrivate() &&
		!ip.IsLoopback() && !ip.IsLinkLocalUnicast() &&
		!ip.IsLinkLocalMulticast() && !ip.IsMulticast() && !ip.IsUnspecified()
}

func rejectDuplicateJSONKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := consumeJSONValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}

func consumeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := make(map[string]struct{})
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return ErrInvalidDescriptor
			}
			if _, exists := seen[key]; exists {
				return fmt.Errorf("duplicate JSON key %q", key)
			}
			seen[key] = struct{}{}
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return ErrInvalidDescriptor
	}
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r *contextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
