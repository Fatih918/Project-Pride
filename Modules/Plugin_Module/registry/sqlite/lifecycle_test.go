package sqlite

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/lifecycle"
)

func setupLifecycleRepository(t *testing.T) (*Repository, *LifecycleRepository, domain.PluginID, func() time.Time, func(time.Duration)) {
	t.Helper()
	repository, _ := openTestRepository(t)
	plugin := testPlugin("plugin.lifecycle.persistence")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)
	if _, err := repository.db.ExecContext(context.Background(), `INSERT INTO installed_plugin_versions (
		plugin_id, version, artifact_path, source_reference, digest_algorithm, digest_value,
		verification_policy, verification_evidence_ref, manifest_json, installed_at, selected
	) VALUES (?, ?, 'installed/plugin.lifecycle.persistence/1.2.3', 'test:source', 'SHA-256', 'digest', 'test-policy', NULL, '{}', ?, 1)`,
		string(plugin.ID), plugin.Manifest.Version, formatLifeTime(now)); err != nil {
		t.Fatal(err)
	}
	current := now
	adapter := newLifecycleRepository(repository.db, func() time.Time { return current })
	advance := func(d time.Duration) { current = current.Add(d) }
	return repository, adapter, plugin.ID, func() time.Time { return current }, advance
}

func lifecycleTestOperation(pluginID domain.PluginID, id string, now time.Time) domain.LifecycleOperation {
	return domain.LifecycleOperation{
		OperationID: id, PrincipalID: "principal-1", PluginID: pluginID,
		PluginVersion: "1.2.3", Action: domain.LifecycleActionEnable,
		Phase: domain.LifecycleOperationPhaseRequested, CreatedAt: now, UpdatedAt: now,
	}
}

