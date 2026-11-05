# Plugin Installation Persistence Contract

**Version:** 0.1.0  
**Status:** Draft — Review Required  
**Category:** Module Technical Specification  
**Document ID:** PTS-0003  
**Related Specifications:** PTS-0001, PTS-0002, PDR-0001  
**Module:** PRIDE Plugin Module

---

## 1. Purpose

This document defines the persistence boundary required by the first Plugin installation increment. It refines decision D5 from PDR-0001 and the conceptual persistence model in PTS-0002.

It covers durable installed-version records, operation journaling, SQLite transaction boundaries, state transitions, and recovery queries. It does not implement package acquisition, archive extraction, cryptographic verification, filesystem promotion, or public SDK lifecycle operations.

This specification remains Draft until reviewed and accepted.

## 2. Design Decisions

1. Keep the existing `domain.Plugin` aggregate focused on discovery-facing Plugin identity, metadata, availability, and state.
2. Do not add artifact paths, digest fields, installation history, or transient operation phases to `domain.Plugin`.
3. Add a separate installation persistence contract and separate records for installed versions and installation operations.
4. Keep the current `registry.Repository` interface unchanged.
5. Implement the installation persistence adapter in the existing Plugin Module SQLite persistence boundary so the final commit can atomically update the discovery-facing `plugins` row and installation tables.
6. Store filesystem paths only as validated paths relative to the Plugin_Storage root. Never persist arbitrary absolute paths.
7. Persist no secrets, private keys, credentials, authorization grants, or executable recovery instructions.

## 3. Ownership and Package Boundary

Proposed package structure:

```text
Modules/Plugin_Module/
├── installation/
│   ├── records.go
│   ├── repository.go
│   └── errors.go
└── registry/
    └── sqlite/
        ├── repository.go
        └── installation.go
```

The `installation` package owns SDK-neutral installation records and repository contracts. It must not import the SDK, generated Protobuf packages, SQLite drivers, filesystem storage implementation, or Marketplace clients.

The existing `registry/sqlite` adapter may implement both `registry.Repository` and `installation.Repository` against the same SQLite database. This is intentional: the final installation commit must atomically write installation records and the discovery-facing Plugin row.

The SDK adapter remains a separate Go module and must not import `installation`, `registry/sqlite`, or `storage`.

## 4. Durable Records

### 4.1 InstalledVersion

An installed version is an immutable record describing one verified Plugin artifact stored locally.

| Field | Type / constraint | Meaning |
|---|---|---|
| PluginID | validated Plugin ID | Stable Plugin identity |
| Version | validated version string | Exact installed version |
| ArtifactPath | relative path | Path under Plugin_Storage/installed |
| SourceReference | opaque non-empty reference | Immutable Registry release/source reference |
| DigestAlgorithm | algorithm identifier | Algorithm used for artifact digest |
| DigestValue | encoded digest | Digest of the exact acquired artifact |
| VerificationPolicy | non-empty policy identifier | Verification policy version applied |
| VerificationEvidenceRef | optional opaque reference | Non-secret reference to verification evidence |
| ManifestJSON | valid JSON snapshot | Validated developer-owned Manifest |
| InstalledAt | UTC timestamp | Time the record was committed |
| Selected | boolean | Whether this version is currently selected |

Constraints:

- Primary key is `(plugin_id, version)`.
- `artifact_path` must be relative and validated by the storage package before persistence.
- `source_reference`, digest, and verification policy are required for a committed installation.
- Digest algorithm and encoding are identifiers, not implicit defaults.
- The persistence layer stores the digest as supplied by the verifier; it does not perform cryptographic verification.
- The Manifest snapshot is immutable for that installed version.
- There may be at most one selected version for a Plugin.
- A version record is inserted only as part of the final COMMITTED transaction.

### 4.2 InstallationOperation

An installation operation is a durable record for one install or update attempt.

