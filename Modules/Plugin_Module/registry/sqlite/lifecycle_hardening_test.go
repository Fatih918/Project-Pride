package sqlite

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/lifecycle"
)

func setupLifecycleHardeningRepository(t *testing.T, pluginID string) (*Repository, *LifecycleRepository, string, time.Time, func(time.Duration)) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin-registry.sqlite")
	repository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })

	plugin := testPlugin(pluginID)
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	if _, err := repository.db.ExecContext(context.Background(), `INSERT INTO installed_plugin_versions (
		plugin_id, version, artifact_path, source_reference, digest_algorithm, digest_value,
		verification_policy, verification_evidence_ref, manifest_json, installed_at, selected
	) VALUES (?, ?, 'installed/' || ? || '/1.2.3', 'test:source', 'SHA-256', 'digest', 'test-policy', NULL, '{}', ?, 1)`,
		string(plugin.ID), plugin.Manifest.Version, string(plugin.ID), formatLifeTime(now)); err != nil {
		t.Fatal(err)
	}
	current := now
	adapter := newLifecycleRepository(repository.db, func() time.Time { return current })
	advance := func(d time.Duration) { current = current.Add(d) }
	return repository, adapter, path, now, advance
}

func TestLifecycleRepositorySerializesReservationsAcrossDatabaseHandles(t *testing.T) {
	firstRepository, firstAdapter, path, now, _ := setupLifecycleHardeningRepository(t, "plugin.lifecycle.concurrent")
	secondRepository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer secondRepository.Close()
	secondAdapter := newLifecycleRepository(secondRepository.db, func() time.Time { return now })

	start := make(chan struct{})
	type outcome struct {
		reservation lifecycle.OperationReservation
		err         error
	}
	outcomes := make(chan outcome, 2)
	var wg sync.WaitGroup
	for i, adapter := range []*LifecycleRepository{firstAdapter, secondAdapter} {
		wg.Add(1)
		go func(i int, adapter *LifecycleRepository) {
			defer wg.Done()
			<-start
			opID := fmt.Sprintf("lifecycle-concurrent-%d", i)
			reservation, err := adapter.ReserveOperation(
				context.Background(),
				lifecycleTestOperation(domain.PluginID("plugin.lifecycle.concurrent"), opID, now),
				lifecycleFingerprint(opID),
			)
			outcomes <- outcome{reservation: reservation, err: err}
		}(i, adapter)
	}
	close(start)
	wg.Wait()
	close(outcomes)

	succeeded, conflicted := 0, 0
	for result := range outcomes {
		switch {
		case result.err == nil:
			succeeded++
		case errors.Is(result.err, lifecycle.ErrPluginOperationConflict):
			conflicted++
		default:
			t.Fatalf("unexpected reservation error: %v", result.err)
		}
	}
	if succeeded != 1 || conflicted != 1 {
		t.Fatalf("cross-handle reservation results: succeeded=%d conflicted=%d; want 1 each", succeeded, conflicted)
	}
	_ = firstRepository
}

func TestLifecycleRepositoryFencingCounterSurvivesReopen(t *testing.T) {
	repository, adapter, path, now, _ := setupLifecycleHardeningRepository(t, "plugin.lifecycle.reopen")
	first, err := adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.reopen", "lifecycle-before-reopen", now),
		lifecycleFingerprint("before-reopen"))
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.ReleaseLease(context.Background(), first.Operation.OperationID, first.Lease); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	next := newLifecycleRepository(reopened.db, func() time.Time { return now })
	second, err := next.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.reopen", "lifecycle-after-reopen", now),
		lifecycleFingerprint("after-reopen"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Lease.FencingToken <= first.Lease.FencingToken {
		t.Fatalf("fencing token regressed after reopen: first=%d second=%d", first.Lease.FencingToken, second.Lease.FencingToken)
	}
}

