# ADR-0007 — User-Controlled Provider Selection

**Status:** Accepted

---

## Context

Artificial Intelligence providers continuously evolve.

Different providers offer different strengths in reasoning, coding, vision, image generation, speech, and other capabilities.

However,

allowing the platform to automatically choose Providers introduces unnecessary complexity, reduces predictability, and removes user control.

A simpler and more transparent strategy is therefore required.

---

## Decision

PRIDE adopts **User-Controlled Provider Selection**.

The active AI Provider is explicitly selected by the user through the User Interface.

The Core never automatically changes the selected Provider.

Every AI request is executed using the Provider currently selected by the user.

---

## Principles

Provider selection belongs to the user.

The Core never decides which Provider should be used.

Plugins never communicate directly with Providers.

All communication with Providers must occur through the AI SDK.

The AI SDK forwards requests to the currently selected Provider.

---

## Provider Responsibilities

Providers are responsible only for performing Artificial Intelligence inference.

Examples include:

* Reasoning
* Coding
* Vision
* Image Generation
* Speech Recognition
* Speech Synthesis
* Translation
* Summarization

Providers never participate in workflow orchestration.

Providers never communicate with Plugins directly.

---

## Provider Configuration

The active Provider is configured through the User Interface.

Changing the active Provider affects future AI requests only.

The workflow itself remains unchanged.

Provider configuration is therefore considered a user preference rather than an architectural decision.

---

## Execution

Whenever a Core component requires semantic reasoning,

it invokes the AI SDK.

Example.

```text
Planning

↓

sdk.ai.plan()

↓

Selected Provider

↓

Result

↓

Planning
```

The Core never communicates directly with the Provider.

Only the AI SDK knows how Provider communication is implemented.

---

## Rationale

Keeping Provider selection under user control produces a more predictable platform.

Users always know which Provider is being used.

The workflow remains deterministic.

The architecture remains independent from Provider implementations.

---

## Consequences

### Positive

* Complete user control
* Predictable behavior
* Simplified architecture
* Provider independence
* Easier maintenance
* Stable workflow

### Trade-offs

* Users are responsible for selecting the desired Provider.
* Automatic Provider optimization is intentionally omitted.
* Switching Providers requires user interaction.

These trade-offs are considered acceptable in exchange for transparency and architectural simplicity.

---

## Alternatives Considered

### Automatic Provider Selection

Allowing the platform to automatically determine the best Provider.

**Rejected.**

Automatic selection reduces transparency, increases architectural complexity, and removes explicit user control.

---

### Capability-Based Provider Selection

Selecting Providers dynamically according to requested capabilities.

**Rejected.**

Although flexible, this introduces additional orchestration complexity and conflicts with PRIDE's deterministic workflow philosophy.

Users should explicitly decide which Provider executes their requests.

---

## Related Foundation

* Human Authority
* Transparency
* Unlimited Evolution

---

## Related ADR

* ADR-0005 — Provider Abstraction Layer
* ADR-0006 — User-Approved Task Execution
* ADR-0010 — Extensible SDK Architecture

---

This decision establishes Provider selection for the PRIDE Core.

Users select the active Provider.

The AI SDK communicates with that Provider.

The Core remains completely independent from Provider implementations.

Provider implementations may evolve.

The workflow should not.
