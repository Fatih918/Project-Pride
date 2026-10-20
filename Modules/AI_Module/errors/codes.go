package errors

// Code represents a standardized PRIDE AI execution error category.
type Code string

const (
	CodeConfigurationError    Code = "CONFIGURATION_ERROR"
	CodeAuthenticationError   Code = "AUTHENTICATION_ERROR"
	CodePermissionError       Code = "PERMISSION_ERROR"
	CodeConnectionError       Code = "CONNECTION_ERROR"
	CodeTimeoutError          Code = "TIMEOUT_ERROR"
	CodeCapabilityUnsupported Code = "CAPABILITY_UNSUPPORTED"
	CodeRateLimit             Code = "RATE_LIMIT"
	CodeRuntimeError          Code = "RUNTIME_ERROR"
	CodeInvalidRequest        Code = "INVALID_REQUEST"
	CodeUnknownError          Code = "UNKNOWN_ERROR"
)
