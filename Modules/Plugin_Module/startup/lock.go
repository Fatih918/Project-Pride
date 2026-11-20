package startup

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrInvalidRecoveryLockPath = errors.New("invalid Plugin recovery lock path")
	ErrRecoveryLockUnsupported = errors.New("cross-process Plugin recovery locking is unsupported on this platform")
)

const recoveryLockPollInterval = 25 * time.Millisecond

// RecoveryLock holds an operating-system advisory lock for the lifetime of the
// returned value. The lock is released when Close is called or the process exits.
type RecoveryLock struct {
	file       *os.File
	closeOnce  sync.Once
	closeError error
}

// AcquireRecoveryLock waits until the process exclusively owns lockPath.
// lockPath must be an absolute path inside a private, local Plugin data directory.
// The lock file is persistent and must never be removed while the application is
// running; deleting it could let processes lock different inodes.
func AcquireRecoveryLock(ctx context.Context, lockPath string) (*RecoveryLock, error) {
	if ctx == nil || lockPath == "" || !filepath.IsAbs(lockPath) {
		return nil, ErrInvalidRecoveryLockPath
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	lockPath = filepath.Clean(lockPath)
	parent := filepath.Dir(lockPath)
	parentInfo, err := os.Lstat(parent)
	if err != nil {
		return nil, fmt.Errorf("inspect Plugin recovery lock directory: %w", err)
	}
	if !parentInfo.IsDir() || parentInfo.Mode()&os.ModeSymlink != 0 {
		return nil, ErrInvalidRecoveryLockPath
	}
	if info, statErr := os.Lstat(lockPath); statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, ErrInvalidRecoveryLockPath
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, fmt.Errorf("inspect Plugin recovery lock file: %w", statErr)
	}

	file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open Plugin recovery lock file: %w", err)
	}
	fail := func(cause error) (*RecoveryLock, error) {
		_ = file.Close()
		return nil, cause
	}
	info, err := file.Stat()
	if err != nil {
		return fail(fmt.Errorf("inspect opened Plugin recovery lock file: %w", err))
	}
	if !info.Mode().IsRegular() {
		return fail(ErrInvalidRecoveryLockPath)
	}
	if runtimeNeedsPrivateMode() {
		if err := file.Chmod(0o600); err != nil {
			return fail(fmt.Errorf("secure Plugin recovery lock file: %w", err))
		}
	}

	ticker := time.NewTicker(recoveryLockPollInterval)
	defer ticker.Stop()
	for {
		acquired, lockErr := tryAcquirePlatformLock(file)
		if lockErr != nil {
			return fail(fmt.Errorf("acquire Plugin recovery lock: %w", lockErr))
		}
		if acquired {
			if err := ctx.Err(); err != nil {
				_ = releasePlatformLock(file)
				return fail(err)
			}
			return &RecoveryLock{file: file}, nil
		}
		select {
		case <-ctx.Done():
			return fail(ctx.Err())
		case <-ticker.C:
		}
	}
}

// Close releases the operating-system lock and closes its file handle.
// It is safe to call more than once.
func (l *RecoveryLock) Close() error {
	if l == nil {
		return nil
	}
	l.closeOnce.Do(func() {
		if l.file == nil {
			return
		}
		unlockErr := releasePlatformLock(l.file)
		closeErr := l.file.Close()
		l.closeError = errors.Join(unlockErr, closeErr)
	})
	return l.closeError
}
