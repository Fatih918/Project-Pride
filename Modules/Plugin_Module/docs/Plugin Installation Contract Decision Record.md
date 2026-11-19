# Plugin Installation Contract Decision Record

**Version:** 1.0.0  
**Status:** Accepted  
**Document ID:** DCR-0001  
**Supersedes:** Decision recommendations in PDR-0001  
**Related Specifications:** PTS-0001, PTS-0002, PTS-0003, PTS-0004  
**Module:** PRIDE Plugin Module

## 1. Accepted decisions

| Decision | Accepted choice |
|---|---|
| Path contract | Relative paths in application/persistence; storage adapter resolves absolute OS paths |
| D1 Artifact descriptor | Versioned provider-neutral Marketplace artifact descriptor |
| D2 Package format | ZIP |
| D3 Integrity/authenticity | SHA-256 plus Marketplace signature |
| Signature algorithm | Ed25519 |
| Trust root | Embedded bootstrap public key plus authenticated remote trust configuration |
| D4 JSON naming | snake_case |
| Schema versioning | Both schema_version and $schema |
| Unknown Manifest properties | Forward-compatible |

## 2. Required interpretation

Persisted artifact paths are canonical slash-separated paths relative to Plugin_Storage, for example temporary/<operation-id> and installed/<plugin-id>/<version>. Application/domain packages MUST NOT persist or construct absolute OS paths. Only the storage adapter resolves relative paths against the configured PRIDE user-data root. Resolution enforces namespace allowlists, traversal rejection, and symlink containment. Process working directory MUST NOT influence persisted paths.

The installation application service MUST request exclusive operation staging-directory creation through an explicit storage port before acquisition begins.

The Marketplace publishes a versioned provider-neutral descriptor referencing immutable external artifact bytes. Registry Repository continues to store Registry documents only. ZIP is the only package format in version 1. SHA-256 and Ed25519 are mandatory. Trust is rooted in embedded public bootstrap key material and authenticated remote trust configuration.

The developer-owned Manifest uses snake_case, both schema_version and $schema, and forward-compatible unknown properties. Unknown properties never confer permissions or platform authority. PTS-0004 is normative for these contracts.

## 3. Explicit non-decisions

This record does not approve or implement a production Marketplace signing service, production bootstrap key material, remote trust endpoint, numeric ZIP limits, Registry network client/downloader, ZIP extractor, signature verifier, startup recovery coordinator, public SDK lifecycle RPCs, Plugin Loader, Runtime, or execution. These require separate implementation work and review.

## 4. Accepted ZIP resource policy

PZIP-0001 — Plugin ZIP Installation Policy v1.0.0 is accepted. Its machine-readable instance is `schemas/zip-installation-policy.v1.json` and its JSON Schema is `schemas/zip-installation-policy.schema.json`.

The mandatory limits are:

| Limit | Accepted value |
|---|---:|
| Compressed archive | 100 MiB |
| Total expanded content | 512 MiB |
| Individual file | 128 MiB |
| ZIP entries (files and directories) | 10,000 |
| Directory depth | 32 components |
| Per-entry compression ratio | 100:1 |
| Normalized UTF-8 path length | 240 bytes |
| Parser metadata allocation budget | 16 MiB |

The definitions, byte-counting rules, path-depth semantics, ratio handling, metadata budget, and fail-closed behavior are normative in PZIP-0001. These decisions unblock implementation of a bounded ZIP inspector/extractor; they do not approve a production installer or relax any PTS-0004 structural/security rule.
