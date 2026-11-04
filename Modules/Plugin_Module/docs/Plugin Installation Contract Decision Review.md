# Plugin Installation Contract Decision Review

**Version:** 0.1.0  
**Status:** Draft — Decision Review  
**Category:** Architecture / Module Design Review  
**Document ID:** PDR-0001  
**Related ADR:** ADR-008 — Marketplace + Registry Architecture; ADR-012 — Local-First Plugin Storage and Persistence  
**Related Specifications:** PTS-0001, PTS-0002, PRIDE Plugin SDK Specification, PRIDE Plugin SDK Contract, PRIDE Plugin SDK Protocol Specification  
**Module:** PRIDE Plugin Module

---

## 1. Purpose

PTS-0002 defines the required installation and recovery guarantees, but deliberately leaves several implementation contracts unresolved. This document turns those blockers into explicit decision points with alternatives and a recommended direction.

This document is a proposal for review. Recommendations below are not accepted architecture until approved and incorporated into the relevant stable specification or accepted technical specification.

No production installer, Registry parser, package extractor, signature verifier, lifecycle RPC, or persistence migration is introduced by this document.

## 2. Current Verified Baseline

The repository currently contains:

- Stable PRIDE Plugin Manifest Specification 1.0.0. It defines required semantic fields, but does not provide a normative JSON example or exact JSON property naming rules.
- Stable PRIDE Registry Format Specification 1.0.0. It states that Registry documents are Marketplace-generated and include Marketplace metadata, verification status, trust score, and a digital signature. It does not define a machine-readable artifact descriptor, package media type, digest representation, or signature verification envelope.
- Stable PRIDE Plugin SDK Specification 1.0.0. It describes the SDK as the communication interface for discovery, lifecycle, execution, metadata, and state.
- Draft Plugin SDK Contract 1.0.0. It describes lifecycle operations including INSTALL, ENABLE, DISABLE, START, STOP, and UNINSTALL.
- Draft Plugin SDK Protocol Specification 1.0.0. It explicitly scopes the initial protocol increment to read-only discovery and defers lifecycle and execution additions.
- Implemented Plugin SDK protocol and Go client/server adapter with five read-only discovery RPCs.
- Plugin Module domain, in-memory and SQLite registry implementations, discovery application service, local storage layout, and SDK discovery integration.
- PTS-0001 accepted and PTS-0002 draft.

The review must preserve the current separation: SDK owns communication contracts; Plugin Module owns Plugin state and behavior; Marketplace owns Registry generation and trust metadata.

## 3. Decision Summary

| ID | Decision | Recommended direction | Blocking |
|---|---|---|---|
| D1 | Artifact descriptor | Versioned Registry artifact descriptor with immutable release reference, URL, media type, and digest | Yes |
| D2 | Package format | ZIP-based package format for the first local installer increment | Yes |
| D3 | Integrity and authenticity | SHA-256 digest plus Marketplace signature verification; explicit trust-root and revocation policy | Yes |
| D4 | Manifest JSON | Normative JSON Schema and canonical property names matching the stable Manifest semantics | Yes |
| D5 | Installation persistence | Separate installed-version records and durable operation journal; atomic SQLite commit boundary | Yes |
| D6 | SDK lifecycle boundary | Add explicit lifecycle RPCs only after operation semantics are approved; SDK remains transport-only | Yes |
| D7 | Recovery and retention | Deterministic startup recovery; never auto-register orphan artifacts; bounded quarantine retention policy | Yes |

## 4. D1 — Registry Artifact Descriptor

### Problem

The current Registry Format Specification describes Registry contents conceptually. The installer still needs an immutable artifact locator and verifiable metadata. A repository URL alone is not an artifact locator and must not cause the installer to guess a release asset.

### Options

1. **Repository convention:** derive an artifact URL from a repository URL and naming convention.
   - Simple, but fragile and ambiguous across hosting providers and release layouts.
2. **Explicit artifact descriptor:** Marketplace publishes a versioned descriptor for each installable release.
   - Clear, provider-neutral, and independently verifiable.
3. **Provider-specific release object:** Registry contains GitHub/GitLab/etc. API-specific fields.
   - More expressive, but leaks hosting-provider details into the platform contract.

### Recommendation

Choose option 2. Each installable release should have an explicit descriptor containing, at minimum:

- Plugin ID and exact Plugin version.
- Immutable source/release identifier.
- Artifact download URI.
- Package media type / format identifier.
- Artifact byte length when available.
- Algorithm-qualified digest.
- Signature or signed-Registry evidence reference.
- Registry schema version.

The Registry must not accept a mutable branch URL as sufficient release identity. Provider-specific acquisition belongs in adapters behind a provider-neutral contract.

**Approval required:** confirm explicit artifact descriptors as the canonical source of installable bytes.

