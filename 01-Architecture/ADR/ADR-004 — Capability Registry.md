# ADR-004 — Capability Registry

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-004

---

# 1. Context

After Planning has produced a Task Tree, PRIDE must determine which installed Plugin is capable of performing each task.

The Task Tree defines **what** must be accomplished, but does not define **who** should accomplish each task.

PRIDE therefore requires a dedicated Core component responsible for capability selection while remaining independent from Plugin implementations.

The Capability Registry exists to provide this architectural boundary.

---

# 2. Decision

PRIDE adopts the **Capability Registry** as the Core component responsible for capability selection.

The Capability Registry shall:

* Receive the Task Tree produced by Planning.
* Discover available Plugin capabilities through the Plugin Module.
* Match each executable task with compatible Plugin capabilities.
* Select the most appropriate Plugin for each task.
* Generate a standardized Execution Plan.
* Forward the completed Execution Plan to the Executor.

The Capability Registry shall not execute Plugins.

This establishes the following architecture:

```text
Task Tree
    │
    ▼
Capability Registry
    │
    ▼
Plugin Module
    │
    ▼
Installed Plugins
    │
    ▼
Capability Registry
    │
    ▼
Execution Plan
    │
    ▼
Executor
```

The Capability Registry therefore separates **Planning**, **capability selection**, and **execution** into independent responsibilities.

---

# 3. Core Ownership

The Capability Registry is part of the PRIDE Core.

Capability matching is a fundamental platform responsibility and shall not depend on an external Plugin.

The Registry itself shall remain independent from individual Plugin implementations.

Plugins may evolve or be replaced without requiring changes to the Capability Registry architecture.

---

# 4. Input Boundary

The Capability Registry receives exactly one primary input:

**The Task Tree produced by Planning.**

The Task Tree contains:

* Task identifiers.
* Task metadata.
* Task goals.
* Task dependencies.
* Required capabilities.

The Capability Registry shall not modify the Task Tree.

---

# 5. Plugin Discovery Boundary

The Capability Registry shall not communicate directly with individual Plugin implementations.

Plugin discovery occurs through the Plugin Module.

The Plugin Module provides the Registry with installed Plugins and their declared capabilities.

This preserves the independence between the Core and individual Plugin implementations.

---

# 6. Capability Matching

The Registry analyzes each task independently and identifies its required capabilities.

It then compares those requirements against the capabilities declared by available Plugins.

Only compatible Plugins become candidates for selection.

---

# 7. Plugin Selection

When multiple compatible Plugins exist, the Capability Registry selects the most appropriate Plugin according to the platform's selection strategy.

The selection strategy itself shall remain implementation-independent.

The Registry does not become dependent on the internal implementation of the selected Plugin.

---

# 8. Deterministic Capability Selection

Given the same Task Tree and identical installed Plugin set, the Capability Registry should produce the same Execution Plan.

This establishes deterministic capability selection as an architectural principle.

The implementation of the selection strategy may evolve, but the architectural responsibility and deterministic outcome requirement remain unchanged.

---

# 9. Execution Plan

The Capability Registry produces the standardized Execution Plan after all tasks have been matched.

The Execution Plan defines:

* Every executable task.
* The Plugin assigned to each task.
* Execution order.
* Task dependencies.

The Execution Plan contains no implementation-specific logic.

It describes which Plugin should perform each task rather than how the Plugin performs the task.

---

# 10. Execution Boundary

The Capability Registry does not execute work.

Its responsibility ends after generating the Execution Plan.

The Executor receives the completed Execution Plan and performs execution.

Therefore:

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
```

Each component owns one responsibility:

* Planning creates tasks.
* Capability Registry selects capabilities.
* Executor performs execution.
* Plugins implement functionality.

---

# 11. Non-Responsibilities

The Capability Registry is not responsible for:

* Planning.
* Plugin execution.
* AI reasoning.
* Memory management.
* SDK communication.
* Plugin installation.
* Plugin lifecycle management.
* Workflow execution.

Those responsibilities belong to other platform components and specifications.

---

# 12. Registry Lifecycle

The Capability Registry follows this execution lifecycle:

```text
Receive Task Tree
        ↓
