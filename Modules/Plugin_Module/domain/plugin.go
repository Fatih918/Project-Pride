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
	Category            string
	Icon                string
	ChangelogURL        string
}

type Capability struct {
	ID          string
	Name        string
	Description string
}

// ProtectedMetadata contains platform/Marketplace-owned declarations and
// analysis. Permission IDs are metadata only; they are not authorization.
type ProtectedMetadata struct {
	Capabilities        []Capability
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
	p.Protected.Capabilities = append([]Capability(nil), p.Protected.Capabilities...)
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
