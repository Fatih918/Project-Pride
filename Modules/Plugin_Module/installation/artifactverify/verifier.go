package artifactverify

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

const (
	descriptorSchema    = "urn:pride:registry-artifact-descriptor:1"
	descriptorMediaType = "application/vnd.pride.plugin+zip"
	maxDescriptorBytes  = 1 << 20
)

var (
	ErrInvalidDescriptor = errors.New("invalid Registry artifact descriptor")
	ErrUntrustedKey      = errors.New("Marketplace signing key is not trusted")
	ErrDigestMismatch    = errors.New("artifact SHA-256 digest mismatch")
	ErrSignatureInvalid  = errors.New("artifact descriptor signature is invalid")
)

// Canonicalizer must implement RFC 8785 JSON Canonicalization Scheme exactly.
// Production wiring must use a conformance-tested implementation; encoding/json
// output alone is not a substitute for JCS.
type Canonicalizer interface {
	Canonicalize([]byte) ([]byte, error)
}

// TrustedKeyResolver returns a key only from a trust configuration that has
// already passed embedded-bootstrap signature verification, expiry checks,
// and persisted sequence/rollback checks.
type TrustedKeyResolver interface {
	ResolveDescriptorKey(context.Context, string, time.Time) (TrustedKey, error)
}

type TrustedKey struct {
	PublicKey  ed25519.PublicKey
	ValidFrom  time.Time
	ValidUntil time.Time
	Revoked    bool
}

// ArtifactSource opens the exact staged ZIP through the confined storage adapter.
type ArtifactSource interface {
	OpenArtifactFile(context.Context, string) (*os.File, error)
}

// FinalPathResolver creates the canonical Plugin_Storage-relative immutable
// destination. It must reject unsafe Plugin IDs and versions.
type FinalPathResolver interface {
	ResolveFinalArtifactPath(context.Context, domain.PluginID, string) (string, error)
}

type Verifier struct {
	source        ArtifactSource
	keys          TrustedKeyResolver
	canonicalizer Canonicalizer
	paths         FinalPathResolver
	now           func() time.Time
}

func New(source ArtifactSource, keys TrustedKeyResolver, canonicalizer Canonicalizer, paths FinalPathResolver) (*Verifier, error) {
	if source == nil || keys == nil || canonicalizer == nil || paths == nil {
		return nil, errors.New("artifact verifier dependencies are required")
	}
	return &Verifier{source: source, keys: keys, canonicalizer: canonicalizer, paths: paths, now: func() time.Time { return time.Now().UTC() }}, nil
}

type descriptor struct {
	Schema        string `json:"$schema"`
	SchemaVersion string `json:"schema_version"`
	PluginID      string `json:"plugin_id"`
	Version       string `json:"version"`
	ReleaseID     string `json:"release_id"`
	DownloadURI   string `json:"download_uri"`
	MediaType     string `json:"media_type"`
	SizeBytes     uint64 `json:"size_bytes"`
	Digest        struct {
		Algorithm string `json:"algorithm"`
		Value     string `json:"value"`
	} `json:"digest"`
	PlatformMetadata struct {
		SchemaVersion       string `json:"schema_version"`
		PluginType          string `json:"plugin_type"`
		Capabilities        []domain.Capability `json:"capabilities"`
		PermissionIDs       []string `json:"permission_ids"`
		CompatibilityStatus string `json:"compatibility_status"`
	} `json:"platform_metadata"`
	Signature struct {
		Algorithm string `json:"algorithm"`
		KeyID     string `json:"key_id"`
		Value     string `json:"value"`
	} `json:"signature"`
}

