# PRIDE Plugin Module

**Status:** Increment 1 — Domain and Registry Foundation.

This Go module establishes the SDK-neutral Plugin domain and the local installed Plugin Registry repository boundary.

## Implemented

- Domain types for Plugin identity, developer-owned Manifest data, Plugin type/state, capability declarations, and protected permission metadata.
- Domain validation for required Manifest fields, identity consistency, valid type/state values, and duplicate/empty capability or permission identifiers.
- A narrow registry repository interface.
- A deterministic in-memory repository for tests and local development, with duplicate-ID rejection and defensive deep copies.

## Boundaries

- The domain model does not depend on generated Protobuf types or the Plugin SDK.
- Manifest identity/descriptive fields remain separate from platform/Marketplace-owned capability and permission metadata.
- Permission IDs are metadata only; authorization remains outside this module.
- Core owns capability matching and candidate selection.
- The in-memory repository is not a production persistence decision.
- Discovery application services, SDK adapter integration, persistent storage, package loading, lifecycle, and runtime are deferred to later increments.

See [Plugin Module Foundation Design](../../01-Architecture/Plugin%20Module%20Foundation%20Design.md) for the staged implementation plan.

Run checks from this directory:

```sh
go test ./...
go vet ./...
```
