package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

const operationColumns = `operation_id, plugin_id, target_version, operation_kind, phase,
	staging_path, final_artifact_path, source_reference, digest_algorithm, digest_value,
	verification_policy, manifest_json, failure_code, created_at, updated_at`

func (r *Repository) BeginOperation(ctx context.Context, operation installation.InstallationOperation) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := operation.ValidateForBegin(); err != nil {
		return err
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO plugin_installation_operations (
		operation_id, plugin_id, target_version, operation_kind, phase, staging_path,
		final_artifact_path, source_reference, digest_algorithm, digest_value,
		verification_policy, manifest_json, failure_code, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, NULL, ?, NULL, NULL, NULL, NULL, NULL, ?, ?)`,
		string(operation.OperationID), string(operation.PluginID), operation.TargetVersion,
		string(operation.Kind), string(operation.Phase), nullableString(operation.StagingPath),
		operation.SourceReference, formatTime(operation.CreatedAt), formatTime(operation.UpdatedAt))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			return fmt.Errorf("%w: operation or active operation already exists", installation.ErrOperationConflict)
		}
		return fmt.Errorf("%w: begin operation", installation.ErrPersistence)
	}
	return nil
}

func (r *Repository) RecordStaged(ctx context.Context, id installation.OperationID, staged installation.StagedArtifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateOperationPath(staged.StagingPath, "temporary"); err != nil {
		return err
	}
	return r.transition(ctx, id, installation.PhasePreparing, installation.PhaseStaged,
		`staging_path = ?, updated_at = ?`, staged.StagingPath, formatTime(time.Now().UTC()))
}

func (r *Repository) RecordVerified(ctx context.Context, id installation.OperationID, verified installation.VerifiedArtifact) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateOperationPath(verified.FinalArtifactPath, "installed"); err != nil {
		return err
	}
	if strings.TrimSpace(verified.DigestAlgorithm) == "" || strings.TrimSpace(verified.DigestValue) == "" ||
		strings.TrimSpace(verified.VerificationPolicy) == "" || !json.Valid([]byte(verified.ManifestJSON)) {
		return fmt.Errorf("%w: verified artifact is incomplete", installation.ErrInvalidRecord)
	}
	return r.transition(ctx, id, installation.PhaseStaged, installation.PhaseVerified,
		`final_artifact_path = ?, digest_algorithm = ?, digest_value = ?,
		verification_policy = ?, manifest_json = ?, updated_at = ?`,
		verified.FinalArtifactPath, verified.DigestAlgorithm, verified.DigestValue,
		verified.VerificationPolicy, verified.ManifestJSON, formatTime(time.Now().UTC()))
}

func (r *Repository) RecordPromoted(ctx context.Context, id installation.OperationID, finalPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validateOperationPath(finalPath, "installed"); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin promotion record", installation.ErrPersistence)
	}
	defer func() { _ = tx.Rollback() }()
	var expected string
	err = tx.QueryRowContext(ctx, "SELECT final_artifact_path FROM plugin_installation_operations WHERE operation_id = ?", string(id)).Scan(&expected)
	if errors.Is(err, sql.ErrNoRows) {
		return installation.ErrOperationNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: read promotion target", installation.ErrPersistence)
	}
	if expected != finalPath {
		return installation.ErrRecordMismatch
	}
	result, err := tx.ExecContext(ctx, `UPDATE plugin_installation_operations SET phase = ?, updated_at = ?
		WHERE operation_id = ? AND phase = ?`, string(installation.PhasePromoted), formatTime(time.Now().UTC()),
		string(id), string(installation.PhaseVerified))
	if err != nil {
		return fmt.Errorf("%w: record promotion", installation.ErrPersistence)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit promotion record", installation.ErrPersistence)
	}
	return nil
}

func (r *Repository) transition(ctx context.Context, id installation.OperationID, from, to installation.Phase, assignments string, args ...any) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin operation transition", installation.ErrPersistence)
	}
	defer func() { _ = tx.Rollback() }()
	var phase string
	err = tx.QueryRowContext(ctx, "SELECT phase FROM plugin_installation_operations WHERE operation_id = ?", string(id)).Scan(&phase)
	if errors.Is(err, sql.ErrNoRows) {
		return installation.ErrOperationNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: read operation phase", installation.ErrPersistence)
	}
	if installation.Phase(phase) != from {
		return fmt.Errorf("%w: expected %s, found %s", installation.ErrInvalidTransition, from, phase)
	}
	query := "UPDATE plugin_installation_operations SET " + assignments + " WHERE operation_id = ? AND phase = ?"
	args = append(args, string(id), string(from))
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("%w: update operation phase", installation.ErrPersistence)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit operation transition", installation.ErrPersistence)
	}
	return nil
}

func (r *Repository) CommitInstallation(ctx context.Context, id installation.OperationID, plugin domain.Plugin, version installation.InstalledVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("%w: invalid Plugin aggregate", installation.ErrInvalidRecord)
	}
	if err := version.Validate(); err != nil {
		return err
	}
	if plugin.ID != version.PluginID || plugin.Manifest.Version != version.Version {
		return installation.ErrRecordMismatch
	}
	manifestJSON, err := json.Marshal(plugin.Manifest)
	if err != nil {
		return fmt.Errorf("%w: encode Plugin Manifest", installation.ErrInvalidRecord)
	}
	var suppliedManifest domain.Manifest
	if err := json.Unmarshal([]byte(version.ManifestJSON), &suppliedManifest); err != nil || !reflect.DeepEqual(suppliedManifest, plugin.Manifest) {
		return installation.ErrRecordMismatch
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin installation commit", installation.ErrPersistence)
	}
	defer func() { _ = tx.Rollback() }()
	op, err := scanOperation(tx.QueryRowContext(ctx, "SELECT "+operationColumns+" FROM plugin_installation_operations WHERE operation_id = ?", string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return installation.ErrOperationNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: read installation operation", installation.ErrPersistence)
	}
	if op.Phase != installation.PhasePromoted {
		return installation.ErrInvalidTransition
	}
	if op.PluginID != plugin.ID || op.TargetVersion != version.Version ||
		op.FinalArtifactPath != version.ArtifactPath || op.SourceReference != version.SourceReference ||
		op.DigestAlgorithm != version.DigestAlgorithm || op.DigestValue != version.DigestValue ||
		op.VerificationPolicy != version.VerificationPolicy || op.ManifestJSON == "" {
		return installation.ErrRecordMismatch
	}
	var verifiedManifest domain.Manifest
	if err := json.Unmarshal([]byte(op.ManifestJSON), &verifiedManifest); err != nil || !reflect.DeepEqual(verifiedManifest, plugin.Manifest) {
		return installation.ErrRecordMismatch
	}
	version.ManifestJSON = string(manifestJSON)
	if _, err := tx.ExecContext(ctx, "UPDATE installed_plugin_versions SET selected = 0 WHERE plugin_id = ?", string(plugin.ID)); err != nil {
		return fmt.Errorf("%w: clear previous selected version", installation.ErrPersistence)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO installed_plugin_versions (
		plugin_id, version, artifact_path, source_reference, digest_algorithm, digest_value,
		verification_policy, verification_evidence_ref, manifest_json, installed_at, selected
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1)`,
		string(version.PluginID), version.Version, version.ArtifactPath, version.SourceReference,
		version.DigestAlgorithm, version.DigestValue, version.VerificationPolicy,
		nullableString(version.VerificationEvidenceRef), version.ManifestJSON, formatTime(version.InstalledAt))
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") || strings.Contains(strings.ToLower(err.Error()), "constraint") {
			return installation.ErrVersionAlreadyExist
		}
		return fmt.Errorf("%w: insert installed version", installation.ErrPersistence)
	}
	protectedJSON, err := json.Marshal(plugin.Protected)
	if err != nil {
		return fmt.Errorf("%w: encode Plugin metadata", installation.ErrInvalidRecord)
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO plugins (
		plugin_id, manifest_json, plugin_type, plugin_state, protected_metadata_json,
		available, status_message, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(plugin_id) DO UPDATE SET
		manifest_json = excluded.manifest_json, plugin_type = excluded.plugin_type,
		plugin_state = excluded.plugin_state, protected_metadata_json = excluded.protected_metadata_json,
		available = excluded.available, status_message = excluded.status_message,
		updated_at = excluded.updated_at`,
		string(plugin.ID), string(manifestJSON), pluginTypeName(plugin.Type), pluginStateName(plugin.State),
		string(protectedJSON), boolToInt(plugin.Available), plugin.StatusMessage,
		formatTime(plugin.CreatedAt), formatTime(plugin.UpdatedAt))
	if err != nil {
		return fmt.Errorf("%w: update discovery Plugin row", installation.ErrPersistence)
	}
	result, err := tx.ExecContext(ctx, `UPDATE plugin_installation_operations SET phase = ?, updated_at = ?
		WHERE operation_id = ? AND phase = ?`, string(installation.PhaseCommitted), formatTime(time.Now().UTC()),
		string(id), string(installation.PhasePromoted))
	if err != nil {
		return fmt.Errorf("%w: commit operation phase", installation.ErrPersistence)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit installation transaction", installation.ErrPersistence)
	}
	return nil
}

func (r *Repository) GetOperation(ctx context.Context, id installation.OperationID) (installation.InstallationOperation, error) {
	if err := ctx.Err(); err != nil {
		return installation.InstallationOperation{}, err
	}
	op, err := scanOperation(r.db.QueryRowContext(ctx, "SELECT "+operationColumns+" FROM plugin_installation_operations WHERE operation_id = ?", string(id)))
	if errors.Is(err, sql.ErrNoRows) {
		return installation.InstallationOperation{}, installation.ErrOperationNotFound
	}
	if err != nil {
		return installation.InstallationOperation{}, fmt.Errorf("%w: read operation", installation.ErrPersistence)
	}
	return op, nil
}

func (r *Repository) ListIncompleteOperations(ctx context.Context) ([]installation.InstallationOperation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, "SELECT "+operationColumns+" FROM plugin_installation_operations WHERE phase IN (?, ?, ?, ?) ORDER BY created_at ASC, operation_id ASC",
		string(installation.PhasePreparing), string(installation.PhaseStaged), string(installation.PhaseVerified), string(installation.PhasePromoted))
	if err != nil {
		return nil, fmt.Errorf("%w: list incomplete operations", installation.ErrPersistence)
	}
	defer rows.Close()
	result := make([]installation.InstallationOperation, 0)
	for rows.Next() {
		op, err := scanOperation(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: decode operation", installation.ErrPersistence)
		}
		result = append(result, op)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate operations", installation.ErrPersistence)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) MarkRolledBack(ctx context.Context, id installation.OperationID, code string) error {
	return r.markTerminal(ctx, id, installation.PhaseRolledBack, code)
}
func (r *Repository) MarkFailed(ctx context.Context, id installation.OperationID, code string) error {
	return r.markTerminal(ctx, id, installation.PhaseFailed, code)
}
func (r *Repository) markTerminal(ctx context.Context, id installation.OperationID, target installation.Phase, code string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(code) == "" || len(code) > 64 {
		return installation.ErrInvalidRecord
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("%w: begin terminal transition", installation.ErrPersistence)
	}
	defer func() { _ = tx.Rollback() }()
	var phase, existing string
	err = tx.QueryRowContext(ctx, "SELECT phase, COALESCE(failure_code, '') FROM plugin_installation_operations WHERE operation_id = ?", string(id)).Scan(&phase, &existing)
	if errors.Is(err, sql.ErrNoRows) {
		return installation.ErrOperationNotFound
	}
	if err != nil {
		return fmt.Errorf("%w: read terminal operation", installation.ErrPersistence)
	}
	if installation.Phase(phase) == target && existing == code {
		return nil
	}
	switch installation.Phase(phase) {
	case installation.PhaseCommitted, installation.PhaseRolledBack, installation.PhaseFailed:
		return installation.ErrInvalidTransition
	}
	result, err := tx.ExecContext(ctx, "UPDATE plugin_installation_operations SET phase = ?, failure_code = ?, updated_at = ? WHERE operation_id = ? AND phase = ?",
		string(target), code, formatTime(time.Now().UTC()), string(id), phase)
	if err != nil {
		return fmt.Errorf("%w: mark terminal operation", installation.ErrPersistence)
	}
	if err := requireOneRow(result); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("%w: commit terminal operation", installation.ErrPersistence)
	}
	return nil
}

func (r *Repository) ListInstalledVersions(ctx context.Context, pluginID domain.PluginID) ([]installation.InstalledVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT plugin_id, version, artifact_path, source_reference,
		digest_algorithm, digest_value, verification_policy, COALESCE(verification_evidence_ref, ''),
		manifest_json, installed_at, selected FROM installed_plugin_versions
		WHERE plugin_id = ? ORDER BY installed_at ASC, version ASC`, string(pluginID))
	if err != nil {
		return nil, fmt.Errorf("%w: list installed versions", installation.ErrPersistence)
	}
	defer rows.Close()
	versions := make([]installation.InstalledVersion, 0)
	for rows.Next() {
		v, err := scanInstalledVersion(rows)
		if err != nil {
			return nil, fmt.Errorf("%w: decode installed version", installation.ErrPersistence)
		}
		versions = append(versions, v)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%w: iterate installed versions", installation.ErrPersistence)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return versions, nil
}

func (r *Repository) GetSelectedVersion(ctx context.Context, pluginID domain.PluginID) (installation.InstalledVersion, error) {
	if err := ctx.Err(); err != nil {
		return installation.InstalledVersion{}, err
	}
	v, err := scanInstalledVersion(r.db.QueryRowContext(ctx, `SELECT plugin_id, version, artifact_path, source_reference,
		digest_algorithm, digest_value, verification_policy, COALESCE(verification_evidence_ref, ''),
		manifest_json, installed_at, selected FROM installed_plugin_versions WHERE plugin_id = ? AND selected = 1`, string(pluginID)))
	if errors.Is(err, sql.ErrNoRows) {
		return installation.InstalledVersion{}, installation.ErrVersionNotFound
	}
	if err != nil {
		return installation.InstalledVersion{}, fmt.Errorf("%w: get selected version", installation.ErrPersistence)
	}
	return v, nil
}

func scanOperation(row scanner) (installation.InstallationOperation, error) {
	var o installation.InstallationOperation
	var id, pluginID, kind, phase string
	var staging, final, digestAlgorithm, digestValue, policy, manifest, failure sql.NullString
	var created, updated string
	err := row.Scan(&id, &pluginID, &o.TargetVersion, &kind, &phase, &staging, &final, &o.SourceReference,
		&digestAlgorithm, &digestValue, &policy, &manifest, &failure, &created, &updated)
	if err != nil {
		return o, err
	}
	o.OperationID, o.PluginID, o.Kind, o.Phase = installation.OperationID(id), domain.PluginID(pluginID), installation.Kind(kind), installation.Phase(phase)
	o.StagingPath, o.FinalArtifactPath = staging.String, final.String
	o.DigestAlgorithm, o.DigestValue, o.VerificationPolicy, o.ManifestJSON, o.FailureCode = digestAlgorithm.String, digestValue.String, policy.String, manifest.String, failure.String
	o.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return o, err
	}
	o.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	if err != nil {
		return o, err
	}
	return o, nil
}

