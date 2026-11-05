package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

type testRepository struct {
	selected     installation.InstalledVersion
	selectedErr  error
	operation    installation.InstallationOperation
	phases       []installation.Phase
	committed    bool
	markedFailed string
	commitErr    error
}

func (r *testRepository) BeginOperation(_ context.Context, op installation.InstallationOperation) error {
	r.operation = op
	r.phases = append(r.phases, op.Phase)
	return nil
}
func (r *testRepository) RecordStaged(_ context.Context, _ installation.OperationID, _ installation.StagedArtifact) error {
	r.phases = append(r.phases, installation.PhaseStaged)
	return nil
}
func (r *testRepository) RecordVerified(_ context.Context, _ installation.OperationID, _ installation.VerifiedArtifact) error {
	r.phases = append(r.phases, installation.PhaseVerified)
	return nil
}
func (r *testRepository) RecordPromoted(_ context.Context, _ installation.OperationID, _ string) error {
	r.phases = append(r.phases, installation.PhasePromoted)
	return nil
}
func (r *testRepository) CommitInstallation(_ context.Context, _ installation.OperationID, _ domain.Plugin, _ installation.InstalledVersion) error {
	if r.commitErr != nil {
		return r.commitErr
	}
	r.committed = true
	r.phases = append(r.phases, installation.PhaseCommitted)
	return nil
}
func (r *testRepository) GetOperation(context.Context, installation.OperationID) (installation.InstallationOperation, error) {
	return installation.InstallationOperation{}, installation.ErrOperationNotFound
}
func (r *testRepository) ListIncompleteOperations(context.Context) ([]installation.InstallationOperation, error) {
	return nil, nil
}
func (r *testRepository) MarkRolledBack(context.Context, installation.OperationID, string) error {
	return nil
}
func (r *testRepository) MarkFailed(_ context.Context, _ installation.OperationID, code string) error {
	r.markedFailed = code
	return nil
}
func (r *testRepository) ListInstalledVersions(context.Context, domain.PluginID) ([]installation.InstalledVersion, error) {
	return nil, nil
}
func (r *testRepository) GetSelectedVersion(context.Context, domain.PluginID) (installation.InstalledVersion, error) {
	if r.selectedErr != nil {
		return installation.InstalledVersion{}, r.selectedErr
	}
	return r.selected, nil
}

type testIDs struct{}

func (testIDs) NewOperationID(context.Context) (installation.OperationID, error) {
	return "op-test-1", nil
}

type testAcquirer struct {
	err  error
	path string
}

func (a testAcquirer) Acquire(_ context.Context, _ string, target string) (installation.StagedArtifact, error) {
	if a.err != nil {
		return installation.StagedArtifact{}, a.err
	}
	if a.path != "" {
		return installation.StagedArtifact{StagingPath: a.path}, nil
	}
	return installation.StagedArtifact{StagingPath: target}, nil
}

type testInspector struct {
	plugin domain.Plugin
	err    error
}

func (i testInspector) Inspect(context.Context, string) (domain.Plugin, error) {
	return i.plugin.Clone(), i.err
}

type testVerifier struct {
	result installation.VerifiedArtifact
	err    error
}

func (v testVerifier) Verify(context.Context, string, domain.Plugin) (installation.VerifiedArtifact, error) {
	return v.result, v.err
}

type testPromoter struct{ err error }

func (p testPromoter) Promote(context.Context, string, string) error { return p.err }

