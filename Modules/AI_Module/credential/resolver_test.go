package credential

import (
	"bytes"
	"context"
	"testing"
)

func TestResolverResolve(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)

	expectedSecret := []byte("test-api-key")

	ref, err := service.Create(ctx, expectedSecret)
	if err != nil {
		t.Fatalf(
			"Create() failed: %v",
			err,
		)
	}

	if ref == "" {
		t.Fatal(
			"Create() returned empty credential reference",
		)
	}

	resolver := NewResolver(service)

	result, err := resolver.Resolve(ctx, ref)
	if err != nil {
		t.Fatalf(
			"Resolve() failed: %v",
			err,
		)
	}

	if !bytes.Equal(result, expectedSecret) {
		t.Fatalf(
			"unexpected credential value: got %q, want %q",
			string(result),
			string(expectedSecret),
		)
	}
}

func TestResolverRejectsNilResolver(t *testing.T) {
	ctx := context.Background()

	var resolver *Resolver

	_, err := resolver.Resolve(
		ctx,
		"cred_test",
	)
	if err == nil {
		t.Fatal(
			"expected nil resolver error",
		)
	}
}

func TestResolverRejectsNilService(t *testing.T) {
	ctx := context.Background()

	resolver := NewResolver(nil)

	_, err := resolver.Resolve(
		ctx,
		"cred_test",
	)
	if err == nil {
		t.Fatal(
			"expected nil service error",
		)
	}
}

func TestResolverRejectsEmptyReference(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)
	resolver := NewResolver(service)

	_, err := resolver.Resolve(
		ctx,
		"",
	)
	if err == nil {
		t.Fatal(
			"expected empty credential reference error",
		)
	}
}

func TestResolverReturnsNotFound(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()
	service := NewService(store)
	resolver := NewResolver(service)

	_, err := resolver.Resolve(
		ctx,
		"cred_missing",
	)
	if err == nil {
		t.Fatal(
			"expected credential not found error",
		)
	}
}

func TestResolverRejectsEmptyCredential(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	// Bypass Service.Create(), because Service itself
	// already rejects empty secrets.
	err := store.Set(
		ctx,
		Credential{
			Ref:   "cred_empty",
			Value: nil,
		},
	)
	if err != nil {
		t.Fatalf(
			"store.Set() failed: %v",
			err,
		)
	}

	service := NewService(store)
	resolver := NewResolver(service)

	_, err = resolver.Resolve(
		ctx,
		"cred_empty",
	)
	if err == nil {
		t.Fatal(
			"expected empty resolved credential error",
		)
	}
}
