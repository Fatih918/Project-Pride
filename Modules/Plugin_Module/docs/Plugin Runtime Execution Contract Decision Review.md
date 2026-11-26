# Plugin Runtime Execution Contract Decision Review

**Status:** Proposed  
**Document ID:** PLR-DR-002  
**Scope:** Runtime process isolation, host supervision, runtime protocol, execution identity, lifecycle semantics, and resource/security boundaries  
**Depends on:** PLR-ADR-0001, PLR-DOM-0001, PLR-ADR-0002, lifecycle application and persistence implementation

## 1. Purpose

The Plugin Module now has durable Enable/Disable operations, per-Plugin leases and fencing, lifecycle recovery classification, and startup gating. It does not execute Plugin code and does not have a concrete runtime adapter.

PLR-ADR-0001 already establishes two constraints:
- third-party Marketplace code must not execute inside the PRIDE Core process;
- the Module may define a narrow runtime boundary, while the concrete isolation technology remains a separate decision.

This review turns those constraints into an implementable runtime contract without prematurely selecting a platform-specific sandbox or adding SDK lifecycle RPCs.

## 2. Current implementation boundary

The current lifecycle application service persists administrative desired state only. It:
- resolves the selected immutable Plugin version;
- obtains a Core authorization decision;
- commits Enabled/Disabled state transactionally;
- preserves RuntimeObservation without inferring that a process started or stopped.

The current lifecycle repository and recovery coordinator do not launch, inspect, terminate, or reconcile runtime processes. The current `lifecycle/ports.go` defines `AuthorizationPort` and `LifecycleRepository`; it does not yet define a concrete runtime adapter contract.

The Core production composition root, host policy implementation, and supported production deployment matrix are not established by this review. They must not be assumed.

## 3. Decisions requested

### R1 — Runtime orchestration ownership

**Option A — Core directly owns runtime process orchestration.** Core calls platform process APIs and mutates Plugin lifecycle state.

**Option B — Plugin Module owns lifecycle orchestration; a host-provided RuntimePort performs process operations.**

**Recommendation: Option B.** The Plugin Module owns Plugin-specific lifecycle invariants, selected-version binding, operation sequencing, and safe state transitions. The host supplies the runtime adapter and platform dependencies. Core remains the authorization authority and must authorize privileged operations before the Module invokes the runtime port. The Module must not embed OS process APIs or sandbox policy.

### R2 — Execution isolation boundary

**Option A — In-process execution**, including Go's `plugin` package.  
**Option B — Out-of-process worker process.**  
**Option C — WebAssembly runtime embedded in the host.**

**Recommendation: Option B as the minimum architectural boundary.** Marketplace code must run outside both the Core process and the Plugin Module process. Go's `plugin` package is not acceptable for untrusted third-party code. WASM may become a runtime implementation later, but it must not be assumed to provide security without a separately reviewed WASI capability and resource policy.

This decision does not claim that a separate process alone is a complete sandbox. Filesystem, network, identity, syscall, and resource restrictions remain mandatory host responsibilities.

### R3 — Runtime adapter abstraction

**Option A — Expose generic process handles and OS-specific details to the Module.**  
**Option B — Define a Plugin-specific, SDK-neutral RuntimePort using opaque runtime-instance identity and typed outcomes.**

**Recommendation: Option B.** The Module must not receive PID values as durable identity, command-line strings, environment maps, shell fragments, or OS handles. A runtime instance identity is opaque and scoped to Plugin ID, immutable Plugin version, and a host-generated instance ID. Platform adapters may internally use PIDs, containers, WASM instances, or another mechanism.

The port contract should eventually cover start, graceful stop, forced termination, and observation/reconciliation. Exact method signatures are deferred to the follow-up boundary-contract slice.

### R4 — Process topology

**Option A — One shared worker process hosts multiple unrelated Plugins.**  
**Option B — One isolated worker instance per active Plugin runtime instance.**  
**Option C — Select topology per runtime implementation.**

**Recommendation: Option B for the first production runtime.** It gives clearer crash containment, resource accounting, identity binding, and termination semantics. It does not by itself prevent filesystem or network access. A future measured implementation may introduce pooling only if it preserves equivalent isolation and per-Plugin accounting.

### R5 — Administrative enablement versus runtime intent

**Option A — Enable immediately means process started; Disable means process stopped.**  
**Option B — Keep administrative enablement distinct from runtime activation intent and runtime observation.**

**Recommendation: Option B.** Enable/Disable remain administrative eligibility controls and must not claim process health. Start/Stop, when introduced, need explicit runtime-intent semantics. In particular, a Plugin that is Enabled but intentionally Stopped must not be automatically restarted by reconciliation. The domain/persistence contract must represent this distinction before Start/Stop are implemented; do not overload RuntimeObservation to represent operator intent.

A Disabled Plugin must never be newly activated. A runtime instance observed as Running while the Plugin is Disabled must be terminated or quarantined by the host reconciliation policy before runtime readiness is reported.

### R6 — Runtime identity and handshake

**Option A — Treat process creation as proof of runtime readiness.**  
**Option B — Require a bounded authenticated startup handshake.**

**Recommendation: Option B.** Process creation is not readiness. Before reporting an instance ready, the host must verify a versioned handshake containing at least the protocol version, Plugin ID, immutable Plugin version, and host-generated runtime-instance ID. The host must authenticate the local peer using platform-supported process/channel identity; a Plugin-supplied identity string is not authentication.

