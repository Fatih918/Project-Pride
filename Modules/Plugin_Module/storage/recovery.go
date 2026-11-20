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

// QuarantineOperation atomically moves an incomplete staging workspace or an
// uncommitted promoted version into quarantine/<operation-id>. Repeating the
// same move is idempotent when the source is already absent and the quarantine
// destination already exists. It never overwrites an existing destination.
func (a *Adapter) QuarantineOperation(ctx context.Context, sourcePath, operationID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.layout == nil || a.root == nil {
		return ErrInvalidPath
	}
	if _, err := a.layout.QuarantineOperationDirectory(operationID); err != nil {
		return err
	}

	switch {
	case strings.HasPrefix(sourcePath, "temporary/"):
		expected, err := a.layout.RelativeTemporaryOperationPath(operationID)
		if err != nil || sourcePath != expected {
			return ErrInvalidPath
		}
		if err := a.validateOperationWorkspace(sourcePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	case strings.HasPrefix(sourcePath, "installed/"):
		if err := a.validateInstalledArtifactDirectory(sourcePath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	default:
		return ErrInvalidPath
	}

	sourceInfo, sourceErr := a.root.Lstat(filepath.FromSlash(sourcePath))
	targetPath := path.Join("quarantine", operationID)
	targetInfo, targetErr := a.root.Lstat(filepath.FromSlash(targetPath))
	if sourceErr != nil && !errors.Is(sourceErr, os.ErrNotExist) {
		return fmt.Errorf("inspect recovery source: %w", sourceErr)
	}
	if targetErr != nil && !errors.Is(targetErr, os.ErrNotExist) {
		return fmt.Errorf("inspect quarantine destination: %w", targetErr)
	}
	if errors.Is(sourceErr, os.ErrNotExist) {
		if targetErr == nil && targetInfo.IsDir() && targetInfo.Mode()&fs.ModeSymlink == 0 {
			return nil
		}
		if errors.Is(targetErr, os.ErrNotExist) {
			return nil
		}
		return ErrInvalidPath
	}
	if !sourceInfo.IsDir() || sourceInfo.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	if targetErr == nil {
		return fmt.Errorf("quarantine destination already exists")
	}
	quarantineInfo, err := a.root.Lstat("quarantine")
	if err != nil || !quarantineInfo.IsDir() || quarantineInfo.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.root.Rename(filepath.FromSlash(sourcePath), filepath.FromSlash(targetPath)); err != nil {
		return fmt.Errorf("move interrupted Plugin artifact into quarantine: %w", err)
	}
	return nil
}

func (a *Adapter) validateInstalledArtifactDirectory(value string) error {
	if err := a.validateFinalArtifactPath(value); err != nil {
		return err
	}
	parts := strings.Split(value, "/")
	for index := 1; index <= 3; index++ {
		candidate := strings.Join(parts[:index], "/")
		info, err := a.root.Lstat(filepath.FromSlash(candidate))
		if err != nil {
			return fmt.Errorf("inspect installed artifact directory %q: %w", candidate, err)
		}
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return ErrInvalidPath
		}
	}
	return nil
}
