package recovery

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
	"pride-plugin-module/installation/artifactverify"
	"pride-plugin-module/installation/zipinspect"
	"pride-plugin-module/registry"
)

const (
	failureInterrupted       = "RECOVERY_INTERRUPTED_BEFORE_PROMOTION"
	failureMissingDescriptor = "RECOVERY_MISSING_DESCRIPTOR_EVIDENCE"
	failureInvalidPackage    = "RECOVERY_PACKAGE_INVALID"
	failureInvalidArtifact   = "RECOVERY_ARTIFACT_INVALID"
)

type PackageInspector interface {
	Inspect(context.Context, string) (installation.InspectedPackage, error)
}

type ArtifactVerifier interface {
	Verify(context.Context, installation.StagedArtifact, installation.InspectedPackage, domain.Plugin) (installation.VerifiedArtifact, error)
}

type QuarantineStorage interface {
	QuarantineOperation(context.Context, string, string) error
}

// Coordinator reconciles the filesystem/database boundary left by interrupted
// installation attempts. It is safe to call repeatedly: terminal journal
// records are excluded by ListIncompleteOperations and PROMOTED records remain
// pending whenever a transient dependency or database commit fails.
type Coordinator struct {
	operations installation.Repository
	plugins    registry.Repository
	storage    QuarantineStorage
	inspector  PackageInspector
	verifier   ArtifactVerifier
	now        func() time.Time
}

