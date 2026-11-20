package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAdapterPromotesWorkspaceAtomically(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "promote-1")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write([]byte("verified-zip")); err != nil {
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

	finalPath := "installed/example.plugin/1.0.0"
	if err := adapter.Promote(context.Background(), workspace, finalPath); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), filepath.FromSlash(workspace))); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("staging workspace still exists after promotion: %v", err)
	}
	for _, relative := range []string{
		filepath.Join("installed", "example.plugin", "1.0.0", OperationArtifactFileName),
		filepath.Join("installed", "example.plugin", "1.0.0", "package", "main.lua"),
	} {
		if _, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), relative)); err != nil {
			t.Fatalf("promoted file %q missing: %v", relative, err)
		}
	}
}

func TestAdapterPromotionDoesNotReplaceExistingVersion(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "promote-2")
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write([]byte("new-artifact")); err != nil {
		t.Fatal(err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.CreateExtractionRoot(context.Background(), workspace); err != nil {
		t.Fatal(err)
	}

	destination := filepath.Join(layout.PluginStorageRoot(), "installed", "example.plugin", "1.0.0")
	if err := os.MkdirAll(destination, 0o700); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(destination, "keep.txt")
	if err := os.WriteFile(sentinel, []byte("existing"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := adapter.Promote(context.Background(), workspace, "installed/example.plugin/1.0.0"); !errors.Is(err, ErrArtifactAlreadyExists) {
		t.Fatalf("expected existing destination conflict, got %v", err)
	}
	content, err := os.ReadFile(sentinel)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "existing" {
		t.Fatalf("existing installed version was modified: %q", content)
	}
	if _, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), filepath.FromSlash(workspace))); err != nil {
		t.Fatalf("staging workspace should remain after conflict: %v", err)
	}
}

func TestAdapterPromotionRejectsUnsafePathsAndIncompleteWorkspace(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "promote-3")
	if err != nil {
		t.Fatal(err)
	}
	for _, candidate := range []string{
		"", "../escape", "/installed/plugin/1.0.0",
		"installed/../plugin/1.0.0", "installed/plugin/../1.0.0",
		"installed/plugin/1.0.0/nested", "temporary/plugin/1.0.0",
		"installed\\plugin\\1.0.0",
	} {
		if err := adapter.Promote(context.Background(), workspace, candidate); err == nil {
			t.Errorf("expected unsafe destination %q to be rejected", candidate)
		}
	}
	if err := adapter.Promote(context.Background(), workspace, "installed/example.plugin/1.0.0"); err == nil {
		t.Fatal("expected incomplete workspace to be rejected")
	}
}

func TestAdapterPromotionHonorsCanceledContext(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "promote-4")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := adapter.Promote(ctx, workspace, "installed/example.plugin/1.0.0"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
