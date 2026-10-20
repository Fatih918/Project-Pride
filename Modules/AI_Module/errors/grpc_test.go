package errors

import (
	stdErrors "errors"
	"strings"
	"testing"

	aiv1 "pride-ai-module/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGRPCStatus(t *testing.T) {
	err := New(
		CodeAuthenticationError,
		"AI provider authentication failed",
	).
		WithRequestID("req-001").
		WithMetadata(map[string]any{
			"provider": "test-provider",
		})

	grpcErr := GRPCStatus(err)

	if grpcErr == nil {
		t.Fatal("GRPCStatus() returned nil")
	}

	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Unauthenticated {
		t.Fatalf(
			"unexpected gRPC code: got %v, want %v",
			st.Code(),
			codes.Unauthenticated,
		)
	}

	var detail *aiv1.AIError

	for _, d := range st.Details() {
		if aiErr, ok := d.(*aiv1.AIError); ok {
			detail = aiErr
			break
		}
	}

	if detail == nil {
		t.Fatal("AIError detail not found")
	}

	if detail.GetRequestId() != "req-001" {
		t.Fatalf(
			"unexpected request ID: got %q",
			detail.GetRequestId(),
		)
	}

	if detail.GetErrorCode() != string(CodeAuthenticationError) {
		t.Fatalf(
			"unexpected error code: got %q",
			detail.GetErrorCode(),
		)
	}

	if detail.GetMessage() != "AI provider authentication failed" {
		t.Fatalf(
			"unexpected message: got %q",
			detail.GetMessage(),
		)
	}

	if detail.GetMetadata() == nil {
		t.Fatal("expected metadata")
	}

	provider := detail.
		GetMetadata().
		GetFields()["provider"].
		GetStringValue()

	if provider != "test-provider" {
		t.Fatalf(
			"unexpected provider metadata: got %q",
			provider,
		)
	}
}

func TestGRPCStatusDoesNotExposeCause(t *testing.T) {
	internalErr := stdErrors.New(
		"api-key=SUPER_SECRET provider_internal_password=SECRET",
	)

	err := New(
		CodeRuntimeError,
		"AI provider execution failed",
	).
		WithRequestID("req-secret").
		WithCause(internalErr)

	grpcErr := GRPCStatus(err)

	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if containsSecret(st.Message()) {
		t.Fatal("gRPC status exposed internal secret")
	}

	detail, ok := FromGRPCStatus(grpcErr)
	if !ok {
		t.Fatal("AIError detail not found")
	}

	if containsSecret(detail.Message) {
		t.Fatal("AIError exposed internal secret")
	}
}

func TestGRPCStatusUnknownErrorIsNormalized(t *testing.T) {
	internalErr := stdErrors.New(
		"provider implementation exploded",
	)

	grpcErr := GRPCStatus(internalErr)

	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatal("expected gRPC status error")
	}

	if st.Code() != codes.Unknown {
		t.Fatalf(
			"unexpected gRPC code: got %v",
			st.Code(),
		)
	}

	if st.Message() != "AI execution failed" {
		t.Fatalf(
			"unexpected message: got %q",
			st.Message(),
		)
	}

	detail, ok := FromGRPCStatus(grpcErr)
	if !ok {
		t.Fatal("AIError detail not found")
	}

	if detail.Code != CodeUnknownError {
		t.Fatalf(
			"unexpected AI error code: got %q",
			detail.Code,
		)
	}
}

func TestFromGRPCStatus(t *testing.T) {
	original := New(
		CodeRateLimit,
		"AI provider rate limit reached",
	).
		WithRequestID("req-rate-limit").
		WithMetadata(map[string]any{
			"retryable": true,
		})

	grpcErr := GRPCStatus(original)

	result, ok := FromGRPCStatus(grpcErr)
	if !ok {
		t.Fatal("FromGRPCStatus() failed")
	}

	if result.RequestID != original.RequestID {
		t.Fatalf(
			"unexpected request ID: got %q, want %q",
			result.RequestID,
			original.RequestID,
		)
	}

	if result.Code != original.Code {
		t.Fatalf(
			"unexpected code: got %q",
			result.Code,
		)
	}

	if result.Message != original.Message {
		t.Fatalf(
			"unexpected message: got %q",
			result.Message,
		)
	}

	if result.Metadata["retryable"] != true {
		t.Fatal("metadata was not preserved")
	}
}

func TestFromGRPCStatusWithoutAIError(t *testing.T) {
	err := status.Error(
		codes.Internal,
		"plain grpc error",
	)

	result, ok := FromGRPCStatus(err)

	if ok {
		t.Fatal("expected no AIError detail")
	}

	if result != nil {
		t.Fatal("expected nil result")
	}
}

func TestGRPCStatusNil(t *testing.T) {
	if result := GRPCStatus(nil); result != nil {
		t.Fatalf(
			"GRPCStatus(nil) = %v, want nil",
			result,
		)
	}
}

func TestGRPCCodeMapping(t *testing.T) {
	tests := []struct {
		name string
		code Code
		want codes.Code
	}{
		{
			name: "configuration",
			code: CodeConfigurationError,
			want: codes.FailedPrecondition,
		},
		{
			name: "authentication",
			code: CodeAuthenticationError,
			want: codes.Unauthenticated,
		},
		{
			name: "permission",
			code: CodePermissionError,
			want: codes.PermissionDenied,
		},
		{
			name: "connection",
			code: CodeConnectionError,
			want: codes.Unavailable,
		},
		{
			name: "timeout",
			code: CodeTimeoutError,
			want: codes.DeadlineExceeded,
		},
		{
			name: "capability",
			code: CodeCapabilityUnsupported,
			want: codes.Unimplemented,
		},
		{
			name: "rate-limit",
			code: CodeRateLimit,
			want: codes.ResourceExhausted,
		},
		{
			name: "runtime",
			code: CodeRuntimeError,
			want: codes.Internal,
		},
		{
			name: "invalid-request",
			code: CodeInvalidRequest,
			want: codes.InvalidArgument,
		},
		{
			name: "unknown",
			code: CodeUnknownError,
			want: codes.Unknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := New(tt.code, "test error")

			grpcErr := GRPCStatus(err)

			st, ok := status.FromError(grpcErr)
			if !ok {
				t.Fatal("expected gRPC status error")
			}

			if st.Code() != tt.want {
				t.Fatalf(
					"unexpected gRPC code: got %v, want %v",
					st.Code(),
					tt.want,
				)
			}
		})
	}
}

func containsSecret(value string) bool {
	secrets := []string{
		"SUPER_SECRET",
		"SECRET",
		"api-key=",
		"provider_internal_password=",
	}

	for _, secret := range secrets {
		if strings.Contains(value, secret) {
			return true
		}
	}

	return false
}