## 5. D2 — Package Format

### Options

1. ZIP archive.
   - Broad tooling support, deterministic enough when normalized, and supported by Go's standard library.
2. TAR archive, optionally compressed.
   - Common in developer ecosystems, but requires an explicit compression and link-entry policy.
3. Platform-specific installers or executable packages.
   - Not suitable for a portable Plugin ecosystem and expands the trusted computing base.

### Recommendation

Choose ZIP for the first package format version, with a PRIDE-defined package layout and strict extraction policy.

The format contract must define:

- Required root-level `plugin.json`.
- Package payload directory/layout.
- Allowed entry types (regular files and directories only for v1).
- Maximum archive size, expanded size, file count, path length, and nesting depth.
- Duplicate-path and case-collision behavior.
- Path normalization rules.
- Rejection of symlinks, hard links, device files, and traversal paths.
- Whether timestamps and file modes are preserved or normalized.

ZIP support does not imply that arbitrary ZIP files are trusted. Artifact verification remains mandatory before promotion.

**Approval required:** confirm ZIP as the first supported package format and approve the limits in a later PTS revision.

## 6. D3 — Integrity and Authenticity

### Required distinction

A digest detects whether bytes differ from an expected value. It does not establish who published those bytes. Authenticity requires a signature whose signer is trusted by an explicit policy.

### Options

1. Digest-only verification.
   - Detects corruption but cannot establish publisher authenticity.
2. Digest plus signed Registry metadata.
   - Allows the Marketplace to attest to the artifact descriptor and its analysis.
3. Developer-signed artifact plus Marketplace-signed Registry.
   - Provides separate publisher and Marketplace attestations, but requires identity, key enrollment, rotation, and revocation systems.

### Recommendation

Do not ship a digest-only production installer. Use algorithm-qualified SHA-256 digests for the initial integrity field and require a verifiable Marketplace signature over a canonical Registry payload or artifact descriptor.

Do not hardcode a trust root or signature algorithm in the installer until the platform trust policy is approved. The policy must define:

- Signature envelope and canonicalization.
- Trusted Marketplace key distribution.
- Key rotation and overlap.
- Revocation and compromise handling.
- Offline behavior and trust-policy freshness.
- Verification-policy version recorded with the installed artifact.
- Fail-closed behavior when required evidence is absent or invalid.

Developer signatures may be added as a second trust layer, but must not be conflated with Marketplace approval.

**Approval required:** approve the two-layer distinction (digest for integrity, Marketplace signature for authenticity) and assign ownership of trust-root lifecycle.

## 7. D4 — Manifest JSON Contract

### Problem

The stable Manifest specification defines semantic fields but not normative JSON property names, schema versioning, unknown-field behavior, or a canonical example. The SDK Protobuf field names are not automatically the Manifest wire format.

### Recommendation

Publish a normative JSON Schema for `plugin.json` and include:

- Explicit schema version or `$schema` URI.
- Canonical property names.
- Required and optional property definitions.
- String length and URL constraints.
- Version syntax and Plugin ID syntax.
- Unknown-field compatibility policy.
- At least one valid example and invalid examples.

The JSON decoder must not infer Manifest properties from Protobuf field names. The Manifest remains developer-owned identity metadata; capabilities, permissions, trust, and security analysis remain Marketplace-owned.

**Approval required:** choose the canonical JSON naming convention and schema-version strategy before implementing strict decoding.

## 8. D5 — Installation Persistence

### Options

1. Extend the existing `domain.Plugin` aggregate and `plugins` table with installer state.
   - Smaller initial diff, but mixes discovery identity with version history, artifact provenance, and transient operation state.
2. Add separate installed-version and operation-journal records.
   - Clear ownership and supports update/recovery without overloading the existing discovery record.

### Recommendation

Choose option 2.

Keep the current `registry.Repository` contract narrow for discovery. Introduce a separate installation persistence boundary with transaction-level operations.

Conceptual records:

- **InstalledVersion:** Plugin ID, exact version, relative artifact path, immutable source reference, digest, verification policy/evidence reference, Manifest snapshot, install timestamp, and selected-version marker.
- **InstallationOperation:** operation ID, Plugin ID, target version, install/update kind, durable phase, relative staging/final paths, expected digest, timestamps, and bounded non-secret failure category.

The final SQLite transaction must atomically update the selected version, installed-version record, discovery-facing Plugin record, and operation state to COMMITTED.

Secrets, credentials, private keys, and authorization grants must never be persisted in these records.

**Approval required:** confirm separate records and a separate installation repository interface. Exact SQL schema and migration remain implementation work after approval.

## 9. D6 — SDK Lifecycle Boundary

### Problem

The stable SDK Specification says platform components communicate with the Plugin Module through the SDK and describes lifecycle use cases. The draft SDK Contract names lifecycle operations. The implemented Protocol Specification and Protobuf currently expose discovery only.

