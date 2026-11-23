package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListAndDeleteQuarantineEntries(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "retention-op-1")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write([]byte("quarantine payload")); err != nil {
		t.Fatal(err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.QuarantineOperation(context.Background(), workspace, "retention-op-1"); err != nil {
		t.Fatal(err)
	}

	entries, err := adapter.ListQuarantineEntries(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].OperationID != "retention-op-1" || entries[0].SizeBytes != int64(len("quarantine payload")) {
		t.Fatalf("unexpected quarantine inventory: %+v", entries)
	}
	if err := adapter.DeleteQuarantineOperation(context.Background(), "retention-op-1"); err != nil {
		t.Fatal(err)
	}
	if err := adapter.DeleteQuarantineOperation(context.Background(), "retention-op-1"); err != nil {
		t.Fatalf("deleting an already-removed quarantine entry should be idempotent: %v", err)
	}
	if _, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), "quarantine", "retention-op-1")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("quarantine directory still exists: %v", err)
	}
}

func TestListQuarantineRejectsSymlinkInsteadOfFollowingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges")
	}
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "sensitive"), []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(layout.PluginStorageRoot(), "quarantine", "linked-op")); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.ListQuarantineEntries(context.Background()); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
	if err := adapter.DeleteQuarantineOperation(context.Background(), "linked-op"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected symlink deletion rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, "sensitive")); err != nil {
		t.Fatalf("outside target was changed: %v", err)
	}
}

func TestListQuarantineHonorsContext(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := adapter.ListQuarantineEntries(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
	if err := adapter.DeleteQuarantineOperation(ctx, "retention-op"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
