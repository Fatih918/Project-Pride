package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const MaxCapabilitySchemaDocumentBytes int64 = 65_536

var ErrInvalidPackageFile = errors.New("invalid installed Plugin package file")

// ReadPackageFile reads one bounded schema file from an immutable installed
// package. A package-scoped os.Root prevents a reference from escaping the
// extracted package even if filesystem entries are replaced concurrently.
func (a *Adapter) ReadPackageFile(ctx context.Context, artifactDirectory, relativePath string, maxBytes int64) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if a == nil || a.root == nil || a.layout == nil || maxBytes <= 0 ||
		maxBytes > MaxCapabilitySchemaDocumentBytes {
		return nil, ErrInvalidPackageFile
	}
	if err := a.validateInstalledArtifactDirectory(artifactDirectory); err != nil {
		return nil, err
	}
	if err := validatePackageSchemaPath(relativePath); err != nil {
		return nil, err
	}

	packagePath := path.Join(artifactDirectory, "package")
	packageInfo, err := a.root.Lstat(filepath.FromSlash(packagePath))
	if err != nil || !packageInfo.IsDir() || packageInfo.Mode()&fs.ModeSymlink != 0 {
		return nil, ErrInvalidPackageFile
	}
	packageRoot, err := a.root.OpenRoot(filepath.FromSlash(packagePath))
	if err != nil {
		return nil, fmt.Errorf("open confined Plugin package root: %w", err)
	}
	defer packageRoot.Close()

	current := ""
	parts := strings.Split(relativePath, "/")
	for index, part := range parts {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if current == "" {
			current = part
		} else {
			current = path.Join(current, part)
		}
		info, statErr := packageRoot.Lstat(filepath.FromSlash(current))
		if statErr != nil {
			return nil, fmt.Errorf("inspect package schema path: %w", statErr)
		}
		if info.Mode()&fs.ModeSymlink != 0 {
			return nil, ErrInvalidPackageFile
		}
		if index < len(parts)-1 && !info.IsDir() {
			return nil, ErrInvalidPackageFile
		}
		if index == len(parts)-1 && !info.Mode().IsRegular() {
			return nil, ErrInvalidPackageFile
		}
	}
	file, err := packageRoot.OpenFile(filepath.FromSlash(relativePath), os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("open package schema document: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() < 0 || info.Size() > maxBytes {
		return nil, ErrInvalidPackageFile
	}

	reader := io.LimitReader(file, maxBytes+1)
	data := make([]byte, 0, info.Size())
	buffer := make([]byte, 8*1024)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, readErr := reader.Read(buffer)
		if n > 0 {
			if int64(len(data))+int64(n) > maxBytes {
				return nil, ErrInvalidPackageFile
			}
			data = append(data, buffer[:n]...)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return nil, fmt.Errorf("read package schema document: %w", readErr)
		}
		if n == 0 {
			return nil, io.ErrNoProgress
		}
	}
	return data, nil
}

func validatePackageSchemaPath(value string) error {
	if value == "" || len([]byte(value)) > 240 ||
		strings.ContainsAny(value, "\\\x00?#") || containsEncodedPackageSeparator(value) ||
		strings.HasPrefix(value, "/") || path.Clean(value) != value ||
		!strings.HasPrefix(value, "schemas/") {
		return ErrInvalidPackageFile
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.Contains(part, ":") {
			return ErrInvalidPackageFile
		}
	}
	return nil
}

func containsEncodedPackageSeparator(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}
