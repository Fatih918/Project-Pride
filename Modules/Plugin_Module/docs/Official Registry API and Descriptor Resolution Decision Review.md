# Official Registry API and Descriptor Resolution — Decision Review

**Status:** Proposed for review; no production Registry endpoint or authentication provider is approved by this document.  
**Scope:** Plugin Module descriptor resolution boundary and the minimum authenticated Registry API contract.  
**Related:** PTS-0004, PZIP-0001, Plugin Module Production Host Integration Contract, installation/acquisition.

## 1. Problem

The installation pipeline accepts an opaque `SourceReference`. The current `DescriptorResolver` port resolves it to raw signed descriptor JSON, after which the artifact acquirer downloads the descriptor's immutable HTTPS artifact URI.

The transport boundary is intentionally incomplete: there is no approved Official Registry API origin, source-reference grammar, authentication mechanism, or HTTP response contract. The host integration contract correctly lists these as production blockers.

This review proposes a provider-neutral contract. It does not invent a deployed Registry hostname, API credential, OAuth issuer, Marketplace implementation, or signing key.

## 2. Invariants

1. Registry resolution returns the exact descriptor bytes published by the trusted Registry. The client MUST NOT synthesize, normalize, re-sign, or silently repair descriptor content.
2. The returned descriptor MUST satisfy the existing PTS-0004 schema and signature contract. The Marketplace descriptor signature remains the installation attestation; TLS and Registry authentication do not replace Ed25519 verification.
3. The resolved descriptor identity MUST match the requested Plugin ID, version, and immutable release identity. A mismatch is a hard failure before artifact download.
4. The Registry API origin and credentials are host configuration. They MUST NOT be supplied by an untrusted Plugin manifest or arbitrary source reference.
5. The Registry resolver MUST be separate from the artifact downloader. Registry API hosts and artifact download hosts are distinct trust/configuration surfaces.
6. Registry credentials MUST never be persisted in installation records, descriptor JSON, logs, error messages, or Plugin metadata.
7. The SDK-neutral application layer MUST depend on a narrow resolver port, not HTTP, OAuth, provider SDKs, or generated transport types.
8. All security-sensitive failures are fail-closed. A Registry lookup failure cannot be treated as an empty descriptor or trigger an unsigned fallback.

## 3. Proposed source-reference model

**Recommendation:** define a versioned, provider-neutral Registry release reference that identifies a Registry namespace, Plugin ID, and immutable release ID. The reference is an identifier, not a URL.

The host configures the trusted Registry API origin and maps the Registry namespace to that origin. The resolver rejects unknown namespaces. The source reference MUST NOT carry a hostname, URL, query credentials, access token, or arbitrary path.

The release ID MUST be immutable and MUST NOT be a mutable channel alias such as `latest`, a branch name, or an unqualified tag. Version remains an explicit expected identity and is checked against the returned descriptor.

The exact serialized grammar is a decision below. Until approved, the current source-reference string remains opaque and no parser is added.

## 4. Proposed logical API contract

The client performs one idempotent descriptor lookup for a known immutable release. The logical operation is:

- Input: configured Registry namespace, Plugin ID, immutable release ID.
- Output: exact raw JSON bytes for one PTS-0004 Registry artifact descriptor.
- The response descriptor's `plugin_id`, `version`, and `release_id` MUST match the resolved request context and installation request.
- The Registry stores descriptor metadata only; ZIP bytes remain on the separately allowlisted artifact host.

**Recommended wire shape for review (not an approved deployed endpoint):**

`GET /v1/plugins/{plugin_id}/releases/{release_id}/descriptor`

The route is versioned and read-only. IDs are path-segment encoded once; encoded separators, dot segments, empty segments, and ambiguous normalization are rejected. No client-supplied absolute URL is accepted.

The successful response is `200 application/json`, with a bounded body containing the exact descriptor JSON. The response MUST NOT wrap the descriptor in an envelope, because the exact signed descriptor bytes are persisted for recovery. If an envelope is required by the eventual Registry implementation, its extraction and byte-preservation rules must be separately specified before implementation.

## 5. Authentication and transport proposal

**Recommendation:** the Plugin Module receives a short-lived, least-privilege bearer token from a host-owned credential provider. The Module does not implement an OAuth authorization flow and does not own long-lived client secrets.

The credential provider contract should return a token scoped to descriptor read access for the configured Registry audience. Token acquisition/refresh, client identity, secret storage, and operator login remain responsibilities of the PRIDE host / credential subsystem.

Transport requirements:

- HTTPS only; TLS 1.2 minimum, with normal certificate and hostname verification.
- Registry origin is configured by the trusted host and validated at startup.
- No redirects, including same-origin redirects.
- No URL user-info, caller-controlled host, proxy inheritance, or credential forwarding to another origin.
- Send only the required authorization and representation headers; never send credentials to the artifact download host.
- Bounded request timeout, response-header size, and descriptor body size (maximum 1 MiB, consistent with the persisted descriptor limit).
- Reject non-identity content encoding unless decompression limits and byte semantics are explicitly specified.
- No automatic retry for authentication, authorization, validation, or not-found responses. Any retry for rate-limit or transient server failures must be bounded, context-aware, and preserve the same immutable request.
- Do not log bearer tokens, authorization headers, raw credential-provider errors, or full sensitive response bodies.

DNS/IP restrictions and exact Registry host allowlisting must be applied to the Registry transport independently of the artifact host allowlist. The implementation must defend against private/loopback/link-local destinations and DNS rebinding using connection-time address validation, consistent with the artifact acquisition boundary.

## 6. Response and error semantics proposal

