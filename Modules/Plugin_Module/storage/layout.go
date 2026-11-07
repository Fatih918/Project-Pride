package storage

import (
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var (
	ErrInvalidPath              = errors.New("invalid plugin storage path")
	ErrOperationDirectoryExists = errors.New("plugin operation directory already exists")
)

type Config struct {
	RootOverride string
	GOOS         string
	HomeDir      string
	LookupEnv    func(string) (string, bool)
}

type Layout struct {
	root string
}

var componentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

func NewLayout(config Config) (*Layout, error) {
	goos := config.GOOS
	if goos == "" {
		goos = runtime.GOOS
	}
	home := config.HomeDir
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("resolve user home: %w", err)
		}
	}
	lookup := config.LookupEnv
	if lookup == nil {
		lookup = os.LookupEnv
	}

	override := config.RootOverride
	if override == "" {
		if value, ok := lookup("PRIDE_USER_DATA_DIR"); ok && value != "" {
			override = value
		}
	}
	var root string
	if override != "" {
		if !filepath.IsAbs(override) {
			return nil, fmt.Errorf("%w: PRIDE_USER_DATA_DIR must be absolute", ErrInvalidPath)
		}
		root = filepath.Clean(override)
	} else {
		switch goos {
		case "windows":
			local, ok := lookup("LOCALAPPDATA")
			if !ok || strings.TrimSpace(local) == "" || !filepath.IsAbs(local) {
				return nil, fmt.Errorf("resolve Windows user data directory: LOCALAPPDATA must be an absolute path")
			}
			root = filepath.Join(local, "PRIDE")
		case "darwin":
			root = filepath.Join(home, "Library", "Application Support", "PRIDE")
		case "linux":
			if xdg, ok := lookup("XDG_DATA_HOME"); ok && xdg != "" {
				if !filepath.IsAbs(xdg) {
					return nil, fmt.Errorf("%w: XDG_DATA_HOME must be absolute", ErrInvalidPath)
				}
				root = filepath.Join(xdg, "pride")
			} else {
				root = filepath.Join(home, ".local", "share", "pride")
			}
		default:
			return nil, fmt.Errorf("unsupported operating system %q", goos)
		}
	}
	return &Layout{root: filepath.Clean(root)}, nil
}

func (l *Layout) Root() string           { return l.root }
func (l *Layout) StateDirectory() string { return filepath.Join(l.root, "state") }
func (l *Layout) RegistryDatabasePath() string {
	return filepath.Join(l.StateDirectory(), "plugin-registry.sqlite")
}
func (l *Layout) PluginStorageRoot() string { return filepath.Join(l.root, "Plugin_Storage") }

func (l *Layout) EnsureStateDirectory() error {
	return l.ensureDirectory(l.StateDirectory())
}

func (l *Layout) EnsurePluginStorage() error {
	root := l.PluginStorageRoot()
	for _, path := range []string{
		root,
		filepath.Join(root, "installed"),
		filepath.Join(root, "data"),
		filepath.Join(root, "cache"),
		filepath.Join(root, "temporary"),
		filepath.Join(root, "quarantine"),
	} {
		if err := l.ensureDirectory(path); err != nil {
			return err
		}
	}
	return nil
}

func (l *Layout) InstalledVersionDirectory(pluginID, version string) (string, error) {
	if err := validateComponent(pluginID); err != nil {
		return "", fmt.Errorf("plugin ID: %w", err)
	}
	if err := validateComponent(version); err != nil {
		return "", fmt.Errorf("plugin version: %w", err)
	}
	return l.safePath("installed", pluginID, version)
}

// TemporaryOperationDirectory resolves the isolated staging directory for an
// operation. The returned path is absolute and remains inside Plugin_Storage.
func (l *Layout) TemporaryOperationDirectory(operationID string) (string, error) {
	if err := validateComponent(operationID); err != nil {
		return "", fmt.Errorf("operation ID: %w", err)
	}
	return l.safePath("temporary", operationID)
}

// QuarantineOperationDirectory resolves the operation-scoped quarantine
// directory. This helper does not move or delete files and makes no retention
// policy decisions.
func (l *Layout) QuarantineOperationDirectory(operationID string) (string, error) {
	if err := validateComponent(operationID); err != nil {
		return "", fmt.Errorf("operation ID: %w", err)
	}
	return l.safePath("quarantine", operationID)
}

// CreateTemporaryOperationDirectory creates a fresh private staging directory.
// Existing paths are never reused, including paths left by interrupted runs.
func (l *Layout) CreateTemporaryOperationDirectory(operationID string) (string, error) {
	path, err := l.TemporaryOperationDirectory(operationID)
	if err != nil {
		return "", err
	}
	if err := l.ensureDirectory(filepath.Join(l.PluginStorageRoot(), "temporary")); err != nil {
		return "", err
	}
	if err := l.verifyContained(l.root, path); err != nil {
		return "", err
	}
	if err := os.Mkdir(path, 0o700); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", ErrOperationDirectoryExists
		}
		return "", fmt.Errorf("create Plugin operation staging directory: %w", err)
	}
	if err := l.verifyContained(l.root, path); err != nil {
		return "", err
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(path, 0o700); err != nil {
			return "", fmt.Errorf("secure Plugin operation staging directory: %w", err)
		}
	}
	info, err := os.Lstat(path)
	if err != nil {
		return "", fmt.Errorf("inspect Plugin operation staging directory: %w", err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", ErrInvalidPath
	}
	return path, nil
}

