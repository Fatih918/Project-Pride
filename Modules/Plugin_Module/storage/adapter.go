package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

const OperationArtifactFileName = "artifact.zip"

var ErrInvalidOperationWorkspace = errors.New("invalid plugin operation workspace")

// Adapter provides confined filesystem operations rooted at Plugin_Storage.
// Persisted paths remain slash-separated and relative to that root.
type Adapter struct {
	layout *Layout
	root   *os.Root
}

// NewAdapter initializes Plugin_Storage and opens a directory handle used for
// all subsequent file operations. os.Root confines operations against path
// traversal and symlink escapes even if a path changes after validation.
func NewAdapter(layout *Layout) (*Adapter, error) {
	if layout == nil {
		return nil, ErrInvalidPath
	}
	if err := layout.EnsurePluginStorage(); err != nil {
		return nil, fmt.Errorf("initialize Plugin_Storage: %w", err)
	}
	root, err := os.OpenRoot(layout.PluginStorageRoot())
	if err != nil {
		return nil, fmt.Errorf("open Plugin_Storage root: %w", err)
	}
	return &Adapter{layout: layout, root: root}, nil
}

// Close releases the confined Plugin_Storage directory handle.
func (a *Adapter) Close() error {
	if a == nil || a.root == nil {
		return nil
	}
	return a.root.Close()
}

// CreateTemporaryOperationDirectory implements the installation application
// port without exposing an absolute filesystem path.
func (a *Adapter) CreateTemporaryOperationDirectory(ctx context.Context, operationID string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if a == nil || a.layout == nil || a.root == nil {
		return "", ErrInvalidPath
	}
	relativePath, err := a.layout.RelativeTemporaryOperationPath(operationID)
	if err != nil {
		return "", err
	}
	parentInfo, err := a.root.Lstat("temporary")
	if err != nil {
		return "", fmt.Errorf("inspect temporary storage directory: %w", err)
	}
	if !parentInfo.IsDir() || parentInfo.Mode()&fs.ModeSymlink != 0 {
		return "", ErrInvalidPath
	}
	if err := a.root.Mkdir(filepath.FromSlash(relativePath), 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", ErrOperationDirectoryExists
		}
		return "", fmt.Errorf("create operation workspace: %w", err)
	}
	if runtime.GOOS != "windows" {
		if err := a.root.Chmod(filepath.FromSlash(relativePath), 0o700); err != nil {
			return "", fmt.Errorf("secure operation workspace: %w", err)
		}
	}
	info, err := a.root.Lstat(filepath.FromSlash(relativePath))
	if err != nil {
		return "", fmt.Errorf("inspect created operation workspace: %w", err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return "", ErrInvalidOperationWorkspace
	}
	return relativePath, nil
}

// CreateArtifactFile creates the single acquired ZIP artifact inside an
// operation workspace. Creation is exclusive and the file is private.
func (a *Adapter) CreateArtifactFile(ctx context.Context, stagingPath string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := a.validateOperationWorkspace(stagingPath); err != nil {
		return nil, err
	}
	filePath := path.Join(stagingPath, OperationArtifactFileName)
	if err := a.rejectExistingSymlink(filePath); err != nil {
		return nil, err
	}
	file, err := a.root.OpenFile(filepath.FromSlash(filePath), os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create staged Plugin artifact: %w", err)
	}
	return file, nil
}

// OpenArtifactFile opens the acquired ZIP as a regular read-only file.
func (a *Adapter) OpenArtifactFile(ctx context.Context, stagingPath string) (*os.File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := a.validateOperationWorkspace(stagingPath); err != nil {
		return nil, err
	}
	filePath := path.Join(stagingPath, OperationArtifactFileName)
	info, err := a.root.Lstat(filepath.FromSlash(filePath))
	if err != nil {
		return nil, fmt.Errorf("inspect staged Plugin artifact: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, ErrInvalidPath
	}
	file, err := a.root.OpenFile(filepath.FromSlash(filePath), os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open staged Plugin artifact: %w", err)
	}
	return file, nil
}

func (a *Adapter) validateOperationWorkspace(stagingPath string) error {
	if a == nil || a.layout == nil || a.root == nil {
		return ErrInvalidPath
	}
	if stagingPath == "" || strings.Contains(stagingPath, "\\") || path.Clean(stagingPath) != stagingPath {
		return ErrInvalidOperationWorkspace
	}
	parts := strings.Split(stagingPath, "/")
	if len(parts) != 2 || parts[0] != "temporary" || parts[1] == "" {
		return ErrInvalidOperationWorkspace
	}
	if _, err := a.layout.ResolvePluginStoragePath(stagingPath); err != nil {
		return err
	}
	info, err := a.root.Lstat(filepath.FromSlash(stagingPath))
	if err != nil {
		return fmt.Errorf("inspect operation workspace: %w", err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidOperationWorkspace
	}
	return nil
}

func (a *Adapter) rejectExistingSymlink(filePath string) error {
	info, err := a.root.Lstat(filepath.FromSlash(filePath))
	if err == nil {
		if info.Mode()&fs.ModeSymlink != 0 {
			return ErrInvalidPath
		}
		return os.ErrExist
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect staged artifact destination: %w", err)
	}
	return nil
}
