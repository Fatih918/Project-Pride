package registry

import (
	"context"
	"errors"
	"testing"

	"pride-plugin-module/domain"
)

func TestMemoryRepositoryListIsSortedAndDefensivelyCopied(t *testing.T) {
	repo := NewMemoryRepository()
	for _, id := range []string{"com.example.zeta", "com.example.alpha"} {
		plugin := testPlugin()
		plugin.ID = domain.PluginID(id)
		plugin.Manifest.PluginID = plugin.ID
		if err := repo.Create(context.Background(), plugin); err != nil {
			t.Fatalf("Create(%q) error = %v", id, err)
		}
	}

	plugins, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(plugins) != 2 || plugins[0].ID != "com.example.alpha" || plugins[1].ID != "com.example.zeta" {
		t.Fatalf("List() IDs = %+v, want sorted IDs", plugins)
	}
	plugins[0].Manifest.Name = "mutated"
	again, err := repo.List(context.Background())
	if err != nil {
		t.Fatalf("second List() error = %v", err)
	}
	if again[0].Manifest.Name == "mutated" {
		t.Fatal("List() returned a mutable reference to stored data")
	}
}

func TestMemoryRepositoryListHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := NewMemoryRepository().List(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("List() error = %v, want context.Canceled", err)
	}
}
