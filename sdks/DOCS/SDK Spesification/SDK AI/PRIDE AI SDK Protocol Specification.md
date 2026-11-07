# PRIDE AI SDK Protocol Specification

**Version:** 1.1.0
**Status:** Draft
**Category:** Protocol Specification
**Document ID:** PRIDE-AI-SDK-PROTOCOL
**Protocol:** Protocol Buffers
**Transport:** gRPC
**Server Language:** Go

---

# 1. Purpose

This specification defines the network protocol used by PRIDE AI SDK Clients to communicate with the AI SDK Server.

The protocol provides a language-independent representation of:

* AI requests.
* AI responses.
* AI errors.
* AI capabilities.
* Request metadata.
* Execution metadata.

The protocol realizes the public semantic boundary defined by the PRIDE AI SDK Contract and must conform to the Stable PRIDE AI SDK and AI Module Specifications.

It does not define AI provider implementations or AI provider-specific request formats.

---


# 2. Protocol Architecture

The protocol defines the standardized request/response communication used by AI SDK clients and server-side AI execution boundaries.

The logical responsibility flow is:

```text
PRIDE Caller / AI Capability
          │
          ▼
      AI SDK Client
          │
          │ gRPC / Protocol Buffers
          ▼
       AI SDK Server
  (request validation and forwarding)
          │
          │ AI Module client boundary
          ▼
        AI Module
  (configuration and execution owner)
          │
          ├── Active AI Configuration
          ├── Credential Reference Resolution
          └── AI Implementation Resolution
                    │
                    ▼
             AI Implementation
                    │
                    ▼
             External AI System
```

The deployed topology may place the SDK-facing service and the AI Module behind separate processes or service endpoints. The protocol does not collapse their logical ownership into one component.

The AI SDK Server may accept a request, validate its envelope, map it to an internal execution DTO, forward it to the AI Module, and map the returned result or error. It does not load configuration, resolve credentials, select implementations, or execute provider/runtime operations.

The AI Module owns those execution responsibilities according to the Stable AI SDK and AI Module Specifications.

# 3. Protocol Principles

The protocol follows these principles:

1. Protocol Buffers is the canonical message schema.
2. gRPC is the canonical transport for the initial implementation.
3. The protocol is independent of AI providers.
4. The protocol is independent of AI models.
5. The protocol is independent of AI runtimes.
6. The protocol is independent of programming language.
7. The protocol does not expose provider-specific request formats.
8. The protocol does not expose provider-specific response formats.
9. The protocol does not contain AI reasoning logic.
10. The protocol does not contain AI Configuration internals.
11. Client implementations remain thin.
12. SDK server components validate and forward requests; the AI Module owns configuration resolution and execution orchestration.

---

# 4. Protocol Definition

The canonical protocol shall be represented using Protocol Buffers.

The source of truth shall be the `.proto` definition.

Generated language-specific code shall not be treated as the canonical protocol definition.

```text
ai_sdk.proto
      │
      ├── Go
      ├── Python
      ├── Rust
      ├── C#
      ├── Java
      ├── JavaScript / TypeScript
      └── Other supported languages
```

Generated SDK clients shall conform to the same protocol definition.

---

# 5. Service Definition

The initial AI SDK service shall expose a standardized AI execution operation.

Conceptual service:

```text
AIService

└── Execute
```

The operation represents a request to execute an AI capability through the active AI Configuration.

The service shall not expose provider-specific operations such as:

```text
OpenAIChat()
GeminiGenerate()
ClaudeMessage()
OllamaGenerate()
```

Provider selection remains outside the protocol contract.

---

# 6. Execute Request

The canonical execution request is:

```text
ExecuteRequest

├── request_id
├── capability
├── input
└── metadata
```

---

## 6.1 request_id

`request_id` uniquely identifies the request.

Requirements:

* Required.
* Must be non-empty.
* Must remain unchanged throughout execution.
* Must be returned in the corresponding response or error.

The request ID shall not encode provider, model, runtime, or storage information.

---

## 6.2 capability

`capability` identifies the standardized AI capability being requested.

The protocol shall treat the capability as an abstract identifier.

The protocol shall not encode provider-specific capability names.

Example conceptual values:

```text
text_generation
structured_generation
embedding
classification
```

The final capability registry is defined outside this protocol.

---

## 6.3 input

`input` contains the capability-specific input.

