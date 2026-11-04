package discovery

import "errors"

var (
	ErrInvalidQuery     = errors.New("invalid discovery query")
	ErrPluginNotFound   = errors.New("plugin not found")
	ErrInvalidPageToken = errors.New("invalid or stale page token")
)
