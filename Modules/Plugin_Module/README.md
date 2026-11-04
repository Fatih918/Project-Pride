# PRIDE Plugin Module

**Status:** Increment 2 — Discovery Application Service.

This Go module owns SDK-neutral installed Plugin records and read-only discovery queries over the local Plugin Registry.

## Implemented

### Domain and registry foundation
- Domain types for Plugin identity, developer-owned Manifest data, Plugin type/state, capability declarations, and protected permission metadata.
- Domain validation for required Manifest fields, identity consistency, valid type/state values, and duplicate/empty capability or permission identifiers.
- A narrow registry repository interface with create, get, and list operations.
- A deterministic in-memory repository for tests and local development, with duplicate-ID rejection and defensive copies.

### Discovery application service
- Five read-only queries: list, get, find by exact capability ID, get metadata, and get operational state.
- Type and state filters; unspecified filters do not restrict results.
- Deterministic ordering by Plugin ID.
- Opaque, versioned pagination tokens with a snapshot digest. Malformed or stale tokens are rejected.
- Default page size of 50 and maximum page size of 100; negative sizes are invalid and larger values are capped.
- Empty capability matches return an empty page; discovery does not rank or select candidates.
- Availability is returned from the stored domain field and is not inferred from Plugin state.
- Context cancellation and repository errors are propagated.

## Boundaries

- The domain, registry, and discovery application packages do not depend on Protobuf-generated types or the Plugin SDK.
- The SDK server.DiscoveryPort adapter and gRPC composition are the next integration increment.
- Manifest identity/descriptive fields remain separate from platform/Marketplace-owned capability and permission metadata.
- Permission IDs are metadata only; authorization remains outside this module.
- Core owns capability matching and candidate selection.
- The in-memory repository is not a production persistence decision.
- Persistent storage, package installation, loader, lifecycle, and runtime remain deferred.

See [ADR-011 — Plugin Module Foundation Design](../../01-Architecture/ADR/ADR-011%20%E2%80%94%20Plugin%20Module%20Foundation%20Design.md) for the staged implementation plan.

Run checks from this directory:

```sh
gofmt -w .
go test ./...
go vet ./...
```