func (l *Layout) PluginDataDirectory(pluginID string) (string, error) {
	if err := validateComponent(pluginID); err != nil {
		return "", fmt.Errorf("plugin ID: %w", err)
	}
	return l.safePath("data", pluginID)
}

func validateComponent(value string) error {
	if value == "" || value == "." || value == ".." || filepath.IsAbs(value) ||
		strings.ContainsAny(value, `/\\:`) || !componentPattern.MatchString(value) {
		return ErrInvalidPath
	}
	return nil
}

func (l *Layout) safePath(parts ...string) (string, error) {
	root := l.PluginStorageRoot()
	candidate := root
	for _, part := range parts {
		if err := validateComponent(part); err != nil {
			return "", err
		}
		candidate = filepath.Join(candidate, part)
	}
	if err := l.verifyContained(l.root, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func (l *Layout) ensureDirectory(path string) error {
	if err := os.MkdirAll(path, 0o700); err != nil {
		return fmt.Errorf("create PRIDE directory: %w", err)
	}
	if err := l.verifyContained(l.root, path); err != nil {
		return err
	}
	if err := os.Chmod(path, 0o700); err != nil && runtime.GOOS != "windows" {
		return fmt.Errorf("secure PRIDE directory permissions: %w", err)
	}
	if err := l.verifyContained(l.root, path); err != nil {
		return err
	}
	return nil
}

func (l *Layout) verifyContained(root, target string) error {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolve storage root: %w", err)
	}
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolve storage path: %w", err)
	}
	rel, err := filepath.Rel(absRoot, absTarget)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return ErrInvalidPath
	}
	realRoot, err := filepath.EvalSymlinks(absRoot)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("resolve storage root symlinks: %w", err)
	}
	if err == nil {
		probe := absTarget
		for {
			realProbe, evalErr := filepath.EvalSymlinks(probe)
			if evalErr == nil {
				realRel, relErr := filepath.Rel(realRoot, realProbe)
				if relErr != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(os.PathSeparator)) {
					return ErrInvalidPath
				}
				break
			}
			if !errors.Is(evalErr, os.ErrNotExist) {
				return fmt.Errorf("resolve storage path symlinks: %w", evalErr)
			}
			parent := filepath.Dir(probe)
			if parent == probe {
				return ErrInvalidPath
			}
			probe = parent
		}
	}
	return nil
}

// RelativeInstalledVersionPath returns a canonical path relative to Plugin_Storage.
// Persist this value; never persist the absolute result of InstalledVersionDirectory.
func (l *Layout) RelativeInstalledVersionPath(pluginID, version string) (string, error) {
	if err := validateComponent(pluginID); err != nil {
		return "", fmt.Errorf("plugin ID: %w", err)
	}
	if err := validateComponent(version); err != nil {
		return "", fmt.Errorf("plugin version: %w", err)
	}
	return path.Join("installed", pluginID, version), nil
}

// RelativeTemporaryOperationPath returns the canonical operation staging path
// relative to Plugin_Storage.
func (l *Layout) RelativeTemporaryOperationPath(operationID string) (string, error) {
	if err := validateComponent(operationID); err != nil {
		return "", fmt.Errorf("operation ID: %w", err)
	}
	return path.Join("temporary", operationID), nil
}

// ResolvePluginStoragePath resolves a canonical Plugin_Storage-relative path.
// It rejects paths outside the managed storage namespaces and checks symlink
// containment before returning an absolute OS path. Callers must not resolve
// these paths relative to the process working directory.
func (l *Layout) ResolvePluginStoragePath(relativePath string) (string, error) {
	if relativePath == "" || strings.Contains(relativePath, "\\") ||
		strings.HasPrefix(relativePath, "/") || path.Clean(relativePath) != relativePath {
		return "", ErrInvalidPath
	}
	parts := strings.Split(relativePath, "/")
	if len(parts) < 2 {
		return "", ErrInvalidPath
	}
	switch parts[0] {
	case "installed", "data", "cache", "temporary", "quarantine":
	default:
		return "", ErrInvalidPath
	}
	for _, part := range parts {
		if err := validateStorageSegment(part); err != nil {
			return "", err
		}
	}
	candidate := l.PluginStorageRoot()
	for _, part := range parts {
		candidate = filepath.Join(candidate, part)
	}
	if err := l.verifyContained(l.root, candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func validateStorageSegment(value string) error {
	if value == "" || value == "." || value == ".." ||
		strings.ContainsAny(value, `/\\:<>:"|?*`) || strings.ContainsRune(value, 0) ||
		strings.HasSuffix(value, ".") || strings.HasSuffix(value, " ") {
		return ErrInvalidPath
	}
	base := strings.ToUpper(strings.SplitN(value, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL",
		"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
		"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return ErrInvalidPath
	}
	return nil
}

// CreateTemporaryOperationDirectoryRelative creates a fresh private staging
// directory and returns its canonical Plugin_Storage-relative path for
// persistence and application-layer coordination.
func (l *Layout) CreateTemporaryOperationDirectoryRelative(operationID string) (string, error) {
	relativePath, err := l.RelativeTemporaryOperationPath(operationID)
	if err != nil {
		return "", err
	}
	if _, err := l.CreateTemporaryOperationDirectory(operationID); err != nil {
		return "", err
	}
	return relativePath, nil
}
