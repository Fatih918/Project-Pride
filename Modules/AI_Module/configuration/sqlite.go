package configuration

import (
	"context"
	"database/sql"
	"encoding/json"
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
	if dsn == "" {
		return nil, fmt.Errorf(
			"configuration database DSN must not be empty",
		)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf(
			"open configuration sqlite database: %w",
			err,
		)
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

func (s *SQLiteStore) initialize(
	ctx context.Context,
) error {
	if s == nil || s.db == nil {
		return errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	exists, err := s.tableExists(
		ctx,
		"ai_configurations",
	)
	if err != nil {
		return fmt.Errorf(
			"check AI configuration table: %w",
			err,
		)
	}

	if exists {
		legacy, err := s.isLegacySchema(ctx)
		if err != nil {
			return fmt.Errorf(
				"inspect AI configuration schema: %w",
				err,
			)
		}

		if legacy {
			if err := s.migrateLegacySchema(ctx); err != nil {
				return err
			}
		}
	}

	const configurationsTable = `
	CREATE TABLE IF NOT EXISTS ai_configurations (
		ref TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		implementation_ref TEXT NOT NULL,
		connection_parameters BLOB,
		credential_ref TEXT,
		enabled INTEGER NOT NULL
	);
	`

	if _, err := s.db.ExecContext(
		ctx,
		configurationsTable,
	); err != nil {
		return fmt.Errorf(
			"initialize AI configuration table: %w",
			err,
		)
	}

	const stateTable = `
	CREATE TABLE IF NOT EXISTS ai_configuration_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		active_ref TEXT
	);
	`

	if _, err := s.db.ExecContext(
		ctx,
		stateTable,
	); err != nil {
		return fmt.Errorf(
			"initialize AI configuration state table: %w",
			err,
		)
	}

	return nil
}

func (s *SQLiteStore) tableExists(
	ctx context.Context,
	tableName string,
) (bool, error) {
	const query = `
	SELECT EXISTS(
		SELECT 1
		FROM sqlite_master
		WHERE type = 'table'
		AND name = ?
	);
	`

	var exists bool

	if err := s.db.QueryRowContext(
		ctx,
		query,
		tableName,
	).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (s *SQLiteStore) isLegacySchema(
	ctx context.Context,
) (bool, error) {
	rows, err := s.db.QueryContext(
		ctx,
		`PRAGMA table_info(ai_configurations);`,
	)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			primaryKey int
		)

		if err := rows.Scan(
			&cid,
			&name,
			&columnType,
			&notNull,
			&defaultVal,
			&primaryKey,
		); err != nil {
			return false, err
		}

		if name == "id" {
			return true, nil
		}
	}

	if err := rows.Err(); err != nil {
		return false, err
	}

	return false, nil
}

func (s *SQLiteStore) migrateLegacySchema(
	ctx context.Context,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin AI configuration schema migration: %w",
			err,
		)
	}

	rollback := func() error {
		if rollbackErr := tx.Rollback(); rollbackErr != nil &&
			!errors.Is(rollbackErr, sql.ErrTxDone) {
			return fmt.Errorf(
				"rollback AI configuration schema migration: %w",
				rollbackErr,
			)
		}

		return nil
	}

	const renameQuery = `
	ALTER TABLE ai_configurations
	RENAME TO ai_configurations_legacy;
	`

	if _, err := tx.ExecContext(ctx, renameQuery); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"rename legacy AI configuration table: %w",
			err,
		)
	}

	const createQuery = `
	CREATE TABLE ai_configurations (
		ref TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		type TEXT NOT NULL,
		implementation_ref TEXT NOT NULL,
		connection_parameters BLOB,
		credential_ref TEXT,
		enabled INTEGER NOT NULL
	);
	`

	if _, err := tx.ExecContext(ctx, createQuery); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"create migrated AI configuration table: %w",
			err,
		)
	}

	const migrateQuery = `
	INSERT INTO ai_configurations (
		ref,
		name,
		type,
		implementation_ref,
		connection_parameters,
		credential_ref,
		enabled
	)
	SELECT
		ref,
		name,
		type,
		implementation_ref,
		connection_parameters,
		credential_ref,
		enabled
	FROM ai_configurations_legacy
	WHERE id = 1;
	`

	if _, err := tx.ExecContext(
		ctx,
		migrateQuery,
	); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"migrate AI configuration data: %w",
			err,
		)
	}

	const createStateQuery = `
	CREATE TABLE ai_configuration_state (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		active_ref TEXT
	);
	`

	if _, err := tx.ExecContext(
		ctx,
		createStateQuery,
	); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"create AI configuration state table: %w",
			err,
		)
	}

	const migrateStateQuery = `
	INSERT INTO ai_configuration_state (
		id,
		active_ref
	)
	SELECT
		1,
		ref
	FROM ai_configurations_legacy
	WHERE id = 1;
	`

	if _, err := tx.ExecContext(
		ctx,
		migrateStateQuery,
	); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"migrate active AI configuration state: %w",
			err,
		)
	}

	const dropQuery = `
	DROP TABLE ai_configurations_legacy;
	`

	if _, err := tx.ExecContext(
		ctx,
		dropQuery,
	); err != nil {
		_ = rollback()

		return fmt.Errorf(
			"remove legacy AI configuration table: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit AI configuration schema migration: %w",
			err,
		)
	}

	return nil
}

