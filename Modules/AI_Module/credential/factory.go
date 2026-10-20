package credential

import (
	"context"
	"fmt"
)

func NewSQLiteEncryptedStore(
	ctx context.Context,
	dsn string,
	keySource KeySource,
) (*EncryptedStore, *SQLiteStore, error) {
	key, err := keySource.GetKey(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"get credential encryption key: %w",
			err,
		)
	}

	encryptor, err := NewAESGCMEncryptor(key)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"create credential encryptor: %w",
			err,
		)
	}

	sqliteStore, err := NewSQLiteStore(ctx, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf(
			"create credential sqlite store: %w",
			err,
		)
	}

	encryptedStore := NewEncryptedStore(
		sqliteStore,
		encryptor,
	)

	return encryptedStore, sqliteStore, nil
}
