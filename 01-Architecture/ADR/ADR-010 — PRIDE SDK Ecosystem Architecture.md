# ADR-010 — PRIDE SDK Ecosystem Architecture

**Status:** Draft

**Date:** 2026-11-01

**Category:** Architecture

**Decision ID:** ADR-010

---

# 1. Context

PRIDE exposes several platform domains through dedicated SDK boundaries.

The current architecture establishes three major SDKs:

* **Plugin SDK** — the official communication boundary for Plugin operations.
* **Memory SDK** — the standardized communication boundary for platform memory operations.
* **AI SDK** — the official communication boundary for Artificial Intelligence execution.

Each SDK represents a distinct platform domain with its own execution module, implementation details, and lifecycle.

These domains are related during the PRIDE Core workflow. Memory participates in context retrieval and memory update. AI may support reasoning and other AI capabilities. Plugins provide executable extensions that perform work on behalf of the platform.

Without an ecosystem-level contract, the SDKs could evolve as isolated products or become directly coupled to one another. Either outcome would weaken the established boundaries:

* Independent SDKs could introduce inconsistent request context, error handling, lifecycle, and compatibility expectations.
* Direct SDK-to-SDK dependencies could create circular dependencies and make one domain's release dependent on another domain's internal changes.
* Individual SDKs could gradually absorb Core orchestration responsibilities.
* Cross-domain implementation details, credentials, provider information, or storage details could leak through public contracts.

PRIDE therefore requires one architectural decision that defines how the Plugin SDK, Memory SDK, and AI SDK coexist and collaborate while preserving their independent domain boundaries.

This ADR establishes the ecosystem contract. It does not replace the individual SDK boundary decisions.

---

# 2. Decision

PRIDE adopts a **Unified SDK Ecosystem Architecture** consisting of three independent, domain-specific SDKs coordinated by the PRIDE Core.

The three SDKs are:

1. Plugin SDK.
2. Memory SDK.
3. AI SDK.

Each SDK shall expose the official communication contract for its own domain.

The PRIDE Core shall remain the composition and orchestration layer that coordinates cross-domain workflows.

The SDKs shall not directly depend on one another as a requirement of their public architecture.

The resulting high-level architecture is:

```text
                    PRIDE Application
                           │
                           ▼
                    PRIDE Core
             (Composition & Orchestration)
                           │
          ┌────────────────┼────────────────┐
          │                │                │
          ▼                ▼                ▼
      Plugin SDK       Memory SDK         AI SDK
          │                │                │
          ▼                ▼                ▼
     Plugin Module    Memory Module       AI Module
          │                │                │
          ▼                ▼                ▼
     Plugin Runtime   Memory Implementation  AI Implementation
                           │                │
                           ▼                ▼
                     Memory Storage    Provider / Runtime
```

The diagram describes logical ownership and communication boundaries. It does not require all SDKs to execute for every request, nor does it prescribe a specific deployment topology or transport.

---

# 3. Domain Ownership

Each SDK owns one domain-specific communication contract.

## 3.1 Plugin SDK — Extension and Plugin Operations

The Plugin SDK is the official communication boundary for Plugin operations.

It provides standardized contracts through which platform components request Plugin discovery, lifecycle, metadata, state, and execution operations as defined by the Plugin architecture.

The Plugin SDK shall not:

* Implement Plugin functionality.
* Own the internal execution of Plugin code.
* Replace the Plugin Module.
* Become the owner of workflow planning or Core orchestration.
* Depend on AI or Memory SDK internals.

Plugin implementation and runtime details remain behind the Plugin Module boundary.

## 3.2 Memory SDK — Memory Operations

The Memory SDK is the standardized communication boundary for platform memory operations.

It forwards standardized memory requests to the Memory Module, which executes them through the active Memory Implementation and the selected storage technology.

The Memory SDK shall not:

* Own memory persistence.
* Select or operate storage engines directly.
* Implement AI reasoning.
* Own the Core workflow.
* Depend on AI or Plugin SDK internals.

Memory capabilities define logical memory behavior independently from the underlying persistence technology.

