# Plugin Lifecycle Boundary Contracts

**Status:** Accepted boundary specification
**Document ID:** PLR-BND-0001
**Decision authority:** PLR-ADR-0001 and PLR-DOM-0001
**Scope:** SDK-neutral contracts between Plugin Module, Core authorization, lifecycle persistence, and a future isolated runtime
**Out of scope:** Concrete Core adapter, database schema, runtime technology, process supervision, SDK RPCs

## 1. Purpose

This document translates the accepted lifecycle domain model into explicit integration boundaries. It is a contract for subsequent implementation, not evidence that the ports or adapters already exist.

The boundaries are:

1. Plugin Module application service → Core authorization through `AuthorizationPort`.
2. Plugin Module application service → durable lifecycle state through `LifecycleRepository`.
3. Plugin Module runtime coordinator → isolated execution environment through a future `RuntimePort`.

All contracts are SDK-neutral. No generated Protobuf, gRPC, Marketplace, or concrete runtime SDK types may cross these boundaries.

## 2. Trust and ownership

| Concern | Owner | Contract |
|---|---|---|
| Plugin lifecycle invariants | Plugin Module | Validates commands and state transitions |
| Caller authentication and principal provenance | Host/Core | Supplies authenticated caller context; caller identity is not trusted from request payload |
| Authorization policy and decision | Core | Sole authority; deny by default |
| Lifecycle persistence and idempotency | Plugin Module repository adapter | Durable, transactional, cross-process safe |
| Runtime isolation and process supervision | Host/runtime adapter | Executes no untrusted code in Core process |
| SDK transport and DTO mapping | SDK adapter | Remains transport-only and discovery-only for now |

The Module may validate a Core decision's structural completeness and binding to the request. It must not reinterpret policy, synthesize an allow decision, or treat Marketplace permission metadata as a grant.

## 3. AuthorizationPort

### 3.1 Responsibility

`AuthorizationPort` asks Core whether one authenticated principal may perform one lifecycle action against one exact immutable Plugin version.

It is a synchronous decision boundary for the initial Enable/Disable commands. It does not authenticate users, store policy, issue long-lived grants, or own credentials.

### 3.2 Request contract

The conceptual SDK-neutral request contains:

| Field | Requirement |
|---|---|
| `RequestID` | Required; same durable lifecycle operation ID |
| `PrincipalID` | Required; injected from trusted host authentication context, never accepted as authoritative caller input |
| `Action` | Required; initially `plugin.enable` or `plugin.disable` |
| `PluginID` | Required; canonical Module Plugin identity |
| `PluginVersion` | Required; exact selected immutable installed version |
| `PermissionIDs` | Verified platform metadata for that exact version; copied defensively |
| `Context` | Trusted host/Core request context; not arbitrary caller-supplied policy attributes |

The host must bind `PrincipalID` and `Context` to the authenticated request. If the host cannot establish a trusted principal, the operation is denied before the Module calls the port.

### 3.3 Decision contract

A successful call returns a structured decision:

| Field | Requirement |
|---|---|
| `Allowed` | Explicit boolean; false is a denial |
| `DecisionID` | Non-empty opaque audit reference |
| `PolicyRevision` | Non-empty policy revision/reference used by Core |
| `EvaluatedAt` | Non-zero timestamp |
| `ReasonCode` | Optional bounded safe code; never raw policy text |

A transport or policy-evaluation error is not an allow or a denial. It is an indeterminate authorization outcome and must fail closed. An empty or malformed decision also fails closed.

A decision is operation-scoped. It must not be cached or reused for another operation, Plugin, version, principal, or action. The Module persists only safe audit references and decision metadata needed for the operation journal; it never persists credentials or raw policy internals.

### 3.4 No-op authorization

The application service must apply one explicit host policy consistently to already-satisfied Enable/Disable requests:

- **Default contract:** authorize before recording a successful no-op, so an unauthorized caller cannot use idempotency to probe protected Plugin state.
- A host may later choose a different no-op disclosure policy only through an explicit Core policy decision; the Module must not silently bypass authorization because the desired state already matches.

