# Plugin Installation and Recovery Technical Specification

**Version:** 0.1.0  
**Status:** Draft — D1–D4 contracts accepted by PTS-0004  
**Category:** Module Technical Specification  
**Document ID:** PTS-0002  
**Related ADR:** ADR-008 — Marketplace + Registry Architecture; ADR-012 — Local-First Plugin Storage and Persistence  
**Related Specification:** PTS-0001 — Plugin Storage and Persistence Technical Specification  
**Module:** PRIDE Plugin Module

---

## 1. Purpose

This document defines the proposed application and persistence contract for installing a Plugin into the local PRIDE environment.

It translates ADR-012's installation consistency requirements into a recoverable workflow between temporary filesystem artifacts, versioned installed artifacts, and the local SQLite registry.

This document is a design proposal. It does not establish an artifact format, cryptographic trust policy, Registry wire schema, or new public Plugin SDK RPC.

## 2. Current Architectural Baseline

The current implementation provides:

- A local-first storage layout under the user-specific PRIDE data directory.
- `Plugin_Storage/temporary`, `installed`, and `quarantine` directories.
- Safe path helpers for Plugin ID and version components.
- A SQLite implementation of the existing `registry.Repository` contract: Create, Get, and List.
- SDK discovery integration only.

The current domain and database record represent the installed Plugin's discovery-facing metadata. They do not yet persist installation provenance, artifact digest, verification evidence, selected artifact location, or a recoverable installation operation.

The installer must not pretend those fields already exist.

## 3. Goals

The installation design must:

1. Keep downloaded and extracted content outside the installed artifact tree until validation succeeds.
2. Validate the developer-owned Manifest independently from Marketplace-owned protected metadata.
3. Require explicit artifact-integrity and trust verification before promotion.
4. Prevent archive/path traversal and symlink escape.
5. Preserve the currently selected version when an update fails.
6. Coordinate filesystem changes and SQLite commits through a durable operation journal.
7. Make interrupted operations detectable and recoverable after process restart.
8. Expose a Plugin through local discovery only after the installed artifact and registry record are consistent.
9. Keep Plugin business execution, permission authorization, and runtime loading outside installation.

## 4. Non-Goals

This specification does not define:

- Marketplace UI or remote Registry hosting.
- A new public SDK lifecycle RPC or changes to the existing Protobuf contract.
- Plugin enable/disable/start/stop behavior.
- Plugin Runtime, Loader execution, or Active Plugin isolation.
- Permission grants or authorization.
- Plugin-owned data migration.
- Uninstall and destructive data cleanup.
- Automatic cleanup of old versions.
- A universal network downloader implementation.
- An approved archive format or cryptographic trust root.

## 5. Ownership and Package Boundaries

The proposed Module-side application boundary is:

```text
Installation Request
        |
        v
Installation Application Service
        |
        +---- Artifact Acquisition Port
        |
        +---- Package Inspection / Extraction Port
        |
        +---- Manifest Validation
        |
        +---- Artifact Verification Port
        |
        +---- Plugin Storage
        |
        +---- Installation Repository (SQLite)
        |
        v
Local Installed Plugin Registry
```

Responsibilities:

- **Installation application service:** coordinates the use case and its failure semantics. It does not implement network protocols, archive parsing, cryptographic algorithms, or authorization.
- **Artifact acquisition adapter:** obtains the artifact from an approved source and writes it into a caller-controlled temporary location. It does not mark a Plugin as installed.
- **Package inspection adapter:** inspects and extracts the selected package format into a staging directory while enforcing extraction limits and path containment.
- **Manifest validator:** validates the extracted developer-owned Manifest against the stable Manifest contract and the current Module domain constraints.
- **Artifact verifier:** verifies integrity and authenticity using an explicitly approved verification policy. It must fail closed when required evidence or trust configuration is unavailable.
- **Storage:** owns safe paths, staging, promotion, and quarantine filesystem operations.
- **Installation repository:** persists operation journal and installed-version metadata transactionally.
- **SDK adapter:** remains a transport/discovery adapter. It must not import storage, SQLite, or installer internals.

