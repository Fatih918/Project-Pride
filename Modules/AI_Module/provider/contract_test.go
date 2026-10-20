package provider

import (
	"context"
	"errors"
	"testing"
)

type contractTestProvider struct {
	response Response
	err      error

	called bool
}

func (p *contractTestProvider) Execute(
	ctx context.Context,
	request Request,
) (Response, error) {
	if ctx == nil {
		return Response{}, errors.New(
			"context must not be nil",
		)
	}

	p.called = true

	if err := ctx.Err(); err != nil {
		return Response{}, err
	}

	if p.err != nil {
		return Response{}, p.err
	}

	return p.response, nil
}

func TestProviderContractCanBeImplemented(t *testing.T) {
	var _ Provider = (*contractTestProvider)(nil)
}

func TestProviderContractExecute(t *testing.T) {
	ctx := context.Background()

	provider := &contractTestProvider{
		response: Response{
			Output: "provider-output",
			Metadata: map[string]any{
				"source": "test-provider",
			},
		},
	}

	request := Request{
		Capability: "test.generate",
		Input:      "provider-input",
		Metadata: map[string]any{
			"model": "test-model",
		},
	}

	result, err := provider.Execute(
		ctx,
		request,
	)
	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if !provider.called {
		t.Fatal(
			"provider Execute() was not called",
		)
	}

	if result.Output != "provider-output" {
		t.Fatalf(
			"unexpected output: got %v, want %v",
			result.Output,
			"provider-output",
		)
	}

	if result.Metadata["source"] != "test-provider" {
		t.Fatalf(
			"unexpected metadata: got %v",
			result.Metadata,
		)
	}
}

func TestProviderContractPropagatesError(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New(
		"provider execution failed",
	)

	provider := &contractTestProvider{
		err: expectedErr,
	}

	_, err := provider.Execute(
		ctx,
		Request{
			Capability: "test.generate",
		},
	)
	if !errors.Is(err, expectedErr) {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			expectedErr,
		)
	}
}

func TestProviderContractRespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(
		context.Background(),
	)

	cancel()

	provider := &contractTestProvider{
		response: Response{
			Output: "should-not-be-returned",
		},
	}

	_, err := provider.Execute(
		ctx,
		Request{
			Capability: "test.generate",
		},
	)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf(
			"unexpected error: got %v, want %v",
			err,
			context.Canceled,
		)
	}
}
