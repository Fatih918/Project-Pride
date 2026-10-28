package readiness_test

import (
	"context"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	geminiadapter "pride-ai-module/implementation/gemini"
	localadapter "pride-ai-module/implementation/local"
	openaadapter "pride-ai-module/implementation/openai"
	"pride-ai-module/provider"
	"pride-ai-module/selection"
)

type readinessProvider struct {
	called bool
	request provider.Request
}

func (p *readinessProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	p.called = true
	p.request = request

	return provider.Response{
		Output: wrapperspb.String("readiness-ok"),
		Metadata: map[string]any{
			"provider": "readiness-test",
			"model":    "test-model",
		},
	}, nil
}

func TestAdaptersPassReadinessGateWithoutCredential(t *testing.T) {
	tests := []struct {
		name string
		new  func(provider.Provider) (implementation.AIImplementation, error)
	}{
		{
			name: "local",
			new: func(p provider.Provider) (implementation.AIImplementation, error) {
				return localadapter.NewAdapter(p)
			},
		},
		{
			name: "openai",
			new: func(p provider.Provider) (implementation.AIImplementation, error) {
				return openaadapter.NewAdapter(p)
			},
		},
		{
			name: "gemini",
			new: func(p provider.Provider) (implementation.AIImplementation, error) {
				return geminiadapter.NewAdapter(p)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			const (
				requestID = "r11-readiness-request"
				capability = "text.generate"
			)

			fakeProvider := &readinessProvider{}
			adapter, err := tt.new(fakeProvider)
			if err != nil {
				t.Fatalf("create adapter: %v", err)
			}

			configStore := configuration.NewMemoryStore()
			config := configuration.AIConfiguration{
				Ref:               "r11." + tt.name,
				Name:              "R11 " + tt.name,
				Type:              tt.name,
				ImplementationRef: "r11." + tt.name,
				Enabled:           true,
			}
			if err := configStore.Create(ctx, config); err != nil {
				t.Fatalf("create configuration: %v", err)
			}
			if err := configStore.Select(ctx, config.Ref); err != nil {
				t.Fatalf("select configuration: %v", err)
			}

			selectionService := selection.NewService(selection.NewMemoryStore())
			if err := selectionService.Select(ctx, config.Ref); err != nil {
				t.Fatalf("select configuration reference: %v", err)
			}

			resolver := implementation.NewResolver()
			if err := resolver.Register(config.ImplementationRef, adapter); err != nil {
				t.Fatalf("register adapter: %v", err)
			}

			// A nil credential resolver is intentional: configurations without a
			// CredentialRef must still execute without touching credential storage.
			executor := execution.NewExecutor(
				configStore,
				selectionService,
				resolver,
				nil,
			)

			input, err := anypb.New(wrapperspb.String("readiness-input"))
			if err != nil {
				t.Fatalf("create input: %v", err)
			}

			result, err := executor.Execute(ctx, executioncontract.AIExecutionRequest{
				RequestID:  requestID,
				Capability: capability,
				Input:      input,
				Metadata:   map[string]any{"trace": "r11"},
			})
			if err != nil {
				t.Fatalf("execute through readiness path: %v", err)
			}

			if !fakeProvider.called {
				t.Fatal("provider was not invoked")
			}
			if fakeProvider.request.Capability != capability {
				t.Fatalf("provider capability = %q, want %q", fakeProvider.request.Capability, capability)
			}
			if fakeProvider.request.Input != input {
				t.Fatal("provider input did not preserve the original Any payload")
			}
			if fakeProvider.request.Metadata["trace"] != "r11" {
				t.Fatalf("provider metadata = %#v, want trace=r11", fakeProvider.request.Metadata)
			}
			if result.RequestID != requestID {
				t.Fatalf("result request ID = %q, want %q", result.RequestID, requestID)
			}
			if result.Capability != capability {
				t.Fatalf("result capability = %q, want %q", result.Capability, capability)
			}
			if result.Metadata["provider"] != "readiness-test" {
				t.Fatalf("result metadata = %#v", result.Metadata)
			}
			if result.Output == nil {
				t.Fatal("result output is nil")
			}

			var output wrapperspb.StringValue
			if err := result.Output.UnmarshalTo(&output); err != nil {
				t.Fatalf("decode output: %v", err)
			}
			if output.Value != "readiness-ok" {
				t.Fatalf("output = %q, want readiness-ok", output.Value)
			}
		})
	}
}

func TestGeminiRealProviderFactoryFailsClosedBeforeExecution(t *testing.T) {
	tests := []struct {
		name              string
		executionContext  executioncontract.AIExecutionContext
		wantErrorContains string
	}{
		{
			name: "missing credential",
			executionContext: executioncontract.AIExecutionContext{
				Configuration: configuration.AIConfiguration{
					ConnectionParameters: map[string]any{"model": "test-model"},
				},
			},
			wantErrorContains: "CONFIGURATION_ERROR: Gemini provider resolution failed",
		},
		{
			name: "missing model",
			executionContext: executioncontract.AIExecutionContext{
				Configuration: configuration.AIConfiguration{
					ConnectionParameters: map[string]any{},
				},
				Credential: []byte("R11_TEST_CREDENTIAL"),
			},
			wantErrorContains: "CONFIGURATION_ERROR: Gemini provider resolution failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			adapter, err := geminiadapter.NewAdapterWithFactory(
				geminiadapter.NewRealProviderFactory(),
			)
			if err != nil {
				t.Fatalf("create Gemini adapter: %v", err)
			}

			_, err = adapter.Execute(
				context.Background(),
				executioncontract.AIExecutionRequest{
					RequestID:  "r11-gemini-validation",
					Capability: geminiadapter.CapabilityTextGenerate,
				},
				tt.executionContext,
			)
			if err == nil {
				t.Fatal("expected invalid provider configuration to fail")
			}
			if !strings.Contains(err.Error(), tt.wantErrorContains) {
				t.Fatalf("error = %q, want it to contain %q", err, tt.wantErrorContains)
			}
		})
	}
}
