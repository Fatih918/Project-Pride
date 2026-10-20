package credential

import (
	"bytes"
	"context"
	"testing"
)

func TestCredentialServiceWithEncryptedSQLite(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	sqliteStore, err := NewSQLiteStore(
		ctx,
		"file:service-sqlite-test?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	encryptedStore := NewEncryptedStore(
		sqliteStore,
		encryptor,
	)

	service := NewService(encryptedStore)

	secret := []byte("real-api-key-test-value")

	// Create credential.
	ref, err := service.Create(ctx, secret)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if ref == "" {
		t.Fatal("Create() returned empty credential reference")
	}

	// Inspect the underlying SQLite storage directly.
	raw, err := sqliteStore.Get(ctx, ref)
	if err != nil {
		t.Fatalf("raw SQLite Get() failed: %v", err)
	}

	// Plaintext must never be stored.
	if bytes.Equal(raw.Value, secret) {
		t.Fatal("SQLite contains plaintext credential")
	}

	// Retrieve through the service.
	result, err := service.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if result.Ref != ref {
		t.Fatalf(
			"unexpected credential ref: got %q, want %q",
			result.Ref,
			ref,
		)
	}

	if !bytes.Equal(result.Value, secret) {
		t.Fatalf(
			"unexpected credential value: got %q, want %q",
			string(result.Value),
			string(secret),
		)
	}
}
func TestCredentialServiceUpdateKeepsReference(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	sqliteStore, err := NewSQLiteStore(
		ctx,
		"file:rotation-test?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	encryptedStore := NewEncryptedStore(
		sqliteStore,
		encryptor,
	)

	service := NewService(encryptedStore)

	oldSecret := []byte("old-api-key")
	newSecret := []byte("new-api-key")

	ref, err := service.Create(ctx, oldSecret)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := service.Update(
		ctx,
		ref,
		newSecret,
	); err != nil {
		t.Fatalf("Update() failed: %v", err)
	}

	// Reference must remain unchanged.
	result, err := service.Get(ctx, ref)
	if err != nil {
		t.Fatalf("Get() after Update() failed: %v", err)
	}

	if result.Ref != ref {
		t.Fatalf(
			"credential reference changed: got %q, want %q",
			result.Ref,
			ref,
		)
	}

	// New secret must be returned.
	if !bytes.Equal(result.Value, newSecret) {
		t.Fatalf(
			"unexpected updated secret: got %q, want %q",
			string(result.Value),
			string(newSecret),
		)
	}

	// Old secret must no longer be returned.
	if bytes.Equal(result.Value, oldSecret) {
		t.Fatal("old credential secret is still active")
	}

	// Underlying SQLite must contain ciphertext.
	raw, err := sqliteStore.Get(ctx, ref)
	if err != nil {
		t.Fatalf("raw SQLite Get() failed: %v", err)
	}

	if bytes.Equal(raw.Value, newSecret) {
		t.Fatal("SQLite contains plaintext updated credential")
	}
}
func TestCredentialServiceDelete(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	sqliteStore, err := NewSQLiteStore(
		ctx,
		"file:delete-test?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	encryptedStore := NewEncryptedStore(
		sqliteStore,
		encryptor,
	)

	service := NewService(encryptedStore)

	ref, err := service.Create(
		ctx,
		[]byte("secret-to-delete"),
	)
	if err != nil {
		t.Fatalf("Create() failed: %v", err)
	}

	if err := service.Delete(ctx, ref); err != nil {
		t.Fatalf("Delete() failed: %v", err)
	}

	_, err = service.Get(ctx, ref)

	if err == nil {
		t.Fatal("expected deleted credential to be unavailable")
	}
}
