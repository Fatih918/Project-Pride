package server

import "errors"

// ErrorCode is a stable Module-to-SDK failure category.
type ErrorCode string

const (
	CodeInvalidRequest      ErrorCode = "INVALID_REQUEST"
	CodeInvalidInput        ErrorCode = "INVALID_INPUT"
	CodePluginNotFound      ErrorCode = "PLUGIN_NOT_FOUND"
	CodePluginUnavailable   ErrorCode = "PLUGIN_UNAVAILABLE"
	CodePluginDisabled      ErrorCode = "PLUGIN_DISABLED"
	CodePluginNotReady      ErrorCode = "PLUGIN_NOT_READY"
	CodeIncompatiblePlugin  ErrorCode = "INCOMPATIBLE_PLUGIN"
	CodeCapabilityNotFound  ErrorCode = "CAPABILITY_NOT_FOUND"
	CodePermissionDenied    ErrorCode = "PERMISSION_DENIED"
	CodeLifecycleError      ErrorCode = "LIFECYCLE_ERROR"
	CodeExecutionError      ErrorCode = "EXECUTION_ERROR"
	CodeTimeout             ErrorCode = "TIMEOUT"
	CodeInternal            ErrorCode = "INTERNAL_ERROR"
	CodeUnknown             ErrorCode = "UNKNOWN_ERROR"
)

// DomainError lets a Module-facing implementation report a stable category
// without importing gRPC or generated SDK protocol types.
type DomainError struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *DomainError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	if e.Cause != nil {
		return e.Cause.Error()
	}
	return string(e.Code)
}

func (e *DomainError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

var ErrNotImplemented = errors.New("discovery operation not implemented")
