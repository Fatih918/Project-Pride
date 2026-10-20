# ADR-009 — Stable Core Workflow

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-009

---

# 1. Context

PRIDE Core requires one stable execution lifecycle for processing user requests.

The platform consists of multiple architectural components with different responsibilities, including:

* User Interface.
* Memory Retrieval.
* Context Builder.
* Planning.
* Capability Registry.
* Permission Manager.
* Executor.
* Plugin.
* Memory Update.

Without a fixed execution order, individual components could evolve in ways that introduce inconsistent execution paths, skipped responsibilities, or uncontrolled dependencies.

PRIDE therefore requires a permanent workflow structure while allowing the internal implementation of each component to evolve.

The existing PRIDE workflow establishes that the execution order of the Core does not change and that no component may be skipped or executed out of order.

---

# 2. Decision

PRIDE adopts a **Stable Core Workflow**.

Every standard execution shall follow one defined architectural sequence:

```text
User
   │
   ▼
UI
   │
   ▼
Memory Retrieval
   │
   ▼
Context Builder
   │
   ▼
Planning
   │
   ▼
Capability Registry
   │
   ▼
Permission Manager
   │
   ▼
Executor
   │
   ▼
Plugin
   │
   ▼
Memory Update
   │
   ▼
UI
   │
   ▼
User
```

This ordering is an architectural invariant of the PRIDE Core.

---

# 3. Workflow Invariant

The workflow itself shall remain stable.

Every standard execution follows the same architectural lifecycle.

No component may:

* Be skipped.
* Be moved to another position.
* Execute before its assigned stage.
* Execute after its assigned stage.
* Replace another component's architectural responsibility.

The workflow therefore represents the stable orchestration structure of the PRIDE Core.

---

# 4. Implementation Independence

The workflow is stable, but the implementation of individual components may evolve.

Programming languages may change.

Internal algorithms may change.

Providers may change.

Plugins may change.

Memory implementations may change.

Component implementations may therefore be replaced or improved without changing the fundamental workflow order.

The workflow provides the stable architectural foundation while implementations remain replaceable.

---

# 5. Execution Lifecycle

The workflow is divided into four major phases.

## Phase 1 — Understanding

The Core constructs the context required to understand the user's request.

Components:

* Memory Retrieval.
* Context Builder.

Objective:

Construct the execution context required for the request.

---

## Phase 2 — Decision Making

The Core determines how the request should be executed.

Components:

* Planning.
* Capability Registry.
* Permission Manager.

Objective:

Generate the required Task Tree, determine the appropriate capability and Plugin, and validate execution permissions.

---

## Phase 3 — Execution

The approved work is performed.

Components:

* Executor.
* Plugin.

Objective:

Execute the approved Task according to the established execution architecture.

---

## Phase 4 — Learning

The completed execution is processed for future memory.

Component:

* Memory Update.

Objective:

Organize useful execution results into structured long-term memory before the final response is presented.

---

# 6. Component Responsibilities

The workflow establishes the following high-level responsibilities.

```text
UI
    │
    └── User interaction

Memory Retrieval
    │
    └── Retrieve relevant memory

Context Builder
    │
    └── Construct execution context

Planning
    │
    └── Determine execution plan

Capability Registry
    │
    └── Resolve available capabilities

Permission Manager
    │
    └── Validate execution permissions

Executor
    │
    └── Execute approved tasks

Plugin
    │
    └── Perform the requested work

Memory Update
    │
    └── Organize useful execution results
```

Each component remains within its assigned architectural responsibility.

---

# 7. Memory Boundary

Memory-related operations remain governed by the Memory Architecture.

The workflow does not make the workflow itself a memory implementation.

Memory Retrieval and Memory Update are workflow stages that use the established Memory architecture and Memory SDK boundaries.

The workflow therefore determines **when** memory operations participate in execution, while the Memory Architecture determines **how memory operations are communicated and executed**.

This preserves the separation established by ADR-006.

---

# 8. AI Boundary

Artificial Intelligence may participate in workflow components where semantic reasoning is required.

The workflow does not make the AI SDK the owner of memory or workflow execution.

The AI SDK remains governed by ADR-007.

AI-related processing may provide metadata or reasoning support required by the relevant workflow component, while memory operations remain governed by the Memory Architecture.

The workflow therefore orchestrates AI-related components without replacing their established architectural boundaries.

---

# 9. Capability Registry Boundary

The Capability Registry participates during the Decision Making phase.

Its responsibility is to resolve the capabilities required for the planned execution.

The workflow does not define individual capabilities.

Capability definitions and Registry behavior remain governed by the Capability Registry architecture.