func validPlugin() domain.Plugin {
	id := domain.PluginID("example.plugin")
	return domain.Plugin{
		ID:        id,
		Manifest:  domain.Manifest{PluginID: id, Name: "Example", Version: "1.2.0", Description: "Example plugin", Author: "PRIDE", License: "MIT", EntryPoint: "main", MinimumPRIDEVersion: "1.0.0"},
		Type:      domain.PluginTypePassive,
		State:     domain.PluginStateInstalled,
		Protected: domain.ProtectedMetadata{Capabilities: []domain.Capability{}},
		CreatedAt: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

func newTestService(t *testing.T, repo *testRepository, acquirer ArtifactAcquirer, inspector PackageInspector, verifier ArtifactVerifier, promoter ArtifactPromoter) *Service {
	t.Helper()
	service, err := NewService(repo, testIDs{}, acquirer, inspector, verifier, promoter)
	if err != nil {
		t.Fatal(err)
	}
	service.now = func() time.Time { return time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC) }
	return service
}

func TestInstallCoordinatesSuccessfulCommit(t *testing.T) {
	repo := &testRepository{selectedErr: installation.ErrVersionNotFound}
	plugin := validPlugin()
	manifest := `{"plugin":"opaque-to-application-layer"}`
	verified := installation.VerifiedArtifact{FinalArtifactPath: "installed/example.plugin/1.2.0", DigestAlgorithm: "sha256", DigestValue: "abc", VerificationPolicy: "policy-v1", ManifestJSON: manifest}
	service := newTestService(t, repo, testAcquirer{}, testInspector{plugin: plugin}, testVerifier{result: verified}, testPromoter{})
	err := service.Install(context.Background(), Request{PluginID: plugin.ID, Version: "1.2.0", SourceReference: "registry-release:opaque", Kind: installation.KindInstall})
	if err != nil {
		t.Fatal(err)
	}
	if !repo.committed {
		t.Fatal("expected atomic installation commit")
	}
	if len(repo.phases) != 5 || repo.phases[4] != installation.PhaseCommitted {
		t.Fatalf("unexpected phases: %#v", repo.phases)
	}
}

func TestInstallMarksPrePromotionFailure(t *testing.T) {
	repo := &testRepository{selectedErr: installation.ErrVersionNotFound}
	service := newTestService(t, repo, testAcquirer{err: errors.New("offline")}, testInspector{}, testVerifier{}, testPromoter{})
	err := service.Install(context.Background(), Request{PluginID: "example.plugin", Version: "1.2.0", SourceReference: "release:1.2.0", Kind: installation.KindInstall})
	if err == nil {
		t.Fatal("expected acquisition error")
	}
	if repo.markedFailed != "ARTIFACT_ACQUISITION_FAILED" {
		t.Fatalf("unexpected failure code: %q", repo.markedFailed)
	}
}

func TestUpdateRequiresExistingSelectedVersion(t *testing.T) {
	repo := &testRepository{selectedErr: installation.ErrVersionNotFound}
	service := newTestService(t, repo, testAcquirer{}, testInspector{}, testVerifier{}, testPromoter{})
	err := service.Install(context.Background(), Request{PluginID: "example.plugin", Version: "2.0.0", SourceReference: "release:2.0.0", Kind: installation.KindUpdate})
	if err == nil {
		t.Fatal("expected update to require an existing selected version")
	}
	if repo.operation.OperationID != "" {
		t.Fatal("operation must not start when update precondition fails")
	}
}

func TestCommitFailureLeavesPromotedOperationRecoverable(t *testing.T) {
	repo := &testRepository{selectedErr: installation.ErrVersionNotFound, commitErr: errors.New("database unavailable")}
	plugin := validPlugin()
	verified := installation.VerifiedArtifact{FinalArtifactPath: "installed/example.plugin/1.2.0", DigestAlgorithm: "sha256", DigestValue: "abc", VerificationPolicy: "policy-v1", ManifestJSON: `{"plugin":"opaque"}`}
	service := newTestService(t, repo, testAcquirer{}, testInspector{plugin: plugin}, testVerifier{result: verified}, testPromoter{})
	err := service.Install(context.Background(), Request{PluginID: plugin.ID, Version: "1.2.0", SourceReference: "release:1.2.0", Kind: installation.KindInstall})
	if err == nil {
		t.Fatal("expected commit failure")
	}
	if repo.markedFailed != "" {
		t.Fatalf("PROMOTED operation must remain recoverable, got failure code %q", repo.markedFailed)
	}
}