| HTTP outcome | Resolver behavior |
|---|---|
| 200 | Validate content type, size, JSON syntax, duplicate keys, schema/version, and request identity; return exact body bytes |
| 400 | Permanent invalid request/configuration error |
| 401 | Authentication failure; no anonymous fallback |
| 403 | Registry authorization denied |
| 404 | Immutable release/descriptor not found |
| 409 | Registry release state conflict; permanent until caller changes request |
| 429 | Rate limited; expose typed retryable error and bounded Retry-After hint |
| 5xx / network timeout | Transient Registry unavailable error |
| 3xx | Reject; do not follow |
| Other status | Typed unexpected Registry response |

Errors exposed to the application layer should be typed and safe for logs. They may include status class and a correlation/request ID, but MUST NOT include credentials or arbitrary response-body text.

The resolver should not retry 401/403/404/409 or malformed responses. Retry policy for 429/5xx is a host policy; the first implementation should prefer returning a typed retryable error rather than hiding repeated requests inside the Module.

## 7. Descriptor validation boundary

The existing acquisition adapter currently checks descriptor schema identifiers, release ID, media type, size, HTTPS URL, and artifact-host allowlist before downloading. The complete contract also requires strict validation of:

- required top-level fields and supported schema version;
- Plugin ID, version, and release ID against the resolved request;
- SHA-256 digest syntax and Ed25519 signature envelope syntax;
- duplicate JSON key rejection;
- maximum descriptor size;
- unknown-field forward compatibility as defined by PTS-0004.

Cryptographic verification remains in the existing artifact verifier against the authenticated trust configuration. Registry TLS authentication is not a substitute for descriptor signature verification.

The resolver may validate the HTTP representation and request identity, but the artifact verifier remains authoritative for signature and trust decisions. The same exact raw descriptor bytes must flow to the verifier and durable operation journal.

## 8. Proposed Go boundary

The application-facing boundary remains SDK-neutral. A likely port shape is:

```go
type DescriptorResolver interface {
    ResolveDescriptor(
        ctx context.Context,
        sourceReference string,
    ) ([]byte, error)
}
```

The production HTTP adapter receives:
- immutable Registry reference parser;
- trusted Registry endpoint configuration;
- host-owned credential/token provider;
- bounded HTTP client/transport;
- no direct access to Plugin storage or SQLite.

The current port can remain unchanged if the source reference fully and unambiguously carries the expected Plugin ID/version/release identity. If it does not, the port should be extended to accept an explicit typed resolution request so descriptor identity can be checked before download. This must be settled before implementation; do not infer identity from untrusted descriptor bytes alone.

## 9. Decisions required

| ID | Decision | Recommended option | Alternatives / consequence |
|---|---|---|---|
| R1 | Source-reference representation | Typed provider-neutral Registry release reference (namespace + Plugin ID + immutable release ID); identifier only, never URL | Opaque Registry-issued token (less parseable but requires server-side resolution); raw URL (rejected due to origin/SSRF and credential-routing risk) |
| R2 | Registry endpoint model | One host-configured HTTPS origin per Registry namespace | One global origin only; multiple provider adapters (more complexity and provider coupling) |
| R3 | API operation | Versioned read-only descriptor lookup route as proposed above | Opaque lookup endpoint; requires a different request/response contract |
| R4 | Authentication ownership | Host-provided short-lived scoped bearer token; Module owns no OAuth flow or long-lived secret | mTLS (requires host certificate lifecycle); static API key (not recommended) |
| R5 | Token provider interface | Host-owned `RegistryTokenProvider` returning audience-scoped token with expiry | Direct integration with a specific identity provider (premature without an approved provider) |
| R6 | Retry behavior | No hidden retries in Module; typed retryable 429/5xx errors for host policy | Bounded adapter retries (adds latency and operational behavior that must be standardized) |
| R7 | Registry transport policy | No redirects/proxies; TLS 1.2+; exact configured host allowlist; connection-time public-IP validation; bounded timeout/headers/body | Redirect allowlist (more complex credential and host-boundary rules) |
| R8 | Response representation | Raw descriptor JSON as the entire 200 response body, max 1 MiB | Envelope response (requires canonical extraction/persistence rules) |
| R9 | API compatibility | Explicit `/v1` route and descriptor schema version independent of API version | Versioned media type or negotiated API version |
| R10 | Registry error contract | HTTP status classes plus optional request ID; no free-form body propagated to callers | Structured machine error body (requires a versioned error schema) |

## 10. Explicit non-goals

This review does not approve or implement:
- a real Registry hostname, DNS name, endpoint, account, or provider;
- OAuth issuer, client ID, client secret, static API key, or mTLS certificate;
- Marketplace signing service or private signing key;
- bootstrap public keys or trust-configuration deployment;
- artifact hosting or artifact download URL generation;
- search/list/browse APIs, pagination, publication workflows, or Marketplace administration;
- production host composition or enabling Plugin installation.

## 11. Acceptance criteria for the implementation phase

After decisions R1–R10 are approved, implementation must include:

1. strict source-reference parser and canonical representation;
2. host-configured Registry endpoint and credential-provider ports;
3. hardened HTTP resolver with bounded response handling and no redirect/proxy behavior;
4. typed error mapping and cancellation propagation;
5. exact raw descriptor preservation;
6. descriptor request-identity checks before artifact download;
7. tests for authentication, authorization, 404, 429, 5xx, redirects, malformed/oversized/duplicate-key JSON, identity mismatch, timeout/cancellation, DNS/private-IP rejection, and credential non-disclosure;
8. no changes to Marketplace Ed25519 signing semantics or SDK public lifecycle APIs.

Until the Registry API and authentication contract are approved and implemented, production installation remains disabled.
