package lifecycle

import (
	"context"
	"errors"
	"fmt"

	"pride-plugin-module/domain"
)

const DefaultRecoveryBatchSize = 1000

var (
	ErrRecoveryInProgress   = errors.New("lifecycle recovery has active operations")
	ErrRecoveryScanLimit    = errors.New("lifecycle recovery candidate scan reached its safety limit")
	ErrRecoveryUnclassified = errors.New("lifecycle recovery found an unclassified operation")
)

// RecoveryReport contains counts only; it deliberately excludes principal IDs,
// authorization evidence, and caller-controlled diagnostic data.
type RecoveryReport struct {
	Classified       int
	RecoveryRequired int
}

// RecoveryCoordinator classifies expired work but never resumes it. Old
// authorization evidence is audit-only and must not be replayed.
type RecoveryCoordinator struct {
	repository RecoveryRepository
	batchSize  int
}

func NewRecoveryCoordinator(repository RecoveryRepository, batchSize int) (*RecoveryCoordinator, error) {
	if repository == nil || batchSize < 1 || batchSize > DefaultRecoveryBatchSize {
		return nil, fmt.Errorf("%w: invalid lifecycle recovery configuration", ErrPersistence)
	}
	return &RecoveryCoordinator{repository: repository, batchSize: batchSize}, nil
}

// Recover implements the startup Recoverer contract.
func (c *RecoveryCoordinator) Recover(ctx context.Context) error {
	_, err := c.RecoverWithReport(ctx)
	return err
}

// RecoverWithReport classifies expired or lease-less operations in bounded
// transactions, then verifies that no operation is still actively owned or
// left in an ambiguous non-terminal phase. RecoveryRequired operations remain
// durable and are never automatically resumed.
func (c *RecoveryCoordinator) RecoverWithReport(ctx context.Context) (RecoveryReport, error) {
	var report RecoveryReport
	if c == nil || c.repository == nil || c.batchSize < 1 {
		return report, fmt.Errorf("%w: uninitialized lifecycle recovery coordinator", ErrPersistence)
	}

	for {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		count, err := c.repository.MarkExpiredOperationsRecoveryRequired(ctx, c.batchSize)
		if err != nil {
			return report, fmt.Errorf("classify expired lifecycle operations: %w", err)
		}
		report.Classified += count
		if count < c.batchSize {
			break
		}
	}

	if err := ctx.Err(); err != nil {
		return report, err
	}
	candidates, err := c.repository.ListRecoveryCandidates(ctx, DefaultRecoveryBatchSize)
	if err != nil {
		return report, fmt.Errorf("inspect lifecycle recovery candidates: %w", err)
	}
	// The repository contract has no continuation token. Exactly-full results
	// may be truncated, so startup must fail closed rather than assume the
	// unreturned tail contains no active lease.
	if len(candidates) == DefaultRecoveryBatchSize {
		return report, ErrRecoveryScanLimit
	}

	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return report, err
		}
		switch candidate.Operation.Phase {
		case domain.LifecycleOperationPhaseRecoveryRequired:
			if candidate.LeaseActive {
				return report, fmt.Errorf("%w: recovery-required operation still has an active lease", ErrRecoveryUnclassified)
			}
			report.RecoveryRequired++
		case domain.LifecycleOperationPhaseRequested,
			domain.LifecycleOperationPhaseAuthorized,
			domain.LifecycleOperationPhaseApplying:
			if candidate.LeaseActive {
				return report, ErrRecoveryInProgress
			}
			return report, ErrRecoveryUnclassified
		default:
			return report, ErrRecoveryUnclassified
		}
	}
	return report, nil
}
