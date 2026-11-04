package registry

import "errors"

var (
	ErrNotFound      = errors.New("plugin not found")
	ErrAlreadyExists = errors.New("plugin already exists")
	ErrInvalidPlugin = errors.New("invalid plugin record")
)
