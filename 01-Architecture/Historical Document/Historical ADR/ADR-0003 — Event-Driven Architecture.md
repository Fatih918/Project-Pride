# ADR-0003 — Event-Driven Architecture

**Status:** Accepted

---

## Context

As the PRIDE ecosystem grows, plugins, providers, and platform components will increasingly need to exchange information.

Allowing components to communicate through direct dependencies creates tight coupling, making the platform harder to maintain, extend, and evolve.

A communication model is required that enables independent development while preserving architectural flexibility.

---

## Decision

PRIDE adopts an **Event-Driven Architecture**.

Components should communicate by publishing and subscribing to events whenever reasonably practical.

Direct dependencies between plugins should be avoided unless they are technically necessary.

The Core provides the Event Bus.

Plugins, providers, and other platform components interact through it.

---

## Principles

Events describe **what happened**, not **what should happen**.

Examples include:

* `task.created`
* `provider.changed`
* `conversation.completed`
* `workspace.opened`
* `plugin.enabled`
* `plugin.disabled`
* `settings.updated`

Components may choose to respond to an event.

They are never required to.

---

## Rationale

An event-driven architecture reduces coupling between components.

Plugins do not need knowledge of one another.

They only need knowledge of the events they publish or consume.

This allows independent development,

simpler maintenance,

greater extensibility,

and a healthier plugin ecosystem.

New plugins can introduce new behavior simply by subscribing to existing events.

Existing plugins remain unchanged.

---

## Consequences

### Positive

* Loose coupling between components
* Easier plugin development
* Better extensibility
* Greater architectural flexibility
* Improved maintainability
* Reduced dependency chains

### Trade-offs

* Event contracts require long-term stability.
* Debugging event flows may become more complex.
* Documentation of events becomes essential.
* Event naming must remain consistent.

These trade-offs are considered acceptable in exchange for a modular architecture.

---

## Alternatives Considered

### Direct Plugin Communication

Allowing plugins to call one another directly.

**Rejected.**

This creates unnecessary dependencies and increases architectural coupling.

---

### Central Feature Manager

Introducing a central coordinator responsible for user-facing functionality.

**Rejected.**

This shifts platform complexity into the Core and weakens the Plugin-First Architecture.

---

## Related Foundation

* Architecture Principles
* Plugin Principles
* Evolution

## Related ADR

* ADR-0001 — Plugin-First Architecture
* ADR-0002 — Core Infrastructure Principle

---

This decision establishes the standard communication model for PRIDE.

Future platform components should prefer publishing and subscribing to events rather than creating direct dependencies whenever reasonably practical.
