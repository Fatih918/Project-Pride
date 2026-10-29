package execution

import (
	"context"
	"fmt"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	selectioncontract "pride-ai-module/contracts/selection"
	"pride-ai-module/implementation"
)

type Executor struct {
	configurations     configuration.AIConfigurationProvider
	selection          selectioncontract.ConfigurationSelector
	resolver           *implementation.Resolver
	credentialResolver executioncontract.CredentialResolver
}

func NewExecutor(
	configurations configuration.AIConfigurationProvider,
	selection selectioncontract.ConfigurationSelector,
	resolver *implementation.Resolver,
	credentialResolver executioncontract.CredentialResolver,
) *Executor {
	return &Executor{
		configurations:     configurations,
		selection:          selection,
		resolver:           resolver,
		credentialResolver: credentialResolver,
	}
}

func (e *Executor) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
) (executioncontract.AIExecutionResult, error) {
	if e == nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"executor must not be nil",
		)
	}

	if ctx == nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"context must not be nil",
		)
	}

	if e.configurations == nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"configuration provider must not be nil",
		)
	}

	if e.selection == nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"configuration selector must not be nil",
		)
	}

	if e.resolver == nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"implementation resolver must not be nil",
		)
	}

	selected, err := e.selection.GetSelected(ctx)
	if err != nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"get selected configuration: %w",
			err,
		)
	}

	if selected.ConfigurationRef == "" {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"selected configuration reference must not be empty",
		)
	}

	config, err := e.configurations.GetByRef(
		ctx,
		selected.ConfigurationRef,
	)
	if err != nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"get configuration %q: %w",
			selected.ConfigurationRef,
			err,
		)
	}

	if !config.Enabled {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"configuration %q is disabled",
			config.Ref,
		)
	}

	aiImplementation, err := e.resolver.Resolve(
		config.ImplementationRef,
	)
	if err != nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"resolve implementation %q: %w",
			config.ImplementationRef,
			err,
		)
	}

	// Credential is optional at the configuration level.
	//
	// CredentialResolver is invoked only when the selected
	// configuration explicitly references a credential.
	var credentialValue []byte

	if config.CredentialRef != "" {
		if e.credentialResolver == nil {
			return executioncontract.AIExecutionResult{}, fmt.Errorf(
				"credential resolver must not be nil",
			)
		}

		credentialValue, err = e.credentialResolver.Resolve(
			ctx,
			config.CredentialRef,
		)
		if err != nil {
			return executioncontract.AIExecutionResult{}, fmt.Errorf(
				"resolve credential: %w",
				err,
			)
		}
	}

	executionContext := executioncontract.AIExecutionContext{
		Configuration: config,
		Credential:    credentialValue,
	}

	result, err := aiImplementation.Execute(
		ctx,
		request,
		executionContext,
	)
	if err != nil {
		return executioncontract.AIExecutionResult{}, fmt.Errorf(
			"execute AI implementation: %w",
			err,
		)
	}

	return result, nil
}
