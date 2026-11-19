package zipextract

import (
	"archive/zip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"pride-plugin-module/installation"
	"pride-plugin-module/installation/zipinspect"
)

type testSource struct{ path string }
func (s testSource) OpenArtifactFile(context.Context, string) (*os.File, error) {
	return os.Open(s.path)
}
type testInspector struct{}
func (testInspector) Inspect(context.Context, string) (installation.InspectedPackage, error) {
	return installation.InspectedPackage{}, nil
}
type testTarget struct {

	root string

	failPath string

}
func (t testTarget) CreateExtractionRoot(_ context.Context, _ string) error {
	return os.Mkdir(filepath.Join(t.root, "package"), 0o700)
}
func (t testTarget) CreateExtractionDirectory(_ context.Context, _, relative string) error {
	return os.MkdirAll(filepath.Join(t.root, filepath.FromSlash(relative)), 0o700)
}
func (t testTarget) CreateExtractionFile(_ context.Context, _, relative string) (*os.File, error) {
	if relative == t.failPath {
		return nil, errors.New("injected file creation failure")
	}
	name := filepath.Join(t.root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(name), 0o700); err != nil {
		return nil, err
	}
	return os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
}
func (t testTarget) RemoveExtractionDirectory(_ context.Context, _, relative string) error {
	return os.RemoveAll(filepath.Join(t.root, relative))
}

func createTestZIP(t *testing.T, entries map[string]string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "artifact.zip")
	file, err := os.Create(name); if err != nil { t.Fatal(err) }
	writer := zip.NewWriter(file)
	for entry, value := range entries {
		output, err := writer.Create(entry); if err != nil { t.Fatal(err) }
		if _, err := output.Write([]byte(value)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return name
}

const testManifest = `{"$schema":"urn:pride:plugin-manifest:1","schema_version":"1.0.0","plugin_id":"example.plugin","name":"Example","version":"1.0.0","description":"Example plugin","author":"PRIDE","license":"MIT","entry_point":"main","minimum_pride_version":"1.0.0"}`

func TestExtractWritesOnlyPackagePayload(t *testing.T) {
	archive := createTestZIP(t, map[string]string{"plugin.json": testManifest, "package/main": "payload", "package/nested/data.txt": "data"})
	root := t.TempDir()
	extractor, err := New(testSource{archive}, testInspector{}, testTarget{root: root}, zipinspect.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	result, err := extractor.Extract(context.Background(), "temporary/op-1")
	if err != nil {
		t.Fatal(err)
	}
	if result != "temporary/op-1/package" {
		t.Fatalf("result = %q", result)
	}
	if _, err := os.Stat(filepath.Join(root, "package", "main")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "plugin.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("Manifest must not be extracted, got %v", err)
	}
}

func TestExtractCleansPartialOutputOnFailure(t *testing.T) {
	archive := createTestZIP(t, map[string]string{"plugin.json": testManifest, "package/main": "payload", "package/fail": "failure"})
	root := t.TempDir()
	extractor, err := New(testSource{archive}, testInspector{}, testTarget{root: root, failPath: "package/fail"}, zipinspect.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := extractor.Extract(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected injected failure")
	}
	if _, err := os.Stat(filepath.Join(root, "package")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("partial output should be removed, got %v", err)
	}
}

func TestPackageRelativePathRejectsTraversalAndOutsideEntries(t *testing.T) {
	for _, name := range []string{"../escape", "package/../escape", "/package/main", "package\\main", "other/file"} {
		if _, _, err := packageRelativePath(name); err == nil {
			t.Errorf("expected %q to be rejected", name)
		}
	}
}
