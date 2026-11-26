package domain

import (
	"errors"
	"strings"
	"time"
)

type LifecycleDesiredState uint8

const (
	LifecycleDesiredStateUnspecified LifecycleDesiredState = iota
	LifecycleDesiredStateDisabled
	LifecycleDesiredStateEnabled
)

type RuntimeObservation uint8

const (
	RuntimeObservationUnknown RuntimeObservation = iota
	RuntimeObservationStopped
	RuntimeObservationStarting
	RuntimeObservationRunning
	RuntimeObservationStopping
	RuntimeObservationFailed
)

type LifecycleAction uint8

const (
	LifecycleActionUnspecified LifecycleAction = iota
	LifecycleActionEnable
	LifecycleActionDisable
)

type LifecycleOperationPhase uint8

const (
	LifecycleOperationPhaseUnspecified LifecycleOperationPhase = iota
	LifecycleOperationPhaseRequested
	LifecycleOperationPhaseAuthorized
	LifecycleOperationPhaseApplying
	LifecycleOperationPhaseSucceeded
	LifecycleOperationPhaseFailed
	LifecycleOperationPhaseRecoveryRequired
)

var (
	ErrInvalidLifecycleRecord = errors.New("invalid lifecycle record")
	ErrInvalidLifecycleAction = errors.New("invalid lifecycle action")
)

// LifecycleRecord is the durable administrative view of one Plugin. Runtime
// observation is deliberately separate from desired state.
type LifecycleRecord struct {
	PluginID           PluginID
	SelectedVersion    string
	DesiredState       LifecycleDesiredState
	RuntimeObservation RuntimeObservation
	Revision           uint64
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

func (r LifecycleRecord) Validate() error {
	if strings.TrimSpace(string(r.PluginID)) == "" ||
		strings.TrimSpace(r.SelectedVersion) == "" ||
		!r.DesiredState.Valid() ||
		!r.RuntimeObservation.Valid() ||
		r.Revision == 0 ||
		r.CreatedAt.IsZero() ||
		r.UpdatedAt.IsZero() ||
		r.UpdatedAt.Before(r.CreatedAt) {
		return ErrInvalidLifecycleRecord
	}
	return nil
}

func (s LifecycleDesiredState) Valid() bool {
	return s == LifecycleDesiredStateDisabled || s == LifecycleDesiredStateEnabled
}

func (s RuntimeObservation) Valid() bool {
	return s <= RuntimeObservationFailed
}

func (a LifecycleAction) Valid() bool {
	return a == LifecycleActionEnable || a == LifecycleActionDisable
}

func (a LifecycleAction) TargetState() (LifecycleDesiredState, error) {
	switch a {
	case LifecycleActionEnable:
		return LifecycleDesiredStateEnabled, nil
	case LifecycleActionDisable:
		return LifecycleDesiredStateDisabled, nil
	default:
		return LifecycleDesiredStateUnspecified, ErrInvalidLifecycleAction
	}
}

// DesiredStateTransition computes the administrative state change only. It
// never changes or infers runtime health.
func DesiredStateTransition(current LifecycleDesiredState, action LifecycleAction) (next LifecycleDesiredState, changed bool, err error) {
	if !current.Valid() {
		return LifecycleDesiredStateUnspecified, false, ErrInvalidLifecycleRecord
	}
	target, err := action.TargetState()
	if err != nil {
		return LifecycleDesiredStateUnspecified, false, err
	}
	return target, current != target, nil
}

// LifecycleOperation stores immutable command identity and bounded outcome
// metadata. It intentionally contains no credentials or caller-controlled
// diagnostic strings.
type LifecycleOperation struct {
	OperationID    string
	PrincipalID    string
	PluginID       PluginID
	PluginVersion  string
	Action         LifecycleAction
	Phase          LifecycleOperationPhase
	FailureCode    string
	DecisionID     string
	PolicyRevision string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (o LifecycleOperation) Validate() error {
	if strings.TrimSpace(o.OperationID) == "" ||
		strings.TrimSpace(o.PrincipalID) == "" ||
		strings.TrimSpace(string(o.PluginID)) == "" ||
		strings.TrimSpace(o.PluginVersion) == "" ||
		!o.Action.Valid() ||
		!o.Phase.Valid() ||
		o.CreatedAt.IsZero() ||
		o.UpdatedAt.IsZero() ||
		o.UpdatedAt.Before(o.CreatedAt) ||
		len(o.FailureCode) > 64 ||
		len(o.DecisionID) > 128 ||
		len(o.PolicyRevision) > 128 {
		return ErrInvalidLifecycleRecord
	}
	return nil
}

func (p LifecycleOperationPhase) Valid() bool {
	return p >= LifecycleOperationPhaseRequested && p <= LifecycleOperationPhaseRecoveryRequired
}
