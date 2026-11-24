# ADR-005 — Execution Architecture

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-005

---

# 1. Context

PRIDE separates workflow planning, capability selection, and execution into independent architectural responsibilities.

Planning determines what work must be accomplished and produces a Task Tree.

The Capability Registry determines which standardized capability should perform each task and transforms the Task Tree into an Execution Plan.

The Executor performs the execution defined by the Execution Plan.

## This separation is required to preserve deterministic execution, implementation independence, and a stable execution contract between capability selection and execution.

# 2. Decision

PRIDE adopts a **separated Execution Architecture** in which workflow execution is performed through the following architectural stages:

```text
Planning

↓

Task Tree

↓

Capability Registry

↓

Execution Plan

↓

Executor

↓

Assigned Capability

↓

SDK

↓

Implementation
```

Each stage has a distinct responsibility.

Planning creates the logical work structure.

Capability Registry assigns capabilities.

Execution Plan represents the standardized executable workflow.

Executor performs the execution.

## Plugins and other implementations provide the underlying functionality through standardized platform interfaces.

# 3. Task Tree Boundary

The Task Tree represents the logical work produced by Planning.

The Task Tree does not contain execution-specific information.

It describes what work must be accomplished without determining how that work will be executed.

Planning shall not produce the Execution Plan.

The Task Tree must first pass through the Capability Registry before execution can begin.

---

# 4. Execution Plan Boundary

The Execution Plan is the standardized executable workflow produced by the Capability Registry.

It extends the Task Tree with the information required for execution without modifying its logical structure.

The Execution Plan contains execution information including:

* Task Identifier.
* Task Description.
* Assigned Capability.
* Dependency Information.
* Context Reference.
* Initial Execution State.

The Execution Plan is the direct input of the Executor.
The Execution Plan is the official execution contract between the Capability Registry and the Executor.

The Executor shall execute the plan exactly as provided.

---

# 5. Executor Boundary

The Executor is the Core component responsible for executing the Execution Plan.

The Executor does not determine what should happen.

The Executor does not select capabilities.

The Executor does not perform AI reasoning.

The Executor performs the execution strategy already defined by previous workflow stages.

The Executor therefore operates as a deterministic execution engine.

---

# 6. Deterministic Execution

The Executor shall execute tasks exactly as defined by the Execution Plan.

Execution shall preserve:

* Task dependencies.
* Execution order.
* Capability assignments.
* Execution metadata.
* Standardized execution behavior.

## The Executor shall not modify the execution strategy during execution.

# 7. Execution Lifecycle

Each execution follows a standardized lifecycle:

```text
Load Execution Plan

↓

Validate Dependencies

↓

Execute Task

↓

Receive Result

↓

Update Task State

↓

Continue Execution

↓

Workflow Complete / Terminated
```

The Executor verifies task dependencies before executing an eligible task and continues until the workflow completes or terminates.

---

# 8. Task State Model

Tasks expose standardized execution states.

The standard states are:

* Pending
* Running
* Completed
* Failed
* Skipped
* Cancelled

Implementation-specific execution states shall not be exposed outside the Executor.

---

# 9. Execution Results

The Executor produces standardized execution results.

Execution results include:

* Final task state.
* Execution outputs.
* Execution errors.
* Workflow completion status.

The Executor shall not modify task definitions during execution.

---

# 10. Implementation Independence

The Execution Architecture shall remain independent from specific implementations.

The Execution Plan shall not reference:

* Plugin implementations.
* SDK implementations.
* Providers.
* Storage technologies.
* Runtime environments.

Only standardized platform capabilities may appear in the Execution Plan.

This allows Plugins, AI providers, Memory implementations, and other implementations to evolve without changing the execution architecture.

---

# 11. Responsibility Separation

PRIDE establishes the following responsibility model:

```text
Planning
    │
    └── Determines what should happen

Capability Registry
    │
    └── Determines which capability performs each task

Execution Plan
    │
    └── Defines the executable workflow

Executor
    │
    └── Performs execution

Plugins / Implementations
    │
    └── Provide functionality
```

Each architectural component shall remain within its defined responsibility.

---

# 12. Architectural Invariants

The following rules are established:

1. Planning produces the Task Tree.
2. Planning does not produce the Execution Plan.
3. The Capability Registry generates the Execution Plan.
4. The Execution Plan preserves the logical structure of the Task Tree.
5. Every executable task references an assigned capability.
6. The Execution Plan is the direct input of the Executor.
7. The Execution Plan is the execution contract between the Capability Registry and Executor.
8. The Executor executes the Execution Plan exactly as provided.
9. The Executor does not perform capability selection.
10. The Executor does not perform AI reasoning.
11. The Executor respects task dependencies and execution order.
12. Implementation-specific behavior remains hidden behind standardized interfaces.
13. Execution results use standardized execution states and outcomes.

---

# 13. Consequences

## Positive Consequences

### Deterministic Execution

Execution behavior is defined before execution begins and is performed according to the Execution Plan.

### Clear Responsibility Boundaries

Planning, capability selection, and execution remain independent.

### Implementation Independence

Plugins and underlying implementations can evolve without changing the execution architecture.

### Stable Execution Contract

The Execution Plan provides a stable contract between capability selection and execution.

### Maintainability

Execution behavior remains centralized within the Executor instead of being distributed across platform components.

---

## Negative Consequences

### Architectural Layering

A request must pass through multiple stages before execution can occur.

### Contract Dependencies

The Capability Registry and Executor must maintain compatibility with the Execution Plan contract.

### Execution Governance

Changes to execution behavior must preserve the established boundaries between Planning, Capability Registry, Execution Plan, and Executor.

---

# 14. Alternatives Considered

## 14.1 Planning Directly Executes Tasks

**Rejected.**

Planning is responsible for determining what should happen, not performing execution.

---

## 14.2 Capability Registry Directly Executes Plugins

**Rejected.**

The Capability Registry is responsible for capability selection and Execution Plan generation. Execution belongs to the Executor.

---

## 14.3 Executor Performs Capability Selection

**Rejected.**

Capability selection belongs exclusively to the Capability Registry.

The Executor consumes the completed Execution Plan rather than determining which capability should perform each task.

---

## 14.4 Execution Plan Depends on Plugin Implementations

**Rejected.**

The Execution Plan must remain implementation-independent and reference standardized capabilities rather than specific implementations.

---

# 15. Relationship With Other Decisions

This ADR establishes the execution architecture that follows the architectural boundaries established by previous PRIDE decisions.

It directly defines the relationship between:

* Task Tree.
* Capability Registry.
* Execution Plan.
* Executor.
* Standardized Capabilities.
* SDKs.
* Implementations.

Future decisions concerning Memory, AI, Marketplace, and Workflow shall integrate with this execution architecture without bypassing its established boundaries.

---

# 16. Decision Outcome

PRIDE shall use a separated execution architecture in which:

```text
Planning
    ↓
Task Tree
    ↓
Capability Registry
    ↓
Execution Plan
    ↓
Executor
    ↓
Capability
    ↓
SDK
    ↓
Implementation
```

Planning determines the logical work.

Capability Registry determines the capabilities required for execution.

Execution Plan defines the executable workflow.

Executor performs the workflow.

This architecture establishes a deterministic, implementation-independent execution model while preserving clear separation of responsibility across the PRIDE platform.
