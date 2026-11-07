package storage

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestAdapterCreatesAndOpensOperationArtifact(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	stagingPath, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "op-123")
	if err != nil {
		t.Fatal(err)
	}
	if stagingPath != "temporary/op-123" {
		t.Fatalf("staging path = %q", stagingPath)
	}

	file, err := adapter.CreateArtifactFile(context.Background(), stagingPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("zip-content")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	if _, err := adapter.CreateArtifactFile(context.Background(), stagingPath); !errors.Is(err, os.ErrExist) {
		t.Fatalf("expected exclusive create to reject existing artifact, got %v", err)
	}

	readFile, err := adapter.OpenArtifactFile(context.Background(), stagingPath)
	if err != nil {
		t.Fatal(err)
	}
	defer readFile.Close()
	content, err := io.ReadAll(readFile)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "zip-content" {
		t.Fatalf("artifact content = %q", content)
	}

	info, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), "temporary", "op-123", OperationArtifactFileName))
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("artifact permissions = %o, want 600", info.Mode().Perm())
	}
}

func TestAdapterRejectsInvalidWorkspacePaths(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	for _, value := range []string{
		"", ".", "../escape", "/absolute", "temporary/../escape",
		"temporary\\op-1", "installed/plugin/1.0.0", "temporary/op-1/nested",
	} {
		if _, err := adapter.CreateArtifactFile(context.Background(), value); err == nil {
			t.Errorf("expected workspace %q to be rejected", value)
		}
	}
}

func TestAdapterRejectsWorkspaceSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges")
	}
	base := t.TempDir()
	layout, err := NewLayout(Config{RootOverride: filepath.Join(base, "pride")})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	outside := t.TempDir()
	workspace := filepath.Join(layout.PluginStorageRoot(), "temporary", "op-escape")
	if err := os.Symlink(outside, workspace); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CreateArtifactFile(context.Background(), "temporary/op-escape"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
}

func TestAdapterHonorsCanceledContext(t *testing.T) {
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
	if _, err := adapter.CreateTemporaryOperationDirectory(ctx, "op-canceled"); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}


func TestAdapterCreatesOperationDirectoryExclusively(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	if _, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "op-exclusive"); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "op-exclusive"); !errors.Is(err, ErrOperationDirectoryExists) {
		t.Fatalf("expected exclusive operation directory error, got %v", err)
	}
}

func TestAdapterRejectsSymlinkedTemporaryParent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges")
	}
	base := t.TempDir()
	layout, err := NewLayout(Config{RootOverride: filepath.Join(base, "pride")})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	temporary := filepath.Join(layout.PluginStorageRoot(), "temporary")
	if err := os.Remove(temporary); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, temporary); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "op-escape"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected symlinked parent rejection, got %v", err)
	}
}
