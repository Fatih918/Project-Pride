package lifecycle

import (
	"context"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
)

type recoveryRepositoryStub struct {
	candidates []RecoveryCandidate
	classifyCounts []int
	classifyErr error
	listErr error
	classifyCalls int
}

func (s *recoveryRepositoryStub) ListRecoveryCandidates(context.Context, int) ([]RecoveryCandidate, error) {
	return append([]RecoveryCandidate(nil), s.candidates...), s.listErr
}

func (s *recoveryRepositoryStub) MarkExpiredOperationsRecoveryRequired(context.Context, int) (int, error) {
	s.classifyCalls++
	if s.classifyErr != nil { return 0, s.classifyErr }
	if len(s.classifyCounts) == 0 { return 0, nil }
	n := s.classifyCounts[0]
	s.classifyCounts = s.classifyCounts[1:]
	return n, nil
}

func TestRecoveryCoordinatorClassifiesExpiredOperationsAndKeepsRecoveryRequired(t *testing.T) {
	repo := &recoveryRepositoryStub{
		classifyCounts: []int{2, 1, 0},
		candidates: []RecoveryCandidate{{
			Operation: domain.LifecycleOperation{Phase: domain.LifecycleOperationPhaseRecoveryRequired},
		}},
	}
	coordinator, err := NewRecoveryCoordinator(repo, 2)
	if err != nil { t.Fatal(err) }
	report, err := coordinator.RecoverWithReport(context.Background())
	if err != nil { t.Fatalf("recover: %v", err) }
	if report.Classified != 3 || report.RecoveryRequired != 1 {
		t.Fatalf("unexpected report: %+v", report)
	}
	if repo.classifyCalls != 2 { t.Fatalf("expected two bounded classification calls, got %d", repo.classifyCalls) }
}

func TestRecoveryCoordinatorFailsClosedForActiveLease(t *testing.T) {
	repo := &recoveryRepositoryStub{candidates: []RecoveryCandidate{{
		Operation: domain.LifecycleOperation{Phase: domain.LifecycleOperationPhaseApplying},
		LeaseActive: true,
		LeaseExpiresAt: time.Now().Add(time.Minute),
	}}}
	coordinator, _ := NewRecoveryCoordinator(repo, 10)
	if err := coordinator.Recover(context.Background()); !errors.Is(err, ErrRecoveryInProgress) {
		t.Fatalf("expected active operation error, got %v", err)
	}
}

func TestRecoveryCoordinatorFailsClosedForUnclassifiedExpiredOperation(t *testing.T) {
	repo := &recoveryRepositoryStub{candidates: []RecoveryCandidate{{
		Operation: domain.LifecycleOperation{Phase: domain.LifecycleOperationPhaseApplying},
	}}}
	coordinator, _ := NewRecoveryCoordinator(repo, 10)
	if err := coordinator.Recover(context.Background()); !errors.Is(err, ErrRecoveryUnclassified) {
		t.Fatalf("expected unclassified operation error, got %v", err)
	}
}

func TestRecoveryCoordinatorDoesNotResumeRecoveryRequiredOperation(t *testing.T) {
	repo := &recoveryRepositoryStub{candidates: []RecoveryCandidate{{
		Operation: domain.LifecycleOperation{Phase: domain.LifecycleOperationPhaseRecoveryRequired},
	}}}
	coordinator, _ := NewRecoveryCoordinator(repo, 10)
	if err := coordinator.Recover(context.Background()); err != nil {
		t.Fatalf("recovery-required operation should remain durable without blocking startup: %v", err)
	}
	if repo.classifyCalls != 1 { t.Fatalf("expected classification inspection, got %d calls", repo.classifyCalls) }
}

func TestRecoveryCoordinatorHonorsCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	repo := &recoveryRepositoryStub{}
	coordinator, _ := NewRecoveryCoordinator(repo, 10)
	if err := coordinator.Recover(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if repo.classifyCalls != 0 { t.Fatal("repository must not be called after cancellation") }
}

func TestNewRecoveryCoordinatorRejectsInvalidConfiguration(t *testing.T) {
	if _, err := NewRecoveryCoordinator(nil, 10); err == nil { t.Fatal("expected nil repository rejection") }
	if _, err := NewRecoveryCoordinator(&recoveryRepositoryStub{}, 0); err == nil { t.Fatal("expected zero batch rejection") }
	if _, err := NewRecoveryCoordinator(&recoveryRepositoryStub{}, DefaultRecoveryBatchSize+1); err == nil { t.Fatal("expected oversized batch rejection") }
}
