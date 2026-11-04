# Plugin Module SDK Adapter

This module is the explicit integration boundary between the SDK-neutral Plugin Module discovery application service and the PRIDE Plugin SDK server port.

## Responsibilities

- Implements `pride-plugin-sdk/server.DiscoveryPort`.
- Delegates all five read-only discovery operations to `pride-plugin-module/discovery`.
- Maps Module-owned domain records and enums to SDK-neutral server DTOs.
- Converts Module discovery errors into stable `server.DomainError` values.
- Preserves context cancellation and deadlines.
- Contains no registry, storage, authorization, selection, lifecycle, or runtime business logic.

The Module's `domain`, `registry`, and `discovery` packages remain independent of SDK and Protobuf packages. This adapter is isolated in its own Go module so the integration dependency does not leak into those packages.

## Integration test

The integration test creates a deterministic in-memory Plugin Registry, composes the Module discovery service with the SDK server adapter, and exercises all five RPCs over gRPC `bufconn`.

Run from this directory:

```sh
gofmt -w .
go test ./...
go vet ./...
```

The test verifies the integration boundary only. It does not claim to validate persistent storage, Plugin installation, lifecycle, or runtime behavior.
