# PRIDE Plugin Module

**Status:** Increment 9 — bounded ZIP inspection and confined ZIP extraction implemented.

The Plugin Module owns SDK-neutral installed Plugin records, local persistence, storage layout, and read-only discovery. The SDK integration adapter is isolated in the nested `sdkadapter` Go module.

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
- Root-confined filesystem adapter for operation workspaces and exclusive `artifact.zip` creation/read access.
- Canonical `Plugin_Storage`-relative path generation and safe resolution to absolute OS paths; persisted paths never depend on the process working directory.
- SQLite-backed implementation of the existing `registry.Repository` contract using `modernc.org/sqlite`.
- Versioned schema initialization, deterministic listing, context-aware operations, JSON persistence, and repository error mapping.
- Storage and SQLite repository tests.

### Installation application service
- Package inspection returns developer-owned Manifest data separately from platform-owned Plugin type, state, capabilities, and permission metadata.
- Bounded ZIP inspector performs classic-ZIP preflight, ZIP64 rejection, path/type/entry validation, duplicate-key-safe Manifest decoding, and PZIP-0001 resource checks.
- Secure ZIP extractor revalidates the staged archive, streams only `package/` payload files, enforces actual per-file and cumulative byte limits, and removes partial output on failure.
- Storage adapter creates extraction directories and files exclusively through its `os.Root` confinement; output files use private permissions and never overwrite existing files.
- Inspection and extraction remain separate components: inspection validates the complete package; the extractor then streams payload files into the same operation staging workspace.
- A dedicated injected metadata resolver supplies platform metadata; `plugin.json` cannot author these fields.
- SDK-neutral orchestration of operation-scoped staging creation, artifact acquisition, package inspection, payload extraction, artifact verification, filesystem promotion, and atomic installation persistence through injected ports.
- Explicit operation-storage port; acquisition and promotion receive canonical relative paths and resolve them only through the storage adapter.
- Install/update preconditions and Plugin identity/version validation.
- Durable operation phase recording and recoverable handling when the post-promotion database commit fails.
- Unit tests for successful installation, pre-promotion failure, update preconditions, and recoverable commit failure.

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

- Registry artifact-source resolution and package download.
- ZIP installation policy v1.0.0, bounded ZIP inspection, and confined ZIP extraction are implemented; concrete artifact verification and atomic filesystem promotion adapters remain unimplemented.
- Production Marketplace signing and remote trust-configuration service.
- Plugin Manifest JSON Schema and Registry artifact descriptor schemas are specified, but production parsers/verifiers are not yet implemented.
- Filesystem promotion adapter and startup recovery coordinator.
- Uninstall and destructive cleanup.
- Plugin Loader, lifecycle transitions, Runtime, and execution.
- Permission authorization.

PTS-0001 defines the accepted local storage and persistence contract. PTS-0004 and DCR-0001 finalize the path, artifact descriptor, ZIP, signature/trust, and Manifest JSON contracts. The application service coordinates injected ports only; production network, extraction, signing, and trust adapters remain separate work.

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


The accepted PZIP-0001 policy fixes the ZIP processing limits: 100 MiB compressed archive, 512 MiB total expanded bytes, 128 MiB per file, 10,000 entries, directory depth 32, per-entry compression ratio 100:1, normalized path length 240 bytes, and 16 MiB parser metadata budget. The policy does not itself implement or enable extraction.
