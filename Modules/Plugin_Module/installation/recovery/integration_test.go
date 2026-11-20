package recovery

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
	"pride-plugin-module/installation/zipinspect"
	"pride-plugin-module/registry/sqlite"
	"pride-plugin-module/storage"
)

type integrationMetadataResolver struct {
	metadata installation.PlatformMetadata
}

func (r integrationMetadataResolver) Resolve(context.Context, domain.PluginID, string) (installation.PlatformMetadata, error) {
	return r.metadata, nil
}

type integrationArtifactVerifier struct {
	finalPath string
	digest    string
	policy    string
}

func (v integrationArtifactVerifier) Verify(_ context.Context, _ installation.StagedArtifact, inspected installation.InspectedPackage, _ domain.Plugin) (installation.VerifiedArtifact, error) {
	return installation.VerifiedArtifact{
		FinalArtifactPath:  v.finalPath,
		DigestAlgorithm:    "sha-256",
		DigestValue:        v.digest,
		VerificationPolicy: v.policy,
		ManifestJSON:       string(inspected.ManifestJSON),
	}, nil
}

type recoveryIntegrationEnvironment struct {
	layout  *storage.Layout
	storage *storage.Adapter
	repo    *sqlite.Repository
	inspect *zipinspect.Inspector
}

func newRecoveryIntegrationEnvironment(t *testing.T) *recoveryIntegrationEnvironment {
	t.Helper()
	root := t.TempDir()
	layout, err := storage.NewLayout(storage.Config{RootOverride: root})
	if err != nil {
		t.Fatal(err)
	}
	adapter, err := storage.NewAdapter(layout)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = adapter.Close() })
	if err := layout.EnsureStateDirectory(); err != nil {
		t.Fatal(err)
	}
	repo, err := sqlite.New(context.Background(), layout.RegistryDatabasePath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = repo.Close() })
	inspector, err := zipinspect.New(adapter, zipinspect.DefaultPolicy())
	if err != nil {
		t.Fatal(err)
	}
	return &recoveryIntegrationEnvironment{layout: layout, storage: adapter, repo: repo, inspect: inspector}
}

