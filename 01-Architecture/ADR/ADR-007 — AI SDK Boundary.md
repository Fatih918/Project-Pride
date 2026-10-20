# ADR-007 — AI SDK as the Official AI Communication Boundary

**Status:** Accepted

**Date:** 2026-08-15

**Category:** Architecture

**Decision ID:** ADR-007

---

# 1. Context

Artificial Intelligence capabilities require a stable communication boundary between PRIDE platform capabilities and Artificial Intelligence implementations.

AI providers and runtimes may differ significantly in:

* APIs.
* Authentication mechanisms.
* Request formats.
* Response formats.
* Model interfaces.
* Runtime environments.
* Streaming mechanisms.
* Tool-calling mechanisms.

Allowing platform capabilities to communicate directly with these implementations would create implementation coupling and multiple incompatible communication paths.

PRIDE therefore requires one standardized communication boundary for Artificial Intelligence execution.

The AI SDK provides this boundary.

---

# 2. Decision

PRIDE adopts the **AI SDK as the official communication boundary for Artificial Intelligence execution**.

AI Capabilities shall communicate with Artificial Intelligence execution through the AI SDK.

Platform components shall not communicate directly with AI Implementations, AI Providers, or AI Runtime environments.

The architecture is:

```text
Execution Plan
        │
        ▼
    Executor
        │
        ▼
 AI Capability
        │
        ▼
     AI SDK
        │
        ▼
   AI Module
        │
        ├── Active AI Configuration
        │
        └── AI Implementation
                    │
                    ▼
          AI Provider / Runtime
```

The AI SDK provides the standardized communication contract.

The AI Module owns configuration and implementation management.

AI Implementations provide technology-specific execution.

---

# 3. AI SDK Boundary

The AI SDK defines the standardized interface through which AI Capabilities request Artificial Intelligence execution.

The AI SDK is responsible for:

* Accepting standardized AI execution requests.
* Providing standardized AI execution responses.
* Providing standardized AI errors.
* Preserving request identity.
* Preserving capability identity.
* Transporting execution input.
* Transporting execution metadata.

The AI SDK is not responsible for managing platform configuration state.

---

# 4. AI Configuration Boundary

AI Configuration is a platform-owned concern.

AI Configurations are managed by the AI Module.

The AI SDK shall not own:

* AI Configuration lifecycle.
* Active configuration selection.
* Configuration creation.
* Configuration modification.
* Configuration deletion.
* Configuration storage.

The AI SDK may execute a request that is resolved by the AI Module using the active AI Configuration.

The configuration itself does not become part of the public AI SDK programming model.

---

# 5. Configuration Resolution

Configuration resolution is performed by the AI Module.

The execution flow is:

```text
AI SDK Request
       │
       ▼
AI Module
       │
       ▼
Resolve Active AI Configuration
       │
       ▼
Resolve Credential Reference
       │
       ▼
Resolve AI Implementation
       │
       ▼
Execute Request
```

The AI SDK does not directly resolve configurations.

This prevents configuration ownership from leaking into the SDK boundary.

---

# 6. Credential Boundary

Credentials are platform-managed security-sensitive data.

AI Configurations reference credentials using a `CredentialRef`.

The AI SDK request shall not contain:

* API keys.
* Access tokens.
* Authentication secrets.
* Encryption keys.
* Provider credentials.

The AI Module and its credential subsystem are responsible for securely resolving the required credential.

Credential storage and encryption remain outside the AI SDK.

---

# 7. AI Request Contract

The AI SDK exposes a standardized execution request.

```text
AIExecutionRequest

├── RequestID
├── Capability
├── Input
└── Metadata
```

The request does not expose implementation-specific configuration.

It does not contain:

```text
Provider
Model
APIKey
Credential
Runtime
Endpoint
Implementation Object
```

unless such information is explicitly defined as part of a future standardized capability contract.

---

# 8. AI Response Contract

The AI SDK returns standardized execution results.

```text
AIExecutionResponse

├── RequestID
├── Capability
├── Output
└── Metadata
```

Provider-specific response formats shall not cross the AI SDK boundary.

AI Implementations are responsible for translating provider-specific responses into the standardized AI SDK response.

---

# 9. AI Error Contract

AI Implementations may produce provider-specific errors.

Those errors shall be translated into standardized AI SDK errors before being returned to AI Capabilities.

