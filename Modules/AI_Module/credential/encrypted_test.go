package credential

import (
	"bytes"
	"context"
	"testing"
)

func TestEncryptedSQLiteStore(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		t.Fatalf("failed to create encryptor: %v", err)
	}

	sqliteStore, err := NewSQLiteStore(
		ctx,
		"file:encrypted-credential-test?mode=memory&cache=shared",
	)
	if err != nil {
		t.Fatalf("failed to create SQLite store: %v", err)
	}
	defer sqliteStore.Close()

	store := NewEncryptedStore(
		sqliteStore,
		encryptor,
	)

	plaintext := []byte("test-api-key-123456")

	credential := Credential{
		Ref:   "cred-sqlite-encrypted-001",
		Value: plaintext,
	}

	// Store credential through encrypted boundary.
	if err := store.Set(ctx, credential); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Inspect raw SQLite storage.
	raw, err := sqliteStore.Get(ctx, credential.Ref)
	if err != nil {
		t.Fatalf("raw SQLite Get() failed: %v", err)
	}

	// Plaintext must never exist in persistent storage.
	if bytes.Equal(raw.Value, plaintext) {
		t.Fatal("SQLite contains plaintext credential")
	}

	// EncryptedStore must transparently decrypt.
	result, err := store.Get(ctx, credential.Ref)
	if err != nil {
		t.Fatalf("EncryptedStore Get() failed: %v", err)
	}

	if result.Ref != credential.Ref {
		t.Fatalf(
			"unexpected credential ref: got %q, want %q",
			result.Ref,
			credential.Ref,
		)
	}

	if !bytes.Equal(result.Value, plaintext) {
		t.Fatalf(
			"decrypted credential mismatch: got %q, want %q",
			string(result.Value),
			string(plaintext),
		)
	}
}
