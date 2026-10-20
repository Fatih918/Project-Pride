package configuration

import "context"

type Store interface {
	GetActive(
		ctx context.Context,
	) (AIConfiguration, error)
}

type ConfigurationStore interface {
	Create(
		ctx context.Context,
		configuration AIConfiguration,
	) error

	GetByRef(
		ctx context.Context,
		configurationRef string,
	) (AIConfiguration, error)

	Delete(
		ctx context.Context,
		configurationRef string,
	) error

	Select(
		ctx context.Context,
		configurationRef string,
	) error
}
