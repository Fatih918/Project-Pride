# Plugin Lifecycle Domain Contract

**Status:** Accepted domain contract
**Document ID:** PLR-DOM-0001
**Decision authority:** PLR-ADR-0001
**Scope:** SDK-neutral lifecycle state, commands, transitions, operation journal semantics, and recovery invariants
**Out of scope:** Concrete runtime technology, database schema, Core policy implementation, SDK RPCs

## 1. Purpose and boundary

This document defines the domain contract required before lifecycle application services or persistence are implemented.

The Plugin Module owns lifecycle invariants and durable lifecycle state. Core remains the authorization authority. Runtime execution is behind a narrow Module-owned port and is not implemented by this contract.

The lifecycle model is independent of installation operations. Installation continues to establish verified immutable installed versions and an explicit selected version. Lifecycle acts on that selected version; it does not acquire, inspect, verify, promote, or mutate artifacts.

## 2. Aggregate identity

A lifecycle aggregate is identified by `PluginID` within one PRIDE environment.

Every lifecycle operation targets:
- PluginID;
- an explicit selected installed version;
- a caller-supplied operation/request ID;
- a command (`Enable`, `Disable`, and, in a later increment, `Start` or `Stop`).

The selected version must resolve to an existing verified `InstalledVersion` belonging to the same Plugin. Version identity is immutable for the lifetime of that installed record.

A request that omits the selected version may be normalized from the repository's current selected-version record only if the application contract performs that resolution atomically under the Plugin's serialization boundary. Otherwise the request must carry the version explicitly. This choice is an application/API decision, not a reason to infer a version from mutable Manifest data.

## 3. State model

### 3.1 Desired administrative state

The durable desired state has exactly two values:

| Value | Meaning |
|---|---|
| `DISABLED` | The Plugin must not receive new runtime activation requests |
| `ENABLED` | The Plugin is administratively eligible for runtime activation, subject to authorization, compatibility, host policy, and runtime availability |

A newly installed Plugin starts with desired state `DISABLED`.

Desired state records operator intent. It does not claim that a runtime process exists or is healthy.

### 3.2 Observed runtime state

Runtime observation is separate from desired state:

| Value | Meaning |
|---|---|
| `UNKNOWN` | No trustworthy current observation is available |
| `STOPPED` | The runtime adapter has confirmed no active instance for the target version |
| `STARTING` | A start operation has been accepted and is in progress |
| `RUNNING` | The runtime adapter has confirmed an active instance for the target version |
| `STOPPING` | A stop operation has been accepted and is in progress |
| `FAILED` | The runtime adapter or supervisor has reported a bounded failure outcome |

The runtime state is an observation, not an authorization result. A stale `RUNNING` observation must be treated as `UNKNOWN` during startup until reconciliation establishes the actual runtime condition. The Module must never infer liveness from a database row alone.

### 3.3 Lifecycle health/fault

Lifecycle operation failure and runtime failure are represented by bounded typed failure codes and operation outcomes. They are not a third desired-state value.

The existing `domain.PluginState` enum (`Installed`, `Enabled`, `Disabled`, `Running`, `Error`) remains unchanged in this increment. It cannot faithfully encode the product of desired state, runtime observation, and operation health. Any legacy projection must be defined in a separate compatibility design; consumers must not treat the enum as the authoritative internal lifecycle model.

## 4. Command semantics

### 4.1 Initial command scope

The first implementation increment supports durable `Enable` and `Disable` only.

- `Enable` changes desired state to `ENABLED`. It does not start a runtime and does not imply `RUNNING`.
- `Disable` changes desired state to `DISABLED` and blocks new activation. If a later runtime implementation has an active instance, the disable workflow must coordinate quiescence/stop and must not report runtime quiescence until the runtime adapter confirms it.
- `Start` and `Stop` are reserved future commands. They must not be exposed before runtime protocol, health, timeout, cancellation, and supervision semantics are accepted.

### 4.2 Preconditions

Before changing desired state, the application service must establish:
1. the Plugin exists in the authoritative registry;
2. the target version is installed, verified, and belongs to that Plugin;
3. the target version is the selected version, or the request is an explicitly approved operation that changes selection under a separate contract;
4. the Plugin is not in an unreconciled installation/recovery condition;
5. Core has authorized this specific caller, Plugin/version, and operation.

A missing Plugin/version, mismatched Plugin/version, unselected version, or unresolved installation operation is a typed rejection. The Module must not fall back to a different version silently.

### 4.3 Idempotency

Repeated `Enable` while desired state is already `ENABLED`, and repeated `Disable` while already `DISABLED`, are idempotent with respect to desired state.

The same operation/request ID with the same normalized command and target must return the original durable outcome. Reuse of an operation ID with different command or target is a conflict. A new operation ID for an already-satisfied desired state may complete as a recorded no-op.

## 5. Transition table

The table describes desired-state transitions for the initial Enable/Disable slice. Runtime observation is orthogonal and is not changed to RUNNING by either command.

| Current desired state | Command | Next desired state | Result |
|---|---|---|---|
| `DISABLED` | `Enable` | `ENABLED` | Apply after preconditions and Core authorization |
| `ENABLED` | `Enable` | `ENABLED` | Idempotent no-op |
| `ENABLED` | `Disable` | `DISABLED` | Apply after preconditions and Core authorization |
| `DISABLED` | `Disable` | `DISABLED` | Idempotent no-op |

No other desired-state transitions are valid in this increment.

### 5.1 Runtime observation constraints

