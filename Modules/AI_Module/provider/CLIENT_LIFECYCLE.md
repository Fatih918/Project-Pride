# Provider Client Lifecycle Contract

## Scope

The provider client cache owns provider instances created for execution. It does
not own configuration storage, credential storage, selection, or execution
orchestration. The Executor remains provider-agnostic.

## Identity and credential changes

A cache key is created from the configuration reference, implementation
reference, connection parameters, and current credential bytes. The credential
is included only in an HMAC fingerprint made with a random, process-local key.
Raw credentials and fingerprints must never be logged or copied into request
metadata, public errors, or configuration fields.

The cache key is opaque and cache-scoped. A cache restart changes the HMAC key,
so keys are not stable identifiers and must not be persisted.

The current credential resolver returns only secret bytes; it does not expose a
version number. Therefore the effective secret value is fingerprinted on each
key construction. If a future credential store exposes a monotonically
increasing version, that version can supplement the fingerprint, but
CredentialRef alone is not sufficient to detect rotation.

## Isolation and sharing

Clients are isolated by configuration reference by default. Two configurations
may reference the same credential, but they do not share a client merely because
their CredentialRef values match. Sharing across configurations requires a
separate explicit policy and proof that every connection parameter, credential
identity, provider option, and lifecycle requirement is compatible.

## Updates, invalidation, and rotation

Changes to the effective credential or connection parameters create a different
key and therefore a new client. The configuration owner must invalidate the
configuration after updates and deletion; credential updates must invalidate
every configuration that depends on that credential. The cache itself does not
subscribe to storage events and cannot infer deletion or rotation from a
reference alone.

Invalidation removes retired clients from future acquisitions immediately.
Existing leases remain valid until released; a provider implementing io.Closer
is closed after its final active lease is released. A client still being built
when invalidation occurs is discarded rather than published.

## Concurrency and ownership

Acquisition is concurrency-safe and coalesces simultaneous builds for the same
key. Every successful acquisition returns a lease that must be released exactly
once (the implementation makes repeated release safe). Invalidation and cache
shutdown do not close a provider while active leases exist.

Providers that do not implement io.Closer can still be cached, but the cache
cannot explicitly release their underlying resources. Provider adapters must
not retain a lease beyond the execution that acquired it unless ownership is
deliberately transferred.

## Current limitations

- The cache is a lifecycle primitive; wiring it into a real provider adapter is
  a separate integration step.
- The cache does not observe database changes. Configuration update/delete and
  credential rotation paths must call invalidation explicitly.
- The Gemini SDK wrapper currently exposes the provider execution interface;
  whether its concrete client can be closed must be verified against the SDK
  version before claiming deterministic underlying connection cleanup.
- Cache keys are generated in memory and are not intended for persistence.
