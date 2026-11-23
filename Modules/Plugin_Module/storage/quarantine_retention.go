package storage

import (
	"context"
	"fmt"
	"io/fs"
	"errors"
	"math"
	"os"
	"path"
	"path/filepath"
	"time"
)

// QuarantineEntry describes one operation-scoped quarantine directory.
// ModifiedAt is the directory's filesystem modification time; SizeBytes is
// the sum of regular-file sizes below it.
type QuarantineEntry struct {
	OperationID string
	ModifiedAt  time.Time
	SizeBytes   int64
}

// ListQuarantineEntries returns a deterministic snapshot of safe quarantine
// directories. Symlinks and special files are rejected rather than followed.
func (a *Adapter) ListQuarantineEntries(ctx context.Context) ([]QuarantineEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.root == nil || a.layout == nil {
		return nil, ErrInvalidPath
	}
	rootInfo, err := a.root.Lstat("quarantine")
	if err != nil {
		return nil, fmt.Errorf("inspect quarantine root: %w", err)
	}
	if !rootInfo.IsDir() || rootInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, ErrInvalidPath
	}
	directories, err := a.root.ReadDir("quarantine")
	if err != nil {
		return nil, fmt.Errorf("list quarantine root: %w", err)
	}
	entries := make([]QuarantineEntry, 0, len(directories))
	for _, directory := range directories {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if !directory.IsDir() {
			return nil, fmt.Errorf("%w: quarantine entry is not a directory", ErrInvalidPath)
		}
		if _, err := a.layout.QuarantineOperationDirectory(directory.Name()); err != nil {
			return nil, err
		}
		relative := path.Join("quarantine", directory.Name())
		info, err := a.root.Lstat(filepath.FromSlash(relative))
		if err != nil {
			return nil, fmt.Errorf("inspect quarantine entry: %w", err)
		}
		if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
			return nil, ErrInvalidPath
		}
		size, err := a.quarantineDirectorySize(ctx, relative)
		if err != nil {
			return nil, fmt.Errorf("measure quarantine entry %q: %w", directory.Name(), err)
		}
		entries = append(entries, QuarantineEntry{
			OperationID: directory.Name(),
			ModifiedAt:  info.ModTime().UTC(),
			SizeBytes:   size,
		})
	}
	return entries, nil
}

func (a *Adapter) quarantineDirectorySize(ctx context.Context, directory string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	children, err := a.root.ReadDir(filepath.FromSlash(directory))
	if err != nil {
		return 0, err
	}
	var total int64
	for _, child := range children {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		childPath := path.Join(directory, child.Name())
		info, err := a.root.Lstat(filepath.FromSlash(childPath))
		if err != nil {
			return 0, err
		}
		switch {
		case info.Mode()&fs.ModeSymlink != 0:
			return 0, ErrInvalidPath
		case info.IsDir():
			size, err := a.quarantineDirectorySize(ctx, childPath)
			if err != nil {
				return 0, err
			}
			if size > math.MaxInt64-total {
				return 0, fmt.Errorf("quarantine size overflow")
			}
			total += size
		case info.Mode().IsRegular():
			if info.Size() < 0 || info.Size() > math.MaxInt64-total {
				return 0, fmt.Errorf("quarantine size overflow")
			}
			total += info.Size()
		default:
			return 0, ErrInvalidPath
		}
	}
	return total, nil
}

// DeleteQuarantineOperation removes one validated operation directory. The
// caller must establish from the durable operation journal that it is terminal
// and no longer needed before invoking this method.
func (a *Adapter) DeleteQuarantineOperation(ctx context.Context, operationID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if a == nil || a.root == nil || a.layout == nil {
		return ErrInvalidPath
	}
	if _, err := a.layout.QuarantineOperationDirectory(operationID); err != nil {
		return err
	}
	relative := path.Join("quarantine", operationID)
	info, err := a.root.Lstat(filepath.FromSlash(relative))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("inspect quarantine directory before deletion: %w", err)
	}
	if !info.IsDir() || info.Mode()&fs.ModeSymlink != 0 {
		return ErrInvalidPath
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := a.root.RemoveAll(filepath.FromSlash(relative)); err != nil {
		return fmt.Errorf("remove quarantined Plugin operation %q: %w", operationID, err)
	}
	return nil
}
