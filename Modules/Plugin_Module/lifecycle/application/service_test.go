package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/lifecycle"
)

type targetResolverStub struct {
	target lifecycle.SelectedTarget
	err    error
}

func (s targetResolverStub) ResolveSelectedTarget(context.Context, domain.PluginID) (lifecycle.SelectedTarget, error) {
	return s.target.Clone(), s.err
}

type authorizationStub struct {
	decision lifecycle.AuthorizationDecision
	err      error
	calls    int
	request  lifecycle.AuthorizationRequest
}

func (s *authorizationStub) AuthorizeLifecycle(_ context.Context, request lifecycle.AuthorizationRequest) (lifecycle.AuthorizationDecision, error) {
	s.calls++
	s.request = request.Clone()
	return s.decision, s.err
}

type repositoryStub struct {
	reservation       lifecycle.OperationReservation
	existingOperation *domain.LifecycleOperation
	reserveErr        error
	recordAuthErr     error
	beginApplyErr     error
	commitErr         error
	failErr           error
	releaseErr        error
	reservedOperation domain.LifecycleOperation
	fingerprint       string
	authorization     lifecycle.AuthorizationDecision
	failureCode       string
	desiredState      domain.LifecycleDesiredState
	events            []string
	record            domain.LifecycleRecord
}

