# ADR-002 — Plugin SDK as the Official Plugin Communication Boundary

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-002

---

# 1. Context

PRIDE uses Plugins as the primary extension mechanism of the platform.

Because Plugins are independently implemented extensions, platform components must not become dependent on Plugin implementation details.

Direct communication between platform components and individual Plugin implementations would create architectural coupling.

Such coupling would make Plugin implementations harder to replace and would allow implementation-specific behavior to leak into higher platform layers.

The architecture therefore requires one stable communication boundary between the PRIDE Platform and the Plugin ecosystem.

The existing Plugin SDK specification defines this boundary by establishing the Plugin SDK as the standardized communication interface between platform components and the Plugin Module.

---

# 2. Decision

PRIDE adopts the **Plugin SDK as the official communication boundary for all Plugin operations**.

Every operation involving a Plugin shall pass through the Plugin SDK.

Platform components shall not communicate directly with:

* installed Plugin implementations,
* individual Plugin runtimes,
* Plugin implementation internals, or
* Plugin Module implementation details.

The Plugin SDK defines the official programming contract through which platform components request Plugin operations.

The Plugin SDK defines **how the platform requests Plugin operations**, not **how Plugin implementations perform those operations internally**.

---

# 3. Architectural Model

The communication relationship is:

```text
Platform Component
        │
        ▼
   Plugin SDK
        │
        ▼
   Plugin Module
        │
        ▼
Plugin Implementation
```

The SDK is therefore the boundary between platform-level communication and Plugin implementation.

Platform components remain independent from the internal architecture of individual Plugins.

---

# 4. Official Communication Gateway

The Plugin SDK is the single official communication gateway for Plugin operations.

The following rule is mandatory:

```text
Platform Component
        ↓
    Plugin SDK
        ↓
   Plugin Module
        ↓
      Plugin
```

The following architecture is prohibited:

```text
Platform Component
        ↓
      Plugin
```

No platform component may bypass the Plugin SDK to invoke a Plugin directly.

The source architecture explicitly defines every Plugin operation as passing through the Plugin SDK.

---

# 5. Responsibilities of the Plugin SDK

The Plugin SDK is responsible for providing standardized interfaces for:

* Plugin discovery.
* Plugin lifecycle operations.
* Plugin execution.
* Plugin metadata access.
* Plugin state information.
* Standardized Plugin communication.
* Standardized SDK responses.
* Standardized SDK errors.

These responsibilities are consistent with the existing Plugin SDK architecture.

---

# 6. Separation of Responsibility

The architecture establishes the following separation:

```text
Plugin SDK
    │
    └── Defines communication

Plugin Module
    │
    └── Defines Plugin implementation management

Plugin
    │
    └── Defines functionality
```

The SDK shall not implement Plugin functionality.

The SDK shall not contain Plugin-specific implementation logic.

The SDK shall not become responsible for executing Plugin internals.

The SDK exists to provide the stable communication contract between the platform and the Plugin ecosystem.

---

# 7. Implementation Independence

The Plugin SDK shall remain independent from:

* Programming languages.
* Runtime environments.
* Plugin execution technologies.
* Packaging formats.
* Plugin storage mechanisms.
* Vendor-specific implementations.

A Plugin may therefore change its internal implementation without requiring platform components to change, provided that the Plugin remains compatible with the established SDK contract.

---

# 8. Plugin Categories

The SDK boundary applies to both major Plugin categories defined by the architecture.

## 8.1 Active Plugins

Active Plugins provide executable functionality and participate in workflow execution.

Examples include:

* File System.
* Browser.
* Git.
* Calculator.
* Python Runtime.
* Terminal.

Active Plugins are executed exclusively through the Plugin SDK.

---

## 8.2 Passive Plugins

Passive Plugins extend platform behavior without participating directly in workflow execution.

Examples include:

* Custom User Interface.
* Memory Explorer.
* Dashboard.
* Sidebar.
* Theme.
* Settings Panel.

Passive Plugins are managed by the Official Interface through the Plugin SDK and are not executed by the Executor.

---

# 9. State Boundary

Plugin state shall remain under the responsibility of the Plugin Module.

The Plugin SDK may expose standardized Plugin state information to platform components.