func New(
	operations installation.Repository,
	plugins registry.Repository,
	storage QuarantineStorage,
	inspector PackageInspector,
	verifier ArtifactVerifier,
) (*Coordinator, error) {
	if operations == nil || plugins == nil || storage == nil || inspector == nil || verifier == nil {
		return nil, errors.New("installation recovery dependencies are required")
	}
	return &Coordinator{
		operations: operations,
		plugins:    plugins,
		storage:    storage,
		inspector:  inspector,
		verifier:   verifier,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// Recover processes the ordered incomplete-operation snapshot. It continues
// after an individual failure and returns the combined errors to the startup
// caller; callers must not enable plugin execution when recovery returns an
// error.
func (c *Coordinator) Recover(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	operations, err := c.operations.ListIncompleteOperations(ctx)
	if err != nil {
		return fmt.Errorf("list incomplete Plugin installation operations: %w", err)
	}
	var failures []error
	for _, operation := range operations {
		if err := ctx.Err(); err != nil {
			return errors.Join(append(failures, err)...)
		}
		if err := c.recoverOne(ctx, operation); err != nil {
			failures = append(failures, fmt.Errorf("recover operation %q: %w", operation.OperationID, err))
		}
	}
	return errors.Join(failures...)
}

func (c *Coordinator) recoverOne(ctx context.Context, operation installation.InstallationOperation) error {
	switch operation.Phase {
	case installation.PhasePreparing, installation.PhaseStaged, installation.PhaseVerified:
		// Promotion has not been durably recorded. Never infer that a package
		// is safe to install from a partial staging workspace.
		if err := c.storage.QuarantineOperation(ctx, operation.StagingPath, string(operation.OperationID)); err != nil {
			return fmt.Errorf("quarantine incomplete staging workspace: %w", err)
		}
		if err := c.operations.MarkRolledBack(ctx, operation.OperationID, failureInterrupted); err != nil {
			return fmt.Errorf("record interrupted operation rollback: %w", err)
		}
		return nil
	case installation.PhasePromoted:
		return c.recoverPromoted(ctx, operation)
	default:
		return fmt.Errorf("unsupported incomplete operation phase %q", operation.Phase)
	}
}

func (c *Coordinator) recoverPromoted(ctx context.Context, operation installation.InstallationOperation) error {
	quarantineAndFail := func(code string, cause error) error {
		if err := c.storage.QuarantineOperation(ctx, operation.FinalArtifactPath, string(operation.OperationID)); err != nil {
			return errors.Join(cause, fmt.Errorf("quarantine untrusted promoted artifact: %w", err))
		}
		if err := c.operations.MarkFailed(ctx, operation.OperationID, code); err != nil {
			return errors.Join(cause, fmt.Errorf("record failed recovery operation: %w", err))
		}
		return nil
	}

	// The exact signed descriptor bytes are required. Parsed or reconstructed
	// descriptor data cannot stand in for the original signature evidence.
	if operation.DescriptorJSON == "" {
		return quarantineAndFail(failureMissingDescriptor, artifactverify.ErrInvalidDescriptor)
	}

	inspected, err := c.inspector.Inspect(ctx, operation.FinalArtifactPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return quarantineAndFail(failureInvalidPackage, err)
		}
		if errors.Is(err, zipinspect.ErrInvalidArchive) ||
			errors.Is(err, zipinspect.ErrInvalidManifest) ||
			errors.Is(err, zipinspect.ErrUnsafeEntry) ||
			errors.Is(err, zipinspect.ErrUnsupportedZIP64) {
			return quarantineAndFail(failureInvalidPackage, err)
		}
		return fmt.Errorf("inspect promoted Plugin package: %w", err)
	}
	if inspected.Manifest.PluginID != operation.PluginID || inspected.Manifest.Version != operation.TargetVersion {
		return quarantineAndFail(failureInvalidPackage, installation.ErrRecordMismatch)
	}

	// Platform metadata is extracted from the exact descriptor bytes persisted
	// in the operation journal. It remains untrusted until Verify succeeds.
	plugin := domain.Plugin{
		ID:        inspected.Manifest.PluginID,
		Manifest:  inspected.Manifest,
		State:     domain.PluginStateInstalled,
		CreatedAt: operation.CreatedAt,
		UpdatedAt: c.now().UTC(),
	}
	if operation.Kind == installation.KindUpdate {
		existing, getErr := c.plugins.Get(ctx, operation.PluginID)
		if getErr != nil {
			return fmt.Errorf("load existing Plugin aggregate for update recovery: %w", getErr)
		}
		plugin.CreatedAt = existing.CreatedAt
	}
	staged := installation.StagedArtifact{
		StagingPath:    operation.FinalArtifactPath,
		DescriptorJSON: []byte(operation.DescriptorJSON),
	}
	verified, err := c.verifier.Verify(ctx, staged, inspected, plugin.Clone())
	if err != nil {
		// Only cryptographic/descriptor failures are definitive. In particular,
		// ErrUntrustedKey may wrap a temporary trust-transport failure and must
		// remain retryable.
		if errors.Is(err, artifactverify.ErrInvalidDescriptor) ||
			errors.Is(err, artifactverify.ErrDigestMismatch) ||
			errors.Is(err, artifactverify.ErrSignatureInvalid) {
			return quarantineAndFail(failureInvalidArtifact, err)
		}
		return fmt.Errorf("reverify promoted Plugin artifact: %w", err)
	}
	plugin.Type = verified.PlatformMetadata.Type
	plugin.Protected = verified.PlatformMetadata.Protected
	if err := plugin.Validate(); err != nil {
		return quarantineAndFail(failureInvalidArtifact, err)
	}

	if verified.FinalArtifactPath != operation.FinalArtifactPath ||
		verified.DigestAlgorithm != operation.DigestAlgorithm ||
		verified.DigestValue != operation.DigestValue ||
		verified.VerificationPolicy != operation.VerificationPolicy ||
		verified.ManifestJSON != operation.ManifestJSON {
		return quarantineAndFail(failureInvalidArtifact, installation.ErrRecordMismatch)
	}

	version := installation.InstalledVersion{
		PluginID:                plugin.ID,
		Version:                 plugin.Manifest.Version,
		ArtifactPath:            verified.FinalArtifactPath,
		SourceReference:         operation.SourceReference,
		DigestAlgorithm:         verified.DigestAlgorithm,
		DigestValue:             verified.DigestValue,
		VerificationPolicy:      verified.VerificationPolicy,
		VerificationEvidenceRef: verified.VerificationEvidenceRef,
		ManifestJSON:            verified.ManifestJSON,
		InstalledAt:             c.now().UTC(),
		Selected:                true,
	}
	if err := c.operations.CommitInstallation(ctx, operation.OperationID, plugin, version); err != nil {
		// Keep PROMOTED durable and retryable. CommitInstallation is atomic and
		// must not alter the previously selected version on failure.
		return fmt.Errorf("commit recovered Plugin installation: %w", err)
	}
	return nil
}
