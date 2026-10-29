package grpc

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"pride-ai-module/configuration"
	executioncontract "pride-ai-module/contracts/execution"
	prideerrors "pride-ai-module/errors"
	"pride-ai-module/execution"
	"pride-ai-module/implementation"
	aiv1 "pride-ai-module/proto"
	"pride-ai-module/selection"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type testConfigurationProvider struct {
	configuration configuration.AIConfiguration
}

func (p *testConfigurationProvider) GetActive(
	ctx context.Context,
) (configuration.AIConfiguration, error) {
	return p.configuration, nil
}

func (p *testConfigurationProvider) GetByRef(
	ctx context.Context,
	ref string,
) (configuration.AIConfiguration, error) {
	if ref != p.configuration.Ref {
		return configuration.AIConfiguration{}, fmt.Errorf(
			"configuration not found: %s",
			ref,
		)
	}

	return p.configuration, nil
}

type testCredentialResolver struct{}

func (r *testCredentialResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	return []byte("test-api-key"), nil
}

var _ executioncontract.CredentialResolver = (*testCredentialResolver)(nil)

type testImplementation struct{}

func (i *testImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	outputMessage := &aiv1.ExecuteResponse{
		RequestId:  request.RequestID,
		Capability: request.Capability,
	}

	output, err := anypb.New(outputMessage)
	if err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     output,
		Metadata: map[string]any{
			"source": "test-implementation",
		},
	}, nil
}

var _ implementation.AIImplementation = (*testImplementation)(nil)

func newTestExecutor(
	configProvider *testConfigurationProvider,
	impl implementation.AIImplementation,
) *execution.Executor {
	selectionStore := selection.NewMemoryStore()

	selectionService := selection.NewService(
		selectionStore,
	)

	ctx := context.Background()

	if err := selectionService.Select(
		ctx,
		configProvider.configuration.Ref,
	); err != nil {
		panic(err)
	}

	resolver := implementation.NewResolver()

	if err := resolver.Register(
		configProvider.configuration.ImplementationRef,
		impl,
	); err != nil {
		panic("Register() failed: " + err.Error())
	}

	credentialResolver := &testCredentialResolver{}

	return execution.NewExecutor(
		configProvider,
		selectionService,
		resolver,
		credentialResolver,
	)
}

func newTestConfigurationProvider() *testConfigurationProvider {
	return &testConfigurationProvider{
		configuration: configuration.AIConfiguration{
			Name:              "test",
			Ref:               "config_test",
			Type:              "test",
			ImplementationRef: "test.impl",
			CredentialRef:     "test-credential",
			Enabled:           true,
		},
	}
}

func newValidRequest() *aiv1.ExecuteRequest {
	return &aiv1.ExecuteRequest{
		RequestId:  "req-test-001",
		Capability: "text.generate",
		Input: &anypb.Any{
			TypeUrl: "test/input",
			Value:   []byte("hello"),
		},
	}
}

// -----------------------------------------------------------------------------
// Existing transport tests
// -----------------------------------------------------------------------------

func TestServerExecute(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	executor := newTestExecutor(
		configProvider,
		&testImplementation{},
	)

	server := NewServer(executor)

	input := &aiv1.ExecuteRequest{
		RequestId:  "req-step2-001",
		Capability: "text.generate",
		Input: &anypb.Any{
			TypeUrl: "test/input",
			Value:   []byte("hello"),
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"source": structpb.NewStringValue("grpc-test"),
			},
		},
	}

	response, err := server.Execute(
		context.Background(),
		input,
	)

	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if response == nil {
		t.Fatal("response must not be nil")
	}

	if response.GetRequestId() != input.GetRequestId() {
		t.Fatalf(
			"request ID mismatch: got %q, want %q",
			response.GetRequestId(),
			input.GetRequestId(),
		)
	}

	if response.GetCapability() != input.GetCapability() {
		t.Fatalf(
			"capability mismatch: got %q, want %q",
			response.GetCapability(),
			input.GetCapability(),
		)
	}

	if response.GetOutput() == nil {
		t.Fatal("response output must not be nil")
	}

	if response.GetMetadata() == nil {
		t.Fatal("response metadata must not be nil")
	}

	sourceField, ok := response.GetMetadata().
		GetFields()["source"]

	if !ok {
		t.Fatal(
			"response metadata source field must exist",
		)
	}

	if sourceField.GetStringValue() != "test-implementation" {
		t.Fatalf(
			"unexpected response metadata source: got %q",
			sourceField.GetStringValue(),
		)
	}
}

