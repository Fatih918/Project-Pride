package execution

import (
	"bytes"
	"context"
	"testing"

	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
)

func TestCredentialResolverResolve(t *testing.T) {
	ctx := context.Background()

	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	secret := []byte("test-api-key")

	ref, err := service.Create(
		ctx,
		secret,
	)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	resolver := NewCredentialServiceResolver(service)

	result, err := resolver.Resolve(
		ctx,
		ref,
	)
	if err != nil {
		t.Fatalf(
			"Resolve() failed: %v",
			err,
		)
	}

	if !bytes.Equal(result, secret) {
		t.Fatalf(
			"unexpected credential value: got %q, want %q",
			string(result),
			string(secret),
		)
	}
}

func TestCredentialResolverRejectsEmptyReference(t *testing.T) {
	ctx := context.Background()

	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	resolver := NewCredentialServiceResolver(service)

	_, err := resolver.Resolve(
		ctx,
		"",
	)
	if err == nil {
		t.Fatal(
			"expected empty credential reference to be rejected",
		)
	}
}

func TestCredentialResolverNotFound(t *testing.T) {
	ctx := context.Background()

	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	resolver := NewCredentialServiceResolver(service)

	_, err := resolver.Resolve(
		ctx,
		"cred_does_not_exist",
	)
	if err == nil {
		t.Fatal(
			"expected missing credential to return an error",
		)
	}
}

func TestCredentialResolverImplementsContract(
	t *testing.T,
) {
	store := credential.NewMemoryStore()
	service := credential.NewService(store)

	var _ executioncontract.CredentialResolver = NewCredentialServiceResolver(service)
}
