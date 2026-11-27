# ADR-003 — Official Interface as the Platform Interaction Layer

**Status:** Accepted (Amended by ADR-013)  
**Original date:** 2026-08-11  
**Amendment date:** 2026-11-27  
**Category:** Architecture  
**Decision ID:** ADR-003  
**Amends:** ADR-013 — Unlimited Plugin Extensibility and Composition

---

## 1. Context

PRIDE requires a consistent and governed interaction model through which users and external interaction surfaces can access platform workflows.

The platform contains independently evolving components, including Plugins, Workflow, AI, Memory, Marketplace, and other platform services. Their implementations must remain independent from the user-interface implementation.

The original ADR established the built-in Official Interface as the permanent interaction layer and allowed Passive Plugins to contribute embedded interfaces. ADR-013 subsequently established that almost all non-kernel implementations must be replaceable and explicitly accepted that a Passive Plugin may replace or disable the built-in Official Interface.

This amendment reconciles the two decisions. It preserves the Official Interface's platform interaction contract and unified-workspace goals while making the built-in interface implementation replaceable.

## 2. Decision

PRIDE adopts the **Official Interface** as the canonical platform interaction contract and reference experience, not as an irreplaceable application implementation.

The Official Interface is the platform-defined interaction boundary through which a compliant user-facing implementation presents PRIDE and submits user intent into platform workflows.

The built-in Official Interface is the default platform-provided implementation of that contract. A compliant Passive Plugin may extend it or replace it as the selected primary interface through a governed platform mechanism.

The platform must preserve a trusted recovery and administration path independently of the currently selected interface implementation.

## 3. Official Interface Definition

The Official Interface defines the canonical user-facing interaction responsibilities and contract for PRIDE.

It establishes how a user-facing implementation:

- Presents PRIDE as a coherent platform.
- Exposes available platform workspaces and user-facing capabilities.
- Submits user intent through supported platform workflow boundaries.
- Presents workflow status, results, and actionable errors.
- Provides access to platform administration functions according to authorization.
- Hosts compatible Plugin-provided interface extensions when the implementation supports extension hosting.

The Official Interface contract is distinct from any specific UI framework, rendering technology, layout, navigation model, or bundled application.

A platform implementation may satisfy this contract without using the built-in Official Interface code.

## 4. Built-in Interface and Replacement Model

The built-in Official Interface is the default implementation supplied by PRIDE. It is not a permanent architectural dependency.

A compliant Passive Plugin may:

- Contribute interface extensions hosted by the currently selected interface.
- Register an alternative implementation of the Official Interface contract.
- Become the selected primary interface when explicitly activated by the user or an authorized platform policy.
- Cause the built-in implementation to be deactivated while the replacement is selected, where the replacement meets the applicable compatibility and recovery requirements.

The platform must define a governed interface-selection mechanism. At minimum, that mechanism must address:

- Interface registration and identity.
- Contract and platform-version compatibility.
- Selection and activation authority.
- Replacement conflicts and deterministic selection.
- Deactivation and rollback.
- Health/readiness checks where applicable.
- Failure fallback to a known-good interface.
- Auditability of selection changes.

This ADR establishes the architectural requirement, not the detailed selection protocol or UI Plugin packaging format. Those require a separate contract.

A Plugin category such as Passive is descriptive. It does not itself grant authority to replace the primary interface. Replacement requires explicit platform authorization and successful contract validation.

## 5. Unified Workspace Principle

PRIDE should provide a coherent platform experience even when the selected interface is implemented by a Plugin.

The unified-workspace principle concerns consistency of platform identity, workflow access, capability presentation, and administration boundaries. It does not require one immutable UI implementation or one visual design.

Different interface implementations may present PRIDE differently, provided they preserve the required platform contracts and do not bypass platform governance.

## 6. Platform-First Presentation

The selected interface presents platform services and Plugin capabilities as parts of the PRIDE ecosystem.

The interface may organize, compose, and present those capabilities according to its own user-experience design. It does not become the owner of the services or capabilities it presents.

A replacement interface may introduce a substantially different interaction paradigm. PRIDE does not require visual or navigational equivalence with the built-in implementation.

## 7. Service Independence

Every platform service remains architecturally independent from the selected interface implementation.

The interface communicates with platform services through supported SDKs and contracts. It must not depend on private implementation details, internal database handles, direct process access, or other unsupported boundaries.

A service may evolve independently without requiring the interface implementation to assume responsibility for its internal behavior.

## 8. Passive Plugin Interface Integration

Passive Plugins may provide user-facing experiences at two levels:

1. **Embedded extension:** contributes a navigation workspace, panel, sidebar, dialog, settings surface, or another supported extension point to the selected interface.
2. **Primary interface replacement:** implements the Official Interface contract and may be selected as the primary user-facing implementation.

The Plugin determines the experience it implements. The platform-defined extension or replacement contract determines how that experience is registered, activated, authorized, and connected to PRIDE.

A compliant interface implementation communicates with platform services through official SDKs and contracts.

## 9. User Interaction and Workflow Boundary

User intent may originate from the built-in Official Interface, a Passive Plugin replacement, another authorized interface, or a supported non-graphical interaction surface.

Regardless of its origin, a user request that invokes platform workflow must enter through the canonical platform workflow boundary.

The interface must not execute or bypass Core-owned workflow orchestration, capability authorization, or platform policy.

The logical interaction flow is:

```text
User or Authorized Interaction Surface
                 |
                 v
       Selected Interface
                 |
                 v
      Official Interface Contract
                 |
                 v
       Core Workflow Boundary
                 |
                 v
       Platform Capabilities
                 |
                 v
       Selected Interface
                 |
                 v
                User
```

