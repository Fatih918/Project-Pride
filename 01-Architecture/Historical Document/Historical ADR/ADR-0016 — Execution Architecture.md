# ADR-0016 — Execution Architecture

**Status:** Accepted

---

# Context

PRIDE executes user-approved tasks across Plugins, SDKs, and platform services.

As the platform grows, execution must remain predictable, deterministic, and independent from implementation details.

Without a unified execution architecture, Plugins, Providers, and platform components would implement their own execution logic, resulting in duplicated orchestration, inconsistent behavior, and reduced interoperability.

A dedicated execution architecture is therefore required.

---

# Decision

PRIDE adopts a unified **Execution Architecture**.

Execution is performed by the **Executor**, a Core component responsible for carrying out previously approved Execution Plans.

The Executor never performs reasoning.

The Executor never creates plans.

The Executor executes approved plans deterministically.

---

# Decision Drivers

This decision is driven by the following architectural principles:

* Stable Foundations
* Human Authority
* Platform Consistency
* Plugin Interoperability
* Provider Independence
* Deterministic Execution
* Long-term Maintainability

---

# Architecture Principles

## Separation of Responsibilities

Execution is separated from reasoning.

Planning determines **what** should happen.

Execution determines **how** the approved plan is carried out.

---

## Executor

The Executor is responsible for:

* Reading Execution Plans
* Validating execution prerequisites
* Calling platform SDKs
* Coordinating execution steps
* Monitoring execution progress
* Reporting execution results

The Executor never modifies the approved plan.

---

## Deterministic Behavior

Given the same approved Execution Plan,

the Executor should produce the same sequence of operations whenever reasonably practical.

Execution should remain predictable.

---

## SDK Abstraction

The Executor never communicates directly with Plugins or AI Providers.

All operations occur through official platform SDKs.

Examples include:

* AI SDK
* Memory SDK
* Workspace SDK
* UI SDK
* Marketplace SDK
* Event SDK
* Permission SDK

Additional SDKs may be introduced without changing the Executor architecture.

---

## Plugin Independence

Plugins implement capabilities.

The Executor invokes those capabilities through SDK abstractions.

Plugins remain independent from execution orchestration.

---

## Provider Independence

AI Providers are never controlled directly by the Executor.

Provider selection occurs before execution through the AI SDK.

The Executor remains completely independent from Provider implementations.

---

## Human Authority

Execution begins only after the required user approvals have been completed.

The Executor must never bypass platform approval policies.

---

## Transparency

Execution progress should remain observable whenever reasonably practical.

Users should be able to understand:

* Current step
* Completed steps
* Failed steps
* Final execution results

Presentation details remain implementation-specific.

---

## Error Handling

When execution fails,

the Executor should:

* Stop the affected operation when appropriate
* Preserve completed work whenever possible
* Report failures clearly
* Avoid leaving the platform in an inconsistent state

Recovery strategies remain implementation-specific.

---

# Execution Flow

```text
User Request
      │
      ▼
Planning
      │
      ▼
Execution Plan
      │
      ▼
User Approval
      │
      ▼
Executor
      │
      ▼
Platform SDKs
      │
      ▼
Plugins / Platform Services
```

Planning decides.

The Executor executes.

SDKs abstract.

Plugins implement.

---

# Considered Alternatives

## Option A — Plugin-Owned Execution

Each Plugin performs its own workflow execution.

### Advantages

* Simple Plugin implementation
* Maximum flexibility

### Disadvantages

* Duplicate orchestration
* Inconsistent behavior
* Poor interoperability

**Decision:** Rejected.

---

## Option B — Provider-Owned Execution

AI Providers control execution directly.

### Advantages

* Simplified platform architecture

### Disadvantages

* Vendor lock-in
* Inconsistent execution behavior
* Violates Provider Abstraction

**Decision:** Rejected.

---

## Option C — Centralized Execution

A dedicated Executor executes approved plans through official SDKs.

### Advantages

* Deterministic execution
* Platform consistency
* Provider independence
* Plugin interoperability
* Stable architecture

### Disadvantages

* Additional Core responsibility
* Requires stable SDK contracts

**Decision:** Accepted.

---

# Consequences

## Positive

* Deterministic execution
* Consistent platform behavior
* Provider independence
* Plugin interoperability
* Simplified Plugin development
* Stable execution architecture

## Trade-offs

* The Executor becomes a Core responsibility.
* SDK contracts require long-term stability.
* Execution behavior becomes highly standardized.

These trade-offs are considered acceptable in exchange for consistency, predictability, and long-term maintainability.

---

# Related Foundation

* Stable Foundations
* Human Authority
* Unlimited Evolution

---

# Related ADR

* ADR-0006 — User-Approved Task Execution
* ADR-0010 — Extensible SDK Architecture
* ADR-0011 — Plugin Lifecycle

---

This decision establishes the Executor as the platform component responsible for deterministic task execution.

Planning determines what should happen.

The Executor carries out the approved plan.

SDKs provide stable abstractions.

Plugins implement platform capabilities.

This separation preserves consistency, transparency, and long-term architectural stability.
