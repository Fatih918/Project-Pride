# ADR-004 — Capability Registry

**Status:** Accepted (Amended by ADR-013)  
**Original date:** 2026-08-11  
**Amendment date:** 2026-11-27  
**Category:** Architecture  
**Decision ID:** ADR-004  
**Amends:** ADR-013 — Unlimited Plugin Extensibility and Composition

---

## 1. Context

After Planning has produced a Task Tree, PRIDE must determine which installed Plugin capabilities can perform the required tasks.

The Task Tree defines **what** must be accomplished, but does not necessarily define **which implementation** should perform each task.

PRIDE therefore requires a Core component responsible for capability resolution and selection while remaining independent from Plugin implementations.

ADR-013 establishes that Plugin capabilities are open-ended, Plugin categories are non-exhaustive, and Plugins may compose capabilities exposed by other Plugins. The Capability Registry must therefore recognize dynamically registered capability metadata without requiring a Core code change for every new capability identifier.

This amendment clarifies that open-ended registration does not mean arbitrary unknown operations can be executed. Capability recognition, compatibility, provider selection, and authorization remain distinct concerns.

## 2. Decision

PRIDE adopts the **Capability Registry** as the Core component responsible for capability resolution and provider selection for executable tasks.

The Capability Registry shall:

- Receive the Task Tree produced by Planning.
- Obtain installed Plugin capability declarations through the Plugin Module.
- Resolve capability identifiers and contract compatibility.
- Match executable tasks against compatible capability providers.
- Select or produce a deterministic set of eligible providers according to platform policy.
- Resolve declared capability dependencies where required by the supported execution-plan contract.
- Generate a standardized Execution Plan.
- Forward the completed Execution Plan to the Executor.

The Capability Registry shall not execute Plugins or implement Plugin-provided capabilities.

The set of capability identifiers is open-ended. The Registry must not rely on a closed, hard-coded list of all capabilities that Plugins may introduce.

## 3. Core Ownership

The Capability Registry is part of PRIDE Core.

Core owns platform-level capability governance, matching, selection policy, and authorization boundaries. The Registry remains independent from individual Plugin implementations and must not contain their feature-specific business logic.

The Plugin Module owns Plugin registry and discovery responsibilities and provides the Registry with installed Plugin metadata and declared capability information.

Plugins own the implementation of their declared capabilities.

Capability metadata supplied by a Plugin is a declaration, not proof that the Plugin is trustworthy, compatible, available, authorized, or capable of successfully completing a task.

## 4. Input Boundary

The Task Tree is the primary input to capability resolution.

It may contain:

- Task identifiers.
- Task metadata and goals.
- Task dependencies.
- Required capability identifiers.
- Required capability contract versions or compatibility constraints.
- Input/output requirements needed for matching.

The Capability Registry must not silently rewrite the semantic intent of the Task Tree. It may produce a separate resolution result and Execution Plan containing provider assignments and resolved dependencies.

## 5. Plugin Discovery Boundary

The Capability Registry does not communicate directly with individual Plugin implementations.

Plugin discovery occurs through the Plugin Module. The Plugin Module provides installed Plugin identities, immutable versions, lifecycle availability, and declared capability metadata through supported contracts.

The Registry uses this information to construct candidate providers. It does not inspect Plugin source code, private runtime state, or implementation-specific storage.

## 6. Open-Ended Capability Model

Capability identifiers are open-ended and must support namespaced ownership.

A capability declaration must be representable as data and must be able to carry the metadata required for safe matching, such as:

- Namespaced capability identifier.
- Capability contract version.
- Human-readable name and description.
- Input and output contract references or schemas, where applicable.
- Declared dependency capabilities and version constraints, where applicable.
- Compatibility requirements.
- Optional descriptive metadata.

The exact canonical schema, namespace ownership rules, versioning semantics, and validation limits must be defined in a separate Capability Contract specification.

The Registry must be able to store, discover, compare, and preserve capability identifiers it does not itself implement. A new identifier alone must not require a Core source-code change.

Unknown capability metadata is inert. The Registry must not interpret arbitrary metadata as executable instructions, dynamic code, authorization grants, or trusted policy.

A capability is executable only when a consumer or execution component supports its contract and the relevant provider passes compatibility, lifecycle, trust, and authorization checks.

## 7. Capability Matching and Compatibility

The Registry compares Task Tree requirements against capability declarations from available Plugin providers.

Matching must distinguish at least:

1. **Identity match:** the namespaced capability identifier represents the required capability.
2. **Contract compatibility:** the provider's contract version and input/output requirements satisfy the task's constraints.
3. **Platform compatibility:** the Plugin and capability are compatible with the current platform/runtime requirements.
4. **Lifecycle availability:** the Plugin is installed and in an eligible administrative/runtime state for the requested operation.
5. **Trust validity:** the installed artifact and its metadata have passed the applicable platform verification requirements.
6. **Authorization:** the relevant principal and execution context are permitted to invoke the capability.

