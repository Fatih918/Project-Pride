package application

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"pride-plugin-module/domain"
	"pride-plugin-module/lifecycle"
)

var (
	ErrInvalidRequest = errors.New("invalid lifecycle request")
	ErrMissingPort    = errors.New("lifecycle application port is not configured")
	ErrOperationFailed = errors.New("lifecycle operation previously failed")
)

const (
	failureAuthorizationUnavailable = "AUTHORIZATION_UNAVAILABLE"
	failureAuthorizationDenied      = "AUTHORIZATION_DENIED"
	failureAuthorizationRecord      = "AUTHORIZATION_RECORD_FAILED"
	failureBeginApply               = "BEGIN_APPLY_FAILED"
	failureCommitDesiredState       = "DESIRED_STATE_COMMIT_FAILED"
)

// Request is an authenticated administrative command. PrincipalID must be
// supplied by trusted host composition, never copied from untrusted input.
type Request struct {
	OperationID string
	PrincipalID string
	PluginID    domain.PluginID
	Action      domain.LifecycleAction
}

// Result describes the durable operation outcome. Replayed is true when the
// operation ID already existed and its request identity matched. Replayed
// successful operations do not include a
// LifecycleRecord because the repository port intentionally exposes no
// separate record-read operation yet.
type Result struct {
	Operation domain.LifecycleOperation
	Record    domain.LifecycleRecord
	HasRecord bool
	Replayed  bool
}

type Service struct {
	targets       lifecycle.SelectedTargetResolver
	authorization lifecycle.AuthorizationPort
	repository   lifecycle.LifecycleRepository
	now          func() time.Time
}

func NewService(
	targets lifecycle.SelectedTargetResolver,
	authorization lifecycle.AuthorizationPort,
	repository lifecycle.LifecycleRepository,
) (*Service, error) {
	if targets == nil || authorization == nil || repository == nil {
		return nil, ErrMissingPort
	}
	return &Service{
		targets:       targets,
		authorization: authorization,
		repository:   repository,
		now:          func() time.Time { return time.Now().UTC() },
	}, nil
}

// Execute performs only the durable administrative Enable/Disable transition.
// It does not start, stop, load, or execute Plugin code. The repository is
// responsible for making CommitDesiredState atomic with the Succeeded phase.
func (s *Service) Execute(ctx context.Context, request Request) (result Result, returnErr error) {
	if err := validateRequest(request); err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	// Resolve an existing operation before looking up the currently selected
	// version. A retry must replay its original result even if the selected
	// Plugin version has changed since the first request.
	existing, err := s.repository.GetOperation(ctx, request.OperationID)
	if err == nil {
		if !sameRequestIdentity(existing, request) {
			return Result{}, lifecycle.ErrOperationIDConflict
		}
		if err := existing.Validate(); err != nil {
			return Result{}, fmt.Errorf("validate existing lifecycle operation: %w", err)
		}
		return replayResult(existing)
	}
	if !errors.Is(err, lifecycle.ErrOperationNotFound) {
		return Result{}, fmt.Errorf("read existing lifecycle operation: %w", err)
	}

	target, err := s.targets.ResolveSelectedTarget(ctx, request.PluginID)
	if err != nil {
		return Result{}, fmt.Errorf("resolve selected Plugin target: %w", err)
	}
	if err := target.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate selected Plugin target: %w", err)
	}
	if target.PluginID != request.PluginID {
		return Result{}, fmt.Errorf("%w: selected target Plugin identity mismatch", ErrInvalidRequest)
	}

	now := s.now().UTC()
	operation := domain.LifecycleOperation{
		OperationID:   request.OperationID,
		PrincipalID:   request.PrincipalID,
		PluginID:      target.PluginID,
		PluginVersion: target.PluginVersion,
		Action:        request.Action,
		Phase:         domain.LifecycleOperationPhaseRequested,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	fingerprint := requestFingerprint(request)
	reservation, err := s.repository.ReserveOperation(ctx, operation, fingerprint)
	if err != nil {
		return Result{}, fmt.Errorf("reserve lifecycle operation: %w", err)
	}
	if err := reservation.Validate(); err != nil {
		return Result{}, fmt.Errorf("validate lifecycle reservation: %w", err)
	}
	if !sameRequestIdentity(reservation.Operation, request) {
		return Result{}, lifecycle.ErrOperationIDConflict
	}
	if reservation.Existing {
		return replayResult(reservation.Operation)
	}
	if !sameOperationIdentity(reservation.Operation, operation) {
		return Result{}, lifecycle.ErrOperationIDConflict
	}

	lease := reservation.Lease
	defer func() {
		if err := s.repository.ReleaseLease(context.WithoutCancel(ctx), request.OperationID, lease); err != nil {
			returnErr = errors.Join(returnErr, fmt.Errorf("release lifecycle operation lease: %w", err))
		}
	}()

	fail := func(code string, cause error) error {
		if err := s.repository.FailOperation(context.WithoutCancel(ctx), request.OperationID, lease, code); err != nil {
			return errors.Join(cause, fmt.Errorf("record lifecycle operation failure: %w", err))
		}
		return cause
	}

	authRequest := lifecycle.AuthorizationRequest{
		RequestID:     request.OperationID,
		PrincipalID:   request.PrincipalID,
		Action:        request.Action,
		PluginID:      target.PluginID,
		PluginVersion: target.PluginVersion,
		PermissionIDs: append([]string(nil), target.PermissionIDs...),
	}
	decision, err := s.authorization.AuthorizeLifecycle(ctx, authRequest.Clone())
	if err != nil {
		return Result{}, fail(failureAuthorizationUnavailable, fmt.Errorf("%w: %v", lifecycle.ErrAuthorizationUnavailable, err))
	}
	if err := decision.Validate(); err != nil {
		return Result{}, fail(failureAuthorizationUnavailable, fmt.Errorf("%w: %v", lifecycle.ErrAuthorizationUnavailable, err))
	}
	if err := s.repository.RecordAuthorization(ctx, request.OperationID, lease, decision); err != nil {
		return Result{}, fail(failureAuthorizationRecord, fmt.Errorf("record lifecycle authorization: %w", err))
	}
	if !decision.Allowed {
		return Result{}, fail(failureAuthorizationDenied, lifecycle.ErrAuthorizationDenied)
	}
	if err := s.repository.BeginApply(ctx, request.OperationID, lease); err != nil {
		if errors.Is(err, lifecycle.ErrLeaseLost) {
			return Result{}, err
		}
		return Result{}, fail(failureBeginApply, fmt.Errorf("begin lifecycle apply: %w", err))
	}

	targetState, err := request.Action.TargetState()
	if err != nil {
		return Result{}, fail(failureBeginApply, err)
	}
	record, err := s.repository.CommitDesiredState(ctx, request.OperationID, lease, targetState)
	if err != nil {
		if errors.Is(err, lifecycle.ErrLeaseLost) {
			return Result{}, err
		}
		return Result{}, fail(failureCommitDesiredState, fmt.Errorf("commit lifecycle desired state: %w", err))
	}
	if err := record.Validate(); err != nil {
		return Result{}, fmt.Errorf("repository returned invalid committed lifecycle record: %w", err)
	}
	if record.PluginID != target.PluginID ||
		record.SelectedVersion != target.PluginVersion ||
		record.DesiredState != targetState {
		return Result{}, fmt.Errorf("%w: committed lifecycle record does not match request", lifecycle.ErrPersistence)
	}

	operation.Phase = domain.LifecycleOperationPhaseSucceeded
	operation.DecisionID = decision.DecisionID
	operation.PolicyRevision = decision.PolicyRevision
	operation.UpdatedAt = s.now().UTC()
	return Result{
		Operation: operation,
		Record:    record,
		HasRecord: true,
	}, nil
}

