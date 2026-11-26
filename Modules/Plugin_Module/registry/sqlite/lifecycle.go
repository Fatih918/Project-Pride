package sqlite

import (
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/lifecycle"
)

const lifecycleLeaseDuration = 120 * time.Second

type LifecycleRepository struct {
	db *sql.DB
	now func() time.Time
}

func NewLifecycleRepository(db *sql.DB) (*LifecycleRepository, error) {
	if db == nil { return nil, fmt.Errorf("%w: nil SQLite database handle", lifecycle.ErrPersistence) }
	return &LifecycleRepository{db: db, now: func() time.Time { return time.Now().UTC() }}, nil
}

// Lifecycle returns an adapter sharing the registry's migrated SQLite handle.
func (r *Repository) Lifecycle() *LifecycleRepository {
	return &LifecycleRepository{db: r.db, now: func() time.Time { return time.Now().UTC() }}
}

func newLifecycleRepository(db *sql.DB, now func() time.Time) *LifecycleRepository {
	return &LifecycleRepository{db: db, now: now}
}

const lifecycleOperationSelect = "SELECT operation_id, request_fingerprint, principal_id, plugin_id, plugin_version, action, phase, failure_code, decision_id, policy_revision, evaluated_at, reason_code, created_at, updated_at FROM plugin_lifecycle_operations"

