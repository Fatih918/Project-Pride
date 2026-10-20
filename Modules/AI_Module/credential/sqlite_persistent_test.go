package credential

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteStorePersistsAcrossConnections(t *testing.T) {
	ctx := context.Background()

	databasePath := filepath.Join(
		t.TempDir(),
		"credentials.db",
	)

	dsn := "file:" + databasePath

	credential := Credential{
		Ref:   "persistent-credential-001",
		Value: []byte("persistent-secret"),
	}

	// First connection.
	store, err := NewSQLiteStore(ctx, dsn)
	if err != nil {
		t.Fatalf(
			"failed to create first SQLite store: %v",
			err,
		)
	}

	if err := store.Set(ctx, credential); err != nil {
		store.Close()

		t.Fatalf(
			"Set() failed: %v",
			err,
		)
	}

	if err := store.Close(); err != nil {
		t.Fatalf(
			"first Close() failed: %v",
			err,
		)
	}

	// Second connection.
	store, err = NewSQLiteStore(ctx, dsn)
	if err != nil {
		t.Fatalf(
			"failed to create second SQLite store: %v",
			err,
		)
	}
	defer store.Close()

	// Credential must survive the first connection being closed.
	result, err := store.Get(ctx, credential.Ref)
	if err != nil {
		t.Fatalf(
			"Get() after reopening database failed: %v",
			err,
		)
	}

	if string(result.Value) != string(credential.Value) {
		t.Fatalf(
			"unexpected persisted value: got %q, want %q",
			string(result.Value),
			string(credential.Value),
		)
	}
}
