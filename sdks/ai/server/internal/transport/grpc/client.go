package grpc

import (
	"context"
	"errors"
	"fmt"

	aiv1 "pride-ai-sdk/proto"
	executioncontract "pride-ai-sdk/server/internal/contracts/execution"
	modulecontract "pride-ai-sdk/server/internal/contracts/module"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

var (
	ErrNilGRPCClient = errors.New("gRPC AI service client is nil")
)

// Compile-time assertion.
//
// GRPCAIClient must implement the logical AI Module boundary.
var _ modulecontract.AIClient = (*GRPCAIClient)(nil)

// GRPCAIClient is the concrete gRPC implementation of module.AIClient.
//
// It is responsible only for translating between:
//
//	executioncontract.AIExecutionRequest
//		↕
//	aiv1.ExecuteRequest
//
// and:
//
//	executioncontract.AIExecutionResult
//		↕
//	aiv1.ExecuteResponse
//
// It does not perform:
//   - configuration resolution
//   - credential resolution
//   - implementation resolution
//   - provider selection
//   - runtime execution
type GRPCAIClient struct {
	client aiv1.AIServiceClient
}

// NewGRPCAIClient creates a new gRPC-backed AI Module client.
//
// The generated gRPC client is injected so connection lifecycle remains
// outside this transport implementation.
func NewGRPCAIClient(client aiv1.AIServiceClient) *GRPCAIClient {
	return &GRPCAIClient{
		client: client,
	}
}

// Execute forwards a logical AI execution request to the AI Module
// through the gRPC contract.
func (c *GRPCAIClient) Execute(
	ctx context.Context,
	request executioncontract.AIExecutionRequest,
) (executioncontract.AIExecutionResult, error) {
	if c == nil || c.client == nil {
		return executioncontract.AIExecutionResult{}, ErrNilGRPCClient
	}

	grpcRequest, err := toProtoRequest(request)
	if err != nil {
		return executioncontract.AIExecutionResult{}, err
	}

	grpcResponse, err := c.client.Execute(ctx, grpcRequest)
	if err != nil {
		return executioncontract.AIExecutionResult{}, fromGRPCError(request.RequestID, err)
	}

	return fromProtoResponse(grpcResponse)
}

// toProtoRequest converts the logical SDK execution contract into
// the gRPC transport contract.
func toProtoRequest(
	request executioncontract.AIExecutionRequest,
) (*aiv1.ExecuteRequest, error) {
	if request.RequestID == "" {
		return nil, newInvalidRequestError(
			request.RequestID,
			"request ID must not be empty",
		)
	}

	if request.Capability == "" {
		return nil, newInvalidRequestError(
			request.RequestID,
			"capability must not be empty",
		)
	}

	if request.Input == nil {
		return nil, newInvalidRequestError(
			request.RequestID,
			"input must not be nil",
		)
	}

	metadata, err := structpb.NewStruct(request.Metadata)
	if err != nil {
		return nil, newInvalidRequestError(
			request.RequestID,
			fmt.Sprintf("invalid metadata: %v", err),
		)
	}

	return &aiv1.ExecuteRequest{
		RequestId:  request.RequestID,
		Capability: request.Capability,
		Input:      request.Input,
		Metadata:   metadata,
	}, nil
}

// fromProtoResponse converts the gRPC response into the logical
// SDK execution result contract.
func fromProtoResponse(
	response *aiv1.ExecuteResponse,
) (executioncontract.AIExecutionResult, error) {
	if response == nil {
		return executioncontract.AIExecutionResult{}, errors.New(
			"AI Module returned an empty response",
		)
	}

	if response.GetRequestId() == "" {
		return executioncontract.AIExecutionResult{}, errors.New(
			"AI Module returned an empty request ID",
		)
	}

	if response.GetCapability() == "" {
		return executioncontract.AIExecutionResult{}, errors.New(
			"AI Module returned an empty capability",
		)
	}

	if response.GetOutput() == nil {
		return executioncontract.AIExecutionResult{}, errors.New(
			"AI Module returned an empty output",
		)
	}

	var metadata map[string]any

	if response.GetMetadata() != nil {
		metadata = response.GetMetadata().AsMap()
	}

	return executioncontract.AIExecutionResult{
		RequestID:  response.GetRequestId(),
		Capability: response.GetCapability(),
		Output:     response.GetOutput(),
		Metadata:   metadata,
	}, nil
}

// fromGRPCError converts a gRPC failure into a PRIDE semantic error.
//
// The gRPC status remains the transport-level semantic.
// AIError, when supplied by the AI Module, remains the PRIDE-level
// semantic error.
func fromGRPCError(requestID string, err error) error {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return &RemoteError{
			RequestID: requestID,
			ErrorCode: "UNKNOWN_ERROR",
			Message:   err.Error(),
			GRPCCode:  codes.Unknown,
		}
	}

	remoteError := &RemoteError{
		RequestID: requestID,
		ErrorCode: mapGRPCCode(st.Code()),
		Message:   st.Message(),
		GRPCCode:  st.Code(),
	}

	for _, detail := range st.Details() {
		aiError, ok := detail.(*aiv1.AIError)
		if !ok {
			continue
		}

		if aiError.GetRequestId() != "" {
			remoteError.RequestID = aiError.GetRequestId()
		}

		if aiError.GetErrorCode() != "" {
			remoteError.ErrorCode = aiError.GetErrorCode()
		}

		if aiError.GetMessage() != "" {
			remoteError.Message = aiError.GetMessage()
		}

		if aiError.GetMetadata() != nil {
			remoteError.Metadata = aiError.GetMetadata().AsMap()
		}

		break
	}

	return remoteError
}

// mapGRPCCode maps gRPC transport semantics into the default
// PRIDE semantic error category.
//
// If the AI Module provides AIError details, those details take
// precedence over this fallback mapping.
func mapGRPCCode(code codes.Code) string {
	switch code {
	case codes.InvalidArgument:
		return "INVALID_REQUEST"

	case codes.Unauthenticated:
		return "AUTHENTICATION_ERROR"

	case codes.PermissionDenied:
		return "PERMISSION_ERROR"

	case codes.Unavailable:
		return "CONNECTION_ERROR"

	case codes.DeadlineExceeded:
		return "TIMEOUT_ERROR"

	case codes.ResourceExhausted:
		return "RATE_LIMIT"

	case codes.Internal:
		return "RUNTIME_ERROR"

	default:
		return "UNKNOWN_ERROR"
	}
}

// RemoteError represents a standardized error returned by the
// AI Module through the gRPC boundary.
type RemoteError struct {
	RequestID string
	ErrorCode string
	Message   string
	Metadata  map[string]any
	GRPCCode  codes.Code
}

func (e *RemoteError) Error() string {
	if e == nil {
		return "unknown AI Module error"
	}

	if e.ErrorCode == "" {
		return e.Message
	}

	if e.Message == "" {
		return e.ErrorCode
	}

	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

func newInvalidRequestError(requestID, message string) error {
	return &RemoteError{
		RequestID: requestID,
		ErrorCode: "INVALID_REQUEST",
		Message:   message,
		GRPCCode:  codes.InvalidArgument,
	}
}