func (r *LifecycleRepository) ReserveOperation(ctx context.Context, op domain.LifecycleOperation, fingerprint string) (lifecycle.OperationReservation, error) {
	if err := ctx.Err(); err != nil { return lifecycle.OperationReservation{}, err }
	if err := op.Validate(); err != nil || op.Phase != domain.LifecycleOperationPhaseRequested {
		return lifecycle.OperationReservation{}, fmt.Errorf("%w: invalid initial operation", lifecycle.ErrPersistence)
	}
	if len(fingerprint) != 64 || strings.ToLower(fingerprint) != fingerprint { return lifecycle.OperationReservation{}, fmt.Errorf("%w: invalid SHA-256 fingerprint", lifecycle.ErrPersistence) }
	if _, err := hex.DecodeString(fingerprint); err != nil { return lifecycle.OperationReservation{}, fmt.Errorf("%w: invalid SHA-256 fingerprint", lifecycle.ErrPersistence) }

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return lifecycle.OperationReservation{}, lifePersist("begin reservation", err) }
	defer func() { _ = tx.Rollback() }()

	if existing, fp, err := scanLifeOperation(tx.QueryRowContext(ctx, lifecycleOperationSelect+" WHERE operation_id = ?", op.OperationID)); err == nil {
		if fp != fingerprint { return lifecycle.OperationReservation{}, lifecycle.ErrOperationIDConflict }
		return lifecycle.OperationReservation{Operation: existing, Existing: true}, nil
	} else if !errors.Is(err, sql.ErrNoRows) { return lifecycle.OperationReservation{}, lifePersist("read existing operation", err) }

	insert, err := tx.ExecContext(ctx, "INSERT OR IGNORE INTO plugin_lifecycle_fencing_tokens (plugin_id, last_token) VALUES (?, 1)", string(op.PluginID))
	if err != nil { return lifecycle.OperationReservation{}, lifePersist("lock fencing counter", err) }
	inserted, err := insert.RowsAffected()
	if err != nil { return lifecycle.OperationReservation{}, lifePersist("inspect fencing counter", err) }

	// Recheck after acquiring SQLite's write reservation to close cross-process races.
	if existing, fp, err := scanLifeOperation(tx.QueryRowContext(ctx, lifecycleOperationSelect+" WHERE operation_id = ?", op.OperationID)); err == nil {
		if fp != fingerprint { return lifecycle.OperationReservation{}, lifecycle.ErrOperationIDConflict }
		return lifecycle.OperationReservation{Operation: existing, Existing: true}, nil
	} else if !errors.Is(err, sql.ErrNoRows) { return lifecycle.OperationReservation{}, lifePersist("recheck operation ID", err) }

	var selected string
	err = tx.QueryRowContext(ctx, "SELECT version FROM installed_plugin_versions WHERE plugin_id = ? AND selected = 1", string(op.PluginID)).Scan(&selected)
	if errors.Is(err, sql.ErrNoRows) { return lifecycle.OperationReservation{}, lifecycle.ErrVersionNotSelected }
	if err != nil { return lifecycle.OperationReservation{}, lifePersist("read selected installed version", err) }
	if selected != op.PluginVersion { return lifecycle.OperationReservation{}, lifecycle.ErrVersionNotSelected }

	now := r.now().UTC()
	nowText := formatLifeTime(now)
	var previousID, expiryText string
	err = tx.QueryRowContext(ctx, "SELECT operation_id, expires_at FROM plugin_lifecycle_leases WHERE plugin_id = ?", string(op.PluginID)).Scan(&previousID, &expiryText)
	if err == nil {
		expiry, parseErr := time.Parse(time.RFC3339Nano, expiryText)
		if parseErr != nil { return lifecycle.OperationReservation{}, lifePersist("parse lease expiry", parseErr) }
		if expiry.After(now) { return lifecycle.OperationReservation{}, lifecycle.ErrPluginOperationConflict }
		if _, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='RECOVERY_REQUIRED', updated_at=? WHERE operation_id=? AND phase IN ('REQUESTED','AUTHORIZED','APPLYING')", nowText, previousID); err != nil {
			return lifecycle.OperationReservation{}, lifePersist("classify expired operation", err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM plugin_lifecycle_leases WHERE plugin_id=? AND operation_id=?", string(op.PluginID), previousID); err != nil {
			return lifecycle.OperationReservation{}, lifePersist("delete expired lease", err)
		}
	} else if !errors.Is(err, sql.ErrNoRows) { return lifecycle.OperationReservation{}, lifePersist("read active lease", err) }

	// A process may have terminated after releasing/losing its lease but
	// before recording a terminal phase. Such orphaned work is never resumed.
	if _, err := tx.ExecContext(ctx, `UPDATE plugin_lifecycle_operations
		SET phase='RECOVERY_REQUIRED', updated_at=?
		WHERE plugin_id=? AND phase IN ('REQUESTED','AUTHORIZED','APPLYING')
		AND NOT EXISTS (SELECT 1 FROM plugin_lifecycle_leases l WHERE l.operation_id=plugin_lifecycle_operations.operation_id)`, nowText, string(op.PluginID)); err != nil {
		return lifecycle.OperationReservation{}, lifePersist("classify orphaned lifecycle operations", err)
	}

	var token int64
	if inserted == 1 { token = 1 } else {
		result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_fencing_tokens SET last_token=last_token+1 WHERE plugin_id=? AND last_token<?", string(op.PluginID), int64(math.MaxInt64))
		if err != nil { return lifecycle.OperationReservation{}, lifePersist("increment fencing token", err) }
		n, err := result.RowsAffected()
		if err != nil { return lifecycle.OperationReservation{}, lifePersist("inspect fencing token", err) }
		if n != 1 { return lifecycle.OperationReservation{}, lifecycle.ErrFencingTokenExhausted }
		if err := tx.QueryRowContext(ctx, "SELECT last_token FROM plugin_lifecycle_fencing_tokens WHERE plugin_id=?", string(op.PluginID)).Scan(&token); err != nil {
			return lifecycle.OperationReservation{}, lifePersist("read fencing token", err)
		}
	}

	expires := now.Add(lifecycleLeaseDuration)
	_, err = tx.ExecContext(ctx, `INSERT INTO plugin_lifecycle_operations (
		operation_id, request_fingerprint, principal_id, plugin_id, plugin_version, action,
		phase, failure_code, decision_id, policy_revision, evaluated_at, reason_code, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, 'REQUESTED', NULL, NULL, NULL, NULL, NULL, ?, ?)`,
		op.OperationID, fingerprint, op.PrincipalID, string(op.PluginID), op.PluginVersion,
		lifeActionName(op.Action), formatLifeTime(op.CreatedAt), formatLifeTime(op.UpdatedAt))
	if err != nil { return lifecycle.OperationReservation{}, lifePersist("insert operation", err) }
	_, err = tx.ExecContext(ctx, "INSERT INTO plugin_lifecycle_leases (plugin_id, operation_id, fencing_token, expires_at) VALUES (?, ?, ?, ?)",
		string(op.PluginID), op.OperationID, token, formatLifeTime(expires))
	if err != nil {
		if isConstraintError(err) { return lifecycle.OperationReservation{}, lifecycle.ErrPluginOperationConflict }
		return lifecycle.OperationReservation{}, lifePersist("insert lease", err)
	}
	if err := tx.Commit(); err != nil {
		if isBusyError(err) { return lifecycle.OperationReservation{}, lifecycle.ErrPluginOperationConflict }
		return lifecycle.OperationReservation{}, lifePersist("commit reservation", err)
	}
	return lifecycle.OperationReservation{Operation: op, Lease: lifecycle.OperationLease{FencingToken: token, ExpiresAt: expires}}, nil
}

