package artifactverify

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type memoryTrustSource struct{ raw []byte }
func (s memoryTrustSource) ReadTrustConfiguration(context.Context)([]byte,error){ return append([]byte(nil),s.raw...),nil }

type memorySequenceStore struct{ sequence uint64; digest [32]byte }
func (s *memorySequenceStore) AcceptTrustConfiguration(_ context.Context, sequence uint64, digest [32]byte)error{
	if sequence<s.sequence || (sequence==s.sequence && s.sequence>0 && digest!=s.digest) { return errors.New("rollback or equivocation") }
	if sequence>s.sequence { s.sequence=sequence; s.digest=digest }
	return nil
}

func TestTrustResolverAuthenticatesBootstrapAndResolvesMarketplaceKey(t *testing.T) {
	bootstrapPublic,bootstrapPrivate,err:=ed25519.GenerateKey(rand.Reader); if err!=nil { t.Fatal(err) }
	marketPublic,_,err:=ed25519.GenerateKey(rand.Reader); if err!=nil { t.Fatal(err) }
	raw:=signedTrustConfig(t,bootstrapPrivate,marketPublic,1)
	store:=&memorySequenceStore{}
	resolver,err:=NewTrustResolver(memoryTrustSource{raw},store,BootstrapKeys{"distribution-root":bootstrapPublic},JCSCanonicalizer{})
	if err!=nil { t.Fatal(err) }
	now:=time.Date(2027,1,1,0,0,0,0,time.UTC)
	resolver.now=func()time.Time{return now}
	key,err:=resolver.ResolveDescriptorKey(context.Background(),"marketplace-key",now)
	if err!=nil { t.Fatal(err) }
	if len(key.PublicKey)!=ed25519.PublicKeySize || key.Revoked || store.sequence!=1 {
		t.Fatalf("unexpected trusted key or sequence state: %#v, %d",key,store.sequence)
	}
}

func TestTrustResolverRejectsRollback(t *testing.T) {
	bootstrapPublic,bootstrapPrivate,err:=ed25519.GenerateKey(rand.Reader); if err!=nil { t.Fatal(err) }
	marketPublic,_,err:=ed25519.GenerateKey(rand.Reader); if err!=nil { t.Fatal(err) }
	store:=&memorySequenceStore{}
	now:=time.Date(2027,1,1,0,0,0,0,time.UTC)
	for _,sequence:=range []uint64{2,1} {
		raw:=signedTrustConfig(t,bootstrapPrivate,marketPublic,sequence)
		resolver,err:=NewTrustResolver(memoryTrustSource{raw},store,BootstrapKeys{"distribution-root":bootstrapPublic},JCSCanonicalizer{})
		if err!=nil { t.Fatal(err) }
		resolver.now=func()time.Time{return now}
		_,err=resolver.ResolveDescriptorKey(context.Background(),"marketplace-key",now)
		if sequence==2 && err!=nil { t.Fatal(err) }
		if sequence==1 && err==nil { t.Fatal("expected lower trust sequence to be rejected") }
	}
}

func signedTrustConfig(t *testing.T, signer ed25519.PrivateKey, marketKey ed25519.PublicKey, sequence uint64) []byte {
	t.Helper()
	fields:=map[string]any{
		"$schema":trustSchema,"schema_version":"1.0.0","sequence":sequence,
		"issued_at":"2026-12-01T00:00:00Z","expires_at":"2027-12-01T00:00:00Z",
		"keys":[]any{map[string]any{
			"key_id":"marketplace-key","algorithm":"Ed25519","public_key":base64.StdEncoding.EncodeToString(marketKey),
			"valid_from":"2026-12-01T00:00:00Z","valid_until":"2027-06-01T00:00:00Z","status":"active",
		}},
	}
	payload,err:=json.Marshal(fields); if err!=nil { t.Fatal(err) }
	canonical,err:=(JCSCanonicalizer{}).Canonicalize(payload); if err!=nil { t.Fatal(err) }
	fields["signature"]=map[string]string{"algorithm":"Ed25519","key_id":"distribution-root","value":base64.StdEncoding.EncodeToString(ed25519.Sign(signer,canonical))}
	raw,err:=json.Marshal(fields); if err!=nil { t.Fatal(err) }
	return raw
}
