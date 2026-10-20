package credential

import (
	"context"
	"strings"
	"testing"
)

func TestCredentialServiceCreate(t *testing.T) {
	ctx := context.Background()

	memoryStore := NewMemoryStore()

	service := NewService(memoryStore)

	secret := []byte("test-api-key")

	ref, err := service.Create(ctx, secret)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if ref == "" {
		t.Fatal("Create() returned empty credential reference")
	}

	if !strings.HasPrefix(ref, "cred_") {
		t.Fatalf(
			"unexpected credential reference: %q",
			ref,
		)
	}

	credential, err := memoryStore.Get(ctx, ref)
	if err != nil {
		t.Fatalf(
			"failed to retrieve created credential: %v",
			err,
		)
	}

	if string(credential.Value) != string(secret) {
		t.Fatalf(
			"unexpected secret: got %q, want %q",
			string(credential.Value),
			string(secret),
		)
	}
}

func TestCredentialServiceGeneratesUniqueReferences(t *testing.T) {
	ctx := context.Background()

	memoryStore := NewMemoryStore()
	service := NewService(memoryStore)

	first, err := service.Create(
		ctx,
		[]byte("secret-one"),
	)
	if err != nil {
		t.Fatalf("first Create() failed: %v", err)
	}

	second, err := service.Create(
		ctx,
		[]byte("secret-two"),
	)
	if err != nil {
		t.Fatalf("second Create() failed: %v", err)
	}

	if first == second {
		t.Fatalf(
			"credential references must be unique: %q",
			first,
		)
	}
}

func TestCredentialServiceRejectsEmptySecret(t *testing.T) {
	ctx := context.Background()

	memoryStore := NewMemoryStore()
	service := NewService(memoryStore)

	_, err := service.Create(ctx, nil)
	if err == nil {
		t.Fatal("expected empty credential secret to be rejected")
	}
}
func TestMemoryStoreDoesNotExposeCredentialValue(t *testing.T) {
	ctx := context.Background()

	store := NewMemoryStore()

	original := Credential{
		Ref:   "cred_test_001",
		Value: []byte("original-secret"),
	}

	if err := store.Set(ctx, original); err != nil {
		t.Fatalf(
			"Set() failed: %v",
			err,
		)
	}

	result, err := store.Get(
		ctx,
		original.Ref,
	)
	if err != nil {
		t.Fatalf(
			"Get() failed: %v",
			err,
		)
	}

	result.Value[0] = 'X'

	again, err := store.Get(
		ctx,
		original.Ref,
	)
	if err != nil {
		t.Fatalf(
			"second Get() failed: %v",
			err,
		)
	}

	if string(again.Value) != string(original.Value) {
		t.Fatalf(
			"MemoryStore exposed internal credential value: got %q, want %q",
			string(again.Value),
			string(original.Value),
		)
	}
}
