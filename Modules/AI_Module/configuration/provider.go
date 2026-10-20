package configuration

import "context"

type AIConfigurationProvider interface {
	GetActive(
		ctx context.Context,
	) (AIConfiguration, error)

	GetByRef(
		ctx context.Context,
		configurationRef string,
	) (AIConfiguration, error)
}