func (r *LifecycleRepository) RecordAuthorization(ctx context.Context, id string, lease lifecycle.OperationLease, decision lifecycle.AuthorizationDecision) error {
	if err := decision.Validate(); err != nil { return err }
	return r.mutate(ctx, id, lease, domain.LifecycleOperationPhaseRequested, func(tx *sql.Tx, _ domain.LifecycleOperation) error {
		result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='AUTHORIZED', decision_id=?, policy_revision=?, evaluated_at=?, reason_code=?, updated_at=? WHERE operation_id=? AND phase='REQUESTED'",
			decision.DecisionID, decision.PolicyRevision, formatLifeTime(decision.EvaluatedAt), decision.ReasonCode, formatLifeTime(r.now().UTC()), id)
		if err != nil { return lifePersist("record authorization", err) }
		return lifeOneRow(result)
	})
}

func (r *LifecycleRepository) BeginApply(ctx context.Context, id string, lease lifecycle.OperationLease) error {
	return r.mutate(ctx, id, lease, domain.LifecycleOperationPhaseAuthorized, func(tx *sql.Tx, _ domain.LifecycleOperation) error {
		result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='APPLYING', updated_at=? WHERE operation_id=? AND phase='AUTHORIZED'", formatLifeTime(r.now().UTC()), id)
		if err != nil { return lifePersist("begin apply", err) }
		return lifeOneRow(result)
	})
}

func (r *LifecycleRepository) CommitDesiredState(ctx context.Context, id string, lease lifecycle.OperationLease, target domain.LifecycleDesiredState) (domain.LifecycleRecord, error) {
	if !target.Valid() { return domain.LifecycleRecord{}, lifecycle.ErrInvalidTransition }
	var committed domain.LifecycleRecord
	err := r.mutate(ctx, id, lease, domain.LifecycleOperationPhaseApplying, func(tx *sql.Tx, op domain.LifecycleOperation) error {
		var selected string
		err := tx.QueryRowContext(ctx, "SELECT version FROM installed_plugin_versions WHERE plugin_id=? AND selected=1", string(op.PluginID)).Scan(&selected)
		if errors.Is(err, sql.ErrNoRows) || (err == nil && selected != op.PluginVersion) { return lifecycle.ErrVersionNotSelected }
		if err != nil { return lifePersist("verify selected version", err) }
		now := r.now().UTC()
		err = scanLifeRecord(tx.QueryRowContext(ctx, "SELECT plugin_id, selected_version, desired_state, runtime_observation, revision, created_at, updated_at FROM plugin_lifecycle_records WHERE plugin_id=?", string(op.PluginID)), &committed)
		if errors.Is(err, sql.ErrNoRows) {
			committed = domain.LifecycleRecord{PluginID: op.PluginID, SelectedVersion: op.PluginVersion, DesiredState: target, RuntimeObservation: domain.RuntimeObservationUnknown, Revision: 1, CreatedAt: now, UpdatedAt: now}
			_, err = tx.ExecContext(ctx, `INSERT INTO plugin_lifecycle_records (plugin_id, selected_version, desired_state, runtime_observation, revision, created_at, updated_at) VALUES (?, ?, ?, 'UNKNOWN', 1, ?, ?)`,
				string(op.PluginID), op.PluginVersion, lifeDesiredName(target), formatLifeTime(now), formatLifeTime(now))
			if err != nil { return lifePersist("insert lifecycle record", err) }
		} else if err != nil { return lifePersist("read lifecycle record", err) } else {
			changed := committed.DesiredState != target
			versionChanged := committed.SelectedVersion != op.PluginVersion
			if changed && committed.Revision >= math.MaxInt64 { return lifePersist("lifecycle revision exhausted", errors.New("revision overflow")) }
			if changed || versionChanged {
				nextRevision := committed.Revision
				if changed { nextRevision++ }
				result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_records SET selected_version=?, desired_state=?, revision=?, updated_at=? WHERE plugin_id=? AND revision=?",
					op.PluginVersion, lifeDesiredName(target), int64(nextRevision), formatLifeTime(now), string(op.PluginID), committed.Revision)
				if err != nil { return lifePersist("update lifecycle record", err) }
				if err := lifeOneRow(result); err != nil { return err }
				committed.SelectedVersion, committed.DesiredState, committed.Revision, committed.UpdatedAt = op.PluginVersion, target, nextRevision, now
			}
		}
		result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='SUCCEEDED', updated_at=? WHERE operation_id=? AND phase='APPLYING'", formatLifeTime(now), id)
		if err != nil { return lifePersist("complete operation", err) }
		if err := lifeOneRow(result); err != nil { return err }
		return nil
	})
	if err != nil { return domain.LifecycleRecord{}, err }
	return committed, nil
}