func lifecycleFingerprint(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func authorizeAndApply(t *testing.T, repo *LifecycleRepository, id string, lease lifecycle.OperationLease, now time.Time) {
	t.Helper()
	decision := lifecycle.AuthorizationDecision{
		Allowed: true, DecisionID: "decision-" + id, PolicyRevision: "policy-v1",
		EvaluatedAt: now, ReasonCode: "ALLOW",
	}
	if err := repo.RecordAuthorization(context.Background(), id, lease, decision); err != nil {
		t.Fatalf("RecordAuthorization: %v", err)
	}
	if err := repo.BeginApply(context.Background(), id, lease); err != nil {
		t.Fatalf("BeginApply: %v", err)
	}
}

func TestLifecycleRepositoryCommitDesiredStateIsAtomicAndPreservesRuntimeObservation(t *testing.T) {
	repository, adapter, pluginID, clock, _ := setupLifecycleRepository(t)
	ctx := context.Background()
	now := clock()
	reservation, err := adapter.ReserveOperation(ctx, lifecycleTestOperation(pluginID, "lifecycle-op-1", now), lifecycleFingerprint("request-1"))
	if err != nil {
		t.Fatal(err)
	}
	authorizeAndApply(t, adapter, "lifecycle-op-1", reservation.Lease, now)

	record, err := adapter.CommitDesiredState(ctx, "lifecycle-op-1", reservation.Lease, domain.LifecycleDesiredStateEnabled)
	if err != nil {
		t.Fatal(err)
	}
	if record.DesiredState != domain.LifecycleDesiredStateEnabled || record.RuntimeObservation != domain.RuntimeObservationUnknown || record.Revision != 1 {
		t.Fatalf("unexpected initial lifecycle record: %#v", record)
	}
	op, err := adapter.GetOperation(ctx, "lifecycle-op-1")
	if err != nil {
		t.Fatal(err)
	}
	if op.Phase != domain.LifecycleOperationPhaseSucceeded || op.DecisionID != "decision-lifecycle-op-1" || op.PolicyRevision != "policy-v1" || op.EvaluatedAt.IsZero() || op.ReasonCode != "ALLOW" {
		t.Fatalf("operation did not retain complete authorization evidence: %#v", op)
	}

	// Simulate a future runtime reconciler changing observation. An
	// administrative desired-state no-op must preserve it and revision.
	if _, err := repository.db.ExecContext(ctx, "UPDATE plugin_lifecycle_records SET runtime_observation='RUNNING' WHERE plugin_id=?", string(pluginID)); err != nil {
		t.Fatal(err)
	}
	if err := adapter.ReleaseLease(ctx, "lifecycle-op-1", reservation.Lease); err != nil {
		t.Fatal(err)
	}
	now = clock()
	second := lifecycleTestOperation(pluginID, "lifecycle-op-2", now)
	second.Action = domain.LifecycleActionEnable
	reservation2, err := adapter.ReserveOperation(ctx, second, lifecycleFingerprint("request-2"))
	if err != nil {
		t.Fatal(err)
	}
	authorizeAndApply(t, adapter, "lifecycle-op-2", reservation2.Lease, now)
	record, err = adapter.CommitDesiredState(ctx, "lifecycle-op-2", reservation2.Lease, domain.LifecycleDesiredStateEnabled)
	if err != nil {
		t.Fatal(err)
	}
	if record.RuntimeObservation != domain.RuntimeObservationRunning || record.Revision != 1 {
		t.Fatalf("no-op changed runtime observation or revision: %#v", record)
	}
}

func TestLifecycleRepositoryIdempotencyAndFingerprintConflict(t *testing.T) {
	_, adapter, pluginID, clock, _ := setupLifecycleRepository(t)
	op := lifecycleTestOperation(pluginID, "lifecycle-idempotent", clock())
	fp := lifecycleFingerprint("same-command")
	first, err := adapter.ReserveOperation(context.Background(), op, fp)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := adapter.ReserveOperation(context.Background(), op, fp)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Existing || replay.Operation.OperationID != first.Operation.OperationID || replay.Lease.FencingToken != 0 {
		t.Fatalf("expected idempotent replay without a new lease: %#v", replay)
	}
	if _, err := adapter.ReserveOperation(context.Background(), op, lifecycleFingerprint("different-command")); !errors.Is(err, lifecycle.ErrOperationIDConflict) {
		t.Fatalf("fingerprint conflict = %v", err)
	}
}

func TestLifecycleRepositoryFencingAndExpiredLeaseRecovery(t *testing.T) {
	_, adapter, pluginID, clock, advance := setupLifecycleRepository(t)
	ctx := context.Background()
	now := clock()
	first, err := adapter.ReserveOperation(ctx, lifecycleTestOperation(pluginID, "lifecycle-stale", now), lifecycleFingerprint("stale"))
	if err != nil {
		t.Fatal(err)
	}
	advance(121 * time.Second)
	now = clock()
	second, err := adapter.ReserveOperation(ctx, lifecycleTestOperation(pluginID, "lifecycle-current", now), lifecycleFingerprint("current"))
	if err != nil {
		t.Fatal(err)
	}
	if second.Lease.FencingToken <= first.Lease.FencingToken {
		t.Fatalf("fencing token did not increase: first=%d second=%d", first.Lease.FencingToken, second.Lease.FencingToken)
	}
	if err := adapter.RecordAuthorization(ctx, "lifecycle-stale", first.Lease, lifecycle.AuthorizationDecision{
		Allowed: true, DecisionID: "old", PolicyRevision: "policy-v1", EvaluatedAt: now, ReasonCode: "ALLOW",
	}); !errors.Is(err, lifecycle.ErrLeaseLost) {
		t.Fatalf("stale worker mutation = %v, want ErrLeaseLost", err)
	}
	old, err := adapter.GetOperation(ctx, "lifecycle-stale")
	if err != nil {
		t.Fatal(err)
	}
	if old.Phase != domain.LifecycleOperationPhaseRecoveryRequired {
		t.Fatalf("expired operation phase = %v, want RecoveryRequired", old.Phase)
	}
}

func TestLifecycleRepositoryRejectsWrongSelectedVersion(t *testing.T) {
	_, adapter, pluginID, clock, _ := setupLifecycleRepository(t)
	op := lifecycleTestOperation(pluginID, "lifecycle-version-mismatch", clock())
	op.PluginVersion = "9.9.9"
	if _, err := adapter.ReserveOperation(context.Background(), op, lifecycleFingerprint("version-mismatch")); !errors.Is(err, lifecycle.ErrVersionNotSelected) {
		t.Fatalf("selected version mismatch = %v", err)
	}
}

func TestLifecycleRepositoryRecoveryQueryAndClassification(t *testing.T) {
	_, adapter, pluginID, clock, advance := setupLifecycleRepository(t)
	ctx := context.Background()
	reservation, err := adapter.ReserveOperation(ctx, lifecycleTestOperation(pluginID, "lifecycle-recovery", clock()), lifecycleFingerprint("recovery"))
	if err != nil {
		t.Fatal(err)
	}
	candidates, err := adapter.ListRecoveryCandidates(ctx, 10)
	if err != nil || len(candidates) != 1 || !candidates[0].LeaseActive {
		t.Fatalf("active recovery candidate = %#v, err=%v", candidates, err)
	}
	advance(121 * time.Second)
	changed, err := adapter.MarkExpiredOperationsRecoveryRequired(ctx, 10)
	if err != nil || changed != 1 {
		t.Fatalf("MarkExpiredOperationsRecoveryRequired = %d, err=%v", changed, err)
	}
	op, err := adapter.GetOperation(ctx, reservation.Operation.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if op.Phase != domain.LifecycleOperationPhaseRecoveryRequired {
		t.Fatalf("phase = %v, want RecoveryRequired", op.Phase)
	}
}
