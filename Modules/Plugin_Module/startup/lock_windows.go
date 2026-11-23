//go:build windows

package startup

import (
	"errors"
	"os"

	"golang.org/x/sys/windows"
)

func openPlatformLockFile(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, ErrInvalidRecoveryLockPath
	}
	handle, err := windows.CreateFile(
		name,
		windows.GENERIC_READ|windows.GENERIC_WRITE,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE,
		nil,
		windows.OPEN_ALWAYS,
		windows.FILE_ATTRIBUTE_NORMAL|windows.FILE_FLAG_OPEN_REPARSE_POINT,
		0,
	)
	if err != nil {
		return nil, err
	}
	fail := func(cause error) (*os.File, error) {
		_ = windows.CloseHandle(handle)
		return nil, cause
	}
	var info windows.ByHandleFileInformation
	if err := windows.GetFileInformationByHandle(handle, &info); err != nil {
		return fail(err)
	}
	if info.FileAttributes&(windows.FILE_ATTRIBUTE_REPARSE_POINT|windows.FILE_ATTRIBUTE_DIRECTORY) != 0 ||
		windows.GetFileType(handle) != windows.FILE_TYPE_DISK {
		return fail(ErrInvalidRecoveryLockPath)
	}
	return os.NewFile(uintptr(handle), path), nil
}

func tryAcquirePlatformLock(file *os.File) (bool, error) {
	overlapped := &windows.Overlapped{}
	err := windows.LockFileEx(
		windows.Handle(file.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0,
		1,
		0,
		overlapped,
	)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, windows.ERROR_LOCK_VIOLATION) {
		return false, nil
	}
	return false, err
}

func releasePlatformLock(file *os.File) error {
	return windows.UnlockFileEx(windows.Handle(file.Fd()), 0, 1, 0, &windows.Overlapped{})
}

func runtimeNeedsPrivateMode() bool { return false }
