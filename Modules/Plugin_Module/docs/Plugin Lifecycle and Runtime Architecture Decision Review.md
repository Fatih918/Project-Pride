# Plugin Lifecycle and Runtime Architecture Decision Review

**Status:** Accepted by PLR-ADR-0001 and PLR-DOM-0001  
**Document ID:** PLR-DR-0001  
**Scope:** Lifecycle state ownership, runtime boundary, authorization, failure recovery, and future SDK protocol evolution  
**Depends on:** Plugin Module domain/registry/discovery foundation; installation and recovery contracts

## 1. Purpose

The Plugin Module currently provides installation, persistence, recovery, and read-only discovery. Its domain already defines `Installed`, `Enabled`, `Disabled`, `Running`, and `Error` states, but there is no accepted lifecycle state machine, loader, runtime contract, authorization integration, or lifecycle RPC.

This review proposes decisions before adding lifecycle implementation. It deliberately does not implement a loader, execute Plugin code, or add SDK wire methods.

The production Core composition root does not yet exist in this repository. Any design must therefore keep host-owned policy and credentials outside the Plugin Module and must not pretend that production runtime integration is available.

## 2. Existing boundaries and constraints

- Plugin Module owns the installed Plugin aggregate, local registry, installation records, and read-only discovery.
- Core owns capability matching and authorization policy. Marketplace `permission_ids` are trusted metadata after descriptor verification, but are not grants or authorization decisions.
- The Plugin SDK currently exposes five read-only discovery RPCs. Its protocol explicitly defers lifecycle and execution methods until contracts are stable.
- The Plugin Module's SDK adapter must remain transport/mapping-only and must not own runtime, registry, or authorization logic.
- Installation/recovery state is distinct from runtime lifecycle state. A crash during installation must continue to be reconciled by the existing operation journal and startup gate.

## 3. Decisions requested

### L1 — Lifecycle state-machine ownership

**Option A — Core-owned state machine.** Core coordinates state transitions and persists Plugin lifecycle state through Module repository operations.

**Option B — Plugin Module-owned state machine; Core-owned authorization.** The Module validates and serializes lifecycle transitions against the authoritative Plugin aggregate. Core supplies an authorization decision through an explicit port before each privileged operation.

**Recommendation: Option B.** The Module owns invariants over its aggregate and durable state, while Core remains the policy decision point. This avoids duplicating Plugin state rules in the host without allowing the Module to invent authorization policy.

### L2 — Lifecycle operations

**Option A — Minimal control plane:** `Enable` and `Disable` only. Runtime start/stop are internal consequences of enablement policy and are not independently exposed initially.

**Option B — Explicit runtime controls:** `Enable`, `Disable`, `Start`, and `Stop` are separate operations.

**Recommendation: Option B as the eventual model, introduced incrementally.** First define durable enable/disable semantics, then add start/stop only after the runtime contract is accepted. Do not expose RPCs for operations whose semantics are not implemented.

### L3 — State semantics

Proposed interpretation to ratify:

| State | Meaning |
|---|---|
| `Installed` | A verified version is installed but has not been enabled |
| `Enabled` | The Plugin is administratively enabled and eligible for runtime activation |
| `Disabled` | The Plugin is administratively disabled; no new activation is permitted |
| `Running` | A runtime instance is confirmed active |
| `Error` | A lifecycle/runtime failure requires explicit recovery or operator action |

The durable administrative intent (enabled/disabled) should be distinguished from observed runtime health. A persisted `Running` value alone must never be treated as proof that a process is still alive after restart.

**Recommendation:** Keep the existing public enum for compatibility, but model desired state and observed runtime status separately in the internal lifecycle record. Startup reconciliation must convert stale runtime observations into a safe non-running state before accepting commands.

### L4 — Runtime execution boundary

**Option A — In-process Go plugin loading.** Simple deployment, but weak isolation, process-wide crash risk, Go toolchain/ABI coupling, and no meaningful containment for untrusted Marketplace code.

**Option B — Out-of-process worker/runtime.** Plugin code runs in a separately supervised process or sandbox with a narrow versioned host protocol.

**Option C — Pluggable runtime abstraction without selecting a concrete isolation technology yet.** Module defines a narrow runtime port; a later platform decision selects process/container/WASM isolation.

**Recommendation: Option C now, with Option B as the security target.** Do not use Go's `plugin` package as the default execution mechanism for third-party Marketplace code. No untrusted Plugin code should execute in the PRIDE Core process.

### L5 — Authorization boundary

**Option A — Module interprets permission IDs.** Rejected: the Module would become a second authorization engine and risk treating declarations as grants.

