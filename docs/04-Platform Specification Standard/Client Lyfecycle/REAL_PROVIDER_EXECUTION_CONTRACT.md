# Real Provider Execution Contract

## Purpose

This contract defines the boundary for roadmap 13.11 real-provider execution. It is a design and test contract; it does not by itself claim that a live external-provider end-to-end request has been completed.

## Execution flow

The supported flow is:

1. Receive an SDK execution request with a request ID, capability, input, and pass-through metadata.
2. Resolve the selected configuration through the Core/AI Module configuration boundary using only its `ConfigurationRef`.
3. Validate the resolved configuration and capability before constructing provider resources.
4. Resolve the optional `CredentialRef` through the credential boundary. Credential bytes remain internal to the Module and provider adapter.
5. Build the effective provider-client key from configuration identity, implementation identity, non-secret connection parameters, and the current credential value. Do not use the raw credential as a key, log field, metadata value, or error detail.
6. Acquire a provider client lease from `ClientCache`. The adapter owns the lease for the duration of one execution and must release it on every return path.
7. Translate the internal request to the provider API, invoke the provider, and map its result to the stable internal response.
8. Normalize provider failures to the existing internal error contract, preserving the request ID without exposing provider secrets or raw SDK errors.
9. Release the lease. Retired clients may be closed only after their active leases have drained.

## Boundary requirements

- The Executor remains provider-agnostic; it must not contain provider-specific SDK calls or credential logic.
- Configuration selection stores and exposes only `ConfigurationRef`; it must not expose provider, implementation, or credential internals.
- The SDK ↔ Module boundary remains gRPC/Protobuf. The SDK must not import Module implementation packages.
- `ConnectionParameters` may contain non-secret model, endpoint, region, or deployment settings. Secrets belong in the credential store, not configuration parameters.
- Metadata is pass-through execution context only. It must never be treated as a credential channel.
- Credential resolution and provider-client construction errors must be normalized. Never return raw credentials, authorization headers, or secret-bearing request bodies.
- A provider adapter must not retain a lease beyond the execution unless ownership is explicitly transferred.
- Configuration changes, deletion, and credential rotation must invalidate all affected configuration entries. A changed effective credential or connection parameter must not reuse an incompatible client.
- In-flight executions may finish with their existing lease during invalidation; new executions must not acquire retired entries.

## Capability and configuration validation

Before calling an external provider, validate that:

- the configuration reference resolves to an existing configuration;
- the configuration is active/selected according to the existing configuration contract;
- the requested capability is supported by the chosen implementation;
- required non-secret connection parameters are present and well-formed;
- any required credential is available and non-empty;
- the input shape is supported by the selected capability.

Validation failures must not trigger an external provider request. Missing or malformed configuration maps to the existing `CONFIGURATION_ERROR` contract; missing or rejected credentials map to `AUTHENTICATION_ERROR` where the provider response supports that conclusion.

## Failure normalization

Provider adapters must map failures to stable internal categories rather than leaking SDK-specific error types:

- authentication/authorization rejection → `AUTHENTICATION_ERROR`;
- invalid or unsupported configuration/model/endpoint → `CONFIGURATION_ERROR`;
- provider throttling or transient availability failures → the existing retryable/provider-failure category, without inventing a new public category in this step;
- timeout/cancellation → preserve the caller's context semantics and request ID;
- malformed provider response or response mapping failure → the existing internal/provider execution error category.

Do not classify every HTTP 4xx response as an authentication error. Use provider error codes/statuses where reliable; otherwise return a safe generic provider failure.

## Required verification before claiming real-provider E2E

1. Unit tests prove configuration, capability, and credential validation occurs before provider invocation.
2. Adapter tests prove request/response mapping and normalized errors using a deterministic fake provider.
3. Cache tests prove effective-key isolation, single construction for concurrent acquisition, lease release on success and failure, and deferred close during invalidation.
4. Module-level gRPC tests prove request ID and error normalization survive the transport boundary.
5. A live-provider smoke test, explicitly gated by environment credentials, proves a successful request against the configured provider without printing or storing the secret.
6. Invalid credential and invalid configuration tests prove the expected normalized categories and confirm that no secret is included in logs or returned errors.

Fake-provider tests and internal gRPC E2E tests do not count as live-provider E2E. Report live-provider verification separately and only when it was actually run.
