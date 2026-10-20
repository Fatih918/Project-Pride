package gemini

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	"pride-ai-module/contracts/execution"
	"pride-ai-module/provider"
)

type ProviderFactory func(
	ctx context.Context,
	executionContext execution.AIExecutionContext,
) (provider.Provider, error)

type Adapter struct {
	provider        provider.Provider
	providerFactory ProviderFactory
}

func NewAdapter(p provider.Provider) (*Adapter, error) {
	if p == nil {
		return nil, fmt.Errorf(
			"Gemini adapter provider must not be nil",
		)
	}

	return &Adapter{
		provider: p,
	}, nil
}

func NewAdapterWithFactory(factory ProviderFactory) (*Adapter, error) {
	if factory == nil {
		return nil, fmt.Errorf(
			"Gemini provider factory must not be nil",
		)
	}

	return &Adapter{
		providerFactory: factory,
	}, nil
}

func (a *Adapter) Execute(
	ctx context.Context,
	request execution.AIExecutionRequest,
	executionContext execution.AIExecutionContext,
) (
	execution.AIExecutionResult,
	error,
) {
	if a == nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini adapter must not be nil",
		)
	}

	if ctx == nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini adapter context must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return execution.AIExecutionResult{}, err
	}

	p, err := a.resolveProvider(ctx, executionContext)
	if err != nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini provider resolution failed: %w",
			err,
		)
	}

	providerRequest := provider.Request{
		Capability: request.Capability,
		Input:      request.Input,
		Metadata:   request.Metadata,
	}

	providerResponse, err := p.Execute(
		ctx,
		providerRequest,
	)
	if err != nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini provider execution failed: %w",
			err,
		)
	}

	output, err := normalizeOutput(providerResponse.Output)
	if err != nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini adapter output normalization failed: %w",
			err,
		)
	}

	return execution.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata:   providerResponse.Metadata,
	}, nil
}

func (a *Adapter) resolveProvider(
	ctx context.Context,
	executionContext execution.AIExecutionContext,
) (provider.Provider, error) {
	if a.providerFactory != nil {
		p, err := a.providerFactory(ctx, executionContext)
		if err != nil {
			return nil, err
		}

		if p == nil {
			return nil, fmt.Errorf(
				"Gemini provider factory returned nil provider",
			)
		}

		return p, nil
	}

	if a.provider == nil {
		return nil, fmt.Errorf(
			"Gemini adapter provider must not be nil",
		)
	}

	return a.provider, nil
}

func normalizeOutput(output any) (*anypb.Any, error) {
	if output == nil {
		return nil, fmt.Errorf(
			"provider output must not be nil",
		)
	}

	if value, ok := output.(*anypb.Any); ok {
		if value == nil {
			return nil, fmt.Errorf(
				"provider output Any must not be nil",
			)
		}

		return value, nil
	}

	message, ok := output.(proto.Message)
	if !ok {
		return nil, fmt.Errorf(
			"provider output type %T cannot be normalized to protobuf Any",
			output,
		)
	}

	value, err := anypb.New(message)
	if err != nil {
		return nil, fmt.Errorf(
			"failed to create protobuf Any: %w",
			err,
		)
	}

	return value, nil
}
