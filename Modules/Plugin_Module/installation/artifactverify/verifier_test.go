package artifactverify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

type testKeys struct {
	key     ed25519.PublicKey
	revoked bool
}

func (k testKeys) ResolveDescriptorKey(context.Context, string, time.Time) (TrustedKey, error) {
	return TrustedKey{
		PublicKey:  k.key,
		ValidFrom:  time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ValidUntil: time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC),
		Revoked:    k.revoked,
	}, nil
}

type testSource struct {
	path string
}

func (s testSource) OpenArtifactFile(ctx context.Context, _ string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return os.Open(s.path)
}

type testPaths struct{}

func (testPaths) ResolveFinalArtifactPath(_ context.Context, id domain.PluginID, version string) (string, error) {
	return "installed/" + string(id) + "/" + version, nil
}

func TestVerifierAcceptsSignedDescriptorAndExactArtifact(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	got, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{ManifestJSON: []byte(`{"plugin_id":"example.plugin"}`)}, plugin)
	if err != nil {
		t.Fatal(err)
	}
	if got.DigestAlgorithm != "sha-256" || got.DigestValue == "" || got.FinalArtifactPath != "installed/example.plugin/1.2.0" {
		t.Fatalf("unexpected verification result: %#v", got)
	}
}

func TestVerifierRejectsChangedArtifactBytes(t *testing.T) {
	verifier, staged, plugin, artifactPath, _ := fixture(t, false)
	if err := os.WriteFile(artifactPath, []byte("changed bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected changed artifact bytes to fail digest verification")
	}
}

func TestVerifierRejectsRevokedKey(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, true)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected revoked signing key to be rejected")
	}
}

func TestVerifierRejectsUnknownSignatureEnvelopeProperties(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var signature map[string]any
	if err := json.Unmarshal(fields["signature"], &signature); err != nil {
		t.Fatal(err)
	}
	signature["untrusted_extension"] = "not allowed"
	fields["signature"], _ = json.Marshal(signature)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected unknown signature envelope property to be rejected")
	}
}

func TestVerifierRejectsDuplicateDescriptorKeys(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	staged.DescriptorJSON = []byte(`{"$schema":"urn:pride:registry-artifact-descriptor:1","$schema":"urn:pride:registry-artifact-descriptor:1"}`)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected duplicate descriptor key to be rejected")
	}
}

func fixture(t *testing.T, revoked bool) (*Verifier, installation.StagedArtifact, domain.Plugin, string, ed25519.PrivateKey) {
	t.Helper()
	dir := t.TempDir()
	artifactPath := filepath.Join(dir, "artifact.zip")
	artifact := []byte("immutable test ZIP bytes")
	if err := os.WriteFile(artifactPath, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	digest := sha256Hex(artifact)
	fields := map[string]any{
		"$schema":           descriptorSchema,
		"schema_version":    "1.1.0",
		"plugin_id":         "example.plugin",
		"version":           "1.2.0",
		"release_id":        "release-123",
		"download_uri":      "https://artifacts.example.test/release.zip",
		"media_type":        descriptorMediaType,
		"size_bytes":        len(artifact),
		"digest":            map[string]string{"algorithm": "sha-256", "value": digest},
		"platform_metadata": map[string]any{"schema_version": "1.0.0", "plugin_type": "passive", "capabilities": []any{}, "permission_ids": []string{}, "compatibility_status": "verified"},
	}
	payload, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	signature := ed25519.Sign(privateKey, payload)
	fields["signature"] = map[string]string{
		"algorithm": "Ed25519",
		"key_id":    "marketplace-2026",
		"value":     base64.StdEncoding.EncodeToString(signature),
	}
	raw, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := New(testSource{path: artifactPath}, testKeys{key: publicKey, revoked: revoked}, JCSCanonicalizer{}, testPaths{})
	if err != nil {
		t.Fatal(err)
	}
	verifier.now = func() time.Time { return time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC) }
	return verifier, installation.StagedArtifact{StagingPath: "temporary/op-1", DescriptorJSON: raw}, domain.Plugin{
		ID:        "example.plugin",
		Type:      domain.PluginTypePassive,
		Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}, PermissionIDs: []string{}, CompatibilityStatus: "verified"},
		Manifest:  domain.Manifest{PluginID: "example.plugin", Version: "1.2.0"},
	}, artifactPath, privateKey
}

func sha256Hex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func TestVerifierRejectsUnsupportedPlatformMetadataSchema(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["schema_version"] = "2.0.0"
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected unsupported metadata schema to fail")
	}
}

func TestVerifierRejectsUnknownPlatformMetadataProperty(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["trusted"] = true
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected unknown metadata property to fail")
	}
}

func TestVerifierReturnsPlatformMetadataOnlyAfterVerification(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	got, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin)
	if err != nil {
		t.Fatal(err)
	}
	if got.PlatformMetadata.Type != domain.PluginTypePassive || got.PlatformMetadata.Protected.CompatibilityStatus != "verified" {
		t.Fatalf("unexpected verified platform metadata: %#v", got.PlatformMetadata)
	}
}

func TestVerifierRejectsDuplicateCapabilityIdentifiers(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["capabilities"] = []any{
		map[string]any{"id": "text.generate", "name": "Text", "description": ""},
		map[string]any{"id": "text.generate", "name": "Text duplicate", "description": ""},
	}
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected duplicate capability IDs to fail")
	}
}

func TestVerifierRejectsMissingRequiredPlatformMetadataArray(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	delete(metadata, "permission_ids")
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected missing permission_ids to fail")
	}
}

func TestVerifierRejectsMissingCapabilityDescription(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]json.RawMessage
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["capabilities"] = json.RawMessage(`[{"id":"text.generate","name":"Text"}]`)
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); err == nil {
		t.Fatal("expected missing capability description to fail")
	}
}

func TestVerifierRejectsTamperedSignedPlatformMetadata(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	var metadata map[string]any
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil {
		t.Fatal(err)
	}
	metadata["compatibility_status"] = "tampered"
	fields["platform_metadata"], _ = json.Marshal(metadata)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); !errors.Is(err, ErrSignatureInvalid) {
		t.Fatalf("expected signature error for modified signed metadata, got %v", err)
	}
}

func TestVerifierRejectsLegacyDescriptorVersion(t *testing.T) {
	verifier, staged, plugin, _, _ := fixture(t, false)
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(staged.DescriptorJSON, &fields); err != nil {
		t.Fatal(err)
	}
	fields["schema_version"] = json.RawMessage(`"1.0.0"`)
	staged.DescriptorJSON, _ = json.Marshal(fields)
	if _, err := verifier.Verify(context.Background(), staged, installation.InspectedPackage{}, plugin); !errors.Is(err, ErrUnsupportedDescriptorVersion) {
		t.Fatalf("expected unsupported descriptor version, got %v", err)
	}
}
