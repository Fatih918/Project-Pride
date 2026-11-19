package application

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/installation"
)

var (
	ErrInvalidRequest = errors.New("invalid installation request")
	ErrMissingPort    = errors.New("installation port is not configured")
	ErrPluginMismatch = errors.New("inspected plugin does not match installation request")
)

// Request identifies an immutable release selected by an upstream Registry
// adapter. The application service deliberately treats SourceReference as
// opaque; it does not parse Registry documents or construct download URLs.
type Request struct {
	PluginID        domain.PluginID
	Version         string
	SourceReference string
	Kind            installation.Kind
}

// IDGenerator supplies operation IDs. Implementations must return a value
// accepted as one safe storage path component.
type IDGenerator interface {
	NewOperationID(context.Context) (installation.OperationID, error)
}

// ArtifactAcquirer obtains the artifact into the supplied operation staging
// directory. stagingPath is canonical and relative to Plugin_Storage; the
// adapter must resolve it through the storage adapter, never via the cwd.
// Network and Registry-specific behavior belongs in its adapter.
// OperationStorage creates the durable operation-scoped staging directory.
// It returns a canonical path relative to Plugin_Storage, never an OS path.
type OperationStorage interface {
	CreateTemporaryOperationDirectory(context.Context, string) (string, error)
}

type ArtifactAcquirer interface {
	Acquire(context.Context, string, string) (installation.StagedArtifact, error)
}

// PackageInspector validates package structure and returns the SDK-neutral
// Plugin aggregate. It owns Manifest decoding; this service assumes no JSON
// property names or package layout.
type PackageInspector interface {
	Inspect(context.Context, string) (installation.InspectedPackage, error)
}

// PackageExtractor materializes only validated package payload files inside
// the same operation-scoped staging workspace.
type PackageExtractor interface {
	Extract(context.Context, string) (string, error)
}

// PlatformMetadataResolver supplies Marketplace/Registry-owned metadata. The
// package Manifest is never allowed to authoritatively set these fields.
type PlatformMetadataResolver interface {
	Resolve(context.Context, domain.PluginID, string) (installation.PlatformMetadata, error)
}

// ArtifactVerifier verifies integrity and authenticity under an explicitly
// configured policy. The service does not choose algorithms or trust roots.
type ArtifactVerifier interface {
	Verify(context.Context, installation.StagedArtifact, installation.InspectedPackage, domain.Plugin) (installation.VerifiedArtifact, error)
}

// ArtifactPromoter atomically promotes a verified staging directory to its
// immutable final path. Both paths are canonical Plugin_Storage-relative
// values and must be resolved through the storage adapter. It must not
// overwrite an existing version directory.
type ArtifactPromoter interface {
	Promote(context.Context, string, string) error
}

type Service struct {
	repository installation.Repository
	ids        IDGenerator
	storage    OperationStorage
	acquirer   ArtifactAcquirer
	inspector  PackageInspector
	extractor  PackageExtractor
	metadata   PlatformMetadataResolver
	verifier   ArtifactVerifier
	promoter   ArtifactPromoter
	now        func() time.Time
}

func NewService(
	repository installation.Repository,
	ids IDGenerator,
	storage OperationStorage,
	acquirer ArtifactAcquirer,
	inspector PackageInspector,
	extractor PackageExtractor,
	metadata PlatformMetadataResolver,
	verifier ArtifactVerifier,
	promoter ArtifactPromoter,
) (*Service, error) {
	if repository == nil || ids == nil || storage == nil || acquirer == nil || inspector == nil || extractor == nil || metadata == nil || verifier == nil || promoter == nil {
		return nil, ErrMissingPort
	}
	return &Service{
		repository: repository,
		ids:        ids,
		storage:    storage,
		acquirer:   acquirer,
		inspector:  inspector,
		extractor:  extractor,
		metadata:   metadata,
		verifier:   verifier,
		promoter:   promoter,
		now:        func() time.Time { return time.Now().UTC() },
	}, nil
}

