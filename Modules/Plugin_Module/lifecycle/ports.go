package lifecycle

import (
	"context"
	"errors"
	"strings"
	"time"

	"pride-plugin-module/domain"
)

var (
	ErrInvalidAuthorizationDecision = errors.New("invalid lifecycle authorization decision")
	ErrInvalidSelectedTarget        = errors.New("invalid selected plugin target")
	ErrInvalidReservation           = errors.New("invalid lifecycle operation reservation")
)

type SelectedTarget struct {
	PluginID      domain.PluginID
	PluginVersion string
	PermissionIDs []string
}

func (t SelectedTarget) Validate() error {
	if strings.TrimSpace(string(t.PluginID)) == "" || strings.TrimSpace(t.PluginVersion) == "" {
		return ErrInvalidSelectedTarget
	}
	if len(t.PermissionIDs) > 256 {
		return ErrInvalidSelectedTarget
	}
	for _, permissionID := range t.PermissionIDs {
		if strings.TrimSpace(permissionID) == "" || len(permissionID) > 128 {
			return ErrInvalidSelectedTarget
		}
	}
	return nil
}

func (t SelectedTarget) Clone() SelectedTarget {
	t.PermissionIDs = append([]string(nil), t.PermissionIDs...)
	return t
}

type SelectedTargetResolver interface {
	ResolveSelectedTarget(context.Context, domain.PluginID) (SelectedTarget, error)
}

type AuthorizationRequest struct {
	RequestID     string
	PrincipalID   string
	Action        domain.LifecycleAction
	PluginID      domain.PluginID
	PluginVersion string
	PermissionIDs []string
}

func (r AuthorizationRequest) Clone() AuthorizationRequest {
	r.PermissionIDs = append([]string(nil), r.PermissionIDs...)
	return r
}

type AuthorizationDecision struct {
	Allowed        bool
	DecisionID     string
	PolicyRevision string
	EvaluatedAt    time.Time
	ReasonCode     string
}

func (d AuthorizationDecision) Validate() error {
	if strings.TrimSpace(d.DecisionID) == "" ||
		len(d.DecisionID) > 128 ||
		strings.TrimSpace(d.PolicyRevision) == "" ||
		len(d.PolicyRevision) > 128 ||
		d.EvaluatedAt.IsZero() ||
		len(d.ReasonCode) > 64 {
		return ErrInvalidAuthorizationDecision
	}
	return nil
}

type AuthorizationPort interface {
	AuthorizeLifecycle(context.Context, AuthorizationRequest) (AuthorizationDecision, error)
}

// OperationLease is a repository-issued fencing capability. The signed
// 64-bit token matches SQLite INTEGER and must never wrap or be reused.
type OperationLease struct {
	FencingToken int64
	ExpiresAt    time.Time
}

func (l OperationLease) Validate() error {
	if l.FencingToken <= 0 || l.ExpiresAt.IsZero() {
		return ErrInvalidReservation
	}
	return nil
}

type OperationReservation struct {
	Operation domain.LifecycleOperation
	Lease     OperationLease
	Existing  bool
}

func (r OperationReservation) Validate() error {
	if err := r.Operation.Validate(); err != nil {
		return ErrInvalidReservation
	}
	if !r.Existing {
		return r.Lease.Validate()
	}
	return nil
}

// LifecycleRepository must implement each state transition transactionally.
// ReserveOperation also enforces operation-ID fingerprint matching and
// cross-process per-Plugin serialization.
type LifecycleRepository interface {
	ReserveOperation(context.Context, domain.LifecycleOperation, string) (OperationReservation, error)
	RecordAuthorization(context.Context, string, OperationLease, AuthorizationDecision) error
	BeginApply(context.Context, string, OperationLease) error
	// CommitDesiredState atomically persists the desired state and transitions
	// the associated operation from Applying to Succeeded.
	CommitDesiredState(context.Context, string, OperationLease, domain.LifecycleDesiredState) (domain.LifecycleRecord, error)
	FailOperation(context.Context, string, OperationLease, string) error
	GetOperation(context.Context, string) (domain.LifecycleOperation, error)
	ReleaseLease(context.Context, string, OperationLease) error
}
