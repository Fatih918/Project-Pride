package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"pride-plugin-module/domain"
)

var (
	ErrInvalidRuntimeRequest   = errors.New("invalid runtime request")
	ErrInvalidRuntimeResult    = errors.New("invalid runtime result")
	ErrRuntimeIdentityMismatch = errors.New("runtime identity mismatch")
	ErrRuntimeUnavailable      = errors.New("runtime unavailable")
	ErrRuntimeOutcomeUnknown   = errors.New("runtime outcome unknown")
)

type RuntimeFailureCode string

const (
	RuntimeFailureUnavailable       RuntimeFailureCode = "RUNTIME_UNAVAILABLE"
	RuntimeFailureOutcomeUnknown    RuntimeFailureCode = "RUNTIME_OUTCOME_UNKNOWN"
	RuntimeFailureStartRejected     RuntimeFailureCode = "RUNTIME_START_REJECTED"
	RuntimeFailureHandshakeRejected RuntimeFailureCode = "RUNTIME_HANDSHAKE_REJECTED"
	RuntimeFailureStopTimedOut      RuntimeFailureCode = "RUNTIME_STOP_TIMED_OUT"
	RuntimeFailureTerminationFailed RuntimeFailureCode = "RUNTIME_TERMINATION_FAILED"
	RuntimeFailureIdentityMismatch  RuntimeFailureCode = "RUNTIME_IDENTITY_MISMATCH"
	RuntimeFailureProtocolMismatch  RuntimeFailureCode = "RUNTIME_PROTOCOL_MISMATCH"
)

var runtimeFailureCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// RuntimeFailure is a bounded, safe failure returned by a host RuntimePort.
// It deliberately carries no raw process output, paths, credentials, or
// adapter error text.
type RuntimeFailure struct {
	Code      RuntimeFailureCode
	Retryable bool
}

func (e RuntimeFailure) Error() string {
	if !e.Valid() {
		return "RUNTIME_FAILURE"
	}
	return string(e.Code)
}

func (e RuntimeFailure) Valid() bool {
	return runtimeFailureCodePattern.MatchString(string(e.Code))
}

// RuntimeArtifact identifies the exact verified immutable artifact. Digest is
// the lowercase hexadecimal SHA-256 digest produced by artifact verification.
type RuntimeArtifact struct {
	PluginID      domain.PluginID
	PluginVersion string
	SHA256        string
}

func (a RuntimeArtifact) Validate() error {
	if strings.TrimSpace(string(a.PluginID)) == "" ||
		len(string(a.PluginID)) > 256 ||
		strings.TrimSpace(a.PluginVersion) == "" ||
		len(a.PluginVersion) > 128 ||
		len(a.SHA256) != 64 ||
		a.SHA256 != strings.ToLower(a.SHA256) {
		return ErrInvalidRuntimeRequest
	}
	for _, char := range a.SHA256 {
		if !(char >= '0' && char <= '9') && !(char >= 'a' && char <= 'f') {
			return ErrInvalidRuntimeRequest
		}
	}
	return nil
}

// RuntimeIdentity is an opaque host-generated identity for one isolated
// runtime instance. It is not a PID, socket address, or operating-system handle.
type RuntimeIdentity struct {
	InstanceID string
	Artifact   RuntimeArtifact
}

func (i RuntimeIdentity) Validate() error {
	if strings.TrimSpace(i.InstanceID) == "" || len(i.InstanceID) > 128 {
		return ErrInvalidRuntimeRequest
	}
	return i.Artifact.Validate()
}

type RuntimeStartRequest struct {
	OperationID         string
	Identity            RuntimeIdentity
	ConfigurationRef    string
	IsolationProfileRef string
}

func (r RuntimeStartRequest) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" || len(r.OperationID) > 128 ||
		r.Identity.Validate() != nil ||
		!validOpaqueReference(r.ConfigurationRef) ||
		!validOpaqueReference(r.IsolationProfileRef) {
		return ErrInvalidRuntimeRequest
	}
	return nil
}

type RuntimeStopRequest struct {
	OperationID string
	Identity    RuntimeIdentity
}

func (r RuntimeStopRequest) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" || len(r.OperationID) > 128 || r.Identity.Validate() != nil {
		return ErrInvalidRuntimeRequest
	}
	return nil
}

type RuntimeTerminateRequest struct {
	OperationID string
	Identity    RuntimeIdentity
}

func (r RuntimeTerminateRequest) Validate() error {
	if strings.TrimSpace(r.OperationID) == "" || len(r.OperationID) > 128 || r.Identity.Validate() != nil {
		return ErrInvalidRuntimeRequest
	}
	return nil
}

type RuntimeObserveRequest struct {
	Identity RuntimeIdentity
}

func (r RuntimeObserveRequest) Validate() error {
	return r.Identity.Validate()
}

// RuntimeInstance is returned only after the host has authenticated the
// worker's bounded startup handshake. Running means readiness was confirmed.
type RuntimeInstance struct {
	Identity    RuntimeIdentity
	Observation domain.RuntimeObservation
}

func (i RuntimeInstance) ValidateFor(request RuntimeStartRequest) error {
	if i.Identity.Validate() != nil ||
		i.Identity != request.Identity ||
		i.Observation != domain.RuntimeObservationRunning {
		return fmt.Errorf("%w: start result must match requested identity and be authenticated Running", ErrInvalidRuntimeResult)
	}
	return nil
}

// RuntimeObservationResult binds an observed state to the exact instance that
// was inspected. Stop and Terminate results must be STOPPED; Observe may return
// any valid state, including UNKNOWN.
type RuntimeObservationResult struct {
	Identity    RuntimeIdentity
	Observation domain.RuntimeObservation
}

func (r RuntimeObservationResult) ValidateFor(expected RuntimeIdentity, requireStopped bool) error {
	if r.Identity.Validate() != nil || r.Identity != expected || !r.Observation.Valid() {
		return ErrInvalidRuntimeResult
	}
	if requireStopped && r.Observation != domain.RuntimeObservationStopped {
		return fmt.Errorf("%w: successful termination must report Stopped", ErrInvalidRuntimeResult)
	}
	return nil
}

// RuntimePort is the SDK-neutral boundary to a host-managed isolated worker.
// Implementations own process creation, peer authentication, IPC, resource
// enforcement, and supervision. They must be idempotent for repeated requests
// with the same OperationID and immutable identity. Stop performs graceful
// shutdown; Terminate is the bounded force-termination fallback. An unknown
// external outcome must be reported as RuntimeFailureOutcomeUnknown.
type RuntimePort interface {
	Start(context.Context, RuntimeStartRequest) (RuntimeInstance, error)
	Stop(context.Context, RuntimeStopRequest) (domain.RuntimeObservation, error)
	Terminate(context.Context, RuntimeTerminateRequest) (domain.RuntimeObservation, error)
	Observe(context.Context, RuntimeObserveRequest) (domain.RuntimeObservation, error)
}

func validOpaqueReference(value string) bool {
	return strings.TrimSpace(value) != "" && len(value) <= 128
}
