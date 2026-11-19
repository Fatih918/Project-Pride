package installation

import "pride-plugin-module/domain"

// InspectedPackage is the untrusted, developer-owned result of parsing a
// package. It intentionally contains no Plugin type/state or protected
// capability/permission metadata.
type InspectedPackage struct {
	Manifest      domain.Manifest
	ManifestJSON  []byte
	EntryCount    int
	ExpandedBytes uint64
}

// PlatformMetadata is supplied by a trusted Registry/Marketplace adapter,
// never inferred from plugin.json.
type PlatformMetadata struct {
	Type      domain.PluginType
	Protected domain.ProtectedMetadata
}