func (r *LifecycleRepository) FailOperation(ctx context.Context, id string, lease lifecycle.OperationLease, code string) error {
	if !validLifeFailureCode(code) { return fmt.Errorf("%w: invalid bounded failure code", lifecycle.ErrPersistence) }
	return r.mutate(ctx, id, lease, 0, func(tx *sql.Tx, _ domain.LifecycleOperation) error {
		result, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='FAILED', failure_code=?, updated_at=? WHERE operation_id=? AND phase IN ('REQUESTED','AUTHORIZED','APPLYING')",
			code, formatLifeTime(r.now().UTC()), id)
		if err != nil { return lifePersist("fail operation", err) }
		return lifeOneRow(result)
	})
}

func (r *LifecycleRepository) GetOperation(ctx context.Context, id string) (domain.LifecycleOperation, error) {
	if err := ctx.Err(); err != nil { return domain.LifecycleOperation{}, err }
	op, _, err := scanLifeOperation(r.db.QueryRowContext(ctx, lifecycleOperationSelect+" WHERE operation_id=?", id))
	if errors.Is(err, sql.ErrNoRows) { return domain.LifecycleOperation{}, lifecycle.ErrOperationNotFound }
	if err != nil { return domain.LifecycleOperation{}, lifePersist("get operation", err) }
	return op, nil
}

func (r *LifecycleRepository) ReleaseLease(ctx context.Context, id string, lease lifecycle.OperationLease) error {
	if err := ctx.Err(); err != nil { return err }
	if lease.FencingToken <= 0 { return lifecycle.ErrLeaseLost }
	_, err := r.db.ExecContext(ctx, "DELETE FROM plugin_lifecycle_leases WHERE operation_id=? AND fencing_token=?", id, lease.FencingToken)
	if err != nil { return lifePersist("release lease", err) }
	return nil
}

func (r *LifecycleRepository) mutate(ctx context.Context, id string, lease lifecycle.OperationLease, expected domain.LifecycleOperationPhase, fn func(*sql.Tx, domain.LifecycleOperation) error) error {
	if err := ctx.Err(); err != nil { return err }
	if lease.FencingToken <= 0 { return lifecycle.ErrLeaseLost }
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return lifePersist("begin fenced mutation", err) }
	defer func() { _ = tx.Rollback() }()
	op, _, err := scanLifeOperation(tx.QueryRowContext(ctx, lifecycleOperationSelect+" WHERE operation_id=?", id))
	if errors.Is(err, sql.ErrNoRows) { return lifecycle.ErrOperationNotFound }
	if err != nil { return lifePersist("read operation", err) }
	var token int64
	var expiryText string
	err = tx.QueryRowContext(ctx, "SELECT fencing_token, expires_at FROM plugin_lifecycle_leases WHERE operation_id=? AND plugin_id=?", id, string(op.PluginID)).Scan(&token, &expiryText)
	if errors.Is(err, sql.ErrNoRows) { return lifecycle.ErrLeaseLost }
	if err != nil { return lifePersist("read lease", err) }
	expiry, err := time.Parse(time.RFC3339Nano, expiryText)
	if err != nil { return lifePersist("parse lease expiry", err) }
	if token != lease.FencingToken || !expiry.After(r.now().UTC()) { return lifecycle.ErrLeaseLost }
	if expected != 0 && op.Phase != expected { return fmt.Errorf("%w: expected %s, got %s", lifecycle.ErrInvalidTransition, lifePhaseName(expected), lifePhaseName(op.Phase)) }
	if err := fn(tx, op); err != nil { return err }
	if err := ctx.Err(); err != nil { return err }
	if err := tx.Commit(); err != nil { return lifePersist("commit fenced mutation", err) }
	return nil
}