func TestServerRejectsMissingRequestID(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	executor := newTestExecutor(
		configProvider,
		&testImplementation{},
	)

	server := NewServer(executor)

	request := newValidRequest()
	request.RequestId = ""

	_, err := server.Execute(
		context.Background(),
		request,
	)

	if err == nil {
		t.Fatal("expected invalid request error")
	}

	assertGRPCCode(
		t,
		err,
		codes.InvalidArgument,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeInvalidRequest,
	)
}

func TestServerRejectsMissingCapability(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	executor := newTestExecutor(
		configProvider,
		&testImplementation{},
	)

	server := NewServer(executor)

	request := newValidRequest()
	request.Capability = ""

	_, err := server.Execute(
		context.Background(),
		request,
	)

	if err == nil {
		t.Fatal("expected invalid request error")
	}

	assertGRPCCode(
		t,
		err,
		codes.InvalidArgument,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeInvalidRequest,
	)
}

func TestServerRejectsMissingInput(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	executor := newTestExecutor(
		configProvider,
		&testImplementation{},
	)

	server := NewServer(executor)

	request := newValidRequest()
	request.Input = nil

	_, err := server.Execute(
		context.Background(),
		request,
	)

	if err == nil {
		t.Fatal("expected invalid request error")
	}

	assertGRPCCode(
		t,
		err,
		codes.InvalidArgument,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeInvalidRequest,
	)
}

func TestServerRejectsNilRequest(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	executor := newTestExecutor(
		configProvider,
		&testImplementation{},
	)

	server := NewServer(executor)

	_, err := server.Execute(
		context.Background(),
		nil,
	)

	if err == nil {
		t.Fatal("expected invalid request error")
	}

	assertGRPCCode(
		t,
		err,
		codes.InvalidArgument,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeInvalidRequest,
	)
}

func TestServerPreservesAnyInput(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	var received *anypb.Any

	implementationUnderTest := &captureImplementation{
		input: &received,
	}

	executor := newTestExecutor(
		configProvider,
		implementationUnderTest,
	)

	server := NewServer(executor)

	input := &anypb.Any{
		TypeUrl: "pride.test/Input",
		Value:   []byte{1, 2, 3, 4},
	}

	_, err := server.Execute(
		context.Background(),
		&aiv1.ExecuteRequest{
			RequestId:  "req-any-001",
			Capability: "test",
			Input:      input,
		},
	)

	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if received == nil {
		t.Fatal(
			"implementation did not receive input",
		)
	}

	if !proto.Equal(
		received,
		input,
	) {
		t.Fatal(
			"Any input was modified across adapter boundary",
		)
	}
}

func TestServerPreservesRequestMetadata(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	capture := &captureImplementation{}

	executor := newTestExecutor(
		configProvider,
		capture,
	)

	server := NewServer(executor)

	request := &aiv1.ExecuteRequest{
		RequestId:  "req-metadata-001",
		Capability: "test",
		Input: &anypb.Any{
			TypeUrl: "pride.test/Input",
			Value:   []byte("hello"),
		},
		Metadata: &structpb.Struct{
			Fields: map[string]*structpb.Value{
				"source": structpb.NewStringValue("sdk"),
				"retry":  structpb.NewNumberValue(2),
			},
		},
	}

	_, err := server.Execute(
		context.Background(),
		request,
	)

	if err != nil {
		t.Fatalf(
			"Execute() failed: %v",
			err,
		)
	}

	if capture.metadata == nil {
		t.Fatal(
			"implementation did not receive metadata",
		)
	}

	if capture.metadata["source"] != "sdk" {
		t.Fatalf(
			"unexpected source metadata: got %v",
			capture.metadata["source"],
		)
	}

	if capture.metadata["retry"] != float64(2) {
		t.Fatalf(
			"unexpected retry metadata: got %v",
			capture.metadata["retry"],
		)
	}
}