A declaration or metadata match alone is not sufficient for selection or execution. Core authorization remains authoritative, and runtime enforcement remains the host's responsibility.

## 8. Provider Selection

A task may have zero, one, or multiple compatible Plugin providers.

The Registry applies platform selection policy to eligible candidates. The policy may account for explicit user preference, compatibility, availability, priority, configured routing, and other documented selection inputs.

The Registry may return a selected provider or a deterministic candidate set when selection is intentionally deferred to a later policy or execution-planning stage.

A Plugin's self-declared priority or metadata must not override platform policy or authorization.

When no compatible and authorized provider exists, the Registry must return a typed resolution result rather than inventing a provider.

## 9. Plugin Composition and Capability Dependencies

A Plugin may declare dependencies on capabilities exposed by other Plugins. A capability may also be fulfilled by a composition of multiple providers when the relevant contracts explicitly support that model.

The Registry may resolve declared dependency relationships into a dependency graph or Execution Plan. It must not execute the graph.

Composition resolution must account for:

- Explicit dependency declarations.
- Compatible contract versions.
- Immutable Plugin version identities.
- Dependency availability and lifecycle state.
- Dependency-cycle detection.
- Deterministic conflict handling.
- Per-invocation authorization.
- Failure and partial-resolution reporting.

A dependency declaration does not grant the dependent Plugin access to the dependency's private data, process, filesystem, credentials, or permissions. Invocation occurs through supported contracts and is authorized under the applicable principal and policy.

The detailed dependency manifest format and composition semantics require a separate decision/specification.

## 10. Deterministic Resolution

The same resolution context should produce the same result.

The resolution context includes, at minimum:

- Task Tree and capability constraints.
- Installed Plugin and capability snapshot.
- Immutable Plugin versions.
- Relevant lifecycle and availability snapshot.
- Applicable selection policy and its revision.
- Relevant compatibility rules and platform version.
- User or administrator routing preferences that affect selection.

Given the same complete resolution context, the Registry should produce the same selected provider set, dependency graph, and Execution Plan.

Time-varying availability, policy changes, or Plugin installation changes are different resolution contexts and may legitimately produce different results.

## 11. Execution Plan

The Registry produces a standardized Execution Plan after resolving the required capabilities.

The plan may define:

- Executable tasks.
- Required capability contracts.
- Selected Plugin identities and immutable versions.
- Capability/provider assignments.
- Task and capability dependency edges.
- Execution order or concurrency constraints.
- Compatibility and policy evidence references needed by the execution boundary.

The Execution Plan describes **which compatible provider is selected and how tasks depend on one another**. It does not contain Plugin implementation logic, private runtime details, raw credentials, or authorization secrets.

The Execution Plan contract must evolve to support composed provider graphs without coupling Core to a particular Plugin implementation.

## 12. Execution Boundary

The Capability Registry does not execute work.

Its responsibility ends after it produces the standardized resolution result and Execution Plan and forwards them to the Executor.

The Executor and runtime host remain responsible for execution under their respective contracts and security boundaries.

```text
Planning
   |
   v
Task Tree
   |
   v
Capability Registry <---- Plugin Module
   |                          |
   |                    Installed Plugin
   |                    Capability Metadata
   |
   v
Resolved Provider Set
   |
   v
Execution Plan
   |
   v
Executor
   |
   v
Plugin Runtime
```

The diagram is logical and does not mandate a specific transport or internal deployment topology.

## 13. Non-Responsibilities

The Capability Registry is not responsible for:

- Planning or changing task intent.
- Executing Plugins.
- Implementing Plugin capabilities.
- Performing AI reasoning or Agentic orchestration.
- Managing Memory.
- Plugin installation or lifecycle mutation.
- Owning Plugin runtime processes.
- Issuing authorization grants.
- Resolving secrets or credentials.
- Bypassing the Plugin Module discovery boundary.

Those responsibilities belong to their respective platform components.

## 14. Registry Lifecycle

The Registry follows this logical lifecycle:

1. Receive and validate the Task Tree.
2. Query the Plugin Module for an appropriate installed Plugin/capability snapshot.
3. Parse and validate capability declarations under the supported metadata contract.
4. Match task requirements to capability identifiers and contract versions.
5. Filter incompatible, unavailable, untrusted, or unauthorized candidates.
6. Resolve required capability dependencies and composition constraints.
7. Apply deterministic provider-selection policy.
8. Produce a typed resolution result and Execution Plan, or a typed failure.
9. Forward a valid Execution Plan to the Executor.

The Registry does not invoke Plugin implementations during resolution.

