package recovery

import (
	"context"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
	"pride-plugin-module/registry"
)

type recoveryRepository struct {
	operations      []installation.InstallationOperation
	rolledBack      string
	failed          string
	committed       bool
	committedPlugin domain.Plugin
	commitErr       error
}

func (r *recoveryRepository) BeginOperation(context.Context, installation.InstallationOperation) error {
	return nil
}
func (r *recoveryRepository) RecordStaged(context.Context, installation.OperationID, installation.StagedArtifact) error {
	return nil
}
func (r *recoveryRepository) RecordVerified(context.Context, installation.OperationID, installation.VerifiedArtifact) error {
	return nil
}
func (r *recoveryRepository) RecordPromoted(context.Context, installation.OperationID, string) error {
	return nil
}
func (r *recoveryRepository) CommitInstallation(_ context.Context, _ installation.OperationID, plugin domain.Plugin, _ installation.InstalledVersion) error {
	if r.commitErr != nil {
		return r.commitErr
	}
	r.committed = true
	r.committedPlugin = plugin.Clone()
	return nil
}
func (r *recoveryRepository) GetOperation(context.Context, installation.OperationID) (installation.InstallationOperation, error) {
	return installation.InstallationOperation{}, installation.ErrOperationNotFound
}
func (r *recoveryRepository) ListIncompleteOperations(context.Context) ([]installation.InstallationOperation, error) {
	return append([]installation.InstallationOperation(nil), r.operations...), nil
}
func (r *recoveryRepository) MarkRolledBack(_ context.Context, _ installation.OperationID, code string) error {
	r.rolledBack = code
	return nil
}
func (r *recoveryRepository) MarkFailed(_ context.Context, _ installation.OperationID, code string) error {
	r.failed = code
	return nil
}
func (r *recoveryRepository) ListInstalledVersions(context.Context, domain.PluginID) ([]installation.InstalledVersion, error) {
	return nil, nil
}
func (r *recoveryRepository) GetSelectedVersion(context.Context, domain.PluginID) (installation.InstalledVersion, error) {
	return installation.InstalledVersion{}, installation.ErrVersionNotFound
}

type recoveryPluginRepository struct {
	plugin domain.Plugin
	err    error
}

func (r recoveryPluginRepository) Create(context.Context, domain.Plugin) error { return nil }
func (r recoveryPluginRepository) Get(context.Context, domain.PluginID) (domain.Plugin, error) {
	return r.plugin, r.err
}
func (r recoveryPluginRepository) List(context.Context) ([]domain.Plugin, error) { return nil, nil }

type recoveryStorage struct {
	source string
	id     string
	err    error
	calls  int
}

func (s *recoveryStorage) QuarantineOperation(_ context.Context, source, id string) error {
	s.calls++
	s.source, s.id = source, id
	return s.err
}

type recoveryInspector struct {
	result installation.InspectedPackage
	err    error
	path   string
}

func (i *recoveryInspector) Inspect(_ context.Context, path string) (installation.InspectedPackage, error) {
	i.path = path
	return i.result, i.err
}

type recoveryVerifier struct {
	result installation.VerifiedArtifact
	err    error
	input  installation.StagedArtifact
}

func (v *recoveryVerifier) Verify(_ context.Context, staged installation.StagedArtifact, _ installation.InspectedPackage, _ domain.Plugin) (installation.VerifiedArtifact, error) {
	v.input = staged
	return v.result, v.err
}

func recoveryManifest() domain.Manifest {
	return domain.Manifest{
		PluginID: "example.plugin", Name: "Example", Version: "1.2.0",
		Description: "Example plugin", Author: "PRIDE", License: "MIT",
		EntryPoint: "main", MinimumPRIDEVersion: "1.0.0",
	}
}

func TestRecoverPrePromotionQuarantinesAndRollsBack(t *testing.T) {
	op := installation.InstallationOperation{
		OperationID: "op-prepared", PluginID: "example.plugin", TargetVersion: "1.2.0",
		Kind: installation.KindInstall, Phase: installation.PhaseStaged,
		StagingPath: "temporary/op-prepared", CreatedAt: time.Now().UTC(),
	}
	repo := &recoveryRepository{operations: []installation.InstallationOperation{op}}
	storage := &recoveryStorage{}
	coordinator, err := New(repo, recoveryPluginRepository{}, storage, &recoveryInspector{}, &recoveryVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if storage.calls != 1 || storage.source != op.StagingPath || storage.id != string(op.OperationID) {
		t.Fatalf("unexpected quarantine request: %#v", storage)
	}
	if repo.rolledBack != failureInterrupted {
		t.Fatalf("rollback code = %q", repo.rolledBack)
	}
}

func TestRecoverPromotedReverifiesAndCommits(t *testing.T) {
	manifest := recoveryManifest()
	now := time.Date(2026, 11, 20, 10, 0, 0, 0, time.UTC)
	op := installation.InstallationOperation{
		OperationID: "op-promoted", PluginID: manifest.PluginID, TargetVersion: manifest.Version,
		Kind: installation.KindInstall, Phase: installation.PhasePromoted,
		StagingPath: "temporary/op-promoted", FinalArtifactPath: "installed/example.plugin/1.2.0",
		SourceReference: "registry:release:1.2.0", DescriptorJSON: "{\"signed\":true}",
		DigestAlgorithm: "sha-256", DigestValue: "digest", VerificationPolicy: "policy-v1",
		ManifestJSON: "{\"manifest\":\"raw\"}", CreatedAt: now, UpdatedAt: now,
	}
	verified := installation.VerifiedArtifact{
		FinalArtifactPath: op.FinalArtifactPath, DigestAlgorithm: op.DigestAlgorithm,
		DigestValue: op.DigestValue, VerificationPolicy: op.VerificationPolicy,
		VerificationEvidenceRef: "marketplace-key:key-1", ManifestJSON: op.ManifestJSON,
		PlatformMetadata: installation.PlatformMetadata{Type: domain.PluginTypePassive, Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}}},
	}
	repo := &recoveryRepository{operations: []installation.InstallationOperation{op}}
	inspector := &recoveryInspector{result: installation.InspectedPackage{Manifest: manifest, ManifestJSON: []byte(op.ManifestJSON)}}
	verifier := &recoveryVerifier{result: verified}
	coordinator, err := New(repo, recoveryPluginRepository{}, &recoveryStorage{}, inspector, verifier)
	if err != nil {
		t.Fatal(err)
	}
	coordinator.now = func() time.Time { return now.Add(time.Minute) }
	if err := coordinator.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if inspector.path != op.FinalArtifactPath {
		t.Fatalf("inspected path = %q", inspector.path)
	}
	if verifier.input.StagingPath != op.FinalArtifactPath || string(verifier.input.DescriptorJSON) != op.DescriptorJSON {
		t.Fatalf("verifier did not receive final artifact and exact descriptor: %#v", verifier.input)
	}
	if !repo.committed {
		t.Fatal("PROMOTED operation was not committed")
	}
	if repo.committedPlugin.Type != domain.PluginTypePassive {
		t.Fatalf("recovered Plugin did not use verified descriptor metadata: %#v", repo.committedPlugin)
	}
}