func (r *LifecycleRepository) ListRecoveryCandidates(ctx context.Context, limit int) ([]lifecycle.RecoveryCandidate, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	if limit < 1 || limit > 1000 { return nil, fmt.Errorf("%w: recovery limit must be 1..1000", lifecycle.ErrPersistence) }
	rows, err := r.db.QueryContext(ctx, `SELECT o.operation_id, o.request_fingerprint, o.principal_id, o.plugin_id, o.plugin_version, o.action, o.phase, o.failure_code, o.decision_id, o.policy_revision, o.evaluated_at, o.reason_code, o.created_at, o.updated_at, l.expires_at
		FROM plugin_lifecycle_operations o LEFT JOIN plugin_lifecycle_leases l ON l.operation_id=o.operation_id
		WHERE o.phase IN ('REQUESTED','AUTHORIZED','APPLYING','RECOVERY_REQUIRED') ORDER BY o.created_at, o.operation_id LIMIT ?`, limit)
	if err != nil { return nil, lifePersist("list recovery candidates", err) }
	defer rows.Close()
	now := r.now().UTC()
	out := make([]lifecycle.RecoveryCandidate, 0)
	for rows.Next() {
		op, expiry, err := scanLifeOperationExpiry(rows)
		if err != nil { return nil, lifePersist("decode recovery candidate", err) }
		candidate := lifecycle.RecoveryCandidate{Operation: op}
		if expiry.Valid {
			candidate.LeaseExpiresAt, err = time.Parse(time.RFC3339Nano, expiry.String)
			if err != nil { return nil, lifePersist("parse recovery expiry", err) }
			candidate.LeaseActive = candidate.LeaseExpiresAt.After(now)
		}
		out = append(out, candidate)
	}
	if err := rows.Err(); err != nil { return nil, lifePersist("iterate recovery candidates", err) }
	return out, nil
}

func (r *LifecycleRepository) MarkExpiredOperationsRecoveryRequired(ctx context.Context, limit int) (int, error) {
	if err := ctx.Err(); err != nil { return 0, err }
	if limit < 1 || limit > 1000 { return 0, fmt.Errorf("%w: recovery limit must be 1..1000", lifecycle.ErrPersistence) }
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil { return 0, lifePersist("begin expiry classification", err) }
	defer func() { _ = tx.Rollback() }()
	now := formatLifeTime(r.now().UTC())
	rows, err := tx.QueryContext(ctx, `SELECT o.operation_id FROM plugin_lifecycle_operations o LEFT JOIN plugin_lifecycle_leases l ON l.operation_id=o.operation_id
		WHERE o.phase IN ('REQUESTED','AUTHORIZED','APPLYING') AND (l.operation_id IS NULL OR l.expires_at<=?)
		ORDER BY o.created_at, o.operation_id LIMIT ?`, now, limit)
	if err != nil { return 0, lifePersist("select expired operations", err) }
	ids := make([]string, 0, limit)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil { _ = rows.Close(); return 0, lifePersist("scan expired operation", err) }
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil { _ = rows.Close(); return 0, lifePersist("iterate expired operations", err) }
	_ = rows.Close()
	for _, id := range ids {
		if _, err := tx.ExecContext(ctx, "UPDATE plugin_lifecycle_operations SET phase='RECOVERY_REQUIRED', updated_at=? WHERE operation_id=? AND phase IN ('REQUESTED','AUTHORIZED','APPLYING')", now, id); err != nil {
			return 0, lifePersist("mark recovery required", err)
		}
		if _, err := tx.ExecContext(ctx, "DELETE FROM plugin_lifecycle_leases WHERE operation_id=? AND expires_at<=?", id, now); err != nil {
			return 0, lifePersist("delete expired lease", err)
		}
	}
	if err := tx.Commit(); err != nil { return 0, lifePersist("commit expiry classification", err) }
	return len(ids), nil
}

