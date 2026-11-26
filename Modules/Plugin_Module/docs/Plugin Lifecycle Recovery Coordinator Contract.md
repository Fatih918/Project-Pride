# Plugin Lifecycle Recovery Coordinator Contract

**Status:** Implemented in the lifecycle persistence follow-up increment  
**Related decision:** PLR-ADR-0002, P8 — Crash recovery

## Purpose

The lifecycle SQLite repository persists operation phases and exposes bounded recovery inspection/classification. This coordinator is the application-level startup policy over that repository contract. It is not a runtime reconciler and does not resume interrupted commands.

## Startup algorithm

1. Call `MarkExpiredOperationsRecoveryRequired` in bounded batches of at most 1,000 operations until a batch is smaller than the configured batch size.
2. Read the bounded recovery-candidate set.
3. Permit startup only when no candidate has an active lease and no non-terminal operation remains outside `RecoveryRequired`.
4. Keep `RecoveryRequired` operations durable for audit and idempotency. They are not automatically authorized again or resumed.
5. Fail closed on repository errors, context cancellation, an active lease, an unclassified candidate, or a candidate scan that reaches the 1,000-row safety limit.

The repository contract currently has no cursor-based pagination. An exactly-full 1,000-candidate result is therefore treated as potentially truncated and blocks startup rather than allowing an incomplete scan to produce a false-safe result.

## Startup integration

Hosts should call `startup.InitializeWithLifecycle` while holding the existing cross-process Plugin startup lock. It performs installation recovery first, lifecycle recovery second, and only then returns the discovery service.

The host must construct and expose lifecycle mutation entry points only after this function succeeds. The lock must remain held for the complete Plugin subsystem lifetime. On any initialization error, the host must close dependencies and expose neither discovery nor lifecycle mutation services.

## Safety boundaries

- Active operations are not stolen or resumed; startup fails with a typed in-progress error.
- Expired and lease-less non-terminal operations are atomically classified by the repository as `RecoveryRequired`.
- Previous authorization evidence remains audit evidence only; a new operation requires a new authorization decision.
- Recovery never changes `RuntimeObservation`.
- `RecoveryRequired` is an explicit unresolved outcome, not proof of success or failure.
- This coordinator does not provide production host composition, lifecycle runtime execution, process reconciliation, or Core authorization.
