# PRIDE Plugin SDK Protocol Specification

**Version:** 1.0.0
**Status:** Draft
**Category:** Protocol Specification
**Document ID:** PRIDE-PLUGIN-SDK-PROTOCOL
**Protocol:** Protocol Buffers
**Transport:** gRPC
**Initial Scope:** Read-only Plugin discovery

---

# 1. Purpose

This document defines the initial wire protocol used by PRIDE Plugin SDK clients to communicate with the Plugin SDK server boundary.

The protocol is derived from the PRIDE Plugin SDK Specification, Plugin SDK Contract, Plugin Module Specification, Plugin Manifest Specification, ADR-001, ADR-002, and ADR-010.

The canonical machine-readable protocol is the plugin_sdk.proto schema. Generated language bindings are derived artifacts and shall not redefine the protocol.

---

# 2. Initial Scope

The first implementation increment establishes read-only discovery operations:

* List installed Plugins.
* Retrieve one Plugin by stable identifier.
* Find Plugins by declared capability.
* Retrieve standardized Plugin metadata.
* Retrieve Plugin state information.

These operations allow platform consumers, including the Capability Registry and Official Interface, to discover Plugins through the official SDK boundary.

Lifecycle and execution are intentionally not included in this first protocol increment. They will be introduced through compatible protocol additions after their operation semantics are reconciled against the stable SDK Specification, Plugin Module Specification, and the draft SDK Contract.

This is a phased implementation decision, not a reduction of the overall Plugin SDK responsibilities.

---

# 3. Protocol Architecture

    Platform Component
            |
            v
       Plugin SDK Client
            | gRPC / Protocol Buffers
            v
       Plugin SDK Server
            |
            v
     Plugin Module Boundary
            |
            v
      Plugin Registry / Module

The SDK client is a thin, language-native interface over the protocol.

The SDK server validates and forwards standardized operations through a Module-facing contract. It does not own Plugin registry state, Plugin lifecycle, package installation, permission evaluation, or Plugin execution.

The Plugin Module implementation is not yet present in the repository. The initial SDK increment therefore defines and tests the communication boundary independently, using a replaceable Module client contract and deterministic test doubles. It must not invent a production Plugin Module implementation inside the SDK.

---

# 4. Protocol Principles

1. Protocol Buffers is the canonical message schema.
2. gRPC is the initial transport.
3. The protocol is independent from Plugin implementation language and runtime.
4. The protocol does not execute Plugin functionality.
5. The protocol does not perform capability selection.
6. The protocol does not evaluate permissions.
7. The protocol does not own Plugin lifecycle or registry state.
8. The protocol exposes standardized Plugin information only.
9. Request context, cancellation, and deadlines are propagated through gRPC.
10. SDK errors are represented using stable contract-level codes.
11. Generated code is derived from the canonical .proto source.
12. New operations are added compatibly without reusing field numbers.

---

# 5. Service Definition

The initial service is PluginService in package pride.plugin.v1.

| RPC | Responsibility | Mutates Plugin state |
|---|---|---:|
| ListPlugins | List installed Plugin records with pagination and optional type/state filters | No |
| GetPlugin | Retrieve one Plugin record by stable Plugin ID | No |
| FindPluginsByCapability | Find records declaring a capability | No |
| GetPluginMetadata | Retrieve standardized descriptive and registry metadata | No |
| GetPluginState | Retrieve the current state and availability | No |

Discovery methods are read-only. They must not load or execute a Plugin as a side effect. Every request carries a required request_id and optional standardized metadata. Every successful response returns the originating request_id and optional response metadata. The server must preserve request identity end-to-end.

---

# 6. Plugin Identity and Manifest

The stable Plugin identifier is the lookup key. It must not be derived from a runtime process ID, memory address, installation path, or framework-specific object.

The protocol PluginManifest message mirrors the fields defined by the PRIDE Plugin Manifest Specification.

Required fields:

* plugin_id
* name
* version
* description
* author
* license
* entry_point
* minimum_pride_version

Optional fields:

* homepage
* documentation_url
* repository_url
* issue_tracker_url
* keywords
* category
* icon
* changelog_url

Capability declarations and permission analysis are not part of the Plugin Manifest. They are represented separately as registry/platform metadata, consistent with the Manifest Specification and Marketplace architecture.

The SDK transports this information; it does not validate repository authenticity or generate protected Marketplace metadata.

---

# 7. Plugin Type and State

The protocol defines two Plugin types:

* ACTIVE — provides executable functionality.
* PASSIVE — provides platform or user-interface extensions and is not executed by the Executor.

The initial state values mirror the Plugin SDK Specification:

* INSTALLED
* ENABLED
* DISABLED
* RUNNING
* ERROR

The SDK only reports state. State transitions remain owned by the Plugin Module.

---

# 8. Capability and Metadata Boundary

Capability declarations are exposed as standardized metadata so the Capability Registry can discover available capabilities without inspecting Plugin implementations.

The SDK does not select a Plugin for a task. The Capability Registry owns matching and selection.

Permission identifiers may be returned as protected registry metadata for inspection where the caller is authorized. The SDK does not grant, revoke, or evaluate permissions. Authorization remains a Core/platform security responsibility and the Plugin Module shall enforce the applicable execution boundary.

The attributes field is reserved for contract-defined, non-secret metadata. It must not carry credentials, tokens, passwords, private keys, internal runtime objects, or arbitrary implementation state.

---

# 9. Pagination and Filtering

List and capability-search operations support page size and page token.

