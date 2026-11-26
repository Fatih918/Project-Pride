package startup

import (
	"context"
	"fmt"

	"pride-plugin-module/discovery"
	"pride-plugin-module/registry"
)

// LifecycleRecoverer classifies interrupted lifecycle operations before the
// host exposes lifecycle mutation entry points.
type LifecycleRecoverer interface {
	Recover(context.Context) error
}

// InitializeWithLifecycle performs both installation and lifecycle recovery
// before returning discovery. The host must construct and expose lifecycle
// mutation services only after this function succeeds, and must retain the
// cross-process startup lock for the full subsystem lifetime.
func InitializeWithLifecycle(
	ctx context.Context,
	installationRecoverer Recoverer,
	lifecycleRecoverer LifecycleRecoverer,
	repository registry.Repository,
) (*discovery.Service, error) {
	if installationRecoverer == nil || lifecycleRecoverer == nil || repository == nil {
		return nil, ErrMissingDependency
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := installationRecoverer.Recover(ctx); err != nil {
		return nil, fmt.Errorf("recover Plugin installations before enabling subsystem: %w", err)
	}
	if err := lifecycleRecoverer.Recover(ctx); err != nil {
		return nil, fmt.Errorf("recover Plugin lifecycle operations before enabling subsystem: %w", err)
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