Platform components shall not directly modify Plugin state through implementation-specific mechanisms.

State transitions remain the responsibility of the Plugin Module.

---

# 10. What This Decision Does Not Define

This ADR does not define:

* Plugin implementation.
* Plugin Module internal architecture.
* Plugin Manifest structure.
* Plugin packaging.
* Marketplace architecture.
* Plugin Registry architecture.
* Plugin permission architecture.
* Workflow planning.
* Workflow execution strategy.
* Capability Registry behavior.
* Memory SDK behavior.
* Official Interface implementation.

Those concerns remain separate architectural decisions or specifications.

---

# 11. Consequences

## Positive Consequences

### Stable Contract

Platform components have one standardized mechanism for communicating with Plugins.

### Implementation Independence

Plugin implementations can evolve without exposing their internals to platform components.

### Reduced Coupling

Platform components do not need to know how individual Plugins are implemented.

### Consistent Architecture

All Plugin operations follow the same communication model.

### Easier Evolution

New Plugin technologies and execution models can be introduced without redesigning the platform communication architecture.

### Clear Responsibility

Communication belongs to the SDK, implementation management belongs to the Plugin Module, and functionality belongs to the Plugin.

---

## Negative Consequences

### SDK Must Remain Stable

Changes to the Plugin SDK can affect a large portion of the Plugin ecosystem.

### Additional Abstraction

Plugin communication requires an SDK boundary rather than direct invocation.

### Contract Governance

The SDK must be carefully maintained to prevent implementation-specific behavior from entering the public contract.

---

# 12. Alternatives Considered

## 12.1 Direct Plugin Invocation

Platform components directly invoke Plugin implementations.

**Rejected.**

This would expose implementation details and create tight coupling between platform components and individual Plugins.

---

## 12.2 Multiple Independent Plugin Interfaces

Each platform component communicates with Plugins through its own interface.

**Rejected.**

This would create inconsistent communication models and duplicate integration logic.

---

## 12.3 SDK Only for Active Plugins

Passive Plugins communicate through separate platform mechanisms.

**Rejected.**

The architecture already establishes the Plugin SDK as the standardized communication layer for both executable and user-facing Plugin extensions. Passive Plugins are managed through the Official Interface via the Plugin SDK.

---

# 13. Architectural Invariants

The following rules are established as architectural invariants:

1. Every Plugin operation passes through the Plugin SDK.
2. Platform components shall never directly invoke Plugin implementations.
3. The Plugin SDK is the official Plugin communication gateway.
4. Plugin implementation details shall remain hidden behind the SDK boundary.
5. Plugin functionality belongs to Plugins, not the SDK.
6. Plugin implementation management belongs to the Plugin Module.
7. Active Plugins use the Plugin SDK for execution.
8. Passive Plugins use the Plugin SDK for platform integration.
9. Plugin state may be exposed through the SDK but shall not be directly manipulated by platform components.
10. SDK consumers shall depend on the SDK contract rather than Plugin implementation details.
11. Plugin implementation technologies may evolve without requiring platform architectural changes as long as the SDK contract remains satisfied.

---

# 14. Relationship With Other Decisions

This ADR depends on:

* **ADR-001 — Plugin-First Architecture**

ADR-001 establishes Plugins as the primary extension mechanism.

ADR-002 establishes the communication boundary that makes that architecture enforceable.

The following decisions build upon this boundary:

* **ADR-003 — Official Interface Architecture**
* **ADR-004 — Capability Registry Architecture**
* **ADR-005 — Execution Architecture**
* **ADR-006 — Memory SDK Boundary**
* **ADR-007 — Memory Architecture**
* **ADR-008 — Marketplace & Registry Architecture**
* **ADR-009 — Workflow Architecture**

---

# 15. Decision Outcome

The PRIDE Platform shall communicate with the Plugin ecosystem exclusively through the Plugin SDK.

The SDK becomes the stable architectural boundary between platform components and Plugin implementations.

Plugin internals remain replaceable.

Platform components remain implementation-independent.

The Plugin ecosystem can therefore evolve without requiring direct architectural coupling to the Core.

This decision establishes the official communication boundary required to enforce the Plugin-First architecture.