This diagram represents the logical boundary. It does not mandate a specific transport or require every internal platform operation to originate from a graphical interface.

## 10. Official Interface Responsibilities

The selected implementation of the Official Interface contract is responsible for the user-facing responsibilities it claims and is authorized to provide, including:

1. Presenting PRIDE as a coherent platform experience.
2. Presenting available platform workspaces and capabilities.
3. Submitting user intent through supported workflow contracts.
4. Presenting platform results, status, and errors.
5. Hosting compatible Passive Plugin interface extensions when supported.
6. Exposing Plugin management, Marketplace, and Settings entry points when its contract profile requires them.
7. Respecting platform authorization and user-confirmation requirements.
8. Maintaining compatibility with the platform interaction contract.

Not every interface implementation must provide every optional presentation surface. Required and optional contract profiles must be defined separately.

## 11. Non-Responsibilities

The Official Interface contract and its implementations are not responsible for:

- Core workflow orchestration.
- Planning or task decomposition as a platform authority.
- AI reasoning or Agentic orchestration.
- Plugin implementation or lifecycle authority.
- Memory implementation.
- Capability matching and authorization decisions.
- Credential or secret ownership.
- SDK transport implementation beyond using supported SDKs.
- Platform trust-root management.

These responsibilities belong to their respective platform components.

## 12. Trusted Recovery and Administration

Replacing the built-in interface must not remove the user's ability to recover or administer PRIDE.

The platform must retain a trusted recovery path that is not controlled solely by the currently selected interface Plugin.

At minimum, recovery must allow an authorized user or operator to:

- Deactivate a failed or incompatible replacement.
- Restore a known-good interface implementation.
- Access Plugin lifecycle and security administration.
- Inspect safe failure information needed for recovery.

The recovery path must be protected by platform authentication and authorization. A replacement interface cannot disable, impersonate, or silently take ownership of this path.

The concrete recovery surface and its host integration are deferred to a dedicated design.

## 13. Stable Interaction Contract

The stable element is the Official Interface contract and its workflow boundary, not the visual implementation.

The contract must be versioned and evolve through explicit compatibility rules. Interface implementations may adopt different UI frameworks, layouts, navigation models, and interaction paradigms while remaining compliant.

## 14. Architectural Invariants

1. PRIDE defines one canonical Official Interface interaction contract.
2. The built-in Official Interface is the default implementation, not an irreplaceable platform authority.
3. A compliant Passive Plugin may extend or replace the selected primary interface through a governed mechanism.
4. Interface replacement requires explicit activation authority and compatibility validation.
5. The platform retains a trusted recovery and administration path independent of the selected replacement.
6. User-facing implementations submit workflow requests through supported platform boundaries.
7. The interface does not own or bypass Core workflow orchestration.
8. The interface does not become the implementation owner of AI, Memory, Plugins, capability matching, or authorization.
9. Interface implementations communicate with platform services through official SDKs and contracts.
10. Unified platform identity and workflow access do not require visual or implementation uniformity.
11. Plugin categories do not independently grant interface replacement authority.
12. The Official Interface contract is versioned and implementation-neutral.

## 15. Consequences

### Positive Consequences

- **Replaceable experience:** users and organizations can choose alternative interface implementations.
- **Stable platform boundary:** workflow and security responsibilities remain independent from UI technology.
- **Plugin extensibility:** Passive Plugins can provide embedded or complete interface experiences.
- **Recovery safety:** a broken replacement cannot permanently remove platform administration.
- **Implementation independence:** interface implementations can evolve without coupling to service internals.

### Costs and Risks

- **Interface governance:** selection, compatibility, activation, and fallback require explicit contracts.
- **Recovery complexity:** the platform must maintain a protected recovery path.
- **Compatibility burden:** interface contracts require versioning and conformance tests.
- **Security surface:** replacement implementations require strict authorization and runtime boundaries.

## 16. Alternatives Considered

### 16.1 Built-in Official Interface as a Permanent Implementation

**Rejected.** This conflicts with ADR-013's replaceability decision and creates unnecessary implementation lock-in.

### 16.2 Independent Uncoordinated Interfaces for Every Service

**Rejected.** Interfaces may differ, but they must still use the canonical platform contracts and preserve coherent access to PRIDE.

### 16.3 Plugin Directly Replaces Core Workflow

**Rejected.** Replacing a presentation implementation does not grant authority over Core workflow orchestration, identity, authorization, or platform policy.

## 17. Relationship With Other Decisions

- **ADR-013 — Unlimited Plugin Extensibility and Composition:** establishes the replaceability requirement and is authoritative where earlier wording treated the built-in interface as permanent.
- **ADR-002 — Plugin SDK:** remains the official communication boundary for Plugin-facing interactions.
- **ADR-004 — Capability Registry:** remains responsible for capability matching and selection, not UI implementation.
- **ADR-009 — Stable Core Workflow:** remains authoritative for Core workflow orchestration.
- **PLR-ADR-0003 — Runtime Execution:** remains authoritative for Plugin runtime isolation and host enforcement.

## 18. Decision Outcome

PRIDE shall maintain a canonical **Official Interface contract** and a default built-in implementation, while allowing a compliant Passive Plugin to replace the selected primary interface through explicit platform governance.

The interface implementation may change. The platform workflow, authorization, trust, and recovery boundaries remain stable.

The Official Interface is therefore a replaceable platform experience over a stable interaction contract, not a permanent lock-in to one UI implementation.
