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

var ErrInvalidExtractionPath = errors.New("invalid Plugin extraction path")

// CreateExtractionRoot creates a new package directory in an operation workspace.
func (a *Adapter) CreateExtractionRoot(ctx context.Context, workspace string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateOperationWorkspace(workspace); err != nil {
		return err
	}
	if err := a.root.Mkdir(filepath.FromSlash(path.Join(workspace, "package")), 0o700); err != nil {
		return fmt.Errorf("create fresh extraction root: %w", err)
	}
	return nil
}

// CreateExtractionDirectory creates a nested directory below the package root.
// Existing components are accepted only when they are real directories.
func (a *Adapter) CreateExtractionDirectory(ctx context.Context, workspace, relativePath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateOperationWorkspace(workspace); err != nil {
		return err
	}
	if err := validateExtractionRelativePath(relativePath); err != nil {
		return err
	}
	if !strings.HasPrefix(relativePath, "package/") {
		return ErrInvalidExtractionPath
	}
	packageRoot := path.Join(workspace, "package")
	rootInfo, err := a.root.Lstat(filepath.FromSlash(packageRoot))
	if err != nil || !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidExtractionPath
	}
	parts := strings.Split(strings.TrimPrefix(relativePath, "package/"), "/")
	current := packageRoot
	for _, part := range parts {
		if err := ctx.Err(); err != nil {
			return err
		}
		current = path.Join(current, part)
		info, err := a.root.Lstat(filepath.FromSlash(current))
		if err == nil {
			if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
				return ErrInvalidExtractionPath
			}
			continue
		}
		if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect extraction directory: %w", err)
		}
		if err := a.root.Mkdir(filepath.FromSlash(current), 0o700); err != nil && !errors.Is(err, os.ErrExist) {
			return fmt.Errorf("create extraction directory: %w", err)
		}
		info, err = a.root.Lstat(filepath.FromSlash(current))
		if err != nil || !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return ErrInvalidExtractionPath
		}
	}
	return nil
}

// CreateExtractionFile exclusively creates a private regular file below the
// operation workspace. Parent directories are created through the confined root.
func (a *Adapter) CreateExtractionFile(ctx context.Context, workspace, relativePath string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := a.validateOperationWorkspace(workspace); err != nil {
		return nil, err
	}
	if err := validateExtractionRelativePath(relativePath); err != nil {
		return nil, err
	}
	packageRoot := path.Join(workspace, "package")
	rootInfo, rootErr := a.root.Lstat(filepath.FromSlash(packageRoot))
	if rootErr != nil || !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, ErrInvalidExtractionPath
	}
	parent := path.Dir(relativePath)
	if parent != "." && parent != "package" {
		if err := a.CreateExtractionDirectory(ctx, workspace, parent); err != nil {
			return nil, err
		}
	}
	target := path.Join(workspace, relativePath)
	file, err := a.root.OpenFile(filepath.FromSlash(target), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create extracted Plugin file: %w", err)
	}
	return file, nil
}

// RemoveExtractionDirectory can remove only the package subtree, never the
// artifact or operation workspace.
func (a *Adapter) RemoveExtractionDirectory(ctx context.Context, workspace, relativePath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateOperationWorkspace(workspace); err != nil {
		return err
	}
	if relativePath != "package" {
		return ErrInvalidExtractionPath
	}
	return a.root.RemoveAll(filepath.FromSlash(path.Join(workspace, relativePath)))
}

func validateExtractionRelativePath(value string) error {
	if value == "" || !strings.HasPrefix(value, "package/") || strings.Contains(value, "\\") || strings.ContainsRune(value, 0) ||
		strings.HasPrefix(value, "/") || path.Clean(value) != value { return ErrInvalidExtractionPath }
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return ErrInvalidExtractionPath
		}
	}
	return nil
}
