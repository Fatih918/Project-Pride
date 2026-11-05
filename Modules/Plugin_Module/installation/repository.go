package installation

import (
	"context"

	"pride-plugin-module/domain"
)

type Repository interface {
	BeginOperation(context.Context, InstallationOperation) error
	RecordStaged(context.Context, OperationID, StagedArtifact) error
	RecordVerified(context.Context, OperationID, VerifiedArtifact) error
	RecordPromoted(context.Context, OperationID, string) error
	CommitInstallation(context.Context, OperationID, domain.Plugin, InstalledVersion) error
	GetOperation(context.Context, OperationID) (InstallationOperation, error)
	ListIncompleteOperations(context.Context) ([]InstallationOperation, error)
	MarkRolledBack(context.Context, OperationID, string) error
	MarkFailed(context.Context, OperationID, string) error
	ListInstalledVersions(context.Context, domain.PluginID) ([]InstalledVersion, error)
	GetSelectedVersion(context.Context, domain.PluginID) (InstalledVersion, error)
}
