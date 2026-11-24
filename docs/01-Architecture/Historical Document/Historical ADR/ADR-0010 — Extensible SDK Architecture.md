# ADR-0010 — Extensible SDK Architecture

**Status:** Accepted

---

## Context

PRIDE is designed as an extensible AI Operating Environment Platform.

While the platform provides a collection of official SDKs, it is impossible to predict every future capability, technology, integration, or Plugin category.

Attempting to design every possible SDK in advance would introduce unnecessary complexity while limiting future evolution.

An extensible SDK architecture is therefore required.

---

## Decision

PRIDE adopts an **Extensible SDK Architecture**.

Official SDKs provide stable abstractions for platform services.

New SDKs may be introduced whenever recurring architectural patterns emerge.

The SDK ecosystem is intentionally designed to evolve alongside the platform.

---

## Principles

The SDK ecosystem is intentionally open-ended.

The number of SDKs is not fixed.

Official SDKs should be introduced only when they solve recurring architectural problems shared across multiple components.

Platform evolution prioritizes maintainability over completeness.

---

## Official SDKs

Official SDKs provide stable interfaces for platform capabilities.

Examples may include:

* AI SDK
* Memory SDK
* Workspace SDK
* UI SDK
* Marketplace SDK
* System SDK
* Event SDK
* Permission SDK

These examples are not exhaustive.

The SDK ecosystem is expected to evolve over time.

---

## SDK Independence

Each SDK owns a single responsibility.

SDKs should remain independent from one another whenever reasonably practical.

SDKs communicate through stable interfaces rather than internal implementation details.

This separation minimizes coupling throughout the platform.

---

## Generic Development

When no specialized SDK exists,

contributors should build using the existing SDKs.

The absence of a dedicated SDK must never prevent Plugin development.

Plugins remain first-class citizens regardless of whether a specialized SDK currently exists.

---

## SDK Evolution

When multiple Plugins repeatedly implement similar architectural patterns,

a new official SDK may be proposed.

SDK creation should be driven by practical implementation experience rather than speculation.

Every new SDK should be introduced through the Architecture Decision Record (ADR) process.

---

## Community Innovation

Contributors are encouraged to experiment using existing SDKs.

Successful implementation patterns may later become official SDKs.

Innovation should originate from practical usage rather than theoretical design.

---

## Stability

Official SDKs should prioritize long-term compatibility.

Breaking changes should be minimized whenever reasonably practical.

New functionality should generally extend existing SDKs rather than replacing them.

Stable abstractions are considered more valuable than rapid architectural changes.

---

## Rationale

An extensible SDK architecture allows PRIDE to support technologies that do not yet exist while preserving a stable development experience.

The platform grows through proven architectural patterns instead of assumptions about the future.

This approach allows the SDK ecosystem to evolve naturally without introducing unnecessary complexity into the Core.

---

## Consequences

### Positive

* Long-term extensibility
* Stable abstractions
* Reduced architectural rigidity
* Encourages experimentation
* Community-driven evolution
* Lower Core complexity
* Sustainable platform growth

### Trade-offs

* Some Plugins may initially require additional implementation effort.
* Specialized SDKs may appear later than desired.
* Architectural review becomes increasingly important as the ecosystem grows.

These trade-offs are considered acceptable in exchange for flexibility, maintainability, and sustainable evolution.

---

## Alternatives Considered

### Complete SDK Design

Attempting to design every possible SDK before future requirements become known.

**Rejected.**

The future cannot be fully predicted.

Speculative SDK design increases unnecessary complexity.

---

### Fixed SDK Architecture

Restricting Plugin development to a predefined collection of SDKs.

**Rejected.**

This limits innovation and conflicts with PRIDE's commitment to Unlimited Evolution.

---

## Related Foundation

* Unlimited Evolution
* Stable Foundations
* Open Collaboration

---

## Related ADR

* ADR-0001 — Plugin-First Architecture
* ADR-0002 — Core Infrastructure Principle
* ADR-0009 — Workspace Architecture

---

This decision establishes an extensible SDK architecture for PRIDE.

Official SDKs provide stable architectural foundations.

Future SDKs emerge through practical implementation, community collaboration, and architectural review.

The SDK ecosystem should evolve as the platform evolves.
