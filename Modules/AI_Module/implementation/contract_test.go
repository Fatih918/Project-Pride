package implementation

import (
	"context"
	"testing"

	executioncontract "pride-ai-module/contracts/execution"

	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type contractTestImplementation struct{}

func (contractTestImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if ctx == nil {
		return executioncontract.AIExecutionResult{}, nil
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
		Metadata:   request.Metadata,
	}, nil
}

func TestAIImplementationContractCanBeImplemented(
	t *testing.T,
) {
	var _ AIImplementation = (*contractTestImplementation)(nil)
}

func TestAIImplementationContractExecute(
	t *testing.T,
) {
	ctx := context.Background()

	input, err := anypb.New(
		wrapperspb.String("test-input"),
	)
	if err != nil {
		t.Fatalf(
			"failed to create input: %v",
			err,
		)
	}

	request := executioncontract.AIExecutionRequest{
		RequestID:  "request-1",
		Capability: "test.generate",
		Input:      input,
		Metadata: map[string]any{
			"model": "test-model",
		},
	}

	implementation := &contractTestImplementation{}

	result, err := implementation.Execute(
		ctx,
		request,
		executioncontract.AIExecutionContext{},
	)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
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

	if result.Output == nil {
		t.Fatal(
			"expected output to be present",
		)
	}

	if result.Output != request.Input {
		t.Fatal(
			"expected output to preserve the request input",
		)
	}

	if result.Metadata["model"] != "test-model" {
		t.Fatalf(
			"unexpected metadata: got %v",
			result.Metadata,
		)
	}
}