**Option B — Core authorization port.** Core evaluates the requested operation, Plugin identity/version, verified permission metadata, caller identity/context, and host policy. Module proceeds only after an affirmative, scoped decision.

**Recommendation: Option B.** Deny by default. Authorization must be checked for each privileged lifecycle or runtime operation; a prior successful decision must not be reused as a permanent grant. Permission metadata is input to Core policy, never an authorization result.

### L6 — Lifecycle persistence and concurrency

**Option A — Best-effort in-memory state.** Rejected for production: process restart loses intent and concurrent requests race.

**Option B — Durable lifecycle journal and per-Plugin serialization.** Persist operation intent and transition outcomes; serialize conflicting operations for the same Plugin; make retries idempotent using caller-supplied operation/request IDs.

**Recommendation: Option B.** Keep lifecycle operations separate from the installation journal, because their state transitions and recovery semantics differ. Define explicit states such as `REQUESTED`, `AUTHORIZED`, `APPLYING`, `SUCCEEDED`, and `FAILED` only after the operation contract is settled.

### L7 — Runtime failure and restart policy

**Option A — Automatic restart without policy.** Rejected: can create restart loops and bypass operator intent.

**Option B — Host-configured bounded restart policy.** Core/host owns retry budget, backoff, and operator overrides; Module records observed health and failure reason using safe codes.

**Recommendation: Option B.** Default to no unbounded automatic restart. Startup must reconcile stale runtime records before discovery/readiness claims runtime availability.

### L8 — Version and update behavior

**Option A — In-place replacement.** Rejected: weak rollback and recovery semantics.

**Option B — Immutable installed versions with explicit active-version selection.** Retain existing installed-version records; lifecycle operates on an explicit selected version; update does not silently change a running instance's code.

**Recommendation: Option B.** Define whether an update requires stop/restart and how rollback changes the selected version before implementation. Never mutate a running version's artifact directory.

### L9 — SDK lifecycle protocol

**Option A — Add lifecycle RPCs immediately.** Rejected: transport contract would precede stable state and authorization semantics.

**Option B — Keep SDK discovery-only until lifecycle application contracts and error semantics are accepted.** Then add typed methods and SDK-neutral ports in a separate protocol increment.

**Recommendation: Option B.** No Protobuf changes in this review.

## 4. Cross-cutting invariants

1. Core remains the authorization authority; Module state validation is not authorization.
2. No Plugin code runs inside the Core process by default.
3. No runtime operation may use developer-owned Manifest fields or unsigned metadata as authority.
4. A Plugin is not reported `Running` solely because a database record says so; runtime health must be observed.
5. Installation recovery and lifecycle recovery remain separate state machines.
6. Conflicting operations for one Plugin are serialized; unrelated Plugins may progress independently.
7. Caller cancellation, deadlines, and safe typed failures propagate across boundaries.
8. Runtime output, Plugin-provided error strings, and health details are untrusted and must be bounded/redacted before exposure.
9. Discovery readiness does not imply runtime readiness or authorization readiness.
10. SDK adapters remain transport-only; Core policy and runtime implementations do not leak into SDK packages.

## 5. Explicit non-goals

This review does not implement:
- Plugin loading, execution, sandboxing, or process supervision;
- permission evaluation or policy storage;
- lifecycle repository/schema migrations;
- lifecycle application services or RPCs;
- automatic restart, health monitoring, or runtime scheduling;
- uninstall, rollback, or version garbage collection.

## 6. Proposed implementation sequence after decisions

1. Accept lifecycle state semantics and ownership boundaries.
2. Define SDK-neutral lifecycle domain records and transition table.
3. Define Core authorization and runtime ports with typed failure contracts.
4. Implement durable lifecycle operation journal and per-Plugin concurrency control.
5. Implement lifecycle application service with recovery tests and fake ports.
6. Select and implement the production-isolated runtime adapter.
7. Add SDK lifecycle protocol methods only after application semantics stabilize.
8. Add host composition, readiness, observability, and end-to-end tests.

## 7. Review outcome

The recommendations L1–L9 were explicitly accepted on 2026-11-26 and are recorded normatively in `Plugin Lifecycle and Runtime Architecture Decision Record.md` (PLR-ADR-0001). The SDK-neutral state and transition semantics are specified in `Plugin Lifecycle Domain Contract.md` (PLR-DOM-0001).

Acceptance is an architecture decision, not evidence of implementation. No lifecycle code, database migration, runtime adapter, authorization implementation, or Protobuf change is implied. Implementation must satisfy the accepted ADR and domain contract and must not infer missing production host, identity, policy, or runtime details.