func (v *Verifier) Verify(ctx context.Context, staged installation.StagedArtifact, inspected installation.InspectedPackage, plugin domain.Plugin) (installation.VerifiedArtifact, error) {
	if err := ctx.Err(); err != nil {
		return installation.VerifiedArtifact{}, err
	}
	if staged.StagingPath == "" || len(staged.DescriptorJSON) == 0 || len(staged.DescriptorJSON) > maxDescriptorBytes {
		return installation.VerifiedArtifact{}, ErrInvalidDescriptor
	}
	if err := rejectDuplicateKeys(staged.DescriptorJSON); err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	if err := validateDescriptorNestedProperties(staged.DescriptorJSON); err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	var d descriptor
	if err := json.Unmarshal(staged.DescriptorJSON, &d); err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	if err := validateDescriptor(d, plugin); err != nil {
		return installation.VerifiedArtifact{}, err
	}

	now := v.now().UTC()
	key, err := v.keys.ResolveDescriptorKey(ctx, d.Signature.KeyID, now)
	if err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("%w: %v", ErrUntrustedKey, err)
	}
	if key.Revoked || len(key.PublicKey) != ed25519.PublicKeySize ||
		key.ValidFrom.IsZero() || key.ValidUntil.IsZero() ||
		now.Before(key.ValidFrom) || !now.Before(key.ValidUntil) {
		return installation.VerifiedArtifact{}, ErrUntrustedKey
	}

	payload, err := descriptorSigningPayload(staged.DescriptorJSON)
	if err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("%w: %v", ErrInvalidDescriptor, err)
	}
	canonical, err := v.canonicalizer.Canonicalize(payload)
	if err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("canonicalize descriptor using RFC 8785: %w", err)
	}
	signature, err := base64.StdEncoding.Strict().DecodeString(d.Signature.Value)
	if err != nil || len(signature) != ed25519.SignatureSize ||
		!ed25519.Verify(key.PublicKey, canonical, signature) {
		return installation.VerifiedArtifact{}, ErrSignatureInvalid
	}

	file, err := v.source.OpenArtifactFile(ctx, staged.StagingPath)
	if err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("open staged artifact: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 1 || uint64(info.Size()) != d.SizeBytes {
		return installation.VerifiedArtifact{}, ErrInvalidDescriptor
	}
	hash := sha256.New()
	if _, err := io.Copy(hash, &contextReader{ctx: ctx, reader: file}); err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("hash staged artifact: %w", err)
	}
	actualDigest := hex.EncodeToString(hash.Sum(nil))
	if actualDigest != d.Digest.Value {
		return installation.VerifiedArtifact{}, ErrDigestMismatch
	}
	if err := ctx.Err(); err != nil {
		return installation.VerifiedArtifact{}, err
	}
	finalPath, err := v.paths.ResolveFinalArtifactPath(ctx, plugin.ID, plugin.Manifest.Version)
	if err != nil {
		return installation.VerifiedArtifact{}, fmt.Errorf("resolve final artifact path: %w", err)
	}
	if strings.TrimSpace(finalPath) == "" {
		return installation.VerifiedArtifact{}, installation.ErrRecordMismatch
	}
	platformType := domain.PluginTypeActive
	if d.PlatformMetadata.PluginType == "passive" { platformType = domain.PluginTypePassive }
	return installation.VerifiedArtifact{
		PlatformMetadata: installation.PlatformMetadata{Type: platformType, Protected: domain.ProtectedMetadata{Capabilities: append([]domain.Capability(nil), d.PlatformMetadata.Capabilities...), PermissionIDs: append([]string(nil), d.PlatformMetadata.PermissionIDs...), CompatibilityStatus: d.PlatformMetadata.CompatibilityStatus}},
		FinalArtifactPath:       finalPath,
		DigestAlgorithm:         "sha-256",
		DigestValue:             actualDigest,
		VerificationPolicy:      "registry-artifact-descriptor-v1+sha256+ed25519",
		VerificationEvidenceRef: "marketplace-key:" + d.Signature.KeyID,
		ManifestJSON:            string(append([]byte(nil), inspected.ManifestJSON...)),
	}, nil
}

func validateDescriptor(d descriptor, plugin domain.Plugin) error {
	if d.Schema != descriptorSchema || d.SchemaVersion != "1.1.0" ||
		d.PluginID == "" || len([]rune(d.PluginID)) > 255 || d.PluginID != string(plugin.ID) ||
		d.Version == "" || len([]rune(d.Version)) > 128 || d.Version != plugin.Manifest.Version ||
		d.ReleaseID == "" || len([]rune(d.ReleaseID)) > 128 || d.SizeBytes == 0 ||
		d.MediaType != descriptorMediaType ||
		d.Digest.Algorithm != "sha-256" || len(d.Digest.Value) != sha256.Size*2 ||
		d.Signature.Algorithm != "Ed25519" || strings.TrimSpace(d.Signature.KeyID) == "" || len([]rune(d.Signature.KeyID)) > 255 {
		return ErrInvalidDescriptor
	}
	if err := validatePlatformMetadata(d.PlatformMetadata); err != nil {
		return fmt.Errorf("%w: invalid platform metadata: %v", ErrInvalidDescriptor, err)
	}
	if _, err := hex.DecodeString(d.Digest.Value); err != nil || strings.ToLower(d.Digest.Value) != d.Digest.Value {
		return ErrInvalidDescriptor
	}
	uri, err := url.Parse(d.DownloadURI)
	if err != nil || uri.Scheme != "https" || uri.Host == "" || uri.User != nil {
		return ErrInvalidDescriptor
	}
	return nil
}

