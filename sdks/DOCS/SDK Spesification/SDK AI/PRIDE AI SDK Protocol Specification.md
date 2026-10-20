# PRIDE AI SDK Protocol Specification

**Version:** 1.0.0
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

The protocol is derived from the PRIDE AI SDK Contract.

It does not define AI provider implementations or AI provider-specific request formats.

---

# 2. Protocol Architecture

The protocol establishes the following communication model:

```text
PRIDE Plugin
(Python / Go / Rust / C# / JavaScript / ...)
        │
        ▼
AI SDK Client
        │
        │ gRPC
        ▼
AI SDK Server
        │
        ▼
Active AI Configuration
        │
        ▼
AI Execution
```

The AI SDK Client is a thin protocol client.

The AI SDK Server is the canonical server-side implementation of the AI SDK boundary.

---

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
12. Server implementations remain responsible for execution.

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

Errors shall use a standardized protocol representation.

Conceptual structure:

```text
AIError

├── request_id
├── error_code
├── message
└── metadata
```

The AI SDK Server is responsible for translating internal execution failures into standardized AI SDK errors.

---

# 9. Error Codes

The initial protocol error categories are:

```text
INVALID_REQUEST
INVALID_INPUT
CAPABILITY_NOT_FOUND
CAPABILITY_UNSUPPORTED
CONFIGURATION_NOT_FOUND
CONFIGURATION_INVALID
AI_UNAVAILABLE
EXECUTION_ERROR
TIMEOUT
PERMISSION_DENIED
INTERNAL_ERROR
UNKNOWN_ERROR
```

The error taxonomy may evolve through compatible protocol versions.

Provider-specific errors shall be normalized into the applicable AI SDK error category.

---

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

The Client does not select a provider by sending provider-specific configuration.

The execution model is:

```text
AI SDK Client
      │
      ▼
Execute Request
      │
      ▼
AI SDK Server
      │
      ▼
Active AI Configuration
      │
      ▼
AI Execution
```

The AI SDK Server resolves and uses the active AI Configuration.

The Client therefore remains unaware of:

* Which provider is active.
* Which model is active.
* Which runtime is active.
* Which endpoint is used.
* Which authentication mechanism is used.

---

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

The required architecture is:

```text
AIRequest
├── request_id
├── capability
├── input
└── metadata
```

The server resolves the appropriate AI execution mechanism through the active AI Configuration.

---

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

* Accepting gRPC requests.
* Validating protocol requests.
* Resolving the active AI Configuration.
* Executing the requested AI capability.
* Translating the execution into the configured AI system.
* Normalizing responses.
* Normalizing errors.
* Returning standardized protocol responses.

The server shall remain independent from the client language.

---

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

The protocol shall not transmit raw provider credentials unless explicitly required by a future security contract.

AI Configuration remains responsible for managing provider-specific authentication information.

Clients should not need direct access to provider credentials.

Conceptually:

```text
Plugin
   │
   │ AI Request
   ▼
AI SDK Server
   │
   │ Active Configuration
   ▼
AI Provider
```

The provider credential remains behind the AI SDK Server boundary.

---

# 24. Protocol Conformance

An implementation conforms to the PRIDE AI SDK Protocol when it:

1. Implements the canonical `.proto` service definition.
2. Accepts valid protocol requests.
3. Produces valid protocol responses.
4. Produces standardized protocol errors.
5. Preserves request IDs.
6. Preserves capability semantics.
7. Does not expose provider-specific structures.
8. Respects protocol versioning rules.
9. Maintains Protocol Buffers compatibility.
10. Maintains the AI SDK Contract boundary.

---

# 25. Protocol Source of Truth

The protocol hierarchy is:

```text
PRIDE AI SDK Contract
          │
          ▼
AI SDK Protocol Specification
          │
          ▼
       ai_sdk.proto
          │
          ▼
 Generated Client / Server Code
          │
          ▼
 Implementation
```

Each lower layer must conform to the layer above it.

Generated code shall not redefine the contract.

Implementation code shall not redefine the protocol.

---

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

The initial protocol can therefore be summarized as:

```text
┌──────────────────────────────┐
│            Plugin            │
│      Any Supported Language  │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│        AI SDK Client         │
│          Thin Client         │
└──────────────┬───────────────┘
               │
               │ gRPC
               │ Protocol Buffers
               ▼
┌──────────────────────────────┐
│        AI SDK Server         │
│             Go               │
│                              │
│ Request Validation           │
│ Configuration Resolution     │
│ Capability Execution         │
│ Response Normalization       │
│ Error Normalization          │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│      Active AI Configuration │
└──────────────┬───────────────┘
               │
               ▼
┌──────────────────────────────┐
│          AI System           │
│      Remote / Local / etc.   │
└──────────────────────────────┘
```

---

# 28. Final Protocol Principle

The PRIDE AI SDK Protocol follows one fundamental rule:

> **Define communication once, generate clients for many languages, and keep AI execution behind the server boundary.**

The `.proto` schema is the canonical communication definition.

The AI SDK Clients remain thin.

The AI SDK Server remains the canonical execution boundary.

AI Configuration determines the active AI execution environment.

Provider-specific implementation details remain outside the protocol.