func TestRecoverPromotedTransientVerifierFailureRemainsRetryable(t *testing.T) {
	manifest := recoveryManifest()
	op := installation.InstallationOperation{
		OperationID: "op-retry", PluginID: manifest.PluginID, TargetVersion: manifest.Version,
		Kind: installation.KindInstall, Phase: installation.PhasePromoted,
		FinalArtifactPath: "installed/example.plugin/1.2.0", DescriptorJSON: "{\"signed\":true}",
		CreatedAt: time.Now().UTC(),
	}
	repo := &recoveryRepository{operations: []installation.InstallationOperation{op}}
	storage := &recoveryStorage{}
	inspector := &recoveryInspector{result: installation.InspectedPackage{Manifest: manifest}}
	verifier := &recoveryVerifier{err: errors.New("trust endpoint unavailable")}
	coordinator, err := New(repo, recoveryPluginRepository{}, storage, inspector, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(context.Background()); err == nil {
		t.Fatal("expected retryable recovery error")
	}
	if storage.calls != 0 || repo.failed != "" || repo.committed {
		t.Fatalf("transient verification failure mutated recovery state: storage=%d failed=%q committed=%v", storage.calls, repo.failed, repo.committed)
	}
}

func TestRecoverPromotedWithoutDescriptorFailsClosed(t *testing.T) {
	manifest := recoveryManifest()
	op := installation.InstallationOperation{
		OperationID: "op-no-evidence", PluginID: manifest.PluginID, TargetVersion: manifest.Version,
		Kind: installation.KindInstall, Phase: installation.PhasePromoted,
		FinalArtifactPath: "installed/example.plugin/1.2.0", CreatedAt: time.Now().UTC(),
	}
	repo := &recoveryRepository{operations: []installation.InstallationOperation{op}}
	storage := &recoveryStorage{}
	coordinator, err := New(repo, recoveryPluginRepository{}, storage, &recoveryInspector{}, &recoveryVerifier{})
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if storage.calls != 1 || repo.failed != failureMissingDescriptor || repo.committed {
		t.Fatalf("legacy operation did not fail closed: storage=%d failed=%q committed=%v", storage.calls, repo.failed, repo.committed)
	}
}

func TestRecoverUpdatePreservesPluginCreationTime(t *testing.T) {
	manifest := recoveryManifest()
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	op := installation.InstallationOperation{
		OperationID: "op-update-recovery", PluginID: manifest.PluginID, TargetVersion: manifest.Version,
		Kind: installation.KindUpdate, Phase: installation.PhasePromoted,
		FinalArtifactPath: "installed/example.plugin/1.2.0", DescriptorJSON: "{\"signed\":true}",
		DigestAlgorithm: "sha-256", DigestValue: "digest", VerificationPolicy: "policy-v1",
		ManifestJSON: "{\"manifest\":\"raw\"}", CreatedAt: created,
	}
	plugin := domain.Plugin{
		ID: manifest.PluginID, Manifest: manifest, Type: domain.PluginTypePassive,
		State: domain.PluginStateInstalled, Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}},
		CreatedAt: created.Add(-time.Hour),
	}
	verified := installation.VerifiedArtifact{
		FinalArtifactPath: op.FinalArtifactPath, DigestAlgorithm: op.DigestAlgorithm,
		DigestValue: op.DigestValue, VerificationPolicy: op.VerificationPolicy, ManifestJSON: op.ManifestJSON,
		PlatformMetadata: installation.PlatformMetadata{Type: domain.PluginTypePassive, Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}}},
	}
	repo := &recoveryRepository{operations: []installation.InstallationOperation{op}}
	plugins := recoveryPluginRepository{plugin: plugin}
	inspector := &recoveryInspector{result: installation.InspectedPackage{Manifest: manifest, ManifestJSON: []byte(op.ManifestJSON)}}
	verifier := &recoveryVerifier{result: verified}
	coordinator, err := New(repo, plugins, &recoveryStorage{}, inspector, verifier)
	if err != nil {
		t.Fatal(err)
	}
	if err := coordinator.Recover(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !repo.committed {
		t.Fatal("update recovery did not commit")
	}
}

// Keep the compile-time dependency explicit: the local Plugin repository is
// distinct from the Marketplace descriptor resolver.
var _ registry.Repository = recoveryPluginRepository{}