func TestLifecycleRepositoryFailsClosedOnFencingTokenExhaustion(t *testing.T) {
	_, adapter, _, now, _ := setupLifecycleHardeningRepository(t, "plugin.lifecycle.exhaustion")
	first, err := adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.exhaustion", "lifecycle-token-seed", now),
		lifecycleFingerprint("token-seed"))
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.ReleaseLease(context.Background(), first.Operation.OperationID, first.Lease); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.db.ExecContext(context.Background(),
		"UPDATE plugin_lifecycle_fencing_tokens SET last_token = ? WHERE plugin_id = ?",
		int64(^uint64(0)>>1), "plugin.lifecycle.exhaustion"); err != nil {
		t.Fatal(err)
	}
	_, err = adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.exhaustion", "lifecycle-token-overflow", now),
		lifecycleFingerprint("token-overflow"))
	if !errors.Is(err, lifecycle.ErrFencingTokenExhausted) {
		t.Fatalf("fencing token exhaustion error = %v", err)
	}
}

func TestLifecycleRepositoryOldLeaseCannotReleaseNewOwner(t *testing.T) {
	_, adapter, _, now, advance := setupLifecycleHardeningRepository(t, "plugin.lifecycle.release-owner")
	old, err := adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.release-owner", "lifecycle-old-owner", now),
		lifecycleFingerprint("old-owner"))
	if err != nil {
		t.Fatal(err)
	}
	advance(121 * time.Second)
	currentTime := now.Add(121 * time.Second)
	current, err := adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation("plugin.lifecycle.release-owner", "lifecycle-new-owner", currentTime),
		lifecycleFingerprint("new-owner"))
	if err != nil {
		t.Fatal(err)
	}
	if err := adapter.ReleaseLease(context.Background(), old.Operation.OperationID, old.Lease); err != nil {
		t.Fatal(err)
	}
	decision := lifecycle.AuthorizationDecision{
		Allowed: true, DecisionID: "decision-new-owner", PolicyRevision: "policy-v1",
		EvaluatedAt: currentTime, ReasonCode: "ALLOW",
	}
	if err := adapter.RecordAuthorization(context.Background(), current.Operation.OperationID, current.Lease, decision); err != nil {
		t.Fatalf("old release removed the new owner's lease: %v", err)
	}
}

