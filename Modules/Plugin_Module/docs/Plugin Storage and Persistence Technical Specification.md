# Plugin Storage and Persistence Technical Specification

**Version:** 0.1.0  
**Status:** Draft  
**Category:** Module Technical Specification  
**Document ID:** PTS-0001  
**Related ADR:** ADR-012 — Local-First Plugin Storage and Persistence  
**Module:** PRIDE Plugin Module

---

## 1. Purpose

This document defines the first implementation-level contract for the local Plugin Storage layout and the persistent local Plugin Registry.

It translates ADR-012 into concrete package boundaries, path-resolution rules, SQLite repository behavior, schema-management requirements, and verification criteria. It does not redefine the local-first architecture.

## 2. Scope

The first persistence increment covers:

1. Resolving the PRIDE user data root.
2. Defining and safely resolving the Plugin_Storage directory layout.
3. Creating the state directory and SQLite database independently from Plugin package installation.
4. Implementing the existing registry.Repository interface with SQLite.
5. Preserving the in-memory repository as a test double.
6. Testing persistence, path containment, initialization, and failure behavior.

The increment does not implement package download, archive extraction, installation, signature verification, update, uninstall, lifecycle transitions, loading, or runtime execution.

## 3. Architectural Boundaries

The package layout shall be:

~~~text
Modules/Plugin_Module/
├── domain/
├── discovery/
├── registry/
│   └── sqlite/
├── storage/
└── sdkadapter/
~~~

Responsibilities:

- domain remains independent of filesystem, SQLite, database/sql drivers, and SDK packages.
- registry.Repository remains the application-facing persistence contract.
- registry/sqlite implements that contract and owns SQL schema details.
- storage owns user-data path resolution and filesystem layout/path safety.
- discovery continues to depend only on registry.Repository.
- sdkadapter remains a separate Go module and must not import SQLite or filesystem packages.

The SQLite implementation must not be placed in domain or in the SDK adapter.

## 4. User Data Root Resolution

The Plugin Module must not write persistent data beside the executable, inside the repository checkout, or into the current working directory by default.

The resolver shall support an explicit PRIDE_USER_DATA_DIR override. When provided, the value represents the complete PRIDE data root and must be absolute. Relative override values are rejected.

Without an override, the default root is platform-specific:

| Platform | Default PRIDE root |
|---|---|
| Windows | %LOCALAPPDATA%/PRIDE |
| macOS | ~/Library/Application Support/PRIDE |
| Linux | $XDG_DATA_HOME/pride, or ~/.local/share/pride when XDG_DATA_HOME is unset |
| Other | Use a documented platform-specific resolver; fail clearly if no safe user data root can be determined |

Path resolution must be injectable in tests. Package-level code must not read environment variables repeatedly during individual repository operations.

The resolved root is user-scoped and must remain stable across application upgrades.

## 5. Filesystem Layout

The logical layout is:

~~~text
<PRIDE_USER_DATA_DIR>/
├── Plugin_Storage/
│   ├── installed/
│   │   └── <plugin-id>/<version>/
│   ├── data/
│   │   └── <plugin-id>/
│   ├── cache/
│   ├── temporary/
│   └── quarantine/
└── state/
    └── plugin-registry.sqlite
~~~

The storage package shall expose resolved paths for these locations without requiring callers to concatenate path strings manually.

Directory initialization has two separate operations:

- EnsureStateDirectory creates the state directory when the local registry database is initialized.
- EnsurePluginStorage creates Plugin_Storage and its child directories when an installation operation first requires them.

Opening the local registry must not create the Plugin package directory tree prematurely.

The package shall provide path helpers for an installed Plugin version directory and a Plugin-owned data directory. These helpers must validate path components and verify that the resolved result remains inside the configured storage root. Absolute paths, path separators, dot and dot-dot components, volume-qualified paths, and traversal attempts must be rejected. Symlink handling must not allow a Plugin-controlled path to escape the storage root.

The first increment creates directories only. It does not copy, extract, move, delete, or execute Plugin package files.

## 6. Filesystem Permissions

On POSIX systems, newly created PRIDE-owned directories should use owner-only permissions (0700) and files created by the storage layer should use owner read/write permissions (0600) unless a stricter platform policy applies.

