package selection

import (
	"context"
	"errors"
	"sync"

	selectioncontract "pride-ai-module/contracts/selection"
)

var (
	ErrConfigurationNotSelected = errors.New(
		"configuration has not been selected",
	)

	ErrConfigurationRefEmpty = errors.New(
		"configuration reference must not be empty",
	)
)

// MemoryStore stores the currently selected configuration
// in memory.
type MemoryStore struct {
	mu sync.RWMutex

	configurationRef string
}

// NewMemoryStore creates an empty configuration selection store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{}
}

// Get returns the currently selected configuration.
func (s *MemoryStore) Get(
	ctx context.Context,
) (selectioncontract.ConfigurationSelection, error) {
	if err := ctx.Err(); err != nil {
		return selectioncontract.ConfigurationSelection{}, err
	}

	if s == nil {
		return selectioncontract.ConfigurationSelection{}, errors.New(
			"selection store must not be nil",
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.configurationRef == "" {
		return selectioncontract.ConfigurationSelection{},
			ErrConfigurationNotSelected
	}

	return selectioncontract.ConfigurationSelection{
		ConfigurationRef: s.configurationRef,
	}, nil
}

// Set stores the selected configuration.
func (s *MemoryStore) Set(
	ctx context.Context,
	selection selectioncontract.ConfigurationSelection,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil {
		return errors.New(
			"selection store must not be nil",
		)
	}

	if selection.ConfigurationRef == "" {
		return ErrConfigurationRefEmpty
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.configurationRef = selection.ConfigurationRef

	return nil
}

// Compile-time contract verification.
var _ selectioncontract.SelectionStore = (*MemoryStore)(nil)
