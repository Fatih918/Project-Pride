# ADR-0001 — Plugin-First Architecture

**Status:** Supersede

---

## Context

PRIDE is intended to remain modular, maintainable, and adaptable over time.

As the platform evolves, there will be continuous pressure to introduce new capabilities into the Core.

While this may simplify short-term implementation, it increases long-term complexity, reduces maintainability, and limits extensibility.

The Foundation establishes that the Core should remain stable and minimal.

This decision defines how that principle is applied architecturally.

---

## Decision

PRIDE adopts a **Plugin-First Architecture**.

Any capability that can reasonably exist outside the Core should be implemented as a plugin.

The Core should provide infrastructure rather than user-facing functionality.

Examples of Core responsibilities include:

* Plugin lifecycle management
* Provider lifecycle management
* Event system
* Permission system
* Workspace management
* Settings infrastructure
* Storage abstraction
* Update infrastructure

User-facing capabilities should remain outside the Core whenever reasonably possible.

Examples include, but are not limited to:

* Planning
* Memory management
* Browser integration
* Research tools
* Calendar integration
* File management
* AI agents
* Automation workflows

These capabilities should be implemented as plugins.

---

## Rationale

Separating infrastructure from functionality provides several long-term benefits.

A smaller Core is easier to understand,

maintain,

test,

and evolve.

Independent plugins allow contributors to innovate without increasing the complexity of the platform itself.

This architecture also enables different users to build different workspaces without requiring changes to the Core.

---

## Consequences

### Positive

* Smaller and more maintainable Core
* Greater architectural stability
* Easier community contribution
* Independent plugin evolution
* Reduced long-term technical debt
* Greater flexibility for different workflows

### Trade-offs

* Plugin APIs must remain stable.
* Documentation becomes more important.
* Initial architecture requires additional planning.
* Plugin compatibility must be maintained responsibly.

These trade-offs are considered acceptable in exchange for long-term maintainability.

---

## Alternatives Considered

### Feature-First Core

Embedding user-facing functionality directly into the Core.

**Rejected.**

This approach increases coupling,

expands Core responsibilities,

and makes long-term maintenance more difficult.

---

### Hybrid Core

Including commonly used features inside the Core while implementing others as plugins.

**Rejected.**

Common usage does not necessarily justify inclusion in the Core.

Capabilities that can exist independently should remain independent whenever reasonably practical.

---

## Related Foundation

* Architecture Principles
* Plugin Principles
* Evolution
* Constitution (Stable Core)

---

This decision establishes the architectural direction of PRIDE.

Future proposals that introduce new Core functionality should demonstrate why the capability cannot reasonably exist as a plugin before being considered for inclusion.
