# PRIDE AI SDK Usage Guide

**Version:** 1.0.0  
**Status:** Draft  
**Category:** SDK Usage Documentation  
**Document ID:** PRIDE-AI-SDK-USAGE  
**Canonical Protocol:** `sdks/ai/proto/ai_sdk.proto`

---

## 1. Purpose

This guide explains how a PRIDE component or AI Capability sends an AI execution request through the PRIDE AI SDK.

The current implementation is a Go-based AI SDK Server exposing a language-neutral gRPC API. Consumers communicate with the generated `AIService` client contract. The server forwards requests across its internal AI Module client boundary.

This guide describes the protocol and implementation that currently exist. It does not imply that every capability, provider, or production deployment path has already been implemented.

## 2. What the AI SDK Provides

The AI SDK provides one standardized execution operation:

```proto
service AIService {
  rpc Execute(ExecuteRequest) returns (ExecuteResponse);
}
```

A consumer submits:
- a request identifier;
- a capability identifier;
- capability-specific input;
- optional request metadata.

A successful call returns:
- the originating request identifier;
- the capability identifier;
- capability-specific output;
- optional response metadata.

The SDK contract does not expose provider-specific request objects, provider selection, model selection, endpoint configuration, or credentials.

## 3. Repository Layout

The current AI SDK implementation is organized as follows:

```text
sdks/ai/
├── go.mod
├── go.sum
├── proto/
│   ├── ai_sdk.proto
│   ├── ai_sdk.pb.go
│   └── ai_sdk_grpc.pb.go
└── server/
    ├── cmd/
    │   ├── ai-sdk-server/
    │   └── ai-sdk-test-client/
    └── internal/
        ├── contracts/
        │   ├── execution/
        │   └── module/
        ├── execution/
        ├── service/ai/
        └── transport/grpc/
```

The authoritative wire contract is `proto/ai_sdk.proto`. Generated Go files are derived artifacts and must not be edited manually.

The server's `internal` packages are implementation details. They are not public Go library packages for downstream consumers.

## 4. Connection Model

The current topology is:

```text
AI Capability / Platform Component
              |
              | gRPC: AIService.Execute
              v
       PRIDE AI SDK Server
              |
              | internal AIClient boundary
              v
        AI Module gRPC API
              |
              v
       Active AI Configuration
              |
              v
       AI Implementation
              |
              v
     Remote AI System / Local Runtime
```

The consumer connects to the AI SDK Server, not directly to a provider or AI Implementation.

The AI SDK Server's configured downstream address is currently read from `PRIDE_AI_MODULE_ADDR` (default `localhost:50052`). Its listening address is read from `PRIDE_AI_SDK_ADDR` (default `:50051`).

These defaults are development bootstrap values, not production deployment recommendations.

## 5. Protocol Messages

### 5.1 ExecuteRequest

The current schema is:

| Field | Type | Required by current server validation | Meaning |
|---|---|---:|---|
| `request_id` | `string` | Yes, non-empty | Correlates one execution request |
| `capability` | `string` | Yes, non-empty | Identifies the requested AI capability |
| `input` | `google.protobuf.Any` | Yes, non-nil | Carries capability-specific input |
| `metadata` | `google.protobuf.Struct` | No | Optional structured execution metadata |

The protocol uses a generic `Any` envelope so the root request does not need provider-specific fields or a new root message for every capability.

The payload type and semantics must be defined by the relevant capability contract. A non-nil `Any` alone does not prove that its payload is valid for the requested capability.

### 5.2 ExecuteResponse

| Field | Type | Meaning |
|---|---|---|
| `request_id` | `string` | Request identifier returned by the execution path |
| `capability` | `string` | Capability associated with the result |
| `output` | `google.protobuf.Any` | Capability-specific result |
| `metadata` | `google.protobuf.Struct` | Optional standardized response metadata |

Consumers should correlate the response with the originating request and validate/unpack `output` using the output type defined by that capability's contract.

### 5.3 Metadata

Metadata is a structured map for request or execution context, such as task identifiers or correlation attributes where the platform contract permits them.

Metadata is not persistent memory, conversation history, or a substitute for the Memory SDK. Do not put credentials, access tokens, or provider secrets in metadata.

## 6. Go gRPC Client Example

The following is a protocol-level example using the generated Go bindings. It is suitable for a Go component that imports the generated protocol package; it does not import the server's `internal` packages.

```go
package main

import (
    "context"
    "log"
    "time"

    aiv1 "pride-ai-sdk/proto"

    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/protobuf/types/known/anypb"
    "google.golang.org/protobuf/types/known/structpb"
)

func main() {
    // Insecure transport is shown only for local development.
    conn, err := grpc.NewClient(
        "localhost:50051",
        grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer conn.Close()

    client := aiv1.NewAIServiceClient(conn)

    input, err := anypb.New(&aiv1.TextGenerateInput{
        Prompt: "Explain the purpose of the PRIDE AI SDK.",
    })
    if err != nil {
        log.Fatal(err)
    }

    metadata, err := structpb.NewStruct(map[string]any{
        "source": "example-component",
    })
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
    defer cancel()

    response, err := client.Execute(ctx, &aiv1.ExecuteRequest{
        RequestId:  "request-unique-001",
        Capability: "text.generate",
        Input:      input,
        Metadata:   metadata,
    })
    if err != nil {
        // Inspect the gRPC status and any structured AIError details.
        log.Fatal(err)
    }

    if response.GetRequestId() != "request-unique-001" {
        log.Fatal("response request_id does not match request")
    }

    var output aiv1.TextGenerateOutput
    if err := response.GetOutput().UnmarshalTo(&output); err != nil {
        log.Fatal(err)
    }

    log.Println(output.GetText())
}
```