func replayResult(operation domain.LifecycleOperation) (Result, error) {
	switch operation.Phase {
	case domain.LifecycleOperationPhaseSucceeded:
		return Result{Operation: operation, Replayed: true}, nil
	case domain.LifecycleOperationPhaseFailed:
		return Result{Operation: operation, Replayed: true}, fmt.Errorf("%w: %s", ErrOperationFailed, operation.FailureCode)
	default:
		return Result{Operation: operation, Replayed: true}, lifecycle.ErrOperationInProgress
	}
}

func validateRequest(request Request) error {
	if strings.TrimSpace(request.OperationID) == "" ||
		len(request.OperationID) > 128 ||
		strings.TrimSpace(request.PrincipalID) == "" ||
		len(request.PrincipalID) > 128 ||
		strings.TrimSpace(string(request.PluginID)) == "" ||
		len(string(request.PluginID)) > 256 ||
		!request.Action.Valid() {
		return ErrInvalidRequest
	}
	return nil
}

func sameRequestIdentity(operation domain.LifecycleOperation, request Request) bool {
	return operation.OperationID == request.OperationID &&
		operation.PrincipalID == request.PrincipalID &&
		operation.PluginID == request.PluginID &&
		operation.Action == request.Action
}

func sameOperationIdentity(left, right domain.LifecycleOperation) bool {
	return left.OperationID == right.OperationID &&
		left.PrincipalID == right.PrincipalID &&
		left.PluginID == right.PluginID &&
		left.PluginVersion == right.PluginVersion &&
		left.Action == right.Action
}

func requestFingerprint(request Request) string {
	hash := sha256.New()
	writeFingerprintPart(hash, request.PrincipalID)
	writeFingerprintPart(hash, string(request.PluginID))
	writeFingerprintPart(hash, strconv.Itoa(int(request.Action)))
	return hex.EncodeToString(hash.Sum(nil))
}

type fingerprintWriter interface {
	Write([]byte) (int, error)
}

func writeFingerprintPart(writer fingerprintWriter, value string) {
	var size [8]byte
	binary.BigEndian.PutUint64(size[:], uint64(len(value)))
	_, _ = writer.Write(size[:])
	_, _ = writer.Write([]byte(value))
}
