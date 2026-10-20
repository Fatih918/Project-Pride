package credential

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	_ "modernc.org/sqlite"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(
	ctx context.Context,
	dsn string,
) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}

	store := &SQLiteStore{
		db: db,
	}

	if err := store.initialize(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return store, nil
}

func (s *SQLiteStore) initialize(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS credentials (
			ref TEXT PRIMARY KEY,
			value BLOB NOT NULL
		);
	`

	if _, err := s.db.ExecContext(ctx, query); err != nil {
		return fmt.Errorf("initialize credentials table: %w", err)
	}

	return nil
}

func (s *SQLiteStore) Get(
	ctx context.Context,
	ref string,
) (Credential, error) {
	const query = `
		SELECT ref, value
		FROM credentials
		WHERE ref = ?;
	`

	var credential Credential

	err := s.db.QueryRowContext(
		ctx,
		query,
		ref,
	).Scan(
		&credential.Ref,
		&credential.Value,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrNotFound
	}

	if err != nil {
		return Credential{}, fmt.Errorf("get credential: %w", err)
	}

	return credential, nil
}

func (s *SQLiteStore) Set(
	ctx context.Context,
	credential Credential,
) error {
	const query = `
		INSERT INTO credentials (ref, value)
		VALUES (?, ?)
		ON CONFLICT(ref)
		DO UPDATE SET value = excluded.value;
	`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		credential.Ref,
		credential.Value,
	); err != nil {
		return fmt.Errorf("set credential: %w", err)
	}

	return nil
}

func (s *SQLiteStore) Delete(
	ctx context.Context,
	ref string,
) error {
	const query = `
		DELETE FROM credentials
		WHERE ref = ?;
	`

	result, err := s.db.ExecContext(
		ctx,
		query,
		ref,
	)
	if err != nil {
		return fmt.Errorf("delete credential: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check deleted credential: %w", err)
	}

	if rows == 0 {
		return ErrNotFound
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

var _ Store = (*SQLiteStore)(nil)
