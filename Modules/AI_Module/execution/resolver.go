package execution

import (
	"context"
	"fmt"

	executioncontract "pride-ai-module/contracts/execution"
	"pride-ai-module/credential"
)

type CredentialServiceResolver struct {
	service *credential.Service
}

func NewCredentialServiceResolver(
	service *credential.Service,
) *CredentialServiceResolver {
	return &CredentialServiceResolver{
		service: service,
	}
}

func (r *CredentialServiceResolver) Resolve(
	ctx context.Context,
	ref string,
) ([]byte, error) {
	if r == nil {
		return nil, fmt.Errorf(
			"credential resolver must not be nil",
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

	result, err := r.service.Get(
		ctx,
		ref,
	)
	if err != nil {
		return nil, fmt.Errorf(
			"get credential: %w",
			err,
		)
	}

	return result.Value, nil
}

var _ executioncontract.CredentialResolver = (*CredentialServiceResolver)(nil)
