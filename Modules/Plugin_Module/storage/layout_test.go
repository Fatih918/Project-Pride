package storage

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func env(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) { value, ok := values[key]; return value, ok }
}

func TestNewLayoutDefaults(t *testing.T) {
	home := t.TempDir()
	tests := []struct {
		name, goos string
		values     map[string]string
		want       string
	}{
		{"linux fallback", "linux", nil, filepath.Join(home, ".local", "share", "pride")},
		{"linux xdg", "linux", map[string]string{"XDG_DATA_HOME": filepath.Join(home, "data")}, filepath.Join(home, "data", "pride")},
		{"darwin", "darwin", nil, filepath.Join(home, "Library", "Application Support", "PRIDE")},
		{"windows", "windows", map[string]string{"LOCALAPPDATA": filepath.Join(home, "Local")}, filepath.Join(home, "Local", "PRIDE")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			layout, err := NewLayout(Config{GOOS: tt.goos, HomeDir: home, LookupEnv: env(tt.values)})
			if err != nil {
				t.Fatal(err)
			}
			if layout.Root() != tt.want {
				t.Fatalf("root = %q, want %q", layout.Root(), tt.want)
			}
		})
	}
}

func TestNewLayoutRejectsRelativeOverrides(t *testing.T) {
	_, err := NewLayout(Config{RootOverride: "relative/path", GOOS: "linux", HomeDir: t.TempDir()})
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
}

func TestEnsureDirectoriesAreIndependentAndIdempotent(t *testing.T) {
	layout, err := NewLayout(Config{RootOverride: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := layout.EnsureStateDirectory(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(layout.PluginStorageRoot()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Plugin_Storage should not exist: %v", err)
	}
	if err := layout.EnsurePluginStorage(); err != nil {
		t.Fatal(err)
	}
	if err := layout.EnsurePluginStorage(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"installed", "data", "cache", "temporary", "quarantine"} {
		if info, err := os.Stat(filepath.Join(layout.PluginStorageRoot(), name)); err != nil || !info.IsDir() {
			t.Fatalf("%s directory missing: %v", name, err)
		}
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(layout.StateDirectory())
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o700 {
			t.Fatalf("state permissions = %o", info.Mode().Perm())
		}
	}
}

func TestPluginPathsRejectUnsafeComponents(t *testing.T) {
	layout, _ := NewLayout(Config{RootOverride: t.TempDir()})
	for _, value := range []string{"", ".", "..", "../escape", "/absolute", `a\b`, "a/b", "C:drive"} {
		if _, err := layout.InstalledVersionDirectory(value, "1.0.0"); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("plugin ID %q: got %v", value, err)
		}
	}
	if _, err := layout.InstalledVersionDirectory("plugin.good", "../escape"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected version traversal rejection, got %v", err)
	}
}

func TestPluginPathRejectsSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires platform-specific privileges")
	}
	base := t.TempDir()
	layout, _ := NewLayout(Config{RootOverride: filepath.Join(base, "pride")})
	if err := layout.EnsurePluginStorage(); err != nil {
		t.Fatal(err)
	}
	outside := t.TempDir()
	if err := os.Symlink(outside, filepath.Join(layout.PluginStorageRoot(), "installed")); err != nil {
		t.Fatal(err)
	}
	if _, err := layout.InstalledVersionDirectory("plugin.good", "1.0.0"); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected symlink escape rejection, got %v", err)
	}
}
