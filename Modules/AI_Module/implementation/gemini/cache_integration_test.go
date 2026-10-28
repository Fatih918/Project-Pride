package gemini

import (
	"context"
	"sync/atomic"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"pride-ai-module/configuration"
	"pride-ai-module/contracts/execution"
	"pride-ai-module/provider"
)

type cacheTrackingProvider struct {
	calls  atomic.Int32
	closes atomic.Int32
}

func (p *cacheTrackingProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	p.calls.Add(1)
	return provider.Response{Output: wrapperspb.String("cached result")}, nil
}

func (p *cacheTrackingProvider) Close() error {
	p.closes.Add(1)
	return nil
}

func cacheTestContext(credential string) execution.AIExecutionContext {
	return execution.AIExecutionContext{
		Configuration: configuration.AIConfiguration{
			Ref:               "config-gemini",
			ImplementationRef: "gemini",
			ConnectionParameters: map[string]any{
				ConnectionParameterModel: "test-model",
			},
			CredentialRef: "credential-gemini",
			Enabled:       true,
		},
		Credential: []byte(credential),
	}
}

func TestAdapterWithFactoryReusesCachedProviderClient(t *testing.T) {
	var factoryCalls atomic.Int32
	var created atomic.Pointer[cacheTrackingProvider]

	adapter, err := NewAdapterWithFactory(func(
		ctx context.Context,
		executionContext execution.AIExecutionContext,
	) (provider.Provider, error) {
		factoryCalls.Add(1)
		client := &cacheTrackingProvider{}
		created.Store(client)
		return client, nil
	})
	if err != nil {
		t.Fatalf("NewAdapterWithFactory returned error: %v", err)
	}
	t.Cleanup(func() {
		if err := adapter.Close(); err != nil {
			t.Errorf("adapter Close returned error: %v", err)
		}
	})

	for i := 0; i < 2; i++ {
		_, err := adapter.Execute(
			context.Background(),
			execution.AIExecutionRequest{RequestID: "request", Capability: CapabilityTextGenerate},
			cacheTestContext("secret-a"),
		)
		if err != nil {
			t.Fatalf("Execute %d returned error: %v", i, err)
		}
	}

	if got := factoryCalls.Load(); got != 1 {
		t.Fatalf("factory calls = %d, want 1", got)
	}
	if client := created.Load(); client == nil || client.calls.Load() != 2 {
		t.Fatalf("cached provider execute calls = %v, want 2", client)
	}
}

func TestAdapterCacheSeparatesCredentialRotationAndDefersClose(t *testing.T) {
	var factoryCalls atomic.Int32
	var clients []*cacheTrackingProvider

	cache, err := provider.NewClientCache()
	if err != nil {
		t.Fatalf("NewClientCache returned error: %v", err)
	}
	adapter, err := NewAdapterWithCache(cache, func(
		ctx context.Context,
		executionContext execution.AIExecutionContext,
	) (provider.Provider, error) {
		factoryCalls.Add(1)
		client := &cacheTrackingProvider{}
		clients = append(clients, client)
		return client, nil
	})
	if err != nil {
		t.Fatalf("NewAdapterWithCache returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{RequestID: "request-a", Capability: CapabilityTextGenerate},
		cacheTestContext("secret-a"),
	)
	if err != nil {
		t.Fatalf("first Execute returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{RequestID: "request-b", Capability: CapabilityTextGenerate},
		cacheTestContext("secret-b"),
	)
	if err != nil {
		t.Fatalf("second Execute returned error: %v", err)
	}

	if got := factoryCalls.Load(); got != 2 {
		t.Fatalf("factory calls = %d, want 2 after credential rotation", got)
	}

	if err := cache.InvalidateConfiguration("config-gemini"); err != nil {
		t.Fatalf("InvalidateConfiguration returned error: %v", err)
	}
	for i, client := range clients {
		if got := client.closes.Load(); got != 1 {
			t.Errorf("client %d close calls = %d, want 1", i, got)
		}
	}

	if err := cache.Close(); err != nil {
		t.Fatalf("cache Close returned error: %v", err)
	}
}