The protocol shall provide a generic representation capable of carrying the standardized capability input.

The protocol shall not define provider-specific request objects.

For example, the protocol shall not contain:

```text
OpenAIRequest
GeminiRequest
ClaudeRequest
OllamaRequest
```

---

## 6.4 metadata

`metadata` contains standardized contextual information associated with the request.

Metadata shall remain protocol-defined and implementation-independent.

It shall not be used to transmit:

* Provider credentials.
* Provider-specific request structures.
* Internal server state.
* AI Configuration secrets.

---

# 7. Execute Response

A successful execution returns:

```text
ExecuteResponse

├── request_id
├── capability
├── output
└── metadata
```

---

## 7.1 request_id

The response shall contain the originating request ID.

---

## 7.2 capability

The response shall identify the capability that produced the result.

---

## 7.3 output

`output` contains the standardized result of the requested AI capability.

The output shall remain independent of the selected provider.

The following shall not cross the protocol boundary:

```text
Provider-specific response objects
Provider-specific SDK objects
Provider runtime objects
Provider-specific error structures
```

---

## 7.4 metadata

Response metadata may contain standardized execution information.

Provider-specific internal metadata shall remain internal unless explicitly promoted into a future protocol contract.

---


# 8. Error Protocol

Errors use the existing standardized protocol representation:

```text
AIError

├── request_id
├── error_code
├── message
└── metadata
```

The AI Module is responsible for classifying configuration, credential, capability-compatibility, and implementation execution failures. It may return the structured `AIError` as a gRPC status detail.

The AI SDK Server and client adapter are responsible for preserving structured error details and mapping transport failures when structured details are absent. Transport status and PRIDE semantic error code are related but distinct.

When structured `AIError` details are present, they take precedence over fallback mapping based on the gRPC status code.


# 9. Error Codes

The protocol carries semantic error codes as strings through the existing `AIError.error_code` field. The currently established categories are:

```text
INVALID_REQUEST
CONFIGURATION_ERROR
AUTHENTICATION_ERROR
PERMISSION_ERROR
CONNECTION_ERROR
TIMEOUT_ERROR
CAPABILITY_UNSUPPORTED
RATE_LIMIT
RUNTIME_ERROR
UNKNOWN_ERROR
```

These values align the protocol documentation with the public AI SDK Contract and the current SDK adapter fallback mapping.

The AI Module may provide a more specific standardized semantic code in structured `AIError` details. The SDK adapter must preserve that code rather than replacing it with a generic transport fallback.

This list documents string values carried by the existing schema. It does not introduce a Protobuf enum, change the type or field number of `error_code`, or otherwise modify the wire schema.

# 10. Error Message

The `message` field provides a human-readable description.

Clients shall use `error_code` for programmatic error handling.

The server shall not expose unnecessary:

* API credentials.
* Provider secrets.
* Internal file paths.
* Internal stack traces.
* Private configuration data.
* Provider-specific implementation details.

---


# 11. AI Configuration Boundary

The protocol does not expose the internal structure of AI Configuration.

The client does not select a provider by sending provider-specific configuration. The SDK Server does not load or resolve the active AI Configuration.

The execution model is:

```text
AI SDK Client
      │
      ▼
Execute Request
      │
      ▼
AI SDK Server
(validate and forward)
      │
      ▼
AI Module
      │
      ├── Resolve active AI Configuration
      ├── Resolve credential reference
      ├── Resolve AI Implementation
      └── Dispatch capability execution
      │
      ▼
AI Implementation
      │
      ▼
External AI System
```

The client and SDK transport boundary remain unaware of:
* Which provider is active.
* Which model or runtime is active.
* Which endpoint is used.
* Which authentication mechanism is used.
* Which credential reference or secret is associated with the configuration.

Configuration and credential state remain behind the AI Module boundary.


# 12. Provider Independence

The protocol shall not contain provider-specific schemas.

The following architecture is prohibited:

```text
AIRequest
├── openai_request
├── gemini_request
├── claude_request
└── ollama_request
```

The required architecture remains:

```text
AIRequest
├── request_id
├── capability
├── input
└── metadata
```

The AI Module resolves the appropriate AI Configuration and execution mechanism. The AI Implementation translates the standardized request into provider/runtime-specific operations.

The SDK Server and clients do not select or resolve providers, models, runtimes, credentials, or implementations.

