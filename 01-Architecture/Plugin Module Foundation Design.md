# PRIDE Plugin Module Foundation Design

**Version:** 1.0.0  
**Status:** Proposed Implementation Design  
**Category:** Module Architecture  
**Document ID:** PRIDE-PLUGIN-MODULE-FOUNDATION

---

## 1. Purpose

This document translates the stable PRIDE Plugin Module Specification and accepted architecture decisions into an implementation plan for the first concrete Plugin Module increment.

It does not replace or silently amend the stable specifications. Where the current SDK implementation is narrower than the stable SDK specification, this document records the current implementation boundary and defers the unresolved contract work.

The first implementation increment is deliberately limited to the Plugin Module's read-only discovery foundation. It does not attempt to implement package installation, lifecycle transitions, Plugin execution, or a concrete runtime.

## 2. Current Repository State

At the time of this design:

- `sdks/plugin` contains the canonical Protobuf contract, generated Go bindings, a public discovery client, and a gRPC server adapter.
- `sdks/plugin/server.DiscoveryPort` defines the SDK-neutral read-only boundary that the Plugin Module must implement.
- The SDK adapter implements five discovery RPCs.
- `Modules/Plugin_Module` does not yet exist.
- `Modules/AI_Module` exists as a separate module and is not a template whose internal implementation should be copied blindly.
- The stable Plugin Module Specification defines Installed Plugins, Plugin Registry, Plugin Runtime, Plugin Loader, and Plugin Storage as logical components.

The first goal is to provide an authoritative in-process discovery service and connect it to the already-existing SDK adapter without introducing storage/runtime assumptions prematurely.

## 3. Architectural Position

The Plugin Module owns installed Plugin records and their operational state. The Plugin SDK owns the public communication contract. Core owns capability matching and workflow orchestration.

```text
Core Capability Registry / Official Interface / Other Consumers
                           |
                           | Plugin SDK client
                           v
                 Plugin SDK gRPC Server
                           |
                           | server.DiscoveryPort
                           v
                 Plugin Module Discovery
                           |
                           v
                    Plugin Registry
                           |
              +------------+------------+
              |                         |
       Installed Plugin Records   Operational State
              |
              v
        Plugin Manifest /
     Protected Registry Metadata
```

The diagram shows logical ownership, not a mandated deployment topology. The SDK and Module may be deployed in the same process or separate processes as long as the SDK-to-Module boundary remains explicit.

## 4. Ownership and Non-Responsibilities

### 4.1 Plugin Module owns

- Installed Plugin records.
- The authoritative local registry view of installed Plugins.
- Plugin manifest data after validation.
- Plugin type and operational state.
- Capability declarations associated with installed Plugins.
- Permission identifier metadata associated with installed Plugins.
- Availability and compatibility status exposed through discovery.
- Later: package loading, storage integration, lifecycle management, and Active Plugin runtime execution.

### 4.2 Plugin SDK owns

- Public gRPC/Protobuf messages and RPC contract.
- Request envelope validation.
- Protocol-to-DTO and DTO-to-protocol mapping.
- Request ID and metadata forwarding.
- gRPC status/error translation.

The SDK adapter must not query Module storage or Registry internals directly.

### 4.3 Core owns

- Capability matching.
- Plugin candidate selection.
- Execution Plan generation.
- Workflow planning and orchestration.

The Plugin Module provides declared capability information. It does not choose the best Plugin for a task and does not produce the Core Execution Plan.

### 4.4 Marketplace owns

- Repository analysis and publication workflow.
- Generation and verification of protected Registry metadata.
- Publication of Registry information to the official Registry Repository through the established Pull Request process.

The Marketplace is not the local installed Plugin database and does not execute installed Plugins.

### 4.5 Permission Manager / platform security owns

- Authorization decisions.
- Permission grants, revocations, and policy evaluation.

The Plugin Module may retain and expose permission identifiers as metadata. It must not interpret those identifiers as authorization decisions or bypass the Permission Manager.

## 5. Registry Concepts Must Remain Distinct

The architecture uses the word Registry in more than one context. Implementations must not collapse these concepts.

