package artifactverify

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPTrustConfigurationSourceRequiresPinnedHTTPSHost(t *testing.T) {
	for _, config := range []HTTPTrustConfigurationSourceConfig{
		{},
		{EndpointURL: "http://trust.example.test/config.json", AllowedHosts: []string{"trust.example.test"}},
		{EndpointURL: "https://attacker.example.test/config.json", AllowedHosts: []string{"trust.example.test"}},
		{EndpointURL: "https://127.0.0.1/config.json", AllowedHosts: []string{"127.0.0.1"}},
		{EndpointURL: "https://user@trust.example.test/config.json", AllowedHosts: []string{"trust.example.test"}},
		{EndpointURL: "https://trust.example.test:8443/config.json", AllowedHosts: []string{"trust.example.test"}},
		{EndpointURL: "https://trust.example.test/config.json", AllowedHosts: []string{"*.example.test"}},
		{EndpointURL: "https://trust.example.test/config.json", AllowedHosts: []string{"trust.example.test"}, Timeout: 2 * time.Minute},
	} {
		if _, err := NewHTTPTrustConfigurationSource(config); err == nil {
			t.Errorf("expected invalid endpoint configuration to fail: %#v", config)
		}
	}
}

func TestHTTPTrustConfigurationSourceReadsJSONDocument(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.Header.Get("Accept") != "application/json" ||
			r.Header.Get("Accept-Encoding") != "identity" {
			t.Errorf("unexpected request: %#v", r)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = io.WriteString(w, "{\"sequence\":1}")
	}))
	defer server.Close()

	parsed, err := parseTestEndpoint(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	source := &HTTPTrustConfigurationSource{
		endpoint: parsed,
		client:   server.Client(),
	}
	got, err := source.ReadTrustConfiguration(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "{\"sequence\":1}" {
		t.Fatalf("response = %q", got)
	}
}

func TestHTTPTrustConfigurationSourceRejectsInvalidResponses(t *testing.T) {
	for _, test := range []struct {
		name        string
		status      int
		contentType string
		encoding    string
		body        string
	}{
		{name: "status", status: http.StatusServiceUnavailable, contentType: "application/json", body: "{}"},
		{name: "content type", status: http.StatusOK, contentType: "text/plain", body: "{}"},
		{name: "encoding", status: http.StatusOK, contentType: "application/json", encoding: "gzip", body: "{}"},
		{name: "empty", status: http.StatusOK, contentType: "application/json"},
	} {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", test.contentType)
				if test.encoding != "" {
					w.Header().Set("Content-Encoding", test.encoding)
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			parsed, err := parseTestEndpoint(server.URL)
			if err != nil {
				t.Fatal(err)
			}
			source := &HTTPTrustConfigurationSource{endpoint: parsed, client: server.Client()}
			if _, err := source.ReadTrustConfiguration(context.Background()); err == nil {
				t.Fatal("expected invalid response to be rejected")
			}
		})
	}
}

func TestHTTPTrustConfigurationSourceRejectsOversizedBody(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, strings.Repeat("x", maxTrustConfigurationBytes+1))
	}))
	defer server.Close()
	parsed, err := parseTestEndpoint(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	source := &HTTPTrustConfigurationSource{endpoint: parsed, client: server.Client()}
	if _, err := source.ReadTrustConfiguration(context.Background()); err == nil {
		t.Fatal("expected oversized response to be rejected")
	}
}

func parseTestEndpoint(raw string) (*url.URL, error) {
	return url.Parse(raw)
}
