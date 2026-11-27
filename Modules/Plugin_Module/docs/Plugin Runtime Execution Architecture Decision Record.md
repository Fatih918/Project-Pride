# Plugin Runtime Execution Architecture Decision Record

**Status:** Accepted  
**Document ID:** PLR-ADR-0003  
**Decision date:** 2026-11-27  
**Supersedes:** The proposed recommendations in PLR-DR-002 where this record is authoritative  
**Depends on:** PLR-ADR-0001, PLR-DOM-0001, PLR-ADR-0002, and the lifecycle implementation

## 1. Context

The Plugin Module has durable Enable/Disable operations, lifecycle persistence with per-Plugin leases and fencing, lifecycle recovery classification, and startup gating. It does not execute Plugin code or provide a production runtime adapter.

PLR-ADR-0001 establishes that third-party Marketplace code must not execute inside the PRIDE Core process and that the concrete runtime technology requires a separate decision. PLR-DR-002 reviewed runtime ownership, isolation, process topology, lifecycle intent, IPC, authorization, resource controls, supervision, and immutable version binding.

The recommendations R1–R11 in PLR-DR-002 are accepted unchanged by the project owner on 2026-11-27 and are made normative by this ADR.

## 2. Accepted decisions

### D1 — Runtime orchestration ownership (R1)

**Accepted: Plugin Module owns lifecycle orchestration; the host supplies RuntimePort.**

The Plugin Module owns Plugin-specific lifecycle invariants, operation sequencing, selected-version binding, and safe state transitions. The host supplies platform runtime adapters and process/isolation dependencies. Core remains the sole authorization authority. The Module must not embed operating-system process APIs or independently define sandbox policy.

### D2 — Execution isolation (R2)

**Accepted: Third-party Plugin code runs out of process, outside both Core and Plugin Module processes.**

In-process loading, including Go's `plugin` package, is not an accepted execution mechanism for untrusted Marketplace code. A separate process is the minimum architectural boundary, not proof of complete sandboxing. Filesystem, network, identity, syscall, and resource restrictions remain host responsibilities.

This ADR does not select a container, WASM engine, or specific sandbox technology. WASM may be considered only with a separately reviewed capability and resource policy.

### D3 — Runtime adapter abstraction (R3)

**Accepted: SDK-neutral RuntimePort with opaque runtime-instance identity and typed outcomes.**

The Module must not depend on PID values as durable identity, command-line strings, shell fragments, environment maps, or OS-specific handles. A runtime instance is bound to Plugin ID, immutable Plugin version, and a host-generated runtime-instance ID. Platform adapters may use process IDs, containers, WASM instances, or another implementation internally.

The exact port signatures and typed result/error vocabulary are follow-up contract work.

### D4 — Process topology (R4)

**Accepted: One isolated worker instance per active Plugin runtime instance for the first production runtime.**

This provides a clear initial boundary for crash containment, resource accounting, identity binding, and termination. Process separation alone does not guarantee filesystem or network isolation. Any future pooling requires an explicit decision demonstrating equivalent isolation and per-Plugin accounting.

### D5 — Administrative enablement and runtime intent (R5)

**Accepted: Administrative desired state, runtime intent, and runtime observation are separate facts.**

Enable/Disable controls administrative eligibility and does not assert that a process has started or stopped. Before Start/Stop is implemented, the domain and persistence model must represent runtime intent independently from administrative desired state and observed runtime health.

At minimum:
- Administrative desired state answers whether the Plugin is administratively enabled.
- Runtime intent answers whether an eligible Plugin should be started or kept stopped.
- Runtime observation reports what the host has actually observed, including Unknown.

An Enabled-but-Stopped Plugin is valid and must not be automatically restarted merely because it is Enabled. A Disabled Plugin must never be newly activated. If a runtime is observed Running while its Plugin is Disabled, host reconciliation must terminate or quarantine it before reporting runtime readiness.

The current implementation does not yet persist runtime intent. This ADR does not imply that Start/Stop or automatic activation is implemented.

### D6 — Runtime startup identity and handshake (R6)

**Accepted: A bounded authenticated startup handshake is required before runtime readiness.**

Process creation is not proof of readiness. The host must verify a versioned handshake containing protocol version, Plugin ID, immutable Plugin version, and host-generated runtime-instance ID. The host authenticates the local peer using platform-supported process/channel identity; a Plugin-provided identity string is not authentication.

Mismatch, timeout, malformed messages, or unsupported protocol versions fail closed and trigger bounded termination.

### D7 — Host-to-runtime communication (R7)

**Accepted: Host-created local IPC with a versioned, bounded protocol.**