| Field | Type / constraint | Meaning |
|---|---|---|
| OperationID | opaque unique ID | Module-generated operation identity |
| PluginID | validated Plugin ID | Target Plugin |
| TargetVersion | validated version string | Version being installed |
| Kind | INSTALL or UPDATE | Operation type |
| Phase | enumerated phase | Durable state-machine phase |
| StagingPath | relative path, nullable | Temporary operation directory |
| FinalArtifactPath | relative path, nullable | Intended immutable installed path |
| SourceReference | opaque reference | Immutable source/release reference |
| DigestAlgorithm | identifier, nullable until known | Expected artifact digest algorithm |
| DigestValue | encoded value, nullable until known | Expected artifact digest |
| VerificationPolicy | identifier, nullable until known | Policy used to verify the artifact |
| ManifestJSON | validated JSON, nullable until verified | Snapshot required for commit/recovery |
| FailureCode | bounded stable code, nullable | Non-secret terminal failure category |
| CreatedAt / UpdatedAt | UTC timestamps | Operation lifecycle timestamps |

Allowed phases:

- `PREPARING`
- `STAGED`
- `VERIFIED`
- `PROMOTED`
- `COMMITTED`
- `ROLLED_BACK`
- `FAILED`

Only `COMMITTED` represents a successful installation. The other phases must not make a Plugin appear as a healthy installed result.

### 4.3 Data deliberately excluded

Do not persist:

- Credentials, tokens, private keys, signing keys, or secrets.
- Authorization decisions or permission grants.
- Arbitrary absolute paths.
- Raw package bytes in SQLite.
- Executable commands, scripts, or recovery instructions from the package.
- Unbounded raw error strings or untrusted Registry payloads.

## 5. SQLite Schema Proposal

The existing Plugin Registry database uses schema version 1. This increment proposes schema version 2. The migration must be transactional and must preserve every existing `plugins` row.

### 5.1 installed_plugin_versions

```sql
CREATE TABLE installed_plugin_versions (
    plugin_id TEXT NOT NULL,
    version TEXT NOT NULL,
    artifact_path TEXT NOT NULL,
    source_reference TEXT NOT NULL,
    digest_algorithm TEXT NOT NULL,
    digest_value TEXT NOT NULL,
    verification_policy TEXT NOT NULL,
    verification_evidence_ref TEXT,
    manifest_json TEXT NOT NULL,
    installed_at TEXT NOT NULL,
    selected INTEGER NOT NULL CHECK (selected IN (0, 1)),
    PRIMARY KEY (plugin_id, version)
);
```

At most one version may be selected per Plugin. Enforce this with a partial unique index:

```sql
CREATE UNIQUE INDEX one_selected_version_per_plugin
ON installed_plugin_versions(plugin_id)
WHERE selected = 1;
```

A foreign key to `plugins(plugin_id)` is intentionally not required for the initial proposal because the final transaction creates the discovery row and installed version together, while a later product decision may define whether disabled or unavailable Plugin metadata may outlive installed bytes. The repository must nevertheless enforce Plugin/version consistency at the application boundary.

### 5.2 plugin_installation_operations

```sql
CREATE TABLE plugin_installation_operations (
    operation_id TEXT PRIMARY KEY NOT NULL,
    plugin_id TEXT NOT NULL,
    target_version TEXT NOT NULL,
    operation_kind TEXT NOT NULL CHECK (operation_kind IN ('INSTALL', 'UPDATE')),
    phase TEXT NOT NULL CHECK (
        phase IN (
            'PREPARING', 'STAGED', 'VERIFIED', 'PROMOTED',
            'COMMITTED', 'ROLLED_BACK', 'FAILED'
        )
    ),
    staging_path TEXT,
    final_artifact_path TEXT,
    source_reference TEXT NOT NULL,
    digest_algorithm TEXT,
    digest_value TEXT,
    verification_policy TEXT,
    manifest_json TEXT,
    failure_code TEXT,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
```

