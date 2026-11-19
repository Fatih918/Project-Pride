package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ErrTrustSequenceRejected indicates a rollback attempt or same-sequence
// equivocation in the accepted Marketplace trust configuration.
var ErrTrustSequenceRejected = errors.New("trust configuration sequence rejected")

// AcceptTrustConfiguration atomically persists the highest accepted sequence
// and its raw-configuration SHA-256 digest. Replaying the exact accepted pair
// is idempotent; lower sequences and same-sequence/different-digest pairs fail.
func (r *Repository) AcceptTrustConfiguration(ctx context.Context, sequence uint64, digest [32]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if r == nil || r.db == nil || sequence == 0 || sequence > uint64(1<<63-1) {
		return fmt.Errorf("%w: invalid trust sequence store input", ErrTrustSequenceRejected)
	}

	result, err := r.db.ExecContext(ctx, `
		INSERT INTO plugin_trust_configuration_state (
			singleton_id, sequence, digest, accepted_at
		) VALUES (1, ?, ?, ?)
		ON CONFLICT(singleton_id) DO UPDATE SET
			sequence = excluded.sequence,
			digest = excluded.digest,
			accepted_at = excluded.accepted_at
		WHERE excluded.sequence > plugin_trust_configuration_state.sequence
		   OR (
				excluded.sequence = plugin_trust_configuration_state.sequence
				AND excluded.digest = plugin_trust_configuration_state.digest
		   )`,
		int64(sequence), digest[:], time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("persist trust configuration sequence: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("inspect trust configuration sequence result: %w", err)
	}
	if affected != 1 {
		return ErrTrustSequenceRejected
	}
	return nil
}

// CurrentTrustConfigurationSequence returns the accepted sequence and digest.
// It is intended for diagnostics and tests, not for making acceptance decisions.
func (r *Repository) CurrentTrustConfigurationSequence(ctx context.Context) (uint64, [32]byte, error) {
	var sequence int64
	var rawDigest []byte
	err := r.db.QueryRowContext(ctx, `
		SELECT sequence, digest
		FROM plugin_trust_configuration_state
		WHERE singleton_id = 1`).Scan(&sequence, &rawDigest)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, [32]byte{}, nil
	}
	if err != nil {
		return 0, [32]byte{}, fmt.Errorf("read trust configuration sequence: %w", err)
	}
	if sequence <= 0 || len(rawDigest) != 32 {
		return 0, [32]byte{}, errors.New("stored trust configuration state is corrupt")
	}
	var digest [32]byte
	copy(digest[:], rawDigest)
	return uint64(sequence), digest, nil
}
