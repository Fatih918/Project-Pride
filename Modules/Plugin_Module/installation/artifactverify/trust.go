package artifactverify

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const trustSchema = "urn:pride:plugin-trust-configuration:1"

var ErrInvalidTrustConfiguration = errors.New("invalid Plugin trust configuration")

// TrustConfigurationSource retrieves the current trust configuration bytes.
// Network transport and endpoint policy belong to its adapter.
type TrustConfigurationSource interface {
	ReadTrustConfiguration(context.Context) ([]byte, error)
}

// TrustSequenceStore atomically accepts a sequence/digest pair. It must reject
// lower sequences and same-sequence configurations with a different digest;
// repeating the exact accepted pair must be idempotent.
type TrustSequenceStore interface {
	AcceptTrustConfiguration(context.Context, uint64, [32]byte) error
}

// BootstrapKeys are public keys provisioned by the PRIDE distribution. They
// must never be populated from the remote trust configuration itself.
type BootstrapKeys map[string]ed25519.PublicKey

type trustConfiguration struct {
	Schema string `json:"$schema"`
	SchemaVersion string `json:"schema_version"`
	Sequence uint64 `json:"sequence"`
	IssuedAt string `json:"issued_at"`
	ExpiresAt string `json:"expires_at"`
	Keys []trustKeyRecord `json:"keys"`
	Signature struct {
		Algorithm string `json:"algorithm"`
		KeyID string `json:"key_id"`
		Value string `json:"value"`
	} `json:"signature"`
}

type trustKeyRecord struct {
	KeyID string `json:"key_id"`
	Algorithm string `json:"algorithm"`
	PublicKey string `json:"public_key"`
	ValidFrom string `json:"valid_from"`
	ValidUntil string `json:"valid_until"`
	Status string `json:"status"`
}

// TrustResolver authenticates remote trust configuration directly with an
// embedded bootstrap key, persists rollback protection, and resolves active
// Marketplace descriptor keys.
type TrustResolver struct {
	source TrustConfigurationSource
	sequences TrustSequenceStore
	bootstrap BootstrapKeys
	canonicalizer Canonicalizer
	now func() time.Time
}

func NewTrustResolver(source TrustConfigurationSource, sequences TrustSequenceStore, bootstrap BootstrapKeys, canonicalizer Canonicalizer) (*TrustResolver, error) {
	if source == nil || sequences == nil || len(bootstrap) == 0 || canonicalizer == nil {
		return nil, errors.New("trust resolver dependencies are required")
	}
	keys := make(BootstrapKeys, len(bootstrap))
	for id, key := range bootstrap {
		if strings.TrimSpace(id) == "" || len(key) != ed25519.PublicKeySize {
			return nil, ErrInvalidTrustConfiguration
		}
		keys[id] = append(ed25519.PublicKey(nil), key...)
	}
	return &TrustResolver{source:source,sequences:sequences,bootstrap:keys,canonicalizer:canonicalizer,now:func()time.Time{return time.Now().UTC()}},nil
}

func (r *TrustResolver) ResolveDescriptorKey(ctx context.Context, keyID string, now time.Time) (TrustedKey,error) {
	raw,err:=r.source.ReadTrustConfiguration(ctx)
	if err!=nil { return TrustedKey{},err }
	if len(raw)==0 || len(raw)>maxDescriptorBytes { return TrustedKey{},ErrInvalidTrustConfiguration }
	if err:=rejectDuplicateKeys(raw); err!=nil { return TrustedKey{},fmt.Errorf("%w: %v",ErrInvalidTrustConfiguration,err) }
	var config trustConfiguration
	if err:=json.Unmarshal(raw,&config); err!=nil { return TrustedKey{},fmt.Errorf("%w: %v",ErrInvalidTrustConfiguration,err) }
	if config.Schema!=trustSchema || config.SchemaVersion!="1.0.0" || config.Sequence==0 ||
		config.Signature.Algorithm!="Ed25519" || config.Signature.KeyID=="" {
		return TrustedKey{},ErrInvalidTrustConfiguration
	}
	issued,err:=time.Parse(time.RFC3339,config.IssuedAt); if err!=nil { return TrustedKey{},ErrInvalidTrustConfiguration }
	expires,err:=time.Parse(time.RFC3339,config.ExpiresAt); if err!=nil || !expires.After(issued) || now.Before(issued) || !now.Before(expires) {
		return TrustedKey{},ErrInvalidTrustConfiguration
	}
	bootstrapKey,ok:=r.bootstrap[config.Signature.KeyID]
	if !ok { return TrustedKey{},ErrInvalidTrustConfiguration }
	payload,err:=descriptorSigningPayload(raw); if err!=nil { return TrustedKey{},fmt.Errorf("%w: %v",ErrInvalidTrustConfiguration,err) }
	canonical,err:=r.canonicalizer.Canonicalize(payload); if err!=nil { return TrustedKey{},err }
	signature,err:=base64.StdEncoding.Strict().DecodeString(config.Signature.Value)
	if err!=nil || len(signature)!=ed25519.SignatureSize || !ed25519.Verify(bootstrapKey,canonical,signature) {
		return TrustedKey{},ErrInvalidTrustConfiguration
	}
	resolved:=make(map[string]TrustedKey,len(config.Keys))
	revoked:=make(map[string]bool,len(config.Keys))
	for _,record:=range config.Keys {
		if record.KeyID=="" || record.Algorithm!="Ed25519" || (record.Status!="active" && record.Status!="revoked") {
			return TrustedKey{},ErrInvalidTrustConfiguration
		}
		if _,exists:=resolved[record.KeyID]; exists { return TrustedKey{},ErrInvalidTrustConfiguration }
		publicKey,err:=base64.StdEncoding.Strict().DecodeString(record.PublicKey)
		if err!=nil || len(publicKey)!=ed25519.PublicKeySize { return TrustedKey{},ErrInvalidTrustConfiguration }
		validFrom,err:=time.Parse(time.RFC3339,record.ValidFrom); if err!=nil { return TrustedKey{},ErrInvalidTrustConfiguration }
		validUntil,err:=time.Parse(time.RFC3339,record.ValidUntil); if err!=nil || !validUntil.After(validFrom) { return TrustedKey{},ErrInvalidTrustConfiguration }
		resolved[record.KeyID]=TrustedKey{PublicKey:ed25519.PublicKey(publicKey),ValidFrom:validFrom,ValidUntil:validUntil}
		revoked[record.KeyID]=record.Status=="revoked"
	}
	digest:=sha256.Sum256(raw)
	if err:=r.sequences.AcceptTrustConfiguration(ctx,config.Sequence,digest); err!=nil {
		return TrustedKey{},fmt.Errorf("%w: sequence rollback/equivocation: %v",ErrInvalidTrustConfiguration,err)
	}
	key,ok:=resolved[keyID]
	if !ok || revoked[keyID] { return TrustedKey{},ErrUntrustedKey }
	return key,nil
}
