# PRIDE AI SDK — Protobuf Design

**Version:** 1.0.0 Draft
**Status:** Draft
**Category:** SDK Protocol Design
**Document ID:** PRIDE-AI-SDK-PROTO-DESIGN

---

# 1. Purpose

This document defines the design of the Protocol Buffers schema for the PRIDE AI SDK.

It translates the requirements established by the:

* PRIDE AI SDK Contract.
* PRIDE AI SDK Protocol Specification.

This document defines the structure of the future `ai_sdk.proto` file.

It does not define the final generated code or AI SDK Server implementation.

---

# 2. Design Position

Protocol Buffers shall be the canonical schema used to define communication between the AI SDK Client and AI SDK Server.

The protocol shall provide one standardized request and response model regardless of:

* Programming language.
* AI provider.
* AI model.
* AI runtime.
* AI implementation.
* Internal execution mechanism.

The `.proto` file shall become the canonical protocol source after this design is approved.

---

# 3. Protocol Package

The initial protocol package shall be:

```text
pride.ai.v1
```

The version is part of the package namespace to allow future protocol evolution.

Future breaking protocol changes may introduce a new major version:

```text
pride.ai.v1
pride.ai.v2
```

---

# 4. Service Design

The initial AI SDK service shall expose one standardized execution operation.

```text
AIService

└── Execute
```

Conceptually:

```proto
service AIService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
}
```

The service shall not expose provider-specific operations.

The protocol shall not contain operations such as:

```text
OpenAIExecute
GeminiExecute
ClaudeExecute
OllamaExecute
```

Provider-specific behavior remains outside the protocol.

---

# 5. Execute Request

The request shall follow the standardized AI SDK request model:

```text
ExecuteRequest

├── request_id
├── capability
├── input
└── metadata
```

This structure follows the AI SDK Contract.

---

# 6. request_id

`request_id` identifies the AI request.

Design:

```proto
string request_id = 1;
```

Requirements:

* Required.
* Must not be empty.
* Must remain associated with the request throughout execution.
* Must be returned in the corresponding response or error.

The request ID shall not encode:

* Provider information.
* Model information.
* Runtime information.
* Credentials.
* Configuration secrets.

---

# 7. capability

`capability` identifies the AI capability requested by the consumer.

Design:

```proto
string capability = 2;
```

The capability field belongs to the AI SDK request model.

It does not represent the Core Capability Registry.

The AI SDK does not use this field to discover or select PRIDE Plugins.

The field identifies the AI capability that the AI SDK is being asked to execute.

Examples may include:

```text
reasoning
generation
embedding
structured_generation
```

The final capability vocabulary is defined outside this protocol design.

---

# 8. input

The request input shall be represented generically at the protocol layer.

Initial design:

```proto
bytes input = 3;
```

The reason for using a generic representation is that the AI SDK Contract defines the input as capability-specific.

For example:

```text
AIRequest
    │
    ├── capability = reasoning
    └── input = reasoning-specific input
```

or:

```text
AIRequest
    │
    ├── capability = generation
    └── input = generation-specific input
```

The protocol shall therefore avoid embedding every AI capability directly into the root request message.

The final encoding semantics of `input` must remain compatible with the applicable capability contract.

---

# 9. Request Metadata

The request shall support standardized execution metadata.

Design:

```proto
RequestMetadata metadata = 4;
```

Metadata may contain:

* Execution context identifiers.
* Task identifiers.
* Correlation information.
* Capability-specific execution metadata.

Metadata is not AI memory.

The AI SDK shall not use metadata as a replacement for the Memory SDK.

---

# 10. Execute Response

The response shall follow the AI SDK Contract:

```text
ExecuteResponse

├── request_id
├── capability
├── output
└── metadata
```

Conceptually:

```proto
message ExecuteResponse {
  string request_id = 1;
  string capability = 2;
  bytes output = 3;
  ResponseMetadata metadata = 4;
}
```

---

# 11. Response request_id

The response shall contain the `request_id` belonging to the originating request.

This allows the consumer to associate a response with its corresponding request.

---

# 12. Response capability

The response shall contain the capability that produced the result.

The returned capability must correspond to the requested capability.

The AI SDK shall not silently transform one requested capability into another capability.

---

# 13. output

The output shall be represented generically at the root protocol layer.

Initial design:

```proto
bytes output = 3;
```

The output structure is capability-specific.

Examples include:

```text
Conversation
      ↓
Conversation Output
```

```text
Embedding
      ↓
Embedding Output
```

```text
Structured Output
      ↓
Structured Output Result
```

The AI SDK shall expose standardized capability results rather than provider-specific response objects.

---

# 14. Response Metadata

The response shall support standardized execution metadata.

Design:

```proto
ResponseMetadata metadata = 4;
```

Provider-specific metadata shall not cross the protocol boundary unless explicitly defined by an applicable contract.

---

# 15. Error Design

A failed execution shall return a standardized AI SDK error model.

Logical structure:

```text
AIError

├── request_id
├── error_code
├── message
└── metadata
```

Conceptually:

```proto
message AIError {
  string request_id = 1;
  string error_code = 2;
  string message = 3;
  ErrorMetadata metadata = 4;
}
```