### 3.5 Illustrative Go boundary

The following is a shape guide, not a committed Go API:

```go
type AuthorizationPort interface {
    AuthorizeLifecycle(
        ctx context.Context,
        request AuthorizationRequest,
    ) (AuthorizationDecision, error)
}
```

The concrete request and decision types belong to a future SDK-neutral lifecycle package. The Core adapter implements this port outside the Module's domain package. No Core policy package is imported by the Plugin domain.

## 4. LifecycleRepository

### 4.1 Responsibility

The repository persists desired administrative state, runtime observations, lifecycle operations, idempotency identity, and per-Plugin serialization metadata.

It is separate from `installation.Repository`. It must not mutate installation journal phases or installed artifact contents.

### 4.2 Conceptual records

**LifecycleRecord**

- PluginID;
- desired state (`ENABLED` or `DISABLED`);
- observed runtime state (`UNKNOWN`, `STOPPED`, `STARTING`, `RUNNING`, `STOPPING`, `FAILED`);
- selected installed version identity;
- monotonically increasing record revision;
- created/updated timestamps.

**LifecycleOperation**

- operation/request ID;
- immutable request fingerprint (principal scope, PluginID, version, action);
- PluginID and exact version;
- action;
- phase;
- target desired state;
- authorization decision audit reference and policy revision, when authorized;
- bounded failure code;
- created/updated timestamps.

Do not persist bearer tokens, secrets, raw authorization policy, raw Plugin output, stack traces, or unbounded caller-controlled text.

### 4.3 Required atomic operations

The repository contract must provide equivalent guarantees to these operations:

1. **Reserve operation** — atomically checks idempotency identity and acquires a per-Plugin operation lease. It creates the `REQUESTED` record or returns the existing operation.
2. **Record authorization** — advances the matching operation from `REQUESTED` to `AUTHORIZED`, bound to the active lease/fencing token and authorization decision reference.
3. **Begin apply** — advances `AUTHORIZED` to `APPLYING` under the same active fencing token.
4. **Commit desired state** — atomically updates desired state/revision and advances the operation to `SUCCEEDED`. This is one storage transaction.
5. **Fail operation** — records a bounded terminal failure where the outcome is known and no desired-state mutation was committed.
6. **Recover operations** — lists non-terminal operations and reconciles them using explicit recovery rules; it must not guess about an external runtime side effect.

The implementation may use different method names, but it must preserve these atomicity and isolation guarantees.

### 4.4 Idempotency behavior

- Same operation ID and same immutable request fingerprint: return the existing operation/outcome; never execute it twice.
- Same operation ID with a different principal scope, Plugin, version, or action: return an operation-ID conflict.
- A duplicate request while the original operation is active must not run concurrently. Return a typed in-progress result or the current operation representation.
- A new operation ID for a state already satisfied may be recorded as an authorized successful no-op.
- Idempotency records must remain available for a documented retention period. They must not be deleted while an operation is non-terminal.

### 4.5 Cross-process per-Plugin serialization

A process-local mutex is insufficient. The repository adapter must enforce a durable per-Plugin lease or equivalent database-backed serialization mechanism across host processes.

The lease contract requires:
- one active mutating lifecycle operation per Plugin;
- a monotonically increasing fencing token on each new lease acquisition;
- lease expiry and renewal semantics based on a trusted repository clock;
- every state-changing repository method validates the current fencing token;
- an expired/stale worker cannot commit after another worker acquires a newer lease;
- cancellation and lease loss stop further work;
- lease release is idempotent and cannot release a newer owner's lease.

The authorization call is made outside a database transaction to avoid holding database locks across network I/O, but while the operation lease is held. If the lease expires during authorization, the subsequent fenced `RecordAuthorization` must fail and the worker must not apply the transition. The authorization decision is discarded for mutation purposes; a retry obtains a new operation/authorization path.

### 4.6 Transactional commit invariant

The desired-state update and lifecycle operation transition to `SUCCEEDED` must commit atomically. There must be no observable state where the desired state changed but the journal still appears safely retryable, or vice versa.

