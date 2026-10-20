package implementation

import (
	"fmt"
	"sync"
)

type Resolver struct {
	mu              sync.RWMutex
	implementations map[string]AIImplementation
}

func NewResolver() *Resolver {
	return &Resolver{
		implementations: make(
			map[string]AIImplementation,
		),
	}
}

func (r *Resolver) Register(
	reference string,
	implementation AIImplementation,
) error {
	if r == nil {
		return fmt.Errorf(
			"AI implementation resolver must not be nil",
		)
	}

	if reference == "" {
		return fmt.Errorf(
			"AI implementation reference must not be empty",
		)
	}

	if implementation == nil {
		return fmt.Errorf(
			"AI implementation must not be nil",
		)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.implementations[reference]; exists {
		return fmt.Errorf(
			"AI implementation already registered: %s",
			reference,
		)
	}

	r.implementations[reference] = implementation

	return nil
}

func (r *Resolver) Resolve(
	reference string,
) (AIImplementation, error) {
	if r == nil {
		return nil, fmt.Errorf(
			"AI implementation resolver must not be nil",
		)
	}

	if reference == "" {
		return nil, fmt.Errorf(
			"AI implementation reference must not be empty",
		)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	implementation, ok := r.implementations[reference]
	if !ok {
		return nil, fmt.Errorf(
			"AI implementation not found: %s",
			reference,
		)
	}

	if implementation == nil {
		return nil, fmt.Errorf(
			"AI implementation is nil: %s",
			reference,
		)
	}

	return implementation, nil
}
