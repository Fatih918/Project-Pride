package openai

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"

	"pride-ai-module/contracts/execution"
	"pride-ai-module/implementation"
	"pride-ai-module/provider"
)

type testProvider struct {
	request provider.Request
	output  any
	err     error
}

func (p *testProvider) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	p.request = request

	if err := ctx.Err(); err != nil {
		return provider.Response{}, err
	}

	if p.err != nil {
		return provider.Response{}, p.err
	}

	return provider.Response{
		Output: p.output,
		Metadata: map[string]any{
			"provider": "openai-test",
		},
	}, nil
}

var _ implementation.AIImplementation = (*Adapter)(nil)

func TestNewAdapter(t *testing.T) {
	p := &testProvider{}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	if adapter == nil {
		t.Fatal("NewAdapter returned nil adapter")
	}

	if adapter.provider != p {
		t.Fatal("adapter provider does not match supplied provider")
	}
}

func TestNewAdapterRejectsNilProvider(t *testing.T) {
	adapter, err := NewAdapter(nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if adapter != nil {
		t.Fatal("expected nil adapter, got non-nil")
	}
}

func TestAdapterExecutePreservesRequestAndNormalizesAny(
	t *testing.T,
) {
	output := wrapperspb.String("openai output")

	outputAny, err := anypb.New(output)
	if err != nil {
		t.Fatalf("failed to create test Any: %v", err)
	}

	p := &testProvider{
		output: outputAny,
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	input := wrapperspb.String("input")

	inputAny, err := anypb.New(input)
	if err != nil {
		t.Fatalf("failed to create input Any: %v", err)
	}

	request := execution.AIExecutionRequest{
		RequestID:  "request-123",
		Capability: "text.generate",
		Input:      inputAny,
		Metadata: map[string]any{
			"test": "value",
		},
	}

	result, err := adapter.Execute(
		context.Background(),
		request,
		execution.AIExecutionContext{},
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
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
		t.Fatal("result Output is nil")
	}

	if result.Output != outputAny {
		t.Fatal("adapter should preserve an existing *anypb.Any output")
	}

	if result.Metadata["provider"] != "openai-test" {
		t.Fatalf("unexpected metadata: %#v", result.Metadata)
	}

	if p.request.Capability != request.Capability {
		t.Fatalf(
			"provider Capability = %q, want %q",
			p.request.Capability,
			request.Capability,
		)
	}

	if p.request.Input != request.Input {
		t.Fatal("provider Input does not match request Input")
	}

	if p.request.Metadata["test"] != "value" {
		t.Fatalf(
			"provider Metadata = %#v, want test=value",
			p.request.Metadata,
		)
	}
}

func TestAdapterExecuteNormalizesProtoMessageToAny(
	t *testing.T,
) {
	output := wrapperspb.String("openai output")

	p := &testProvider{
		output: output,
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	result, err := adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  "request-456",
			Capability: "text.generate",
		},
		execution.AIExecutionContext{},
	)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if result.Output == nil {
		t.Fatal("result Output is nil")
	}

	var decoded wrapperspb.StringValue

	if err := result.Output.UnmarshalTo(&decoded); err != nil {
		t.Fatalf(
			"failed to unmarshal normalized output: %v",
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
}

func TestAdapterExecutePropagatesProviderError(t *testing.T) {
	expectedErr := errors.New("provider failed")

	p := &testProvider{
		err: expectedErr,
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  "request-error",
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
}

func TestAdapterExecuteRejectsUnsupportedOutput(t *testing.T) {
	p := &testProvider{
		output: "plain string",
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  "request-invalid-output",
			Capability: "text.generate",
		},
		execution.AIExecutionContext{},
	)

	if err == nil {
		t.Fatal("expected normalization error, got nil")
	}
}

func TestAdapterExecuteRejectsNilOutput(t *testing.T) {
	p := &testProvider{
		output: nil,
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  "request-nil-output",
			Capability: "text.generate",
		},
		execution.AIExecutionContext{},
	)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestAdapterExecuteRespectsCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	p := &testProvider{
		output: wrapperspb.String("unused"),
	}

	adapter, err := NewAdapter(p)
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	_, err = adapter.Execute(
		ctx,
		execution.AIExecutionRequest{
			RequestID:  "request-cancelled",
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
}
