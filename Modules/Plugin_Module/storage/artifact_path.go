package storage

import (
	"context"

	"pride-plugin-module/domain"
)

// ResolveFinalArtifactPath implements the artifact verifier's destination
// resolver. It returns a canonical Plugin_Storage-relative path only.
func (a *Adapter) ResolveFinalArtifactPath(ctx context.Context, pluginID domain.PluginID, version string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if a == nil || a.layout == nil || a.root == nil {
		return "", ErrInvalidPath
	}
	return a.layout.RelativeInstalledVersionPath(string(pluginID), version)
}