The workflow only establishes the Registry's position within the execution lifecycle.

---

# 10. Plugin Boundary

Plugins participate during the Execution phase.

The Executor invokes the appropriate Plugin after the planning, capability, and permission stages have completed.

The Plugin performs the requested work while remaining independent from the internal implementation of the Core.

Plugin communication remains governed by the Plugin SDK architecture established by earlier ADRs.

---

# 11. Forward-Only Execution

The standard workflow is forward-only.

Execution progresses from:

```text
User
  ↓
Understanding
  ↓
Decision Making
  ↓
Execution
  ↓
Learning
  ↓
User
```

A component shall not arbitrarily return execution to an earlier workflow stage.

Any internal processing required by a component remains an implementation concern and shall not alter the canonical workflow ordering.

Forward-only execution is an architectural characteristic of the PRIDE Core.

---

# 12. Architectural Invariants

The following rules are established as architectural invariants:

1. PRIDE Core has one canonical execution workflow.
2. The workflow order is stable.
3. Every standard execution follows the canonical workflow.
4. No workflow component may be skipped.
5. No workflow component may execute out of order.
6. Each component remains within its assigned responsibility.
7. The workflow is forward-only.
8. The workflow structure is independent from component implementation details.
9. Component implementations may evolve without changing the canonical workflow.
10. Memory operations remain governed by the Memory Architecture.
11. AI operations remain governed by the AI SDK architecture.
12. Capability resolution remains governed by the Capability Registry architecture.
13. Plugin execution remains governed by the Plugin architecture.
14. Workflow architecture does not replace the responsibilities of individual platform subsystems.
15. The canonical workflow remains the architectural foundation for future PRIDE implementations.

---

# 13. Alternatives Considered

## 13.1 Dynamic Component Ordering

Components may execute in different orders depending on the request.

**Rejected.**

A variable workflow would weaken architectural predictability and make subsystem responsibilities harder to maintain.

---

## 13.2 Optional Workflow Stages

Components may be skipped when they appear unnecessary.

**Rejected.**

The PRIDE Core defines one complete lifecycle and explicitly establishes that no component may be skipped.

---

## 13.3 Component-Defined Workflow

Individual components determine which components execute next.

**Rejected.**

This would distribute workflow authority across the platform and weaken the unified Core architecture.

---

## 13.4 Implementation-Defined Workflow

Each implementation may define its own execution lifecycle.

**Rejected.**

Implementations are allowed to evolve, but the architectural workflow must remain stable.

---

# 14. Consequences

## Positive Consequences

### Predictable Execution

Every request follows one recognizable lifecycle.

### Architectural Stability

Components can evolve without changing the Core's fundamental orchestration model.

### Clear Responsibility

Every workflow stage has a defined architectural role.

### Easier Maintenance

Changes to internal implementations do not automatically change the workflow.

### Reduced Coupling

Subsystems remain governed by their own SDK and architectural boundaries.

### Long-Term Evolution

The platform can replace implementations while preserving the same execution architecture.

---

## Negative Consequences

### Workflow Rigidity

Components must conform to the canonical execution order.

### Architectural Governance

Changes to the workflow require architectural review because the workflow is an established Core invariant.

### Potential Internal Complexity

A component may require sophisticated internal behavior while still conforming to its fixed workflow position.

---

# 15. Relationship With Other Decisions

This ADR follows:

* **ADR-000 — Software Design Record Concept**
* **ADR-001 — Plugin-First Architecture**
* **ADR-002 — Plugin SDK Boundary**
* **ADR-003 — Official Interface**
* **ADR-004 — Capability Registry**
* **ADR-005 — Execution Architecture**
* **ADR-006 — Unified Memory Architecture**
* **ADR-007 — AI SDK Boundary**
* **ADR-008 — Marketplace + Registry Architecture**

This ADR establishes the canonical execution order that connects those architectural decisions during runtime.

The Workflow does not replace any previous architectural boundary.

---

# 16. Decision Outcome

PRIDE shall use one stable Core Workflow:

```text
User
 ↓
UI
 ↓
Memory Retrieval
 ↓
Context Builder
 ↓
Planning
 ↓
Capability Registry
 ↓
Permission Manager
 ↓
Executor
 ↓
Plugin
 ↓
Memory Update
 ↓
UI
 ↓
User
```

The workflow structure shall remain stable.

The implementations behind the workflow may evolve independently.

The workflow therefore becomes the permanent orchestration foundation of the PRIDE Core while allowing the individual components, technologies, providers, algorithms, Plugins, and Memory Implementations to evolve over time.
