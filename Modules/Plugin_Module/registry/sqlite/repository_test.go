package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/registry"

	_ "modernc.org/sqlite"
)

func testPlugin(id string) domain.Plugin {
	now := time.Date(2026, 11, 4, 8, 0, 0, 123456789, time.FixedZone("WIB", 7*60*60))
	return domain.Plugin{
		ID: domain.PluginID(id),
		Manifest: domain.Manifest{
			PluginID: domain.PluginID(id), Name: "Example", Version: "1.2.3",
			Description: "Test plugin", Author: "PRIDE", License: "MIT",
			EntryPoint: "main", MinimumPRIDEVersion: "1.0.0",
			Keywords: []string{"tools", "test"},
		},
		Type: domain.PluginTypeActive, State: domain.PluginStateInstalled,
		Protected: domain.ProtectedMetadata{
			Capabilities:        []domain.Capability{{ID: "text.generate", Name: "Text"}},
			PermissionIDs:       []string{"storage.read"},
			CompatibilityStatus: "compatible",
			Attributes:          map[string]any{"nested": map[string]any{"enabled": true}, "labels": []any{"a", "b"}},
		},
		Available: true, StatusMessage: "ready", CreatedAt: now, UpdatedAt: now,
	}
}

func openTestRepository(t *testing.T) (*Repository, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "plugin-registry.sqlite")
	repository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repository.Close() })
	return repository, path
}

func TestRepositoryCreateGetAndReopen(t *testing.T) {
	repository, path := openTestRepository(t)
	want := testPlugin("plugin.example")
	want.CreatedAt = want.CreatedAt.UTC()
	want.UpdatedAt = want.UpdatedAt.UTC()
	if err := repository.Create(context.Background(), want); err != nil {
		t.Fatal(err)
	}
	got, err := repository.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want.Clone()) {
		t.Fatalf("round trip mismatch:\n got: %#v\nwant: %#v", got, want)
	}

	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err = reopened.Get(context.Background(), want.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, want.Clone()) {
		t.Fatal("record did not persist after reopening database")
	}
}

func TestRepositoryDuplicateAndNotFound(t *testing.T) {
	repository, _ := openTestRepository(t)
	plugin := testPlugin("plugin.example")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	if err := repository.Create(context.Background(), plugin); !errors.Is(err, registry.ErrAlreadyExists) {
		t.Fatalf("duplicate error = %v", err)
	}
	if _, err := repository.Get(context.Background(), "missing"); !errors.Is(err, registry.ErrNotFound) {
		t.Fatalf("missing error = %v", err)
	}
}

func TestRepositoryListIsOrderedAndReturnsDefensiveCopies(t *testing.T) {
	repository, _ := openTestRepository(t)
	for _, id := range []string{"plugin.z", "plugin.a", "plugin.m"} {
		if err := repository.Create(context.Background(), testPlugin(id)); err != nil {
			t.Fatal(err)
		}
	}
	plugins, err := repository.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(plugins) != 3 || plugins[0].ID != "plugin.a" || plugins[1].ID != "plugin.m" || plugins[2].ID != "plugin.z" {
		t.Fatalf("unexpected order: %#v", plugins)
	}
	plugins[0].Manifest.Keywords[0] = "mutated"
	plugins[0].Protected.Attributes["nested"].(map[string]any)["enabled"] = false
	again, err := repository.Get(context.Background(), "plugin.a")
	if err != nil {
		t.Fatal(err)
	}
	if again.Manifest.Keywords[0] != "tools" || again.Protected.Attributes["nested"].(map[string]any)["enabled"] != true {
		t.Fatal("returned plugin mutated persisted data")
	}
}

func TestRepositoryRejectsInvalidPluginAndCanceledContext(t *testing.T) {
	repository, _ := openTestRepository(t)
	if err := repository.Create(context.Background(), domain.Plugin{}); !errors.Is(err, registry.ErrInvalidPlugin) {
		t.Fatalf("invalid plugin error = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.Create(ctx, testPlugin("plugin.cancelled")); !errors.Is(err, context.Canceled) {
		t.Fatalf("create cancellation = %v", err)
	}
	if _, err := repository.Get(ctx, "plugin.cancelled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("get cancellation = %v", err)
	}
	if _, err := repository.List(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("list cancellation = %v", err)
	}
}

func TestRepositoryMalformedJSONReturnsError(t *testing.T) {
	repository, _ := openTestRepository(t)
	plugin := testPlugin("plugin.malformed")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.db.ExecContext(context.Background(), "UPDATE plugins SET manifest_json = ? WHERE plugin_id = ?", "{", plugin.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := repository.Get(context.Background(), plugin.ID); err == nil {
		t.Fatal("expected malformed JSON error")
	}
}

func TestRepositoryRejectsFutureSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "future.sqlite")
	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("PRAGMA user_version = 99"); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	repository, err := New(context.Background(), path)
	if err == nil {
		repository.Close()
		t.Fatal("expected unsupported future schema error")
	}
}
