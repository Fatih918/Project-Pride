package credential

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
	}
}

func (s *Service) Create(
	ctx context.Context,
	secret []byte,
) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("credential secret must not be empty")
	}

	ref := "cred_" + uuid.NewString()

	err := s.store.Set(
		ctx,
		Credential{
			Ref:   ref,
			Value: secret,
		},
	)
	if err != nil {
		return "", fmt.Errorf("store credential: %w", err)
	}

	return ref, nil
}

func (s *Service) Get(
	ctx context.Context,
	ref string,
) (Credential, error) {
	if ref == "" {
		return Credential{}, fmt.Errorf("credential reference must not be empty")
	}

	credential, err := s.store.Get(ctx, ref)
	if err != nil {
		return Credential{}, fmt.Errorf("get credential: %w", err)
	}

	return credential, nil
}

func (s *Service) Update(
	ctx context.Context,
	ref string,
	secret []byte,
) error {
	if ref == "" {
		return fmt.Errorf("credential reference must not be empty")
	}

	if len(secret) == 0 {
		return fmt.Errorf("credential secret must not be empty")
	}

	if _, err := s.store.Get(ctx, ref); err != nil {
		return fmt.Errorf("get existing credential: %w", err)
	}

	if err := s.store.Set(ctx, Credential{
		Ref:   ref,
		Value: secret,
	}); err != nil {
		return fmt.Errorf("update credential: %w", err)
	}

	return nil
}

func (s *Service) Delete(
	ctx context.Context,
	ref string,
) error {
	if ref == "" {
		return fmt.Errorf("credential reference must not be empty")
	}

	if err := s.store.Delete(ctx, ref); err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}

	return nil
}
