package gemini

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/protobuf/types/known/wrapperspb"

	"pride-ai-module/contracts/execution"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/provider"
)

type failingProviderForNormalization struct {
	err error
}

func (p failingProviderForNormalization) Execute(
	ctx context.Context,
	request provider.Request,
) (provider.Response, error) {
	return provider.Response{}, p.err
}

func TestAdapterNormalizesProviderFailureAndPreservesRequestID(t *testing.T) {
	const requestID = "request-failure-123"
	const secret = "api-key=DO_NOT_EXPOSE"
	cause := errors.New(secret)

	adapter, err := NewAdapter(failingProviderForNormalization{err: cause})
	if err != nil {
		t.Fatalf("NewAdapter returned error: %v", err)
	}

	_, err = adapter.Execute(
		context.Background(),
		execution.AIExecutionRequest{
			RequestID:  requestID,
			Capability: CapabilityTextGenerate,
			Input:      wrapperspb.String("test"),
		},
		execution.AIExecutionContext{},
	)
	if err == nil {
		t.Fatal("expected provider failure")
	}

	var normalized *prideerrors.Error
	if !errors.As(err, &normalized) {
		t.Fatalf("expected normalized PRIDE error, got %T", err)
	}
	if normalized.Code != prideerrors.CodeRuntimeError {
		t.Fatalf("error code = %q, want %q", normalized.Code, prideerrors.CodeRuntimeError)
	}
	if normalized.RequestID != requestID {
		t.Fatalf("request ID = %q, want %q", normalized.RequestID, requestID)
	}
	if !errors.Is(err, cause) {
		t.Fatal("expected internal cause to remain inspectable")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("public error string exposed provider details")
	}
	if normalized.Cause == nil {
		t.Fatal("expected internal cause to be retained")
	}
}

func TestPublicPRIDEErrorStringDoesNotExposeCause(t *testing.T) {
	const secret = "token=PRIVATE_VALUE"
	cause := errors.New(secret)
	err := prideerrors.New(
		prideerrors.CodeAuthenticationError,
		"provider authentication failed",
	).WithCause(cause)

	if strings.Contains(err.Error(), secret) {
		t.Fatal("public error string exposed the internal cause")
	}
	if !errors.Is(err, cause) {
		t.Fatal("cause should remain inspectable through errors.Is")
	}
}
