package execution

import (
	"context"
	"errors"
	"testing"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	selectioncontract "pride-ai-module/contracts/selection"
	"pride-ai-module/implementation"
	"pride-ai-module/selection"
)

type integrationImplementation struct {
	called bool
}

func (i *integrationImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if ctx == nil {
		return executioncontract.AIExecutionResult{}, errors.New(
			"context must not be nil",
		)
	}

	i.called = true

	if executionContext.Configuration.Ref != "config_integration" {
		return executioncontract.AIExecutionResult{}, errors.New(
			"unexpected configuration reference",
		)
	}

	if executionContext.Configuration.ImplementationRef !=
		"test.integration" {
		return executioncontract.AIExecutionResult{}, errors.New(
			"unexpected implementation reference",
		)
	}

	if string(executionContext.Credential) !=
		"integration-secret" {
		return executioncontract.AIExecutionResult{}, errors.New(
			"unexpected credential",
		)
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
		Metadata: map[string]any{
			"source": "integration-test",
		},
	}, nil
}

type integrationCredentialResolver struct {
	ref   string
	value []byte
}

func (r *integrationCredentialResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if ref != r.ref {
		return nil, errors.New(
			"unexpected credential reference",
		)
	}

	return r.value, nil
}

func newIntegrationSelection(
	t *testing.T,
	ctx context.Context,
	ref string,
) selectioncontract.ConfigurationSelector {
	t.Helper()

	store := selection.NewMemoryStore()
	service := selection.NewService(store)

	if err := service.Select(ctx, ref); err != nil {
		t.Fatalf(
			"Select() failed: %v",
			err,
		)
	}

	return service
}

func TestExecutorIntegration(
	t *testing.T,
) {
	ctx := context.Background()

	configuration := configuration.AIConfiguration{
		Ref:               "config_integration",
		Name:              "integration-test",
		Type:              "test",
		ImplementationRef: "test.integration",
		CredentialRef:     "cred_integration",
		Enabled:           true,
		ConnectionParameters: map[string]any{
			"model": "integration-model",
		},
	}

	configProvider := &integrationConfigurationProvider{
		configuration: configuration,
	}

	impl := &integrationImplementation{}

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		configuration.ImplementationRef,
		impl,
	); err != nil {
		t.Fatalf(
			"Register() failed: %v",
			err,
		)
	}

	credentialResolver := &integrationCredentialResolver{
		ref:   "cred_integration",
		value: []byte("integration-secret"),
	}

	selector := newIntegrationSelection(
		t,
		ctx,
		configuration.Ref,
	)

	executor := NewExecutor(
		configProvider,
		selector,
		implementationResolver,
		credentialResolver,
	)

	request := executioncontract.AIExecutionRequest{
		RequestID:  "req_integration",
		Capability: "test.generate",
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

	if !impl.called {
		t.Fatal(
			"implementation was not executed",
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

	if result.Metadata["source"] != "integration-test" {
		t.Fatalf(
			"unexpected result metadata: got %v",
			result.Metadata,
		)
	}
}

type integrationConfigurationProvider struct {
	configuration configuration.AIConfiguration
	err           error
}

func (p *integrationConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if p.err != nil {
		return configuration.AIConfiguration{}, p.err
	}

	return p.configuration, nil
}

func (p *integrationConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if p.err != nil {
		return configuration.AIConfiguration{}, p.err
	}

	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{},
			configuration.ErrConfigurationRefNotFound
	}

	return p.configuration, nil
}

var _ configuration.AIConfigurationProvider = (*integrationConfigurationProvider)(nil)