## 15. Error Model

The Registry may encounter errors including:

- Invalid Task Tree.
- Invalid Capability Declaration.
- Unknown Capability Contract.
- Unsupported Contract Version.
- Missing Capability.
- No Compatible Provider.
- No Authorized Provider.
- Plugin Unavailable.
- Plugin Disabled.
- Dependency Not Found.
- Dependency Version Conflict.
- Dependency Cycle.
- Incompatible Execution Plan.
- Plugin Module Discovery Failure.
- Registry Failure.
- Unknown Internal Error.

Errors must use the standardized PRIDE platform error model and must not expose secrets or unsafe implementation details.

## 16. Architectural Invariants

1. The Capability Registry is part of PRIDE Core.
2. Capability matching and provider selection are Core responsibilities.
3. The Registry receives the Task Tree from Planning.
4. Plugin discovery occurs through the Plugin Module.
5. The Registry does not communicate directly with Plugin implementations.
6. Capability identifiers are open-ended and namespaced; they are not a closed Core enum.
7. Unknown capability metadata may be retained and discovered but is not automatically executable.
8. Capability declarations are metadata, not authorization grants or proof of trust.
9. Contract compatibility, lifecycle availability, trust, and authorization are evaluated separately from identifier matching.
10. The Registry selects providers but does not execute them.
11. The Registry may resolve explicit Plugin/capability dependencies without executing them.
12. Plugin composition uses versioned contracts and authorized invocation boundaries.
13. A Plugin does not inherit permissions from its dependencies.
14. The Execution Plan contains provider identities, immutable versions, and dependency relationships, not implementation-specific logic or secrets.
15. Identical complete resolution contexts should produce deterministic results.
16. New compatible Plugin capabilities may be introduced without changing Core source code solely to register their identifiers.
17. Core authorization remains authoritative and the runtime host enforces execution isolation.

## 17. Consequences

### Positive Consequences

- **Open extensibility:** new Plugin capabilities can be registered without expanding a hard-coded Core enum.
- **Stable Core:** capability matching remains a platform responsibility while capability implementations remain in Plugins.
- **Composition:** tasks can be resolved to explicit provider/dependency graphs.
- **Security clarity:** declarations, compatibility, trust, availability, and authorization are not conflated.
- **Deterministic planning:** provider resolution is reproducible for the same complete context.
- **Implementation independence:** Plugin implementations can evolve behind stable contracts.

### Negative Consequences

- **Metadata governance:** capability identifiers and contract schemas require validation and versioning.
- **Resolution complexity:** compatibility and dependency graphs increase the Registry's responsibility.
- **Policy complexity:** provider choice may depend on user preference, platform policy, availability, and authorization.
- **Testing burden:** composed plans require contract and integration testing.

These costs are accepted to support the open-ended Plugin ecosystem established by ADR-013.

## 18. Alternatives Considered

### 18.1 Closed Capability Enum in Core

**Rejected.** It would require Core changes whenever the ecosystem introduces a new capability identifier and would contradict ADR-013.

### 18.2 Direct Plugin Selection by Planning

**Rejected.** This merges planning and provider resolution responsibilities.

### 18.3 Direct Registry Discovery of Plugin Implementations

**Rejected.** Discovery remains mediated by the Plugin Module.

### 18.4 Registry-Owned Plugin Execution

**Rejected.** Capability resolution and execution remain separate responsibilities.

## 19. Relationship With Other Decisions

- **ADR-001 — Plugin-First Architecture:** establishes the Plugin-first platform direction.
- **ADR-002 — Plugin SDK:** defines the official Plugin communication boundary.
- **ADR-003 — Official Interface:** defines the replaceable Official Interface contract and user interaction boundary.
- **ADR-009 — Stable Core Workflow:** remains authoritative for workflow orchestration.
- **ADR-013 — Unlimited Plugin Extensibility and Composition:** establishes open-ended capability, replaceability, and Plugin composition requirements.
- **PLR-ADR-0002 — Lifecycle Persistence:** remains authoritative for durable lifecycle state and recovery.
- **PLR-ADR-0003 — Runtime Execution:** remains authoritative for runtime isolation and host enforcement.

## 20. Decision Outcome

PRIDE shall use the Capability Registry as the Core capability-resolution and provider-selection boundary.

The Registry receives task requirements, discovers Plugin capability declarations through the Plugin Module, evaluates identity and contract compatibility, filters candidates by platform constraints, resolves supported dependencies, applies deterministic selection policy, and produces a standardized Execution Plan.

The Registry remains independent from Plugin implementations and does not execute them.

Capability identifiers and Plugin use cases remain open-ended. The Core provides stable governance and resolution contracts rather than a closed catalog of all capabilities the PRIDE ecosystem may ever provide.
