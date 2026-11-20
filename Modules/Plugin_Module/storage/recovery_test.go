package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAdapterQuarantinesIncompleteWorkspaceIdempotently(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "recover-temp-1")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}

	if err := adapter.QuarantineOperation(context.Background(), workspace, "recover-temp-1"); err != nil {
		t.Fatal(err)
	}
	quarantined := filepath.Join(layout.PluginStorageRoot(), "quarantine", "recover-temp-1", OperationArtifactFileName)
	if _, err := os.Stat(quarantined); err != nil {
		t.Fatalf("quarantined artifact missing: %v", err)
	}
	if err := adapter.QuarantineOperation(context.Background(), workspace, "recover-temp-1"); err != nil {
		t.Fatalf("repeated quarantine should be idempotent: %v", err)
	}
}

func TestAdapterQuarantinesPromotedArtifactAndAllowsConfinedRead(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "recover-final-1")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write([]byte("verified zip")); err != nil {
		t.Fatal(err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CreateExtractionRoot(context.Background(), workspace); err != nil {
		t.Fatal(err)
	}
	payload, err := adapter.CreateExtractionFile(context.Background(), workspace, "package/main.lua")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := payload.Write([]byte("return true")); err != nil {
		t.Fatal(err)
	}
	if err := payload.Close(); err != nil {
		t.Fatal(err)
	}
	finalPath := "installed/recover.plugin/1.0.0"
	if err := adapter.Promote(context.Background(), workspace, finalPath); err != nil {
		t.Fatal(err)
	}

	file, err := adapter.OpenArtifactFile(context.Background(), finalPath)
	if err != nil {
		t.Fatalf("open promoted artifact through confined adapter: %v", err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.QuarantineOperation(context.Background(), finalPath, "recover-final-1"); err != nil {
		t.Fatal(err)
	}
	quarantined := filepath.Join(layout.PluginStorageRoot(), "quarantine", "recover-final-1", OperationArtifactFileName)
	if _, err := os.Stat(quarantined); err != nil {
		t.Fatalf("promoted artifact was not quarantined: %v", err)
	}
	if _, err := adapter.OpenArtifactFile(context.Background(), finalPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("original installed path should be absent after quarantine, got %v", err)
	}
}

func TestAdapterQuarantineRejectsUnsafeSourceAndDestinationCollision(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	if err := adapter.QuarantineOperation(context.Background(), "../escape", "recover-unsafe"); err == nil {
		t.Fatal("expected unsafe source path rejection")
	}
	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "recover-collision")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(layout.PluginStorageRoot(), "quarantine", "recover-collision"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := adapter.QuarantineOperation(context.Background(), workspace, "recover-collision"); err == nil {
		t.Fatal("expected existing quarantine destination rejection")
	}
	if _, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), filepath.FromSlash(workspace))); err != nil {
		t.Fatalf("source must remain when quarantine destination conflicts: %v", err)
	}
}
