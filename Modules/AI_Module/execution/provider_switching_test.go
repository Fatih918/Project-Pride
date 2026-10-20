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

//
// ------------------------------------------------------------
// Configuration Provider
// ------------------------------------------------------------
//

type switchingConfigurationProvider struct {
	configurations map[string]configuration.AIConfiguration
}

func (p *switchingConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	return configuration.AIConfiguration{}, errors.New(
		"GetActive must not be used by switching test",
	)
}

func (p *switchingConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return configuration.AIConfiguration{}, err
	}

	config, exists := p.configurations[ref]
	if !exists {
		return configuration.AIConfiguration{},
			configuration.ErrConfigurationRefNotFound
	}

	return config, nil
}

//
// ------------------------------------------------------------
// Provider A / B
// ------------------------------------------------------------
//

type switchingProvider struct {
	name   string
	called int
}

func (p *switchingProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	p.called++

	return provider.Response{
		Output: request.Input,
		Metadata: map[string]any{
			"provider": p.name,
		},
	}, nil
}

var _ provider.Provider = (*switchingProvider)(nil)

//
// ------------------------------------------------------------
// Implementation using Provider Resolver
// ------------------------------------------------------------
//
// IMPORTANT:
//
// Implementation tidak memegang provider secara langsung.
//
// Implementation hanya mengetahui provider resolver.
//
// Provider yang digunakan ditentukan oleh:
//
// executionContext.Configuration.ImplementationRef
//
// Dengan demikian switching terjadi melalui configuration +
// provider resolver.
//

type switchingResolverImplementation struct {
	resolver *provider.Resolver
}

func (i *switchingResolverImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if i == nil {
		return executioncontract.AIExecutionResult{},
			errors.New("implementation must not be nil")
	}

	if i.resolver == nil {
		return executioncontract.AIExecutionResult{},
			errors.New("provider resolver must not be nil")
	}

	implementationRef :=
		executionContext.Configuration.ImplementationRef

	if implementationRef == "" {
		return executioncontract.AIExecutionResult{},
			errors.New("implementation reference must not be empty")
	}

	resolvedProvider, err := i.resolver.Resolve(
		implementationRef,
	)
	if err != nil {
		return executioncontract.AIExecutionResult{},
			err
	}

	response, err := resolvedProvider.Execute(
		ctx,
		provider.Request{
			Capability: request.Capability,
			Input:      request.Input,
			Metadata:   request.Metadata,
		},
	)
	if err != nil {
		return executioncontract.AIExecutionResult{},
			err
	}

	output, ok := response.Output.(*anypb.Any)
	if !ok {
		return executioncontract.AIExecutionResult{},
			errors.New("provider output must be *anypb.Any")
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata:   response.Metadata,
	}, nil
}

var _ implementation.AIImplementation = (*switchingResolverImplementation)(nil)

//
// ------------------------------------------------------------
// Credential Resolver
// ------------------------------------------------------------
//

type switchingCredentialResolver struct{}

func (r *switchingCredentialResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if ref == "" {
		return nil, errors.New(
			"credential reference must not be empty",
		)
	}

	return []byte("test-credential"), nil
}

var _ executioncontract.CredentialResolver = (*switchingCredentialResolver)(nil)

//
// ------------------------------------------------------------
// Provider Switching Test
// ------------------------------------------------------------
//