func scanInstalledVersion(row scanner) (installation.InstalledVersion, error) {
	var v installation.InstalledVersion
	var pluginID string
	var installed string
	var selected int
	err := row.Scan(&pluginID, &v.Version, &v.ArtifactPath, &v.SourceReference, &v.DigestAlgorithm,
		&v.DigestValue, &v.VerificationPolicy, &v.VerificationEvidenceRef, &v.ManifestJSON, &installed, &selected)
	if err != nil {
		return v, err
	}
	v.PluginID = domain.PluginID(pluginID)
	v.InstalledAt, err = time.Parse(time.RFC3339Nano, installed)
	if err != nil {
		return v, err
	}
	if selected != 0 && selected != 1 {
		return v, fmt.Errorf("invalid selected flag")
	}
	v.Selected = selected == 1
	return v, nil
}

func validateOperationPath(value, root string) error {
	// Reuse the same path invariants without making installation depend on storage.
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") {
		return installation.ErrInvalidRecord
	}
	parts := strings.Split(value, "/")
	for _, part := range parts {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return installation.ErrInvalidRecord
		}
	}
	if parts[0] != root {
		return installation.ErrInvalidRecord
	}
	return nil
}
func nullableString(value string) any {
	if value == "" {
		return nil
	}
	return value
}
func formatTime(value time.Time) string { return value.UTC().Format(time.RFC3339Nano) }
func requireOneRow(result sql.Result) error {
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("%w: inspect update result", installation.ErrPersistence)
	}
	if n != 1 {
		return installation.ErrInvalidTransition
	}
	return nil
}