# 13. Client SDK Responsibilities

AI SDK Clients are responsible for:

* Providing a language-native API.
* Constructing protocol requests.
* Performing basic request validation.
* Serializing requests through Protocol Buffers.
* Sending gRPC requests.
* Receiving gRPC responses.
* Deserializing responses.
* Exposing standardized errors.

AI SDK Clients are not responsible for:

* AI provider selection.
* AI Configuration resolution.
* AI execution.
* AI provider authentication logic.
* AI model management.
* AI reasoning.
* Memory management.

---


# 14. AI SDK Server Responsibilities

The AI SDK Server is responsible for:
* Accepting protocol requests.
* Validating required request-envelope fields.
* Preserving request and capability identity.
* Mapping protocol messages to the internal execution DTO.
* Forwarding the request through the AI Module client boundary.
* Propagating context cancellation and deadlines.
* Mapping standardized responses to protocol responses.
* Preserving structured AI Module errors and mapping transport failures when structured details are absent.

The AI SDK Server is not responsible for:
* Resolving the active AI Configuration.
* Loading AI Configuration records.
* Resolving credential references or raw credentials.
* Resolving or selecting AI Implementations.
* Validating implementation capability support using Module-owned state.
* Constructing provider-specific requests.
* Executing remote providers or local runtimes.

These execution responsibilities belong to the AI Module and AI Implementations. The SDK Server remains independent from the client language.

# 15. Serialization

Protocol Buffers shall be used as the canonical serialization format for gRPC communication.

The protocol shall not require JSON as the primary wire representation.

JSON may be used for auxiliary tooling or debugging only when separately defined.

---

# 16. Transport

The initial protocol transport is:

```text
gRPC
```

The protocol service definitions shall therefore be expressed using gRPC-compatible Protocol Buffers service definitions.

The protocol shall remain logically independent from provider communication protocols.

For example:

```text
Plugin
   │
   │ gRPC
   ▼
AI SDK Server
   │
   │ Provider-specific communication
   ▼
AI System
```

The protocol does not require the AI provider to use gRPC.

---

# 17. Streaming

Streaming is not required for the initial protocol contract.

The initial protocol shall prioritize a stable request/response execution model.

Streaming may be introduced as a future protocol capability when PRIDE requires:

* Token streaming.
* Incremental output.
* Long-running AI execution.
* Progress events.

Such functionality shall be introduced through explicit protocol extensions rather than implicit transport behavior.

---

# 18. Versioning

The Protocol Buffers schema shall use explicit protocol versioning.

Initial package:

```text
pride.ai.v1
```

Future breaking protocol changes shall use a new major version.

Example:

```text
pride.ai.v1
pride.ai.v2
```

Compatible changes should preserve existing field numbers and semantics.

Existing field numbers shall never be reused for unrelated fields.

---

# 19. Field Compatibility

Protocol fields shall follow Protocol Buffers compatibility rules.

When a field becomes obsolete:

```text
reserved
```

shall be used where appropriate.

Field numbers shall not be reused after removal.

This protects compatibility between different versions of AI SDK Clients and the AI SDK Server.

---

# 20. Generated Client SDKs

Language-specific SDK Clients may be generated from the canonical `.proto` definition.

The generation pipeline is:

```text
ai_sdk.proto
      │
      ▼
    protoc
      │
      ├── protoc-gen-go
      ├── protoc-gen-python
      ├── protoc-gen-java
      ├── protoc-gen-csharp
      ├── protoc-gen-rust
      └── Other generators
```

Generated code shall be treated as derived artifacts.

The `.proto` definition remains authoritative.

---

# 21. Go Server Generation

The Go AI SDK Server shall use generated Go code from the canonical Protocol Buffers definition.

The initial toolchain consists of:

```text
protoc
protoc-gen-go
protoc-gen-go-grpc
```

The runtime dependencies are:

```text
google.golang.org/protobuf
google.golang.org/grpc
```

No additional server framework is required by this protocol.

---

# 22. Dependency Principle

The protocol implementation shall follow the PRIDE dependency principle:

> **Use the standard library whenever it provides the required functionality. Add an external dependency only when the requirement cannot reasonably be satisfied by the existing standard library or required protocol tooling.**

The protocol itself requires the Protocol Buffers and gRPC ecosystem.

Additional dependencies shall not be introduced merely for convenience.

---


# 23. Security Boundary