The server shall:

* Apply a documented maximum page size.
* Return an opaque next-page token when more results are available.
* Treat an empty page token as the first page.
* Reject malformed or invalid tokens with a standardized error.
* Preserve filter semantics across pages.

A page token is opaque to SDK consumers and must not expose storage cursors or internal registry implementation details.

---

# 10. Request Validation

The server and client boundary shall reject invalid inputs before forwarding them to the Plugin Module.

At minimum:

* plugin_id must be non-empty for ID-based operations.
* capability_id must be non-empty for capability search.
* page_size must be non-negative; zero requests the server default.
* Page tokens are opaque and must not be interpreted by clients.

A valid request must include a non-empty request_id, preserve that identifier in the response or structured error, and preserve the caller's context, including cancellation and deadline. Optional request metadata is contextual data only; it must not be used to transfer secrets or bypass domain contracts.

---

# 11. Error Protocol

A failed operation shall use a non-OK gRPC status with a structured PluginError detail.

The PluginError contains request_id, error_code, message, and metadata. The request_id must match the originating request whenever it is available.

Initial error categories are derived from the Plugin SDK Contract:

* PLUGIN_NOT_FOUND
* PLUGIN_UNAVAILABLE
* PLUGIN_DISABLED
* PLUGIN_NOT_READY
* CAPABILITY_NOT_FOUND
* INVALID_REQUEST
* INVALID_INPUT
* PERMISSION_DENIED
* INCOMPATIBLE_PLUGIN
* LIFECYCLE_ERROR
* EXECUTION_ERROR
* TIMEOUT
* INTERNAL_ERROR
* UNKNOWN_ERROR

The initial discovery increment is expected to use a subset of these codes. Future lifecycle and execution operations may use the remaining categories.

Public error messages must not disclose file paths, credentials, secrets, stack traces, or private runtime details. Consumers should branch on error_code, not message text.

---

# 12. Transport and Cancellation

gRPC is the initial transport. The SDK client accepts caller-provided context.Context and must pass it unchanged to the generated gRPC client.

The SDK must not silently replace a caller's deadline or ignore cancellation.

Connection construction and ownership remain explicit. The SDK client shall not create hidden global connections. Connection lifecycle policy will be defined in the Go implementation contract.

---

# 13. Versioning and Compatibility

The initial package is pride.plugin.v1.

Protocol compatibility rules:

1. Existing field numbers and meanings are stable once published.
2. Removed field numbers must be reserved.
3. Field numbers must never be reused.
4. Compatible fields and RPCs may be added to v1.
5. Breaking changes require a new major package, such as pride.plugin.v2.
6. Generated bindings are regenerated from the canonical schema.

---

# 14. Explicit Non-Goals

This protocol increment does not define or implement:

* Plugin installation, update, reload, enable, disable, start, stop, or uninstall.
* Plugin package download or distribution.
* Plugin manifest repository validation.
* Marketplace publication or Registry generation.
* Permission evaluation or authorization decisions.
* Plugin runtime execution.
* Workflow planning or capability selection.
* Memory or AI SDK integration.
* A concrete Plugin Module implementation.
* A shared SDK runtime across Plugin, Memory, and AI.

These responsibilities remain with the components and specifications that own them.

---

# 15. Known Contract Alignment Items

The current repository contains a stable Plugin SDK Specification and a draft Plugin SDK Contract with different lifecycle operation lists:

* Stable SDK Specification: Install, Update, Enable, Disable, Reload, Uninstall.
* Draft SDK Contract: Install, Enable, Disable, Start, Stop, Uninstall.

The initial read-only protocol avoids locking either lifecycle list into the wire contract. Before lifecycle RPCs are added, the contract documents must be reconciled and reviewed.

The Plugin Manifest Specification also explicitly excludes capability and permission declarations from the Manifest. This protocol therefore keeps Plugin Manifest fields separate from capability and permission metadata.

---

# 16. Conformance

An implementation conforms to this initial protocol increment when it:

1. Implements the canonical pride.plugin.v1.PluginService.
2. Supports all five discovery RPCs.
3. Preserves stable Plugin identity.
4. Returns Manifest fields according to the Manifest Specification.
5. Keeps capability and permission metadata separate from the Manifest.
6. Reports state without mutating it.
7. Supports pagination and documented filters.
8. Propagates context, cancellation, and deadlines.
9. Returns standardized errors without leaking sensitive details.
10. Does not execute Plugins or perform lifecycle transitions as a side effect.
11. Remains independent from AI SDK and Memory SDK.

---

# 17. Source of Truth

    ADR-001 / ADR-002 / ADR-010
                  |
                  v
    Plugin SDK Specification + Plugin SDK Contract
                  |
                  v
    Plugin SDK Protocol Specification
                  |
                  v
             plugin_sdk.proto
                  |
                  v
       Generated Protocol Bindings
                  |
                  v
          SDK Client / SDK Server
                  |
                  v
           Plugin Module Boundary

The .proto schema is the canonical machine-readable representation of this protocol increment. Generated code and implementation code must conform to it.

---

# 18. Decision Summary

The first Plugin SDK implementation increment establishes a language-neutral, read-only discovery protocol over gRPC.

It provides a stable starting point for Plugin listing, lookup, capability discovery, metadata access, and state inspection while deferring lifecycle and execution until their contracts are consistent.

This phased approach allows PRIDE to establish and validate the SDK communication boundary without prematurely implementing Plugin Module responsibilities or locking unresolved lifecycle semantics into a public protocol.
