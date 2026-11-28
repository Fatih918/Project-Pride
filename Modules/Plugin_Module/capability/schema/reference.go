package schema

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"pride-plugin-module/installation"
)

const (
	MaxReferenceBytes = 240
	MaxDocumentBytes  = 65_536
)

var (
	ErrInvalidReference  = errors.New("invalid capability schema reference")
	ErrUnverifiedPackage = errors.New("capability schemas require verified package evidence")
	ErrSchemaNotFound    = errors.New("capability schema document not found")
	ErrSchemaTooLarge    = errors.New("capability schema document exceeds size limit")
)

// PackageReader reads a bounded file relative to an installed package root.
// Implementations must confine all filesystem access to that package.
type PackageReader interface {
	ReadPackageFile(context.Context, string, string, int64) ([]byte, error)
}

// Resolver loads schema documents only from an installed package whose
// artifact-verification evidence was produced by installation verification.
type Resolver struct {
	reader PackageReader
}

func NewResolver(reader PackageReader) (*Resolver, error) {
	if reader == nil {
		return nil, errors.New("capability schema package reader is required")
	}
	return &Resolver{reader: reader}, nil
}

// Resolve loads one immutable package-relative schema document. The caller
// must pass the direct VerifiedArtifact result returned after descriptor
// signature and artifact digest verification; a path or Plugin declaration
// alone is not verification evidence.
func (r *Resolver) Resolve(ctx context.Context, verified installation.VerifiedArtifact, reference string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r == nil || r.reader == nil {
		return nil, errors.New("capability schema resolver is not configured")
	}
	if err := validateVerifiedArtifact(verified); err != nil {
		return nil, err
	}
	if err := ValidateReference(reference); err != nil {
		return nil, err
	}

	document, err := r.reader.ReadPackageFile(ctx, verified.FinalArtifactPath, reference, MaxDocumentBytes)
	if err != nil {
		return nil, fmt.Errorf("read verified capability schema: %w", err)
	}
	if len(document) == 0 {
		return nil, ErrSchemaNotFound
	}
	if len(document) > MaxDocumentBytes {
		return nil, ErrSchemaTooLarge
	}
	if !utf8.Valid(document) || strings.HasPrefix(string(document), "\xef\xbb\xbf") {
		return nil, ErrInvalidReference
	}
	return append([]byte(nil), document...), nil
}

// ValidateReference enforces the POCS-0001 package-relative schemas/<file>
// reference grammar. It performs no filesystem or network access.
func ValidateReference(reference string) error {
	if reference == "" || len([]byte(reference)) > MaxReferenceBytes ||
		!utf8.ValidString(reference) || strings.ContainsAny(reference, "\\\x00?#") ||
		containsEncodedSeparator(reference) || path.IsAbs(reference) ||
		path.Clean(reference) != reference || !strings.HasPrefix(reference, "schemas/") {
		return ErrInvalidReference
	}
	for _, r := range reference {
		if r < 0x20 || r == 0x7f {
			return ErrInvalidReference
		}
	}
	for _, segment := range strings.Split(reference, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.Contains(segment, ":") {
			return ErrInvalidReference
		}
	}
	return nil
}

func validateVerifiedArtifact(artifact installation.VerifiedArtifact) error {
	if artifact.FinalArtifactPath == "" || strings.Contains(artifact.FinalArtifactPath, "\\") ||
		path.Clean(artifact.FinalArtifactPath) != artifact.FinalArtifactPath {
		return ErrUnverifiedPackage
	}
	parts := strings.Split(artifact.FinalArtifactPath, "/")
	if len(parts) != 3 || parts[0] != "installed" || parts[1] == "" || parts[2] == "" ||
		parts[1] == "." || parts[1] == ".." || parts[2] == "." || parts[2] == ".." ||
		strings.ContainsAny(parts[1]+parts[2], ":") ||
		artifact.DigestAlgorithm != "sha-256" || len(artifact.DigestValue) != 64 ||
		strings.ToLower(artifact.DigestValue) != artifact.DigestValue ||
		artifact.VerificationPolicy == "" || artifact.VerificationEvidenceRef == "" {
		return ErrUnverifiedPackage
	}
	if _, err := hex.DecodeString(artifact.DigestValue); err != nil {
		return ErrUnverifiedPackage
	}
	return nil
}

func containsEncodedSeparator(value string) bool {
	lower := strings.ToLower(value)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}
