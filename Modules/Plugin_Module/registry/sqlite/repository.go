package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/registry"

	_ "modernc.org/sqlite"
)

const schemaVersion = 5

type Repository struct {
	db *sql.DB
}

func New(ctx context.Context, databasePath string) (*Repository, error) {
	if databasePath == "" {
		return nil, fmt.Errorf("SQLite database path is required")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(databasePath) {
		return nil, fmt.Errorf("SQLite database path must be absolute")
	}
	if info, err := os.Lstat(databasePath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("SQLite database path must not be a symbolic link")
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect plugin registry database path: %w", err)
	}

	db, err := sql.Open("sqlite", databasePath)
	if err != nil {
		return nil, fmt.Errorf("open plugin registry database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)

	repository := &Repository{db: db}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("connect plugin registry database: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(databasePath, 0o600); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("secure plugin registry database permissions: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, "PRAGMA busy_timeout = 5000"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configure SQLite busy timeout: %w", err)
	}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable SQLite foreign key enforcement: %w", err)
	}
	var foreignKeys int
	if err := db.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&foreignKeys); err != nil || foreignKeys != 1 {
		_ = db.Close()
		return nil, fmt.Errorf("verify SQLite foreign key enforcement: enabled=%d err=%v", foreignKeys, err)
	}
	if err := repository.migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return repository, nil
}

func (r *Repository) Close() error {
	if r == nil || r.db == nil {
		return nil
	}
	return r.db.Close()
}

func (r *Repository) migrate(ctx context.Context) error {
	var version int
	if err := r.db.QueryRowContext(ctx, "PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read plugin registry schema version: %w", err)
	}
	if version > schemaVersion {
		return fmt.Errorf("plugin registry schema version %d is newer than supported version %d", version, schemaVersion)
	}
	if version == schemaVersion {
		return nil
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin plugin registry migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if version < 1 {
		const createSchema = `CREATE TABLE plugins (
			plugin_id TEXT PRIMARY KEY NOT NULL,
			manifest_json TEXT NOT NULL,
			plugin_type TEXT NOT NULL,
			plugin_state TEXT NOT NULL,
			protected_metadata_json TEXT NOT NULL,
			available INTEGER NOT NULL CHECK (available IN (0, 1)),
			status_message TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`
		if _, err := tx.ExecContext(ctx, createSchema); err != nil {
			return fmt.Errorf("create plugin registry schema: %w", err)
		}
		version = 1
	}
	if version < 2 {
		statements := []string{
			`CREATE TABLE installed_plugin_versions (
				plugin_id TEXT NOT NULL,
				version TEXT NOT NULL,
				artifact_path TEXT NOT NULL,
				source_reference TEXT NOT NULL,
				digest_algorithm TEXT NOT NULL,
				digest_value TEXT NOT NULL,
				verification_policy TEXT NOT NULL,
				verification_evidence_ref TEXT,
				manifest_json TEXT NOT NULL,
				installed_at TEXT NOT NULL,
				selected INTEGER NOT NULL CHECK (selected IN (0, 1)),
				PRIMARY KEY (plugin_id, version)
			)`,
			`CREATE UNIQUE INDEX one_selected_version_per_plugin
				ON installed_plugin_versions(plugin_id) WHERE selected = 1`,
			`CREATE TABLE plugin_installation_operations (
				operation_id TEXT PRIMARY KEY NOT NULL,
				plugin_id TEXT NOT NULL,
				target_version TEXT NOT NULL,
				operation_kind TEXT NOT NULL CHECK (operation_kind IN ('INSTALL', 'UPDATE')),
				phase TEXT NOT NULL CHECK (phase IN ('PREPARING', 'STAGED', 'VERIFIED', 'PROMOTED', 'COMMITTED', 'ROLLED_BACK', 'FAILED')),
				staging_path TEXT,
				final_artifact_path TEXT,
				source_reference TEXT NOT NULL,
				digest_algorithm TEXT,
				digest_value TEXT,
				verification_policy TEXT,
				manifest_json TEXT,
				failure_code TEXT,
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			)`,
			`CREATE UNIQUE INDEX one_active_installation_per_plugin
				ON plugin_installation_operations(plugin_id)
				WHERE phase IN ('PREPARING', 'STAGED', 'VERIFIED', 'PROMOTED')`,
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("create plugin installation schema: %w", err)
			}
		}
	}
	if version < 3 {
		const createTrustState = `CREATE TABLE plugin_trust_configuration_state (
			singleton_id INTEGER PRIMARY KEY NOT NULL CHECK (singleton_id = 1),
			sequence INTEGER NOT NULL CHECK (sequence > 0),
			digest BLOB NOT NULL CHECK (length(digest) = 32),
			accepted_at TEXT NOT NULL
		)`
		if _, err := tx.ExecContext(ctx, createTrustState); err != nil {
			return fmt.Errorf("create Plugin trust configuration state schema: %w", err)
		}
		version = 3
	}
	if version < 4 {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE plugin_installation_operations ADD COLUMN descriptor_json TEXT`); err != nil {
			return fmt.Errorf("add persisted artifact descriptor to installation journal: %w", err)
		}
	}
	if version < 5 {
		statements := []string{
			`CREATE TABLE plugin_lifecycle_records (
				plugin_id TEXT PRIMARY KEY NOT NULL REFERENCES plugins(plugin_id) ON DELETE RESTRICT,
				selected_version TEXT NOT NULL CHECK (length(selected_version) BETWEEN 1 AND 128),
				desired_state TEXT NOT NULL CHECK (desired_state IN ('DISABLED', 'ENABLED')),
				runtime_observation TEXT NOT NULL CHECK (runtime_observation IN ('UNKNOWN', 'STOPPED', 'STARTING', 'RUNNING', 'STOPPING', 'FAILED')),
				revision INTEGER NOT NULL CHECK (revision > 0),
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL
			)`,
			`CREATE TABLE plugin_lifecycle_operations (
				operation_id TEXT PRIMARY KEY NOT NULL CHECK (length(operation_id) BETWEEN 1 AND 128),
				request_fingerprint TEXT NOT NULL CHECK (length(request_fingerprint) = 64),
				principal_id TEXT NOT NULL CHECK (length(principal_id) BETWEEN 1 AND 128),
				plugin_id TEXT NOT NULL REFERENCES plugins(plugin_id) ON DELETE RESTRICT,
				plugin_version TEXT NOT NULL CHECK (length(plugin_version) BETWEEN 1 AND 128),
				action TEXT NOT NULL CHECK (action IN ('ENABLE', 'DISABLE')),
				phase TEXT NOT NULL CHECK (phase IN ('REQUESTED', 'AUTHORIZED', 'APPLYING', 'SUCCEEDED', 'FAILED', 'RECOVERY_REQUIRED')),
				failure_code TEXT CHECK (failure_code IS NULL OR length(failure_code) <= 64),
				decision_id TEXT CHECK (decision_id IS NULL OR length(decision_id) BETWEEN 1 AND 128),
				policy_revision TEXT CHECK (policy_revision IS NULL OR length(policy_revision) BETWEEN 1 AND 128),
				evaluated_at TEXT,
				reason_code TEXT CHECK (reason_code IS NULL OR length(reason_code) <= 64),
				created_at TEXT NOT NULL,
				updated_at TEXT NOT NULL,
				UNIQUE (operation_id, plugin_id),
				CHECK (
					(decision_id IS NULL AND policy_revision IS NULL AND evaluated_at IS NULL AND reason_code IS NULL)
					OR
					(decision_id IS NOT NULL AND policy_revision IS NOT NULL AND evaluated_at IS NOT NULL AND reason_code IS NOT NULL)
				)
			)`,
			`CREATE INDEX plugin_lifecycle_operations_plugin_created
				ON plugin_lifecycle_operations(plugin_id, created_at, operation_id)`,
			`CREATE INDEX plugin_lifecycle_operations_phase_updated
				ON plugin_lifecycle_operations(phase, updated_at, operation_id)`,
			`CREATE TABLE plugin_lifecycle_leases (
				plugin_id TEXT PRIMARY KEY NOT NULL REFERENCES plugins(plugin_id) ON DELETE RESTRICT,
				operation_id TEXT NOT NULL UNIQUE,
				fencing_token INTEGER NOT NULL CHECK (fencing_token > 0),
				expires_at TEXT NOT NULL,
				FOREIGN KEY (operation_id, plugin_id)
					REFERENCES plugin_lifecycle_operations(operation_id, plugin_id) ON DELETE RESTRICT
			)`,
			`CREATE TABLE plugin_lifecycle_fencing_tokens (
				plugin_id TEXT PRIMARY KEY NOT NULL REFERENCES plugins(plugin_id) ON DELETE RESTRICT,
				last_token INTEGER NOT NULL CHECK (last_token > 0)
			)`,
		}
		for _, statement := range statements {
			if _, err := tx.ExecContext(ctx, statement); err != nil {
				return fmt.Errorf("create Plugin lifecycle persistence schema: %w", err)
			}
		}
		version = 5
	}
	if version != schemaVersion {
		return fmt.Errorf("plugin registry migration stopped at schema version %d, expected %d", version, schemaVersion)
	}
	if _, err := tx.ExecContext(ctx, "PRAGMA user_version = 5"); err != nil {
		return fmt.Errorf("advance plugin registry schema version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit plugin registry migration: %w", err)
	}
	return nil
}

func (r *Repository) Create(ctx context.Context, plugin domain.Plugin) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("%w: %v", registry.ErrInvalidPlugin, err)
	}

	manifestJSON, err := json.Marshal(plugin.Manifest)
	if err != nil {
		return fmt.Errorf("encode plugin manifest: %w", err)
	}
	protectedJSON, err := json.Marshal(plugin.Protected)
	if err != nil {
		return fmt.Errorf("encode plugin protected metadata: %w", err)
	}

	result, err := r.db.ExecContext(ctx, `INSERT INTO plugins (
		plugin_id, manifest_json, plugin_type, plugin_state,
		protected_metadata_json, available, status_message, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(plugin_id) DO NOTHING`,
		string(plugin.ID), string(manifestJSON), pluginTypeName(plugin.Type), pluginStateName(plugin.State),
		string(protectedJSON), boolToInt(plugin.Available), plugin.StatusMessage,
		plugin.CreatedAt.UTC().Format(time.RFC3339Nano), plugin.UpdatedAt.UTC().Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("create plugin %q: %w", plugin.ID, err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect plugin creation result: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("%w: %s", registry.ErrAlreadyExists, plugin.ID)
	}
	return nil
}

func (r *Repository) Get(ctx context.Context, id domain.PluginID) (domain.Plugin, error) {
	if err := ctx.Err(); err != nil {
		return domain.Plugin{}, err
	}
	row := r.db.QueryRowContext(ctx, `SELECT plugin_id, manifest_json, plugin_type, plugin_state,
		protected_metadata_json, available, status_message, created_at, updated_at
		FROM plugins WHERE plugin_id = ?`, string(id))
	plugin, err := scanPlugin(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Plugin{}, fmt.Errorf("%w: %s", registry.ErrNotFound, id)
	}
	if err != nil {
		return domain.Plugin{}, fmt.Errorf("get plugin %q: %w", id, err)
	}
	if err := ctx.Err(); err != nil {
		return domain.Plugin{}, err
	}
	return plugin.Clone(), nil
}

func (r *Repository) List(ctx context.Context) ([]domain.Plugin, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT plugin_id, manifest_json, plugin_type, plugin_state,
		protected_metadata_json, available, status_message, created_at, updated_at
		FROM plugins ORDER BY plugin_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list plugins: %w", err)
	}
	defer rows.Close()

	plugins := make([]domain.Plugin, 0)
	for rows.Next() {
		plugin, err := scanPlugin(rows)
		if err != nil {
			return nil, fmt.Errorf("decode plugin registry row: %w", err)
		}
		plugins = append(plugins, plugin.Clone())
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate plugin registry rows: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return plugins, nil
}

type scanner interface{ Scan(...any) error }

func scanPlugin(row scanner) (domain.Plugin, error) {
	var plugin domain.Plugin
	var manifestJSON, protectedJSON, pluginType, pluginState, createdAt, updatedAt string
	var available int
	if err := row.Scan(&plugin.ID, &manifestJSON, &pluginType, &pluginState, &protectedJSON,
		&available, &plugin.StatusMessage, &createdAt, &updatedAt); err != nil {
		return domain.Plugin{}, err
	}

	if err := json.Unmarshal([]byte(manifestJSON), &plugin.Manifest); err != nil {
		return domain.Plugin{}, fmt.Errorf("decode manifest JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(protectedJSON), &plugin.Protected); err != nil {
		return domain.Plugin{}, fmt.Errorf("decode protected metadata JSON: %w", err)
	}
	var err error
	plugin.Type, err = parsePluginType(pluginType)
	if err != nil {
		return domain.Plugin{}, err
	}
	plugin.State, err = parsePluginState(pluginState)
	if err != nil {
		return domain.Plugin{}, err
	}
	if available != 0 && available != 1 {
		return domain.Plugin{}, fmt.Errorf("invalid available value %d", available)
	}
	plugin.Available = available == 1
	plugin.CreatedAt, err = time.Parse(time.RFC3339Nano, createdAt)
	if err != nil {
		return domain.Plugin{}, fmt.Errorf("parse created_at: %w", err)
	}
	plugin.UpdatedAt, err = time.Parse(time.RFC3339Nano, updatedAt)
	if err != nil {
		return domain.Plugin{}, fmt.Errorf("parse updated_at: %w", err)
	}
	return plugin, nil
}

func pluginTypeName(value domain.PluginType) string {
	switch value {
	case domain.PluginTypeActive:
		return "active"
	case domain.PluginTypePassive:
		return "passive"
	default:
		return "unspecified"
	}
}
func parsePluginType(value string) (domain.PluginType, error) {
	switch value {
	case "active":
		return domain.PluginTypeActive, nil
	case "passive":
		return domain.PluginTypePassive, nil
	default:
		return 0, fmt.Errorf("unknown plugin type %q", value)
	}
}
func pluginStateName(value domain.PluginState) string {
	switch value {
	case domain.PluginStateInstalled:
		return "installed"
	case domain.PluginStateEnabled:
		return "enabled"
	case domain.PluginStateDisabled:
		return "disabled"
	case domain.PluginStateRunning:
		return "running"
	case domain.PluginStateError:
		return "error"
	default:
		return "unspecified"
	}
}
func parsePluginState(value string) (domain.PluginState, error) {
	switch value {
	case "installed":
		return domain.PluginStateInstalled, nil
	case "enabled":
		return domain.PluginStateEnabled, nil
	case "disabled":
		return domain.PluginStateDisabled, nil
	case "running":
		return domain.PluginStateRunning, nil
	case "error":
		return domain.PluginStateError, nil
	default:
		return 0, fmt.Errorf("unknown plugin state %q", value)
	}
}
func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