The existing `registry.Repository` contract remains unchanged for read-only discovery until a separately reviewed repository extension is approved.

## 6. Registry and SDK Contract Constraints

### 6.1 Official Registry source

ADR-008 establishes that the Official Plugin Registry contains Marketplace-generated metadata and references developer-controlled Plugin sources. The current Registry Format Specification is descriptive and does not define a concrete machine-readable schema for:

- Artifact download URL or release asset selection.
- Expected digest and digest algorithm.
- Signature representation and verification key/certificate reference.
- Package format and package media type.
- Source revision or immutable release identifier.

The installer must not guess these fields or scrape arbitrary repository pages as a substitute for a Registry contract.

A versioned machine-readable Registry schema and source-resolution contract are prerequisites for implementing a production Marketplace-to-installer path.

### 6.2 Plugin SDK boundary

The stable Plugin SDK Specification describes lifecycle operations including Install and Update. The currently implemented SDK protocol exposes only five read-only discovery RPCs. ADR-008 also describes installation as governed by the Plugin SDK.

This specification does not silently resolve that contract gap. The first Module installation application service may be tested through an internal composition boundary, but it must not be exposed as a public SDK operation until the SDK lifecycle contract and Protobuf API are reviewed and reconciled.

### 6.3 Manifest boundary

The developer-owned Manifest and Marketplace-owned protected Registry metadata remain separate. A Manifest cannot grant capabilities, permissions, trust, or authorization. Permission identifiers remain metadata only.

The exact JSON field naming and schema version for `plugin.json` must be confirmed against the stable Manifest contract before a strict decoder is implemented.

## 7. Proposed Installation Data Model

The current `domain.Plugin` aggregate should not be overloaded with filesystem paths, cryptographic evidence, or transient installer state.

The proposed design introduces a separate installation record and operation journal. Names below are conceptual and may be refined during review.

### 7.1 Installed version record

An installed-version record should persist at least:

| Field | Purpose |
|---|---|
| Plugin ID | Stable Plugin identity |
| Version | Exact installed version |
| Relative artifact path | Path relative to Plugin_Storage, never an arbitrary absolute path |
| Source reference | Immutable source/release reference from the approved Registry contract |
| Artifact digest | Algorithm-qualified digest of the exact artifact or canonical package |
| Verification policy/version | Identifies the verification rules applied |
| Verification result | Successful verification outcome and relevant evidence reference |
| Installed timestamp | UTC timestamp |
| Selected status | Whether this version is currently selected |
| Manifest snapshot | Validated Manifest associated with this installed version |

Secrets, private keys, credentials, and authorization grants must never be stored in these records.

### 7.2 Installation operation journal

A durable operation record should contain:

- Unique operation ID generated by the Module.
- Plugin ID and target version.
- Operation kind: install or update.
- Current durable phase.
- Relative staging and final artifact paths.
- Expected artifact digest and verification policy reference.
- Validated Manifest/installation metadata required to resume or safely roll back.
- Timestamps and a bounded, non-secret failure category.

The journal must not persist arbitrary executable instructions or trust decisions supplied by the package itself.

### 7.3 Repository boundary

A new installation persistence boundary is required. It should support transaction-level operations rather than composing several independent calls to the current `registry.Repository`.

At minimum, the contract must be able to:

- Begin an operation and reject conflicting active operations for the same Plugin.
- Persist a verified staging result.
- Atomically commit the installed-version record, selected version, discovery-facing Plugin record, and completed operation state.
- Read incomplete operations for startup recovery.
- Mark an operation rolled back or failed with a stable non-secret reason.

The SQLite adapter must perform each multi-record state change in one SQL transaction. The existing discovery repository API must remain narrow and SDK-neutral.

