# PRIDE Plugin SDK

This module owns the Go bindings and, in subsequent increments, the public Go client and server-side transport adapter for the canonical Plugin SDK protocol.

## Source of truth

- Protocol schema: `proto/plugin_sdk.proto`
- Protobuf package: `pride.plugin.v1`
- Go module: `pride-plugin-sdk`
- Go package: `pluginv1` in `pride-plugin-sdk/proto`

Generated files are derived artifacts. Do not edit generated `*.pb.go` files manually.

## Prerequisites

- Go version compatible with the `go.mod` directive.
- Buf CLI.
- Network access to Buf remote plugins for the first generation or when the remote plugin cache is empty.

The generator versions are pinned in `buf.gen.yaml`:
- `protoc-gen-go`: v1.36.12
- `protoc-gen-go-grpc`: v1.6.2

## Generate bindings

From this directory:

```sh
buf lint
buf generate
go test ./...
go vet ./...
```

Or use the Makefile:

```sh
make proto-lint
make generate
make verify
```

Generated files are written beside the source schema under `proto/` because the generator uses `paths=source_relative`.

## Compatibility rules

1. Edit `proto/plugin_sdk.proto`, not generated Go files.
2. Never reuse published Protobuf field numbers.
3. Keep the `pride.plugin.v1` package stable for compatible additions.
4. Introduce a new package version for breaking wire-contract changes.
5. Regenerate and commit generated bindings whenever the schema changes.
6. Keep Module domain types independent of generated Protobuf types.

## Current scope

The current protocol increment is read-only discovery and defines five RPCs. Lifecycle and execution are intentionally deferred until their contracts are reconciled. This module does not contain a concrete Plugin Module implementation.
