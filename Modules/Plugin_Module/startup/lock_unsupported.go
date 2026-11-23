//go:build !windows && !linux && !darwin

package startup

import "os"

func openPlatformLockFile(string) (*os.File, error) {\n\treturn nil, ErrRecoveryLockUnsupported\n}\n\nfunc tryAcquirePlatformLock(*os.File) (bool, error) {
	return false, ErrRecoveryLockUnsupported
}

func releasePlatformLock(*os.File) error { return ErrRecoveryLockUnsupported }

func runtimeNeedsPrivateMode() bool { return false }
