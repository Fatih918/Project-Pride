package domain

import "time"

// PluginID is the stable identity of an installed Plugin within one PRIDE
// environment. It is never a filesystem path or runtime identifier.
type PluginID string

type PluginType uint8

const (
	PluginTypeUnspecified PluginType = iota
	PluginTypeActive
	PluginTypePassive
)

type PluginState uint8

const (
	PluginStateUnspecified PluginState = iota
	PluginStateInstalled
	PluginStateEnabled
	PluginStateDisabled
	PluginStateRunning
	PluginStateError
)

// Manifest contains developer-owned identity and descriptive information.
// Capability and permission declarations intentionally live outside Manifest.
type Manifest struct {
	PluginID            PluginID
	Name                string
	Version             string
	Description         string
	Author              string
	License             string
	EntryPoint          string
	MinimumPRIDEVersion string
	Homepage            string
	DocumentationURL    string
	RepositoryURL       string
	IssueTrackerURL     string
	Keywords            []string
	Screenshots         []string
	Category            string
	Icon                string
	ChangelogURL        string
}

type Capability struct {
	ID          string
	Name        string
	Description string
}

// CapabilityContract is a versioned, signed declaration exposed by a Plugin.
// It remains declarative metadata: storing it does not resolve dependencies,
// authorize access, or enable invocation.
type CapabilityContract struct {
	ID                    string                       `json:"id"`
	Name                  string                       `json:"name"`
	Description           string                       `json:"description"`
	ContractVersion       string                       `json:"contract_version"`
	InvocationMode        string                       `json:"invocation_mode"`
	InputSchemaRef        string                       `json:"input_schema_ref"`
	OutputSchemaRef       string                       `json:"output_schema_ref"`
	PlatformCompatibility *CapabilityPlatformCompatibility `json:"platform_compatibility,omitempty"`
	Dependencies          []CapabilityDependency       `json:"dependencies,omitempty"`
}

type CapabilityPlatformCompatibility struct {
	PrideVersion      string   `json:"pride_version,omitempty"`
	RuntimeProfileIDs []string `json:"runtime_profile_ids,omitempty"`
}

type CapabilityDependency struct {
	CapabilityID       string `json:"capability_id"`
	VersionRequirement string `json:"version_requirement"`
	Kind               string `json:"kind"`
}

// ProtectedMetadata contains platform/Marketplace-owned declarations and
// analysis. Permission IDs are metadata only; they are not authorization.
type ProtectedMetadata struct {
	Capabilities        []Capability
	CapabilityContracts []CapabilityContract `json:"capability_contracts,omitempty"`
	PermissionIDs       []string
	CompatibilityStatus string
	Attributes          map[string]any
}

type Plugin struct {
	ID            PluginID
	Manifest      Manifest
	Type          PluginType
	State         PluginState
	Protected     ProtectedMetadata
	Available     bool
	StatusMessage string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// Clone returns a defensive deep copy so callers cannot mutate stored records.
func (p Plugin) Clone() Plugin {
	p.Manifest.Keywords = append([]string(nil), p.Manifest.Keywords...)
	p.Manifest.Screenshots = append([]string(nil), p.Manifest.Screenshots...)
	p.Protected.Capabilities = append([]Capability(nil), p.Protected.Capabilities...)
	p.Protected.CapabilityContracts = cloneCapabilityContracts(p.Protected.CapabilityContracts)
	p.Protected.PermissionIDs = append([]string(nil), p.Protected.PermissionIDs...)
	p.Protected.Attributes = cloneMap(p.Protected.Attributes)
	return p
}

func cloneMap(input map[string]any) map[string]any {
	if input == nil {
		return nil
	}
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = cloneValue(value)
	}
	return output
}

func cloneValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		return cloneMap(typed)
	case []any:
		output := make([]any, len(typed))
		for i, item := range typed {
			output[i] = cloneValue(item)
		}
		return output
	case []string:
		return append([]string(nil), typed...)
	default:
		return value
	}
}
