package credential

import "context"

type EncryptedStore struct {
	storage   Store
	encryptor Encryptor
}

func NewEncryptedStore(
	storage Store,
	encryptor Encryptor,
) *EncryptedStore {
	return &EncryptedStore{
		storage:   storage,
		encryptor: encryptor,
	}
}

func (s *EncryptedStore) Get(
	ctx context.Context,
	ref string,
) (Credential, error) {
	stored, err := s.storage.Get(ctx, ref)
	if err != nil {
		return Credential{}, err
	}

	plaintext, err := s.encryptor.Decrypt(ctx, stored.Value)
	if err != nil {
		return Credential{}, err
	}

	return Credential{
		Ref:   stored.Ref,
		Value: plaintext,
	}, nil
}

func (s *EncryptedStore) Set(
	ctx context.Context,
	credential Credential,
) error {
	ciphertext, err := s.encryptor.Encrypt(ctx, credential.Value)
	if err != nil {
		return err
	}

	return s.storage.Set(ctx, Credential{
		Ref:   credential.Ref,
		Value: ciphertext,
	})
}

func (s *EncryptedStore) Delete(
	ctx context.Context,
	ref string,
) error {
	return s.storage.Delete(ctx, ref)
}