func (s *SQLiteStore) Create(
	ctx context.Context,
	configuration AIConfiguration,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil || s.db == nil {
		return errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	if err := configuration.Validate(); err != nil {
		return err
	}

	parameters, err := json.Marshal(
		configuration.ConnectionParameters,
	)
	if err != nil {
		return fmt.Errorf(
			"encode AI configuration connection parameters: %w",
			err,
		)
	}

	enabled := 0
	if configuration.Enabled {
		enabled = 1
	}

	const query = `
	INSERT INTO ai_configurations (
		ref,
		name,
		type,
		implementation_ref,
		connection_parameters,
		credential_ref,
		enabled
	)
	VALUES (?, ?, ?, ?, ?, ?, ?);
	`

	if _, err := s.db.ExecContext(
		ctx,
		query,
		configuration.Ref,
		configuration.Name,
		configuration.Type,
		configuration.ImplementationRef,
		parameters,
		configuration.CredentialRef,
		enabled,
	); err != nil {
		// Normalize duplicate-reference behavior with MemoryStore.
		// A failed insert may be caused by a duplicate primary key.
		if _, lookupErr := s.getByRef(
			ctx,
			s.db,
			configuration.Ref,
		); lookupErr == nil {
			return ErrConfigurationAlreadyExists
		}

		return fmt.Errorf(
			"create AI configuration: %w",
			err,
		)
	}

	return nil
}

func (s *SQLiteStore) GetByRef(
	ctx context.Context,
	configurationRef string,
) (AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if s == nil || s.db == nil {
		return AIConfiguration{}, errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	if configurationRef == "" {
		return AIConfiguration{}, ErrConfigurationRefNotFound
	}

	return s.getByRef(
		ctx,
		s.db,
		configurationRef,
	)
}

func (s *SQLiteStore) getByRef(
	ctx context.Context,
	queryer interface {
		QueryRowContext(
			context.Context,
			string,
			...any,
		) *sql.Row
	},
	configurationRef string,
) (AIConfiguration, error) {
	const query = `
	SELECT
		ref,
		name,
		type,
		implementation_ref,
		connection_parameters,
		credential_ref,
		enabled
	FROM ai_configurations
	WHERE ref = ?;
	`

	var (
		configuration AIConfiguration
		parameters    []byte
		enabled       int
	)

	err := queryer.QueryRowContext(
		ctx,
		query,
		configurationRef,
	).Scan(
		&configuration.Ref,
		&configuration.Name,
		&configuration.Type,
		&configuration.ImplementationRef,
		&parameters,
		&configuration.CredentialRef,
		&enabled,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return AIConfiguration{}, ErrConfigurationRefNotFound
	}

	if err != nil {
		return AIConfiguration{}, fmt.Errorf(
			"get AI configuration by reference: %w",
			err,
		)
	}

	if len(parameters) > 0 {
		if err := json.Unmarshal(
			parameters,
			&configuration.ConnectionParameters,
		); err != nil {
			return AIConfiguration{}, fmt.Errorf(
				"decode AI configuration connection parameters: %w",
				err,
			)
		}
	}

	configuration.Enabled = enabled != 0

	return configuration, nil
}

func (s *SQLiteStore) GetActive(
	ctx context.Context,
) (AIConfiguration, error) {
	if err := ctx.Err(); err != nil {
		return AIConfiguration{}, err
	}

	if s == nil || s.db == nil {
		return AIConfiguration{}, errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	const stateQuery = `
	SELECT active_ref
	FROM ai_configuration_state
	WHERE id = 1;
	`

	var activeRef sql.NullString

	err := s.db.QueryRowContext(
		ctx,
		stateQuery,
	).Scan(&activeRef)

	if errors.Is(err, sql.ErrNoRows) {
		return AIConfiguration{}, ErrConfigurationNotFound
	}

	if err != nil {
		return AIConfiguration{}, fmt.Errorf(
			"get active AI configuration state: %w",
			err,
		)
	}

	if !activeRef.Valid || activeRef.String == "" {
		return AIConfiguration{}, ErrConfigurationNotFound
	}

	return s.getByRef(
		ctx,
		s.db,
		activeRef.String,
	)
}

func (s *SQLiteStore) Delete(
	ctx context.Context,
	configurationRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil || s.db == nil {
		return errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	if configurationRef == "" {
		return ErrConfigurationRefNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin delete AI configuration: %w",
			err,
		)
	}

	rollback := func() {
		_ = tx.Rollback()
	}

	const deleteQuery = `
	DELETE FROM ai_configurations
	WHERE ref = ?;
	`

	result, err := tx.ExecContext(
		ctx,
		deleteQuery,
		configurationRef,
	)
	if err != nil {
		rollback()

		return fmt.Errorf(
			"delete AI configuration: %w",
			err,
		)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		rollback()

		return fmt.Errorf(
			"check deleted AI configuration: %w",
			err,
		)
	}

	if rowsAffected == 0 {
		rollback()

		return ErrConfigurationRefNotFound
	}

	const clearActiveQuery = `
	UPDATE ai_configuration_state
	SET active_ref = NULL
	WHERE id = 1
	AND active_ref = ?;
	`

	if _, err := tx.ExecContext(
		ctx,
		clearActiveQuery,
		configurationRef,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"clear active AI configuration: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit delete AI configuration: %w",
			err,
		)
	}

	return nil
}

func (s *SQLiteStore) Select(
	ctx context.Context,
	configurationRef string,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if s == nil || s.db == nil {
		return errors.New(
			"configuration sqlite store must not be nil",
		)
	}

	if configurationRef == "" {
		return ErrConfigurationRefNotFound
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf(
			"begin select AI configuration: %w",
			err,
		)
	}

	rollback := func() {
		_ = tx.Rollback()
	}

	var exists int

	const existsQuery = `
	SELECT 1
	FROM ai_configurations
	WHERE ref = ?;
	`

	err = tx.QueryRowContext(
		ctx,
		existsQuery,
		configurationRef,
	).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		rollback()

		return ErrConfigurationRefNotFound
	}

	if err != nil {
		rollback()

		return fmt.Errorf(
			"check selected AI configuration: %w",
			err,
		)
	}

	const disableAllQuery = `
	UPDATE ai_configurations
	SET enabled = 0;
	`

	if _, err := tx.ExecContext(
		ctx,
		disableAllQuery,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"disable previous AI configurations: %w",
			err,
		)
	}

	const enableSelectedQuery = `
	UPDATE ai_configurations
	SET enabled = 1
	WHERE ref = ?;
	`

	if _, err := tx.ExecContext(
		ctx,
		enableSelectedQuery,
		configurationRef,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"enable selected AI configuration: %w",
			err,
		)
	}

	const stateQuery = `
	INSERT INTO ai_configuration_state (
		id,
		active_ref
	)
	VALUES (1, ?)
	ON CONFLICT(id)
	DO UPDATE SET
		active_ref = excluded.active_ref;
	`

	if _, err := tx.ExecContext(
		ctx,
		stateQuery,
		configurationRef,
	); err != nil {
		rollback()

		return fmt.Errorf(
			"set selected AI configuration as active: %w",
			err,
		)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf(
			"commit select AI configuration: %w",
			err,
		)
	}

	return nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}

	return s.db.Close()
}

var _ Store = (*SQLiteStore)(nil)

var _ ConfigurationStore = (*SQLiteStore)(nil)

var _ AIConfigurationProvider = (*SQLiteStore)(nil)