type lifeScanner interface { Scan(...any) error }

func scanLifeOperation(row lifeScanner) (domain.LifecycleOperation, string, error) {
	var op domain.LifecycleOperation
	var fp, pluginID, action, phase, created, updated string
	var failure, decision, policy, evaluated, reason sql.NullString
	err := row.Scan(&op.OperationID, &fp, &op.PrincipalID, &pluginID, &op.PluginVersion, &action, &phase, &failure, &decision, &policy, &evaluated, &reason, &created, &updated)
	if err != nil { return domain.LifecycleOperation{}, "", err }
	op.PluginID = domain.PluginID(pluginID)
	op.Action, err = parseLifeAction(action); if err != nil { return op, "", err }
	op.Phase, err = parseLifePhase(phase); if err != nil { return op, "", err }
	op.FailureCode, op.DecisionID, op.PolicyRevision, op.ReasonCode = failure.String, decision.String, policy.String, reason.String
	if evaluated.Valid { op.EvaluatedAt, err = time.Parse(time.RFC3339Nano, evaluated.String); if err != nil { return op, "", err } }
	op.CreatedAt, err = time.Parse(time.RFC3339Nano, created); if err != nil { return op, "", err }
	op.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); if err != nil { return op, "", err }
	if len(fp) != 64 || strings.ToLower(fp) != fp {
		return op, "", errors.New("invalid persisted lifecycle fingerprint")
	}
	if _, err := hex.DecodeString(fp); err != nil {
		return op, "", errors.New("invalid persisted lifecycle fingerprint")
	}
	if err := op.Validate(); err != nil { return op, "", err }
	return op, fp, nil
}

func scanLifeOperationExpiry(row lifeScanner) (domain.LifecycleOperation, sql.NullString, error) {
	var op domain.LifecycleOperation
	var fp, pluginID, action, phase, created, updated string
	var failure, decision, policy, evaluated, reason, expiry sql.NullString
	err := row.Scan(&op.OperationID, &fp, &op.PrincipalID, &pluginID, &op.PluginVersion, &action, &phase, &failure, &decision, &policy, &evaluated, &reason, &created, &updated, &expiry)
	if err != nil { return op, expiry, err }
	op.PluginID = domain.PluginID(pluginID)
	op.Action, err = parseLifeAction(action); if err != nil { return op, expiry, err }
	op.Phase, err = parseLifePhase(phase); if err != nil { return op, expiry, err }
	op.FailureCode, op.DecisionID, op.PolicyRevision, op.ReasonCode = failure.String, decision.String, policy.String, reason.String
	if evaluated.Valid { op.EvaluatedAt, err = time.Parse(time.RFC3339Nano, evaluated.String); if err != nil { return op, expiry, err } }
	op.CreatedAt, err = time.Parse(time.RFC3339Nano, created); if err != nil { return op, expiry, err }
	op.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); if err != nil { return op, expiry, err }
	if err := op.Validate(); err != nil { return op, expiry, err }
	return op, expiry, nil
}

func scanLifeRecord(row lifeScanner, record *domain.LifecycleRecord) error {
	var pluginID, desired, observation, created, updated string
	var revision int64
	if err := row.Scan(&pluginID, &record.SelectedVersion, &desired, &observation, &revision, &created, &updated); err != nil { return err }
	record.PluginID = domain.PluginID(pluginID)
	var err error
	record.DesiredState, err = parseLifeDesired(desired); if err != nil { return err }
	record.RuntimeObservation, err = parseLifeObservation(observation); if err != nil { return err }
	if revision <= 0 { return errors.New("invalid lifecycle revision") }
	record.Revision = uint64(revision)
	record.CreatedAt, err = time.Parse(time.RFC3339Nano, created); if err != nil { return err }
	record.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); if err != nil { return err }
	return record.Validate()
}