func TestExecutorProviderSwitching(t *testing.T) {
	ctx := context.Background()

	// --------------------------------------------------------
	// Create Provider A
	// --------------------------------------------------------

	providerA := &switchingProvider{
		name: "provider-a",
	}

	// --------------------------------------------------------
	// Create Provider B
	// --------------------------------------------------------

	providerB := &switchingProvider{
		name: "provider-b",
	}

	// --------------------------------------------------------
	// Create Provider Resolver
	// --------------------------------------------------------

	providerResolver := provider.NewResolver()

	if err := providerResolver.Register(
		"implementation.provider.a",
		providerA,
	); err != nil {
		t.Fatalf(
			"register provider A: %v",
			err,
		)
	}

	if err := providerResolver.Register(
		"implementation.provider.b",
		providerB,
	); err != nil {
		t.Fatalf(
			"register provider B: %v",
			err,
		)
	}

	// --------------------------------------------------------
	// Create ONE Implementation
	// --------------------------------------------------------
	//
	// There are deliberately NOT two implementations.
	//
	// The same implementation will resolve different providers
	// based on Configuration.ImplementationRef.
	//

	resolverImplementation :=
		&switchingResolverImplementation{
			resolver: providerResolver,
		}

	// --------------------------------------------------------
	// Configuration A
	// --------------------------------------------------------

	configA := configuration.AIConfiguration{
		Ref:               "config_provider_a",
		Name:              "Provider A",
		Type:              "test",
		ImplementationRef: "implementation.provider.a",
		CredentialRef:     "credential.provider.a",
		Enabled:           true,
	}

	// --------------------------------------------------------
	// Configuration B
	// --------------------------------------------------------

	configB := configuration.AIConfiguration{
		Ref:               "config_provider_b",
		Name:              "Provider B",
		Type:              "test",
		ImplementationRef: "implementation.provider.b",
		CredentialRef:     "credential.provider.b",
		Enabled:           true,
	}

	// --------------------------------------------------------
	// Configuration Provider
	// --------------------------------------------------------

	configProvider := &switchingConfigurationProvider{
		configurations: map[string]configuration.AIConfiguration{
			configA.Ref: configA,
			configB.Ref: configB,
		},
	}

	// --------------------------------------------------------
	// Selection
	// --------------------------------------------------------

	selectionStore := selection.NewMemoryStore()

	selectionService := selection.NewService(
		selectionStore,
	)

	// --------------------------------------------------------
	// Implementation Resolver
	// --------------------------------------------------------
	//
	// Both configurations resolve to the SAME implementation.
	//
	// Provider selection happens inside that implementation
	// through provider.Resolver.
	//

	implementationResolver := implementation.NewResolver()

	if err := implementationResolver.Register(
		configA.ImplementationRef,
		resolverImplementation,
	); err != nil {
		t.Fatalf(
			"register implementation A: %v",
			err,
		)
	}

	if err := implementationResolver.Register(
		configB.ImplementationRef,
		resolverImplementation,
	); err != nil {
		t.Fatalf(
			"register implementation B: %v",
			err,
		)
	}
	// --------------------------------------------------------
	// Executor
	// --------------------------------------------------------

	executor := NewExecutor(
		configProvider,
		selectionService,
		implementationResolver,
		&switchingCredentialResolver{},
	)

	// --------------------------------------------------------
	// Input
	// --------------------------------------------------------

	input, err := anypb.New(
		&emptypb.Empty{},
	)
	if err != nil {
		t.Fatalf(
			"create input Any: %v",
			err,
		)
	}

	// ========================================================
	// PROVIDER A
	// ========================================================

	t.Run("provider A", func(t *testing.T) {
		// ----------------------------------------------------
		// Select configuration A
		// ----------------------------------------------------

		if err := selectionService.Select(
			ctx,
			configA.Ref,
		); err != nil {
			t.Fatalf(
				"select provider A: %v",
				err,
			)
		}

		// ----------------------------------------------------
		// Verify selection
		// ----------------------------------------------------

		selected, err := selectionService.GetSelected(ctx)
		if err != nil {
			t.Fatalf(
				"GetSelected() failed: %v",
				err,
			)
		}

		if selected.ConfigurationRef != configA.Ref {
			t.Fatalf(
				"unexpected selected configuration: got %q, want %q",
				selected.ConfigurationRef,
				configA.Ref,
			)
		}

		// ----------------------------------------------------
		// Execute
		// ----------------------------------------------------

		result, err := executor.Execute(
			ctx,
			executioncontract.AIExecutionRequest{
				RequestID:  "req_provider_a",
				Capability: "test.generate",
				Input:      input,
			},
		)
		if err != nil {
			t.Fatalf(
				"execute with provider A: %v",
				err,
			)
		}

		// ----------------------------------------------------
		// Verify provider result
		// ----------------------------------------------------

		if result.Metadata["provider"] != "provider-a" {
			t.Fatalf(
				"unexpected provider result: got %v, want provider-a",
				result.Metadata["provider"],
			)
		}

		// ----------------------------------------------------
		// Verify call counts
		// ----------------------------------------------------

		if providerA.called != 1 {
			t.Fatalf(
				"provider A call count: got %d, want 1",
				providerA.called,
			)
		}

		if providerB.called != 0 {
			t.Fatalf(
				"provider B must not be called before switching: got %d, want 0",
				providerB.called,
			)
		}
	})

	// ========================================================
	// SWITCH TO PROVIDER B
	// ========================================================

	t.Run("provider B after switching", func(t *testing.T) {
		// ----------------------------------------------------
		// Switch configuration
		// ----------------------------------------------------

		if err := selectionService.Select(
			ctx,
			configB.Ref,
		); err != nil {
			t.Fatalf(
				"select provider B: %v",
				err,
			)
		}

		// ----------------------------------------------------
		// Verify selection changed
		// ----------------------------------------------------

		selected, err := selectionService.GetSelected(ctx)
		if err != nil {
			t.Fatalf(
				"GetSelected() failed: %v",
				err,
			)
		}

		if selected.ConfigurationRef != configB.Ref {
			t.Fatalf(
				"unexpected selected configuration: got %q, want %q",
				selected.ConfigurationRef,
				configB.Ref,
			)
		}

		// ----------------------------------------------------
		// Execute again
		// ----------------------------------------------------

		result, err := executor.Execute(
			ctx,
			executioncontract.AIExecutionRequest{
				RequestID:  "req_provider_b",
				Capability: "test.generate",
				Input:      input,
			},
		)
		if err != nil {
			t.Fatalf(
				"execute with provider B: %v",
				err,
			)
		}

		// ----------------------------------------------------
		// Verify provider result
		// ----------------------------------------------------

		if result.Metadata["provider"] != "provider-b" {
			t.Fatalf(
				"unexpected provider result: got %v, want provider-b",
				result.Metadata["provider"],
			)
		}

		// ----------------------------------------------------
		// Provider A must NOT receive another request
		// ----------------------------------------------------

		if providerA.called != 1 {
			t.Fatalf(
				"provider A must not receive request after switching: got %d calls, want 1",
				providerA.called,
			)
		}

		// ----------------------------------------------------
		// Provider B must receive exactly one request
		// ----------------------------------------------------

		if providerB.called != 1 {
			t.Fatalf(
				"provider B call count: got %d, want 1",
				providerB.called,
			)
		}
	})

	// ========================================================
	// FINAL STATE
	// ========================================================

	t.Run("final selection state", func(t *testing.T) {
		selected, err := selectionService.GetSelected(ctx)
		if err != nil {
			t.Fatalf(
				"GetSelected() failed: %v",
				err,
			)
		}

		if selected.ConfigurationRef != configB.Ref {
			t.Fatalf(
				"unexpected final selected configuration: got %q, want %q",
				selected.ConfigurationRef,
				configB.Ref,
			)
		}
	})
}