### 5.1 Official Plugin Registry Repository

The Registry Repository contains standardized published Plugin Registry information. It is the source used by the Marketplace for discovery and publication. It does not host Plugin source code or binaries.

### 5.2 Local Plugin Module Registry

The local Plugin Registry is the Module-owned operational view of Plugins installed in the current PRIDE environment. It contains the records required for SDK discovery and later lifecycle/runtime operations.

A published Registry entry is not itself an installed Plugin record. Installation obtains and verifies a Plugin package, validates its manifest, and then registers the installed Plugin in the local Module Registry.

## 6. Data Model Boundaries

The first increment should use explicit domain records rather than exposing Protobuf types as the Module's internal model.

The initial domain model needs to represent:

- Plugin identity, based on `plugin_id`.
- Manifest fields defined by the stable Plugin Manifest Specification.
- Plugin type: Active or Passive.
- Declared capabilities.
- Permission identifiers as metadata.
- Operational state using the currently published SDK enum vocabulary.
- Availability.
- Compatibility status.
- Additional standardized attributes where the SDK contract requires them.

Manifest and protected metadata are separate inputs:

- The Manifest is developer-provided identity and descriptive metadata.
- Capability declarations, permission metadata, compatibility analysis, security status, verification status, and other protected fields are platform/Marketplace-generated metadata.
- The Module must not treat capability or permission declarations as fields owned by the developer's Manifest.

The first increment should define the domain shape and validation boundary. It should not invent a new persistent schema or package format.

## 7. Discovery Contract Mapping

The Module discovery application service must implement the existing SDK port:

`sdks/plugin/server.DiscoveryPort`

| SDK operation | Module application behavior | Data returned |
|---|---|---|
| `ListPlugins` | Query installed Plugin records using page, type, and state filters | Plugin records and next-page token |
| `GetPlugin` | Resolve one installed Plugin by ID | Manifest, type, state, capabilities, availability, compatibility |
| `FindPluginsByCapability` | Filter installed Plugin records by declared capability ID | Matching records and next-page token |
| `GetPluginMetadata` | Read standardized metadata for one installed Plugin | Identity, type, capabilities, permission IDs, compatibility, attributes |
| `GetPluginState` | Read operational state for one installed Plugin | State, availability, status message |

All five operations are read-only. They must not load a package, mutate state, start a runtime, evaluate permissions, or execute a Plugin.

### 7.1 Pagination

The SDK contract uses opaque page tokens. The Module owns their semantics.

The first implementation may use a deterministic in-memory pagination strategy for tests and local development. The token must remain opaque to SDK consumers. The implementation must reject malformed or stale tokens using a standardized error rather than silently restarting at the first page.

The default and maximum page sizes must be explicitly chosen and tested by the Module implementation. Do not assume an undocumented default from the SDK adapter.

### 7.2 Filtering and ordering

- Type and state filters are applied to installed Plugin records.
- An unspecified filter means no restriction for that dimension.
- Capability lookup matches the exact standardized capability identifier.
- Results must have deterministic ordering so pagination remains stable for an unchanged registry snapshot.
- The Module must not perform fuzzy capability matching or rank candidates; that belongs to Core.

### 7.3 Metadata forwarding

Request metadata is transport context. It must not be treated as a command to mutate registry data or bypass authorization. Response metadata should contain only documented, non-secret diagnostic or correlation information.

## 8. Proposed Package Boundaries

The initial Go module should live at:

`Modules/Plugin_Module`

The proposed initial package layout is:

```text
Modules/Plugin_Module/
├── go.mod
├── README.md
├── domain/
│   ├── plugin.go
│   ├── manifest.go
│   ├── capability.go
│   └── state.go
├── registry/
│   ├── repository.go
│   ├── service.go
│   └── errors.go
├── discovery/
│   ├── service.go
│   ├── query.go
│   └── pagination.go
├── storage/
│   └── (introduced with a selected storage adapter)
├── runtime/
│   └── (introduced after runtime contract and isolation design)
├── loader/
│   └── (introduced with package and installation verification design)
├── transport/
│   └── grpc/
│       └── (Module-side service adapter, if required by deployment topology)
└── server/
    └── (composition root and process lifecycle)
```

