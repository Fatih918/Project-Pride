# PRIDE Plugin SDK — Plugin Module Boundary Design

**Status:** Proposed
**Version:** 1.0.0
**Scope:** Initial read-only discovery integration
**Depends on:** ADR-002, ADR-010, PRIDE Plugin SDK Specification, PRIDE Plugin Module Specification, Plugin SDK Protocol Specification

---

## 1. Purpose

This document defines the implementation boundary between the PRIDE Plugin SDK protocol and the Plugin Module domain.

The goal is to let the Plugin SDK communicate with Plugin Module capabilities without depending on the Module's registry storage, runtime, loader, storage technology, or internal domain model.

This design covers only the five read-only discovery RPCs defined by `pride.plugin.v1.PluginService`.

It does not introduce lifecycle or execution behavior.

## 2. Architectural Decision

The Protobuf service defined in `sdks/plugin/proto/plugin_sdk.proto` is the authoritative transport contract for this integration increment.

The Plugin SDK client is the consumer-facing communication API. The Plugin Module provides the domain operations and authoritative Plugin data. A transport adapter translates between the Protobuf contract and Module-owned discovery operations.

The transport adapter must remain a thin translation layer. It must not become a second Plugin Registry or implement Plugin domain rules.

### Logical topology

```text
Platform Component
       |
       | public SDK API
       v
Plugin SDK Client
       |
       | pride.plugin.v1 gRPC contract
       v
Plugin SDK Transport Adapter
       |
       | Module discovery port
       v
Plugin Module
       |
       +-- Plugin Registry
       +-- Installed Plugin information
       +-- Capability registrations
       +-- Plugin state
```

The transport adapter may be hosted in the Plugin Module process, but its responsibility remains protocol translation. Deployment and process topology are not fixed by this document.

## 3. Ownership Boundaries

| Concern | Owner | Boundary rule |
|---|---|---|
| Public SDK API and Protobuf contract | Plugin SDK | Defines stable request, response, and error representations |
| gRPC transport and protocol mapping | SDK transport adapter | Converts protocol messages to/from Module discovery operations |
| Installed Plugin records | Plugin Module | Owns authoritative Plugin information |
| Plugin Registry queries | Plugin Module | Owns registry semantics and query behavior |
| Capability registration data | Plugin Module | Supplies declared capability information |
| Capability selection and execution planning | Capability Registry | Not performed by discovery RPCs |
| Plugin state and state transitions | Plugin Module | SDK exposes state; it does not own or mutate it |
| Permission evaluation | Permission Manager / platform security domain | SDK may transport permission identifiers as metadata, but does not authorize |
| Cross-domain workflow orchestration | Core | No orchestration is introduced by this boundary |

The adapter must not access Plugin storage, runtime, loader, or registry implementation directly. It may call only the Module's declared discovery-facing operations.

## 4. Module Discovery Port

The Plugin Module must expose a narrow read-only application boundary to the transport adapter.

The conceptual operation set is:

| Module operation | Protocol RPC | Required behavior |
|---|---|---|
| List registered Plugins | `ListPlugins` | Return records matching the supported type/state filters and pagination request |
| Get Plugin by stable ID | `GetPlugin` | Return one registered Plugin record or a not-found result |
| Find registered Plugins by capability ID | `FindPluginsByCapability` | Return matching records; do not rank, select, authorize, or execute them |
| Read Plugin metadata | `GetPluginMetadata` | Return standardized metadata for one Plugin |
| Read Plugin state | `GetPluginState` | Return the current state and availability information |

These are application-level operations, not a mandate for a specific Go interface or package layout. The Module implementation may use its own domain types and internal architecture.

The adapter must not pass Protobuf-generated types into the Plugin Registry or other Module domain services. It must translate at the boundary.

## 5. Request Flow

For each RPC, the adapter follows this sequence:

1. Receive the Protobuf request.
2. Preserve the incoming `request_id` and request metadata.
3. Perform protocol-level validation required to safely interpret the request.
4. Translate the request into the corresponding Module discovery query.
5. Invoke the Module discovery port with the request context.
6. Translate the Module result into the Protobuf response.
7. Return the originating `request_id` and permitted response metadata.

The adapter must not silently replace a caller-supplied request ID.

A missing or invalid required request field must fail as a contract-level error rather than being silently interpreted as a valid query.

## 6. Response Mapping

The adapter maps Module-owned values into the protocol representations defined in `plugin_sdk.proto`.

Mapping rules:

