# ADR-006 — Unified Memory Architecture

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-006

---

# 1. Context

Memory is a platform-level capability within PRIDE.

PRIDE requires memory to remain independent from the technology used to persist it. Memory functionality must therefore be standardized at the platform level while allowing the underlying persistence technology to evolve independently.

The architecture defined in the platform documentation separates:

* Memory communication.
* Memory capabilities.
* Memory execution.
* Memory implementation.
* Memory storage.

The Memory Capability Specification establishes that a Memory Capability defines what a memory operation accomplishes, while the Memory Module executes the request using the available implementation.

The Memory Module Specification further establishes the Memory Module as the execution domain for standardized requests received through the Memory SDK and separates that execution from storage-specific behavior.

A unified architectural decision is therefore required to establish the permanent boundary between these memory layers.

---

# 2. Decision

PRIDE adopts a **Unified Memory Architecture**.

All platform memory operations shall use the Memory SDK as their communication boundary.

The Memory SDK forwards standardized Memory Capability requests to the Memory Module.

The Memory Module executes those requests through the active Memory Implementation.

The Memory Implementation translates the standardized operation into storage-specific operations performed by Memory Storage.

The resulting architecture is:

```text
Platform Component
        │
        ▼
   Memory SDK
        │
        ▼
  Memory Module
        │
        ▼
Active Memory Implementation
        │
        ▼
  Memory Storage
```

The reverse path returns the standardized execution result through the same architectural boundary.

---

# 3. Memory SDK Boundary

The Memory SDK is the standardized communication boundary for memory operations.

Platform components shall request memory operations through the Memory SDK rather than communicating directly with memory implementations or storage technologies.

The Memory SDK is responsible for forwarding standardized memory requests.

It does not determine how those requests are physically executed.

---

# 4. Memory Capability Boundary

Memory Capabilities define the logical functions that memory must provide.

A capability describes **what memory is expected to accomplish**, not how the operation is implemented.

Capabilities remain independent from:

* Databases.
* File systems.
* Vector databases.
* Graph databases.
* Cloud storage.
* Other persistence technologies.

Executable tasks request standardized capabilities while the underlying implementation remains hidden.

---

# 5. Memory Module Boundary

The Memory Module is the execution domain of the Memory SDK.

It receives standardized memory requests and delegates their execution to the active Memory Implementation.

The Memory Module shall not define:

* Storage technologies.
* Storage engine internals.
* Database-specific behavior.
* Memory capability definitions.
* Workflow execution.
* AI reasoning.
* User Interface behavior.

Its responsibility is to provide the unified execution boundary between the Memory SDK and Memory Implementations.

---

# 6. Memory Implementation Independence

Memory Implementations provide the technical layer that translates standardized Memory requests into storage-specific operations.

An implementation may use different persistence technologies without changing the Memory SDK contract.

Examples include:

* SQLite.
* PostgreSQL.
* MySQL.
* Local File Storage.
* Vector Database.
* Graph Database.
* Cloud Storage.
* Future Storage Technologies.

The internal behavior of each implementation remains implementation-defined.

Platform components shall remain unaware of which implementation is active.

---

# 7. Active Memory Implementation

The Memory Module shall operate through an active Memory Implementation.

At any given time, the active implementation serves standardized memory requests received through the Memory SDK.

Changing the active implementation shall not require changes to the public Memory SDK contract.

Executable tasks shall remain independent from implementation changes.

---

# 8. Memory Storage Boundary

Memory Storage is the persistence layer.

Its responsibility is to physically persist memory rather than define memory capabilities or execute platform-level memory operations.

Storage technology remains an implementation concern.

The platform architecture shall therefore remain independent from any particular storage technology.

---

# 9. Unified Execution Model

Memory execution follows one standardized architectural path:

```text
Memory SDK
     ↓
Memory Module
     ↓
Active Memory Implementation
     ↓
Memory Storage
```

