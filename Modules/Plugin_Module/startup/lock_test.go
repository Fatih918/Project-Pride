//go:build linux || darwin || windows

package startup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestAcquireRecoveryLockRejectsInvalidPaths(t *testing.T) {
	dir := t.TempDir()
	link := filepath.Join(dir, "link")
	if err := os.Symlink(dir, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := AcquireRecoveryLock(context.Background(), filepath.Join(link, "lock")); !errors.Is(err, ErrInvalidRecoveryLockPath) {
		t.Fatalf("expected invalid parent path error, got %v", err)
	}
	target := filepath.Join(dir, "target")
	if err := os.WriteFile(target, []byte("do not lock through a symlink"), 0o600); err != nil {
		t.Fatal(err)
	}
	lockLink := filepath.Join(dir, "lock-link")
	if err := os.Symlink(target, lockLink); err != nil {
		t.Skipf("file symlinks unavailable: %v", err)
	}
	if _, err := AcquireRecoveryLock(context.Background(), lockLink); !errors.Is(err, ErrInvalidRecoveryLockPath) {
		t.Fatalf("expected invalid lock target error, got %v", err)
	}
	if _, err := AcquireRecoveryLock(context.Background(), "relative.lock"); !errors.Is(err, ErrInvalidRecoveryLockPath) {
		t.Fatalf("expected absolute path validation error, got %v", err)
	}
}

func TestAcquireRecoveryLockHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := AcquireRecoveryLock(ctx, filepath.Join(t.TempDir(), "lock"))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}

func TestRecoveryLockIsExclusiveAcrossProcessesAndReleasedOnCrash(t *testing.T) {
	if os.Getenv("PRIDE_RECOVERY_LOCK_HELPER") == "1" {
		recoveryLockHelperProcess(t)
		return
	}

	dir := t.TempDir()
	lockPath := filepath.Join(dir, "startup-recovery.lock")
	readyPath := filepath.Join(dir, "ready")
	releasePath := filepath.Join(dir, "release")

	cmd := exec.Command(os.Args[0], "-test.run=^TestRecoveryLockIsExclusiveAcrossProcessesAndReleasedOnCrash$")
	cmd.Env = append(os.Environ(),
		"PRIDE_RECOVERY_LOCK_HELPER=1",
		"PRIDE_RECOVERY_LOCK_PATH="+lockPath,
		"PRIDE_RECOVERY_LOCK_READY="+readyPath,
		"PRIDE_RECOVERY_LOCK_RELEASE="+releasePath,
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("start lock helper process: %v", err)
	}
	defer func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = cmd.Wait()
	}()

	waitForFile(t, readyPath, 5*time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 125*time.Millisecond)
	defer cancel()
	if _, err := AcquireRecoveryLock(ctx, lockPath); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("second process should wait for lock, got %v", err)
	}

	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("terminate lock holder: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		// A killed helper is expected.
	}
	cmd.Process = nil

	acquireCtx, acquireCancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer acquireCancel()
	lock, err := AcquireRecoveryLock(acquireCtx, lockPath)
	if err != nil {
		t.Fatalf("lock should be released after holder process exits: %v", err)
	}
	if err := lock.Close(); err != nil {
		t.Fatalf("release recovery lock: %v", err)
	}
}

func recoveryLockHelperProcess(t *testing.T) {
	t.Helper()
	lock, err := AcquireRecoveryLock(context.Background(), os.Getenv("PRIDE_RECOVERY_LOCK_PATH"))
	if err != nil {
		t.Fatalf("helper acquire lock: %v", err)
	}
	defer lock.Close()
	if err := os.WriteFile(os.Getenv("PRIDE_RECOVERY_LOCK_READY"), []byte("ready"), 0o600); err != nil {
		t.Fatalf("write ready marker: %v", err)
	}
	release := os.Getenv("PRIDE_RECOVERY_LOCK_RELEASE")
	deadline := time.Now().Add(10 * time.Second)
	for {
		if _, err := os.Stat(release); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("helper timed out waiting for release marker")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func waitForFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		if _, err := os.Stat(path); err == nil {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", path)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
