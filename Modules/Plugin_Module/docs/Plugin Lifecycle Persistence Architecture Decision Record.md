# Plugin Lifecycle Persistence Architecture Decision Record

**Status:** Accepted  
**Document ID:** PLR-ADR-0002  
**Decision authority:** PLR-ADR-0001, PLR-DOM-0001, PLR-BND-0001  
**Scope:** SQLite lifecycle journal, desired-state persistence, idempotency, leases/fencing, retention, and crash recovery  
**Out of scope:** Runtime execution, Core authorization implementation, SDK lifecycle RPCs, uninstall, update orchestration

## 1. Purpose

The lifecycle domain, boundary contracts, and Enable/Disable application service are now present. The next implementation step is a durable SQLite adapter. This review makes the remaining persistence decisions explicit before schema and adapter code are introduced.

The lifecycle repository remains logically separate from the installation journal. It may share the existing SQLite database and connection infrastructure, but it must not reuse installation operation tables or phases.

## 2. Accepted decisions

The project owner accepted P1–P10 without modification on 2026-11-26. The recommendations below are normative implementation decisions for the lifecycle persistence adapter.

### P1 — Persistence adapter placement

**Accepted: Implement a dedicated lifecycle repository adapter in the existing SQLite infrastructure package, backed by the same database handle as the Plugin registry and installation repositories.**

The adapter is a distinct type implementing `lifecycle.LifecycleRepository`; it does not add lifecycle methods to the installation repository contract. Sharing the database handle ensures migrations and transactions use one SQLite database and avoids independent connection/migration ownership. The adapter may live beside the existing SQLite repository implementation initially; a later infrastructure-package reorganization is not a prerequisite.

**Alternatives considered**
- A separate lifecycle SQLite database: rejected because lifecycle records must be transactionally consistent with Plugin installation/selected-version identity and complicate host backup/recovery.
- A second independently opened handle with independent migration ownership: rejected for the first implementation because it creates avoidable schema-version and initialization coordination.
- Reusing installation operation tables: rejected because installation and lifecycle have different state machines and recovery semantics.

### P2 — Schema evolution

**Recommendation: Add schema version 5 to the existing SQLite database using the current transactional `PRAGMA user_version` migration mechanism.**

Create separate tables for:
- `plugin_lifecycle_records`: one row per Plugin, containing selected version, desired state, runtime observation, revision, and timestamps.
- `plugin_lifecycle_operations`: immutable operation identity, request fingerprint, principal, Plugin/version/action, phase, failure code, authorization decision reference, policy revision, and timestamps.
- `plugin_lifecycle_leases`: current per-Plugin operation owner, fencing token, and expiry.
- `plugin_lifecycle_fencing_tokens`: durable per-Plugin monotonic fencing-token counter.

Use constrained text representations for enum values, explicit CHECK constraints, primary/unique indexes, and bounded column validation in Go. Migration must preserve all schema v4 data and be transactional.

### P3 — Operation identity and idempotency fingerprint

**Recommendation: Fingerprint the caller-visible immutable command identity: authenticated PrincipalID, PluginID, and Action. Persist the exact selected PluginVersion separately on the operation at first reservation.**

This preserves the existing service behavior: a retry is looked up by OperationID before resolving the currently selected version, so a version change after the original operation does not turn a valid retry into a conflict or cause it to target new code.

Rules:
- Same OperationID and same fingerprint: return the stored operation/outcome; never execute twice.
- Same OperationID and different fingerprint: return `ErrOperationIDConflict`.
- Same fingerprint but a different currently selected version on a later retry: replay the original operation and original version.
- A different OperationID is a new command and resolves the currently selected version.
- The repository must compare fingerprints inside the reservation transaction, not rely only on application-level checks.

The stored PluginVersion remains immutable audit evidence and is always supplied to Core authorization for the original operation.

### P4 — Per-Plugin lease and fencing

**Recommendation: Use a durable database-backed lease with a monotonically increasing positive signed-64-bit fencing token per Plugin (matching SQLite INTEGER). Initial lease duration: 120 seconds.**

Reservation and lease acquisition occur in one write transaction:
1. Check whether OperationID already exists and compare its fingerprint.
2. Reject a different active operation for the same Plugin.
3. If no unexpired lease exists, increment the durable per-Plugin fencing counter and assign the new token. If the counter reaches SQLite's maximum signed 64-bit integer, reservation fails closed rather than wrapping or reusing a token.
4. Persist the operation and lease atomically.

Every mutating method must verify OperationID, PluginID, fencing token, and lease expiry in the same transaction as its state mutation. A stale or expired worker receives `ErrLeaseLost`. Release is idempotent and may expire/delete only the exact lease owner/token; it must never release a newer owner's lease.

The repository clock is authoritative for lease creation, expiry, and comparisons. Clock access must be injectable for deterministic tests. Wall-clock rollback must not permit a stale token to commit; fencing-token equality remains mandatory even when a lease is considered unexpired.

**Alternative rejected:** process-local mutexes, because they do not protect against multiple processes or stale workers.

### P5 — Lease renewal

**Recommendation: Do not add renewal in the first adapter increment. Keep the application operation bounded by the host request deadline and a 120-second lease.**

The initial flow has no external runtime side effect and performs one Core authorization call followed by short SQLite transactions. If production measurements show a need for longer operations, add an explicit fenced renewal method in a separately reviewed contract change. The adapter must fail closed on expiry; it must not silently extend a lease during unrelated writes.