func integrationManifestJSON(t *testing.T) []byte {
	t.Helper()
	manifest := map[string]any{
		"$schema": "urn:pride:plugin-manifest:1",
		"schema_version": "1.0.0",
		"plugin_id": "sample.plugin",
		"name": "Sample Plugin",
		"version": "1.0.0",
		"description": "Recovery integration fixture",
		"author": "PRIDE",
		"license": "MIT",
		"entry_point": "main.js",
		"minimum_pride_version": "1.0.0",
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func createIntegrationPackage(t *testing.T, env *recoveryIntegrationEnvironment, operationID string) (string, string, []byte) {
	t.Helper()
	stagingPath, err := env.storage.CreateTemporaryOperationDirectory(context.Background(), operationID)
	if err != nil {
		t.Fatal(err)
	}
	manifestJSON := integrationManifestJSON(t)
	var archive bytes.Buffer
	zw := zip.NewWriter(&archive)
	manifestWriter, err := zw.Create("plugin.json")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := manifestWriter.Write(manifestJSON); err != nil {
		t.Fatal(err)
	}
	entryWriter, err := zw.Create("package/main.js")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entryWriter.Write([]byte("export default {};")); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	artifact, err := env.storage.CreateArtifactFile(context.Background(), stagingPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := artifact.Write(archive.Bytes()); err != nil {
		_ = artifact.Close()
		t.Fatal(err)
	}
	if err := artifact.Close(); err != nil {
		t.Fatal(err)
	}
	packageDir := filepath.Join(env.layout.PluginStorageRoot(), filepath.FromSlash(stagingPath), "package")
	if err := os.Mkdir(packageDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packageDir, "main.js"), []byte("export default {};"), 0o600); err != nil {
		t.Fatal(err)
	}
	return stagingPath, filepath.Join(env.layout.PluginStorageRoot(), filepath.FromSlash(stagingPath)), manifestJSON
}

func integrationOperation(operationID, stagingPath string, now time.Time) installation.InstallationOperation {
	return installation.InstallationOperation{
		OperationID: installation.OperationID(operationID),
		PluginID: "sample.plugin",
		TargetVersion: "1.0.0",
		Kind: installation.KindInstall,
		Phase: installation.PhasePreparing,
		StagingPath: stagingPath,
		SourceReference: "registry:sample.plugin:1.0.0",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestRecoveryIntegrationSQLiteAndFilesystemRollsBackStagedOperation(t *testing.T) {
	env := newRecoveryIntegrationEnvironment(t)
	ctx := context.Background()
	stagingPath, stagingAbsolute, _ := createIntegrationPackage(t, env, "integration-rollback")
	op := integrationOperation("integration-rollback", stagingPath, time.Now().UTC())
	if err := env.repo.BeginOperation(ctx, op); err != nil {
		t.Fatal(err)
	}
	if err := env.repo.RecordStaged(ctx, op.OperationID, installation.StagedArtifact{
		StagingPath: stagingPath, DescriptorJSON: []byte(`{"schema_version":"1.0.0"}`),
	}); err != nil {
		t.Fatal(err)
	}

	coordinator, err := New(env.repo, env.repo, env.storage, env.inspect,
		integrationMetadataResolver{metadata: installation.PlatformMetadata{Type: domain.PluginTypePassive}},
		integrationArtifactVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	persisted, err := env.repo.GetOperation(ctx, op.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Phase != installation.PhaseRolledBack || persisted.FailureCode != failureInterrupted {
		t.Fatalf("operation state = %s/%s, want ROLLED_BACK/%s", persisted.Phase, persisted.FailureCode, failureInterrupted)
	}
	if _, err := os.Stat(stagingAbsolute); !os.IsNotExist(err) {
		t.Fatalf("staging workspace still exists after recovery: %v", err)
	}
	quarantine := filepath.Join(env.layout.PluginStorageRoot(), "quarantine", string(op.OperationID))
	if info, err := os.Stat(quarantine); err != nil || !info.IsDir() {
		t.Fatalf("quarantine workspace missing: info=%v err=%v", info, err)
	}
}

func TestRecoveryIntegrationSQLiteAndFilesystemCommitsPromotedArtifact(t *testing.T) {
	env := newRecoveryIntegrationEnvironment(t)
	ctx := context.Background()
	stagingPath, stagingAbsolute, manifestJSON := createIntegrationPackage(t, env, "integration-promoted")
	now := time.Now().UTC().Truncate(time.Nanosecond)
	op := integrationOperation("integration-promoted", stagingPath, now)
	if err := env.repo.BeginOperation(ctx, op); err != nil {
		t.Fatal(err)
	}
	descriptor := []byte(`{"schema_version":"1.0.0"}`)
	if err := env.repo.RecordStaged(ctx, op.OperationID, installation.StagedArtifact{
		StagingPath: stagingPath, DescriptorJSON: descriptor,
	}); err != nil {
		t.Fatal(err)
	}
	finalRelative := "installed/sample.plugin/1.0.0"
	finalAbsolute := filepath.Join(env.layout.PluginStorageRoot(), filepath.FromSlash(finalRelative))
	if err := os.MkdirAll(filepath.Dir(finalAbsolute), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(stagingAbsolute, finalAbsolute); err != nil {
		t.Fatal(err)
	}
	verified := installation.VerifiedArtifact{
		FinalArtifactPath: finalRelative,
		DigestAlgorithm: "sha-256",
		DigestValue: "integration-digest",
		VerificationPolicy: "integration-policy-v1",
		ManifestJSON: string(manifestJSON),
	}
	if err := env.repo.RecordVerified(ctx, op.OperationID, verified); err != nil {
		t.Fatal(err)
	}
	if err := env.repo.RecordPromoted(ctx, op.OperationID, finalRelative); err != nil {
		t.Fatal(err)
	}

	coordinator, err := New(env.repo, env.repo, env.storage, env.inspect,
		integrationMetadataResolver{metadata: installation.PlatformMetadata{
			Type: domain.PluginTypePassive,
			Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}},
		}},
		integrationArtifactVerifier{finalPath: finalRelative, digest: verified.DigestValue, policy: verified.VerificationPolicy})
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(ctx); err != nil {
		t.Fatal(err)
	}
	persisted, err := env.repo.GetOperation(ctx, op.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Phase != installation.PhaseCommitted {
		t.Fatalf("operation phase = %s, want COMMITTED", persisted.Phase)
	}
	plugin, err := env.repo.Get(ctx, op.PluginID)
	if err != nil {
		t.Fatal(err)
	}
	if plugin.Manifest.Version != op.TargetVersion {
		t.Fatalf("recovered Plugin version = %s, want %s", plugin.Manifest.Version, op.TargetVersion)
	}
	selected, err := env.repo.GetSelectedVersion(ctx, op.PluginID)
	if err != nil {
		t.Fatal(err)
	}
	if !selected.Selected || selected.ArtifactPath != finalRelative {
		t.Fatalf("unexpected selected version: %#v", selected)
	}
	versions, err := env.repo.ListInstalledVersions(ctx, op.PluginID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("installed version count = %d, want 1", len(versions))
	}
}
