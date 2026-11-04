package registry

import (
	"context"
	"errors"
	"testing"

	"pride-plugin-module/domain"
)

func testPlugin() domain.Plugin {
	id := domain.PluginID("com.example.notes")
	return domain.Plugin{
		ID: id,
		Manifest: domain.Manifest{
			PluginID:            id,
			Name:                "Notes",
			Version:             "1.0.0",
			Description:         "Example notes plugin",
			Author:              "PRIDE",
			License:             "MIT",
			EntryPoint:          "main.go",
			MinimumPRIDEVersion: "1.0.0",
		},
		Type:  domain.PluginTypeActive,
		State: domain.PluginStateInstalled,
	}
}

func TestMemoryRepositoryCreateAndGet(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	plugin := testPlugin()
	if err := repo.Create(ctx, plugin); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	got, err := repo.Get(ctx, plugin.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Manifest.Name != plugin.Manifest.Name {
		t.Fatalf("Get() name = %q, want %q", got.Manifest.Name, plugin.Manifest.Name)
	}
}

func TestMemoryRepositoryRejectsDuplicateIdentity(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	plugin := testPlugin()
	if err := repo.Create(ctx, plugin); err != nil {
		t.Fatalf("first Create() error = %v", err)
	}
	if err := repo.Create(ctx, plugin); !errors.Is(err, ErrAlreadyExists) {
		t.Fatalf("second Create() error = %v, want ErrAlreadyExists", err)
	}
}

func TestMemoryRepositoryReturnsNotFound(t *testing.T) {
	_, err := NewMemoryRepository().Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get() error = %v, want ErrNotFound", err)
	}
}

func TestMemoryRepositoryDefensivelyCopiesRecords(t *testing.T) {
	repo := NewMemoryRepository()
	ctx := context.Background()
	plugin := testPlugin()
	plugin.Manifest.Keywords = []string{"original"}
	if err := repo.Create(ctx, plugin); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.Get(ctx, plugin.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	got.Manifest.Keywords[0] = "mutated"
	again, err := repo.Get(ctx, plugin.ID)
	if err != nil {
		t.Fatalf("second Get() error = %v", err)
	}
	if again.Manifest.Keywords[0] != "original" {
		t.Fatalf("stored keyword = %q, want original", again.Manifest.Keywords[0])
	}
}