func parseLifeAction(v string) (domain.LifecycleAction,error) { switch v { case "ENABLE":return domain.LifecycleActionEnable,nil; case "DISABLE":return domain.LifecycleActionDisable,nil; default:return 0,fmt.Errorf("unknown action %q",v) } }
func lifeActionName(v domain.LifecycleAction) string { if v==domain.LifecycleActionEnable{return "ENABLE"}; if v==domain.LifecycleActionDisable{return "DISABLE"}; return "" }
func parseLifePhase(v string) (domain.LifecycleOperationPhase,error) { switch v { case "REQUESTED":return domain.LifecycleOperationPhaseRequested,nil; case "AUTHORIZED":return domain.LifecycleOperationPhaseAuthorized,nil; case "APPLYING":return domain.LifecycleOperationPhaseApplying,nil; case "SUCCEEDED":return domain.LifecycleOperationPhaseSucceeded,nil; case "FAILED":return domain.LifecycleOperationPhaseFailed,nil; case "RECOVERY_REQUIRED":return domain.LifecycleOperationPhaseRecoveryRequired,nil; default:return 0,fmt.Errorf("unknown phase %q",v) } }
func lifePhaseName(v domain.LifecycleOperationPhase) string { switch v { case domain.LifecycleOperationPhaseRequested:return "REQUESTED"; case domain.LifecycleOperationPhaseAuthorized:return "AUTHORIZED"; case domain.LifecycleOperationPhaseApplying:return "APPLYING"; case domain.LifecycleOperationPhaseSucceeded:return "SUCCEEDED"; case domain.LifecycleOperationPhaseFailed:return "FAILED"; case domain.LifecycleOperationPhaseRecoveryRequired:return "RECOVERY_REQUIRED"; default:return "UNSPECIFIED" } }
func lifeDesiredName(v domain.LifecycleDesiredState) string { if v==domain.LifecycleDesiredStateEnabled{return "ENABLED"}; if v==domain.LifecycleDesiredStateDisabled{return "DISABLED"}; return "" }
func parseLifeDesired(v string) (domain.LifecycleDesiredState,error) { switch v { case "ENABLED":return domain.LifecycleDesiredStateEnabled,nil; case "DISABLED":return domain.LifecycleDesiredStateDisabled,nil; default:return 0,fmt.Errorf("unknown desired state %q",v) } }
func parseLifeObservation(v string) (domain.RuntimeObservation,error) { switch v { case "UNKNOWN":return domain.RuntimeObservationUnknown,nil; case "STOPPED":return domain.RuntimeObservationStopped,nil; case "STARTING":return domain.RuntimeObservationStarting,nil; case "RUNNING":return domain.RuntimeObservationRunning,nil; case "STOPPING":return domain.RuntimeObservationStopping,nil; case "FAILED":return domain.RuntimeObservationFailed,nil; default:return 0,fmt.Errorf("unknown runtime observation %q",v) } }
func validLifeFailureCode(v string) bool { if len(v)==0||len(v)>64{return false}; for _,c:=range v { if (c<'A'||c>'Z')&&(c<'0'||c>'9')&&c!='_' {return false} }; return true }
func lifeOneRow(result sql.Result) error { n,err:=result.RowsAffected(); if err!=nil{return lifePersist("inspect rows",err)}; if n!=1{return lifecycle.ErrInvalidTransition}; return nil }
func lifePersist(action string,err error) error { return fmt.Errorf("%w: %s: %v",lifecycle.ErrPersistence,action,err) }
func isConstraintError(err error) bool { s:=strings.ToLower(err.Error()); return strings.Contains(s,"constraint")||strings.Contains(s,"unique") }
func isBusyError(err error) bool { s:=strings.ToLower(err.Error()); return strings.Contains(s,"database is locked")||strings.Contains(s,"database is busy")||strings.Contains(s,"sqlite_busy") }

func formatLifeTime(value time.Time) string {
	return value.UTC().Format("2006-01-02T15:04:05.000000000Z")
}

var _ lifecycle.LifecycleRepository = (*LifecycleRepository)(nil)
var _ lifecycle.RecoveryRepository = (*LifecycleRepository)(nil)

var _ lifecycle.LifecycleRepository = (*LifecycleRepository)(nil)
var _ lifecycle.RecoveryRepository = (*LifecycleRepository)(nil)