## 3.3 AI SDK — Artificial Intelligence Execution

The AI SDK is the official communication boundary for Artificial Intelligence execution.

It transports standardized AI execution requests and responses between AI Capabilities and the AI Module.

The AI SDK shall not:

* Own AI Configuration lifecycle.
* Resolve or store credentials.
* Own provider or runtime implementations.
* Become a workflow execution engine.
* Own memory persistence or Plugin lifecycle.
* Depend on Plugin or Memory SDK internals.

AI Configuration, credential resolution, and AI Implementation management remain responsibilities of the AI Module and their designated platform boundaries.

---

# 4. Core as the Composition Boundary

The PRIDE Core owns cross-domain coordination.

When a workflow requires more than one SDK, the Core coordinates the sequence and passes the required domain-level data through the appropriate SDK contracts.

For example, a standard workflow may use:

```text
User Request
     │
     ▼
PRIDE Core Workflow
     │
     ├── Memory Retrieval
     │       └── Memory SDK
     │
     ├── Context Construction and Planning
     │
     ├── AI-assisted processing, when required
     │       └── AI SDK
     │
     ├── Capability Resolution and Permission
     │
     ├── Plugin Execution, when required
     │       └── Plugin SDK
     │
     └── Memory Update
             └── Memory SDK
```

This is an illustrative interaction sequence consistent with the Core workflow. It does not redefine the canonical workflow order or require AI execution for every request.

The Core is responsible for:

* Deciding when a domain operation is required.
* Coordinating calls to the appropriate SDK.
* Managing the cross-domain workflow and its state.
* Passing only the data required by each domain contract.
* Applying platform-level policy and permission decisions.
* Handling the overall workflow result.

The Core shall not bypass an SDK to communicate directly with that SDK's Module or implementation.

---

# 5. SDK Independence and Dependency Direction

The three SDKs are sibling domain boundaries.

The following dependency rule is mandatory:

```text
                 PRIDE Core
                /     |     \
               ▼      ▼      ▼
          Plugin SDK Memory SDK AI SDK
```

The Core may coordinate multiple SDKs through their public contracts.

An SDK shall not require another SDK as a direct dependency merely to perform its own domain responsibility.

In particular:

* Plugin SDK shall not import Memory SDK or AI SDK.
* Memory SDK shall not import Plugin SDK or AI SDK.
* AI SDK shall not import Plugin SDK or Memory SDK.

This rule applies to public SDK packages and their required runtime dependencies. It does not prohibit an application or Core-owned orchestration component from using multiple SDKs.

If a domain requires another domain's capability, the interaction shall be expressed through a Core-owned orchestration flow or an explicitly defined adapter at the appropriate composition boundary. Such an adapter shall not expose the other SDK's implementation internals.

Circular dependencies between SDKs are prohibited.

---

# 6. Cross-SDK Collaboration

Cross-domain collaboration is an orchestration concern, not a reason to merge SDK responsibilities.

Examples:

## 6.1 AI-Assisted Memory Retrieval

A Memory operation may use embeddings or AI-assisted processing where the Memory architecture explicitly requires it.

The Memory domain remains responsible for the memory operation and its persistence semantics.

Any use of AI functionality shall occur through an explicit integration contract or adapter. The Memory SDK public contract shall not become a wrapper around AI SDK internals.

## 6.2 AI-Assisted Plugin Execution

AI may help determine which capability or Plugin should be used.

The Core remains responsible for planning, capability resolution, permission checks, and execution orchestration.

Plugin execution continues through the Plugin SDK. The AI SDK does not directly execute Plugins.

## 6.3 Memory Context for AI Execution

The Core may retrieve relevant memory through the Memory SDK and include the resulting, policy-approved context in an AI execution request.

The AI SDK transports the request. It does not retrieve, mutate, or own the memory.

The Core and applicable platform policies remain responsible for deciding what context may be supplied.

These examples establish collaboration boundaries, not mandatory features or implementation details.

---

# 7. Shared Contract Principles

The SDK ecosystem shall follow consistent platform-level contract principles without forcing all domains to use one generic request or response model.

## 7.1 Request Identity and Context

