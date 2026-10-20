package provider

import (
	"fmt"
	"sync"
)

type Resolver struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewResolver() *Resolver {
	return &Resolver{
		providers: make(map[string]Provider),
	}
}

func (r *Resolver) Register(ref string, provider Provider) error {
	if r == nil {
		return fmt.Errorf("provider resolver must not be nil")
	}

	if ref == "" {
		return fmt.Errorf("provider reference must not be empty")
	}

	if provider == nil {
		return fmt.Errorf("provider must not be nil")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[ref]; exists {
		return fmt.Errorf("provider already registered: %s", ref)
	}

	r.providers[ref] = provider

	return nil
}

func (r *Resolver) Resolve(ref string) (Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("provider resolver must not be nil")
	}

	if ref == "" {
		return nil, fmt.Errorf("provider reference must not be empty")
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	provider, exists := r.providers[ref]
	if !exists {
		return nil, fmt.Errorf("provider not found: %s", ref)
	}

	return provider, nil
}