On Windows, creation must rely on the current user's application-data directory ACLs. The implementation must not attempt to weaken inherited ACLs.

Permission behavior must be covered where the operating system permits deterministic testing. Tests must not assume POSIX permission semantics on Windows.

## 7. SQLite Repository

### 7.1 Driver and API

The adapter shall use Go's database/sql API with modernc.org/sqlite, matching the pure-Go SQLite driver already used by Modules/AI_Module.

SQLite-specific imports and SQL statements must remain inside registry/sqlite.

The repository constructor must accept an explicit database path or DSN so tests can use isolated temporary databases. Production composition obtains the database path from the storage layout.

The repository must expose Close for deterministic resource cleanup.

### 7.2 Existing Repository Contract

The first adapter implements the current interface without widening it:

- Create(context.Context, domain.Plugin) error
- Get(context.Context, domain.PluginID) (domain.Plugin, error)
- List(context.Context) ([]domain.Plugin, error)

Do not add update, delete, installation, or lifecycle methods as part of this increment. Those operations require separate application contracts and invariants.

### 7.3 Initial Schema

The initial schema shall use a versioned migration and a plugins table with:

| Column | Type | Constraint / meaning |
|---|---|---|
| plugin_id | TEXT | Primary key; canonical domain Plugin ID |
| manifest_json | TEXT | JSON representation of developer-owned Manifest |
| plugin_type | TEXT | Stable textual Plugin type |
| plugin_state | TEXT | Stable textual operational state |
| protected_metadata_json | TEXT | JSON representation of platform-owned protected metadata |
| available | INTEGER | Boolean constrained to 0 or 1 |
| status_message | TEXT | Non-secret operational status message |
| created_at | TEXT | UTC RFC3339Nano timestamp |
| updated_at | TEXT | UTC RFC3339Nano timestamp |

The schema must use explicit textual mappings for Plugin type and state. It must not persist Go enum ordinals or rely on numeric equivalence with SDK Protobuf enums.

Manifest and protected metadata remain separate JSON values to preserve their ownership boundary. JSON encoding failures, including unsupported values in Attributes, must be returned as persistence errors rather than silently dropping data.

The initial schema persists the current domain.Plugin aggregate. Installation-specific values such as source reference, artifact checksum, verification result, installed timestamp, selected-version history, and data-directory reference must be added through a separately reviewed domain/schema extension before the installer depends on them. Their absence in this first adapter must not be represented as verified or trusted state.

### 7.4 Schema Migrations

Schema versioning shall use SQLite PRAGMA user_version.

Each migration must:

1. Run inside a database transaction.
2. Advance the schema version only after all migration statements succeed.
3. Roll back on any failure.
4. Reject a database schema version newer than the implementation supports.
5. Be tested against both a fresh database and a prior-version fixture.

The migration mechanism must be deterministic and must not silently drop existing Plugin records.

### 7.5 Repository Semantics

- Create validates the Plugin before writing it.
- Duplicate Plugin IDs map to registry.ErrAlreadyExists.
- Get maps a missing row to registry.ErrNotFound.
- List returns all records ordered by Plugin ID ascending.
- All operations honor context cancellation.
- Returned values must preserve the defensive-copy behavior of the in-memory repository.
- SQL and serialization errors must be wrapped with operation context without exposing filesystem secrets or connection credentials.
- The adapter must normalize externally observable repository errors with the in-memory implementation wherever the existing contract defines an equivalent error.

The adapter must not implement discovery filtering, candidate ranking, authorization, or SDK error mapping.

## 8. SQLite Operational Requirements

The implementation must configure SQLite for safe local use and deterministic tests.

At minimum:

- Enable foreign-key enforcement for connections if foreign keys are introduced.
- Configure a bounded busy timeout.
- Use transactions for schema migrations and any future multi-statement write.
- Ensure database handles are closed on constructor failure.
- Avoid unbounded connection creation for a single local SQLite file.
- Document journal-mode choices and test reopen behavior.

WAL mode may be enabled if the implementation verifies its behavior on supported platforms and handles the associated -wal and -shm files as part of the database lifecycle. WAL must not be enabled merely as an untested performance toggle.

## 9. Filesystem and Database Authority