| Desired state | Runtime observation | Interpretation |
|---|---|---|
| `DISABLED` | `STOPPED` | Disabled and quiescent |
| `DISABLED` | `STOPPING` | Disable intent is durable; shutdown is in progress |
| `DISABLED` | `RUNNING` | Not quiescent; no new activation allowed; reconciliation/stop is required |
| `ENABLED` | `STOPPED` | Enabled but not running |
| `ENABLED` | `RUNNING` | Enabled and runtime-confirmed active |
| Either | `UNKNOWN` | Runtime health is not established; do not claim running or quiescent |
| Either | `FAILED` | Runtime failure is recorded; desired state remains independent |

The first Enable/Disable implementation has no runtime adapter side effects. Since no start command exists yet, the initial implementation should only encounter `STOPPED` or `UNKNOWN` observations. Any `RUNNING` observation is a reconciliation blocker, not a state to overwrite.

## 6. Core authorization contract

Core authorization is evaluated for every non-no-op privileged command. A no-op may still require authorization according to host policy; the application service must not assume no-op means permission-free.

The request supplied to Core must identify:
- caller/principal and trusted request context;
- operation name;
- PluginID and exact selected version;
- verified platform permission metadata for that immutable version;
- relevant host policy context.

The authorization result is scoped to that request and operation. It is not cached as a permanent grant. Missing, unavailable, malformed, or indeterminate authorization results fail closed.

Permission metadata is descriptive policy input only. It never grants permission by itself.

## 7. Lifecycle operation journal

Lifecycle operations use a journal separate from the installation journal.

The initial conceptual phases are:

| Phase | Meaning |
|---|---|
| `REQUESTED` | Validated operation intent and idempotency identity durably recorded |
| `AUTHORIZED` | Core returned an affirmative decision for this operation |
| `APPLYING` | Desired-state mutation is being applied |
| `SUCCEEDED` | Desired-state transition or idempotent no-op is durably committed |
| `FAILED` | Operation ended with a bounded terminal failure code |
| `RECOVERY_REQUIRED` | Outcome cannot safely be inferred; reconciliation is required |

Only the following forward transitions are valid:

- `REQUESTED` → `AUTHORIZED` or `FAILED`
- `AUTHORIZED` → `APPLYING` or `FAILED`
- `APPLYING` → `SUCCEEDED`, `FAILED`, or `RECOVERY_REQUIRED`
- `RECOVERY_REQUIRED` → a reconciled terminal outcome under an explicit recovery procedure

Terminal outcomes are immutable. Repeating the same request ID returns the stored outcome; it must not execute the side effect again.

The journal must record enough immutable request identity to detect operation-ID reuse with a different Plugin, version, command, or caller scope. Raw credentials, bearer tokens, secrets, and unbounded caller/Plugin error strings must never be persisted.

## 8. Concurrency and atomicity

- Operations for the same Plugin are serialized across processes, not merely within one goroutine or application instance.
- Operations for different Plugins may proceed concurrently.
- The serialization boundary covers reading selected version and desired state, recording operation intent, authorization binding, applying the transition, and committing the terminal journal result.
- A conflict must return a stable typed conflict outcome; it must not silently queue an unbounded number of operations.
- Persistence must atomically commit desired-state mutation and the corresponding successful journal outcome. If the storage technology cannot provide this atomicity, the implementation must define and test a recoverable intermediate state before shipping.

## 9. Startup and recovery

Lifecycle recovery is distinct from installation recovery.

- Installation recovery continues to run first under the existing Plugin startup recovery gate.
- Lifecycle recovery must run before lifecycle commands or runtime readiness are exposed.
- Persisted `RUNNING`, `STARTING`, or `STOPPING` observations from a previous host process are stale until reconciled with the runtime adapter.
- If no runtime adapter is configured, the system must not claim runtime availability. It may expose discovery independently, consistent with the existing readiness contract.
- An ambiguous in-flight lifecycle operation becomes `RECOVERY_REQUIRED`; the system must not guess whether an external side effect occurred.
- Recovery must be idempotent and bounded. It must not automatically restart a Plugin unless an explicit bounded host policy authorizes that behavior.

## 10. Version and update invariants

- Every lifecycle operation is bound to one immutable installed version.
- The selected version is resolved explicitly and checked again under the Plugin serialization boundary.
- An update installs a new immutable version; it does not overwrite the old artifact directory.
- Changing selected version while the old version is running is prohibited until a separate stop/switch/restart/rollback contract is accepted.
- Rollback is a selected-version change plus a separately specified runtime transition, not mutation of an installed artifact.

## 11. Error contract

Application and adapter boundaries return typed, bounded error categories, including:
- Plugin or version not found;
- version does not belong to Plugin;
- version is not selected;
- Plugin is not in a lifecycle-ready condition;
- authorization denied or unavailable;
- operation ID conflict;
- per-Plugin operation conflict;
- persistence failure;
- runtime unavailable or runtime outcome unknown;
- recovery required.

External responses must not include credentials, raw Registry responses, filesystem paths, stack traces, or untrusted Plugin-provided diagnostics.

## 12. Implementation gates

The following must be separately specified before implementation:
1. SDK-neutral AuthorizationPort request/decision schema and trust boundary.
2. RuntimePort lifecycle and health protocol, including timeout/cancellation and process identity.
3. Lifecycle repository schema, transactional semantics, cross-process locking, and retention.
4. Enable/Disable application service, including exact no-op authorization policy.
5. Runtime recovery and host readiness behavior.
6. Start/Stop semantics and SDK protocol evolution.

This contract authorizes domain/application design work only. It does not imply that any of these ports, services, schemas, adapters, or runtime features already exist.
