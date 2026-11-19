# PRIDE Plugin Module

**Status:** Increment 12 — bounded ZIP inspection/extraction, signed artifact verification, persistent trust-sequence storage, and bounded HTTP artifact acquisition implemented.

The Plugin Module owns SDK-neutral installed Plugin records, local persistence, storage layout, and read-only discovery. The SDK integration adapter is isolated in the nested `sdkadapter` Go module.

## Increment 12 — Registry artifact acquisition

- Added an SDK-neutral `DescriptorResolver` port. It resolves an opaque immutable `SourceReference` into the exact signed Marketplace descriptor; the application service does not parse Registry documents.
- Added `installation/acquisition.HTTPArtifactAcquirer` to validate descriptor routing fields and download the exact declared ZIP byte length.
- Artifact downloads require HTTPS and an exact configured hostname allowlist. IP literals, non-443 ports, user-info, fragments, redirects, proxy use, and non-public DNS resolutions are rejected.
- HTTP uses TLS 1.2 or newer, bounded timeouts and response headers, and context-aware streaming.
- Compressed artifact size is bounded by PZIP-0001's 100 MiB maximum. Declared size and actual bytes must match exactly.
- Partial artifact files are removed after stream, size, sync, or cancellation failures.
- The original descriptor bytes are preserved in `StagedArtifact.DescriptorJSON`; acquisition does not claim to authenticate it. The existing artifact verifier remains responsible for Ed25519/JCS signature and SHA-256 verification.
- Added tests for successful acquisition, duplicate JSON keys, unapproved hosts, oversized artifacts, size mismatches, redirects, resolver errors, cancellation, and unsafe configuration.
- The `DescriptorResolver` remains an injected port: a production Official Registry network client is not included because the repository has not yet approved a Registry API/endpoint contract.

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
- SQLite installation journal and installed-version persistence, plus monotonic trust-sequence storage.

### Installation application service
- Package inspection returns developer-owned Manifest data separately from platform-owned Plugin type, state, capabilities, and permission metadata.
- Bounded ZIP inspector enforces PZIP-0001 and validates Manifest/package structure.
- Secure ZIP extractor streams only `package/` payload files through confined storage.
- Artifact verifier validates descriptor schema, Ed25519/JCS signature, SHA-256 digest, and trust configuration.
- Installation orchestration verifies before extraction and again afterward, then delegates promotion and atomic persistence through injected ports.
- Operation paths remain canonical and relative to `Plugin_Storage`; SDK and domain packages do not depend on generated Protobuf types.

## Not implemented yet

- Production Official Registry descriptor resolver and its authenticated API contract.
- Production Marketplace signing and remote trust-configuration service.
- Distribution bootstrap-key provisioning and production composition-root wiring.
- Filesystem atomic promotion adapter and startup recovery coordinator.
- Public SDK lifecycle RPCs, Plugin Loader, Runtime, execution, authorization, uninstall, and destructive cleanup.

Production installation remains disabled. The acquisition adapter is a bounded transport component, not proof that a complete production installation pipeline is ready. A production resolver, trust/bootstrap wiring, promotion, and startup recovery are still required.

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

PZIP-0001 limits: 100 MiB compressed archive, 512 MiB expanded content, 128 MiB per file, 10,000 ZIP entries, depth 32, 100:1 per-entry compression ratio, 240-byte normalized UTF-8 paths, and 16 MiB parser metadata.
