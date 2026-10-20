package implementation_test

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"pride-ai-module/contracts/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/implementation/gemini"
	"pride-ai-module/implementation/local"
	"pride-ai-module/implementation/openai"
	"pride-ai-module/provider"
)

type adapterFactory struct {
	name string
	new  func(provider.Provider) (implementation.AIImplementation, error)
}

type testProvider struct {
	output any
	err    error
}

func (p *testProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	if p.err != nil {
		return provider.Response{}, p.err
	}

	return provider.Response{
		Output: p.output,
		Metadata: map[string]any{
			"provider":   "adapter-contract-test",
			"capability": request.Capability,
		},
	}, nil
}

func adapterFactories() []adapterFactory {
	return []adapterFactory{
		{
			name: "gemini",
			new: func(p provider.Provider) (
				implementation.AIImplementation,
				error,
			) {
				return gemini.NewAdapter(p)
			},
		},
		{
			name: "openai",
			new: func(p provider.Provider) (
				implementation.AIImplementation,
				error,
			) {
				return openai.NewAdapter(p)
			},
		},
		{
			name: "local",
			new: func(p provider.Provider) (
				implementation.AIImplementation,
				error,
			) {
				return local.NewAdapter(p)
			},
		},
	}
}

func TestAdaptersImplementContract(t *testing.T) {
	for _, factory := range adapterFactories() {
		t.Run(factory.name, func(t *testing.T) {
			output := wrapperspb.String(
				factory.name + " output",
			)

			adapter, err := factory.new(
				&testProvider{
					output: output,
				},
			)
			if err != nil {
				t.Fatalf(
					"failed to create adapter: %v",
					err,
				)
			}

			if adapter == nil {
				t.Fatal("adapter must not be nil")
			}

			var _ implementation.AIImplementation = adapter
		})
	}
}

func TestAdaptersPreserveExecutionContract(t *testing.T) {
	for _, factory := range adapterFactories() {
		t.Run(factory.name, func(t *testing.T) {
			output := wrapperspb.String(
				factory.name + " output",
			)

			adapter, err := factory.new(
				&testProvider{
					output: output,
				},
			)
			if err != nil {
				t.Fatalf(
					"failed to create adapter: %v",
					err,
				)
			}

			input := wrapperspb.String("input")

			inputAny, err := anypb.New(input)
			if err != nil {
				t.Fatalf(
					"failed to create input Any: %v",
					err,
				)
			}

			request := execution.AIExecutionRequest{
				RequestID:  "adapter-contract-request",
				Capability: "text.generate",
				Input:      inputAny,
				Metadata: map[string]any{
					"test": "adapter-contract",
				},
			}

			result, err := adapter.Execute(
				context.Background(),
				request,
				execution.AIExecutionContext{},
			)
			if err != nil {
				t.Fatalf(
					"Execute returned error: %v",
					err,
				)
			}

			if result.RequestID != request.RequestID {
				t.Fatalf(
					"RequestID = %q, want %q",
					result.RequestID,
					request.RequestID,
				)
			}

			if result.Capability != request.Capability {
				t.Fatalf(
					"Capability = %q, want %q",
					result.Capability,
					request.Capability,
				)
			}

			if result.Output == nil {
				t.Fatal("result Output must not be nil")
			}

			var decoded wrapperspb.StringValue

			if err := result.Output.UnmarshalTo(&decoded); err != nil {
				t.Fatalf(
					"failed to decode result Output: %v",
					err,
				)
			}

			if !proto.Equal(&decoded, output) {
				t.Fatalf(
					"decoded output = %v, want %v",
					&decoded,
					output,
				)
			}

			if result.Metadata["provider"] != "adapter-contract-test" {
				t.Fatalf(
					"unexpected metadata: %#v",
					result.Metadata,
				)
			}
		})
	}
}

func TestAdaptersPropagateProviderErrors(t *testing.T) {
	expectedErr := errors.New("provider execution failed")

	for _, factory := range adapterFactories() {
		t.Run(factory.name, func(t *testing.T) {
			adapter, err := factory.new(
				&testProvider{
					err: expectedErr,
				},
			)
			if err != nil {
				t.Fatalf(
					"failed to create adapter: %v",
					err,
				)
			}

			_, err = adapter.Execute(
				context.Background(),
				execution.AIExecutionRequest{
					RequestID:  "adapter-error-request",
					Capability: "text.generate",
				},
				execution.AIExecutionContext{},
			)

			if err == nil {
				t.Fatal("expected error, got nil")
			}

			if !errors.Is(err, expectedErr) {
				t.Fatalf(
					"error = %v, want wrapped provider error %v",
					err,
					expectedErr,
				)
			}
		})
	}
}

func TestAdaptersRejectUnsupportedOutput(t *testing.T) {
	for _, factory := range adapterFactories() {
		t.Run(factory.name, func(t *testing.T) {
			adapter, err := factory.new(
				&testProvider{
					output: "unsupported output",
				},
			)
			if err != nil {
				t.Fatalf(
					"failed to create adapter: %v",
					err,
				)
			}

			_, err = adapter.Execute(
				context.Background(),
				execution.AIExecutionRequest{
					RequestID:  "adapter-invalid-output",
					Capability: "text.generate",
				},
				execution.AIExecutionContext{},
			)

			if err == nil {
				t.Fatal("expected normalization error, got nil")
			}
		})
	}
}

func TestAdaptersRespectContextCancellation(t *testing.T) {
	for _, factory := range adapterFactories() {
		t.Run(factory.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(
				context.Background(),
			)
			cancel()

			adapter, err := factory.new(
				&testProvider{
					output: wrapperspb.String("unused"),
				},
			)
			if err != nil {
				t.Fatalf(
					"failed to create adapter: %v",
					err,
				)
			}

			_, err = adapter.Execute(
				ctx,
				execution.AIExecutionRequest{
					RequestID:  "adapter-cancelled",
					Capability: "text.generate",
				},
				execution.AIExecutionContext{},
			)

			if err == nil {
				t.Fatal("expected context cancellation error, got nil")
			}

			if !errors.Is(err, context.Canceled) {
				t.Fatalf(
					"error = %v, want context.Canceled",
					err,
				)
			}
		})
	}
}
