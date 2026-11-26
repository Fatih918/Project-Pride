package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"pride-plugin-module/domain"

	_ "modernc.org/sqlite"
)

func TestLifecycleSchemaV5CreatesTablesAndConstraints(t *testing.T) {
	repository, _ := openTestRepository(t)

	var version int
	if err := repository.db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 5 {
		t.Fatalf("schema version = %d, want 5", version)
	}

	for _, table := range []string{
		"plugin_lifecycle_records",
		"plugin_lifecycle_operations",
		"plugin_lifecycle_leases",
		"plugin_lifecycle_fencing_tokens",
	} {
		var name string
		if err := repository.db.QueryRowContext(context.Background(),
			"SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&name); err != nil {
			t.Fatalf("table %q missing: %v", table, err)
		}
	}

	plugin := testPlugin("plugin.lifecycle.schema")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.ExecContext(context.Background(), `INSERT INTO plugin_lifecycle_fencing_tokens (plugin_id, last_token) VALUES (?, 0)`, plugin.ID); err == nil {
		t.Fatal("fencing token constraint accepted zero")
	}
	if _, err := repository.db.ExecContext(context.Background(), `INSERT INTO plugin_lifecycle_records (
		plugin_id, selected_version, desired_state, runtime_observation, revision, created_at, updated_at
	) VALUES (?, ?, 'RUNNING', 'UNKNOWN', 1, '2026-11-26T00:00:00Z', '2026-11-26T00:00:00Z')`,
		plugin.ID, plugin.Manifest.Version); err == nil {
		t.Fatal("desired-state constraint accepted an invalid value")
	}
}

func TestLifecycleSchemaV5UpgradePreservesV4Data(t *testing.T) {
	path := filepath.Join(t.TempDir(), "plugin-registry.sqlite")
	repository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	plugin := testPlugin("plugin.lifecycle.migration")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}

	// Revert only the newly introduced v5 objects to model an existing v4
	// database while preserving all registry, installation, and trust data.
	for _, table := range []string{
		"plugin_lifecycle_leases",
		"plugin_lifecycle_fencing_tokens",
		"plugin_lifecycle_operations",
		"plugin_lifecycle_records",
	} {
		if _, err := repository.db.ExecContext(context.Background(), "DROP TABLE "+table); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := repository.db.ExecContext(context.Background(), "PRAGMA user_version = 4"); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}

	upgraded, err := New(context.Background(), path)
	if err != nil {
		t.Fatalf("upgrade v4 database: %v", err)
	}
	defer upgraded.Close()

	var version int
	if err := upgraded.db.QueryRowContext(context.Background(), "PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 5 {
		t.Fatalf("schema version after upgrade = %d, want 5", version)
	}
	got, err := upgraded.Get(context.Background(), plugin.ID)
	if err != nil {
		t.Fatalf("read preserved Plugin after upgrade: %v", err)
	}
	if got.ID != plugin.ID || got.Manifest.Version != plugin.Manifest.Version {
		t.Fatalf("v4 Plugin data changed during migration: %#v", got)
	}
}

func TestLifecycleSchemaV5RejectsInvalidAuthorizationEvidenceShape(t *testing.T) {
	repository, _ := openTestRepository(t)
	plugin := testPlugin("plugin.lifecycle.evidence")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	_, err := repository.db.ExecContext(context.Background(), `INSERT INTO plugin_lifecycle_operations (
		operation_id, request_fingerprint, principal_id, plugin_id, plugin_version, action, phase,
		decision_id, policy_revision, evaluated_at, reason_code, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, 'ENABLE', 'REQUESTED', ?, NULL, NULL, NULL, ?, ?)`,
		"operation-evidence", "0123456789012345678901234567890123456789012345678901234567890123",
		"principal-1", plugin.ID, plugin.Manifest.Version, "decision-1",
		"2026-11-26T00:00:00Z", "2026-11-26T00:00:00Z")
	if err == nil {
		t.Fatal("authorization evidence constraint accepted a partial decision")
	}
}

var _ = sql.ErrNoRows
var _ domain.Plugin
