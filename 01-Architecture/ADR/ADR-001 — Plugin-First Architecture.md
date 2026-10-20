# ADR-001 — Plugin-First Architecture

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-001

---

# 1. Context

PRIDE is designed as an extensible platform whose functionality must be able to evolve without requiring continuous modification of the platform Core.

The platform must support functionality developed independently from the internal implementation of the Core while maintaining stable architectural boundaries.

The architecture therefore requires a clear distinction between:

* platform functionality,
* platform Core,
* developer extensions,
* application-level functionality, and
* infrastructure.

The architecture must also allow the Plugin ecosystem to evolve independently from the internal implementation of PRIDE.

The PRIDE architecture already establishes Plugins as the functional extension layer of the platform and defines that Applications interact with the platform through Plugins rather than directly communicating with internal Core or Infrastructure components.

---

# 2. Decision

PRIDE adopts a **Plugin-First Architecture**.

Plugins are the primary mechanism through which external functionality is extended into the PRIDE platform.

The platform Core shall provide stable platform capabilities and architectural foundations, while Plugins provide functional extensions that consume those capabilities.

Applications shall interact with the PRIDE platform through Plugins rather than directly depending on internal Core or Infrastructure components.

The Core shall remain independent from individual Plugin implementations.

Plugin implementations shall remain replaceable without requiring the Core architecture to be redesigned.

---

# 3. Architectural Model

The resulting logical architecture is:

```text
Applications
      │
      ▼
   Plugins
      │
      ▼
     SDK
      │
      ▼
     Core
      │
      ▼
Infrastructure
```

This structure establishes Plugins as the extension boundary between applications and the platform's internal architecture.

Applications belong above the Plugin layer.

Plugins belong above the SDK and Core layers.

Core components remain independent from Applications and Plugin implementations.

Infrastructure remains below the Core and shall not define application-level or business behavior.

---

# 4. Core Principles

## 4.1 Plugins Are Platform Extensions

Plugins exist to extend the functionality of PRIDE.

A Plugin may provide functionality that is not implemented directly inside the Core.

The Core therefore does not need to contain every possible capability that may eventually exist within the PRIDE ecosystem.

---

## 4.2 Core Independence

The Core shall not depend on individual Plugin implementations.

The existence, implementation technology, programming language, runtime, or internal architecture of a Plugin shall not become a requirement of the Core.

The Core defines platform behavior independently of individual Plugins.

---

## 4.3 Application Isolation

Applications shall not directly communicate with internal Core components or Infrastructure.

Applications interact with the platform through the Plugin layer.

This prevents application implementations from becoming coupled to internal platform architecture.

---

## 4.4 Extension Over Core Modification

When new functionality can be provided as a Plugin without violating platform contracts, Plugin extension shall be preferred over modifying the Core.

The Core should only be modified when the required functionality represents a fundamental platform responsibility that cannot appropriately exist as an extension.

---

## 4.5 Implementation Independence

Plugin functionality shall remain independent from the internal implementation of the platform.

The platform must be able to evolve its Core and Infrastructure without requiring every Plugin implementation to be redesigned.

Likewise, Plugin implementations may evolve without requiring architectural changes to the Core.

---

# 5. Responsibility Boundaries

The Plugin-First architecture establishes the following responsibility boundaries.

### Application

Responsible for application-level functionality and user-facing application behavior.

### Plugin

Responsible for providing functional extensions to the platform.

### SDK

Responsible for exposing stable contracts through which Plugins communicate with the platform.

### Core

Responsible for platform rules, orchestration, lifecycle management, service coordination, and internal execution.

### Infrastructure

Responsible for low-level technical capabilities such as storage, networking, operating-system integration, filesystem access, and provider implementations.

Each layer shall maintain its own responsibility.

---

# 6. Dependency Direction

Dependencies shall follow the established architectural direction:

```text
Applications
      ↓
Plugins
      ↓
SDK
      ↓
Core
      ↓
Infrastructure
```

Higher layers may depend upon lower layers through official platform contracts.

Lower layers shall remain independent from higher layers.

Circular dependencies between architectural layers are prohibited.

---

# 7. What This Decision Does Not Define

This ADR does not define:

