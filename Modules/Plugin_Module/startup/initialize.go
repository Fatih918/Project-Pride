// Package startup provides the fail-closed entry point for exposing Plugin
// discovery after interrupted installation recovery has completed.
package startup

import (
	"context"
	"errors"
	"fmt"

	"pride-plugin-module/discovery"
	"pride-plugin-module/registry"
)

var ErrMissingDependency = errors.New("plugin startup dependency is required")

// Recoverer reconciles incomplete installation operations. Implementations
// must return an error when any operation cannot be safely reconciled.
type Recoverer interface {
	Recover(context.Context) error
}

// Initialize runs installation recovery before constructing the discovery
// service. On recovery failure it returns a nil service, so callers cannot
// accidentally expose discovery through this startup path.
//
// The caller owns repository/storage lifetimes and must close them on error.
// The caller must also serialize competing process startups until a
// cross-process recovery lease is provided by the host application.
func Initialize(ctx context.Context, recoverer Recoverer, repository registry.Repository) (*discovery.Service, error) {
	if recoverer == nil || repository == nil {
		return nil, ErrMissingDependency
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := recoverer.Recover(ctx); err != nil {
		return nil, fmt.Errorf("recover Plugin installations before enabling discovery: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	service, err := discovery.NewService(repository)
	if err != nil {
		return nil, fmt.Errorf("initialize Plugin discovery after recovery: %w", err)
	}
	return service, nil
}