// -----------------------------------------------------------------------------
// R1.3 — Canonical Error Matrix
// -----------------------------------------------------------------------------

func TestServerExecuteAuthenticationError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeAuthenticationError,
		codes.Unauthenticated,
		"provider authentication failed",
	)
}

func TestServerExecutePermissionError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodePermissionError,
		codes.PermissionDenied,
		"provider permission denied",
	)
}

func TestServerExecuteRateLimitError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeRateLimit,
		codes.ResourceExhausted,
		"provider rate limit exceeded",
	)
}

func TestServerExecuteRuntimeError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeRuntimeError,
		codes.Internal,
		"provider execution failed",
	)
}

func TestServerExecuteTimeoutError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeTimeoutError,
		codes.DeadlineExceeded,
		"provider execution timed out",
	)
}

func TestServerExecuteConnectionError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeConnectionError,
		codes.Unavailable,
		"provider connection failed",
	)
}

func TestServerExecuteCapabilityUnsupportedError(t *testing.T) {
	assertServerExecutionError(
		t,
		prideerrors.CodeCapabilityUnsupported,
		codes.Unimplemented,
		"capability is not supported",
	)
}

func TestServerExecuteUnknownError(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	implementationUnderTest := &errorImplementation{
		err: fmt.Errorf(
			"internal provider failure",
		),
	}

	executor := newTestExecutor(
		configProvider,
		implementationUnderTest,
	)

	server := NewServer(executor)

	_, err := server.Execute(
		context.Background(),
		newValidRequest(),
	)

	if err == nil {
		t.Fatal("expected execution error")
	}

	assertGRPCCode(
		t,
		err,
		codes.Unknown,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeUnknownError,
	)

	st := mustStatus(t, err)

	if strings.Contains(
		st.Message(),
		"internal provider failure",
	) {
		t.Fatal(
			"internal error leaked into gRPC response",
		)
	}
}

func TestServerExecuteWrappedCanonicalError(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	canonicalErr := prideerrors.New(
		prideerrors.CodeAuthenticationError,
		"provider authentication failed",
	)

	wrappedErr := fmt.Errorf(
		"implementation execution failed: %w",
		canonicalErr,
	)

	implementationUnderTest := &errorImplementation{
		err: wrappedErr,
	}

	executor := newTestExecutor(
		configProvider,
		implementationUnderTest,
	)

	server := NewServer(executor)

	_, err := server.Execute(
		context.Background(),
		newValidRequest(),
	)

	if err == nil {
		t.Fatal("expected execution error")
	}

	assertGRPCCode(
		t,
		err,
		codes.Unauthenticated,
	)

	assertAIErrorCode(
		t,
		err,
		prideerrors.CodeAuthenticationError,
	)
}

func TestServerExecuteDoesNotLeakCredentialOrProviderError(t *testing.T) {
	configProvider := newTestConfigurationProvider()

	const secret = "SUPER_SECRET_API_KEY_123"

	providerCause := fmt.Errorf(
		"provider returned invalid API key %s",
		secret,
	)

	canonicalErr := prideerrors.New(
		prideerrors.CodeAuthenticationError,
		"provider authentication failed",
	).WithCause(
		providerCause,
	)

	implementationUnderTest := &errorImplementation{
		err: canonicalErr,
	}

	executor := newTestExecutor(
		configProvider,
		implementationUnderTest,
	)

	server := NewServer(executor)

	_, err := server.Execute(
		context.Background(),
		newValidRequest(),
	)

	if err == nil {
		t.Fatal("expected authentication error")
	}

	st := mustStatus(t, err)

	if st.Code() != codes.Unauthenticated {
		t.Fatalf(
			"unexpected gRPC code: got %v, want %v",
			st.Code(),
			codes.Unauthenticated,
		)
	}

	if strings.Contains(
		st.Message(),
		secret,
	) {
		t.Fatal(
			"credential secret leaked into gRPC status message",
		)
	}

	if strings.Contains(
		st.Message(),
		"invalid API key",
	) {
		t.Fatal(
			"provider internal error leaked into gRPC status message",
		)
	}

	aiErr := extractAIError(t, err)

	if aiErr.GetErrorCode() != string(
		prideerrors.CodeAuthenticationError,
	) {
		t.Fatalf(
			"unexpected AI error code: got %q, want %q",
			aiErr.GetErrorCode(),
			string(prideerrors.CodeAuthenticationError),
		)
	}

	if strings.Contains(
		aiErr.GetMessage(),
		secret,
	) {
		t.Fatal(
			"credential secret leaked into AIError message",
		)
	}
}