The protocol shall not transmit raw provider credentials or expose internal AI Configuration records.

AI Configuration and credential resolution remain within the AI Module and platform credential boundary.

Conceptually:

```text
Plugin / Caller
      │
      │ Standardized AI Request
      ▼
AI SDK Client
      │
      ▼
AI SDK Server
      │
      │ Forward standardized request
      ▼
AI Module
      │
      │ Resolve credential reference securely
      ▼
AI Implementation
      │
      ▼
External AI System
```

Clients and SDK transport adapters must not require direct access to provider credentials. Sensitive configuration and secret material must not be placed in ordinary request metadata.


# 24. Protocol Conformance

An implementation conforms to the PRIDE AI SDK Protocol when it:
1. Implements the canonical `.proto` service definition.
2. Accepts valid protocol requests.
3. Preserves request IDs and capability identity.
4. Produces valid protocol responses.
5. Preserves structured AI Module errors when supplied.
6. Uses the documented semantic error-code values.
7. Does not expose provider-specific structures or secrets.
8. Respects protocol versioning and Protobuf compatibility rules.
9. Keeps SDK transport responsibilities separate from AI Module execution ownership.
10. Maintains the public AI SDK Contract boundary.

Conformance to this protocol does not authorize the SDK Server to resolve configuration or execute provider/runtime operations.


# 25. Document Authority and Protocol Source of Truth

The following authority layers have distinct scopes:

1. The Stable PRIDE AI SDK Specification defines architectural ownership and the AI SDK boundary.
2. The Stable PRIDE AI Module Specification defines configuration, credential-reference, implementation-resolution, and execution ownership.
3. The PRIDE AI SDK Contract defines the public semantic request, response, and error contract.
4. This Protocol Specification defines the transport representation and protocol conformance rules.
5. The versioned `.proto` file is the source of truth for Protobuf messages, field types, field numbers, and service declarations.
6. Generated language-specific code is derived from the `.proto` source.
7. Runtime implementation must conform to all applicable layers and may not redefine them.

The hierarchy is therefore not a single chain in which a Draft document can override a Stable architectural specification. Architectural authority and wire-schema authority are separate.

A documentation revision does not by itself authorize a wire-schema or runtime behavior change.

# 26. Implementation Boundary

This specification intentionally does not define:

* AI provider implementation details.
* AI Configuration internal schema.
* AI model internals.
* AI reasoning algorithms.
* Provider-specific SDK usage.
* Database implementation.
* Server framework.
* Deployment architecture.
* Internal package architecture.

Those concerns belong to implementation-specific documentation.

---


# 27. Initial Protocol Model

The initial protocol can be summarized as:

```text
┌──────────────────────────────┐
│            Caller            │
│       AI Capability          │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│        AI SDK Client         │
│          Thin Client         │
└──────────────┬───────────────┘
               │
               │ gRPC / Protocol Buffers
               ▼
┌──────────────────────────────┐
│        AI SDK Server         │
│                              │
│ Request Validation           │
│ Protocol / DTO Mapping       │
│ Request Forwarding           │
│ Response / Error Mapping     │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│           AI Module          │
│                              │
│ Active Configuration         │
│ Credential Reference         │
│ Implementation Resolution    │
│ Capability Validation        │
│ Execution Dispatch           │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│       AI Implementation      │
│    Remote / Local Runtime    │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│       External AI System     │
└──────────────────────────────┘
```

The diagram expresses logical ownership and does not require a change to the existing service definition, message fields, field numbers, or deployment topology.

# 28. Final Protocol Principle

The PRIDE AI SDK Protocol follows one fundamental rule:

> **Define communication once, generate clients for many languages, and keep configuration resolution and AI execution inside the AI Module boundary.**

The `.proto` schema is the canonical communication definition.

The AI SDK Clients remain thin.

The AI SDK Server remains the canonical protocol ingress/forwarding boundary. The AI Module remains the configuration and execution owner.

AI Configuration determines the active AI execution environment.

Provider-specific implementation details remain outside the protocol.


---

# 29. Related Specifications

* PRIDE AI SDK Specification v2.1.0.
* PRIDE AI Module Specification v1.0.0.
* PRIDE AI SDK Contract v1.1.0.
* PRIDE AI SDK Boundary Design.
* `sdks/ai/proto/ai_sdk.proto` (wire-schema source of truth).
