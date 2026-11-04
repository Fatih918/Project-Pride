# PRIDE AI SDK Boundary Design

**Version:** 1.0.0  
**Status:** Draft  
**Category:** SDK Architecture / Boundary Design  
**Document ID:** PRIDE-AI-SDK-BOUNDARY  
**Related Stable Specification:** PRIDE AI SDK Specification v2.1.0

---

## 1. Purpose

This document makes the implementation and ownership boundaries of the PRIDE AI SDK explicit.

It complements the stable AI SDK Specification. It does not replace the AI Module, AI Configuration, AI Capability, Memory, or Core specifications.

The central rule is:

> The AI SDK transports standardized AI capability execution requests. The AI Module owns configuration resolution and AI implementation execution.

## 2. Architectural Position

The AI SDK is the official communication boundary between PRIDE AI Capabilities and the AI Module execution domain.

It is not an AI provider abstraction library embedded into every consumer, nor is it the owner of AI configuration or provider execution.

```text
PRIDE PLATFORM / CALLER SIDE
┌────────────────────────────────────────────┐
│ Core Execution / Executable Task           │
│                  │                         │
│                  ▼                         │
│             AI Capability                 │
└──────────────────┬─────────────────────────┘
                   │ ExecuteRequest
                   ▼
┌────────────────────────────────────────────┐
│              PRIDE AI SDK                  │
│                                            │
│  Public gRPC contract: AIService.Execute   │
│  Request validation                        │
│  Protocol ↔ internal DTO mapping           │
│  Request/response correlation              │
│  Transport error normalization             │
└──────────────────┬─────────────────────────┘
                   │ Internal AIClient port
                   ▼
┌────────────────────────────────────────────┐
│                AI MODULE                   │
│                                            │
│  Active AI Configuration resolution        │
│  Credential reference resolution           │
│  AI Implementation resolution              │
│  Capability execution dispatch             │
└──────────────────┬─────────────────────────┘
                   │
                   ▼
┌────────────────────────────────────────────┐
│            AI IMPLEMENTATION               │
│  Remote integration / Local runtime        │
└──────────────────┬─────────────────────────┘
                   │
                   ▼
          External AI System
```

The diagram expresses logical ownership. The current repository implements the SDK-facing gRPC service and an internal gRPC client adapter to the AI Module. The AI Module and its provider/runtime implementation are separate components.

## 3. Boundary Responsibilities

### 3.1 AI Capability / Caller

Responsible for:
- deciding when an AI operation is required as part of its own domain behavior;
- choosing a standardized AI capability identifier defined by the AI Capability architecture;
- constructing capability-specific input;
- supplying request identity and permitted execution metadata;
- interpreting the capability-specific output;
- handling standardized execution failures.

Not responsible for:
- selecting a provider, model, or runtime;
- reading provider credentials;
- calling vendor APIs directly;
- managing AI Configuration;
- owning persistent memory through the AI SDK.

### 3.2 AI SDK

Responsible for:
- exposing the canonical `AIService.Execute` protocol;
- validating the generic request envelope;
- preserving request ID and capability identity across the boundary;
- mapping Protobuf messages to the SDK's internal execution DTOs and back;
- forwarding the request through the AI Module client boundary;
- returning standardized response data;
- translating downstream gRPC failures into the SDK's `RemoteError` representation where applicable.

Not responsible for:
- creating, editing, deleting, or selecting AI Configurations;
- resolving the active AI Configuration;
- resolving CredentialRef or raw credentials;
- selecting or implementing providers, models, or runtimes;
- constructing vendor-specific HTTP/RPC requests;
- running inference;
- capability discovery or Core Capability Registry selection;
- workflow planning or Core Execution Plan orchestration;
- persistent memory, conversation history, or session storage;
- automatic fallback to another AI Configuration or implementation.

### 3.3 AI Module

Responsible for:
- owning the AI Configuration domain;
- resolving the active configuration according to platform state;
- resolving the configuration's implementation reference;
- resolving required credentials through the platform credential boundary;
- validating whether the selected implementation supports the requested capability;
- dispatching the request to the selected AI Implementation;
- normalizing implementation results and failures to the AI SDK contract.

The AI Module does not become the Core workflow planner or the owner of platform memory.

### 3.4 AI Configuration

AI Configuration is user-owned platform state managed by the AI Module domain. It determines the execution target and its required settings.

The SDK request does not expose the internal configuration schema. The consumer does not send a provider/model selection field as a substitute for the active configuration.

### 3.5 Credential Store / Credential Boundary

Responsible for protected storage and resolution of secret material.

Credential references and raw credentials must not be exposed to AI Capabilities through the public AI SDK contract. Raw secrets must not be placed in request metadata or capability input unless a separately approved security contract explicitly requires a different mechanism.

