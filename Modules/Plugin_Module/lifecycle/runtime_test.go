package lifecycle

import (
	"errors"
	"testing"

	"pride-plugin-module/domain"
)

func validRuntimeStartRequest() RuntimeStartRequest {
	return RuntimeStartRequest{
		OperationID: "operation-1",
		Identity: RuntimeIdentity{
			InstanceID: "instance-1",
			Artifact: RuntimeArtifact{
				PluginID:      "plugin.example",
				PluginVersion: "1.2.3",
				SHA256:        "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			},
		},
		ConfigurationRef:    "runtime-config-1",
		IsolationProfileRef: "isolation-profile-1",
	}
}

func TestRuntimeStartRequestValidation(t *testing.T) {
	request := validRuntimeStartRequest()
	if err := request.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	request.Identity.Artifact.SHA256 = "not-a-digest"
	if !errors.Is(request.Validate(), ErrInvalidRuntimeRequest) {
		t.Fatal("invalid artifact digest was accepted")
	}
}

func TestRuntimeStartResultRequiresAuthenticatedRunningIdentity(t *testing.T) {
	request := validRuntimeStartRequest()
	result := RuntimeInstance{
		Identity:    request.Identity,
		Observation: domain.RuntimeObservationRunning,
	}
	if err := result.ValidateFor(request); err != nil {
		t.Fatalf("valid authenticated result rejected: %v", err)
	}

	result.Observation = domain.RuntimeObservationStarting
	if !errors.Is(result.ValidateFor(request), ErrInvalidRuntimeResult) {
		t.Fatal("Starting was incorrectly accepted as ready")
	}

	result.Observation = domain.RuntimeObservationRunning
	result.Identity.InstanceID = "different-instance"
	if !errors.Is(result.ValidateFor(request), ErrInvalidRuntimeResult) {
		t.Fatal("mismatched runtime identity was accepted")
	}
}

func TestRuntimeFailureIsBoundedAndSafe(t *testing.T) {
	failure := RuntimeFailure{Code: RuntimeFailureHandshakeRejected}
	if !failure.Valid() || failure.Error() != "RUNTIME_HANDSHAKE_REJECTED" {
		t.Fatal("valid safe failure was not preserved")
	}

	failure.Code = RuntimeFailureCode("worker crashed: /private/path")
	if failure.Valid() || failure.Error() != "RUNTIME_FAILURE" {
		t.Fatal("unsafe failure text was exposed")
	}
}

func TestRuntimeStopAndTerminateRequireExactIdentity(t *testing.T) {
	identity := validRuntimeStartRequest().Identity
	if err := (RuntimeStopRequest{OperationID: "stop-1", Identity: identity}).Validate(); err != nil {
		t.Fatalf("valid stop request rejected: %v", err)
	}
	if err := (RuntimeTerminateRequest{OperationID: "terminate-1", Identity: identity}).Validate(); err != nil {
		t.Fatalf("valid terminate request rejected: %v", err)
	}
	if err := (RuntimeObserveRequest{Identity: identity}).Validate(); err != nil {
		t.Fatalf("valid observe request rejected: %v", err)
	}
}

func TestRuntimeObservationResultBindsIdentityAndStopOutcome(t *testing.T) {
	identity := validRuntimeStartRequest().Identity
	result := RuntimeObservationResult{
		Identity: identity,
		Observation: domain.RuntimeObservationStopped,
	}
	if err := result.ValidateFor(identity, true); err != nil {
		t.Fatalf("valid stopped result rejected: %v", err)
	}

	result.Observation = domain.RuntimeObservationUnknown
	if !errors.Is(result.ValidateFor(identity, true), ErrInvalidRuntimeResult) {
		t.Fatal("unknown stop outcome was accepted as stopped")
	}
	if err := result.ValidateFor(identity, false); err != nil {
		t.Fatalf("valid unknown observation rejected: %v", err)
	}

	result.Identity.InstanceID = "other-instance"
	if !errors.Is(result.ValidateFor(identity, false), ErrInvalidRuntimeResult) {
		t.Fatal("observation for another instance was accepted")
	}
}
