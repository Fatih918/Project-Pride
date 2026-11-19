package storage

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"path"
)

// RemoveArtifactFile removes only the regular artifact.zip file belonging to
// an existing operation workspace. It never removes the workspace itself.
func (a *Adapter) RemoveArtifactFile(ctx context.Context, stagingPath string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.validateOperationWorkspace(stagingPath); err != nil {
		return err
	}
	filePath := path.Join(stagingPath, OperationArtifactFileName)
	info, err := a.root.Lstat(filepath.FromSlash(filePath))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect staged artifact for removal: %w", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	if err := a.root.Remove(filepath.FromSlash(filePath)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove partial staged artifact: %w", err)
	}
	return nil
}
