package errors

import "fmt"

// Error represents a standardized PRIDE AI error.
//
// This type is transport-independent.
// It does not know anything about gRPC.
type Error struct {
	RequestID string
	Code      Code
	Message   string
	Metadata  map[string]any

	Cause error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}

	if e.Cause != nil {
		return fmt.Sprintf(
			"%s: %s: %v",
			e.Code,
			e.Message,
			e.Cause,
		)
	}

	return fmt.Sprintf(
		"%s: %s",
		e.Code,
		e.Message,
	)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.Cause
}

// New creates a standardized PRIDE AI error.
func New(
	code Code,
	message string,
) *Error {
	return &Error{
		Code:    code,
		Message: message,
	}
}

// WithRequestID attaches the originating request ID.
func (e *Error) WithRequestID(
	requestID string,
) *Error {
	if e == nil {
		return nil
	}

	e.RequestID = requestID

	return e
}

// WithMetadata attaches standardized metadata.
func (e *Error) WithMetadata(
	metadata map[string]any,
) *Error {
	if e == nil {
		return nil
	}

	e.Metadata = metadata

	return e
}

// WithCause attaches the underlying internal error.
func (e *Error) WithCause(
	cause error,
) *Error {
	if e == nil {
		return nil
	}

	e.Cause = cause

	return e
}