### P6 — Lifecycle record initialization and revision

**Recommendation: Lazily create the lifecycle record on the first successful desired-state commit. The initial baseline is Disabled + Unknown, revision 1.**

- The first successful Enable produces Enabled + Unknown, revision 1.
- The first successful Disable produces Disabled + Unknown, revision 1.
- Subsequent desired-state changes increment revision exactly once.
- An authorized no-op records a successful operation but does not increment revision.
- RuntimeObservation is preserved exactly by Enable/Disable commits; this slice must not infer Running or Stopped.
- SelectedVersion must match the immutable version captured by the reserved operation and the currently selected installed version at first commit. A later version selection change must not silently retarget an in-flight operation.

If the selected installed version changed between reservation and commit, the commit fails safely and does not mutate the lifecycle record.

### P7 — Authorization evidence and operation phases

**Recommendation: Persist the complete safe decision evidence required by the domain operation: DecisionID, PolicyRevision, EvaluatedAt, and bounded ReasonCode. Persist no credentials, raw policy text, or raw adapter errors.**

- An explicit denial is recorded before the operation transitions to Failed.
- Authorization transport errors or malformed decisions fail closed and are recorded with bounded failure codes; they are not represented as a denial.
- `RecordAuthorization` must be fenced and transition only the matching Requested operation.
- `BeginApply` transitions Authorized → Applying under the same current lease.
- `CommitDesiredState` atomically writes the lifecycle record and transitions Applying → Succeeded.
- `FailOperation` may mark only a non-terminal operation Failed. It must not overwrite Succeeded or another terminal outcome.

The existing domain operation may be extended with EvaluatedAt and ReasonCode if retaining those fields is accepted; otherwise the persistence adapter must document why they are intentionally not part of the durable domain record. Do not store extra opaque fields without a domain contract.

### P8 — Crash recovery

**Recommendation: Add a lifecycle recovery query and coordinator before enabling lifecycle mutations in production. For the first SQLite adapter, expose incomplete operations and classify expired operations as RecoveryRequired; do not replay authorization or infer an outcome.**

The initial Enable/Disable flow has no RuntimePort side effects, so a process crash before the atomic desired-state commit leaves no external Plugin execution to reconcile. Recovery rules:
- Succeeded and Failed are terminal and returned unchanged.
- Requested, Authorized, or Applying operations with a valid active lease remain in progress.
- Expired non-terminal operations become RecoveryRequired under a new recovery fencing token or a transactionally equivalent recovery claim.
- RecoveryRequired operations are not automatically resumed using an old authorization decision.
- A caller retry with the same OperationID receives a typed recovery/in-progress result; an operator or explicit retry policy must decide whether to create a new operation ID and obtain fresh authorization.
- Recovery must never modify RuntimeObservation.

The recovery coordinator and startup gate are a follow-up implementation slice; this adapter PR must not claim lifecycle production readiness until they exist.

### P9 — Operation retention

**Recommendation: Retain lifecycle operation rows indefinitely in the initial implementation.**

No cleanup job or automatic retention policy is introduced until an audit/retention requirement is defined. Non-terminal and RecoveryRequired operations must never be deleted. A future retention design must preserve idempotency guarantees and terminal audit evidence.

### P10 — Runtime observation ownership

**Recommendation: SQLite lifecycle persistence may store RuntimeObservation, but only a future RuntimePort/reconciliation coordinator may update it. Enable/Disable application service may only preserve the current observation.**

This prevents administrative intent from being mistaken for actual process health and keeps the initial application service free of runtime side effects.

## 3. Required invariants and test matrix

The implementation must prove at least:

1. Migration from schema v4 to v5 preserves existing registry, installation, and trust data.
2. Fresh database migration creates all lifecycle tables and constraints.
3. Same operation ID + same fingerprint replays without acquiring a second lease.
4. Same operation ID + different fingerprint returns conflict.
5. Concurrent operations for one Plugin cannot both acquire an active lease.
6. Different Plugins can hold independent leases.
7. Fencing tokens increase monotonically after expiry/release and across repository reopen.
8. Expired/stale fencing tokens cannot record authorization, begin apply, fail, or commit.
9. Lease release cannot release a newer owner's lease.
10. Authorization evidence and phase changes are transactionally consistent.
11. Desired state and Succeeded phase commit atomically.
12. A failed transaction leaves both desired state and operation phase unchanged.
13. RuntimeObservation is preserved by desired-state commits.
14. No-op commits do not increment revision.
15. Selected-version mismatch fails without mutating state.
16. Recovery queries are deterministic and never treat old authorization as fresh.
17. All persisted values are bounded and validated; malformed rows fail closed.

## 4. Explicit non-goals

This review does not authorize:
- RuntimePort implementation or Plugin process execution;
- Start/Stop operations;
- lifecycle SDK RPCs or Protobuf changes;
- Core authorization policy implementation;
- automatic restart;
- uninstall, update switching, rollback, or artifact garbage collection;
- automatic lifecycle operation pruning;
- production host composition or readiness claims.

## 5. Implementation requirements

The implementation PR must follow P1–P10. Any required deviation must be brought back for explicit review before implementation. Update PLR-BND-0001 if the concrete repository semantics require clarification.