Where applicable, SDK operations should preserve the platform request identity and execution context required for tracing and cancellation.

Domain-specific request and response types remain owned by their respective SDKs.

A shared context convention may be established by a separate technical specification. This ADR does not prescribe a concrete struct, protobuf message, package, or field layout.

## 7.2 Cancellation and Deadlines

SDK operations that perform work across a process or service boundary shall support the cancellation and deadline semantics of their transport and execution environment.

An SDK shall not silently discard cancellation or extend a caller's deadline without an explicit contract.

## 7.3 Error Consistency

Each SDK shall expose errors that are stable and meaningful to its consumers.

Domain-specific error codes remain owned by the relevant SDK. Platform-wide error conventions may be shared, but one SDK shall not expose another SDK's private error types as part of its public API.

Underlying causes may be retained internally where safe and useful. Public error messages and metadata shall not disclose secrets or implementation-sensitive information.

## 7.4 Metadata

Metadata may carry explicitly supported contextual information required for an operation.

Metadata shall not become an unstructured channel for bypassing domain contracts or transferring credentials, access tokens, passwords, private keys, or other secrets.

Each SDK shall define and validate the metadata it accepts as part of its own technical contract.

## 7.5 Compatibility

Each SDK shall maintain a clear compatibility policy for its public contract.

A change to one SDK's public contract shall not require unrelated SDKs to change unless an explicitly shared contract has changed.

Concrete versioning rules and release procedures belong in a Technical Specification.

---

# 8. Security and Trust Boundaries

The SDK ecosystem shall preserve the security ownership of each domain.

* Plugin permissions and execution authorization remain governed by the Plugin and Core security architecture.
* Memory access policy, isolation, and storage protection remain governed by the Memory and Core architecture.
* AI configuration and credential resolution remain governed by the AI Module and platform credential boundaries.
* The Core remains responsible for coordinating platform-level authorization and policy decisions across a workflow.

No SDK may use another SDK as a shortcut around authorization, permission checks, or credential boundaries.

Raw credentials and secrets shall not be passed between SDKs through request metadata, generic context, or cross-domain payloads.

When data crosses a domain boundary, the sending and receiving components must use the receiving domain's supported public contract and applicable platform policy.

---

# 9. Lifecycle and Failure Isolation

Each SDK and its corresponding Module retain ownership of their own domain lifecycle.

The Core coordinates the overall workflow lifecycle but shall not take over internal resource management belonging to an SDK or Module.

A failure in one domain shall be represented through that domain's error contract and handled by the Core according to workflow policy.

An SDK shall not silently invoke another SDK as an implicit fallback.

Retries, compensation, partial completion, and recovery across multiple domains must be defined by the Core workflow or a dedicated technical specification. They shall not be independently invented by individual SDKs.

This decision does not prescribe a shared connection pool, shared runtime, deployment unit, or process boundary.

---

# 10. What This Decision Does Not Define

This ADR does not define:

* Concrete SDK public interfaces or method signatures.
* Protobuf schemas or gRPC service definitions.
* Go package names or module layouts.
* Plugin manifests or packaging formats.
* Memory schemas, storage models, or retrieval algorithms.
* AI provider adapters, models, or configuration schemas.
* A universal request or response DTO.
* A shared SDK runtime or mandatory shared client.
* Cross-SDK retry or compensation algorithms.
* SDK release automation or semantic versioning rules.
* Implementation order, milestones, or production-readiness criteria.

Those concerns belong to the relevant ADRs, Technical Specifications, or implementation plans.

---

# 11. Alternatives Considered

## 11.1 One Unified SDK

All Plugin, Memory, and AI operations are exposed through one combined SDK.

**Rejected.**

This would combine unrelated domain responsibilities, enlarge the public contract, and make independent evolution more difficult.

## 11.2 Direct SDK-to-SDK Dependencies

Each SDK may import and invoke the other SDKs when it needs cross-domain functionality.

**Rejected.**

This would create hidden coupling, complicate compatibility, and risk circular dependencies. Cross-domain coordination belongs to the Core or an explicitly defined adapter boundary.

