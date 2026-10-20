package selection

import "context"

// ConfigurationSelection identifies the configuration currently
// selected for execution.
//
// ConfigurationRef refers to an existing AI configuration.
// It does not identify a provider directly.
type ConfigurationSelection struct {
	ConfigurationRef string
}

// SelectionStore defines persistence for configuration selection.
type SelectionStore interface {
	Get(
		ctx context.Context,
	) (ConfigurationSelection, error)

	Set(
		ctx context.Context,
		selection ConfigurationSelection,
	) error
}

// ConfigurationSelector defines the application-level boundary
// for reading and changing the selected configuration.
type ConfigurationSelector interface {
	GetSelected(
		ctx context.Context,
	) (ConfigurationSelection, error)

	Select(
		ctx context.Context,
		configurationRef string,
	) error
}