If the chosen persistence engine cannot guarantee this transaction, implementation must introduce a separately reviewed recovery protocol before shipping. Do not approximate atomicity with two independent writes.

## 5. RuntimePort

### 5.1 Scope and status

The accepted architecture requires a narrow runtime boundary and targets out-of-process isolation. It does not select process, container, WASM, or another concrete runtime.

The first lifecycle implementation slice is Enable/Disable only and has no RuntimePort side effects. The following is a future boundary contract, not authorization to execute Marketplace code now.

### 5.2 Responsibility

A future RuntimePort is responsible for controlled interaction with an isolated runtime adapter:

- start an exact PluginID/version instance;
- stop an exact runtime instance;
- inspect runtime health and observed state;
- return opaque runtime instance identity and bounded status;
- propagate context cancellation and deadlines.

It must not decide whether the caller is authorized, select a Plugin version, mutate lifecycle desired state, or access the Module repository.

### 5.3 Request identity

Every future runtime request must bind:
- a unique operation/request ID;
- PluginID;
- exact immutable installed version;
- opaque artifact reference resolved by trusted host composition;
- runtime configuration reference, not raw credentials in the request;
- host-enforced resource/isolation profile reference.

The runtime adapter must reject mismatched Plugin/version/artifact identity. It must not resolve "latest" or follow mutable Manifest URLs.

### 5.4 Result and health semantics

A start result must not be treated as `RUNNING` until the adapter confirms the instance is active. It returns an opaque instance ID, exact Plugin/version identity, and a bounded observed status.

A stop result must not be treated as `STOPPED` until the adapter confirms termination. Timeout or lost contact yields `UNKNOWN`/reconciliation-required, not a fabricated stopped result.

Health inspection returns bounded structured status and safe failure codes. Plugin-controlled strings, logs, and output are untrusted and must be size-limited and redacted before persistence or exposure.

### 5.5 Isolation and resource controls

The host/runtime adapter, not the Plugin Module domain, enforces the selected isolation technology and resource limits. At minimum, the eventual runtime decision must cover:
- process identity and lifecycle supervision;
- filesystem and network access policy;
- CPU, memory, process-count, and execution-time limits;
- secret delivery and revocation without exposing secrets to the SDK;
- stdout/stderr and IPC message size limits;
- termination behavior and orphan-process reconciliation;
- runtime adapter authentication and protocol versioning.

No untrusted Plugin code may execute inside the PRIDE Core process. No Go `plugin` package integration is implied.

### 5.6 Illustrative Go boundary

The future API may have operations equivalent to:

```go
type RuntimePort interface {
    Start(ctx context.Context, request RuntimeStartRequest) (RuntimeInstance, error)
    Stop(ctx context.Context, request RuntimeStopRequest) (RuntimeObservation, error)
    Inspect(ctx context.Context, request RuntimeInspectRequest) (RuntimeObservation, error)
}
```

These names and DTOs are illustrative until a concrete runtime protocol and its failure semantics are separately accepted.

## 6. Error taxonomy

Adapters and application services expose typed, bounded categories. The following names are conceptual and may be mapped to the Module's established error conventions:

| Category | Meaning | Retry guidance |
|---|---|---|
| `AUTHORIZATION_DENIED` | Core explicitly denied the operation | Do not retry unchanged |
| `AUTHORIZATION_UNAVAILABLE` | No trustworthy decision could be obtained | Retry only as a new attempt under host policy |
| `OPERATION_ID_CONFLICT` | Same ID reused with different immutable request identity | Caller must use a new ID |
| `OPERATION_IN_PROGRESS` | Same request is already active | Observe existing operation; do not execute again |
| `PLUGIN_OPERATION_CONFLICT` | Another mutating operation owns the Plugin lease | Bounded retry/backoff |
| `PLUGIN_NOT_FOUND` | Plugin aggregate is absent | Do not retry unchanged |
| `VERSION_NOT_FOUND` | Exact installed version is absent | Do not silently choose another version |
| `VERSION_NOT_SELECTED` | Target version is not selected | Requires explicit selection workflow |
| `LIFECYCLE_NOT_READY` | Installation/startup recovery is unresolved | Retry after readiness is restored |
| `LEASE_LOST` | Worker lost its fencing lease | Abort; never commit with stale token |
| `PERSISTENCE_FAILURE` | Durable operation/state could not be committed | Reconcile by operation ID before retry |
| `RUNTIME_UNAVAILABLE` | Future runtime adapter unavailable | Host policy; no unbounded restart |
| `RUNTIME_OUTCOME_UNKNOWN` | External runtime outcome cannot be established | Reconcile; do not assume stopped/running |
| `RECOVERY_REQUIRED` | Safe outcome cannot be inferred | Operator/host recovery path |

