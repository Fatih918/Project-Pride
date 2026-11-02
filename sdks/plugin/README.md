# PRIDE Plugin SDK

This module owns the Go bindings and public Go client for the canonical Plugin SDK protocol.

## Source of truth

- Protocol schema: `proto/plugin_sdk.proto`
- Protobuf package: `pride.plugin.v1`
- Go module: `pride-plugin-sdk`
- Generated Go package: `pride-plugin-sdk/proto`
- Public client package: `pride-plugin-sdk/client`

Generated files are derived artifacts. Do not edit generated `*.pb.go` files manually.

## Create a client

The caller supplies an existing gRPC connection and retains ownership of it:

```go
conn, err := grpc.NewClient(target, grpc.WithTransportCredentials(creds))
if err != nil {
    return err
}
defer conn.Close()

pluginClient, err := client.New(conn)
if err != nil {
    return err
}

plugins, err := pluginClient.ListPlugins(ctx, &pluginv1.ListPluginsRequest{
    RequestId: "request-123",
    PageSize:  50,
})
if err != nil {
    return err
}
_ = plugins
```

The client forwards the caller's context and optional gRPC call options. It validates required request identifiers and returns transport errors unchanged. Connection setup, credentials, retries, and connection lifecycle remain the caller's responsibility.

## Generate and verify

The Buf configuration files define schema linting and code-generation inputs/outputs. The Makefile provides short aliases for common commands; the underlying commands can also be run directly:

```sh
buf lint
buf generate
go test ./...
go vet ./...
```

Run these commands from this module directory. Generated files are written under `proto/` because generation uses source-relative paths.

## Current scope

The current protocol increment exposes five read-only discovery RPCs. Lifecycle and execution are intentionally deferred until their contracts are reconciled. The SDK client does not select Plugins, evaluate permissions, orchestrate workflows, or implement Plugin behavior.
