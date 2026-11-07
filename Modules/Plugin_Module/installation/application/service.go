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
type ArtifactAcquirer interface {
	Acquire(context.Context, string, string) (installation.StagedArtifact, error)
}

// PackageInspector validates package structure and returns the SDK-neutral
// Plugin aggregate. It owns Manifest decoding; this service assumes no JSON
// property names or package layout.
type PackageInspector interface {
	Inspect(context.Context, string) (domain.Plugin, error)
}

// ArtifactVerifier verifies integrity and authenticity under an explicitly
// configured policy. The service does not choose algorithms or trust roots.
type ArtifactVerifier interface {
	Verify(context.Context, string, domain.Plugin) (installation.VerifiedArtifact, error)
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
	acquirer   ArtifactAcquirer
	inspector  PackageInspector
	verifier   ArtifactVerifier
	promoter   ArtifactPromoter
	now        func() time.Time
}

func NewService(
	repository installation.Repository,
	ids IDGenerator,
	acquirer ArtifactAcquirer,
	inspector PackageInspector,
	verifier ArtifactVerifier,
	promoter ArtifactPromoter,
) (*Service, error) {
	if repository == nil || ids == nil || acquirer == nil || inspector == nil || verifier == nil || promoter == nil {
		return nil, ErrMissingPort
	}
	return &Service{
		repository: repository,
		ids:        ids,
		acquirer:   acquirer,
		inspector:  inspector,
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

	plugin, err := s.inspector.Inspect(ctx, staged.StagingPath)
	if err != nil {
		return fail("inspect Plugin package", err)
	}
	if err := plugin.Validate(); err != nil {
		return fail("validate inspected Plugin", err)
	}
	if plugin.ID != request.PluginID || plugin.Manifest.PluginID != request.PluginID || plugin.Manifest.Version != request.Version {
		return fail("match inspected Plugin identity", ErrPluginMismatch)
	}
	if plugin.CreatedAt.IsZero() {
		plugin.CreatedAt = now
	}
	plugin.UpdatedAt = s.now().UTC()

	verified, err := s.verifier.Verify(ctx, staged.StagingPath, plugin.Clone())
	if err != nil {
		return fail("verify Plugin artifact", err)
	}
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
	case "validate inspected Plugin":
		return "PLUGIN_VALIDATION_FAILED"
	case "match inspected Plugin identity":
		return "PLUGIN_IDENTITY_MISMATCH"
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