### 3.6 AI Implementation

Responsible for translating standardized capability requests into the selected remote provider or local runtime's native protocol, authenticating through the approved credential path, executing the operation, and normalizing the result.

Implementation-specific types and errors must not leak into the AI Capability-facing contract.

### 3.7 Core

Core remains responsible for workflow planning, Execution Plan construction, and task orchestration. Core or an executable task may invoke an AI Capability when AI assistance is required; this does not make the AI SDK responsible for planning or orchestration.

### 3.8 Memory SDK / Memory Module

Memory remains an independent domain. The AI SDK is stateless and does not own persistent memory, history, context storage, or session synchronization.

A caller may provide context required for a single AI operation, but the source, persistence, retrieval, and lifecycle of that context remain outside the AI SDK boundary.

## 4. Current Code Boundary

The current Go implementation contains these internal seams:

| Package / Type | Role |
|---|---|
| `server/internal/service/ai.Service` | Implements the generated `AIServiceServer` and validates/maps the incoming request |
| `server/internal/execution.Executor` | Delegates the logical execution request through the injected AI client |
| `server/internal/contracts/execution.AIExecutionRequest` | SDK-internal request DTO |
| `server/internal/contracts/execution.AIExecutionResult` | SDK-internal result DTO |
| `server/internal/contracts/module.AIClient` | SDK-internal Module-facing port |
| `server/internal/transport/grpc.GRPCAIClient` | Implements the Module-facing port using generated `AIServiceClient` and maps transport messages/errors |

The generated public protocol is in `sdks/ai/proto`:
- package: `pride.ai.v1`;
- Go import path: `pride-ai-sdk/proto`;
- service: `AIService`;
- current RPC: `Execute`.

The `server/internal` packages are private to the Go module according to Go's `internal` import rules. External consumers should use the generated protocol package or a separately published public client package if one is introduced later.

## 5. Request and Response Boundary

### 5.1 Request path

1. The caller creates an `ExecuteRequest`.
2. The AI SDK service validates that the request, request ID, capability, and input are present.
3. The service maps the Protobuf request into `AIExecutionRequest`.
4. The SDK Executor delegates to the injected `module.AIClient`.
5. The gRPC adapter maps the internal DTO to the AI Module's `ExecuteRequest` and forwards the caller's context.
6. The AI Module resolves the active AI Configuration, credential reference, and implementation.
7. The selected implementation executes the requested capability.
8. The result returns through the same boundary and is mapped to `ExecuteResponse`.

Steps 6–7 are AI Module / AI Implementation responsibilities, not work performed by the SDK's generic transport adapter.

### 5.2 Response invariants

- Preserve the originating `request_id`.
- Preserve the requested capability identity.
- Return capability-specific output using the applicable capability contract.
- Return only standardized response metadata.
- Do not expose provider-specific response objects or credentials.

The current adapter rejects an empty response, empty response request ID, empty capability, or nil output. A capability-specific validator may impose additional constraints.

### 5.3 Error path

The protocol represents an `AIError` with request ID, error code, message, and metadata. gRPC status communicates the transport-level failure. When the downstream service attaches an `AIError` as a gRPC status detail, the current adapter preserves that structured information in `RemoteError`.

If structured details are absent, the adapter uses a fallback mapping from gRPC status codes. The current code does not establish that every server-side failure is already packed into an `AIError` detail; this remains an integration/conformance requirement.

## 6. Dependency Rules

The following dependency direction is required:

```text
AI Capability / Caller
          |
          v
       AI SDK
          |
          v
      AI Module
          |
          v
   AI Implementation
          |
          v
   External AI System
```

Rules:

1. AI Capabilities depend on the AI SDK contract, not on AI Module internals.
2. The AI SDK depends on its internal Module-facing port, not on provider packages.
3. AI Module owns configuration and implementation resolution.
4. AI Implementations depend on their own provider/runtime protocols behind the Module boundary.
5. The AI SDK must not depend directly on the Memory SDK or Plugin SDK.
6. Memory and Plugin domains must not call provider APIs as alternate AI execution paths.
7. Core orchestrates workflows but must not absorb provider-specific AI execution code.
8. No layer may bypass the active AI Configuration by silently choosing another provider or implementation.

## 7. Data That May Cross the Boundary

| Data | Cross AI SDK request boundary? | Rule |
|---|---:|---|
| Request ID | Yes | Required, non-empty |
| Capability identifier | Yes | Required, standardized |
| Capability-specific input | Yes | Packed in `Any`, schema defined by capability contract |
| Execution metadata | Yes | Optional, structured, non-secret |
| Capability output | Yes, in response | Standardized capability output |
| Response metadata | Yes, in response | Optional and standardized |
| Provider name/model selection | No, not as caller-controlled provider routing | Determined by active AI Configuration / platform policy |
| Provider endpoint | No | AI Module / implementation internal |
| Credential reference | No, not in public request | Resolved behind platform boundary |
| Raw API key/token/secret | No | Must remain protected |
| Persistent conversation history | No ownership transfer | Memory domain remains authoritative |
| Provider-native request/response objects | No | Translated inside AI Implementation |