The exact mechanism for exposing the error through gRPC remains part of the implementation design.

---

# 16. Standard Error Categories

The initial standardized error categories are:

```text
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

Provider-specific errors shall be normalized into the applicable AI SDK error category.

---

# 17. Configuration Boundary

The protocol shall not expose the internal structure of AI Configuration.

The client does not select or modify an AI Configuration through provider-specific fields.

The execution flow is:

```text
AI SDK Client
      ↓
Execute Request
      ↓
AI SDK Server
      ↓
Active AI Configuration
      ↓
AI Execution
```

The selected AI Configuration determines the execution environment.

The client remains unaware of:

* Active provider.
* Active model.
* Runtime technology.
* Provider endpoint.
* Authentication mechanism.

---

# 18. Statelessness

The AI SDK protocol shall preserve the stateless nature of the AI SDK.

The protocol shall not define persistent conversation state.

The SDK does not own:

* Long-term memory.
* User history.
* Context storage.
* Conversation persistence.
* Session synchronization.

Context required for an AI operation may be supplied by higher-level platform components through the request input or standardized metadata.

---

# 19. Provider Independence

Provider-specific structures shall not be embedded into the protocol.

Prohibited design:

```text
AIRequest
├── openai_request
├── gemini_request
├── claude_request
└── ollama_request
```

Required direction:

```text
AIRequest
├── request_id
├── capability
├── input
└── metadata
```

The AI SDK Server is responsible for resolving the active AI Configuration and executing the request.

---

# 20. Versioning

The protocol shall follow Protocol Buffers compatibility rules.

The initial package is:

```text
pride.ai.v1
```

Field numbers shall remain stable after publication.

Removed fields shall not have their field numbers reused.

Reserved fields shall be used when required.

Breaking changes shall require a new major protocol version.

---

# 21. Initial Schema Draft

The current design translates into the following preliminary schema:

```proto
syntax = "proto3";

package pride.ai.v1;

service AIService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
}

message ExecuteRequest {
  string request_id = 1;
  string capability = 2;
  bytes input = 3;
  RequestMetadata metadata = 4;
}

message ExecuteResponse {
  string request_id = 1;
  string capability = 2;
  bytes output = 3;
  ResponseMetadata metadata = 4;
}

message AIError {
  string request_id = 1;
  string error_code = 2;
  string message = 3;
  ErrorMetadata metadata = 4;
}

message RequestMetadata {
}

message ResponseMetadata {
}

message ErrorMetadata {
}
```

This code is a **schema design draft**, not yet the canonical `ai_sdk.proto`.

---

# 22. Design Constraints

The final `.proto` must satisfy the following constraints:

1. Must conform to the AI SDK Contract.
2. Must conform to the AI SDK Protocol Specification.
3. Must remain provider-independent.
4. Must remain language-independent.
5. Must remain implementation-independent.
6. Must preserve AI SDK statelessness.
7. Must not expose AI Configuration internals.
8. Must not expose provider-specific structures.
9. Must support capability-specific input and output.
10. Must support standardized errors.
11. Must follow Protocol Buffers compatibility rules.
12. Must not introduce unnecessary protocol complexity.

---

# 23. Relationship With PRIDE Core

The AI SDK protocol shall not become part of the Core Capability Registry protocol.

The distinction is:

```text
PRIDE Capability Registry
        ↓
Plugin / Capability discovery
        ↓
Execution Plan
```

while:

```text
AI SDK
        ↓
AI operation request
        ↓
AI SDK Server
        ↓
AI Configuration
        ↓
AI execution
```

AI may be invoked by Core when AI assistance is required, including situations where additional reasoning may be needed to construct or improve an executable Execution Plan.

This does not make the AI SDK responsible for Capability discovery or Execution Plan construction.

---

# 24. Canonical Protocol Source

After approval, the AI SDK Protocol shall be represented by a canonical `.proto` schema.

The canonical schema shall:

* Define the official AI SDK gRPC service.
* Define the official Protocol Buffers messages.
* Remain the machine-readable representation of the AI SDK Protocol.
* Serve as the source from which protocol bindings may be generated.

The final repository location and directory structure of the `.proto` source are **not defined by this specification**.

Repository organization shall be determined by the PRIDE SDK project structure and implementation documentation.

Generated code shall remain derived from the canonical `.proto` schema and shall not become the source of truth for the protocol.

---

# 25. Generation Pipeline

After the schema is finalized:

```text
AI SDK Contract
      ↓
AI SDK Protocol Specification
      ↓
AI SDK Proto Design
      ↓
ai_sdk.proto
      ↓
protoc
      ↓
Generated Protocol Code
      ↓
AI SDK Server / Client SDK
```

The `.proto` file is therefore the transition point between architectural documentation and implementation.

---

# 26. Status

This document is currently:

**Draft**

The following items must be reviewed before `ai_sdk.proto` is locked:

* Final input representation.
* Final output representation.
* Metadata fields.
* Error transport semantics.
* Capability identifier format.
* Protocol field numbering.
* gRPC service options.
* Compatibility requirements.

No implementation should treat the preliminary schema in this document as the final protocol until those decisions are approved.
