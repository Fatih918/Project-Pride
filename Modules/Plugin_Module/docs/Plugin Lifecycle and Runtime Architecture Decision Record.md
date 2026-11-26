# Plugin Lifecycle and Runtime Architecture Decision Record

**Status:** Accepted
**Document ID:** PLR-ADR-0001
**Supersedes:** PLR-DR-0001 recommendations where this record is authoritative
**Decision date:** 2026-11-26
**Scope:** Lifecycle ownership, operation model, state semantics, runtime boundary, authorization, persistence, failure policy, version behavior, and SDK evolution

## 1. Context

The Plugin Module currently owns the installed Plugin aggregate, installation records, startup installation recovery, and read-only discovery. The domain enum exposes `Installed`, `Enabled`, `Disabled`, `Running`, and `Error`, but there is not yet an accepted lifecycle state machine or runtime implementation.

Installation state and runtime lifecycle state have different invariants and recovery semantics. The existing installation journal must not be overloaded to represent enablement, runtime process state, or runtime failures.

## 2. Accepted decisions

### D1 — Lifecycle ownership

**Accepted: Plugin Module owns lifecycle state invariants; Core owns authorization policy.**

The Module validates and serializes state transitions for its Plugin aggregate and persists lifecycle intent and outcomes. Core remains the sole policy decision point. The Module must not implement a second authorization engine.

### D2 — Lifecycle operations

**Accepted: Enable, Disable, Start, and Stop are the eventual control model, introduced incrementally.**

The first lifecycle implementation slice is durable Enable/Disable. Start/Stop must not be exposed or implemented until the runtime contract, health semantics, and process supervision behavior are separately accepted. Enablement alone does not assert that a Plugin process is running.

### D3 — State semantics

**Accepted: Separate desired administrative state from observed runtime state.**

The durable administrative intent is Enabled or Disabled. Runtime state is a separate observation and must include an explicit unknown/non-running representation. A persisted observation of Running is never proof that a process survived a host restart.

The existing public `PluginState` enum remains unchanged for compatibility in this increment. Its values are legacy presentation/projection values, not a sufficient internal lifecycle data model. A future migration must define projection and compatibility behavior before changing its wire or persistence representation.

### D4 — Runtime boundary

**Accepted: Define a narrow runtime port; target out-of-process isolation.**

The Module may define an SDK-neutral runtime boundary, but this decision does not select process, container, or WASM technology. Third-party Marketplace code must not execute inside the PRIDE Core process. Go's `plugin` package is not an accepted default execution mechanism.

### D5 — Authorization

**Accepted: Core AuthorizationPort, deny by default.**

For each privileged lifecycle/runtime operation, Core evaluates caller identity/context, operation, Plugin identity and immutable selected version, verified platform permission metadata, and host policy. The Module proceeds only on a positive, operation-scoped decision. Permission IDs are declarations/inputs to policy; they are never grants.

### D6 — Persistence and concurrency

**Accepted: Separate durable lifecycle operation journal and per-Plugin serialization.**

Lifecycle operations must be durable and retry-safe. Conflicting operations for one Plugin are serialized; unrelated Plugins may progress independently. Caller-supplied request/operation IDs provide idempotency. Lifecycle journal states and recovery rules are specified separately from installation journal states.

### D7 — Failure and restart policy

**Accepted: Host-configured bounded restart policy; no unbounded automatic restart.**

Core/host owns restart budgets, backoff, and operator overrides. The Module records observed health and bounded safe failure codes. The default policy is no automatic restart unless a bounded host policy explicitly enables it.

### D8 — Installed versions and updates

**Accepted: Immutable installed versions with explicit selected-version identity.**

Lifecycle operations target an explicit selected installed version. Updating must not mutate an installed artifact directory or silently replace code used by a running instance. Stop/restart, selection switch, and rollback semantics require a separate contract before update orchestration is implemented.

### D9 — SDK protocol evolution

**Accepted: Keep the Plugin SDK discovery-only for now.**

No lifecycle Protobuf methods are added in this increment. Typed lifecycle RPCs may be introduced only after the application contract, authorization behavior, idempotency, and safe error semantics are stable. SDK adapters remain transport/mapping-only.

## 3. Cross-cutting invariants

1. Core is the only authorization authority.
2. No untrusted Plugin code executes inside the Core process.
3. Only verified platform metadata may be supplied to Core policy; metadata does not itself authorize.
4. Administrative intent and runtime observation are distinct facts.
5. Installation recovery and lifecycle recovery are separate state machines and journals.
6. Conflicting operations for one Plugin are serialized.
7. Cancellation, deadlines, and typed safe failures propagate across boundaries.
8. Runtime output and Plugin-provided diagnostics are untrusted, bounded, and redacted before exposure.
9. Discovery readiness does not imply runtime or authorization readiness.
10. Installed artifact versions are immutable.

## 4. Consequences

- The first implementation work is an SDK-neutral lifecycle domain contract, followed by typed authorization/runtime ports and persistence contracts.
- No lifecycle schema migration, runtime adapter, process supervisor, authorization implementation, or SDK RPC is implied by this ADR.
- The Core composition root and production host integration remain explicit dependencies, not assumed existing capabilities.
- The legacy PluginState enum is retained until a separately reviewed compatibility migration is justified.

## 5. Implementation sequence

1. Define lifecycle records, desired/runtime state semantics, transition table, and operation invariants.
2. Define SDK-neutral AuthorizationPort and RuntimePort contracts with typed outcomes.
3. Define lifecycle journal persistence and per-Plugin serialization.
4. Implement Enable/Disable application service and crash/retry tests.
5. Specify Start/Stop and runtime recovery once the concrete runtime contract is selected.
6. Select and implement isolated runtime adapter and bounded host supervision.
7. Add lifecycle SDK protocol only after application contracts stabilize.
8. Complete host composition, readiness, observability, and end-to-end tests.

## 6. Explicit non-goals

This record does not authorize implementation of Plugin execution, sandboxing, process/container/WASM selection, policy storage, lifecycle database migrations, lifecycle RPCs, automatic restart, uninstall, rollback, or version garbage collection.