func TestLifecycleRepositoryFencedMutationsRejectExpiredLease(t *testing.T) {
	tests := []struct {
		name  string
		apply func(*testing.T, *LifecycleRepository, string, lifecycle.OperationLease, time.Time, func(time.Duration))
	}{
		{
			name: "record authorization",
			apply: func(t *testing.T, repo *LifecycleRepository, id string, lease lifecycle.OperationLease, now time.Time, advance func(time.Duration)) {
				advance(121 * time.Second)
				err := repo.RecordAuthorization(context.Background(), id, lease, lifecycle.AuthorizationDecision{
					Allowed: true, DecisionID: "decision-expired", PolicyRevision: "policy-v1",
					EvaluatedAt: now, ReasonCode: "ALLOW",
				})
				if !errors.Is(err, lifecycle.ErrLeaseLost) {
					t.Fatalf("RecordAuthorization error = %v", err)
				}
			},
		},
		{
			name: "begin apply",
			apply: func(t *testing.T, repo *LifecycleRepository, id string, lease lifecycle.OperationLease, now time.Time, advance func(time.Duration)) {
				decision := lifecycle.AuthorizationDecision{Allowed: true, DecisionID: "decision-expired", PolicyRevision: "policy-v1", EvaluatedAt: now, ReasonCode: "ALLOW"}
				if err := repo.RecordAuthorization(context.Background(), id, lease, decision); err != nil {
					t.Fatal(err)
				}
				advance(121 * time.Second)
				if err := repo.BeginApply(context.Background(), id, lease); !errors.Is(err, lifecycle.ErrLeaseLost) {
					t.Fatalf("BeginApply error = %v", err)
				}
			},
		},
		{
			name: "commit desired state",
			apply: func(t *testing.T, repo *LifecycleRepository, id string, lease lifecycle.OperationLease, now time.Time, advance func(time.Duration)) {
				decision := lifecycle.AuthorizationDecision{Allowed: true, DecisionID: "decision-expired", PolicyRevision: "policy-v1", EvaluatedAt: now, ReasonCode: "ALLOW"}
				if err := repo.RecordAuthorization(context.Background(), id, lease, decision); err != nil {
					t.Fatal(err)
				}
				if err := repo.BeginApply(context.Background(), id, lease); err != nil {
					t.Fatal(err)
				}
				advance(121 * time.Second)
				if _, err := repo.CommitDesiredState(context.Background(), id, lease, domain.LifecycleDesiredStateEnabled); !errors.Is(err, lifecycle.ErrLeaseLost) {
					t.Fatalf("CommitDesiredState error = %v", err)
				}
			},
		},
		{
			name: "fail operation",
			apply: func(t *testing.T, repo *LifecycleRepository, id string, lease lifecycle.OperationLease, now time.Time, advance func(time.Duration)) {
				advance(121 * time.Second)
				if err := repo.FailOperation(context.Background(), id, lease, "TEST_FAILURE"); !errors.Is(err, lifecycle.ErrLeaseLost) {
					t.Fatalf("FailOperation error = %v", err)
				}
			},
		},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, repo, _, now, advance := setupLifecycleHardeningRepository(t, fmt.Sprintf("plugin.lifecycle.expired.%d", i))
			id := fmt.Sprintf("lifecycle-expired-%d", i)
			reservation, err := repo.ReserveOperation(context.Background(),
				lifecycleTestOperation(domain.PluginID(fmt.Sprintf("plugin.lifecycle.expired.%d", i)), id, now),
				lifecycleFingerprint(id))
			if err != nil {
				t.Fatal(err)
			}
			test.apply(t, repo, id, reservation.Lease, now, advance)
		})
	}
}

func TestLifecycleRepositoryCommitFailureRollsBackDesiredStateAndOperation(t *testing.T) {
	_, adapter, pluginID, now, _ := setupLifecycleHardeningRepository(t, "plugin.lifecycle.rollback")
	reservation, err := adapter.ReserveOperation(context.Background(),
		lifecycleTestOperation(pluginID, "lifecycle-rollback", now),
		lifecycleFingerprint("rollback"))
	if err != nil {
		t.Fatal(err)
	}
	authorizeAndApply(t, adapter, "lifecycle-rollback", reservation.Lease, now)

	if _, err := adapter.db.ExecContext(context.Background(), `CREATE TRIGGER reject_lifecycle_success
		BEFORE UPDATE OF phase ON plugin_lifecycle_operations
		WHEN NEW.phase = 'SUCCEEDED'
		BEGIN
			SELECT RAISE(ABORT, 'injected lifecycle commit failure');
		END`); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CommitDesiredState(context.Background(), "lifecycle-rollback", reservation.Lease, domain.LifecycleDesiredStateEnabled); err == nil {
		t.Fatal("expected injected commit failure")
	}

	var records int
	if err := adapter.db.QueryRowContext(context.Background(),
		"SELECT COUNT(*) FROM plugin_lifecycle_records WHERE plugin_id = ?", string(pluginID)).Scan(&records); err != nil {
		t.Fatal(err)
	}
	if records != 0 {
		t.Fatalf("desired-state row survived failed transaction: count=%d", records)
	}
	op, err := adapter.GetOperation(context.Background(), "lifecycle-rollback")
	if err != nil {
		t.Fatal(err)
	}
	if op.Phase != domain.LifecycleOperationPhaseApplying {
		t.Fatalf("operation phase after rollback = %v, want APPLYING", op.Phase)
	}
}