## 8. Filesystem Layout and Operation Paths

The existing PTS-0001 layout is retained:

```text
Plugin_Storage/
├── installed/<plugin-id>/<version>/
│   ├── manifest.json
│   └── package/
├── temporary/<operation-id>/
│   ├── artifact
│   └── extracted/
└── quarantine/<operation-id>/
```

The exact artifact filename and archive representation are intentionally unresolved until the package format is selected.

Rules:

1. Every operation receives a unique opaque operation ID generated by PRIDE.
2. Staging and final paths are derived only through storage path helpers.
3. Plugin ID and version are validated before any path is created.
4. Extraction must reject absolute paths, parent traversal, volume-qualified paths, invalid path separators, and entries escaping the staging root.
5. Symlinks, hard links, special files, device nodes, and link targets must be rejected unless a future package-format policy explicitly permits a narrowly defined safe subset.
6. Extraction must enforce configured limits for total expanded bytes, individual file size, entry count, path length, and nesting depth.
7. The staging and final directories must reside on the same filesystem if promotion relies on atomic rename.
8. Promotion must never overwrite an existing installed version directory.
9. The installer must not follow attacker-controlled symlinks when writing, verifying, or promoting content.
10. POSIX permissions follow PTS-0001. Windows relies on the user's application-data directory ACLs.

## 9. Proposed Installation State Machine

The durable operation journal should represent these phases:

| Phase | Meaning |
|---|---|
| PREPARING | Operation exists; artifact acquisition or extraction may be in progress |
| STAGED | Package content has been written to an isolated staging location |
| VERIFIED | Manifest, compatibility, integrity, and required trust checks have succeeded |
| PROMOTED | Verified content has been moved to the immutable versioned installed path |
| COMMITTED | Installed-version metadata and discovery-facing Plugin record are committed in SQLite |
| ROLLED_BACK | The operation has been safely reversed and is no longer active |
| FAILED | Operation cannot proceed; diagnostic category is persisted |

Only COMMITTED is an installed, discoverable result. PREPARING, STAGED, VERIFIED, and PROMOTED operations must not appear as healthy installed Plugins.

A verification failure must never transition to PROMOTED or COMMITTED.

## 10. Installation Workflow

The proposed first-install flow is:

1. Validate the request and reject an invalid Plugin ID/version or conflicting active operation.
2. Resolve an immutable source reference through a future approved Registry adapter.
3. Create a durable PREPARING operation record.
4. Acquire the artifact into the operation's temporary directory.
5. Inspect the package and extract it into an isolated staging directory under strict resource and path limits.
6. Locate and strictly decode the one required Manifest; validate identity and version consistency against the requested Registry record.
7. Verify the artifact digest and required authenticity evidence through the approved verifier.
8. Persist the validated installation metadata and VERIFIED phase.
9. Promote the verified staged directory to `installed/<plugin-id>/<version>` using a no-overwrite same-filesystem operation.
10. Persist PROMOTED.
11. In one SQLite transaction, create/update the discovery-facing Plugin record, record the installed version and selected version, and mark the operation COMMITTED.
12. Return success only after the database transaction commits.

The installer must not enable, start, load, or execute the Plugin as part of installation.

## 11. Update Semantics

An update is a new versioned installation operation.

- The currently selected version remains selected while the new version is acquired, extracted, and verified.
- The new version is promoted to its own immutable version directory.
- The selected-version pointer and discovery-facing Plugin record change only in the final SQLite commit.
- If any pre-commit stage fails, the previous selected version remains unchanged.
- The previous version's artifact directory is retained. Cleanup and rollback retention policies are deferred.
- An update targeting a version already present must return a conflict or enter an explicitly designed repair flow; it must not overwrite the existing version directory.

The update path must not be implemented until installed-version metadata and atomic selected-version persistence are included in the reviewed schema.

