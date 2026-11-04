package registry

import (
	"context"

	"pride-plugin-module/domain"
)

// Repository is the narrow persistence boundary for the local installed
// Plugin Registry. It does not represent the Marketplace's official Registry.
type Repository interface {
	Create(context.Context, domain.Plugin) error
	Get(context.Context, domain.PluginID) (domain.Plugin, error)
	List(context.Context) ([]domain.Plugin, error)
}
