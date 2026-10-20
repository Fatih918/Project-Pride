package provider

import (
	"context"
	"testing"
)

// TestAdapter is a deterministic provider implementation
// used for provider boundary and integration tests.
type TestAdapter struct {
	Output any
}

func (a *TestAdapter) Execute(
	ctx context.Context,
	request Request,
) (Response, error) {
	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	output := a.Output

	if output == nil {
		output = request.Input
	}

	return Response{
		Output: output,
		Metadata: map[string]any{
			"provider": "test",
		},
	}, nil
}

func TestTestAdapterImplementsProvider(t *testing.T) {
	var _ Provider = (*TestAdapter)(nil)
}

func TestTestAdapterExecute(t *testing.T) {
	ctx := context.Background()

	adapter := &TestAdapter{
		Output: "test-output",
	}

	request := Request{
		Capability: "test.generate",
		Input:      "test-input",
		Metadata: map[string]any{
			"request_id": "req_test",
		},
	}

	result, err := adapter.Execute(ctx, request)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if result.Output != "test-output" {
		t.Fatalf(
			"unexpected output: got %v, want %v",
			result.Output,
			"test-output",
		)
	}

	if result.Metadata["provider"] != "test" {
		t.Fatalf(
			"unexpected provider metadata: got %v",
			result.Metadata["provider"],
		)
	}
}

func TestTestAdapterDefaultsOutputToInput(t *testing.T) {
	ctx := context.Background()

	adapter := &TestAdapter{}

	request := Request{
		Capability: "test.echo",
		Input:      "hello",
	}

	result, err := adapter.Execute(ctx, request)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if result.Output != request.Input {
		t.Fatalf(
			"unexpected output: got %v, want %v",
			result.Output,
			request.Input,
		)
	}
}

func TestTestAdapterRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	cancel()

	adapter := &TestAdapter{}

	_, err := adapter.Execute(
		ctx,
		Request{
			Capability: "test.generate",
			Input:      "input",
		},
	)

	if err == nil {
		t.Fatal(
			"expected context cancellation error",
		)
	}

	if err != context.Canceled {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			context.Canceled,
		)
	}
}

var _ Provider = (*TestAdapter)(nil)