## 12. Filesystem / SQLite Consistency and Recovery

Filesystem rename and SQLite commit cannot participate in one atomic transaction. A durable journal is therefore mandatory.

The intended ordering is:

```text
SQLite: PREPARING
    |
Filesystem: acquire + extract to temporary
    |
SQLite: VERIFIED (persist validated metadata)
    |
Filesystem: atomic promote to installed/<id>/<version>
    |
SQLite: PROMOTED
    |
SQLite transaction:
  - persist installed-version record
  - persist selected version
  - persist discovery-facing Plugin record
  - mark operation COMMITTED
```

Startup recovery must inspect incomplete journal entries before reporting the affected Plugin as healthy.

Minimum recovery behavior:

| Journal / filesystem condition | Required behavior |
|---|---|
| PREPARING or STAGED, no final artifact | Remove or quarantine incomplete temporary data; mark rolled back |
| VERIFIED, staging exists, final absent | Revalidate staged artifact against persisted digest/policy before promotion; otherwise roll back |
| PROMOTED, final exists, no installed DB record | Reverify final artifact and complete the final DB transaction only if all persisted evidence remains valid; otherwise move to quarantine or remove safely and roll back |
| PROMOTED, final missing | Do not register the Plugin; mark operation failed and preserve previous selected version |
| COMMITTED, final artifact missing | Mark installation inconsistent/unavailable; never claim the artifact is healthy |
| Unreferenced final version directory | Do not auto-register it; quarantine or leave for a separately defined reconciliation policy |
| DB record references a path outside Plugin_Storage | Treat as corrupt/unsafe metadata and fail closed |

Recovery must be idempotent. Re-running it after a crash must not duplicate installed records, overwrite a version, or select an unverified artifact.

A journal record alone is not proof that an artifact still matches its previously verified bytes. Recovery must re-check the artifact against the recorded digest and approved policy before completing a commit.

## 13. Quarantine and Failure Handling

Quarantine is for packages rejected by verification or retained for diagnosis. Quarantined content:

- Must not be listed as installed.
- Must not be passed to a loader or runtime.
- Must not become a source for automatic retry without a new explicit verification operation.
- Must not contain credentials or secrets written by the installer.
- Must have bounded retention and disk-usage policies before production rollout.

Failure messages returned across application boundaries must use stable categories and must not expose absolute paths, credentials, signature material, or raw package contents.

## 14. Security Requirements

The implementation must:

- Fail closed if required digest or authenticity evidence is absent, malformed, unsupported, or cannot be validated.
- Never equate a successful download with a trusted package.
- Never treat Manifest-declared permissions or capabilities as authorization grants.
- Verify identity and version consistency between the trusted Registry metadata, the Manifest, and the requested operation.
- Use immutable source references where the approved Registry format supports them.
- Enforce archive extraction limits and path containment.
- Prevent symlink and hard-link escape.
- Avoid executing package content during inspection or installation.
- Avoid logging full Manifest attributes or Registry payloads if they may contain sensitive data.
- Keep all filesystem paths out of public SDK discovery responses.
- Use cryptographic algorithms and trust roots only after a separate security decision specifies them.

No insecure fallback such as "skip signature verification when unavailable" is permitted.

## 15. Testing Requirements

### Application workflow

- Successful first install.
- Successful update preserving the old selected version until commit.
- Duplicate operation and conflicting operation rejection.
- Manifest identity/version mismatch.
- Missing or malformed required verification evidence.
- Verification failure never promotes an artifact.
- Context cancellation at every external port boundary.
- Stable error categories with no path/secret leakage.

### Filesystem and package inspection

- Traversal and absolute path entries.
- Symlink/hard-link escape.
- Special files and unsupported entry types.
- Duplicate archive entries and path collisions.
- Excessive expanded size, file count, depth, and path length.
- Existing target version is never overwritten.
- Staging and final paths on incompatible filesystems fail safely.

