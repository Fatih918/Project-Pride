package configuration

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"pride-ai-module/credential"
)

type ConfigurationService struct {
	store             ConfigurationStore
	credentialService *credential.Service
}

func NewConfigurationService(
	store ConfigurationStore,
	credentialService *credential.Service,
) *ConfigurationService {
	return &ConfigurationService{
		store:             store,
		credentialService: credentialService,
	}
}

type CreateConfigurationRequest struct {
	Name string
	Type string

	ImplementationRef string

	ConnectionParameters map[string]any

	CredentialRef string

	Secret []byte
}

func (s *ConfigurationService) Create(
	ctx context.Context,
	request CreateConfigurationRequest,
) (AIConfiguration, error) {
	if s == nil {
		return AIConfiguration{}, fmt.Errorf(
			"configuration service must not be nil",
		)
	}

	if s.store == nil {
		return AIConfiguration{}, fmt.Errorf(
			"configuration store must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if request.Name == "" {
		return AIConfiguration{}, fmt.Errorf(
			"configuration name must not be empty",
		)
	}

	if request.Type == "" {
		return AIConfiguration{}, fmt.Errorf(
			"configuration type must not be empty",
		)
	}

	if request.ImplementationRef == "" {
		return AIConfiguration{}, fmt.Errorf(
			"configuration implementation reference must not be empty",
		)
	}

	if request.CredentialRef != "" &&
		len(request.Secret) > 0 {
		return AIConfiguration{}, fmt.Errorf(
			"credential reference and secret must not both be provided",
		)
	}

	credentialRef := request.CredentialRef

	if len(request.Secret) > 0 {
		if s.credentialService == nil {
			return AIConfiguration{}, fmt.Errorf(
				"credential service must not be nil",
			)
		}

		ref, err := s.credentialService.Create(
			ctx,
			request.Secret,
		)
		if err != nil {
			return AIConfiguration{}, fmt.Errorf(
				"create credential: %w",
				err,
			)
		}

		credentialRef = ref
	}

	configuration := AIConfiguration{
		Ref: uuid.NewString(),

		Name: request.Name,

		Type: request.Type,

		ImplementationRef: request.ImplementationRef,

		ConnectionParameters: cloneMap(
			request.ConnectionParameters,
		),

		CredentialRef: credentialRef,

		Enabled: false,
	}

	if err := configuration.Validate(); err != nil {
		return AIConfiguration{}, err
	}

	if err := s.store.Create(
		ctx,
		configuration,
	); err != nil {
		if request.CredentialRef == "" &&
			credentialRef != "" &&
			s.credentialService != nil {
			_ = s.credentialService.Delete(
				context.Background(),
				credentialRef,
			)
		}

		return AIConfiguration{}, fmt.Errorf(
			"create AI configuration: %w",
			err,
		)
	}

	return configuration, nil
}

func (s *ConfigurationService) Select(
	ctx context.Context,
	configurationRef string,
) (AIConfiguration, error) {
	if s == nil {
		return AIConfiguration{}, fmt.Errorf(
			"configuration service must not be nil",
		)
	}

	if s.store == nil {
		return AIConfiguration{}, fmt.Errorf(
			"configuration store must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if configurationRef == "" {
		return AIConfiguration{}, ErrConfigurationRefNotFound
	}

	if err := s.store.Select(
		ctx,
		configurationRef,
	); err != nil {
		return AIConfiguration{}, fmt.Errorf(
			"select AI configuration: %w",
			err,
		)
	}

	configuration, err := s.store.GetByRef(
		ctx,
		configurationRef,
	)
	if err != nil {
		return AIConfiguration{}, fmt.Errorf(
			"get selected AI configuration: %w",
			err,
		)
	}

	return configuration, nil
}

func cloneMap(
	value map[string]any,
) map[string]any {
	if value == nil {
		return nil
	}

	result := make(
		map[string]any,
		len(value),
	)

	for key, item := range value {
		result[key] = item
	}

	return result
}