## 8. Capability Ownership

The AI SDK transports a capability identifier; it does not own the capability taxonomy or its behavioral definition.

The AI Capability architecture defines:
- capability identifiers;
- input and output schemas;
- semantic behavior;
- capability-specific validation;
- support/compatibility expectations.

The current `ai_sdk.proto` includes `TextGenerateInput` and `TextGenerateOutput` as canonical text-generation payload messages. This does not mean the SDK itself implements text generation or that every AI Module implementation supports it.

Capability discovery, capability matching, and implementation selection are not responsibilities of the AI SDK transport layer.

## 9. Configuration Selection and Failure Policy

The active AI Configuration is selected through platform configuration behavior and resolved by the AI Module.

The AI SDK must not:
- create a hidden configuration;
- modify user configuration;
- select a provider or model on behalf of the user;
- silently switch to another active configuration;
- perform automatic provider fallback.

If the active configuration is unavailable, invalid, disabled, or cannot support the requested capability, the AI Module must return an appropriate standardized failure. The SDK forwards that failure; it does not repair configuration state.

## 10. Security and Operational Boundary

The current server bootstrap is minimal and uses insecure gRPC transport credentials for its downstream connection. The current test client also uses insecure credentials.

This is a development-only setup. Production integration owners must define:
- encrypted transport and service identity;
- caller authentication;
- authorization policy;
- network exposure and service-to-service trust;
- credential storage and secret access policy;
- request size and timeout limits;
- observability that avoids logging secrets or sensitive payloads.

The SDK contract does not, by itself, prove that these deployment controls are configured.

## 11. Conformance Requirements

An AI SDK integration conforms to this boundary when it:

1. Uses the canonical `AIService.Execute` contract.
2. Preserves request identity and capability identity.
3. Uses capability-defined input/output schemas.
4. Does not expose provider-specific data through the public request/response.
5. Leaves active configuration resolution to the AI Module.
6. Keeps credentials behind the platform credential boundary.
7. Does not create an alternate AI execution path through provider APIs.
8. Keeps persistent memory ownership outside the AI SDK.
9. Propagates context cancellation/deadlines across the SDK-to-Module call.
10. Returns standardized errors without leaking secrets.
11. Does not silently switch configuration or implementation.
12. Preserves Protobuf compatibility and versioning rules.

## 12. Known Documentation / Contract Alignment Items

The repository contains earlier draft material whose wording does not fully match the stable SDK specification and current code. These differences must be resolved deliberately in a separate contract/specification change; this boundary document does not silently rewrite those documents.

| Existing material | Difference | Current boundary used by this document |
|---|---|---|
| Draft `PRIDE AI SDK Contract.md` sections 9–10 and 24 | Some passages describe configuration resolution as SDK work and show configuration/implementation directly below the SDK | Stable SDK Specification v2.1.0 and Module Specification assign active configuration, credential, and implementation resolution to the AI Module |
| Draft contract error taxonomy | Uses categories such as `CONFIGURATION_ERROR` and `CAPABILITY_UNSUPPORTED` | The current gRPC adapter also maps transport statuses to fallback codes such as `INVALID_REQUEST`; protocol and implementation error vocabularies need a canonical reconciliation |
| Protocol Specification | Describes the SDK Server as resolving configuration and executing the capability | The current server implementation delegates through an internal AI client to the separate AI Module; the stable boundary assigns resolution/execution ownership to that Module |
| Current server bootstrap | Uses insecure gRPC credentials and a minimal composition root | Development bootstrap only; production transport/security requirements must be supplied by deployment integration |

Recommended follow-up is to reconcile the Draft Contract and Protocol Specification with the stable SDK Specification, then update their status/versioning. Do not treat this usage/boundary documentation as permission to change the wire schema or runtime behavior.

## 13. Related Specifications

- [PRIDE AI SDK Specification](../../../04-Platform%20Specification%20Standard/SDK%20Specification/PRIDE%20AI%20SDK%20Specification.md)
- [PRIDE AI SDK Contract](PRIDE%20AI%20SDK%20Contract.md)
- [PRIDE AI SDK Protocol Specification](PRIDE%20AI%20SDK%20Protocol%20Specification.md)
- [PRIDE AI Module Specification](../../../04-Platform%20Specification%20Standard/Module%20Specification/PRIDE%20AI%20Module%20Specification.md)
- [AI SDK Usage Guide](PRIDE%20AI%20SDK%20Usage%20Guide.md)