The SQLite database is the authoritative installed Plugin inventory for discovery. The filesystem is authoritative for artifact bytes and artifact existence.

A database row must not be treated as proof that an artifact exists, passed verification, or is safe to load.

This increment does not reconcile database records with package directories because it does not install or load packages. Reconciliation belongs to the later installation/storage integration increment.

## 10. Error and Security Requirements

The implementation must:

- Reject invalid or escaping storage paths.
- Never derive filesystem paths from unvalidated Plugin IDs or versions.
- Keep absolute local paths out of SDK discovery responses.
- Keep credentials and secrets out of the Plugin Registry database.
- Treat capabilities and permission IDs as metadata, not authorization grants.
- Avoid logging complete Plugin metadata when it may contain sensitive attributes.
- Return errors with operation context while preserving errors.Is compatibility for registry sentinel errors and context cancellation.

The storage layer does not establish Plugin trust. Package verification and quarantine policy remain deferred to the installation design.

## 11. Testing Requirements

### SQLite repository tests

- Create and retrieve a valid Plugin.
- Persist and reopen a file-backed database.
- Reject duplicate IDs.
- Return ErrNotFound for missing IDs.
- Return deterministic Plugin ID ordering.
- Preserve nested Manifest and protected metadata values.
- Preserve defensive-copy semantics.
- Reject invalid domain records.
- Propagate canceled contexts.
- Handle malformed persisted JSON as an error.
- Roll back failed schema migrations.
- Reject unsupported future schema versions.

### Filesystem layout tests

- Resolve explicit absolute root overrides.
- Reject relative root overrides.
- Resolve platform defaults through injectable environment/platform inputs.
- Create the state directory independently.
- Create Plugin_Storage only when explicitly requested.
- Resolve installed-version and Plugin-data paths.
- Reject traversal, absolute paths, separators, and invalid components.
- Verify resolved paths remain inside the configured root.
- Verify repeated initialization is idempotent.

Tests must use temporary directories and isolated SQLite databases. No test may write to a real user data directory.

## 12. CI Requirements

The existing Plugin Module workflow must continue to run:

- gofmt validation.
- go test ./... from Modules/Plugin_Module.
- go vet ./... from Modules/Plugin_Module.
- The independent SDK adapter formatting, test, and vet jobs from Modules/Plugin_Module/sdkadapter.

The SDK adapter module must remain independently testable after the root module gains its SQLite dependency. Any resulting module-graph or checksum changes must be explicit and validated in CI.

## 13. Explicitly Deferred

The following are outside this technical specification's first implementation increment:

- Plugin package download and source resolution.
- Archive extraction and archive traversal protection.
- Signature, provenance, checksum, and compatibility verification.
- Installation staging and atomic promotion.
- Cross-resource recovery between filesystem moves and database commits.
- Plugin updates, rollback, uninstall, and old-version retention.
- Plugin-owned data migrations.
- Quarantine retention and cleanup policies.
- Runtime loading and execution.
- Permission grants and authorization.

These require a separate installation and lifecycle design.

## 14. Acceptance Criteria

This specification is ready for implementation only when:

1. ADR-012 has completed architectural review and is accepted by the responsible maintainer.
2. The user-data root and override semantics are approved.
3. The storage layout and path-safety contract are approved.
4. The initial SQLite schema and migration approach are approved.
5. The dependency choice is accepted and does not compromise the separate SDK adapter module.
6. The tests and CI requirements are reflected in the implementation plan.

## 15. References

- [ADR-012 — Local-First Plugin Storage and Persistence](../../../01-Architecture/ADR/ADR-012%20%E2%80%94%20Local-First%20Plugin%20Storage%20and%20Persistence.md)
- [ADR-011 — Plugin Module Foundation Design](../../../01-Architecture/ADR/ADR-011%20%E2%80%94%20Plugin%20Module%20Foundation%20Design.md)
- [PRIDE Plugin Module Specification](../../../04-Platform%20Specification%20Standard/Module%20Specification/PRIDE%20Plugin%20Module%20Specification.md)
- [PSS-0001 — PRIDE Specification Writing Standard](../../../03-Standards/PSS-0001%20%E2%80%94%20PRIDE%20Specification%20Writing%20Standard.md)
