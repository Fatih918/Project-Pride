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
	clientCache     *provider.ClientCache
	ownsCache       bool
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

// NewAdapterWithFactory creates an adapter whose provider clients are cached
// by effective configuration and credential identity. The adapter owns the
// cache and must be closed when it is no longer used.
func NewAdapterWithFactory(factory ProviderFactory) (*Adapter, error) {
	cache, err := provider.NewClientCache()
	if err != nil {
		return nil, fmt.Errorf("create Gemini provider client cache")
	}

	adapter, err := NewAdapterWithCache(cache, factory)
	if err != nil {
		_ = cache.Close()
		return nil, err
	}
	adapter.ownsCache = true
	return adapter, nil
}

// NewAdapterWithCache creates an adapter using a caller-owned provider cache.
// The caller remains responsible for closing the cache after all adapters
// using it have stopped executing.
func NewAdapterWithCache(
	cache *provider.ClientCache,
	factory ProviderFactory,
) (*Adapter, error) {
	if cache == nil {
		return nil, fmt.Errorf(
			"Gemini provider client cache must not be nil",
		)
	}
	if factory == nil {
		return nil, fmt.Errorf(
			"Gemini provider factory must not be nil",
		)
	}

	return &Adapter{
		providerFactory: factory,
		clientCache:     cache,
	}, nil
}

// Close retires clients owned by this adapter. Active executions may finish;
// their retired clients are closed when their leases are released.
// For adapters created with NewAdapterWithCache, close the shared cache
// separately after all users of that cache have stopped.
func (a *Adapter) Close() error {
	if a == nil || a.clientCache == nil || !a.ownsCache {
		return nil
	}
	return a.clientCache.Close()
}

func (a *Adapter) Execute(
	ctx context.Context,
	request execution.AIExecutionRequest,
	executionContext execution.AIExecutionContext,
) (result execution.AIExecutionResult, err error) {
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

	p, release, err := a.resolveProvider(ctx, executionContext)
	if err != nil {
		return execution.AIExecutionResult{}, fmt.Errorf(
			"Gemini provider resolution failed",
		)
	}
	if release != nil {
		defer func() {
			if releaseErr := release(); releaseErr != nil && err == nil {
				err = fmt.Errorf("Gemini provider lease release failed")
			}
		}()
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
) (provider.Provider, func() error, error) {
	if a.providerFactory != nil {
		if a.clientCache == nil {
			p, err := a.providerFactory(ctx, executionContext)
			if err != nil {
				return nil, nil, err
			}
			if p == nil {
				return nil, nil, fmt.Errorf("Gemini provider factory returned nil provider")
			}
			return p, nil, nil
		}

		config := executionContext.Configuration
		key, err := a.clientCache.NewKey(
			config.Ref,
			config.ImplementationRef,
			config.ConnectionParameters,
			executionContext.Credential,
		)
		if err != nil {
			return nil, nil, fmt.Errorf("prepare Gemini provider client cache key: %w", err)
		}

		lease, err := a.clientCache.Acquire(
			ctx,
			key,
			func(factoryContext context.Context) (provider.Provider, error) {
				return a.providerFactory(factoryContext, executionContext)
			},
		)
		if err != nil {
			return nil, nil, fmt.Errorf("acquire Gemini provider client: %w", err)
		}

		return lease.Provider(), lease.Release, nil
	}

	if a.provider == nil {
		return nil, nil, fmt.Errorf(
			"Gemini adapter provider must not be nil",
		)
	}

	return a.provider, nil, nil
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
