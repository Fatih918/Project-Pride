package discovery

import "pride-plugin-module/domain"

type PluginQuery struct {
	PluginID string
	Metadata map[string]any
}

type ListQuery struct {
	PageSize  int32
	PageToken string
	Type      domain.PluginType
	State     domain.PluginState
	Metadata  map[string]any
}

type CapabilityQuery struct {
	CapabilityID string
	PageSize     int32
	PageToken    string
	Metadata     map[string]any
}

type PluginPage struct {
	Plugins          []PluginRecord
	NextPageToken    string
	ResponseMetadata map[string]any
}

type PluginRecord struct {
	Manifest            domain.Manifest
	Type                domain.PluginType
	State               domain.PluginState
	Capabilities        []domain.Capability
	Available           bool
	CompatibilityStatus string
}

type PluginMetadata struct {
	PluginID            string
	Name                string
	Version             string
	Type                domain.PluginType
	Capabilities        []domain.Capability
	PermissionIDs       []string
	CompatibilityStatus string
	Attributes          map[string]any
	ResponseMetadata    map[string]any
}

type PluginStateInfo struct {
	PluginID         string
	State            domain.PluginState
	Available        bool
	StatusMessage    string
	ResponseMetadata map[string]any
}
