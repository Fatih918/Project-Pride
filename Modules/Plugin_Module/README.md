# PRIDE Plugin Module

**Status:** Increment 14 — bounded ZIP inspection/extraction, signed artifact verification, persistent trust-sequence storage, bounded HTTP artifact acquisition, bounded HTTPS trust transport, and confined filesystem promotion implemented.

The Plugin Module owns SDK-neutral installed Plugin records, local persistence, storage layout, and read-only discovery. The SDK integration adapter is isolated in the nested `sdkadapter` Go module.

## Increment 13 — Trust configuration transport

- Added `artifactverify.HTTPTrustConfigurationSource` for a distribution-configured HTTPS endpoint; no Marketplace endpoint or authentication protocol is invented.
- Requires an exact hostname allowlist, HTTPS, port 443, TLS 1.2+, no redirects, no environment proxy, and DNS resolution that rejects non-public destinations.
- Bounds response headers to 64 KiB, trust JSON to 1 MiB, and request timeout to 1–60 seconds; requires HTTP 200 and `application/json`.
- Keeps transport separate from trust: fetched bytes remain untrusted until `TrustResolver` verifies the Ed25519 signature against distribution-provisioned bootstrap public keys.
- Documents the release-controlled bootstrap-key provisioning boundary. No fabricated production key or insecure environment override is included.
- The actual trust endpoint, bootstrap public-key value, Marketplace service authentication, and production composition root remain distribution/service responsibilities.

## Increment 12 — Registry artifact acquisition

- Added an SDK-neutral `DescriptorResolver` port that resolves an opaque immutable source reference into the exact signed Marketplace descriptor.
- Added `installation/acquisition.HTTPArtifactAcquirer` to download the ZIP from the descriptor's HTTPS locator while preserving the exact descriptor bytes for downstream verification.
- Enforced exact artifact-host allowlisting, TLS 1.2+, no redirects/proxies, non-public DNS address rejection, bounded timeouts/headers, context-aware streaming, and PZIP-0001's 100 MiB compressed-size ceiling.
- Required actual downloaded bytes to match the descriptor's declared size; partial artifact files are removed after acquisition failures.
- Kept signature and SHA-256 verification in `installation/artifactverify`; acquisition does not treat downloaded metadata as trusted.
- The production Official Registry resolver remains an injected port because no Registry API/endpoint contract has been approved.

## Implemented

### Domain and registry foundation
- Domain types for Plugin identity, developer-owned Manifest data, Plugin type/state, capability declarations, and protected permission metadata.
- Domain validation for required Manifest fields, identity consistency, valid type/state values, and duplicate/empty capability or permission identifiers.
- A narrow registry repository interface with create, get, and list operations.
- A deterministic in-memory repository for tests and local development, with duplicate-ID rejection and defensive copies.

### Discovery application service
- Five read-only queries: list, get, find by exact capability ID, get metadata, and get operational state.
- Type and state filters, deterministic Plugin ID ordering, opaque versioned pagination tokens, and context/error propagation.

### SDK discovery integration
- A separate `sdkadapter` Go module implements `pride-plugin-sdk/server.DiscoveryPort`.
- All five SDK discovery operations delegate to the Module discovery application service.
- Explicit mapping between Module domain types and SDK-neutral server DTOs.
- Stable domain-error mapping; cancellation and deadlines are preserved.
- gRPC `bufconn` integration tests cover all five RPCs.

### Local storage and SQLite persistence
- Platform-aware PRIDE user-data root resolution, including `PRIDE_USER_DATA_DIR` override.
- Local `Plugin_Storage` layout for installed artifacts, plugin data, cache, temporary files, and quarantine.
- Validated path helpers with traversal and symlink-containment checks.
- Operation-scoped temporary and quarantine path resolution, plus exclusive private staging-directory creation.
- Root-confined filesystem adapter for operation workspaces and exclusive `artifact.zip` creation/read/removal.
- Canonical `Plugin_Storage`-relative path generation and safe resolution to absolute OS paths; persisted paths never depend on the process working directory.
- SQLite-backed implementation of the existing `registry.Repository` contract using `modernc.org/sqlite`.
- Versioned schema initialization, deterministic listing, context-aware operations, JSON persistence, and repository error mapping.
- SQLite schema v3 persists the accepted trust sequence and raw-configuration SHA-256 digest; a single conditional UPSERT atomically rejects rollback/equivocation and permits exact replay.

