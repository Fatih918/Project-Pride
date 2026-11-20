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
6. Construct the trusted platform metadata resolver. It must return platform-owned Plugin type/protected metadata; never derive these fields from developer-controlled Manifest JSON.
7. Construct the recovery coordinator with the real operation repository, Plugin registry repository, confined quarantine storage, package inspector, platform metadata resolver, and artifact verifier.
8. Acquire the host's cross-process startup/recovery exclusion mechanism.
9. Call `startup.Initialize(ctx, recoverer, pluginRepository)`.
10. Publish the discovery service and mark Plugin discovery ready only after initialization succeeds. Keep discovery and Plugin execution unavailable on any error.
11. Release the startup exclusion only after recovery and readiness publication have completed. The host must prevent a second process from racing the recovery window.

The current `startup.Initialize` implements step 9's fail-closed ordering only. It does not acquire a cross-process lock, create dependencies, or publish application readiness.

## 3. Required production dependencies

| Dependency | Host responsibility | Fail-closed condition |
|---|---|---|
| Registry descriptor resolver | Resolve opaque immutable source references to exact signed Marketplace descriptor bytes using an approved authenticated Registry API | No approved endpoint/authentication contract, unsigned or synthesized descriptor, or unavailable required service |
| Trust configuration source | Fetch signed trust configuration from the distribution-configured HTTPS endpoint | Invalid endpoint policy, transport failure, invalid signature, stale/rollback sequence, or unavailable bootstrap trust |
| Bootstrap public keys | Supply release-controlled Ed25519 trust anchors through the distribution/build process | Missing, malformed, or unapproved key material |
| Artifact host allowlist | Supply exact HTTPS hostnames permitted by the acquisition adapter | Empty or invalid allowlist |
| Platform metadata resolver | Resolve trusted platform-owned Plugin metadata for a Plugin ID and version | Missing or ambiguous metadata |
| SQLite repositories | Persist Plugin records, installed versions, operation journal, and trust sequence | Migration, integrity, or database errors |
| Confined storage | Provide operation workspaces, artifact reads/writes, extraction, promotion, and quarantine under one canonical storage root | Path escape, symlink violation, unsafe destination, or filesystem error |

No production URL, credential, bootstrap key, Marketplace signer, or authentication mechanism may be invented in the Module to make composition appear complete.

## 4. Readiness and failure behavior

- The host must expose no Plugin discovery service before recovery succeeds.
- The host must not enable Plugin loading, lifecycle transitions, or execution merely because discovery initialization succeeded; those capabilities require their own implemented and authorized runtime boundaries.
- A recovery error means startup is not ready. The host must retain the error for diagnostics and must not silently continue with an empty or partially recovered registry.
- If initialization fails after resources are opened, the host must close them in reverse construction order and leave durable operation records/artifacts for a later recovery attempt.
- A transient trust, metadata, or database error during PROMOTED recovery must remain retryable. The host must not convert it into an artifact-invalid result.
- The host should expose a health/readiness state that distinguishes Plugin subsystem unavailable, recovery in progress, and discovery ready. It must not expose secrets or internal credential material in health responses.

## 5. Cross-process coordination

Until a cross-process recovery lease/lock is implemented and verified, the host must guarantee that only one process can perform Plugin startup recovery against a given Plugin_Storage root and SQLite database at a time.

An in-process mutex is insufficient. A process-local flag, PID file without robust stale-owner handling, or SQLite connection pool setting is not a cross-process exclusion mechanism.

The chosen mechanism must:
- be scoped to the canonical Plugin data root/database identity;
- prevent two hosts from running recovery concurrently;
- be acquired before calling `startup.Initialize`;
- be released on success, failure, cancellation, and process shutdown;
- handle abnormal process termination without permanently blocking future startup;
- define behavior for unsupported/network filesystems;
- have concurrency, crash-release, and stale-owner tests.

The Module currently does not provide this mechanism. Until it does, single-process startup is an explicit deployment constraint, not an optional recommendation.

## 6. Shutdown and resource ownership

The host owns the lifetime of SQLite handles, HTTP transports, trust sources, storage adapters, and the discovery service. Shutdown must stop accepting new Plugin operations before closing dependencies. The host must not close repositories while recovery or discovery requests are in flight.

## 7. Production enablement checklist

Plugin installation must remain disabled until all of the following are true:

- [ ] Official Registry API and authenticated descriptor-resolution contract are approved and implemented.
- [ ] Marketplace signing and trust-configuration service are operational.
- [ ] Release-controlled Ed25519 bootstrap keys are provisioned and rotation/revocation procedures are tested.
- [ ] Production platform metadata resolver is implemented.
- [ ] Production host composition wires the actual adapters and calls `startup.Initialize` before exposing discovery.
- [ ] Cross-process recovery exclusion is implemented and tested.
- [ ] Startup failure cleanup and readiness behavior are tested.
- [ ] End-to-end installation/recovery is tested with real SQLite, filesystem, ZIP inspector/extractor, and cryptographic verification adapters.

Completing this document does not satisfy the checklist. It defines the boundary and acceptance criteria for subsequent implementation work.