The `TextGenerateInput` and `TextGenerateOutput` messages are the canonical text-generation payload examples currently declared in `ai_sdk.proto`. The example assumes the downstream AI Module implements and accepts the `text.generate` capability. The SDK repository alone does not establish that a working provider implementation is available.

For a component written in another language, generate a client from the same canonical `.proto` schema. The service and message contract remain the same.

## 7. Request Construction Rules

Before calling `Execute`:

1. Generate a non-empty request ID. It should be unique within the applicable execution/correlation scope.
2. Set the capability identifier defined by the AI Capability contract.
3. Pack the capability-specific input into `google.protobuf.Any`.
4. Include only permitted, non-secret execution metadata.
5. Set a deadline on the gRPC context appropriate to the operation.

The current server rejects a nil request, an empty request ID, an empty capability, or a nil input. Capability-specific payload validation is the responsibility of the applicable capability contract and downstream execution path; the current generic envelope validation does not fully validate arbitrary `Any` payloads.

## 8. Error Handling

The protocol declares an `AIError` message with:
- `request_id`;
- `error_code`;
- `message`;
- `metadata`.

The gRPC call reports failures through gRPC status. The AI SDK's internal gRPC client inspects status details for an `AIError`. When such a detail is present, its fields are preserved in the SDK's `RemoteError`. If no structured detail is present, the client maps the gRPC status code to a fallback PRIDE error category.

Current fallback mappings in `server/internal/transport/grpc/client.go`:

| gRPC status | Fallback PRIDE error code |
|---|---|
| `InvalidArgument` | `INVALID_REQUEST` |
| `Unauthenticated` | `AUTHENTICATION_ERROR` |
| `PermissionDenied` | `PERMISSION_ERROR` |
| `Unavailable` | `CONNECTION_ERROR` |
| `DeadlineExceeded` | `TIMEOUT_ERROR` |
| `ResourceExhausted` | `RATE_LIMIT` |
| `Internal` | `RUNTIME_ERROR` |
| Other statuses | `UNKNOWN_ERROR` |

Use the standardized error code for programmatic handling. Treat the message as human-readable diagnostic context, not as a stable machine-readable identifier.

The current implementation does not yet demonstrate a complete server-side mapping layer that packs every internal error into an `AIError` status detail. Do not assume every failure will already contain structured details.

## 9. Context, Deadlines, and Cancellation

The server passes the incoming Go `context.Context` through its execution layer to the internal AI Module client. Consumers should set deadlines and cancel contexts when work is no longer needed.

A deadline exceeded or canceled request may terminate the transport operation. The exact cancellation behavior of the downstream AI implementation depends on the AI Module and its implementation adapter.

## 10. Configuration and Credentials

Consumers do not send provider names, model names, endpoints, API keys, or credential references in the normal execution request.

The AI Module resolves the active AI Configuration and its associated implementation according to platform rules. Credential resolution remains behind the platform credential boundary. Raw credential material must not cross the AI SDK consumer boundary.

The SDK does not silently switch to another configuration or implementation when execution fails.

## 11. Stateless Execution

The AI SDK does not own persistent conversation state, user history, long-term memory, or session storage.

A consumer supplies the context required for an individual operation through the capability input and permitted metadata. Persistent memory remains the responsibility of the Memory architecture and its SDK boundary.

## 12. Running the Local Server

From the repository:

```powershell
cd sdks/ai
go test ./...
go vet ./...
```

Start the server from the module directory:

```powershell
$env:PRIDE_AI_MODULE_ADDR = "localhost:50052"
$env:PRIDE_AI_SDK_ADDR = ":50051"
go run ./server/cmd/ai-sdk-server
```

The server expects an AI Module gRPC service at the configured downstream address. Starting the SDK server does not start or create an AI Module implementation.

The repository also contains a test client under `server/cmd/ai-sdk-test-client`. It is a protocol smoke-test utility and uses a generic test capability; it is not a substitute for a real AI Module integration test.

## 13. Transport Security

The current bootstrap code uses insecure gRPC transport credentials for its downstream connection, and the test client also uses insecure credentials. This is appropriate only for isolated local development.

A production deployment must define and configure transport security, service identity, authentication, authorization, network exposure, and secret handling at the deployment/integration boundary. The current bootstrap code must not be interpreted as providing those production controls.

## 14. Current Implementation Limitations

The following limitations are important when integrating the current repository state:

- The public wire contract currently exposes one unary RPC: `AIService.Execute`.
- There is no documented, dedicated public Go convenience-client package in the current `sdks/ai` tree; consumers can use the generated Protobuf gRPC client.
- The server's internal packages are intentionally under Go's `internal` visibility boundary and should not be imported by external modules.
- The SDK server forwards to an AI Module; it does not itself contain a production provider/runtime implementation.
- A generic `Any` input/output envelope requires capability-specific type contracts.
- Streaming is not part of the current unary RPC.
- The repository's server bootstrap is a minimal development composition and does not establish production transport security.

## 15. Related Documents

- [PRIDE AI SDK Specification](../../../04-Platform%20Specification%20Standard/SDK%20Specification/PRIDE%20AI%20SDK%20Specification.md)
- [PRIDE AI SDK Contract](PRIDE%20AI%20SDK%20Contract.md)
- [PRIDE AI SDK Protocol Specification](PRIDE%20AI%20SDK%20Protocol%20Specification.md)
- [PRIDE AI SDK Boundary Design](PRIDE%20AI%20SDK%20Boundary%20Design.md)
- [PRIDE AI Module Specification](../../../04-Platform%20Specification%20Standard/Module%20Specification/PRIDE%20AI%20Module%20Specification.md)