This is a proposed boundary map, not a requirement to create empty directories or placeholder packages in the first code PR.

Rules:

1. `domain` contains SDK-neutral Plugin records and value types.
2. `registry` owns authoritative installed Plugin record operations.
3. `discovery` implements read-only application queries and the SDK `DiscoveryPort` through a narrow integration adapter.
4. Storage technology is hidden behind a repository interface and selected in a separate decision.
5. Runtime and loader packages are deferred until their contracts are designed.
6. Protobuf-generated types must not become the domain model.
7. The Module must not import Core's Capability Registry implementation.
8. The Module may depend on the Plugin SDK's narrow server port as an integration dependency, but domain and registry packages should not depend on SDK transport packages.

## 9. Initial Integration Composition

The SDK already exposes a server adapter constructor that accepts `server.DiscoveryPort`. The first Module integration should compose the components explicitly:

```text
Module Registry Repository
          |
          v
Module Discovery Service
          |
          | implements server.DiscoveryPort
          v
Plugin SDK server.Adapter
          |
          v
gRPC PluginService
```

The Module should provide an adapter in its integration/composition layer. Do not place Plugin Module business logic inside `sdks/plugin/server`.

The Module should not duplicate request validation, Protobuf mapping, or gRPC status translation already owned by the SDK adapter.

## 10. State and Availability Semantics

The current SDK protocol defines:

- `INSTALLED`
- `ENABLED`
- `DISABLED`
- `RUNNING`
- `ERROR`

The first discovery increment must expose these values without inventing additional wire states.

Availability is a separate boolean and must not be inferred blindly from state. The Module implementation must document the exact availability rule before returning it. For example, an installed-but-disabled Plugin may be installed and discoverable while not being executable. The exact behavior should be captured in tests.

Passive Plugins remain discoverable and manageable, but they are not executable by the Executor. The discovery layer must preserve their type.

No lifecycle transition API is included in this increment.

## 11. Error Boundary

The Module should return SDK-neutral domain/application errors to the integration adapter. The SDK adapter maps these to gRPC status and structured `PluginError` details.

Initial discovery errors should distinguish at least:

- Plugin not found.
- Invalid query or pagination token.
- Registry unavailable or inconsistent.
- Internal failure.

Do not expose storage paths, package internals, stack traces, credentials, or implementation-specific errors through the public SDK response.

The exact error-code vocabulary must align with the existing Plugin SDK `DomainError` mapping and be covered by contract tests. This design does not add new public error codes.

## 12. Explicitly Deferred Work

The following are not part of the discovery foundation:

- Plugin package download and installation.
- Manifest loading from package files.
- Package signature verification and trust evaluation.
- Plugin update and uninstall.
- Enable/disable/reload lifecycle transitions.
- Active Plugin execution runtime.
- Passive Plugin UI integration.
- Permission evaluation or authorization.
- Marketplace publication and Registry Repository synchronization.
- Core capability matching and Execution Plan generation.
- Persistent storage selection and migrations.
- Distributed or remote Plugin execution.

These features require their own contracts, security model, and tests.

## 13. Contract Issues to Resolve Before Lifecycle and Execution

The stable Plugin SDK Specification and the draft SDK Contract do not use identical lifecycle verbs. The stable specification lists INSTALL, UPDATE, ENABLE, DISABLE, RELOAD, UNINSTALL, while the draft contract lists INSTALL, ENABLE, DISABLE, START, STOP, UNINSTALL.

The current Protobuf protocol intentionally contains only five read-only discovery RPCs. Do not add lifecycle or execution RPCs until the lifecycle and execution contracts are reconciled.

The Plugin Module Specification also says that Plugin Runtime receives execution requests from the SDK, while the current SDK protocol has no execution RPC. This is a known sequencing gap, not permission to add an unreviewed RPC.

## 14. Implementation Sequence

The recommended sequence after this design is accepted:

### Increment 1 — Domain and Registry Foundation

