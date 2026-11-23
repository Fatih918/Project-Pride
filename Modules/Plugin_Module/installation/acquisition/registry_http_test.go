package acquisition

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type registryTokenProviderFunc func(context.Context, string, string) (BearerToken, error)

func (f registryTokenProviderFunc) Token(ctx context.Context, namespace, audience string) (BearerToken, error) {
	return f(ctx, namespace, audience)
}

func newTestRegistryResolver(status int, body string, headers http.Header, token BearerToken) (*HTTPRegistryDescriptorResolver, *http.Request, error) {
	var captured *http.Request
	client := &http.Client{Transport: roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		captured = request.Clone(request.Context())
		captured.Header = request.Header.Clone()
		response := &http.Response{
			StatusCode: status,
			Header:     headers.Clone(),
			Body:       io.NopCloser(strings.NewReader(body)),
			ContentLength: int64(len(body)),
		}
		return response, nil
	})}
	provider := registryTokenProviderFunc(func(_ context.Context, namespace, audience string) (BearerToken, error) {
		if namespace != "community" || audience != "https://registry.example.test" {
			return BearerToken{}, errors.New("unexpected token audience")
		}
		return token, nil
	})
	resolver := &HTTPRegistryDescriptorResolver{
		origins: map[string]*url.URL{"community": {Scheme: "https", Host: "registry.example.test"}},
		tokens: provider,
		client: client,
		now: func() time.Time { return time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC) },
	}
	return resolver, captured, nil
}

func validRegistryDescriptor() string {
	return `{"$schema":"urn:pride:registry-artifact-descriptor:1","schema_version":"1.0.0","plugin_id":"example.plugin","version":"1.2.3","release_id":"rel_01J9K8M7N6P5Q4R3S2T1","download_uri":"https://downloads.example.test/plugin.zip","media_type":"application/vnd.pride.plugin+zip","size_bytes":42,"digest":{"algorithm":"sha-256","value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"signature":{"algorithm":"Ed25519","key_id":"marketplace-1","value":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="}}`
}