## 11.3 Independent SDKs Without Ecosystem Rules

Each SDK evolves with its own conventions and integration model.

**Rejected.**

This would allow inconsistent context, error, security, and lifecycle behavior across a platform whose workflows use multiple domains.

## 11.4 Core Bypasses SDKs

The Core communicates directly with the Plugin, Memory, or AI Modules and implementations.

**Rejected.**

This would undermine the official communication boundaries established by the individual SDK architecture decisions.

---

# 12. Consequences

## Positive Consequences

### Independent Domain Evolution

Plugin, Memory, and AI SDKs can evolve independently while preserving their own public contracts.

### Clear Orchestration Ownership

The Core remains the single architectural owner of cross-domain workflow coordination.

### Reduced Coupling

SDKs do not require direct dependencies on one another.

### Consistent Platform Expectations

Shared principles for context, cancellation, errors, metadata, security, and compatibility guide the ecosystem.

### Replaceable Implementations

Modules, providers, runtimes, and storage implementations remain behind their respective boundaries.

### Incremental Development

Each SDK can be implemented and hardened in independently reviewable stages without changing the overall ecosystem architecture.

## Negative Consequences

### Core Coordination Responsibility

The Core must coordinate cross-domain workflows and handle partial failures.

### Contract Governance

Changes to shared conventions require deliberate compatibility management.

### Integration Work

A workflow using multiple SDKs requires explicit composition and integration tests.

### Additional Boundaries

Requests may cross multiple SDK and Module boundaries, adding transport and lifecycle considerations.

These trade-offs are accepted to preserve independent domain ownership and long-term platform evolution.

---

# 13. Architectural Invariants

The following rules are established as architectural invariants:

1. PRIDE maintains separate Plugin, Memory, and AI SDKs.
2. Each SDK is the official communication boundary for its own domain.
3. The PRIDE Core owns cross-domain composition and orchestration.
4. SDKs shall not directly depend on one another as a requirement of their public architecture.
5. Circular dependencies between SDKs are prohibited.
6. Cross-domain interactions shall occur through Core orchestration or an explicitly defined adapter boundary.
7. The Core shall not bypass an SDK to communicate directly with its Module or implementation.
8. Each domain retains ownership of its request, response, error, and lifecycle contracts.
9. Shared conventions shall not force unrelated domains into one generic request or response model.
10. Metadata and context shall not be used to transfer secrets or bypass domain contracts.
11. Authorization, permission, and credential boundaries remain enforced by their owning platform domains and the Core.
12. Failure recovery across multiple SDK domains is coordinated by the Core or a dedicated workflow contract.
13. Individual SDKs and their implementations may evolve independently while remaining compatible with their established public contracts.
14. The existence of three SDKs does not require all three to participate in every workflow.

---

# 14. Relationship With Other Decisions

This ADR builds upon:

* **ADR-001 — Plugin-First Architecture**
* **ADR-002 — Plugin SDK as the Official Plugin Communication Boundary**
* **ADR-006 — Unified Memory Architecture**
* **ADR-007 — AI SDK as the Official AI Communication Boundary**
* **ADR-009 — Stable Core Workflow**

The individual ADRs define the boundaries and responsibilities of their respective domains.

This ADR defines how those boundaries coexist as one SDK ecosystem and how the Core coordinates their participation.

This ADR does not supersede or weaken any of the decisions listed above.

---

# 15. Decision Outcome

PRIDE shall maintain three independent SDK boundaries:

* Plugin SDK for Plugin operations.
* Memory SDK for memory operations.
* AI SDK for Artificial Intelligence execution.

The PRIDE Core shall coordinate workflows that require one or more of these domains.

SDKs shall remain independent from one another at the public architecture level. Cross-domain collaboration shall be explicit and shall occur through Core orchestration or a separately defined adapter boundary.

The ecosystem shall share platform-level expectations for context, cancellation, errors, metadata, security, and compatibility without collapsing domain-specific contracts into one universal SDK model.

This decision establishes the architectural contract for the PRIDE SDK ecosystem while preserving the independent evolution of Plugin, Memory, and AI domains.
