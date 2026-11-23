package acquisition

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type descriptorResolverFunc func(context.Context, string) ([]byte, error)

func (f descriptorResolverFunc) ResolveDescriptor(ctx context.Context, request DescriptorResolutionRequest) ([]byte, error) {
	return f(ctx, request.Reference.String())
}

type artifactFileStore struct{ path string }

func (s artifactFileStore) CreateArtifactFile(_ context.Context, _ string) (*os.File, error) {
	return os.OpenFile(s.path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
}

func (s artifactFileStore) RemoveArtifactFile(_ context.Context, _ string) error {
	err := os.Remove(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

func descriptor(uri, size string) []byte {
	return []byte(`{"$schema":"urn:pride:registry-artifact-descriptor:1","schema_version":"1.0.0","plugin_id":"example.plugin","version":"1.0.0","release_id":"rel_01J9K8M7N6P5Q4R3S2T1","download_uri":"` + uri + `","media_type":"application/vnd.pride.plugin+zip","size_bytes":` + size + `,"digest":{"algorithm":"sha-256","value":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"signature":{"algorithm":"Ed25519","key_id":"key-1","value":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=="}}`)
}

func newTestAcquirer(t *testing.T, rawDescriptor []byte, response *http.Response, maxBytes int64) (*HTTPArtifactAcquirer, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "artifact.zip")
	resolver := descriptorResolverFunc(func(context.Context, string) ([]byte, error) {
		return append([]byte(nil), rawDescriptor...), nil
	})
	files := artifactFileStore{path: path}
	hosts := map[string]struct{}{"downloads.example.test": {}}
	client := &http.Client{Transport: roundTripperFunc(func(*http.Request) (*http.Response, error) {
		return response, nil
	})}
	return newHTTPArtifactAcquirer(resolver, files, hosts, maxBytes, client), path
}

func response(status int, body string) *http.Response {
	return &http.Response{
		StatusCode:    status,
		Body:          io.NopCloser(strings.NewReader(body)),
		ContentLength: int64(len(body)),
		Header:        make(http.Header),
	}
}

func TestAcquireDownloadsExactArtifactAndRetainsDescriptor(t *testing.T) {
	body := "zip-bytes"
	raw := descriptor("https://downloads.example.test/plugin.zip", "9")
	acquirer, path := newTestAcquirer(t, raw, response(http.StatusOK, body), 100)
	result, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.StagingPath != "temporary/op-1" || string(result.DescriptorJSON) != string(raw) {
		t.Fatalf("unexpected staged artifact: %#v", result)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != body {
		t.Fatalf("artifact content = %q", content)
	}
}

func TestAcquireRejectsUnapprovedHostBeforeRequest(t *testing.T) {
	raw := descriptor("https://attacker.example.test/plugin.zip", "9")
	acquirer, path := newTestAcquirer(t, raw, response(http.StatusOK, "zip-bytes"), 100)
	_, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1")
	if !errors.Is(err, ErrUnsafeArtifactURL) {
		t.Fatalf("expected unsafe URL rejection, got %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("artifact file must not be created, stat error = %v", statErr)
	}
}

func TestAcquireRejectsDuplicateDescriptorKeys(t *testing.T) {
	raw := []byte(`{"$schema":"urn:pride:registry-artifact-descriptor:1","$schema":"urn:pride:registry-artifact-descriptor:1"}`)
	acquirer, _ := newTestAcquirer(t, raw, response(http.StatusOK, "x"), 100)
	if _, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1"); !errors.Is(err, ErrInvalidDescriptor) {
		t.Fatalf("expected duplicate-key rejection, got %v", err)
	}
}

func TestAcquireRemovesPartialArtifactOnSizeMismatch(t *testing.T) {
	raw := descriptor("https://downloads.example.test/plugin.zip", "10")
	stream := response(http.StatusOK, "short")
	stream.ContentLength = -1
	acquirer, path := newTestAcquirer(t, raw, stream, 100)
	if _, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1"); !errors.Is(err, ErrArtifactSize) {
		t.Fatalf("expected size mismatch, got %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("partial artifact must be removed, stat error = %v", statErr)
	}
}

func TestAcquireEnforcesStreamingMaximumAndRemovesPartialArtifact(t *testing.T) {
	raw := descriptor("https://downloads.example.test/plugin.zip", "99")
	stream := response(http.StatusOK, strings.Repeat("x", 101))
	stream.ContentLength = -1
	acquirer, path := newTestAcquirer(t, raw, stream, 100)
	if _, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1"); !errors.Is(err, ErrArtifactTooLarge) {
		t.Fatalf("expected archive limit rejection, got %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("oversized partial artifact must be removed, stat error = %v", statErr)
	}
}

func TestAcquireRejectsDescriptorAboveConfiguredMaximum(t *testing.T) {
	raw := descriptor("https://downloads.example.test/plugin.zip", "101")
	acquirer, path := newTestAcquirer(t, raw, response(http.StatusOK, strings.Repeat("x", 101)), 100)
	if _, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1"); !errors.Is(err, ErrArtifactTooLarge) {
		t.Fatalf("expected declared archive limit rejection, got %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("artifact file must not be created, stat error = %v", statErr)
	}
}

func TestAcquireRejectsRedirectAndUnexpectedStatus(t *testing.T) {
	raw := descriptor("https://downloads.example.test/plugin.zip", "9")
	acquirer, path := newTestAcquirer(t, raw, response(http.StatusFound, ""), 100)
	if _, err := acquirer.Acquire(context.Background(), "pride-registry:v1/community/example.plugin/rel_01J9K8M7N6P5Q4R3S2T1", "example.plugin", "1.0.0", "temporary/op-1"); !errors.Is(err, ErrUnexpectedStatus) {
		t.Fatalf("expected redirect/status rejection, got %v", err)
	}
	if _, statErr := os.Stat(path); !errors.Is(statErr, os.ErrNotExist) {
		t.Fatalf("artifact file must not be created, stat error = %v", statErr)
	}
}

func TestAcquirePreservesResolverErrors(t *testing.T) {
	want := errors.New("registry unavailable")
	resolver := descriptorResolverFunc(func(context.Context, string) ([]byte, error) { return nil, want })
	acquirer, err := NewHTTPArtifactAcquirer(resolver, artifactFileStore{path: filepath.Join(t.TempDir(), "artifact.zip")}, Config{AllowedHosts: []string{"downloads.example.test"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = acquirer.Acquire(context.Background(), "release:1", "temporary/op-1")
	if !errors.Is(err, want) {
		t.Fatalf("expected resolver error, got %v", err)
	}
}

func TestNewHTTPArtifactAcquirerRejectsUnsafeConfiguration(t *testing.T) {
	resolver := descriptorResolverFunc(func(context.Context, string) ([]byte, error) { return nil, nil })
	files := artifactFileStore{path: filepath.Join(t.TempDir(), "artifact.zip")}
	for _, config := range []Config{
		{},
		{AllowedHosts: []string{"*.example.test"}},
		{AllowedHosts: []string{"127.0.0.1"}},
		{AllowedHosts: []string{"downloads.example.test"}, MaxArchiveBytes: maxArchiveBytes + 1},
	} {
		if _, err := NewHTTPArtifactAcquirer(resolver, files, config); err == nil {
			t.Errorf("expected unsafe config rejection: %#v", config)
		}
	}
}

func TestContextReaderStopsOnCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reader := &contextReader{ctx: ctx, reader: strings.NewReader("data")}
	if _, err := reader.Read(make([]byte, 4)); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
}
