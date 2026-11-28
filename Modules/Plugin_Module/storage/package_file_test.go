package storage

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestReadPackageFileReadsBoundedSchemaFromInstalledPackage(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(Config{RootOverride: root})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	packageRoot := filepath.Join(layout.PluginStorageRoot(), "installed", "example.plugin", "1.2.3", "package")
	if err := os.MkdirAll(filepath.Join(packageRoot, "schemas"), 0o700); err != nil {
		t.Fatal(err)
	}
	want := []byte(`{"type":"object"}`)
	if err := os.WriteFile(filepath.Join(packageRoot, "schemas", "request.json"), want, 0o600); err != nil {
		t.Fatal(err)
	}

	got, err := adapter.ReadPackageFile(context.Background(), "installed/example.plugin/1.2.3", "schemas/request.json", 1024)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(want) {
		t.Fatalf("document = %q, want %q", got, want)
	}
}

func TestReadPackageFileRejectsTraversalAndOversizedFiles(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(Config{RootOverride: root})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	packageRoot := filepath.Join(layout.PluginStorageRoot(), "installed", "example.plugin", "1.2.3", "package")
	if err := os.MkdirAll(filepath.Join(packageRoot, "schemas"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageRoot, "schemas", "large.json"), []byte("12345"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := adapter.ReadPackageFile(context.Background(), "installed/example.plugin/1.2.3", "schemas/../secret.json", 16); !errors.Is(err, ErrInvalidPackageFile) {
		t.Fatalf("traversal error = %v, want ErrInvalidPackageFile", err)
	}
	if _, err := adapter.ReadPackageFile(context.Background(), "installed/example.plugin/1.2.3", "schemas/large.json", 4); !errors.Is(err, ErrInvalidPackageFile) {
		t.Fatalf("oversize error = %v, want ErrInvalidPackageFile", err)
	}
}

func TestReadPackageFileRejectsSymlinkAndUninstalledPaths(t *testing.T) {
	root := t.TempDir()
	layout, err := NewLayout(Config{RootOverride: root})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	defer adapter.Close()

	packageRoot := filepath.Join(layout.PluginStorageRoot(), "installed", "example.plugin", "1.2.3", "package")
	if err := os.MkdirAll(filepath.Join(packageRoot, "schemas"), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(packageRoot, "schemas", "target.json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(packageRoot, "schemas", "link.json")
	if err := os.Symlink(target, link); err == nil {
		if _, err := adapter.ReadPackageFile(context.Background(), "installed/example.plugin/1.2.3", "schemas/link.json", 16); !errors.Is(err, ErrInvalidPackageFile) {
			t.Fatalf("symlink error = %v, want ErrInvalidPackageFile", err)
		}
	}

	if _, err := adapter.ReadPackageFile(context.Background(), "temporary/op", "schemas/target.json", 16); !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("uninstalled path error = %v, want ErrInvalidPath", err)
	}
}
