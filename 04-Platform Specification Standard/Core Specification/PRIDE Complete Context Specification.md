# PRIDE Complete Context Specification

**Version:** 1.0.0 Stable

**Status:** Stable

**Category:** Core Specification

**Document ID:** PRIDE-COMPLETE-CONTEXT

**Author:** PRIDE Project

---

# 1. Introduction

Before Planning can determine what should be executed, the platform must first understand the user's situation.

A user request alone rarely contains sufficient information for intelligent decision making.

Relevant memories, user preferences, historical interactions, and platform knowledge may all contribute to understanding the user's actual objective.

The Complete Context represents this unified understanding.

Rather than allowing Planning to gather information from multiple platform components, the PRIDE Platform consolidates all relevant information into one standardized context object.

Planning always receives exactly one Complete Context.

---

# 2. Purpose

The Complete Context exists to provide a unified information object for the Planning component.

Its objectives are:

* Combine the user request with relevant memory.
* Present one complete understanding of the current situation.
* Eliminate direct communication between Planning and Memory.
* Standardize contextual information across the platform.
* Preserve implementation independence.
* Support future context sources.

---

# 3. Scope

This specification defines:

* Complete Context architecture.
* Complete Context composition.
* Context ownership.
* Context responsibilities.
* Context lifecycle.
* Context conformance.

This specification does not define:

* Memory Retrieval.
* Context Builder implementation.
* Planning algorithms.
* Artificial Intelligence reasoning.
* Memory storage.
* Workflow execution.

Those concerns are defined by their respective specifications.

---

# 4. Complete Context Definition

The Complete Context is the standardized information object produced by the Context Builder.

It represents the complete understanding of the current workflow before Planning begins.

The Complete Context is not responsible for reasoning.

It is not responsible for planning.

It does not perform execution.

Instead, it provides every piece of contextual information required by Planning to construct an appropriate Task Tree.

Planning shall always receive exactly one Complete Context.

---

# 5. Design Philosophy

Understanding should occur before decision making.

Rather than allowing every Core component to collect its own contextual information independently, PRIDE centralizes contextual information into one standardized object.

This simplifies workflow communication while preserving architectural separation.

Planning consumes context.

It never constructs context.

Context Builder constructs context.

It never performs planning.

This separation preserves deterministic workflow behavior and long-term maintainability.
# 6. Design Principles

## 6.1 Single Context Source

Planning shall receive contextual information exclusively through the Complete Context.

Planning shall never retrieve information directly from Memory, AI, Plugins, or any other platform component.

---

## 6.2 Unified Representation

All contextual information shall be represented as one unified object.

Regardless of how many internal components contribute information, the Planning component always receives one Complete Context.

---

## 6.3 Immutable During Planning

Once the Complete Context has been constructed, it shall remain unchanged throughout the Planning stage.

If new information becomes available after Planning has started, a new Complete Context shall be constructed for a subsequent workflow rather than modifying the existing one.

---

## 6.4 Implementation Independence

The Complete Context remains independent from:

* Memory implementations.
* AI providers.
* Plugin technologies.
* SDK implementations.
* Storage technologies.

Its structure defines logical information rather than implementation-specific behavior.

---

## 6.5 Extensibility

The Complete Context is designed to support future context sources.

New platform components may contribute contextual information without requiring changes to the Planning architecture.

---

# 7. Context Architecture

The Complete Context is constructed by combining multiple sources of information into one standardized object.

```text id="completecontext01"
User Request

        +

Memory Retrieval

        ↓

Context Builder

        ↓

Complete Context

        ↓

Planning
```

The Context Builder is solely responsible for creating the Complete Context.

Planning never constructs or modifies contextual information.

---

# 8. Context Components

A Complete Context consists of multiple logical information components.

## 8.1 User Request

The original request submitted by the user.

This represents the primary objective of the workflow.

The original request shall always be preserved.

---

## 8.2 Retrieved Memory

Relevant information retrieved during the Memory Retrieval stage.

Retrieved Memory may include:

* Previous conversations.
* User preferences.
* Stored knowledge.
* Historical workflow data.
* Other memory relevant to the current objective.

Memory Retrieval determines relevance.

The Complete Context only stores the retrieved result.

---

## 8.3 Context Metadata

Metadata describing the contextual environment.

Examples include:

* Context Identifier.
* Creation Timestamp.
* Workflow Identifier.
* Context Version.

Metadata exists to support workflow consistency.

---

## 8.4 Platform Constraints

Platform-level information that may influence planning.

Examples include:

* Current platform limitations.
* Permission restrictions.
* Execution environment.
* System configuration.

The exact contents remain platform-defined.

---

# 9. Responsibilities

The Complete Context is responsible for:

* Representing the complete understanding of the current workflow.
* Combining user input with retrieved memory.
* Providing one standardized input to Planning.
* Remaining implementation-independent.
* Preserving contextual consistency throughout Planning.

---

# 10. Non-Responsibilities

The Complete Context is not responsible for:

* Retrieving memory.
* Building itself.
* Planning tasks.
* Selecting capabilities.
* Executing workflows.
* Updating memory.
* Artificial Intelligence reasoning.

These responsibilities belong to other Core components.

---

# 11. Lifecycle

The Complete Context follows a simple lifecycle.

```text id="completecontext02"
User Request

↓

Memory Retrieval

↓

Context Builder

↓

Complete Context Created

↓

Planning

↓

Context Discarded
```

The Complete Context exists only for the duration of the current workflow.

It is reconstructed for every new workflow.

---

# 12. Conformance

A Complete Context is considered PRIDE compliant only if it:

* Contains the original user request.
* Contains the retrieved memory provided by Memory Retrieval.
* Is constructed exclusively by the Context Builder.
* Is provided as the only input to Planning.
* Remains immutable during Planning.
* Preserves implementation independence.

---

# 13. Related Specifications

This specification complements:

* Context Builder Specification
* Memory Retrieval Specification
* Planning Specification
* Workflow Specification
* Platform Specification Standard (PSS)

Each specification defines one independent architectural concern.

---

# 14. Future Evolution

Future versions of the Complete Context may introduce additional contextual components.

Possible future additions include:

* Active workflow state.
* Environmental context.
* User session context.
* Organization context.
* Shared collaboration context.

Such additions shall preserve backward compatibility while maintaining the standardized Complete Context contract.

---

# 15. Conclusion

The Complete Context defines the standardized information object used by the Planning component.

Rather than allowing Planning to collect information from multiple platform components, PRIDE consolidates relevant contextual information into one unified object through the Context Builder.

This separation preserves architectural consistency, implementation independence, deterministic planning behavior, and long-term platform maintainability while allowing future context sources to be integrated without modifying the Planning architecture.
