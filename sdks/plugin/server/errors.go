package server

import "errors"

// ErrorCode is a stable Module-to-SDK failure category.
type ErrorCode string

const (
	CodeInvalidRequest ErrorCode = "INVALID_REQUEST"
	CodeInvalidInput ErrorCode = "INVALID_INPUT"
	CodePluginNotFound ErrorCode = "PLUGIN_NOT_FOUND"
	CodeCapabilityNotFound ErrorCode = "CAPABILITY_NOT_FOUND"
	CodePluginUnavailable ErrorCode = "PLUGIN_UNAVAILABLE"
	CodePermissionDenied ErrorCode = "PERMISSION_DENIED"
	CodeInternal ErrorCode = "INTERNAL_ERROR"
)

// DomainError lets a Module-facing implementation report a stable category
// without importing gRPC or generated SDK protocol types.
type DomainError struct {
	Code ErrorCode
	Message string
	Cause error
}

func (e *DomainError) Error() string {
	if e == nil { return "" }
	if e.Message != "" { return e.Message }
	if e.Cause != nil { return e.Cause.Error() }
	return string(e.Code)
}

func (e *DomainError) Unwrap() error { if e == nil { return nil }; return e.Cause }

var ErrNotImplemented = errors.New("discovery operation not implemented")