Prevent concurrent non-terminal operations for the same Plugin:

```sql
CREATE UNIQUE INDEX one_active_installation_per_plugin
ON plugin_installation_operations(plugin_id)
WHERE phase IN ('PREPARING', 'STAGED', 'VERIFIED', 'PROMOTED');
```

The index deliberately permits a new operation after the previous operation reaches COMMITTED, ROLLED_BACK, or FAILED. A future policy may impose additional update serialization or cooldown rules.

### 5.3 Migration requirements

- Read `PRAGMA user_version` and reject a database newer than the supported schema.
- Upgrade version 1 to version 2 inside one SQLite transaction.
- Do not rewrite or drop the existing `plugins` table.
- Create the new tables and indexes idempotently within the migration transaction.
- Advance `user_version` only after every schema statement succeeds.
- On any failure, roll back the migration and leave version 1 usable.
- Keep the existing SQLite connection policy and busy timeout from PTS-0001.
- Test migration from a populated version-1 database, not only an empty database.

## 6. Repository Contract

The installation package exposes an SDK-neutral interface. The following is the conceptual Go contract; names may be adjusted during implementation without changing semantics:

```go
type Repository interface {
    BeginOperation(context.Context, InstallationOperation) error
    RecordStaged(context.Context, OperationID, StagedArtifact) error
    RecordVerified(context.Context, OperationID, VerifiedArtifact) error
    RecordPromoted(context.Context, OperationID, RelativeArtifactPath) error
    CommitInstallation(context.Context, OperationID, domain.Plugin, InstalledVersion) error
    GetOperation(context.Context, OperationID) (InstallationOperation, error)
    ListIncompleteOperations(context.Context) ([]InstallationOperation, error)
    MarkRolledBack(context.Context, OperationID, FailureCode) error
    MarkFailed(context.Context, OperationID, FailureCode) error
    ListInstalledVersions(context.Context, PluginID) ([]InstalledVersion, error)
    GetSelectedVersion(context.Context, PluginID) (InstalledVersion, error)
}
```

The implementation must use typed records and sentinel errors. It must not expose raw SQL errors as public domain behavior.

The exact Go names and parameter grouping may be refined during implementation, but the atomicity and state-transition guarantees below are normative.

## 7. Transaction and State Transition Rules

### 7.1 BeginOperation

One SQLite transaction must:

1. Validate required operation fields and phase PREPARING.
2. Reject duplicate OperationID.
3. Reject an existing non-terminal operation for the same Plugin.
4. Insert the operation record.

### 7.2 RecordStaged

- Only PREPARING may transition to STAGED.
- Staging path must be a validated relative path under Plugin_Storage/temporary.
- The operation identity and target version cannot change.

### 7.3 RecordVerified

- Only STAGED may transition to VERIFIED.
- Persist the expected digest, algorithm, verification policy reference, verified Manifest snapshot, and final relative path.
- The persistence layer records the verifier's result; it does not independently trust or recreate that result.
- All required fields must be present before VERIFIED can be persisted.

### 7.4 RecordPromoted

- Only VERIFIED may transition to PROMOTED.
- The final relative path must match the path persisted during verification.
- This method records the filesystem promotion result; it does not perform the rename.

### 7.5 CommitInstallation

This is the critical atomic boundary. In one SQL transaction:

1. Confirm the operation is PROMOTED.
2. Confirm Plugin ID and version agree across operation, Plugin aggregate, and InstalledVersion.
3. Confirm the InstalledVersion digest, source reference, policy, Manifest, and final path match the verified operation.
4. Clear the previous selected-version marker for the Plugin.
5. Insert the new immutable installed-version record with selected = 1.
6. Create or update the discovery-facing `plugins` row using the existing Plugin persistence mapping.
7. Transition the operation to COMMITTED.
8. Commit the SQL transaction.