The AI SDK therefore exposes:

```text
AIError
├── Code
├── Message
├── RequestID
└── Metadata
```

Provider-specific error structures shall remain hidden behind the implementation boundary.

---

# 10. AI Module Responsibility

The AI Module is responsible for managing the AI execution domain.

Its responsibilities include:

* Managing AI Configurations.
* Managing AI Implementations.
* Resolving the active AI Configuration.
* Resolving Credential References.
* Resolving the referenced AI Implementation.
* Executing requests through the selected implementation.
* Normalizing implementation-specific results.
* Returning standardized AI SDK responses.

The AI Module does not perform workflow planning or task scheduling.

---

# 11. AI Implementation Boundary

AI Implementations provide the technology-specific execution layer.

Examples include:

* Remote provider integrations.
* Local AI runtimes.
* Future AI execution technologies.

Implementations may contain:

* Provider SDK usage.
* Authentication handling.
* HTTP communication.
* Runtime communication.
* Model-specific translation.
* Provider-specific error handling.

These details shall remain hidden from the AI SDK.

---

# 12. Executor Boundary

The Executor remains independent from AI implementation details.

The Executor executes the Execution Plan and invokes the assigned capability.

The Executor does not:

* Resolve AI Configuration.
* Resolve credentials.
* Select AI providers.
* Select AI models.
* Communicate with AI Implementations.
* Perform AI reasoning.

The resulting architecture is:

```text
Execution Plan
      │
      ▼
Executor
      │
      ▼
AI Capability
      │
      ▼
AI SDK
      │
      ▼
AI Module
      │
      ▼
AI Implementation
```

---

# 13. Architectural Invariants

The following rules are established:

1. AI SDK is the official AI communication boundary.
2. AI Capabilities use the AI SDK for AI execution.
3. Platform components shall not bypass the AI SDK.
4. AI Configuration is owned by the AI Module.
5. Configuration resolution is performed by the AI Module.
6. Credentials are not exposed through the AI SDK request contract.
7. AI Implementations remain hidden behind the AI Module boundary.
8. Provider-specific behavior shall not leak into AI Capabilities.
9. AI SDK requests remain implementation-independent.
10. AI SDK responses remain implementation-independent.
11. Executor remains independent from AI-specific implementation details.
12. AI Module does not replace the Core Executor.
13. AI SDK does not become a workflow execution engine.
14. AI Configuration does not become part of the AI SDK programming model.

---

# 14. Consequences

## Positive Consequences

### Stable AI Interface

AI Capabilities depend on one standardized interface.

### Configuration Independence

AI Configuration can evolve without changing the AI SDK contract.

### Provider Independence

Providers and runtimes can change without requiring changes to AI Capabilities.

### Credential Isolation

Sensitive credentials remain outside the AI SDK request contract.

### Clear Ownership

AI Module owns configuration and implementation management.

### Deterministic Workflow Execution

The Executor remains focused on executing the Execution Plan.

---

## Negative Consequences

### Additional Boundary

Requests pass through the AI SDK and AI Module before reaching an implementation.

### AI Module Complexity

The AI Module must manage configuration, credential resolution, implementation resolution, and execution.

### Contract Governance

The AI SDK contract must remain implementation-independent as the AI ecosystem evolves.

---

# 15. Alternatives Considered

## 15.1 Direct AI Provider Access

Rejected.

Platform capabilities would become coupled to provider APIs.

---

## 15.2 Configuration-Owned SDK

Rejected.

The SDK would become responsible for platform state that belongs to the AI Module.

---

## 15.3 Credential-Owned SDK

Rejected.

The SDK would become coupled to credential storage and security mechanisms.

---

## 15.4 Executor-Owned AI Execution

Rejected.

The Executor is responsible for executing workflow tasks, not Artificial Intelligence implementation logic.

---

# 16. Decision Outcome

PRIDE shall use the AI SDK as the standardized communication boundary between AI Capabilities and the AI execution domain.

The AI Module owns AI Configurations and AI Implementations.

Credentials remain protected behind the platform credential boundary.

The Executor remains responsible only for executing the Execution Plan.

AI Implementations remain replaceable and provider-independent from the perspective of the platform.

This architecture provides a stable AI programming boundary while allowing configurations, credentials, providers, runtimes, and implementation technologies to evolve independently.