* Plugin SDK communication details.
* Plugin lifecycle operations.
* Plugin Manifest structure.
* Plugin packaging.
* Plugin Registry.
* Marketplace architecture.
* Plugin permission architecture.
* Plugin execution implementation.
* Workflow planning.
* Individual Plugin capabilities.
* Plugin implementation technologies.

Those concerns are defined by their respective specifications or ADRs.

---

# 8. Consequences

## Positive Consequences

### Extensibility

New functionality can be introduced through Plugins without requiring the Core to contain every possible feature.

### Core Stability

The Core can remain focused on fundamental platform responsibilities.

### Implementation Independence

Plugin implementations can evolve independently from the internal platform implementation.

### Reduced Coupling

Applications and Plugins do not need direct knowledge of internal Core or Infrastructure implementations.

### Ecosystem Growth

Independent developers and organizations can extend PRIDE without becoming dependent on internal Core architecture.

### Long-Term Evolution

The platform can evolve while preserving a stable extension model.

---

## Negative Consequences

### Additional Architectural Boundaries

Functionality provided through Plugins introduces additional contracts between Applications, Plugins, SDKs, and the Core.

### SDK Dependency

Plugins depend on stable SDK contracts rather than directly accessing internal platform components.

### Extension Discipline

Not every feature should automatically become a Plugin.

Fundamental platform responsibilities must remain within the appropriate Core layer.

### Architectural Governance

Plugin boundaries must be maintained carefully to prevent Plugin implementations from becoming tightly coupled to internal platform behavior.

---

# 9. Alternatives Considered

## 9.1 Core-First Architecture

Under a Core-First model, most platform functionality would be implemented directly inside the Core.

This was rejected because continuously expanding the Core would increase coupling, reduce extensibility, and make long-term evolution more difficult.

---

## 9.2 Direct Application-to-Core Architecture

Under this model, Applications would communicate directly with Core components.

This was rejected because it exposes internal platform architecture to Applications and increases coupling between applications and internal implementation.

---

## 9.3 Independent Extension Without a Defined Plugin Layer

Under this model, external functionality could integrate with the platform through multiple mechanisms without a standardized extension layer.

This was rejected because it would create inconsistent integration patterns and make architectural boundaries difficult to maintain.

---

# 10. Relationship With Other Decisions

This ADR establishes the architectural foundation for subsequent decisions.

It precedes:

* **ADR-002 — Plugin SDK Boundary**
* **ADR-003 — Official Interface Architecture**
* **ADR-004 — Capability Registry Architecture**
* **ADR-005 — Execution Architecture**
* **ADR-008 — Marketplace & Registry Architecture**
* **ADR-009 — Workflow Architecture**

ADR-002 shall define how the Plugin-First architecture is enforced through the Plugin SDK.

ADR-003 shall define the Official Interface layer through which users interact with platform functionality.

ADR-004 shall define capability discovery and matching.

ADR-005 shall define execution behavior.

ADR-008 shall define Marketplace and Registry architecture.

ADR-009 shall define the resulting Workflow architecture.

---

# 11. Architectural Invariants

The following statements are architectural invariants established by this decision:

1. Plugins are the primary extension mechanism of PRIDE.
2. Applications interact with the platform through Plugins.
3. Applications shall not directly depend on internal Core components.
4. Plugins shall remain independent from individual Core implementations.
5. The Core shall remain independent from individual Plugin implementations.
6. Infrastructure shall remain below the Core.
7. New functionality should prefer Plugin extension when it does not represent a fundamental Core responsibility.
8. Architectural boundaries shall be maintained through explicit platform contracts.
9. Plugin implementation details shall not become Core architectural dependencies.
10. The Plugin ecosystem shall be capable of evolving independently from the Core.

---

# 12. Decision Outcome

PRIDE shall use a Plugin-First Architecture as the foundational extension model of the platform.

The Core provides the stable foundation.

The SDK provides the official contract boundary.

Plugins provide functional extensions.

Applications consume platform functionality through the Plugin layer.

This decision establishes the architectural foundation upon which the subsequent Plugin SDK, Official Interface, Capability Registry, Execution, Marketplace, and Workflow decisions are built.
