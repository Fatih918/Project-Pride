package capability

import "errors"

var (
	ErrInvalidIdentifier       = errors.New("invalid capability identifier")
	ErrReservedNamespace       = errors.New("capability namespace is reserved")
	ErrInvalidRuntimeProfileID = errors.New("invalid runtime profile identifier")
	ErrInvalidVersion          = errors.New("invalid semantic version")
	ErrInvalidVersionRequirement = errors.New("invalid semantic version requirement")
)
