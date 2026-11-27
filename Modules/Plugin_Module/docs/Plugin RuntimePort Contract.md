# Plugin RuntimePort Contract

**Status:** Accepted implementation contract for the first SDK-neutral runtime boundary  
**Depends on:** PLR-ADR-0003, PLR-BND-0001  
**Scope:** Go types and host-facing RuntimePort interface only

## 1. Ownership

The Plugin Module defines the port and validates identity/result invariants. The host implements it and owns worker creation, OS integration, peer authentication, IPC, resource limits, and supervision. Core remains the authorization authority. No platform process API, PID, socket address, shell command, environment map, OS handle, SDK, gRPC, or Protobuf type crosses this boundary.

## 2. Immutable runtime identity

Every runtime instance is identified by:
- host-generated opaque `InstanceID`;
- canonical Plugin ID;
- exact immutable Plugin version;
- lowercase hexadecimal SHA-256 digest of the verified artifact.

The identity is fixed for the lifetime of the instance. An update creates a different immutable identity; it never mutates the artifact behind an existing instance. The Module does not resolve mutable names such as `latest`.

## 3. Start request and readiness

A start request carries an operation ID, the pre-generated runtime identity, and opaque references to host-owned runtime configuration and isolation profile. It never carries raw credentials or arbitrary environment variables.

A successful Start result must match the requested identity exactly and report `RUNNING`. The host adapter may return success only after it has authenticated the worker's bounded, versioned handshake and confirmed readiness. Process creation or an unauthenticated worker-provided identity string is not readiness. A mismatch, timeout, malformed handshake, or protocol mismatch fails closed.

The Module's type validation rejects a mismatched identity or any result other than `RUNNING`.

## 4. Stop, terminate, and observe

- `Stop` requests graceful shutdown. Its context deadline bounds the graceful phase.
- `Terminate` requests forced termination after graceful shutdown fails or times out. The host owns the termination deadline.
- `Observe` reports host-confirmed runtime observation for the exact identity.
- Every observation result includes the opaque runtime identity. The Module rejects results for another instance.
- Successful Stop/Terminate results must report `STOPPED`. A lost or ambiguous external outcome must return a typed `RUNTIME_OUTCOME_UNKNOWN` failure, never a fabricated `STOPPED`.
- Stop and Terminate implementations must be idempotent for the same operation ID and identity.

The port itself does not decide when to invoke these operations. Application orchestration and durable runtime intent are later slices.

## 5. Typed failures

Runtime failures expose only a bounded uppercase code and a retryability hint. They must not contain raw adapter errors, stack traces, worker output, filesystem paths, credentials, or unbounded Plugin-controlled strings.

Initial safe codes include:
- `RUNTIME_UNAVAILABLE`
- `RUNTIME_OUTCOME_UNKNOWN`
- `RUNTIME_START_REJECTED`
- `RUNTIME_HANDSHAKE_REJECTED`
- `RUNTIME_STOP_TIMED_OUT`
- `RUNTIME_TERMINATION_FAILED`
- `RUNTIME_IDENTITY_MISMATCH`
- `RUNTIME_PROTOCOL_MISMATCH`

## 6. Non-goals and implementation gates

This contract does not implement a process adapter, worker binary, sandbox, IPC framing, Core authorization adapter, runtime-intent persistence, Start/Stop application commands, runtime reconciliation, automatic restart, host composition, or lifecycle SDK/Protobuf methods.

Before a host adapter is production-ready, it must separately demonstrate authenticated IPC, OS-enforced isolation and resource limits, secret/capability brokering, bounded supervision, orphan reconciliation, and cross-platform tests. A process boundary alone is not a complete sandbox.
