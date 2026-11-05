package installation

import "errors"

var (
	ErrInvalidRecord       = errors.New("invalid installation record")
	ErrOperationNotFound   = errors.New("installation operation not found")
	ErrVersionNotFound     = errors.New("installed version not found")
	ErrOperationConflict   = errors.New("installation operation conflict")
	ErrInvalidTransition   = errors.New("invalid installation state transition")
	ErrVersionAlreadyExist = errors.New("installed version already exists")
	ErrRecordMismatch      = errors.New("installation record mismatch")
	ErrPersistence         = errors.New("installation persistence failure")
)
