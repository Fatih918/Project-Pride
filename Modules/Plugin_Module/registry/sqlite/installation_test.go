package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

func validOperation(id, pluginID, version string, now time.Time) installation.InstallationOperation {
	return installation.InstallationOperation{
		OperationID: installation.OperationID(id), PluginID: domain.PluginID(pluginID),
		TargetVersion: version, Kind: installation.KindInstall, Phase: installation.PhasePreparing,
		SourceReference: "registry://release/" + pluginID + "/" + version,
		CreatedAt:       now, UpdatedAt: now,
	}
}

func completeInstallation(t *testing.T, repository *Repository, operation installation.InstallationOperation, plugin domain.Plugin, digest string) {
	t.Helper()
	ctx := context.Background()
	if err := repository.BeginOperation(ctx, operation); err != nil {
		t.Fatal(err)
	}
	staging := "temporary/" + string(operation.OperationID)
	finalPath := "installed/" + string(plugin.ID) + "/" + plugin.Manifest.Version
	if err := repository.RecordStaged(ctx, operation.OperationID, installation.StagedArtifact{StagingPath: staging}); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(plugin.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	verified := installation.VerifiedArtifact{FinalArtifactPath: finalPath, DigestAlgorithm: "sha256", DigestValue: digest, VerificationPolicy: "marketplace-policy-v1", ManifestJSON: string(manifest)}
	if err := repository.RecordVerified(ctx, operation.OperationID, verified); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordPromoted(ctx, operation.OperationID, finalPath); err != nil {
		t.Fatal(err)
	}
	version := installation.InstalledVersion{PluginID: plugin.ID, Version: plugin.Manifest.Version, ArtifactPath: finalPath,
		SourceReference: operation.SourceReference, DigestAlgorithm: verified.DigestAlgorithm, DigestValue: digest,
		VerificationPolicy: verified.VerificationPolicy, ManifestJSON: string(manifest), InstalledAt: time.Now().UTC(), Selected: true}
	if err := repository.CommitInstallation(ctx, operation.OperationID, plugin, version); err != nil {
		t.Fatal(err)
	}
}

func TestInstallationCommitAndReopen(t *testing.T) {
	repository, path := openTestRepository(t)
	plugin := testPlugin("plugin.install")
	op := validOperation("op-install-1", string(plugin.ID), plugin.Manifest.Version, time.Now().UTC())
	completeInstallation(t, repository, op, plugin, "digest-one")
	selected, err := repository.GetSelectedVersion(context.Background(), plugin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !selected.Selected || selected.Version != plugin.Manifest.Version {
		t.Fatalf("unexpected selected version: %#v", selected)
	}
	persistedPlugin, err := repository.Get(context.Background(), plugin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedPlugin.Manifest.Version != plugin.Manifest.Version {
		t.Fatal("discovery Plugin row was not committed")
	}
	persistedOperation, err := repository.GetOperation(context.Background(), op.OperationID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedOperation.Phase != installation.PhaseCommitted {
		t.Fatalf("phase = %s", persistedOperation.Phase)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	versions, err := reopened.ListInstalledVersions(context.Background(), plugin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 || versions[0].DigestValue != "digest-one" {
		t.Fatalf("unexpected versions after reopen: %#v", versions)
	}
}

func TestInstallationUpdateSwitchesSelectionOnlyAtCommit(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx := context.Background()
	first := testPlugin("plugin.update")
	op1 := validOperation("op-update-1", string(first.ID), first.Manifest.Version, time.Now().UTC())
	completeInstallation(t, repository, op1, first, "digest-v1")
	second := first.Clone()
	second.Manifest.Version = "2.0.0"
	op2 := validOperation("op-update-2", string(second.ID), second.Manifest.Version, time.Now().UTC())
	op2.Kind = installation.KindUpdate
	if err := repository.BeginOperation(ctx, op2); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordStaged(ctx, op2.OperationID, installation.StagedArtifact{StagingPath: "temporary/op-update-2"}); err != nil {
		t.Fatal(err)
	}
	manifest, err := json.Marshal(second.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	finalPath := "installed/" + string(second.ID) + "/2.0.0"
	verified := installation.VerifiedArtifact{FinalArtifactPath: finalPath, DigestAlgorithm: "sha256", DigestValue: "digest-v2", VerificationPolicy: "marketplace-policy-v1", ManifestJSON: string(manifest)}
	if err := repository.RecordVerified(ctx, op2.OperationID, verified); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordPromoted(ctx, op2.OperationID, finalPath); err != nil {
		t.Fatal(err)
	}
	selected, err := repository.GetSelectedVersion(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Version != "1.2.3" {
		t.Fatalf("selection changed before commit: %s", selected.Version)
	}
	version := installation.InstalledVersion{PluginID: second.ID, Version: second.Manifest.Version, ArtifactPath: finalPath,
		SourceReference: op2.SourceReference, DigestAlgorithm: verified.DigestAlgorithm, DigestValue: verified.DigestValue,
		VerificationPolicy: verified.VerificationPolicy, ManifestJSON: string(manifest), InstalledAt: time.Now().UTC(), Selected: true}
	if err := repository.CommitInstallation(ctx, op2.OperationID, second, version); err != nil {
		t.Fatal(err)
	}
	selected, err = repository.GetSelectedVersion(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Version != "2.0.0" {
		t.Fatalf("selected version = %s", selected.Version)
	}
	versions, err := repository.ListInstalledVersions(ctx, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Fatalf("installed version count = %d", len(versions))
	}
}

func TestInstallationCommitMismatchDoesNotChangeSelection(t *testing.T) {
	repository, _ := openTestRepository(t)
	first := testPlugin("plugin.atomic")
	op1 := validOperation("op-atomic-1", string(first.ID), first.Manifest.Version, time.Now().UTC())
	completeInstallation(t, repository, op1, first, "digest-v1")
	second := first.Clone()
	second.Manifest.Version = "2.0.0"
	op2 := validOperation("op-atomic-2", string(second.ID), second.Manifest.Version, time.Now().UTC())
	if err := repository.BeginOperation(context.Background(), op2); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordStaged(context.Background(), op2.OperationID, installation.StagedArtifact{StagingPath: "temporary/op-atomic-2"}); err != nil {
		t.Fatal(err)
	}
	manifest, _ := json.Marshal(second.Manifest)
	finalPath := "installed/" + string(second.ID) + "/2.0.0"
	verified := installation.VerifiedArtifact{FinalArtifactPath: finalPath, DigestAlgorithm: "sha256", DigestValue: "digest-v2", VerificationPolicy: "policy-v1", ManifestJSON: string(manifest)}
	if err := repository.RecordVerified(context.Background(), op2.OperationID, verified); err != nil {
		t.Fatal(err)
	}
	if err := repository.RecordPromoted(context.Background(), op2.OperationID, finalPath); err != nil {
		t.Fatal(err)
	}
	version := installation.InstalledVersion{PluginID: second.ID, Version: second.Manifest.Version, ArtifactPath: finalPath,
		SourceReference: op2.SourceReference, DigestAlgorithm: "sha256", DigestValue: "wrong-digest",
		VerificationPolicy: "policy-v1", ManifestJSON: string(manifest), InstalledAt: time.Now().UTC(), Selected: true}
	if err := repository.CommitInstallation(context.Background(), op2.OperationID, second, version); !errors.Is(err, installation.ErrRecordMismatch) {
		t.Fatalf("commit mismatch error = %v", err)
	}
	selected, err := repository.GetSelectedVersion(context.Background(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if selected.Version != "1.2.3" {
		t.Fatalf("failed commit changed selected version to %s", selected.Version)
	}
	versions, err := repository.ListInstalledVersions(context.Background(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("failed commit left partial version record: %d", len(versions))
	}
}

func TestInstallationRejectsInvalidTransitionsAndConcurrentOperation(t *testing.T) {
	repository, _ := openTestRepository(t)
	now := time.Now().UTC()
	op := validOperation("op-state-1", "plugin.state", "1.0.0", now)
	if err := repository.BeginOperation(context.Background(), op); err != nil {
		t.Fatal(err)
	}
	if err := repository.BeginOperation(context.Background(), validOperation("op-state-2", "plugin.state", "2.0.0", now)); !errors.Is(err, installation.ErrOperationConflict) {
		t.Fatalf("active operation conflict = %v", err)
	}
	if err := repository.RecordPromoted(context.Background(), op.OperationID, "installed/plugin.state/1.0.0"); !errors.Is(err, installation.ErrInvalidTransition) {
		t.Fatalf("invalid transition = %v", err)
	}
	if err := repository.RecordStaged(context.Background(), op.OperationID, installation.StagedArtifact{StagingPath: "../escape"}); !errors.Is(err, installation.ErrInvalidRecord) {
		t.Fatalf("path traversal error = %v", err)
	}
}

func TestInstallationIncompleteOperationsAreOrdered(t *testing.T) {
	repository, _ := openTestRepository(t)
	base := time.Date(2026, 11, 5, 1, 0, 0, 0, time.UTC)
	for _, id := range []string{"op-b", "op-a"} {
		op := validOperation(id, "plugin."+id, "1.0.0", base)
		if err := repository.BeginOperation(context.Background(), op); err != nil {
			t.Fatal(err)
		}
	}
	ops, err := repository.ListIncompleteOperations(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(ops) != 2 || ops[0].OperationID != "op-a" || ops[1].OperationID != "op-b" {
		t.Fatalf("unexpected incomplete operation order: %#v", ops)
	}
}

func TestInstallationSchemaMigrationPreservesExistingPlugins(t *testing.T) {
	path := filepath.Join(t.TempDir(), "legacy.sqlite")
	repository, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	plugin := testPlugin("plugin.migration")
	if err := repository.Create(context.Background(), plugin); err != nil {
		t.Fatal(err)
	}
	if err := repository.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := New(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	got, err := reopened.Get(context.Background(), plugin.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Manifest.Name != plugin.Manifest.Name {
		t.Fatal("existing Plugin row was not preserved")
	}
	var version int
	if err := reopened.db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("schema version = %d, want 2", version)
	}
}

func TestInstallationContextCancellation(t *testing.T) {
	repository, _ := openTestRepository(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := repository.BeginOperation(ctx, validOperation("op-cancel", "plugin.cancel", "1.0.0", time.Now().UTC())); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled BeginOperation = %v", err)
	}
	if _, err := repository.ListIncompleteOperations(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ListIncompleteOperations = %v", err)
	}
	if _, err := repository.ListInstalledVersions(ctx, "plugin.cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled ListInstalledVersions = %v", err)
	}
	if _, err := repository.GetSelectedVersion(ctx, "plugin.cancel"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled GetSelectedVersion = %v", err)
	}
}
