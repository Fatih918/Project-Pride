package zipinspect

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fileSource struct{ path string }

func (s fileSource) OpenArtifactFile(context.Context, string) (*os.File, error) {
	return os.Open(s.path)
}

func makeArchive(t *testing.T, entries map[string]string) string {
	t.Helper()
	name := filepath.Join(t.TempDir(), "artifact.zip")
	f, err := os.Create(name)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	for entry, content := range entries {
		h := &zip.FileHeader{Name: entry, Method: zip.Deflate}
		out, err := w.CreateHeader(h)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = out.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	return name
}

const validManifest = `{"$schema":"urn:pride:plugin-manifest:1","schema_version":"1.0.0","plugin_id":"example.plugin","name":"Example","version":"1.0.0","description":"Example plugin","author":"PRIDE","license":"MIT","entry_point":"main","minimum_pride_version":"1.0.0","future_field":{"enabled":true}}`

func TestInspectValidPackageAndPreservesUnknownManifestFields(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/main": "package entry"})
	inspector, err := New(fileSource{file}, DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	result, err := inspector.Inspect(context.Background(), "temporary/op-1")
	if err != nil {
		t.Fatal(err)
	}
	if result.Manifest.PluginID != "example.plugin" || result.EntryCount != 2 || result.ExpandedBytes == 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
	if !bytes.Contains(result.ManifestJSON, []byte("future_field")) {
		t.Fatal("raw Manifest snapshot must preserve forward-compatible fields")
	}
}

func TestInspectRejectsTraversal(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/main": "ok", "package/../escape": "bad"})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	if _, err := inspector.Inspect(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected traversal rejection")
	}
}

func TestInspectRejectsDuplicateManifestKeys(t *testing.T) {
	manifest := `{"$schema":"urn:pride:plugin-manifest:1","schema_version":"1.0.0","plugin_id":"one","plugin_id":"two","name":"Example","version":"1","description":"x","author":"a","license":"MIT","entry_point":"main","minimum_pride_version":"1"}`
	file := makeArchive(t, map[string]string{"plugin.json": manifest, "package/main": "ok"})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	if _, err := inspector.Inspect(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected duplicate-key rejection")
	}
}

func TestPolicyFailsClosedWhenValuesDiffer(t *testing.T) {
	p := DefaultPolicy()
	p.Limits.MaxEntryCount++
	if _, err := New(fileSource{"unused"}, p); err == nil {
		t.Fatal("expected modified policy rejection")
	}
}

func TestInspectHonorsContextCancellation(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/main": "ok"})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := inspector.Inspect(ctx, "temporary/op-1"); err == nil {
		t.Fatal("expected context cancellation")
	}
}

func TestInspectRejectsZIP64EOCDSentinel(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/main": "ok"})
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	offset := bytes.LastIndex(data, []byte{0x50, 0x4b, 0x05, 0x06})
	if offset < 0 {
		t.Fatal("EOCD not found in test archive")
	}
	binary.LittleEndian.PutUint16(data[offset+8:offset+10], 0xffff)
	binary.LittleEndian.PutUint16(data[offset+10:offset+12], 0xffff)
	if err := os.WriteFile(file, data, 0o600); err != nil {
		t.Fatal(err)
	}
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	_, err = inspector.Inspect(context.Background(), "temporary/op-1")
	if !errors.Is(err, ErrUnsupportedZIP64) {
		t.Fatalf("expected ZIP64 rejection, got %v", err)
	}
}

func TestInspectRejectsCompressionRatioOverPolicy(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/main": strings.Repeat("a", 20000)})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	if _, err := inspector.Inspect(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected compression-ratio rejection")
	}
}

func TestInspectRejectsCaseFoldCollision(t *testing.T) {
	file := makeArchive(t, map[string]string{"plugin.json": validManifest, "package/Main": "one", "package/main": "two"})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	if _, err := inspector.Inspect(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected case-fold collision rejection")
	}
}

func TestLoadPolicyAcceptsExactPublishedPolicy(t *testing.T) {
	data, err := json.Marshal(DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := LoadPolicy(bytes.NewReader(data)); err != nil {
		t.Fatal(err)
	}
}

func TestLoadPolicyRejectsDuplicateAndTrailingJSON(t *testing.T) {
	if _, err := LoadPolicy(strings.NewReader(`{"$schema":"a","$schema":"b"}`)); err == nil {
		t.Fatal("expected duplicate-key rejection")
	}
	if _, err := LoadPolicy(strings.NewReader(`{} {}`)); err == nil {
		t.Fatal("expected trailing JSON rejection")
	}
}

func TestInspectRejectsFileDirectoryPathConflict(t *testing.T) {
	file := makeArchive(t, map[string]string{
		"plugin.json":        validManifest,
		"package/main":       "file",
		"package/main/child": "nested",
	})
	inspector, _ := New(fileSource{file}, DefaultPolicy())
	if _, err := inspector.Inspect(context.Background(), "temporary/op-1"); err == nil {
		t.Fatal("expected file/directory path conflict rejection")
	}
}
