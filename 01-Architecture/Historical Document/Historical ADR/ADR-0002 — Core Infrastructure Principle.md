# ADR-0002 — Core Infrastructure Principle

**Status:** Supersede

---

## Context

The Plugin-First Architecture established by ADR-0001 defines that user-facing capabilities should exist as plugins whenever reasonably possible.

As PRIDE evolves, there will be continuous pressure to introduce new functionality directly into the Core.

While this may provide short-term convenience, it expands the responsibilities of the Core and weakens the long-term modular architecture of the platform.

A clear definition of the Core's purpose is therefore required.

---

## Decision

The Core exists to provide infrastructure for the platform.

It does not exist to provide functionality for users.

The relationship between the platform components is intentionally simple.

> **The Core serves plugins. Plugins serve users.**

The Core should evolve only to strengthen the platform itself.

It should not evolve by absorbing user-facing capabilities.

---

## Core Responsibilities

The Core may evolve to improve platform infrastructure, including but not limited to:

* Plugin lifecycle
* Provider lifecycle
* Event system
* Permission system
* Workspace management
* Storage abstraction
* Settings infrastructure
* Update infrastructure
* Security
* Performance
* Stability
* Compatibility
* Developer APIs

These responsibilities exist to enable the platform,

not to expand its functionality.

---

## Non-Core Responsibilities

The following capabilities should remain outside the Core whenever reasonably possible:

* Planning
* Memory
* Browser integration
* Calendar integration
* Research tools
* File management
* AI agents
* Automation
* Productivity workflows
* User interface enhancements
* AI providers

These capabilities serve users directly and should therefore be implemented as plugins or external modules.

---

## Rationale

Separating infrastructure from functionality creates a healthier architecture.

A focused Core is easier to understand,

maintain,

secure,

test,

and evolve.

Independent plugins allow innovation without increasing the complexity of the platform itself.

This separation also enables different users to build different experiences without modifying the Core.

---

## Consequences

### Positive

* Stable Core responsibilities
* Smaller long-term maintenance burden
* Clear architectural boundaries
* Greater plugin flexibility
* Better scalability of the ecosystem
* Reduced architectural coupling

### Trade-offs

* Infrastructure APIs require careful long-term maintenance.
* Plugin SDK quality becomes critical.
* More functionality depends on external modules.
* Strong documentation becomes increasingly important.

These trade-offs are considered acceptable in exchange for long-term architectural stability.

---

## Alternatives Considered

### Expanding Core Responsibilities

Adding user-facing functionality directly into the Core.

**Rejected.**

This approach increases complexity,

expands maintenance responsibilities,

and reduces architectural flexibility.

---

### Mixed Responsibility Core

Allowing the Core to gradually absorb commonly used features.

**Rejected.**

Popularity alone is not sufficient justification for inclusion in the Core.

Infrastructure and functionality should remain separate whenever reasonably practical.

---

## Related Foundation

* Architecture Principles
* Plugin Principles
* Stable Core
* Evolution

---

This decision defines the purpose of the Core.

Future proposals that modify the Core should demonstrate that the change strengthens platform infrastructure rather than introducing user-facing functionality.