- Plugin identity is represented by the stable `plugin_id`, never by a file path, runtime process ID, or memory address.
- Manifest fields map only to the fields defined by the Plugin Manifest Specification.
- Capabilities remain separate from Manifest fields.
- Permission identifiers remain metadata and must not be interpreted as authorization decisions.
- Plugin type and state map to the protocol enums. Unknown or unsupported domain values must not be misrepresented as a valid known value.
- Availability and compatibility information must come from Module/platform-owned facts; the adapter must not infer them from implementation details.
- Metadata attributes must contain contract-permitted descriptive data, not runtime objects, credentials, secrets, or private implementation references.
- Pagination tokens are opaque to SDK consumers and are passed between the adapter and Module without consumers depending on their internal format.

## 7. Error Boundary

The Module may use internal errors and domain-specific failure types. Those types must not leak into the public SDK contract.

The adapter translates failures into the Plugin SDK error taxonomy documented by the Plugin SDK Contract and Protocol Specification.

For this discovery increment, the adapter should preserve the distinction between at least:

- Invalid request.
- Plugin not found.
- Capability not found or no matching result, according to the operation's defined semantics.
- Unavailable Module/domain dependency.
- Internal failure.

The adapter must preserve the originating request ID in structured error details when the transport supports those details.

Human-readable messages must not expose credentials, secrets, private file paths, runtime internals, or other unnecessary implementation details.

The exact gRPC status-code mapping is an implementation detail to be finalized alongside the generated bindings and server implementation. It must not create a second public error taxonomy.

## 8. Context, Cancellation, and Deadlines

The adapter must propagate the incoming request context to the Module discovery operation.

It must not detach work from the caller's context or create an unbounded background operation for a request.

When the caller cancels a request or its deadline expires, the adapter must stop waiting and pass cancellation to the Module boundary. The Module implementation should honor that context where its underlying operations permit it.

## 9. Dependency Direction

The required dependency direction is:

```text
Protobuf transport adapter
          |
          v
Module discovery-facing application boundary
          |
          v
Module-owned domain services
```

The Module's domain services must not depend on Protobuf-generated SDK transport types.

The adapter must not depend on concrete registry repositories, storage drivers, plugin loaders, or runtime implementations.

This keeps the transport contract replaceable and the Module internals independently evolvable.

## 10. Explicit Non-Goals

This boundary increment does not implement or define:

- Plugin installation, update, enable, disable, reload, start, stop, or uninstall.
- Plugin execution or runtime invocation.
- Plugin package download, distribution, or Marketplace behavior.
- Manifest repository validation or package loading.
- Permission evaluation or permission grants.
- Capability ranking, selection, or execution-plan generation.
- Core workflow orchestration.
- AI SDK or Memory SDK integration.
- A concrete Plugin Registry, storage engine, loader, or runtime.
- A mandatory process topology or deployment model.

Lifecycle operations remain deferred until the lifecycle terminology mismatch between the stable Plugin SDK Specification and draft Plugin SDK Contract is resolved.

## 11. Implementation Sequence

The next implementation increments should follow this order:

1. Generate Go Protobuf and gRPC bindings from the canonical `plugin_sdk.proto`.
2. Define the concrete Module discovery port at the integration point selected by the Plugin Module implementation.
3. Implement the thin transport adapter against that port.
4. Implement the public Plugin SDK client.
5. Add deterministic fake Module implementations and transport-level contract tests.
6. Integrate with a concrete Plugin Module when that Module exists.

The repository currently has no concrete `Modules/Plugin_Module` implementation. Therefore, tests in the initial SDK implementation must use a deterministic fake of the discovery port and must not claim to validate a production Plugin Module.

## 12. Conformance Checklist

An implementation conforms to this boundary when:

- [ ] All five discovery RPCs use the canonical `pride.plugin.v1` contract.
- [ ] The adapter contains translation logic only, not registry or runtime business logic.
- [ ] Plugin Module remains the owner of authoritative Plugin records and state.
- [ ] Protobuf types do not leak into Module domain services.
- [ ] Module implementation details do not leak into SDK responses.
- [ ] Request IDs are preserved across successful responses and structured errors.
- [ ] Request context, cancellation, and deadlines are propagated.
- [ ] Capabilities and permission identifiers remain separate from Manifest fields.
- [ ] Permission evaluation and capability selection remain outside this adapter.
- [ ] The discovery boundary is read-only and does not trigger lifecycle or execution operations.
- [ ] Tests can exercise the adapter with a deterministic fake Module boundary.

---

## 13. Related Documents

- ADR-002 — Plugin SDK as the Official Plugin Communication Boundary
- ADR-010 — PRIDE SDK Ecosystem Architecture
- PRIDE Plugin SDK Specification
- PRIDE Plugin Module Specification
- PRIDE Plugin SDK Contract
- PRIDE Plugin SDK Protocol Specification
- PRIDE SDK Code Standard
