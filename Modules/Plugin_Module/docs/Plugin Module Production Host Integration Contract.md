# Plugin Module Production Host Integration Contract

**Status:** Draft contract for the production host. This document defines the composition boundary; it does not claim that a production host currently exists or that installation is enabled.

## 1. Purpose and boundary

The Plugin Module provides SDK-neutral domain, registry, discovery, installation, persistence, storage, verification, and recovery components. It does not own the process entry point, application configuration, service lifecycle, Registry credentials, release trust-root provisioning, or the decision to expose Plugin discovery to callers.

The production host (Core/application composition root) is responsible for constructing the real adapters, wiring them into the Module, and controlling readiness. The host must not bypass the startup gate.

## 2. Required startup ordering

The host must execute these steps in order:

1. Load and validate the host configuration. Missing production Registry endpoints, trust endpoints, artifact host allowlists, or bootstrap public keys are fatal when installation is enabled.
2. Resolve the PRIDE user-data root and initialize the Plugin_Storage layout. Resolve the SQLite database path as an absolute path beneath the configured application data root.
3. Open the SQLite-backed registry and installation-operation repositories. Apply supported migrations and verify that the database is usable. The host owns and closes these resources.
4. Construct the confined storage adapter against the canonical Plugin_Storage root. Do not pass arbitrary user-controlled absolute paths into storage adapters.
5. Construct the ZIP package inspector/extractor and artifact verifier using the approved PZIP-0001 policy and the production trust resolver.
6. Construct the recovery coordinator with the real operation repository, Plugin registry repository, confined quarantine storage, package inspector, and artifact verifier. The verifier must extract platform metadata from the exact descriptor bytes and return it only after successful Ed25519 and SHA-256 verification.
7. Ensure the canonical Plugin_Storage directory exists and acquire `startup.AcquireRecoveryLock(ctx, layout.StartupRecoveryLockPath())`.
8. Retain the returned lock handle for the entire lifetime of the Plugin subsystem, not merely until recovery completes. This prevents a second process from recovering an operation while the first process is actively installing or updating a Plugin.
9. Call `startup.Initialize(ctx, recoverer, pluginRepository)`.
10. Publish the discovery service and mark Plugin discovery ready only after initialization succeeds. Keep discovery and Plugin execution unavailable on any error.
11. During shutdown, stop accepting new Plugin operations, wait for in-flight operations to finish, close Module dependencies, and then release the recovery lock. On initialization failure, close constructed dependencies and release the lock.

`startup.Initialize` implements the fail-closed discovery ordering. `startup.AcquireRecoveryLock` provides the process-shared advisory lock; the host remains responsible for acquiring it before initialization and retaining it until Plugin subsystem shutdown.

## 3. Required production dependencies

| Dependency | Host responsibility | Fail-closed condition |
|---|---|---|
| Registry descriptor resolver | Wire the implemented HTTPS resolver with host-configured namespace origins and a host-owned audience-scoped token provider; resolve exact signed Marketplace descriptor bytes | Missing/invalid origin mapping or token provider, unsigned or synthesized descriptor, identity mismatch, or unavailable required service |
| Trust configuration source | Fetch signed trust configuration from the distribution-configured HTTPS endpoint | Invalid endpoint policy, transport failure, invalid signature, stale/rollback sequence, or unavailable bootstrap trust |
| Bootstrap public keys | Supply release-controlled Ed25519 trust anchors through the distribution/build process | Missing, malformed, or unapproved key material |
| Artifact host allowlist | Supply exact HTTPS hostnames permitted by the acquisition adapter | Empty or invalid allowlist |
| Signed platform metadata | Marketplace emits descriptor schema 1.1.0 with the required typed metadata; the verifier validates it and exposes it only after signature/digest verification | Missing, malformed, unsupported, or unsigned metadata; descriptor schema 1.0.0 cannot be installed or auto-committed during recovery |
| SQLite repositories | Persist Plugin records, installed versions, operation journal, and trust sequence | Migration, integrity, or database errors |
| Confined storage | Provide operation workspaces, artifact reads/writes, extraction, promotion, and quarantine under one canonical storage root | Path escape, symlink violation, unsafe destination, or filesystem error |

No production URL, credential, bootstrap key, Marketplace signer, or authentication mechanism may be invented in the Module to make composition appear complete.

## 4. Readiness and failure behavior

- The host must expose no Plugin discovery service before recovery succeeds.
- The host must not enable Plugin loading, lifecycle transitions, or execution merely because discovery initialization succeeded; those capabilities require their own implemented and authorized runtime boundaries.
- A recovery error means startup is not ready. The host must retain the error for diagnostics and must not silently continue with an empty or partially recovered registry.
- If initialization fails after resources are opened, the host must close them in reverse construction order and leave durable operation records/artifacts for a later recovery attempt.
- A transient trust or database error during PROMOTED recovery must remain retryable. The host must not convert it into an artifact-invalid result.
- The host should expose a health/readiness state that distinguishes Plugin subsystem unavailable, recovery in progress, and discovery ready. It must not expose secrets or internal credential material in health responses.

## 5. Cross-process coordination

The host must hold the cross-process recovery lock for the full lifetime of the Plugin subsystem. Serializing only the recovery call is insufficient: another process must not recover the first process's live PREPARING/STAGED installation operation.

An in-process mutex is insufficient. A process-local flag, PID file without robust stale-owner handling, or SQLite connection pool setting is not a cross-process exclusion mechanism.

The Module exposes `startup.AcquireRecoveryLock` and `Layout.StartupRecoveryLockPath`. The lock is an OS advisory lock on a persistent file scoped to the canonical Plugin_Storage root. It is released on explicit close or automatically by the operating system when the process exits abnormally. The lock file itself must never be deleted while processes may use the storage root.

The lock is supported on Linux, macOS, and Windows. Other operating systems fail closed. It requires a local filesystem with reliable advisory file-lock semantics; network filesystems and shared folders are unsupported unless their locking behavior is independently validated. The host must acquire the lock before `startup.Initialize` and retain it until Plugin subsystem shutdown. An in-process mutex or a PID file is not a substitute.

## 6. Shutdown and resource ownership

The host owns the lifetime of SQLite handles, HTTP transports, trust sources, storage adapters, and the discovery service. Shutdown must stop accepting new Plugin operations before closing dependencies. The host must not close repositories while recovery or discovery requests are in flight.

## 7. Production enablement checklist

Plugin installation must remain disabled until all of the following are true:

- [ ] Production Registry namespace-to-origin configuration and a real host-owned audience-scoped token provider are supplied and integration-tested with the implemented descriptor resolver.
- [ ] Marketplace signing and trust-configuration service are operational.
- [ ] Release-controlled Ed25519 bootstrap keys are provisioned and rotation/revocation procedures are tested.
- [ ] Marketplace signing pipeline emits schema 1.1.0 descriptors with the accepted `platform_metadata` object, and end-to-end signature verification covers that exact object.
- [ ] Production host composition wires the actual adapters and calls `startup.Initialize` before exposing discovery.
- [ ] Cross-process recovery exclusion is implemented and tested.
- [ ] Startup failure cleanup and readiness behavior are tested.
- [ ] End-to-end installation/recovery is tested with real SQLite, filesystem, ZIP inspector/extractor, and cryptographic verification adapters.

Completing this document does not satisfy the checklist. It defines the boundary and acceptance criteria for subsequent implementation work.
