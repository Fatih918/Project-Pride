package registry

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"pride-plugin-module/domain"
)

// MemoryRepository is a deterministic test/local-development repository.
// It is not a production persistence implementation.
type MemoryRepository struct {
	mu      sync.RWMutex
	plugins map[domain.PluginID]domain.Plugin
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{plugins: make(map[domain.PluginID]domain.Plugin)}
}

func (r *MemoryRepository) Create(ctx context.Context, plugin domain.Plugin) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := plugin.Validate(); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidPlugin, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, exists := r.plugins[plugin.ID]; exists {
		return fmt.Errorf("%w: %s", ErrAlreadyExists, plugin.ID)
	}
	r.plugins[plugin.ID] = plugin.Clone()
	return nil
}

func (r *MemoryRepository) Get(ctx context.Context, id domain.PluginID) (domain.Plugin, error) {
	if err := ctx.Err(); err != nil {
		return domain.Plugin{}, err
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return domain.Plugin{}, err
	}
	plugin, exists := r.plugins[id]
	if !exists {
		return domain.Plugin{}, fmt.Errorf("%w: %s", ErrNotFound, id)
	}
	return plugin.Clone(), nil
}

func (r *MemoryRepository) List(ctx context.Context) ([]domain.Plugin, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	plugins := make([]domain.Plugin, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		plugins = append(plugins, plugin.Clone())
	}
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})
	return plugins, nil
}
