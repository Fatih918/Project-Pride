// Package server implements the PRIDE Plugin SDK gRPC server boundary.
package server

import "context"

// DiscoveryPort is the narrow, read-only application boundary supplied by the
// Plugin Module. Implementations own registry semantics and authoritative data.
// The SDK adapter must not access repositories, loaders, runtimes, or storage.
type DiscoveryPort interface {
	ListPlugins(context.Context, ListQuery) (PluginPage, error)
	GetPlugin(context.Context, string) (Plugin, error)
	FindPluginsByCapability(context.Context, CapabilityQuery) (PluginPage, error)
	GetPluginMetadata(context.Context, string) (PluginMetadata, error)
	GetPluginState(context.Context, string) (PluginStateInfo, error)
}

type ListQuery struct {
	PageSize int32
	PageToken string
	Type PluginType
	State PluginState
	Metadata map[string]any
}

type CapabilityQuery struct {
	CapabilityID string
	PageSize int32
	PageToken string
	Metadata map[string]any
}

type PluginPage struct {
	Plugins []Plugin
	NextPageToken string
	ResponseMetadata map[string]any
}

type Plugin struct {
	Manifest Manifest
	Type PluginType
	State PluginState
	Capabilities []Capability
	Available bool
	CompatibilityStatus string
}

type Manifest struct {
	PluginID string
	Name string
	Version string
	Description string
	Author string
	License string
	EntryPoint string
	MinimumPRIDEVersion string
	Homepage string
	DocumentationURL string
	RepositoryURL string
	IssueTrackerURL string
	Keywords []string
	Category string
	Icon string
	ChangelogURL string
}

type Capability struct {
	ID string
	Name string
	Description string
}

type PluginMetadata struct {
	PluginID string
	Name string
	Version string
	Type PluginType
	Capabilities []Capability
	PermissionIDs []string
	CompatibilityStatus string
	Attributes map[string]any
	ResponseMetadata map[string]any
}

type PluginStateInfo struct {
	PluginID string
	State PluginState
	Available bool
	StatusMessage string
	ResponseMetadata map[string]any
}

type PluginType int32
const (
	PluginTypeUnspecified PluginType = iota
	PluginTypeActive
	PluginTypePassive
)

type PluginState int32
const (
	PluginStateUnspecified PluginState = iota
	PluginStateInstalled
	PluginStateEnabled
	PluginStateDisabled
	PluginStateRunning
	PluginStateError
)