Analyze Tasks
        ↓
Query Plugin Module
        ↓
Match Capabilities
        ↓
Select Plugins
        ↓
Generate Execution Plan
        ↓
Forward to Executor
```

The Registry completes its responsibility when the standardized Execution Plan has been produced and forwarded to the Executor.

---

# 13. Error Model

The Capability Registry may encounter errors including:

* Invalid Task Tree.
* Missing Capability.
* No Compatible Plugin.
* Plugin Unavailable.
* Plugin Disabled.
* Registry Failure.
* Unknown Error.

All failures shall use the standardized PRIDE platform error model.

---

# 14. Architectural Invariants

The following rules are established by this decision:

1. The Capability Registry is part of the Core.
2. Capability matching is a Core responsibility.
3. The Registry receives the Task Tree from Planning.
4. The Registry does not modify the Task Tree.
5. Plugin discovery occurs through the Plugin Module.
6. The Registry does not communicate directly with Plugin implementations.
7. Capability matching occurs before execution.
8. The Registry selects Plugins but does not execute them.
9. The Registry generates the Execution Plan.
10. The Execution Plan is passed to the Executor.
11. The Execution Plan contains no implementation-specific logic.
12. The Registry remains independent from individual Plugin implementations.
13. The same Task Tree and identical installed Plugins should produce the same Execution Plan.
14. New Plugins may be introduced without modifying the Registry architecture.

---

# 15. Consequences

## Positive Consequences

### Clear Responsibility Boundary

Planning, capability selection, and execution remain separate.

### Plugin Independence

Plugins can evolve or be replaced without requiring changes to the Core Registry architecture.

### Standardized Execution Planning

All executable tasks pass through one capability-selection stage before reaching the Executor.

### Core Stability

Capability matching remains a stable Core responsibility while the Plugin ecosystem evolves independently.

### Extensibility

New Plugins can be introduced without redesigning the Capability Registry architecture.

---

## Negative Consequences

### Registry Dependency

Execution planning depends on the Capability Registry successfully resolving capabilities.

### Plugin Metadata Dependency

Plugins must expose sufficient capability information for reliable matching.

### Additional Architectural Layer

Tasks cannot proceed directly from Planning to execution; they must pass through capability selection and Execution Plan generation.

---

# 16. Alternatives Considered

## 16.1 Direct Plugin Selection by Planning

Planning directly selects the Plugin responsible for each task.

**Rejected.**

This would merge Planning and capability selection responsibilities.

---

## 16.2 Direct Plugin Discovery

The Registry communicates directly with individual Plugin implementations.

**Rejected.**

The established architecture requires Plugin discovery through the Plugin Module to preserve implementation independence.

---

## 16.3 Registry-Owned Execution

The Capability Registry selects and executes Plugins itself.

**Rejected.**

Capability selection and execution are separate responsibilities. The Registry generates the Execution Plan, while the Executor performs execution.

---

# 17. Relationship With Other Decisions

This ADR follows:

* **ADR-000 — SDR Concept**
* **ADR-001 — Plugin-First Architecture**
* **ADR-002 — Plugin SDK Boundary**
* **ADR-003 — Official Interface**

This decision establishes the architectural foundation for:

* Execution Architecture.
* AI SDK Boundary.
* Workflow Architecture.
* Plugin execution.
* Future capability-related specifications.

---

# 18. Decision Outcome

PRIDE shall use the Capability Registry as the Core capability-selection boundary.

The Registry receives a Task Tree, discovers Plugin capabilities through the Plugin Module, matches tasks with compatible capabilities, selects the appropriate Plugins, generates a standardized Execution Plan, and forwards that plan to the Executor.

The Registry does not execute Plugins and does not depend on individual Plugin implementations.

This establishes capability selection as a distinct and stable architectural responsibility within PRIDE.