// Install coordinates one install or update attempt. It intentionally leaves
// PROMOTED operations incomplete if the final database commit fails: startup
// recovery must reconcile the already-promoted filesystem artifact rather
// than incorrectly marking it failed or deleting the previous selected version.
func (s *Service) Install(ctx context.Context, request Request) error {
	if err := validateRequest(request); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	selected, selectedErr := s.repository.GetSelectedVersion(ctx, request.PluginID)
	switch request.Kind {
	case installation.KindInstall:
		if selectedErr == nil {
			return fmt.Errorf("%w: plugin %q already has selected version %q", installation.ErrOperationConflict, request.PluginID, selected.Version)
		}
		if !errors.Is(selectedErr, installation.ErrVersionNotFound) {
			return fmt.Errorf("read selected Plugin version: %w", selectedErr)
		}
	case installation.KindUpdate:
		if selectedErr != nil {
			return fmt.Errorf("update requires an installed Plugin: %w", selectedErr)
		}
		if selected.Version == request.Version {
			return fmt.Errorf("%w: target version is already selected", ErrInvalidRequest)
		}
	}

	id, err := s.ids.NewOperationID(ctx)
	if err != nil {
		return fmt.Errorf("generate installation operation ID: %w", err)
	}
	if strings.TrimSpace(string(id)) == "" || strings.ContainsAny(string(id), "/\\:") || id == "." || id == ".." {
		return fmt.Errorf("%w: ID generator returned an unsafe operation ID", ErrInvalidRequest)
	}

	now := s.now().UTC()
	stagingPath := "temporary/" + string(id)
	operation := installation.InstallationOperation{
		OperationID:     id,
		PluginID:        request.PluginID,
		TargetVersion:   request.Version,
		Kind:            request.Kind,
		Phase:           installation.PhasePreparing,
		StagingPath:     stagingPath,
		SourceReference: request.SourceReference,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := s.repository.BeginOperation(ctx, operation); err != nil {
		return fmt.Errorf("begin installation operation: %w", err)
	}

	fail := func(stage string, cause error) error {
		if markErr := s.repository.MarkFailed(context.WithoutCancel(ctx), id, failureCode(stage)); markErr != nil {
			return errors.Join(fmt.Errorf("%s: %w", stage, cause), fmt.Errorf("persist failed operation: %w", markErr))
		}
		return fmt.Errorf("%s: %w", stage, cause)
	}

	createdStagingPath, err := s.storage.CreateTemporaryOperationDirectory(ctx, string(id))
	if err != nil {
		return fail("create Plugin staging directory", err)
	}
	if createdStagingPath != stagingPath {
		return fail("validate staging location", installation.ErrRecordMismatch)
	}

	staged, err := s.acquirer.Acquire(ctx, request.SourceReference, stagingPath)
	if err != nil {
		return fail("acquire Plugin artifact", err)
	}
	if staged.StagingPath != stagingPath {
		return fail("validate staging location", installation.ErrRecordMismatch)
	}
	if err := s.repository.RecordStaged(ctx, id, staged); err != nil {
		return fail("record staged artifact", err)
	}

	inspected, err := s.inspector.Inspect(ctx, staged.StagingPath)
	if err != nil {
		return fail("inspect Plugin package", err)
	}
	if inspected.Manifest.PluginID != request.PluginID || inspected.Manifest.Version != request.Version {
		return fail("match inspected Plugin identity", ErrPluginMismatch)
	}
	platformMetadata, err := s.metadata.Resolve(ctx, request.PluginID, request.Version)
	if err != nil {
		return fail("resolve platform Plugin metadata", err)
	}
	plugin := domain.Plugin{
		ID:        inspected.Manifest.PluginID,
		Manifest:  inspected.Manifest,
		Type:      platformMetadata.Type,
		State:     domain.PluginStateInstalled,
		Protected: platformMetadata.Protected,
		CreatedAt: now,
		UpdatedAt: s.now().UTC(),
	}
	if err := plugin.Validate(); err != nil {
		return fail("validate assembled Plugin", err)
	}
	// Verify the exact staged ZIP and signed descriptor before extracting
	// developer-controlled payload into the operation workspace.
	verified, err := s.verifier.Verify(ctx, staged, inspected, plugin.Clone())
	if err != nil {
		return fail("verify Plugin artifact", err)
	}

	extractedPath, err := s.extractor.Extract(ctx, staged.StagingPath)
	if err != nil {
		return fail("extract Plugin package", err)
	}
	if extractedPath != staged.StagingPath+"/package" {
		return fail("validate extracted package location", installation.ErrRecordMismatch)
	}

	// Re-hash and re-verify after extraction so a changed staged ZIP cannot be
	// promoted merely because an earlier verification succeeded.
	reverified, err := s.verifier.Verify(ctx, staged, inspected, plugin.Clone())
	if err != nil {
		return fail("reverify Plugin artifact after extraction", err)
	}
	if reverified != verified {
		return fail("compare Plugin artifact verification evidence", installation.ErrRecordMismatch)
	}
	verified = reverified
	if err := s.repository.RecordVerified(ctx, id, verified); err != nil {
		return fail("record verified artifact", err)
	}
	if err := s.promoter.Promote(ctx, staged.StagingPath, verified.FinalArtifactPath); err != nil {
		return fail("promote verified artifact", err)
	}
	if err := s.repository.RecordPromoted(ctx, id, verified.FinalArtifactPath); err != nil {
		// The filesystem may already contain the promoted artifact. Leave the
		// journal recoverable instead of recording a false terminal failure.
		return fmt.Errorf("record promoted artifact (operation remains recoverable): %w", err)
	}

	version := installation.InstalledVersion{
		PluginID:                plugin.ID,
		Version:                 plugin.Manifest.Version,
		ArtifactPath:            verified.FinalArtifactPath,
		SourceReference:         request.SourceReference,
		DigestAlgorithm:         verified.DigestAlgorithm,
		DigestValue:             verified.DigestValue,
		VerificationPolicy:      verified.VerificationPolicy,
		VerificationEvidenceRef: verified.VerificationEvidenceRef,
		ManifestJSON:            verified.ManifestJSON,
		InstalledAt:             s.now().UTC(),
		Selected:                true,
	}
	if err := s.repository.CommitInstallation(ctx, id, plugin, version); err != nil {
		// CommitInstallation is atomic. A failed commit must preserve the old
		// selected version; the PROMOTED journal entry is handled by recovery.
		return fmt.Errorf("commit installed Plugin (operation remains recoverable): %w", err)
	}
	return nil
}

func validateRequest(request Request) error {
	if strings.TrimSpace(string(request.PluginID)) == "" ||
		strings.TrimSpace(request.Version) == "" ||
		strings.TrimSpace(request.SourceReference) == "" ||
		(request.Kind != installation.KindInstall && request.Kind != installation.KindUpdate) {
		return ErrInvalidRequest
	}
	if strings.ContainsAny(request.Version, "/\\:") || request.Version == "." || request.Version == ".." {
		return ErrInvalidRequest
	}
	return nil
}

func failureCode(stage string) string {
	switch stage {
	case "acquire Plugin artifact":
		return "ARTIFACT_ACQUISITION_FAILED"
	case "validate staging location":
		return "INVALID_STAGING_LOCATION"
	case "record staged artifact":
		return "STAGING_RECORD_FAILED"
	case "inspect Plugin package":
		return "PACKAGE_INSPECTION_FAILED"
	case "validate assembled Plugin":
		return "PLUGIN_VALIDATION_FAILED"
	case "resolve platform Plugin metadata":
		return "PLATFORM_METADATA_RESOLUTION_FAILED"
	case "match inspected Plugin identity":
		return "PLUGIN_IDENTITY_MISMATCH"
	case "extract Plugin package":
		return "PACKAGE_EXTRACTION_FAILED"
	case "validate extracted package location":
		return "INVALID_EXTRACTION_LOCATION"
	case "verify Plugin artifact":
		return "ARTIFACT_VERIFICATION_FAILED"
	case "record verified artifact":
		return "VERIFICATION_RECORD_FAILED"
	case "promote verified artifact":
		return "ARTIFACT_PROMOTION_FAILED"
	default:
		return "INSTALLATION_FAILED"
	}
}