If any step fails, no partial database state may remain. The previous selected version and Plugin record remain unchanged.

### 7.6 Terminal transitions

- COMMITTED, ROLLED_BACK, and FAILED are terminal.
- Terminal operations cannot transition back to an active phase.
- Repeated terminal calls must be idempotent only when the requested terminal result and persisted data match; conflicting retries return a stable conflict error.
- Failure codes are bounded identifiers, not raw exception strings.

## 8. Recovery Queries and Guarantees

`ListIncompleteOperations` returns PREPARING, STAGED, VERIFIED, and PROMOTED records in deterministic order by creation time then OperationID.

The repository does not itself inspect or mutate filesystem content. Recovery orchestration uses the storage package and verifier port to reconcile each operation.

Required guarantees:

- A PREPARING/STAGED/VERIFIED/PROMOTED operation is never treated as a committed installation.
- A COMMITTED operation has an installed-version record and matching discovery-facing Plugin row.
- A failed SQL transaction cannot leave a selected-version pointer partially changed.
- Re-running a committed operation does not insert a duplicate version or switch selection unexpectedly.
- An orphan filesystem directory is never inserted into the database automatically.
- A committed record whose artifact is later missing must be surfaced as inconsistent/unavailable by a separate reconciliation action; persistence alone cannot prove file existence.

## 9. Error Contract

The repository should expose stable errors for:

- Invalid operation or installed-version record.
- Duplicate operation ID.
- Conflicting active operation.
- Operation not found.
- Invalid state transition.
- Installed version already exists.
- Selected version not found.
- Operation/Plugin/version mismatch.
- Context cancellation or deadline.
- Persistence failure.

Error messages must not expose absolute filesystem paths, secrets, raw Registry payloads, or signature material.

## 10. Test Requirements

### Migration
- Empty schema version 0 to version 2.
- Populated schema version 1 to version 2 preserving every Plugin row.
- Migration rollback on injected failure.
- Future schema version rejection.
- Index constraints for selected versions and active operations.

### Operation journal
- Begin operation and duplicate rejection.
- Conflicting active operation rejection.
- Every valid and invalid phase transition.
- Deterministic incomplete operation ordering.
- Terminal transition idempotency and conflict behavior.
- Context cancellation on every repository operation.

### Atomic commit
- First install creates version record, Plugin row, selected marker, and COMMITTED state together.
- Update switches selected version only at commit.
- Failed commit leaves old selected version and Plugin row unchanged.
- Duplicate installed version cannot overwrite an existing record.
- Mismatched Plugin ID/version/digest/source/policy/path is rejected.
- Database reopen preserves installed versions and operation journal.
- Defensive copies prevent mutation of returned Manifest snapshots.

All tests must use temporary directories and isolated SQLite databases.

## 11. Implementation Sequence

After this specification is accepted:

1. Add SDK-neutral installation records, validation, sentinel errors, and repository interface.
2. Add SQLite schema version 2 migration and installation repository implementation.
3. Add atomic transaction tests, migration tests, and crash/failure injection around database boundaries.
4. Keep the installer coordinator and filesystem promotion out of this increment.
5. Do not add lifecycle RPCs in this increment.

## 12. Explicit Non-Goals

This increment does not implement:

- Registry artifact resolution or network download.
- ZIP/archive parsing or extraction.
- Digest computation or signature verification.
- Filesystem promotion or rollback.
- Startup recovery orchestration.
- Install/update application service.
- SDK lifecycle RPCs.
- Loader, runtime, execution, authorization, or uninstall.

---

## Review Checklist

- [ ] Separate InstalledVersion and InstallationOperation records approved.
- [ ] Schema version 2 and migration strategy approved.
- [ ] Single SQLite adapter implements both discovery and installation persistence contracts.
- [ ] Atomic final commit semantics approved.
- [ ] Active-operation and selected-version uniqueness constraints approved.
- [ ] Error and recovery query contracts approved.