This is a phased implementation gap, not a reason to put installation behavior inside the SDK.

### Options

1. Expose installation only through an internal Module application service indefinitely.
   - Useful for early integration tests, but does not satisfy the platform-wide SDK communication boundary.
2. Add explicit SDK lifecycle RPCs after defining their semantics.
   - Preserves the SDK as the official communication gateway.
3. Add a generic `ExecuteLifecycle(operation, input)` RPC.
   - Flexible, but weakens typed contracts and makes compatibility/error semantics harder to reason about.

### Recommendation

Choose option 2. Keep the Module installation coordinator internal while its behavior is developed and tested. Add explicit, typed SDK lifecycle RPCs only after the operation contracts are approved.

The first lifecycle protocol increment should define install and update separately, including request identity, idempotency, operation status, progress semantics (if any), cancellation behavior, and stable errors. Uninstall, enable/disable, start/stop, and execution should be separate increments rather than being bundled into installation.

The SDK client/server adapter remains transport and mapping only. It must not acquire artifacts, inspect archives, verify signatures, write SQLite, or own lifecycle state.

**Approval required:** confirm explicit typed lifecycle RPCs and whether Marketplace requests installation through the SDK directly or through a platform orchestration component that itself uses the SDK.

## 10. D7 — Recovery and Quarantine Retention

### Options

1. Delete all incomplete staging data at startup.
   - Simple, but unsafe if an operation had already promoted files and needs reconciliation.
2. Resume every incomplete operation automatically.
   - Risks repeating untrusted external actions and requires resumable acquisition semantics.
3. Reconcile from the durable journal and filesystem state, then resume only safe local transitions or roll back.
   - Deterministic and consistent with PTS-0002.

### Recommendation

Choose option 3.

Startup recovery should:

- Read all non-terminal journal records.
- Inspect only paths derived from validated operation IDs and storage helpers.
- Revalidate artifact digest before any commit after restart.
- Complete only idempotent local transitions whose prerequisites are proven.
- Roll back incomplete staging when safe.
- Quarantine ambiguous or rejected content without registering it.
- Never infer installation from an artifact directory alone.
- Never auto-register an orphan artifact without a committed journal/database transaction.

Quarantine retention must be bounded by both age and total storage budget. The exact defaults should be configurable and documented, with cleanup performed only after ensuring no active operation references the content.

**Approval required:** confirm journal-driven reconciliation and bounded quarantine retention. Exact retention defaults can be set in PTS-0002 after review.

## 11. Decision Dependencies and Implementation Order

The recommended dependency order is:

1. Approve D1, D2, D3, and D4 before implementing Registry resolution, package inspection, or verification.
2. Approve D5 before implementing the installation coordinator or recovery.
3. Approve D6 before exposing installation through the public SDK protocol.
4. Approve D7 before enabling startup recovery or automated quarantine cleanup.

Implementation can be staged after the relevant decisions are accepted:

- Registry and Manifest schema contracts.
- Package inspection/extraction adapter.
- Artifact verification policy and adapter.
- Installation persistence records and SQLite transactions.
- Installation application service and storage promotion.
- Recovery coordinator and crash-injection tests.
- SDK lifecycle protocol/client/server integration.

No stage may treat an unverified package as installed or executable.

## 12. Acceptance Criteria

This review is ready to close only when:

- Each decision D1–D7 has an explicit accepted or rejected outcome.
- Rejected recommendations have a documented alternative and rationale.
- The Registry, Manifest, SDK, and Module specifications do not contradict one another.
- PTS-0002 is updated to reference the accepted decisions.
- No implementation assumes an unapproved package format, trust root, signature algorithm, or public RPC.
- The Plugin SDK remains independent from storage, SQLite, archive parsing, and installer internals.
- The Plugin Module remains the owner of installed state and lifecycle behavior.

## 13. Explicitly Out of Scope

This review does not approve or implement:

- A Marketplace service or Registry hosting infrastructure.
- A specific signature algorithm or production trust root.
- Plugin execution, sandboxing, or runtime isolation.
- Permission authorization.
- Plugin uninstall or destructive data cleanup.
- Automatic updates.
- Dependency resolution between Plugins.
- Remote Plugin execution.
- UI or Marketplace user experience.

---

## Review Checklist

- [ ] D1 — Explicit versioned artifact descriptor
- [ ] D2 — ZIP as first package format
- [ ] D3 — Digest and Marketplace authenticity policy
- [ ] D4 — Normative Manifest JSON Schema
- [ ] D5 — Separate installed-version and operation-journal persistence
- [ ] D6 — Typed lifecycle RPCs after Module semantics are stable
- [ ] D7 — Journal-driven recovery and bounded quarantine retention
