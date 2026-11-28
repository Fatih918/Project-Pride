package schema

import (
	"context"
	"errors"
	"strings"
	"testing"

	"pride-plugin-module/installation"
)

type memoryPackageReader struct {
	path  string
	ref   string
	data  []byte
	err   error
	limit int64
}

func (r *memoryPackageReader) ReadPackageFile(_ context.Context, artifactPath, reference string, maxBytes int64) ([]byte, error) {
	r.path, r.ref, r.limit = artifactPath, reference, maxBytes
	if r.err != nil {
		return nil, r.err
	}
	if maxBytes < int64(len(r.data)) {
		return nil, errors.New("reader limit exceeded")
	}
	return append([]byte(nil), r.data...), nil
}

func verifiedArtifact() installation.VerifiedArtifact {
	return installation.VerifiedArtifact{
		FinalArtifactPath:       "installed/example.plugin/1.2.3",
		DigestAlgorithm:         "sha-256",
		DigestValue:             strings.Repeat("a", 64),
		VerificationPolicy:      "descriptor+sha256+ed25519",
		VerificationEvidenceRef: "marketplace-key:test-key",
	}
}

func TestValidateReference(t *testing.T) {
	valid := []string{"schemas/request.json", "schemas/nested/response.schema.json", "schemas/a%20b.json"}
	for _, value := range valid {
		if err := ValidateReference(value); err != nil {
			t.Errorf("ValidateReference(%q): %v", value, err)
		}
	}

	invalid := []string{
		"", "request.json", "/schemas/request.json", "../schemas/request.json",
		"schemas/../secret.json", "schemas//request.json", "schemas/./request.json",
		"schemas\\request.json", "schemas/a%2fb.json", "schemas/a%5Csecret.json",
		"schemas/request.json?download=1", "schemas/request.json#fragment",
		"schemas/C:/request.json", "schemas/\x00request.json",
		strings.Repeat("a", MaxReferenceBytes+1),
	}
	for _, value := range invalid {
		if err := ValidateReference(value); !errors.Is(err, ErrInvalidReference) {
			t.Errorf("ValidateReference(%q) = %v, want ErrInvalidReference", value, err)
		}
	}
}

func TestResolverReadsOnlyFromVerifiedPackage(t *testing.T) {
	reader := &memoryPackageReader{data: []byte(`{"$schema":"https://json-schema.org/draft/2020-12/schema","type":"object"}`)}
	resolver, err := NewResolver(reader)
	if err != nil {
		t.Fatal(err)
	}
	got, err := resolver.Resolve(context.Background(), verifiedArtifact(), "schemas/request.json")
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(reader.data) {
		t.Fatalf("document = %q, want %q", got, reader.data)
	}
	if reader.path != "installed/example.plugin/1.2.3" ||
		reader.ref != "schemas/request.json" || reader.limit != MaxDocumentBytes {
		t.Fatalf("reader called with path=%q ref=%q limit=%d", reader.path, reader.ref, reader.limit)
	}
	got[0] = 'x'
	if reader.data[0] != '{' {
		t.Fatal("resolver must return an owned copy of document bytes")
	}
}

func TestResolverRejectsMissingVerificationEvidence(t *testing.T) {
	resolver, _ := NewResolver(&memoryPackageReader{data: []byte("{}")})
	artifact := verifiedArtifact()
	artifact.VerificationEvidenceRef = ""
	if _, err := resolver.Resolve(context.Background(), artifact, "schemas/request.json"); !errors.Is(err, ErrUnverifiedPackage) {
		t.Fatalf("Resolve error = %v, want ErrUnverifiedPackage", err)
	}
}

func TestResolverRejectsInvalidReferenceBeforeReading(t *testing.T) {
	reader := &memoryPackageReader{data: []byte("{}")}
	resolver, _ := NewResolver(reader)
	if _, err := resolver.Resolve(context.Background(), verifiedArtifact(), "schemas/../secret.json"); !errors.Is(err, ErrInvalidReference) {
		t.Fatalf("Resolve error = %v, want ErrInvalidReference", err)
	}
	if reader.ref != "" {
		t.Fatal("reader must not be called for an invalid reference")
	}
}

func TestResolverPropagatesContextAndReaderErrors(t *testing.T) {
	reader := &memoryPackageReader{data: []byte("{}"), err: errors.New("read failed")}
	resolver, _ := NewResolver(reader)
	if _, err := resolver.Resolve(context.Background(), verifiedArtifact(), "schemas/request.json"); err == nil {
		t.Fatal("expected package reader error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := resolver.Resolve(ctx, verifiedArtifact(), "schemas/request.json"); !errors.Is(err, context.Canceled) {
		t.Fatalf("Resolve error = %v, want context.Canceled", err)
	}
}
