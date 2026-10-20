# ADR-0017 — Execution Context Architecture

**Status:** Supersede

---

# Context

PRIDE executes intelligent tasks by combining information from multiple platform subsystems before invoking AI reasoning.

Providing raw platform data directly to AI Providers would expose unnecessary information, increase implementation complexity, and reduce consistency across the ecosystem.

A unified mechanism is therefore required to prepare only the information relevant to the current task.

---

# Decision

PRIDE adopts a unified **Execution Context Architecture**.

Execution Context is a temporary, task-specific collection of information prepared by the Planning subsystem.

Execution Context is consumed through the AI SDK.

AI Providers never construct platform context independently.

---

# Decision Drivers

This decision is driven by the following architectural principles:

* Human-Centered Computing
* Stable Foundations
* Provider Independence
* Plugin Interoperability
* Local-First Computing
* Privacy by Default
* Transparency
* Long-term Maintainability

---

# Architecture Principles

## Execution Context

Execution Context represents only the information required to perform the current task.

It is temporary.

It exists only for the duration of a task.

---

## Planning Responsibility

Planning is responsible for constructing the Execution Context.

Planning determines which platform information is relevant.

Other platform components provide information.

Planning decides what should be included.

---

## Context Sources

Execution Context may be composed from multiple platform subsystems, including:

* Workspace
* Memory Retrieval
* System Profile
* Current Session
* Current Task
* User Preferences
* Permission State
* Active Platform State

Additional context sources may be introduced without changing the architecture.

---

## AI SDK

Execution Context is delivered through the AI SDK.

The AI SDK remains responsible for communicating with AI Providers.

Execution Context is independent from Provider implementations.

---

## Provider Independence

AI Providers consume Execution Context.

Providers never gather platform information independently.

Providers remain completely replaceable without affecting platform behavior.

---

## Plugin Independence

Plugins contribute platform data only through official SDKs.

Plugins never construct or modify Execution Context directly.

---

## Relevance

Execution Context should contain only information relevant to the current task.

Unnecessary information should not be included.

This reduces complexity while improving privacy and reasoning quality.

---

## Transparency

Users should be able to understand what information is being shared during task execution whenever reasonably practical.

Execution Context should remain observable rather than hidden.

---

## Privacy

Execution Context follows the platform Permission System.

Protected information must never become part of the Execution Context without explicit authorization.

---

# Architectural Relationship

```text
Workspace
Memory Retrieval
System Profile
Permission State
Current Task
Current Session
        │
        ▼
     Planning
        │
        ▼
 Execution Context
        │
        ▼
      AI SDK
        │
        ▼
    AI Provider
```

Platform subsystems provide information.

Planning constructs the Execution Context.

The AI SDK delivers it.

AI Providers consume it.

---

# Considered Alternatives

## Option A — Provider-Owned Context

Each AI Provider gathers its own contextual information.

### Advantages

* Simpler Provider implementation

### Disadvantages

* Vendor lock-in
* Inconsistent platform behavior
* Duplicate implementations
* Violates Provider Abstraction

**Decision:** Rejected.

---

## Option B — Plugin-Owned Context

Each Plugin prepares contextual information independently.

### Advantages

* Flexible Plugin implementation

### Disadvantages

* Fragmented ecosystem
* Duplicate context generation
* Poor interoperability

**Decision:** Rejected.

---

## Option C — Platform Execution Context

Planning prepares a unified Execution Context before AI reasoning begins.

### Advantages

* Consistent platform behavior
* Provider independence
* Better interoperability
* Improved privacy
* Stable architecture

### Disadvantages

* Additional Planning responsibility
* Requires standardized context structures

**Decision:** Accepted.

---

# Consequences

## Positive

* Consistent AI behavior
* Provider independence
* Plugin interoperability
* Better reasoning quality
* Improved privacy
* Stable execution architecture
* Supports future platform evolution

## Trade-offs

* Planning becomes responsible for context preparation.
* Context structures require long-term compatibility.
* Multiple platform subsystems cooperate to construct the Execution Context.

These trade-offs are considered acceptable in exchange for consistency, transparency, and long-term maintainability.

---

# Related Foundation

* Human First
* Stable Foundations
* Privacy
* User Ownership

---

# Related ADR

* ADR-0005 — Provider Abstraction Layer
* ADR-0008 — System Profile Service
* ADR-0014 — Memory Subsystem Architecture
* ADR-0015 — Adaptive Behavior Architecture
* ADR-0016 — Execution Architecture

---

This decision establishes Execution Context as the standardized information package prepared by Planning before AI reasoning begins.

Platform subsystems provide information.

Planning selects relevant information.

The AI SDK delivers the Execution Context.

AI Providers consume it without requiring knowledge of the platform's internal architecture.