A mismatch, timeout, malformed message, or unsupported protocol version fails closed and triggers bounded termination. Runtime readiness must remain distinct from process existence.

### R7 — Host-to-runtime communication

**Option A — Expose a network listener on the Plugin worker.**  
**Option B — Use a host-created local IPC channel with a versioned, bounded protocol.**

**Recommendation: Option B.** The host creates and owns the communication channel. The worker must not choose its own listening address or accept arbitrary inbound network connections for host control. The protocol must define request IDs, deadlines/cancellation, maximum frame sizes, typed errors, and compatibility behavior. Every message is untrusted input and must be validated. No raw gRPC/SDK types should leak into the runtime boundary.

### R8 — Permissions, credentials, and host APIs

**Option A — Pass declared permission IDs and secrets directly to the worker as environment variables.**  
**Option B — Deny by default and broker privileged host capabilities through explicit host APIs.**

**Recommendation: Option B.** Verified platform permission metadata is policy input only. Core decides whether a principal/Plugin/version may perform an operation. The worker receives no ambient host credentials, database handles, signing keys, Registry tokens, or unrestricted filesystem access. Any future host capability must be explicit, scoped, revocable, audited, and authorized by Core. Permission IDs must never be interpreted as grants by the runtime adapter.

### R9 — Resource and output limits

**Option A — Rely on Plugin cooperation.**  
**Option B — Enforce host-side limits and bounded diagnostics.**

**Recommendation: Option B.** The host runtime adapter must enforce configurable CPU, memory, process-count, execution-time, IPC-frame, stdout/stderr, and temporary-storage limits using controls supported by the deployment platform. Limits must fail closed when required isolation controls are unavailable. Plugin output and diagnostics are untrusted, size-bounded, and redacted before logging or returning to callers. Default limits and platform-specific enforcement are a later deployment-policy decision.

### R10 — Stop, crash, and restart behavior

**Option A — Unbounded automatic restart.**  
**Option B — Bounded host-owned supervision with explicit reconciliation.**

**Recommendation: Option B.** Graceful stop uses a bounded deadline; if the worker does not exit, the host forcibly terminates it. Stop/terminate requests must be idempotent. Unexpected exit updates observed runtime health but must not silently change administrative enablement. Automatic restart is disabled by default; any future restart policy is host-configured, bounded, observable, and must respect runtime intent and current authorization policy.

After host restart, old in-memory process handles are invalid. Reconciliation must identify stale instances and establish a fresh observation; a persisted Running observation is never proof that the old process still exists. Recovery must not replay a prior authorization decision as a standing grant.

### R11 — Version binding and update safety

**Option A — Resolve the currently selected version each time a running worker is inspected or stopped.**  
**Option B — Bind every runtime instance to the exact immutable version captured at start.**

**Recommendation: Option B.** Start captures Plugin ID, immutable version, verified artifact location/digest, and runtime-instance ID. Stop, inspect, and termination target that exact instance and version, never whichever version is currently selected later. Updating selection must not mutate or replace an existing runtime's artifact directory. Update/rollback orchestration remains out of scope until explicit stop/restart and selection-switch semantics are accepted.

## 4. Cross-cutting invariants

1. No third-party Marketplace code executes inside the Core or Plugin Module process.
2. The Plugin Module owns lifecycle invariants; Core owns authorization; the host owns platform process and isolation mechanisms.
3. Process existence, handshake readiness, administrative enablement, runtime intent, and observed health are separate facts.
4. Every runtime instance is bound to one Plugin ID and one immutable installed version.
5. No ambient credentials or unrestricted host capabilities are inherited by the worker.
6. Runtime IPC is host-created, authenticated, versioned, bounded, and validated.
7. Resource limits and termination are enforced by the host, not trusted to Plugin cooperation.
8. Runtime failures do not silently change administrative desired state.
9. Recovery never treats persisted Running as proof of a live process and never reuses stale authorization evidence as a standing grant.
10. SDK lifecycle RPCs remain deferred until application and runtime semantics are stable.

## 5. Explicit non-goals

This review does not implement:
- a runtime adapter, process launcher, worker executable, sandbox, or WASM engine;
- a Core authorization policy implementation or credential broker;
- lifecycle Start/Stop application operations or Protobuf methods;
- a runtime database migration or runtime observation reconciler;
- platform-specific resource limits, syscall policy, or deployment packaging;
- automatic restart, update orchestration, uninstall, rollback, or version garbage collection.

## 6. Proposed implementation sequence after decisions

1. Accept the runtime ownership, isolation, identity, IPC, authorization, and lifecycle-intent decisions.
2. Define SDK-neutral RuntimePort request/result types and typed failure semantics.
3. Define the runtime-intent domain and persistence changes required to distinguish Enabled-but-Stopped from Enabled-and-Started.
4. Implement a fake RuntimePort and lifecycle orchestration tests without executing third-party code.
5. Select the first supported OS sandbox/process adapter based on the actual PRIDE deployment matrix and threat model.
6. Implement bounded worker supervision, authenticated local IPC, resource controls, and crash reconciliation.
7. Add Start/Stop application operations only after runtime intent and recovery contracts are implemented.
8. Add host composition and end-to-end tests; add SDK lifecycle RPCs only in a separate protocol increment.

## 7. Review outcome

This document is a proposal for explicit review. No recommendation is accepted until the project owner records the decisions in a normative ADR. Acceptance does not imply runtime production readiness or authorize executing third-party code.