func validateDescriptorNestedProperties(raw []byte) error {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return err
	}
	if err := validateOnlyJSONProperties(fields["digest"], "algorithm", "value"); err != nil {
		return fmt.Errorf("digest: %w", err)
	}
	if err := validateOnlyJSONProperties(fields["signature"], "algorithm", "key_id", "value"); err != nil {
		return fmt.Errorf("signature: %w", err)
	}
	if err := validateOnlyJSONProperties(fields["platform_metadata"], "schema_version", "plugin_type", "capabilities", "permission_ids", "compatibility_status"); err != nil {
		return fmt.Errorf("platform_metadata: %w", err)
	}
	var metadata struct { Capabilities []json.RawMessage `json:"capabilities"` }
	if err := json.Unmarshal(fields["platform_metadata"], &metadata); err != nil { return err }
	for i, capability := range metadata.Capabilities {
		if err := validateOnlyJSONProperties(capability, "id", "name", "description"); err != nil { return fmt.Errorf("platform_metadata.capabilities[%d]: %w", i, err) }
	}
	return nil
}

func validateOnlyJSONProperties(raw json.RawMessage, allowed ...string) error {
	var fields map[string]json.RawMessage
	if len(raw) == 0 || json.Unmarshal(raw, &fields) != nil || fields == nil {
		return errors.New("expected JSON object")
	}
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, key := range allowed {
		allowedSet[key] = struct{}{}
	}
	for key := range fields {
		if _, ok := allowedSet[key]; !ok {
			return fmt.Errorf("unknown property %q", key)
		}
	}
	return nil
}

func validatePlatformMetadata(metadata struct {
	SchemaVersion string `json:"schema_version"`
	PluginType string `json:"plugin_type"`
	Capabilities []domain.Capability `json:"capabilities"`
	PermissionIDs []string `json:"permission_ids"`
	CompatibilityStatus string `json:"compatibility_status"`
}) error {
	if metadata.SchemaVersion != "1.0.0" || (metadata.PluginType != "active" && metadata.PluginType != "passive") || len(metadata.Capabilities) > 256 || len(metadata.PermissionIDs) > 256 || !boundedNonEmpty(metadata.CompatibilityStatus, 64) { return errors.New("unsupported schema, plugin type, or collection/string limit") }
	seenCapabilities := make(map[string]struct{}, len(metadata.Capabilities))
	for _, capability := range metadata.Capabilities {
		if !boundedNonEmpty(capability.ID, 255) || !boundedNonEmpty(capability.Name, 128) || len([]rune(capability.Description)) > 1024 { return errors.New("invalid capability field") }
		if _, exists := seenCapabilities[capability.ID]; exists { return errors.New("duplicate capability ID") }; seenCapabilities[capability.ID] = struct{}{}
	}
	seenPermissions := make(map[string]struct{}, len(metadata.PermissionIDs))
	for _, permissionID := range metadata.PermissionIDs {
		if !boundedNonEmpty(permissionID, 255) { return errors.New("invalid permission ID") }
		if _, exists := seenPermissions[permissionID]; exists { return errors.New("duplicate permission ID") }; seenPermissions[permissionID] = struct{}{}
	}
	return nil
}

func boundedNonEmpty(value string, max int) bool { return strings.TrimSpace(value) != "" && len([]rune(value)) <= max }

func descriptorSigningPayload(raw []byte) ([]byte, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, err
	}
	if _, ok := fields["signature"]; !ok {
		return nil, ErrInvalidDescriptor
	}
	delete(fields, "signature")
	return json.Marshal(fields)
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

func rejectDuplicateKeys(raw []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	if err := consumeValue(decoder); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("trailing JSON content")
	}
	return nil
}

func consumeValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
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
			if err := consumeValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := consumeValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return ErrInvalidDescriptor
	}
}
