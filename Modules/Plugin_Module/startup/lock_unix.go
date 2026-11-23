//go:build linux || darwin

package startup

import (
	"errors"
	"os"

	"golang.org/x/sys/unix"
)

func openPlatformLockFile(path string) (*os.File, error) {
	fd, err := unix.Open(path, unix.O_CREAT|unix.O_RDWR|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0o600)
	if err != nil {
		if errors.Is(err, unix.ELOOP) {
			return nil, ErrInvalidRecoveryLockPath
		}
		return nil, err
	}
	return os.NewFile(uintptr(fd), path), nil
}

func tryAcquirePlatformLock(file *os.File) (bool, error) {
	err := unix.Flock(int(file.Fd()), unix.LOCK_EX|unix.LOCK_NB)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, unix.EWOULDBLOCK) || errors.Is(err, unix.EAGAIN) {
		return false, nil
	}
	return false, err
}

func releasePlatformLock(file *os.File) error {
	return unix.Flock(int(file.Fd()), unix.LOCK_UN)
}

func runtimeNeedsPrivateMode() bool { return true }
