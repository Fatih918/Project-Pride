package lifecycle

import "errors"

var (
	ErrOperationNotFound        = errors.New("lifecycle operation not found")
	ErrOperationIDConflict      = errors.New("lifecycle operation ID conflict")
	ErrOperationInProgress      = errors.New("lifecycle operation in progress")
	ErrPluginOperationConflict  = errors.New("plugin lifecycle operation conflict")
	ErrPluginNotFound           = errors.New("plugin not found")
	ErrVersionNotFound          = errors.New("selected plugin version not found")
	ErrVersionNotSelected       = errors.New("plugin version is not selected")
	ErrAuthorizationDenied      = errors.New("lifecycle authorization denied")
	ErrAuthorizationUnavailable = errors.New("lifecycle authorization unavailable")
	ErrLeaseLost                = errors.New("lifecycle operation lease lost")
	ErrInvalidTransition        = errors.New("invalid lifecycle transition")
	ErrPersistence              = errors.New("lifecycle persistence failure")
)
