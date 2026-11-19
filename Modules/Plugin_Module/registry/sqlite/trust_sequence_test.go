package sqlite

import (
	"context"
	"crypto/sha256"
	"errors"
	"path/filepath"
	"testing"
)

func TestTrustSequenceStoreAcceptsMonotonicSequenceAndExactReplay(t *testing.T) {
	repository, databasePath := openTestRepository(t)
	ctx := context.Background()
	first := sha256.Sum256([]byte("trust-config-1"))
	second := sha256.Sum256([]byte("trust-config-2"))

	if err := repository.AcceptTrustConfiguration(ctx, 1, first); err != nil {
		t.Fatal(err)
	}
	if err := repository.AcceptTrustConfiguration(ctx, 1, first); err != nil {
		t.Fatalf("exact replay must be idempotent: %v", err)
	}
	if err := repository.AcceptTrustConfiguration(ctx, 2, second); err != nil {
		t.Fatalf("higher sequence must be accepted: %v", err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(ctx, databasePath)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	sequence, digest, err := reopened.CurrentTrustConfigurationSequence(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sequence != 2 || digest != second {
		t.Fatalf("persisted trust state = (%d, %x), want (2, %x)", sequence, digest, second)
	}
}

func TestTrustSequenceStoreRejectsRollbackAndEquivocation(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx := context.Background()
	accepted := sha256.Sum256([]byte("accepted"))
	other := sha256.Sum256([]byte("different"))

	if err := repository.AcceptTrustConfiguration(ctx, 7, accepted); err != nil {
		t.Fatal(err)
	}
	if err := repository.AcceptTrustConfiguration(ctx, 6, other); !errors.Is(err, ErrTrustSequenceRejected) {
		t.Fatalf("rollback error = %v", err)
	}
	if err := repository.AcceptTrustConfiguration(ctx, 7, other); !errors.Is(err, ErrTrustSequenceRejected) {
		t.Fatalf("same-sequence equivocation error = %v", err)
	}
	sequence, digest, err := repository.CurrentTrustConfigurationSequence(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if sequence != 7 || digest != accepted {
		t.Fatalf("rejected update mutated trust state: (%d, %x)", sequence, digest)
	}
}

func TestTrustSequenceStoreRejectsInvalidInputAndCanceledContext(t *testing.T) {
	repository, _ := openTestRepository(t)
	digest := sha256.Sum256([]byte("config"))
	if err := repository.AcceptTrustConfiguration(context.Background(), 0, digest); !errors.Is(err, ErrTrustSequenceRejected) {
		t.Fatalf("zero sequence error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.AcceptTrustConfiguration(ctx, 1, digest); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled context error = %v", err)
	}
}

func TestTrustSequenceSchemaMigrationFromVersionTwo(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin-registry-v2.sqlite")
	repository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec("DROP TABLE plugin_trust_configuration_state"); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.Exec("PRAGMA user_version = 2"); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	migrated, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer migrated.Close()
	var version int
	if err := migrated.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 3 {
		t.Fatalf("schema version = %d, want 3", version)
	}
	if err := migrated.AcceptTrustConfiguration(context.Background(), 1, sha256.Sum256([]byte("v3"))); err != nil {
		t.Fatalf("trust table unavailable after migration: %v", err)
	}
}