- Add `Modules/Plugin_Module` Go module.
- Define SDK-neutral Plugin, Manifest, Capability, Permission metadata, and State domain types.
- Define a narrow Registry repository interface.
- Add deterministic in-memory repository for tests only.
- Add validation and registry error contracts.
- Unit-test identity uniqueness and record invariants.

### Increment 2 — Discovery Application Service

- Implement list, get, find-by-capability, metadata, and state queries.
- Add deterministic pagination and filters.
- Add tests for empty registry, filters, pagination, not-found, invalid tokens, and context cancellation.

### Increment 3 — SDK Adapter Integration

- Implement an adapter that satisfies `server.DiscoveryPort`.
- Compose it with `server.New` and the generated Plugin gRPC service.
- Add bufconn integration tests that call all five public SDK RPCs through the Module implementation.
- Verify request/response metadata and error mapping.

### Increment 4 — Persistent Storage Decision and Adapter

- Select the storage mechanism based on the platform storage specification and operational requirements.
- Implement repository persistence and migration strategy.
- Keep the in-memory repository as a test double.

### Increment 5 — Loader and Lifecycle Design

- Define package acquisition, integrity verification, manifest validation, atomic installation, rollback, and state transitions.
- Reconcile SDK lifecycle contract before adding public lifecycle operations.

### Increment 6 — Runtime and Execution

- Define Active Plugin runtime isolation, process model, resource limits, invocation protocol, cancellation, and failure recovery.
- Keep Passive Plugins outside Executor execution.
- Add SDK execution contract only after architecture review.

## 15. Acceptance Criteria for the First Code Increment

The first implementation PR following this design should be considered complete only when:

1. The Module has its own Go module and domain package.
2. Domain types are independent of Protobuf transport types.
3. The local Plugin Registry has a narrow repository interface.
4. A deterministic in-memory repository exists as a test double, not as an implied production persistence layer.
5. Plugin identity uniqueness is enforced.
6. Manifest data and protected capability/permission metadata are separate.
7. Permission IDs remain metadata; no permission evaluation is implemented.
8. No Core capability matching or selection logic is added.
9. No lifecycle or runtime behavior is added.
10. Unit tests cover domain and registry invariants.
11. The Module's public integration surface is limited to the five existing read-only discovery operations.
12. Existing Plugin SDK tests remain unaffected.

## 16. Related Documents

- [PRIDE Plugin Module Specification](../04-Platform%20Specification%20Standard/Module%20Specification/PRIDE%20Plugin%20Module%20Specification.md)
- [PRIDE Plugin SDK Specification](../04-Platform%20Specification%20Standard/SDK%20Specification/PRIDE%20Plugin%20SDK%20Specification.md)
- [PRIDE Plugin SDK Module Boundary Design](../sdks/DOCS/SDK%20Spesification/SDK%20Plugin/PRIDE%20Plugin%20SDK%20Module%20Boundary%20Design.md)
- [PRIDE Plugin SDK Discovery Integration Guide](../sdks/DOCS/SDK%20Spesification/SDK%20Plugin/PRIDE%20Plugin%20SDK%20Discovery%20Integration%20Guide.md)
- [PRIDE Plugin Manifest Specification](../04-Platform%20Specification%20Standard/Core%20Standard%20Spesification/PRIDE%20Plugin%20Manifest%20Specification.md)
- [ADR-002 — Plugin SDK as the Official Plugin Communication Boundary](ADR/ADR-002%20%E2%80%94%20Plugin%20SDK%20as%20the%20Official%20Plugin%20Communication%20Boundary.md)
- [ADR-004 — Capability Registry](ADR/ADR-004%20%E2%80%94%20Capability%20Registry.md)
- [ADR-008 — Marketplace + Registry Architecture](ADR/ADR-008%20%E2%80%94%20Marketplace%20%2B%20Registry%20Architecture.md)
- [ADR-010 — PRIDE SDK Ecosystem Architecture](ADR/ADR-010%20%E2%80%94%20PRIDE%20SDK%20Ecosystem%20Architecture.md)
