package storage

import "path/filepath"

// StartupRecoveryLockPath returns the persistent lock-file path used to
// serialize Plugin startup recovery for this Plugin_Storage root. The lock file
// must never be removed while any PRIDE process may use this storage root.
func (l *Layout) StartupRecoveryLockPath() string {
	return filepath.Join(l.PluginStorageRoot(), "startup-recovery.lock")
}
