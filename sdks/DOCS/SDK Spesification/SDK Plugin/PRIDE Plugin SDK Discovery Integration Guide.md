# PRIDE Plugin SDK — Discovery Integration Guide

**Status:** Implementation Guide
**Version:** 1.0.0
**Scope:** Initial read-only discovery integration
**Protocol:** `pride.plugin.v1`
**Transport:** gRPC

---

## 1. Purpose and scope

This guide describes how a platform consumer and a Plugin Module integration point use the existing PRIDE Plugin SDK discovery boundary.

The current SDK exposes five read-only operations:

- `ListPlugins`
- `GetPlugin`
- `FindPluginsByCapability`
- `GetPluginMetadata`
- `GetPluginState`

It covers connection ownership, client construction, server adapter registration, the Module-facing port, request and response handling, errors, pagination, and verification.

This guide does not define a concrete Plugin Module. The repository currently provides the SDK-neutral `server.DiscoveryPort` interface and a gRPC adapter, but no production implementation of that port. Any Module implementation must be designed and owned in the Plugin Module project.

Lifecycle and execution RPCs are out of scope. Do not infer that the current SDK supports install, update, enable, disable, reload, start, stop, uninstall, or Plugin execution.

## 2. Integration topology

The logical request path is:

```text
Platform Component
       |
       v
Plugin SDK public client
       |
       | pride.plugin.v1 / gRPC
       v
Plugin SDK server adapter
       |
       | server.DiscoveryPort
       v
Plugin Module application boundary
       |
       v
Module-owned registry and state services
```

Ownership:

| Component | Responsibility |
|---|---|
| Platform consumer | Chooses when to query; owns its gRPC connection and call context |
| Public SDK client | Validates required request fields and forwards calls |
| SDK server adapter | Validates protocol inputs, maps DTOs and Protobuf, translates errors |
| Plugin Module | Owns authoritative Plugin records, registry query semantics, and Plugin state |
| Capability Registry | Owns capability matching/selection; discovery RPCs only return declared information |
| Core / platform security | Owns cross-domain orchestration and permission decisions |

The adapter must not reach into repositories, storage drivers, loaders, or runtimes. It calls only the Module-facing discovery port. The Module's domain layer must not depend on generated Protobuf types.

The adapter may be hosted in the Plugin Module process, but this guide does not mandate a process or deployment topology.

## 3. Public client integration

The SDK client wraps a caller-provided `grpc.ClientConnInterface`. The caller owns connection creation, credentials, lifecycle, and closure.

Example:

```go
package consumer

import (
    "context"

    pluginclient "pride-plugin-sdk/client"
    pluginv1 "pride-plugin-sdk/proto"

    "google.golang.org/grpc"
)

func listPlugins(ctx context.Context, conn grpc.ClientConnInterface) (*pluginv1.ListPluginsResponse, error) {
    sdk, err := pluginclient.New(conn)
    if err != nil {
        return nil, err
    }

    return sdk.ListPlugins(ctx, &pluginv1.ListPluginsRequest{
        RequestId: "request-unique-to-this-operation",
        PageSize:  50,
    })
}
```

The example assumes the caller has already established a configured gRPC connection. It intentionally does not prescribe TLS credentials, target discovery, retry policy, interceptors, or connection pooling; those are connection-owner decisions.

### Client behavior

The public client:

- Rejects a nil request.
- Requires a non-empty `request_id` for every operation.
- Requires `plugin_id` for ID-based operations.
- Requires `capability_id` for capability search.
- Forwards the original context and supplied gRPC call options.
- Returns transport errors unchanged.
- Returns `client.ErrEmptyResponse` if the transport returns neither a response nor an error.

The client does not retry, select Plugins, evaluate permissions, create connections, or close caller-owned connections.

## 4. Server adapter integration

The SDK server adapter is constructed with a Module-owned implementation of `server.DiscoveryPort`.

```go
package plugintransport

import (
    pluginv1 "pride-plugin-sdk/proto"
    pluginserver "pride-plugin-sdk/server"

    "google.golang.org/grpc"
)

func Register(serverGRPC *grpc.Server, discovery pluginserver.DiscoveryPort) error {
    adapter, err := pluginserver.New(discovery)
    if err != nil {
        return err
    }

    pluginv1.RegisterPluginServiceServer(serverGRPC, adapter)
    return nil
}
```

The caller creates and configures `grpc.Server`, establishes its listener and transport security, and decides how to serve and shut it down. The snippet shows registration only; it is not a complete server bootstrap.

The `DiscoveryPort` implementation is the integration seam. It must translate the five operations into the Plugin Module's own application/domain services. It must not be implemented by querying storage directly from the SDK adapter.

## 5. Module-facing discovery operations

The current SDK-neutral interface is:

```go
type DiscoveryPort interface {
    ListPlugins(context.Context, ListQuery) (PluginPage, error)
    GetPlugin(context.Context, PluginQuery) (Plugin, error)
    FindPluginsByCapability(context.Context, CapabilityQuery) (PluginPage, error)
    GetPluginMetadata(context.Context, PluginQuery) (PluginMetadata, error)
    GetPluginState(context.Context, PluginQuery) (PluginStateInfo, error)
}
```

The interface and DTOs are defined in `sdks/plugin/server/port.go`.

Implementation guidance:

1. Keep Module-owned business rules inside Module application/domain services.
2. Translate Module entities into the SDK-neutral DTOs at the Module integration boundary.
3. Keep Manifest fields separate from capability and permission metadata.
4. Return permission identifiers only as metadata; do not interpret them as an authorization grant.
5. Return Plugin type, state, compatibility, and availability from authoritative Module/platform facts. Do not guess these values in the adapter.
6. Return stable Plugin IDs, never paths, runtime process IDs, or memory addresses.
7. Honor the provided `context.Context`, including cancellation and deadlines where underlying operations support them.
8. Return typed `server.DomainError` values for expected domain failures so the adapter can map them to stable gRPC statuses.

