package lifecycle

import (
	"errors"
	"testing"
	"time"

	"pride-plugin-module/domain"
)

func TestSelectedTargetValidateAndClone(t *testing.T) {
	target := SelectedTarget{
		PluginID:      domain.PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		PermissionIDs: []string{"filesystem.read"},
	}
	if err := target.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	clone := target.Clone()
	clone.PermissionIDs[0] = "network.access"
	if target.PermissionIDs[0] != "filesystem.read" {
		t.Fatal("Clone() shares permission ID storage")
	}

	invalid := target
	invalid.PermissionIDs = []string{" "}
	if !errors.Is(invalid.Validate(), ErrInvalidSelectedTarget) {
		t.Fatal("Validate() should reject an empty permission ID")
	}
}

func TestAuthorizationDecisionValidate(t *testing.T) {
	valid := AuthorizationDecision{
		Allowed:        false,
		DecisionID:     "decision-123",
		PolicyRevision: "policy-v4",
		EvaluatedAt:    time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC),
		ReasonCode:     "POLICY_DENIED",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	invalid := valid
	invalid.DecisionID = " "
	if !errors.Is(invalid.Validate(), ErrInvalidAuthorizationDecision) {
		t.Fatal("Validate() should reject an empty decision ID")
	}
}

func TestOperationReservationValidate(t *testing.T) {
	now := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	operation := domain.LifecycleOperation{
		OperationID:   "operation-123",
		PrincipalID:   "principal-1",
		PluginID:      domain.PluginID("example.plugin"),
		PluginVersion: "1.2.3",
		Action:        domain.LifecycleActionEnable,
		Phase:         domain.LifecycleOperationPhaseRequested,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	reservation := OperationReservation{
		Operation: operation,
		Lease: OperationLease{
			FencingToken: 1,
			ExpiresAt:    now.Add(time.Minute),
		},
	}
	if err := reservation.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	reservation.Lease.FencingToken = 0
	if !errors.Is(reservation.Validate(), ErrInvalidReservation) {
		t.Fatal("Validate() should reject a missing fencing token")
	}

	reservation.Existing = true
	if err := reservation.Validate(); err != nil {
		t.Fatalf("existing reservation should not require an active lease: %v", err)
	}
}

func TestAuthorizationRequestClone(t *testing.T) {
	request := AuthorizationRequest{PermissionIDs: []string{"filesystem.read"}}
	clone := request.Clone()
	clone.PermissionIDs[0] = "network.access"
	if request.PermissionIDs[0] != "filesystem.read" {
		t.Fatal("Clone() shares permission ID storage")
	}
}

func TestOperationLeaseUsesPositiveSignedSQLiteInteger(t *testing.T) {
	if err := (OperationLease{FencingToken: 1, ExpiresAt: time.Now().UTC()}).Validate(); err != nil {
		t.Fatalf("positive fencing token rejected: %v", err)
	}
	if err := (OperationLease{FencingToken: -1, ExpiresAt: time.Now().UTC()}).Validate(); !errors.Is(err, ErrInvalidReservation) {
		t.Fatalf("negative fencing token error = %v, want ErrInvalidReservation", err)
	}
	if err := (OperationLease{FencingToken: int64(^uint64(0) >> 1), ExpiresAt: time.Now().UTC()}).Validate(); err != nil {
		t.Fatalf("maximum signed fencing token rejected: %v", err)
	}
}