### Persistence and recovery

- Journal phase transitions are transactional.
- Installed-version record, selected version, Plugin record, and COMMITTED state are atomic.
- Crash/failure injection after every durable phase.
- Recovery is idempotent.
- Recovery revalidates artifact digest before completing commit.
- Previous selected version remains intact after failed update.
- Missing artifact never appears healthy through discovery.
- Unreferenced artifacts are never auto-registered.

All tests use temporary directories and isolated SQLite databases. No test writes to a real PRIDE user data directory or downloads an arbitrary remote package.

## 16. Decisions Required Before Implementation

The following are explicit blockers, not implementation details to guess:

1. **Registry machine schema:** define the versioned structure for immutable source/release references, artifact location, digest, signature/provenance evidence, and package media type.
2. **Package format:** select the supported archive/container format and define its extraction semantics.
3. **Integrity and trust policy:** specify digest algorithms, signature/provenance requirements, trust roots, key rotation, revocation, and failure behavior.
4. **Manifest wire schema:** define exact JSON field names, schema versioning, strict unknown-field policy, and version semantics.
5. **Installation persistence schema:** review the new installed-version and operation-journal tables plus their transactional repository contract.
6. **SDK lifecycle contract:** reconcile the stable SDK specification and ADR-008 with the currently discovery-only Protobuf API. Do not add lifecycle RPCs as part of this Module-only design.
7. **Recovery policy:** approve whether invalid/incomplete staging data is deleted or quarantined, and define retention limits.

## 17. Explicitly Deferred

Even after the first installation increment, the following remain separate work:

- Plugin enable/disable/start/stop/reload.
- Runtime loading and execution.
- Permission authorization.
- Plugin-owned data migration.
- Uninstall and destructive cleanup.
- Old-version garbage collection.
- Marketplace publication and registry repository synchronization.
- Automatic updates and background update checks.
- Remote/distributed Plugin execution.

## 18. Acceptance Criteria

This specification is ready for implementation only when:

1. The Registry machine schema and source-reference contract are approved.
2. The package format and extraction safety policy are approved.
3. The integrity/signature/provenance policy is approved by the responsible security/architecture owner.
4. The Manifest JSON schema is unambiguous.
5. The installation journal and installed-version persistence contract are approved.
6. Recovery semantics and failure injection tests are accepted.
7. The SDK lifecycle contract gap is explicitly recorded and no unreviewed RPC is introduced.
8. The implementation plan preserves the existing discovery contract and SDK adapter module boundary.

## 19. References

- ADR-008 — Marketplace + Registry Architecture
- ADR-012 — Local-First Plugin Storage and Persistence
- PTS-0001 — Plugin Storage and Persistence Technical Specification
- PRIDE Plugin Manifest Specification
- PRIDE Registry Format Specification
- PRIDE Plugin Module Specification
- PRIDE Plugin SDK Specification


## 12. Accepted D1–D4 contract baseline

D1–D4 are no longer open design choices. They are finalized by the accepted Plugin Installation Contract Decision Record (DCR-0001) and the normative Plugin Artifact, Package, Trust, and Manifest Contract (PTS-0004).

The accepted baseline is:
- Provider-neutral versioned Marketplace artifact descriptor with immutable release identifier and external HTTPS artifact URI.
- ZIP package format and the root plugin.json plus package/ payload layout.
- SHA-256 digest plus Marketplace Ed25519 signature.
- Embedded bootstrap public key plus authenticated remote trust configuration, including sequence rollback protection, expiry, rotation, and revocation.
- snake_case Manifest JSON with both schema_version and $schema; unknown Manifest properties are forward-compatible but confer no platform authority.

PTS-0004 is normative for these contracts. PTS-0002 remains Draft for installation/recovery orchestration details not settled by D1–D4. Numeric ZIP resource limits must be published in a versioned installation policy before a production extractor is enabled.
