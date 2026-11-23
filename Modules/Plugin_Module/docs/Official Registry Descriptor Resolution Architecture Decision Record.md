# Official Registry Descriptor Resolution — Architecture Decision Record

**Status:** Accepted for implementation
**Date:** 2026-11-23
**Scope:** Provider-neutral Registry release resolution for Plugin installation.
**Supersedes:** The proposed options in [Official Registry API and Descriptor Resolution Decision Review](Official%20Registry%20API%20and%20Descriptor%20Resolution%20Decision%20Review.md).
**Related:** PTS-0004, PZIP-0001, Plugin Module Production Host Integration Contract.

## Context

The Plugin installation application service accepts an opaque source reference. The existing `DescriptorResolver` port resolves that reference into exact signed descriptor bytes, while the artifact acquirer downloads the immutable ZIP from a separately allowlisted artifact host.

The Registry transport boundary is not implemented. Before adding it, PRIDE needs a stable source-reference grammar, a host-owned authentication boundary, and explicit transport/response semantics. This record adopts the recommended options from the decision review without inventing a deployed Registry hostname, identity provider, credential, signing key, or service endpoint.

## Decisions

### R1 — Source reference

Adopt a versioned, provider-neutral Registry release reference containing:
- Registry namespace;
- Plugin ID;
- immutable release ID.

The reference is an identifier, never a URL. It must not contain a hostname, arbitrary path, query credential, token, or caller-selected endpoint. Mutable aliases such as `latest`, branches, and unqualified tags are not valid immutable release IDs. The exact textual grammar will be defined by the parser contract before implementation.

### R2 — Registry endpoint configuration

The trusted host configures one HTTPS API origin per Registry namespace. Unknown namespaces are rejected. Registry API origins and artifact download hosts are separate allowlists and trust surfaces.

### R3 — Descriptor lookup operation

Adopt a versioned, read-only descriptor lookup operation with the logical route:

`GET /v1/plugins/{plugin_id}/releases/{release_id}/descriptor`

This is a contract template, not a deployed URL. IDs are encoded as individual path segments exactly once. Encoded separators, dot segments, empty segments, and ambiguous normalization are rejected.

### R4 — Authentication ownership

The host supplies a short-lived, least-privilege bearer token. The Plugin Module does not implement an OAuth login/refresh flow and does not own long-lived client secrets. No anonymous fallback is permitted.

### R5 — Token provider boundary

The HTTP Registry adapter depends on a host-owned token-provider port that returns an audience-scoped token and its expiry. Identity-provider selection, token refresh, client identity, and secret storage remain host responsibilities.

### R6 — Retry behavior

The Module performs no hidden retries. It returns typed errors for retryable rate-limit and transient server failures; retry orchestration belongs to the host. Authentication, authorization, not-found, conflict, and malformed-response failures are not retryable by default.

### R7 — Transport security

The Registry client:
- requires HTTPS and TLS 1.2 or newer with normal certificate and hostname verification;
- uses only the host-configured exact Registry origin;
- disables redirects and environment proxy inheritance;
- rejects caller-controlled hosts, URL user-info, and credential forwarding to artifact hosts;
- validates resolved destination addresses at connection time and rejects non-public destinations;
- applies bounded request timeout, response headers, and body size;
- rejects non-identity content encoding unless a future contract defines bounded decompression and byte-preservation semantics.

The Registry transport policy is independent from the artifact downloader's policy.

### R8 — Response representation

A successful response is `200 application/json`, and the response body is the exact raw descriptor JSON, bounded to 1 MiB. The API must not wrap the descriptor in an envelope. The client must preserve these bytes unchanged for signature verification and durable recovery evidence.

### R9 — API versioning

The Registry route uses an explicit `/v1` API version. Descriptor schema versioning remains independent and follows PTS-0004.

### R10 — Error contract

The client maps HTTP status classes into typed, safe application errors:
- 400 invalid request/configuration;
- 401 authentication failure;
- 403 authorization denied;
- 404 immutable release/descriptor not found;
- 409 immutable release conflict;
- 429 rate limited, with a bounded retry hint where valid;
- 5xx/network timeout transient Registry unavailability;
- 3xx rejected without following;
- all other statuses unexpected Registry response.

Only a bounded request/correlation ID may be retained from an error response. Arbitrary response bodies, authorization headers, and credentials must not enter logs or returned errors.

## Descriptor identity and trust

Before artifact download, the resolver must bind the returned descriptor to the requested Registry namespace, Plugin ID, immutable release ID, and expected version. Identity mismatch is a hard failure.

The resolver validates HTTP representation, descriptor syntax/schema, and request identity. The existing artifact verifier remains authoritative for Ed25519 signature, trust configuration, and SHA-256 digest verification. TLS and bearer authentication do not replace Marketplace signature verification.

The exact raw descriptor bytes must pass unchanged from resolver to verifier and durable installation journal. No normalization, re-serialization, repair, or unsigned fallback is allowed.

## Go boundary

Keep the application-facing port SDK-neutral. The current `DescriptorResolver.ResolveDescriptor(context.Context, string) ([]byte, error)` shape may be retained only if the source reference carries all expected identity fields unambiguously. Otherwise, replace it with a typed resolution request before implementing the HTTP adapter. The HTTP adapter receives endpoint configuration, token provider, bounded transport, and parser dependencies; it does not access Plugin storage or SQLite.

## Consequences

- The official Registry hostname, namespace mapping, token issuer, and credential source remain deployment configuration and are not hardcoded in the Module.
- Registry descriptor resolution and artifact download remain separate components.
- The first implementation is limited to immutable descriptor lookup; search, browse, publication, and Marketplace administration are out of scope.
- Production installation remains disabled until the Registry resolver, live trust service, real bootstrap public key provisioning, and production host composition are wired and validated.

## Implementation acceptance criteria

1. Strict parser and canonical representation for the versioned Registry release reference.
2. Typed resolution request carrying expected Plugin ID, version, and immutable release ID where not fully encoded in the reference.
3. Host-configured namespace-to-origin mapping and token-provider port.
4. Hardened HTTPS resolver with redirects/proxies disabled and bounded response handling.
5. Typed error mapping and context cancellation propagation.
6. Exact raw descriptor preservation and identity checks before artifact download.
7. Tests for authn/authz, not found/conflict, 429/5xx, redirects, wrong origin, private/mixed DNS answers, malformed/duplicate/oversized JSON, identity mismatch, timeout/cancellation, and credential non-disclosure.
8. No changes to Marketplace Ed25519 signing semantics or public SDK lifecycle APIs.
