package installation

import (
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

	"pride-plugin-module/domain"
)

type OperationID string
type Kind string
type Phase string

const (
	KindInstall Kind = "INSTALL"
	KindUpdate  Kind = "UPDATE"

	PhasePreparing  Phase = "PREPARING"
	PhaseStaged     Phase = "STAGED"
	PhaseVerified   Phase = "VERIFIED"
	PhasePromoted   Phase = "PROMOTED"
	PhaseCommitted  Phase = "COMMITTED"
	PhaseRolledBack Phase = "ROLLED_BACK"
	PhaseFailed     Phase = "FAILED"
)

type InstalledVersion struct {
	PluginID                domain.PluginID
	Version                 string
	ArtifactPath            string
	SourceReference         string
	DigestAlgorithm         string
	DigestValue             string
	VerificationPolicy      string
	VerificationEvidenceRef string
	ManifestJSON            string
	InstalledAt             time.Time
	Selected                bool
}

type InstallationOperation struct {
	OperationID        OperationID
	PluginID           domain.PluginID
	TargetVersion      string
	Kind               Kind
	Phase              Phase
	StagingPath        string
	FinalArtifactPath  string
	SourceReference    string
	DigestAlgorithm    string
	DigestValue        string
	VerificationPolicy string
	ManifestJSON       string
	FailureCode        string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type StagedArtifact struct {
	StagingPath string
}

type VerifiedArtifact struct {
	FinalArtifactPath       string
	DigestAlgorithm         string
	DigestValue             string
	VerificationPolicy      string
	VerificationEvidenceRef string
	ManifestJSON            string
}

func (o InstallationOperation) ValidateForBegin() error {
	if strings.TrimSpace(string(o.OperationID)) == "" ||
		strings.TrimSpace(string(o.PluginID)) == "" ||
		strings.TrimSpace(o.TargetVersion) == "" ||
		strings.TrimSpace(o.SourceReference) == "" ||
		(o.Kind != KindInstall && o.Kind != KindUpdate) ||
		o.Phase != PhasePreparing || o.CreatedAt.IsZero() || o.UpdatedAt.IsZero() {
		return fmt.Errorf("%w: operation requires identity, target, source, kind, PREPARING phase and timestamps", ErrInvalidRecord)
	}
	if err := validateRelativePath(o.StagingPath, "temporary"); o.StagingPath != "" && err != nil {
		return err
	}
	if o.FinalArtifactPath != "" {
		return fmt.Errorf("%w: final artifact path is assigned only after verification", ErrInvalidRecord)
	}
	if o.DigestAlgorithm != "" || o.DigestValue != "" || o.VerificationPolicy != "" || o.ManifestJSON != "" {
		return fmt.Errorf("%w: verification fields must be empty before verification", ErrInvalidRecord)
	}
	if len(o.FailureCode) > 64 {
		return fmt.Errorf("%w: failure code exceeds 64 characters", ErrInvalidRecord)
	}
	return nil
}

func (v InstalledVersion) Validate() error {
	if strings.TrimSpace(string(v.PluginID)) == "" || strings.TrimSpace(v.Version) == "" ||
		strings.TrimSpace(v.SourceReference) == "" || strings.TrimSpace(v.DigestAlgorithm) == "" ||
		strings.TrimSpace(v.DigestValue) == "" || strings.TrimSpace(v.VerificationPolicy) == "" ||
		!json.Valid([]byte(v.ManifestJSON)) || v.InstalledAt.IsZero() {
		return fmt.Errorf("%w: installed version is missing required data", ErrInvalidRecord)
	}
	if err := validateRelativePath(v.ArtifactPath, "installed"); err != nil {
		return err
	}
	return nil
}

func validateRelativePath(value, requiredRoot string) error {
	if value == "" || strings.Contains(value, "\\") || strings.HasPrefix(value, "/") || path.Clean(value) != value {
		return fmt.Errorf("%w: invalid relative storage path", ErrInvalidRecord)
	}
	for _, segment := range strings.Split(value, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, ":") {
			return fmt.Errorf("%w: invalid relative storage path", ErrInvalidRecord)
		}
	}
	if value != requiredRoot && !strings.HasPrefix(value, requiredRoot+"/") {
		return fmt.Errorf("%w: storage path is outside %s", ErrInvalidRecord, requiredRoot)
	}
	return nil
}
