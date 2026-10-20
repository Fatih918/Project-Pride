package credential

import (
	"context"
	"errors"
	"sync"
)

var ErrNotFound = errors.New("credential not found")

type MemoryStore struct {
	mu          sync.RWMutex
	credentials map[string]Credential
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		credentials: make(map[string]Credential),
	}
}

func (s *MemoryStore) Get(
	ctx context.Context,
	ref string,
) (Credential, error) {
	if err := ctx.Err(); err != nil {
		return Credential{}, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	credential, ok := s.credentials[ref]
	if !ok {
		return Credential{}, ErrNotFound
	}

	return cloneCredential(credential), nil
}

func (s *MemoryStore) Set(
	ctx context.Context,
	credential Credential,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.credentials[credential.Ref] =
		cloneCredential(credential)

	return nil
}

func (s *MemoryStore) Delete(
	ctx context.Context,
	ref string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.credentials[ref]; !ok {
		return ErrNotFound
	}

	delete(s.credentials, ref)

	return nil
}

func cloneCredential(
	credential Credential,
) Credential {
	result := credential

	if credential.Value != nil {
		result.Value = make(
			[]byte,
			len(credential.Value),
		)

		copy(
			result.Value,
			credential.Value,
		)
	}

	return result
}
