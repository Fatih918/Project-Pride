package acquisition

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"pride-plugin-module/domain"
)

const registrySourceReferencePrefix = "pride-registry:v1/"

var (
	ErrInvalidSourceReference   = errors.New("invalid Plugin Registry source reference")
	ErrInvalidResolutionRequest = errors.New("invalid Plugin Registry descriptor resolution request")
)

// RegistryReleaseReference identifies one immutable release in a configured
// Registry namespace. It is an identifier, never a URL or endpoint.
type RegistryReleaseReference struct {
	Namespace string
	PluginID  domain.PluginID
	ReleaseID string
}

// DescriptorResolutionRequest binds an immutable source reference to the
// identity and version requested by the installation application service.
type DescriptorResolutionRequest struct {
	Reference        RegistryReleaseReference
	ExpectedPluginID domain.PluginID
	ExpectedVersion  string
}

// ParseRegistryReleaseReference parses the canonical textual form:
//
// pride-registry:v1/{namespace}/{plugin-id}/{release-id}
//
// Each component is a single unescaped path-safe identifier. URL syntax,
// percent-encoding, whitespace, separators, dot segments, and mutable aliases
// are rejected. The Registry is responsible for issuing immutable release IDs.
func ParseRegistryReleaseReference(value string) (RegistryReleaseReference, error) {
	if value == "" || len(value) > 512 || !utf8.ValidString(value) ||
		strings.TrimSpace(value) != value || strings.ContainsAny(value, "%?#\\:@") {
		return RegistryReleaseReference{}, ErrInvalidSourceReference
	}
	if !strings.HasPrefix(value, registrySourceReferencePrefix) {
		return RegistryReleaseReference{}, ErrInvalidSourceReference
	}
	parts := strings.Split(strings.TrimPrefix(value, registrySourceReferencePrefix), "/")
	if len(parts) != 3 {
		return RegistryReleaseReference{}, ErrInvalidSourceReference
	}
	namespace, pluginID, releaseID := parts[0], parts[1], parts[2]
	if !validNamespace(namespace) || !validPluginID(pluginID) || !validReleaseID(releaseID) {
		return RegistryReleaseReference{}, ErrInvalidSourceReference
	}
	return RegistryReleaseReference{
		Namespace: namespace,
		PluginID:  domain.PluginID(pluginID),
		ReleaseID: releaseID,
	}, nil
}

// NewDescriptorResolutionRequest validates that the requested installation
// identity agrees with the immutable Registry source reference.
func NewDescriptorResolutionRequest(reference RegistryReleaseReference, expectedPluginID domain.PluginID, expectedVersion string) (DescriptorResolutionRequest, error) {
	if !validNamespace(reference.Namespace) || !validPluginID(string(reference.PluginID)) ||
		!validReleaseID(reference.ReleaseID) || !validPluginID(string(expectedPluginID)) ||
		expectedPluginID != reference.PluginID || !validVersion(expectedVersion) {
		return DescriptorResolutionRequest{}, ErrInvalidResolutionRequest
	}
	return DescriptorResolutionRequest{
		Reference:        reference,
		ExpectedPluginID: expectedPluginID,
		ExpectedVersion:  expectedVersion,
	}, nil
}

// String returns the canonical representation suitable for persistence.
func (r RegistryReleaseReference) String() string {
	return fmt.Sprintf("%s%s/%s/%s", registrySourceReferencePrefix, r.Namespace, r.PluginID, r.ReleaseID)
}

func validNamespace(value string) bool {
	if len(value) == 0 || len(value) > 63 || value[0] < 'a' || value[0] > 'z' {
		return false
	}
	for i := 1; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return value[len(value)-1] != '-'
}

func validPluginID(value string) bool {
	if len(value) == 0 || len(value) > 255 {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '.' || c == '_' || c == '-') {
			return false
		}
	}
	return value[0] != '.' && value[0] != '-' && value[0] != '_' &&
		value[len(value)-1] != '.' && value[len(value)-1] != '-' && value[len(value)-1] != '_'
}

func validReleaseID(value string) bool {
	if len(value) == 0 || len(value) > 128 || strings.EqualFold(value, "latest") ||
		strings.EqualFold(value, "stable") || strings.EqualFold(value, "current") {
		return false
	}
	for i := 0; i < len(value); i++ {
		c := value[i]
		if !(c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' ||
			c >= '0' && c <= '9' || c == '-' || c == '_' || c == '.') {
			return false
		}
	}
	return value[0] != '.' && value[len(value)-1] != '.'
}

func validVersion(value string) bool {
	return value != "" && len(value) <= 128 && strings.TrimSpace(value) == value &&
		!strings.ContainsAny(value, "/\\?#%:")
}