### Installation application service
- Package inspection returns developer-owned Manifest data separately from platform-owned Plugin type, state, capabilities, and permission metadata.
- Bounded ZIP inspector performs classic-ZIP preflight, ZIP64 rejection, path/type/entry validation, duplicate-key-safe Manifest decoding, and PZIP-0001 resource checks.
- Secure ZIP extractor revalidates the staged archive, streams only `package/` payload files, enforces actual per-file and cumulative byte limits, and removes partial output on failure.
- Artifact verifier validates the versioned Registry descriptor, checks Plugin/version/media type/HTTPS locator, verifies Ed25519 signatures over RFC 8785 JCS canonical payloads, and hashes the exact staged ZIP with SHA-256.
- Trust resolver authenticates remote trust configurations against distribution-provisioned Ed25519 bootstrap keys, enforces key validity/revocation, and delegates monotonic sequence/equivocation protection to the SQLite-backed store.
- Bounded HTTPS trust transport retrieves the signed trust document without treating transport success as authenticity.
- Installation orchestration verifies before extraction and verifies again after extraction before recording VERIFIED, reducing the chance that a changed staged archive is promoted.
- Storage adapter resolves immutable installation destinations as canonical Plugin_Storage-relative paths and creates extraction outputs exclusively through its `os.Root` confinement.
- A dedicated injected metadata resolver supplies platform metadata; `plugin.json` cannot author platform-owned fields.
- SDK-neutral orchestration of operation-scoped staging creation, artifact acquisition, package inspection, payload extraction, artifact verification, filesystem promotion, and atomic installation persistence through injected ports.
- Confined filesystem promotion adapter atomically renames the complete verified operation workspace from `temporary/<operation-id>` to `installed/<plugin-id>/<version>` on the same Plugin_Storage filesystem; validates canonical paths and staged contents, and rejects an existing destination.
- Durable operation phase recording and recoverable handling when the post-promotion database commit fails.

## Architectural boundaries

- Domain, registry, discovery, and installation application packages do not depend on generated Protobuf types or the Plugin SDK.
- SDK integration dependency is isolated in `sdkadapter/go.mod`.
- The SDK adapter contains translation logic only; it does not access repositories directly or implement authorization, capability selection, lifecycle, loading, or execution.
- Manifest identity/descriptive fields remain separate from platform/Marketplace-owned capability and permission metadata.
- Permission IDs are metadata only; authorization remains outside this module.
- Core owns capability matching and candidate selection.
- SQLite stores structured metadata; Plugin package files belong in local `Plugin_Storage`.
- The application service treats Registry source references and package/verification formats as opaque adapter contracts; it does not invent unresolved wire formats or trust policy.

## Not implemented yet

- Production Official Registry descriptor resolver and authenticated Registry API/endpoint contract.
- Production Marketplace signing and remote trust-configuration service.
- Actual distribution bootstrap public-key value and release/build provisioning.
- Production composition-root wiring for the Registry resolver, trust source, bootstrap keys, artifact verifier, and final-path adapter.
- Startup recovery coordinator and idempotent reconciliation of incomplete installation operations.
- Uninstall and destructive cleanup.
- Plugin Loader, lifecycle transitions, Runtime, and execution.
- Permission authorization.

PTS-0001 defines the accepted local storage and persistence contract. PTS-0004 and DCR-0001 finalize the path, artifact descriptor, ZIP, signature/trust, and Manifest JSON contracts. The application service coordinates injected ports only. Verification, bounded artifact acquisition, and bounded trust transport primitives are available, but production installation remains disabled until the Official Registry resolver, live trust service, actual distribution bootstrap key, production composition, filesystem promotion, and startup recovery are implemented.

The detailed deployment boundary is in `docs/Trust Configuration Transport and Bootstrap-Key Provisioning.md`.

## Checks

Run Module checks from this directory:

```sh
gofmt -w .
go test ./...
go vet ./...
```

Run SDK adapter integration checks from `sdkadapter/`:

```sh
gofmt -w .
go test ./...
go vet ./...
```

The accepted PZIP-0001 policy fixes the ZIP processing limits: 100 MiB compressed archive, 512 MiB total expanded bytes, 128 MiB per file, 10,000 entries, directory depth 32, per-entry compression ratio 100:1, normalized path length 240 bytes, and 16 MiB parser metadata budget.
