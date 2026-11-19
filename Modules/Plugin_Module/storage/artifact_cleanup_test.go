package storage

import (
	"context"
	"errors"
	"os"
	"testing"
)

func TestRemoveArtifactFileRemovesOnlyRegularArtifact(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	workspace, err := adapter.CreateTemporaryOperationDirectory(context.Background(), "remove-1")
	if err != nil {
		t.Fatal(err)
	}
	file, err := adapter.CreateArtifactFile(context.Background(), workspace)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("partial")); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if err := adapter.RemoveArtifactFile(context.Background(), workspace); err != nil {
		t.Fatal(err)
	}
	if _, err := adapter.OpenArtifactFile(context.Background(), workspace); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected removed artifact, got %v", err)
	}
	if err := adapter.RemoveArtifactFile(context.Background(), workspace); err != nil {
		t.Fatalf("removing absent artifact should be idempotent: %v", err)
	}
}