func validResolution(t *testing.T) DescriptorResolutionRequest {
	t.Helper()
	reference, err := ParseRegistryReleaseReference("pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1")
	if err != nil {
		t.Fatal(err)
	}
	resolution, err := NewDescriptorResolutionRequest(reference, "example.plugin", "1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	return resolution
}

func TestRegistryResolverReturnsExactDescriptorAndUsesScopedBearerToken(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	token := BearerToken{Value: "short-lived-token", ExpiresAt: now.Add(5 * time.Minute)}
	resolver, _, _ := newTestRegistryResolver(http.StatusOK, validRegistryDescriptor(), http.Header{"Content-Type": []string{"application/json; charset=utf-8"}}, token)
	var captured *http.Request
	resolver.client.Transport = roundTripperFunc(func(request *http.Request) (*http.Response, error) {
		captured = request.Clone(request.Context())
		captured.Header = request.Header.Clone()
		return &http.Response{
			StatusCode: http.StatusOK,
			Header: http.Header{"Content-Type": []string{"application/json"}},
			Body: io.NopCloser(strings.NewReader(validRegistryDescriptor())),
			ContentLength: int64(len(validRegistryDescriptor())),
		}, nil
	})
	resolver.now = func() time.Time { return now }

	got, err := resolver.ResolveDescriptor(context.Background(), validResolution(t))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != validRegistryDescriptor() {
		t.Fatal("resolver must preserve exact descriptor bytes")
	}
	if captured == nil || captured.URL.String() != "https://registry.example.test/v1/plugins/example.plugin/releases/rel_01J9K8M7N6P5Q4R3S2T1/descriptor" {
		t.Fatalf("unexpected Registry request: %#v", captured)
	}
	if captured.Header.Get("Authorization") != "Bearer short-lived-token" ||
		captured.Header.Get("Accept-Encoding") != "identity" {
		t.Fatal("expected bearer authentication and identity encoding")
	}
}

func TestRegistryResolverRejectsExpiredOrMalformedToken(t *testing.T) {
	now := time.Date(2026, 11, 23, 12, 0, 0, 0, time.UTC)
	for _, token := range []BearerToken{
		{Value: "expired", ExpiresAt: now},
		{Value: "bad token", ExpiresAt: now.Add(time.Minute)},
	} {
		resolver, _, _ := newTestRegistryResolver(http.StatusOK, validRegistryDescriptor(), make(http.Header), token)
		resolver.now = func() time.Time { return now }
		if _, err := resolver.ResolveDescriptor(context.Background(), validResolution(t)); !errors.Is(err, ErrRegistryAuthentication) {
			t.Fatalf("expected authentication rejection, got %v", err)
		}
	}
}

func TestRegistryResolverMapsSafeHTTPErrors(t *testing.T) {
	tests := []struct {
		status int
		want   error
	}{
		{http.StatusUnauthorized, ErrRegistryAuthentication},
		{http.StatusForbidden, ErrRegistryAuthorization},
		{http.StatusNotFound, ErrRegistryNotFound},
		{http.StatusConflict, ErrRegistryConflict},
		{http.StatusTooManyRequests, ErrRegistryRateLimited},
		{http.StatusServiceUnavailable, ErrRegistryUnavailable},
		{http.StatusMovedPermanently, ErrRegistryResponse},
	}
	token := BearerToken{Value: "token", ExpiresAt: time.Now().Add(time.Minute)}
	for _, test := range tests {
		resolver, _, _ := newTestRegistryResolver(test.status, "secret response body", http.Header{
			"X-Request-ID": []string{"req_123"},
			"Retry-After": []string{"30"},
		}, token)
		_, err := resolver.ResolveDescriptor(context.Background(), validResolution(t))
		if !errors.Is(err, test.want) {
			t.Fatalf("status %d: expected %v, got %v", test.status, test.want, err)
		}
		var responseErr *RegistryHTTPError
		if !errors.As(err, &responseErr) {
			t.Fatalf("status %d: expected RegistryHTTPError, got %T", test.status, err)
		}
		if strings.Contains(err.Error(), "secret response body") || strings.Contains(err.Error(), "token") {
			t.Fatalf("status %d leaked response or token: %v", test.status, err)
		}
		if test.status == http.StatusTooManyRequests && responseErr.RetryAfter != 30*time.Second {
			t.Fatalf("unexpected Retry-After: %s", responseErr.RetryAfter)
		}
		if responseErr.RequestID != "req_123" {
			t.Fatalf("unexpected safe request ID: %q", responseErr.RequestID)
		}
	}
}

func TestRegistryResolverRejectsIdentityMismatch(t *testing.T) {
	body := strings.Replace(validRegistryDescriptor(), `"version":"1.2.3"`, `"version":"9.9.9"`, 1)
	resolver, _, _ := newTestRegistryResolver(http.StatusOK, body, http.Header{"Content-Type": []string{"application/json"}}, BearerToken{Value: "token", ExpiresAt: time.Now().Add(time.Minute)})
	if _, err := resolver.ResolveDescriptor(context.Background(), validResolution(t)); !errors.Is(err, ErrRegistryIdentity) {
		t.Fatalf("expected identity mismatch, got %v", err)
	}
}

func TestRegistryResolverRejectsInvalidRepresentation(t *testing.T) {
	cases := []struct {
		name    string
		body    string
		headers http.Header
	}{
		{"wrong content type", validRegistryDescriptor(), http.Header{"Content-Type": []string{"text/plain"}}},
		{"compressed response", validRegistryDescriptor(), http.Header{"Content-Type": []string{"application/json"}, "Content-Encoding": []string{"gzip"}}},
		{"duplicate keys", strings.Replace(validRegistryDescriptor(), `"$schema":`, `"$schema":"urn:pride:registry-artifact-descriptor:1","$schema":`, 1), http.Header{"Content-Type": []string{"application/json"}}},
		{"empty response", "", http.Header{"Content-Type": []string{"application/json"}}},
	}
	for _, test := range cases {
		resolver, _, _ := newTestRegistryResolver(http.StatusOK, test.body, test.headers, BearerToken{Value: "token", ExpiresAt: time.Now().Add(time.Minute)})
		if _, err := resolver.ResolveDescriptor(context.Background(), validResolution(t)); !errors.Is(err, ErrRegistryResponse) {
			t.Errorf("%s: expected invalid response, got %v", test.name, err)
		}
	}
}

func TestRegistryResolverRejectsUnsafeOriginConfiguration(t *testing.T) {
	provider := registryTokenProviderFunc(func(context.Context, string, string) (BearerToken, error) {
		return BearerToken{}, nil
	})
	for _, origin := range []string{
		"http://registry.example.test",
		"https://user:pass@registry.example.test",
		"https://registry.example.test/path",
		"https://127.0.0.1",
		"https://registry.example.test:444",
	} {
		if _, err := NewHTTPRegistryDescriptorResolver(RegistryResolverConfig{Origins: map[string]string{"community": origin}}, provider); !errors.Is(err, ErrRegistryConfiguration) {
			t.Errorf("origin %q: expected configuration rejection, got %v", origin, err)
		}
	}
}
