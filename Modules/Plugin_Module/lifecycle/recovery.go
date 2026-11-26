package lifecycle

import (
	"context"
	"time"

	"pride-plugin-module/domain"
)

// RecoveryCandidate is a durable, read-only view of an operation that has not
// reached a terminal phase. It does not imply that the operation is safe to
// resume or that its previous authorization remains valid.
type RecoveryCandidate struct {
	Operation      domain.LifecycleOperation
	LeaseExpiresAt time.Time
	LeaseActive    bool
}

// RecoveryRepository exposes the minimal journal inspection and expiry
// classification operations required by a future startup recovery coordinator.
// It is deliberately separate from LifecycleRepository so normal command
// execution does not gain recovery authority.
type RecoveryRepository interface {
	ListRecoveryCandidates(context.Context, int) ([]RecoveryCandidate, error)
	MarkExpiredOperationsRecoveryRequired(context.Context, int) (int, error)
}