The host creates and owns the communication channel. A worker must not select its own listening address or accept arbitrary inbound network connections for host control. The protocol must define request IDs, deadlines/cancellation, maximum frame sizes, typed errors, and compatibility behavior. Every message is untrusted input and must be validated. Runtime IPC types remain independent of SDK Protobuf types.

### D8 — Permissions, credentials, and host APIs (R8)

**Accepted: Deny by default; privileged host capabilities are brokered explicitly.**

Verified platform permission metadata is policy input only. Core decides whether an operation is authorized. Workers receive no ambient host credentials, database handles, signing keys, Registry tokens, or unrestricted filesystem access.

Any future host capability must be explicit, scoped, revocable, audited, and authorized by Core. Permission IDs are never interpreted as grants by the runtime adapter.

### D9 — Resource and output limits (R9)

**Accepted: Enforce limits at the host boundary.**

The host runtime adapter must enforce configurable CPU, memory, process-count, execution-time, IPC-frame, stdout/stderr, and temporary-storage limits using controls supported by the deployment platform. Required isolation controls unavailable on a target platform must cause fail-closed behavior. Plugin output and diagnostics are untrusted, bounded, and redacted before logging or returning to callers.

Default values and platform-specific enforcement are separate deployment-policy decisions.

### D10 — Stop, crash, and restart behavior (R10)

**Accepted: Bounded host-owned supervision and explicit reconciliation.**

Graceful stop has a bounded deadline; a worker that does not exit is forcibly terminated. Stop and terminate requests are idempotent. Unexpected process exit updates runtime observation but does not silently change administrative desired state.

Automatic restart is disabled by default. Any future restart policy must be host-configured, bounded, observable, consistent with runtime intent, and subject to current authorization policy.

After host restart, old in-memory process handles are invalid. Reconciliation must establish fresh observations. Persisted Running is never proof that a process remains alive. Recovery must not reuse a previous authorization decision as a standing grant.

### D11 — Immutable runtime version binding (R11)

**Accepted: Every runtime instance is bound to the exact immutable Plugin version captured at start.**

Start captures Plugin ID, immutable Plugin version, verified artifact identity/digest, and runtime-instance ID. Inspect, stop, and terminate target that exact instance/version, not whichever version is currently selected later. Updating the selected version must not mutate or replace an artifact directory used by an existing runtime.

Update orchestration, rollback, and selection-switch semantics remain deferred until separately specified.

## 3. Normative cross-cutting invariants

1. Third-party Marketplace code never executes inside Core or Plugin Module processes.
2. Plugin Module owns lifecycle invariants; Core owns authorization; host owns process and isolation mechanisms.
3. Administrative enablement, runtime intent, process existence, handshake readiness, and observed health are distinct facts.
4. Every runtime instance is bound to one Plugin ID and one immutable installed version.
5. Workers receive no ambient credentials or unrestricted host capabilities.
6. Runtime IPC is host-created, authenticated, versioned, bounded, and validated.
7. Resource limits and termination are enforced by the host, not trusted to Plugin cooperation.
8. Runtime failures do not silently change administrative desired state.
9. Recovery never treats persisted Running as proof of a live process or old authorization as a standing grant.
10. SDK lifecycle RPCs remain deferred until application and runtime semantics are stable.

## 4. Consequences and implementation constraints

- Define the SDK-neutral RuntimePort contract before implementing a concrete worker adapter.
- Define and persist runtime intent before implementing Start/Stop or automatic activation.
- Use a fake RuntimePort to test lifecycle orchestration without executing third-party code.
- Select a concrete process/sandbox implementation only after the supported deployment OS matrix and threat model are documented.
- Production readiness requires host composition, authenticated IPC, enforced isolation/resource controls, crash reconciliation, and end-to-end security tests.
- The current Enable/Disable service remains administrative-only and must preserve runtime observation.

## 5. Explicit non-goals

This ADR does not implement:
- a runtime adapter, worker executable, process launcher, or sandbox;
- a Core authorization policy implementation or credential broker;
- runtime-intent schema migration or Start/Stop application operations;
- runtime observation reconciliation or automatic restart;
- platform-specific resource limits or deployment packaging;
- lifecycle SDK/Protobuf methods;
- update orchestration, uninstall, rollback, or version garbage collection.

## 6. Next implementation sequence

1. Define SDK-neutral RuntimePort request/result types, identity, and typed failure semantics.
2. Define the runtime-intent domain and persistence changes required to distinguish Enabled-but-Stopped from Enabled-and-Started.
3. Implement fake RuntimePort orchestration and failure/retry tests.
4. Document the supported host deployment matrix and threat model.
5. Select and implement the isolated worker adapter, authenticated local IPC, and host-enforced controls.
6. Implement runtime reconciliation and Start/Stop only after intent and recovery contracts are complete.
7. Complete host composition and end-to-end tests; add SDK lifecycle RPCs only in a separate protocol increment.
