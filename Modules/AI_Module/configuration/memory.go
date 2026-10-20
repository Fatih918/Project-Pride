package configuration

import (
	"context"
	"errors"
	"sync"
)

var (
	ErrConfigurationNotFound = errors.New(
		"active AI configuration not found",
	)

	ErrConfigurationRefNotFound = errors.New(
		"AI configuration reference not found",
	)

	ErrConfigurationAlreadyExists = errors.New(
		"AI configuration already exists",
	)
)

type MemoryStore struct {
	mu             sync.RWMutex
	active         *AIConfiguration
	configurations map[string]AIConfiguration
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		configurations: make(map[string]AIConfiguration),
	}
}

func (s *MemoryStore) Create(
	ctx context.Context,
	configuration AIConfiguration,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil {
		return errors.New("configuration store must not be nil")
	}

	if err := configuration.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.configurations[configuration.Ref]; exists {
		return ErrConfigurationAlreadyExists
	}

	cloned := cloneConfiguration(configuration)
	s.configurations[configuration.Ref] = cloned

	return nil
}

func (s *MemoryStore) GetByRef(
	ctx context.Context,
	configurationRef string,
) (AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if s == nil {
		return AIConfiguration{}, errors.New(
			"configuration store must not be nil",
		)
	}

	if configurationRef == "" {
		return AIConfiguration{}, ErrConfigurationRefNotFound
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	configuration, ok := s.configurations[configurationRef]
	if !ok {
		return AIConfiguration{}, ErrConfigurationRefNotFound
	}

	return cloneConfiguration(configuration), nil
}

func (s *MemoryStore) Delete(
	ctx context.Context,
	configurationRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil {
		return errors.New("configuration store must not be nil")
	}

	if configurationRef == "" {
		return ErrConfigurationRefNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.configurations[configurationRef]; !ok {
		return ErrConfigurationRefNotFound
	}

	delete(s.configurations, configurationRef)

	if s.active != nil &&
		s.active.Ref == configurationRef {
		s.active = nil
	}

	return nil
}

func (s *MemoryStore) GetActive(
	ctx context.Context,
) (AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if s == nil {
		return AIConfiguration{}, errors.New(
			"configuration store must not be nil",
		)
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.active == nil {
		return AIConfiguration{}, ErrConfigurationNotFound
	}

	return cloneConfiguration(*s.active), nil
}

func cloneConfiguration(
	configuration AIConfiguration,
) AIConfiguration {
	result := configuration

	if configuration.ConnectionParameters != nil {
		result.ConnectionParameters = make(
			map[string]any,
			len(configuration.ConnectionParameters),
		)

		for key, value := range configuration.ConnectionParameters {
			result.ConnectionParameters[key] = value
		}
	}

	return result
}

func (s *MemoryStore) Select(
	ctx context.Context,
	configurationRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil {
		return errors.New(
			"configuration store must not be nil",
		)
	}

	if configurationRef == "" {
		return ErrConfigurationRefNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	selected, ok := s.configurations[configurationRef]
	if !ok {
		return ErrConfigurationRefNotFound
	}

	// Only one configuration may be selected,
	// active, and enabled at a time.
	for ref, configuration := range s.configurations {
		configuration.Enabled = false
		s.configurations[ref] = configuration
	}

	selected.Enabled = true
	s.configurations[configurationRef] = selected

	active := selected
	s.active = &active

	return nil
}

var _ Store = (*MemoryStore)(nil)

var _ ConfigurationStore = (*MemoryStore)(nil)

var _ AIConfigurationProvider = (*MemoryStore)(nil)