The DTOs are transport-neutral Go types, not a requirement that the Module adopt the SDK's package or internal model.

## 6. Discovery operation mapping

| RPC | Request requirements | Module port operation | Result |
|---|---|---|---|
| `ListPlugins` | `request_id`; optional page size/token, type/state filters, metadata | `ListPlugins` | Plugin page and next-page token |
| `GetPlugin` | `request_id`, `plugin_id` | `GetPlugin` | One Plugin record |
| `FindPluginsByCapability` | `request_id`, `capability_id`; optional page size/token, metadata | `FindPluginsByCapability` | Matching Plugin page |
| `GetPluginMetadata` | `request_id`, `plugin_id` | `GetPluginMetadata` | Standardized descriptive metadata |
| `GetPluginState` | `request_id`, `plugin_id` | `GetPluginState` | State and availability information |

Discovery is read-only. In particular, `FindPluginsByCapability` returns matching declarations; it does not rank candidates, select a Plugin, authorize an operation, create an execution plan, or execute a Plugin.

## 7. Request identity, metadata, and context

Every request must carry a non-empty `request_id`. The SDK adapter preserves it in successful responses and structured `PluginError` details.

Optional request metadata is contextual information. It must not be used to bypass SDK contracts or to smuggle credentials, tokens, passwords, or other secrets.

The incoming gRPC context is passed to the Module-facing port. Do not replace it with `context.Background()` or detach work from the caller's cancellation/deadline.

## 8. Pagination and filters

`ListPlugins` and `FindPluginsByCapability` accept page size and page token.

- A page size below zero is rejected.
- A page size of zero requests the server-side default.
- The Module owns the maximum page size and actual pagination semantics.
- Page tokens are opaque to SDK consumers.
- Consumers should pass the returned next-page token unchanged to request the next page.
- The Module must preserve the applicable filters between pages.

Do not expose database cursors, storage offsets, or internal registry query details through page tokens.

## 9. Errors

The server adapter translates `server.DomainError` and context failures into gRPC status errors with a structured `pluginv1.PluginError` detail when available.

The current mapping includes:

| Domain error | gRPC status |
|---|---|
| `INVALID_REQUEST`, `INVALID_INPUT` | `InvalidArgument` |
| `PLUGIN_NOT_FOUND`, `CAPABILITY_NOT_FOUND` | `NotFound` |
| `PLUGIN_UNAVAILABLE`, `PLUGIN_NOT_READY` | `Unavailable` |
| `PLUGIN_DISABLED`, `INCOMPATIBLE_PLUGIN` | `FailedPrecondition` |
| `PERMISSION_DENIED` | `PermissionDenied` |
| `TIMEOUT` | `DeadlineExceeded` |
| `INTERNAL_ERROR`, `LIFECYCLE_ERROR`, `EXECUTION_ERROR`, `UNKNOWN_ERROR` | `Internal` |

Unknown domain error codes also map to an internal error. Context cancellation maps to `Canceled`; context deadline expiration maps to `DeadlineExceeded`.

Consumers should branch on gRPC status and structured `error_code`, not on human-readable message text. Error messages must not disclose credentials, secrets, private file paths, stack traces, or runtime internals.

Permission-related error mapping does not mean the SDK performs permission evaluation. Authorization remains the responsibility of the platform security boundary and Plugin Module enforcement.

## 10. Security and operational responsibilities

The SDK provides request validation and protocol mapping, not a complete security or operational deployment policy.

The integration owner remains responsible for:

- Configuring authenticated and appropriately encrypted transport.
- Restricting access to the gRPC service.
- Applying authorization before sensitive discovery data is returned.
- Protecting permission identifiers and metadata according to platform policy.
- Configuring server limits, timeouts, observability, and graceful shutdown.
- Avoiding secrets in request metadata, response metadata, attributes, and error messages.
- Ensuring Module operations honor request cancellation and deadlines.

No particular authentication mechanism, authorization middleware, retry strategy, or deployment topology is mandated by the current SDK protocol.

## 11. Testing an integration

The SDK repository tests the public client and server adapter independently with deterministic fakes and gRPC `bufconn`. A future Plugin Module implementation should add its own tests for:

- Mapping Module-owned records to SDK-neutral DTOs.
- Stable Plugin identity and Manifest mapping.
- Type/state and availability values.
- Capability declarations and pagination/filter behavior.
- Domain error translation at the Module boundary.
- Context cancellation and deadline handling.
- Ensuring discovery does not trigger lifecycle or execution side effects.

A passing SDK test suite proves the SDK boundary behavior; it does not prove that a production Plugin Module exists or behaves correctly.

Run the SDK checks from `sdks/plugin`:

```sh
buf lint
buf generate
go test ./...
go vet ./...
```

Review generated-file diffs after generation. Do not edit generated `*.pb.go` files manually.

## 12. Related sources

- `sdks/plugin/proto/plugin_sdk.proto` — canonical wire schema.
- `sdks/plugin/client` — public Go client.
- `sdks/plugin/server` — gRPC adapter, discovery port, DTOs, and error mapping.
- `sdks/DOCS/SDK Spesification/SDK Plugin/PRIDE Plugin SDK Protocol Specification.md`.
- `sdks/DOCS/SDK Spesification/SDK Plugin/PRIDE Plugin SDK Module Boundary Design.md`.
- PRIDE Plugin SDK Specification and PRIDE Plugin Manifest Specification.