func (s *repositoryStub) ReserveOperation(_ context.Context, operation domain.LifecycleOperation, fingerprint string) (lifecycle.OperationReservation, error) {
	s.events = append(s.events, "reserve")
	s.reservedOperation = operation
	s.fingerprint = fingerprint
	if s.reserveErr != nil {
		return lifecycle.OperationReservation{}, s.reserveErr
	}
	if s.reservation.Operation.OperationID != "" {
		return s.reservation, nil
	}
	return lifecycle.OperationReservation{
		Operation: operation,
		Lease: lifecycle.OperationLease{
			FencingToken: 7,
			ExpiresAt:    time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (s *repositoryStub) RecordAuthorization(_ context.Context, _ string, _ lifecycle.OperationLease, decision lifecycle.AuthorizationDecision) error {
	s.events = append(s.events, "authorize")
	s.authorization = decision
	return s.recordAuthErr
}

func (s *repositoryStub) BeginApply(context.Context, string, lifecycle.OperationLease) error {
	s.events = append(s.events, "begin")
	return s.beginApplyErr
}

func (s *repositoryStub) CommitDesiredState(_ context.Context, _ string, _ lifecycle.OperationLease, state domain.LifecycleDesiredState) (domain.LifecycleRecord, error) {
	s.events = append(s.events, "commit")
	s.desiredState = state
	return s.record, s.commitErr
}

func (s *repositoryStub) FailOperation(_ context.Context, _ string, _ lifecycle.OperationLease, code string) error {
	s.events = append(s.events, "fail")
	s.failureCode = code
	return s.failErr
}

func (s *repositoryStub) GetOperation(context.Context, string) (domain.LifecycleOperation, error) {
	s.events = append(s.events, "get")
	if s.existingOperation != nil {
		return *s.existingOperation, nil
	}
	return domain.LifecycleOperation{}, lifecycle.ErrOperationNotFound
}

func (s *repositoryStub) ReleaseLease(context.Context, string, lifecycle.OperationLease) error {
	s.events = append(s.events, "release")
	return s.releaseErr
}

func newTestService(t *testing.T, repository *repositoryStub, authorization *authorizationStub) *Service {
	t.Helper()
	now := time.Date(2026, 11, 26, 11, 0, 0, 0, time.UTC)
	target := lifecycle.SelectedTarget{
		PluginID:      domain.PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		PermissionIDs: []string{"filesystem.read", "network.request"},
	}
	service, err := NewService(
		targetResolverStub{target: target},
		authorization,
		repository,
	)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	service.now = func() time.Time { return now }
	repository.record = domain.LifecycleRecord{
		PluginID:           target.PluginID,
		SelectedVersion:    target.PluginVersion,
		DesiredState:       domain.LifecycleDesiredStateEnabled,
		RuntimeObservation: domain.RuntimeObservationUnknown,
		Revision:           2,
		CreatedAt:          now.Add(-time.Minute),
		UpdatedAt:          now,
	}
	authorization.decision = lifecycle.AuthorizationDecision{
		Allowed:        true,
		DecisionID:     "decision-1",
		PolicyRevision: "policy-9",
		EvaluatedAt:    now,
		ReasonCode:     "POLICY_ALLOWED",
	}
	return service
}

func validRequest() Request {
	return Request{
		OperationID: "operation-1",
		PrincipalID: "principal-1",
		PluginID:    domain.PluginID("example.plugin"),
		Action:      domain.LifecycleActionEnable,
	}
}

func TestExecuteEnableCommitsDesiredStateWithoutRuntimeSideEffects(t *testing.T) {
	repository := &repositoryStub{}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)

	result, err := service.Execute(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.HasRecord || result.Record.DesiredState != domain.LifecycleDesiredStateEnabled {
		t.Fatalf("unexpected result: %+v", result)
	}
	if result.Operation.Phase != domain.LifecycleOperationPhaseSucceeded {
		t.Fatalf("operation phase = %v, want Succeeded", result.Operation.Phase)
	}
	if result.Operation.DecisionID != "decision-1" || result.Operation.PolicyRevision != "policy-9" {
		t.Fatalf("authorization evidence missing from result: %+v", result.Operation)
	}
	if authorization.calls != 1 {
		t.Fatalf("authorization calls = %d, want 1", authorization.calls)
	}
	if len(authorization.request.PermissionIDs) != 2 {
		t.Fatalf("authorization permission metadata = %v", authorization.request.PermissionIDs)
	}
	wantEvents := []string{"get", "reserve", "authorize", "begin", "commit", "release"}
	if len(repository.events) != len(wantEvents) {
		t.Fatalf("repository events = %v, want %v", repository.events, wantEvents)
	}
	for i := range wantEvents {
		if repository.events[i] != wantEvents[i] {
			t.Fatalf("repository events = %v, want %v", repository.events, wantEvents)
		}
	}
	if repository.desiredState != domain.LifecycleDesiredStateEnabled {
		t.Fatalf("desired state = %v, want Enabled", repository.desiredState)
	}
}

func TestExecuteDeniedAuthorizationFailsOperation(t *testing.T) {
	repository := &repositoryStub{}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)
	authorization.decision.Allowed = false
	authorization.decision.ReasonCode = "POLICY_DENIED"

	_, err := service.Execute(context.Background(), validRequest())
	if !errors.Is(err, lifecycle.ErrAuthorizationDenied) {
		t.Fatalf("Execute() error = %v, want authorization denied", err)
	}
	if repository.failureCode != failureAuthorizationDenied {
		t.Fatalf("failure code = %q", repository.failureCode)
	}
	if repository.desiredState != domain.LifecycleDesiredStateUnspecified {
		t.Fatal("denied request must not commit desired state")
	}
}

func TestExecuteAuthorizationUnavailableFailsClosed(t *testing.T) {
	repository := &repositoryStub{}
	authorization := &authorizationStub{err: errors.New("policy backend offline")}
	service := newTestService(t, repository, authorization)

	_, err := service.Execute(context.Background(), validRequest())
	if !errors.Is(err, lifecycle.ErrAuthorizationUnavailable) {
		t.Fatalf("Execute() error = %v, want authorization unavailable", err)
	}
	if repository.failureCode != failureAuthorizationUnavailable {
		t.Fatalf("failure code = %q", repository.failureCode)
	}
	if repository.desiredState != domain.LifecycleDesiredStateUnspecified {
		t.Fatal("authorization failure must not commit desired state")
	}
}

func TestExecuteReplaysSucceededOperationWithoutReauthorizing(t *testing.T) {
	now := time.Date(2026, 11, 26, 11, 0, 0, 0, time.UTC)
	operation := domain.LifecycleOperation{
		OperationID:   "operation-1",
		PrincipalID:   "principal-1",
		PluginID:      domain.PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		Action:        domain.LifecycleActionEnable,
		Phase:         domain.LifecycleOperationPhaseSucceeded,
		CreatedAt:     now.Add(-time.Minute),
		UpdatedAt:     now,
	}
	repository := &repositoryStub{existingOperation: &operation}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)

	result, err := service.Execute(context.Background(), validRequest())
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if !result.Replayed || result.Operation.Phase != domain.LifecycleOperationPhaseSucceeded {
		t.Fatalf("unexpected replay result: %+v", result)
	}
	if authorization.calls != 0 {
		t.Fatalf("authorization calls = %d, want 0", authorization.calls)
	}
	if len(repository.events) != 1 || repository.events[0] != "get" {
		t.Fatalf("unexpected repository events: %v", repository.events)
	}
}

func TestExecuteRejectsInProgressReplay(t *testing.T) {
	now := time.Date(2026, 11, 26, 11, 0, 0, 0, time.UTC)
	operation := domain.LifecycleOperation{
		OperationID:   "operation-1",
		PrincipalID:   "principal-1",
		PluginID:      domain.PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		Action:        domain.LifecycleActionEnable,
		Phase:         domain.LifecycleOperationPhaseApplying,
		CreatedAt:     now.Add(-time.Minute),
		UpdatedAt:     now,
	}
	repository := &repositoryStub{existingOperation: &operation}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)

	_, err := service.Execute(context.Background(), validRequest())
	if !errors.Is(err, lifecycle.ErrOperationInProgress) {
		t.Fatalf("Execute() error = %v, want operation in progress", err)
	}
	if authorization.calls != 0 {
		t.Fatal("in-progress replay must not be reauthorized")
	}
}

func TestExecuteRejectsInvalidRequestBeforePorts(t *testing.T) {
	repository := &repositoryStub{}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)
	request := validRequest()
	request.Action = domain.LifecycleActionUnspecified

	_, err := service.Execute(context.Background(), request)
	if !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("Execute() error = %v, want invalid request", err)
	}
	if len(repository.events) != 0 || authorization.calls != 0 {
		t.Fatal("invalid request must not call external ports")
	}
}

func TestExecuteDoesNotMarkOperationFailedWhenLeaseIsLost(t *testing.T) {
	repository := &repositoryStub{beginApplyErr: lifecycle.ErrLeaseLost}
	authorization := &authorizationStub{}
	service := newTestService(t, repository, authorization)

	_, err := service.Execute(context.Background(), validRequest())
	if !errors.Is(err, lifecycle.ErrLeaseLost) {
		t.Fatalf("Execute() error = %v, want lease lost", err)
	}
	for _, event := range repository.events {
		if event == "fail" {
			t.Fatal("must not mutate operation after lease has been lost")
		}
	}
}
