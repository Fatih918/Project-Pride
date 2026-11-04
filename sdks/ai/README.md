# PRIDE AI SDK

The PRIDE AI SDK is the standardized gRPC communication boundary used by PRIDE AI Capabilities to request AI execution from the AI Module.

## Documentation

- [AI SDK Usage Guide](../DOCS/SDK%20Spesification/SDK%20AI/PRIDE%20AI%20SDK%20Usage%20Guide.md) — protocol usage, request/response structure, Go gRPC example, errors, context, and local development.
- [AI SDK Boundary Design](../DOCS/SDK%20Spesification/SDK%20AI/PRIDE%20AI%20SDK%20Boundary%20Design.md) — ownership, dependencies, request flow, configuration and credential boundaries.
- [AI SDK Specification](../../04-Platform%20Specification%20Standard/SDK%20Specification/PRIDE%20AI%20SDK%20Specification.md) — stable architectural specification.
- [AI SDK Contract](../DOCS/SDK%20Spesification/SDK%20AI/PRIDE%20AI%20SDK%20Contract.md) — draft logical contract.
- [AI SDK Protocol Specification](../DOCS/SDK%20Spesification/SDK%20AI/PRIDE%20AI%20SDK%20Protocol%20Specification.md) — draft gRPC/Protobuf protocol specification.
- [AI Module Specification](../../04-Platform%20Specification%20Standard/Module%20Specification/PRIDE%20AI%20Module%20Specification.md) — stable AI Module responsibilities.

## Canonical Protocol

The source of truth for the current wire protocol is `proto/ai_sdk.proto`:

- Protobuf package: `pride.ai.v1`
- Go package: `pride-ai-sdk/proto`
- gRPC service: `AIService`
- Current RPC: `Execute`

Generated Go bindings are derived from the canonical schema. Do not edit generated files manually.

## Current Scope

The current protocol exposes one unary execution RPC. The AI SDK Server forwards requests to a separate AI Module; it does not contain provider/runtime implementations or own AI Configuration and credentials.

See the usage and boundary documents for current implementation limitations and the known draft-document alignment items.
