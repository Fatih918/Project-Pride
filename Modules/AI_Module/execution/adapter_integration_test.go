package execution

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/provider"
	"pride-ai-module/selection"
)

type adapterIntegrationProvider struct {
	called bool
}

func (p *adapterIntegrationProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	p.called = true

	return provider.Response{
		Output: request.Input,
		Metadata: map[string]any{
			"provider": "adapter-integration-test",
		},
	}, nil
}

var _ provider.Provider = (*adapterIntegrationProvider)(nil)

type adapterIntegrationImplementation struct {
	provider provider.Provider
}

func (i *adapterIntegrationImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if i == nil || i.provider == nil {
		return executioncontract.AIExecutionResult{}, errors.New(
			"provider must not be nil",
		)
	}

	response, err := i.provider.Execute(
		ctx,
		provider.Request{
			Capability: request.Capability,
			Input:      request.Input,
			Metadata:   request.Metadata,
		},
	)
	if err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	output, ok := response.Output.(*anypb.Any)
	if !ok {
		return executioncontract.AIExecutionResult{}, errors.New(
			"provider returned unexpected output type",
		)
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata:   response.Metadata,
	}, nil
}

type adapterIntegrationCredentialResolver struct{}

func (r *adapterIntegrationCredentialResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if ref != "cred_integration" {
		return nil, errors.New(
			"unexpected credential reference",
		)
	}

	return []byte("integration-secret"), nil
}

type adapterIntegrationConfigurationProvider struct {
	configuration configuration.AIConfiguration
}

func (p *adapterIntegrationConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	return p.configuration, nil
}

func (p *adapterIntegrationConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{},
			configuration.ErrConfigurationRefNotFound
	}

	return p.configuration, nil
}

func TestExecutorAdapterIntegration(t *testing.T) {
	ctx := context.Background()

	configurationValue := configuration.AIConfiguration{
		Ref:               "config_adapter_integration",
		Name:              "adapter-integration",
		Type:              "test",
		ImplementationRef: "test.adapter.integration",
		CredentialRef:     "cred_integration",
		Enabled:           true,
	}

	configProvider := &adapterIntegrationConfigurationProvider{
		configuration: configurationValue,
	}

	selectorStore := selection.NewMemoryStore()
	selectorService := selection.NewService(selectorStore)

	if err := selectorService.Select(
		ctx,
		configurationValue.Ref,
	); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	testProvider := &adapterIntegrationProvider{}

	testImplementation := &adapterIntegrationImplementation{
		provider: testProvider,
	}

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		configurationValue.ImplementationRef,
		testImplementation,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	executor := NewExecutor(
		configProvider,
		selectorService,
		implementationResolver,
		&adapterIntegrationCredentialResolver{},
	)

	input, err := anypb.New(
		&emptypb.Empty{},
	)
	if err != nil {
		t.Fatalf(
			"create input Any: %v",
			err,
		)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_adapter_integration",
		Capability: "test.generate",
		Input:      input,
	}

	result, err := executor.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if !testProvider.called {
		t.Fatal(
			"provider adapter was not called",
		)
	}

	if result.RequestID != request.RequestID {
		t.Fatalf(
			"unexpected request ID: got %q, want %q",
			result.RequestID,
			request.RequestID,
		)
	}

	if result.Capability != request.Capability {
		t.Fatalf(
			"unexpected capability: got %q, want %q",
			result.Capability,
			request.Capability,
		)
	}

	if result.Output != input {
		t.Fatal(
			"provider output was not propagated",
		)
	}

	if result.Metadata["provider"] !=
		"adapter-integration-test" {
		t.Fatalf(
			"unexpected provider metadata: got %v",
			result.Metadata,
		)
	}
}
