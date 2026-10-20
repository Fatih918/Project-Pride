package credential

import (
	"context"
	"fmt"
)

// Resolver adapts the credential Service to the execution-layer
// CredentialResolver contract.
type Resolver struct {
	service *Service
}

// NewResolver creates a credential resolver backed by Credential Service.
func NewResolver(service *Service) *Resolver {
	return &Resolver{
		service: service,
	}
}

// Resolve retrieves a credential by reference.
//
// Storage, encryption, and persistence remain entirely inside the
// credential package. The execution layer receives only the resolved
// credential value.
func (r *Resolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf(
			"credential resolver must not be nil",
		)
	}

	if ctx == nil {
		return nil, fmt.Errorf(
			"context must not be nil",
		)
	}

	if r.service == nil {
		return nil, fmt.Errorf(
			"credential service must not be nil",
		)
	}

	if ref == "" {
		return nil, fmt.Errorf(
			"credential reference must not be empty",
		)
	}

	credential, err := r.service.Get(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf(
			"resolve credential %q: %w",
			ref,
			err,
		)
	}

	if len(credential.Value) == 0 {
		return nil, fmt.Errorf(
			"resolved credential %q is empty",
			ref,
		)
	}

	return credential.Value, nil
}

// Compile-time contract verification.
//
// This intentionally verifies the method signature without importing
// the execution package, keeping the credential package dependency-free
// from execution.
var _ interface {
	Resolve(context.Context, string) ([]byte, error)
} = (*Resolver)(nil)
