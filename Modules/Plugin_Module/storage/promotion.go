package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var ErrArtifactAlreadyExists = errors.New("installed Plugin version already exists")

// Promote atomically moves a complete operation workspace into its immutable
// installed-version directory. Staging and destination are both beneath the
// same Plugin_Storage root, so the rename cannot cross filesystems.
//
// The installation repository's active-operation uniqueness constraint
// serializes PRIDE-managed promotions for a Plugin. The destination is checked
// immediately before rename and is never intentionally replaced.
func (a *Adapter) Promote(ctx context.Context, stagingPath, finalPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateOperationWorkspace(stagingPath); err != nil {
		return err
	}
	if err := a.validateFinalArtifactPath(finalPath); err != nil {
		return err
	}

	artifactInfo, err := a.root.Lstat(filepath.FromSlash(path.Join(stagingPath, OperationArtifactFileName)))
	if err != nil || !artifactInfo.Mode().IsRegular() {
		return fmt.Errorf("%w: staged artifact is missing or not a regular file", ErrInvalidOperationWorkspace)
	}
	packageInfo, err := a.root.Lstat(filepath.FromSlash(path.Join(stagingPath, "package")))
	if err != nil || !packageInfo.IsDir() || packageInfo.Mode()&fs.ModeSymlink != 0 {
		return fmt.Errorf("%w: extracted package directory is missing or unsafe", ErrInvalidOperationWorkspace)
	}

	parent := path.Dir(finalPath)
	if err := a.ensureInstalledPluginDirectory(parent); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := a.root.Lstat(filepath.FromSlash(finalPath)); err == nil {
		return ErrArtifactAlreadyExists
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect installed Plugin destination: %w", err)
	}

	if err := a.root.Rename(filepath.FromSlash(stagingPath), filepath.FromSlash(finalPath)); err != nil {
		if errors.Is(err, os.ErrExist) || errors.Is(err, fs.ErrExist) {
			return ErrArtifactAlreadyExists
		}
		return fmt.Errorf("atomically promote Plugin workspace: %w", err)
	}
	return nil
}

func (a *Adapter) validateFinalArtifactPath(value string) error {
	if a == nil || a.layout == nil || a.root == nil ||
		value == "" || strings.Contains(value, "\\") || path.Clean(value) != value {
		return ErrInvalidPath
	}
	parts := strings.Split(value, "/")
	if len(parts) != 3 || parts[0] != "installed" {
		return ErrInvalidPath
	}
	expected, err := a.layout.RelativeInstalledVersionPath(parts[1], parts[2])
	if err != nil || expected != value {
		return ErrInvalidPath
	}
	if _, err := a.layout.ResolvePluginStoragePath(value); err != nil {
		return err
	}
	return nil
}

func (a *Adapter) ensureInstalledPluginDirectory(relativePath string) error {
	if relativePath == "" || !strings.HasPrefix(relativePath, "installed/") {
		return ErrInvalidPath
	}
	installedInfo, err := a.root.Lstat("installed")
	if err != nil || !installedInfo.IsDir() || installedInfo.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	info, err := a.root.Lstat(filepath.FromSlash(relativePath))
	if errors.Is(err, os.ErrNotExist) {
		if err := a.root.Mkdir(filepath.FromSlash(relativePath), 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create installed Plugin parent: %w", err)
		}
		info, err = a.root.Lstat(filepath.FromSlash(relativePath))
	}
	if err != nil {
		return fmt.Errorf("inspect installed Plugin parent: %w", err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	return nil
}