Underlying errors may be retained in restricted diagnostics only if redacted and bounded. Public SDK/API responses must not expose raw errors, credentials, filesystem paths, stack traces, or Plugin-controlled messages.

## 7. Lifecycle operation protocol

The initial Enable/Disable application flow is:

1. Host authenticates caller and supplies trusted principal context.
2. Module validates PluginID, exact selected version, command, and request ID.
3. Repository reserves the operation and acquires the per-Plugin lease/fencing token.
4. Module loads verified permission metadata for the exact immutable version.
5. Module calls Core AuthorizationPort outside a database transaction while retaining the lease.
6. Repository records the decision under the current fencing token. Denial or indeterminate authorization fails closed.
7. Repository marks the operation `APPLYING`.
8. Repository atomically commits desired state, record revision, and `SUCCEEDED` journal outcome.
9. Module returns the durable operation result and releases the lease.

No RuntimePort call occurs in this initial flow. Enable does not start a process; Disable does not claim process termination until a future runtime contract and implementation exist.

## 8. Crash and retry rules

| Crash point | Durable evidence | Required behavior |
|---|---|---|
| Before operation reservation | No operation record | New request may proceed |
| After `REQUESTED`, before authorization | Request and lease/journal state | Lease expires; recovery marks/reconciles without applying; retry must obtain a fresh authorization |
| During authorization | No trusted decision persisted | Treat as indeterminate; do not apply |
| After authorization, before apply | Authorized operation record | Revalidate operation lease and authorization freshness policy before apply; if uncertain, require a new authorization |
| During atomic desired-state commit | Transaction either committed or rolled back | Read operation by ID; return success if committed, otherwise recover/re-authorize; never repeat blindly |
| After `SUCCEEDED` | Terminal durable result | Return stored result for same request ID |

A host restart invalidates process-local ownership. Any lease held by the previous process is reclaimed only after expiry or safe lease fencing. Recovery must never use a stale fencing token.

## 9. Startup and readiness

- Installation recovery and its cross-process lock remain the first existing startup gate.
- Lifecycle journal recovery must complete before Enable/Disable commands are accepted.
- Runtime readiness is a separate condition and is not required for discovery-only readiness.
- If RuntimePort is absent, discovery and initial administrative Enable/Disable may be available only if host policy explicitly permits; the system must not claim runtime execution readiness.
- A stale runtime observation is reconciled to `UNKNOWN` until a future adapter confirms its state.
- A failure to reconcile a non-terminal lifecycle operation blocks lifecycle mutations for the affected Plugin; host readiness policy may choose to block the lifecycle subsystem entirely.

## 10. Implementation gates

Before Go implementation, approve or specify:
1. Concrete SDK-neutral Go package location and exported DTO names.
2. Trusted caller/principal context adapter between Core and Module.
3. Exact Core authorization decision and audit-reference contract.
4. Lifecycle repository schema, lease clock, fencing-token allocation, transaction isolation, and idempotency retention.
5. Error mapping and public disclosure policy.
6. Runtime protocol, artifact-handle delivery, isolation profile, resource limits, and process reconciliation.
7. Host composition and lifecycle readiness integration.

This document authorizes boundary design, not runtime execution or production authorization. Any implementation must preserve the accepted ADR and domain contract.
