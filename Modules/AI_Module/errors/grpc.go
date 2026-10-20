package errors

import (
	"errors"

	aiv1 "pride-ai-module/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

// GRPCStatus converts an internal PRIDE AI error into a gRPC status.
//
// The returned status contains:
//   - an appropriate gRPC transport code
//   - AIError as structured protobuf detail
//
// Secrets and underlying internal error messages are intentionally
// not exposed through AIError.
func GRPCStatus(err error) error {
	if err == nil {
		return nil
	}

	prideErr := normalize(err)

	grpcCode := mapGRPCCode(prideErr.Code)

	detail := &aiv1.AIError{
		RequestId: prideErr.RequestID,
		ErrorCode: string(prideErr.Code),
		Message:   prideErr.Message,
	}

	if len(prideErr.Metadata) > 0 {
		metadata, err := structpb.NewStruct(prideErr.Metadata)
		if err == nil {
			detail.Metadata = metadata
		}
	}

	st, err := status.New(
		grpcCode,
		prideErr.Message,
	).WithDetails(detail)

	if err != nil {
		return status.Error(
			grpcCode,
			prideErr.Message,
		)
	}

	return st.Err()
}

// FromGRPCStatus extracts the standardized AIError from a gRPC error.
//
// If the error does not contain an AIError detail, ok is false.
func FromGRPCStatus(
	err error,
) (*Error, bool) {
	if err == nil {
		return nil, false
	}

	st, ok := status.FromError(err)
	if !ok {
		return nil, false
	}

	for _, detail := range st.Details() {
		aiErr, ok := detail.(*aiv1.AIError)
		if !ok {
			continue
		}

		result := &Error{
			RequestID: aiErr.GetRequestId(),
			Code:      Code(aiErr.GetErrorCode()),
			Message:   aiErr.GetMessage(),
		}

		if metadata := aiErr.GetMetadata(); metadata != nil {
			result.Metadata = metadata.AsMap()
		}

		return result, true
	}

	return nil, false
}

func normalize(err error) *Error {
	var prideErr *Error

	if errors.As(err, &prideErr) && prideErr != nil {
		return prideErr
	}

	return &Error{
		Code:    CodeUnknownError,
		Message: "AI execution failed",
		Cause:   err,
	}
}

func mapGRPCCode(code Code) codes.Code {
	switch code {
	case CodeInvalidRequest:
		return codes.InvalidArgument

	case CodeConfigurationError:
		return codes.FailedPrecondition

	case CodeAuthenticationError:
		return codes.Unauthenticated

	case CodePermissionError:
		return codes.PermissionDenied

	case CodeConnectionError:
		return codes.Unavailable

	case CodeTimeoutError:
		return codes.DeadlineExceeded

	case CodeCapabilityUnsupported:
		return codes.Unimplemented

	case CodeRateLimit:
		return codes.ResourceExhausted

	case CodeRuntimeError:
		return codes.Internal

	case CodeUnknownError:
		return codes.Unknown

	default:
		return codes.Unknown
	}
}

var _ error = (*Error)(nil)
