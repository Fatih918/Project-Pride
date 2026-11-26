package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDesiredStateTransition(t *testing.T) {
	tests := []struct {
		name    string
		current LifecycleDesiredState
		action  LifecycleAction
		want    LifecycleDesiredState
		changed bool
		wantErr error
	}{
		{"enable disabled", LifecycleDesiredStateDisabled, LifecycleActionEnable, LifecycleDesiredStateEnabled, true, nil},
		{"enable enabled is no-op", LifecycleDesiredStateEnabled, LifecycleActionEnable, LifecycleDesiredStateEnabled, false, nil},
		{"disable enabled", LifecycleDesiredStateEnabled, LifecycleActionDisable, LifecycleDesiredStateDisabled, true, nil},
		{"disable disabled is no-op", LifecycleDesiredStateDisabled, LifecycleActionDisable, LifecycleDesiredStateDisabled, false, nil},
		{"reject unspecified state", LifecycleDesiredStateUnspecified, LifecycleActionEnable, LifecycleDesiredStateUnspecified, false, ErrInvalidLifecycleRecord},
		{"reject unspecified action", LifecycleDesiredStateDisabled, LifecycleActionUnspecified, LifecycleDesiredStateUnspecified, false, ErrInvalidLifecycleAction},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := DesiredStateTransition(tt.current, tt.action)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want || changed != tt.changed {
				t.Fatalf("transition = (%v, %v), want (%v, %v)", got, changed, tt.want, tt.changed)
			}
		})
	}
}

func TestLifecycleRecordValidation(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	valid := LifecycleRecord{
		PluginID:           PluginID("example.plugin"),
		SelectedVersion:    "1.2.3",
		DesiredState:       LifecycleDesiredStateDisabled,
		RuntimeObservation: RuntimeObservationUnknown,
		Revision:           1,
		CreatedAt:          now,
		UpdatedAt:          now,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid record rejected: %v", err)
	}

	invalid := valid
	invalid.SelectedVersion = " "
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidLifecycleRecord) {
		t.Fatalf("invalid record error = %v, want ErrInvalidLifecycleRecord", err)
	}
}

func TestLifecycleOperationValidation(t *testing.T) {
	now := time.Date(2026, time.November, 26, 12, 0, 0, 0, time.UTC)
	valid := LifecycleOperation{
		OperationID:   "op-123",
		PrincipalID:   "principal-1",
		PluginID:      PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		Action:        LifecycleActionEnable,
		Phase:         LifecycleOperationPhaseRequested,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid operation rejected: %v", err)
	}

	invalid := valid
	invalid.FailureCode = string(make([]byte, 65))
	if err := invalid.Validate(); !errors.Is(err, ErrInvalidLifecycleRecord) {
		t.Fatalf("invalid operation error = %v, want ErrInvalidLifecycleRecord", err)
	}
}

func TestRuntimeObservationDoesNotChangeWithDesiredTransition(t *testing.T) {
	currentObservation := RuntimeObservationUnknown
	next, changed, err := DesiredStateTransition(LifecycleDesiredStateDisabled, LifecycleActionEnable)
	if err != nil {
		t.Fatalf("transition failed: %v", err)
	}
	if !changed || next != LifecycleDesiredStateEnabled {
		t.Fatalf("unexpected transition: state=%v changed=%v", next, changed)
	}
	if currentObservation != RuntimeObservationUnknown {
		t.Fatalf("runtime observation changed unexpectedly: %v", currentObservation)
	}
}