The execution result returns through the same layers:

```text
Memory Storage
     ↓
Active Memory Implementation
     ↓
Memory Module
     ↓
Memory SDK
```

The underlying persistence technology does not alter the standardized memory execution model.

---

# 10. Architectural Invariants

The following rules are established as architectural invariants:

1. Memory is a platform-level subsystem.
2. The Memory SDK is the standardized communication boundary for memory operations.
3. Memory capabilities define logical memory behavior independently from implementation.
4. The Memory Module is the execution domain of the Memory SDK.
5. Memory Implementations translate standardized requests into implementation-specific operations.
6. Memory Storage is responsible for persistence.
7. Platform components shall not depend on storage technology.
8. Platform components shall not bypass the Memory SDK to access memory implementations.
9. The active Memory Implementation may change without changing the public Memory SDK contract.
10. Storage-specific behavior shall remain isolated from executable tasks.
11. Memory capability definitions shall remain independent from individual storage technologies.
12. New storage technologies shall be introduced through implementations rather than by changing the fundamental memory architecture.

## These boundaries preserve the separation already established by the Memory Capability and Memory Module specifications.

# 11. Alternatives Considered

## 11.1 Direct Storage Access

Platform components communicate directly with databases or storage engines.

**Rejected.**

This would expose implementation-specific behavior to the rest of the platform and break storage independence.

---

## 11.2 Component-Owned Memory

Each platform component maintains its own memory implementation.

**Rejected.**

This would fragment the memory architecture and create multiple persistence boundaries.

---

## 11.3 Direct Memory Implementation Access

Platform components communicate directly with the active Memory Implementation while bypassing the Memory SDK.

**Rejected.**

This would bypass the standardized memory communication boundary and expose implementation-specific behavior.

---

## 11.4 Storage Technology as the Platform Boundary

PRIDE defines its memory architecture around a specific database or storage technology.

**Rejected.**

Storage technologies are expected to evolve independently. The platform therefore standardizes memory behavior above the storage layer.

---

# 12. Consequences

## Positive Consequences

### Unified Memory Access

All platform memory operations follow one standardized architectural path.

### Implementation Independence

Memory implementations can evolve without requiring platform components to change.

### Storage Independence

PRIDE is not architecturally coupled to one database or persistence technology.

### Centralized Execution

Memory execution is isolated within the Memory Module.

### Stable Capability Model

Memory capabilities remain independent from storage-specific implementation details.

### Long-Term Evolution

Future persistence technologies can participate through new implementations without redesigning the fundamental memory architecture.

---

## Negative Consequences

### Additional Architectural Layers

Memory operations pass through several boundaries before reaching persistence.

### SDK Dependency

Components requiring memory depend on the Memory SDK contract.

### Boundary Governance

The Memory SDK, Memory Module, Memory Implementation, and Memory Storage boundaries must remain clearly separated.

---

# 13. Relationship With Other Specifications

This decision establishes the architectural relationship between:

* Memory SDK Specification.
* Memory Capability Specification.
* Memory Module Specification.
* Memory Storage Specification.

The Memory Capability Specification defines the logical memory functions.

The Memory Module Specification defines the execution domain and implementation management.

The Memory Storage Specification defines the persistence layer and separates storage from execution.

This ADR establishes the architectural relationship between those concerns.

---

# 14. Decision Outcome

PRIDE shall use a unified memory architecture in which:

```text
Memory Capability
       ↓
   Memory SDK
       ↓
  Memory Module
       ↓
Memory Implementation
       ↓
 Memory Storage
```

The architecture separates **what memory does**, **how memory requests are communicated**, **how memory operations are executed**, and **where memory is persisted**.

Memory implementations and storage technologies may evolve independently while the platform retains one stable memory architecture.

This decision establishes the architectural foundation for future Memory SDK, Memory Capability, Memory Module, Memory Storage, and memory-related platform specifications.
