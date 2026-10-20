package credential

import (
	"bytes"
	"context"
	"testing"
)

type testKeySource struct {
	key []byte
}

func (s testKeySource) GetKey(ctx context.Context) ([]byte, error) {
	return s.key, nil
}

func TestNewSQLiteEncryptedStore(t *testing.T) {
	ctx := context.Background()

	key := []byte("01234567890123456789012345678901")

	keySource := testKeySource{
		key: key,
	}

	store, sqliteStore, err := NewSQLiteEncryptedStore(
		ctx,
		"file:factory-test?mode=memory&cache=shared",
		keySource,
	)
	if err != nil {
		t.Fatalf(
			"NewSQLiteEncryptedStore() failed: %v",
			err,
		)
	}

	defer sqliteStore.Close()

	credential := Credential{
		Ref:   "factory-credential-001",
		Value: []byte("factory-secret"),
	}

	if err := store.Set(ctx, credential); err != nil {
		t.Fatalf("Set() failed: %v", err)
	}

	// Verify raw SQLite does not contain plaintext.
	raw, err := sqliteStore.Get(ctx, credential.Ref)
	if err != nil {
		t.Fatalf("raw SQLite Get() failed: %v", err)
	}

	if bytes.Equal(raw.Value, credential.Value) {
		t.Fatal("factory-created store persisted plaintext")
	}

	// Verify encrypted store transparently decrypts.
	result, err := store.Get(ctx, credential.Ref)
	if err != nil {
		t.Fatalf("Get() failed: %v", err)
	}

	if !bytes.Equal(result.Value, credential.Value) {
		t.Fatalf(
			"unexpected decrypted value: got %q, want %q",
			string(result.Value),
			string(credential.Value),
		)
	}
}