// -----------------------------------------------------------------------------
// R1.3 — Helpers
// -----------------------------------------------------------------------------

func assertServerExecutionError(
	t *testing.T,
	expectedAIErrorCode prideerrors.Code,
	expectedGRPCCode codes.Code,
	message string,
) {
	t.Helper()

	configProvider := newTestConfigurationProvider()

	implementationUnderTest := &errorImplementation{
		err: prideerrors.New(
			expectedAIErrorCode,
			message,
		),
	}

	executor := newTestExecutor(
		configProvider,
		implementationUnderTest,
	)

	server := NewServer(executor)

	_, err := server.Execute(
		context.Background(),
		newValidRequest(),
	)

	if err == nil {
		t.Fatal("expected execution error")
	}

	assertGRPCCode(
		t,
		err,
		expectedGRPCCode,
	)

	assertAIErrorCode(
		t,
		err,
		expectedAIErrorCode,
	)
}

func assertGRPCCode(
	t *testing.T,
	err error,
	expected codes.Code,
) {
	t.Helper()

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf(
			"expected gRPC status error, got %T: %v",
			err,
			err,
		)
	}

	if st.Code() != expected {
		t.Fatalf(
			"unexpected gRPC code: got %v, want %v",
			st.Code(),
			expected,
		)
	}
}

func assertAIErrorCode(
	t *testing.T,
	err error,
	expected prideerrors.Code,
) {
	t.Helper()

	aiErr := extractAIError(
		t,
		err,
	)

	expectedString := string(expected)

	if aiErr.GetErrorCode() != expectedString {
		t.Fatalf(
			"unexpected AI error code: got %q, want %q",
			aiErr.GetErrorCode(),
			expectedString,
		)
	}
}

func extractAIError(
	t *testing.T,
	err error,
) *aiv1.AIError {
	t.Helper()

	st := mustStatus(
		t,
		err,
	)

	for _, detail := range st.Details() {
		if aiErr, ok := detail.(*aiv1.AIError); ok {
			return aiErr
		}
	}

	t.Fatalf(
		"AIError detail not found in gRPC status",
	)

	return nil
}

func mustStatus(
	t *testing.T,
	err error,
) *status.Status {
	t.Helper()

	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf(
			"expected gRPC status error, got %T: %v",
			err,
			err,
		)
	}

	return st
}

// -----------------------------------------------------------------------------
// Test implementations
// -----------------------------------------------------------------------------

type errorImplementation struct {
	err error
}

func (i *errorImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	return executioncontract.AIExecutionResult{}, i.err
}

var _ implementation.AIImplementation = (*errorImplementation)(nil)

type captureImplementation struct {
	input    **anypb.Any
	metadata map[string]any
}

func (i *captureImplementation) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
	executionContext executioncontract.AIExecutionContext,
) (
	executioncontract.AIExecutionResult,
	error,
) {
	if i.input != nil {
		*i.input = request.Input
	}

	i.metadata = request.Metadata

	return executioncontract.AIExecutionResult{
		RequestID:  request.RequestID,
		Capability: request.Capability,
		Output:     request.Input,
	}, nil
}

var _ implementation.AIImplementation = (*captureImplementation)(nil)

// R5.5 — Credentials must not be implicitly propagated into response metadata.
func TestServerDoesNotPropagateCredentialIntoResponseMetadata(t *testing.T) {
	configProvider := newTestConfigurationProvider()
	executor := newTestExecutor(configProvider, &testImplementation{})
	server := NewServer(executor)

	response, err := server.Execute(context.Background(), newValidRequest())
	if err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
	if response == nil {
		t.Fatal("response must not be nil")
	}

	const credentialSentinel = "test-api-key"
	metadataText := fmt.Sprint(response.GetMetadata().AsMap())
	if strings.Contains(metadataText, credentialSentinel) {
		t.Fatal("credential secret was propagated into response metadata")
	}
}
