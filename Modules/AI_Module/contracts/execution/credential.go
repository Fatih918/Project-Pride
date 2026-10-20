package execution

import "context"

// CredentialResolver resolves a credential reference
// into its usable secret value.
//
// This interface defines the contract between the execution layer
// and the credential subsystem.
type CredentialResolver interface {
	Resolve(
		ctx context.Context,
		ref string,
	) ([]byte, error)
}
