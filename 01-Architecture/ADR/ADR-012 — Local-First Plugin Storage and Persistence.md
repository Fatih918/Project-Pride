# ADR-012 — Local-First Plugin Storage and Persistence

**Status:** Draft

**Date:** 2026-11-04

**Category:** Architecture

**Decision ID:** ADR-012

---

# 1. Context

PRIDE is designed as a **local-first platform**. Its Plugin Module must manage Plugins installed in the current user's PRIDE environment without requiring a remote service to maintain the local installed inventory or access installed Plugin artifacts.

The Plugin architecture already distinguishes three different concerns:

- The developer-controlled repository or release is the source of the Plugin implementation.
- The Official Plugin Registry describes published Plugins and provides their distribution locations.
- The local Plugin Module Registry represents Plugins installed in the current PRIDE environment.

The architecture still needs an explicit contract for where installed Plugin artifacts live and how their installation metadata is persisted.

Treating a database as the storage location for Plugin binaries would mix file lifecycle with structured application state. Relying only on directory scanning would also make installation state, version selection, verification status, and lifecycle management difficult to maintain consistently.

PRIDE therefore requires a local storage architecture that separates Plugin artifacts from the structured records used to manage them.

# 2. Decision

PRIDE adopts a **Local-First Plugin Storage Architecture** based on two complementary persistence mechanisms:

1. **Local filesystem** for Plugin packages, installed artifacts, and Plugin-owned persistent files.
2. **Local SQLite database** for the authoritative installed Plugin inventory and structured installation metadata.

The Plugin Module owns the local installed Plugin Registry and coordinates changes between the filesystem and database.

The Marketplace and Official Plugin Registry remain separate from this local persistence architecture. They provide published Plugin metadata and distribution references; they do not store the user's installed Plugin artifacts.

The architecture is:

```text
        Developer Repository / Release
                    |
                    v
          Official Plugin Registry
                    |
              Marketplace
                    |
             Install Request
                    |
                    v
          Plugin Installation Flow
                    |
          Verify and Stage Package
                    |
             +------+------+
             |             |
             v             v
       Local Filesystem  Local SQLite
       Plugin_Storage    Installed Registry
             |             |
             +------+------+
                    |
                    v
          Plugin Module Registry
                    |
                    v
           Discovery / Lifecycle
```

The filesystem and database have distinct responsibilities. Neither replaces the other.

# 3. Local-First Ownership and Scope

The local Plugin Storage belongs to the current PRIDE user environment.

The default storage location shall be resolved from the operating system's user-specific application data directory. PRIDE must not assume that the application executable directory is writable or that user data should live inside the source repository.

The logical root is:

```text
<PRIDE_USER_DATA_DIR>/
├── Plugin_Storage/
└── state/
    └── plugin-registry.sqlite
```

The exact operating-system path is platform-specific and belongs in a Technical Specification.

The `Plugin_Storage` directory shall be created automatically when the first Plugin installation requires it. The local database and its parent state directory may be initialized independently by the application.

A user must be able to retain installed Plugin data when the PRIDE application binaries are upgraded or reinstalled, provided the user data directory is preserved.

# 4. Plugin_Storage Directory

The Plugin Module shall use the following logical directory structure:

```text
Plugin_Storage/
├── installed/
│   └── <plugin-id>/
│       └── <version>/
│           ├── manifest.json
│           └── package/
│
├── data/
│   └── <plugin-id>/
│
├── cache/
├── temporary/
└── quarantine/
```

## 4.1 installed/

Contains verified Plugin artifacts that have completed installation.

Installed artifacts shall be organized by Plugin ID and version. Keeping versioned artifacts allows the installer to stage an update without immediately destroying the previously installed version and provides a basis for rollback.

The local registry identifies which version is currently selected for the installed Plugin.

## 4.2 data/

Contains persistent data owned by individual Plugins.

This directory is separate from the installed package. Updating or replacing a Plugin package must not automatically delete its persistent data.

Plugin-specific data migration and compatibility rules must be defined by the relevant Plugin and platform contracts. PRIDE must not assume that arbitrary data can be safely shared between incompatible Plugin versions.

## 4.3 cache/

Contains regenerable data used to improve download or installation performance.

Cache contents are not authoritative and may be removed under a defined cleanup policy.

## 4.4 temporary/

Contains incomplete downloads, extracted packages, and staging data during installation or update.

A package in this directory is not considered installed or executable.

## 4.5 quarantine/

Contains packages rejected by integrity, signature, manifest, compatibility, or security verification, when retaining them is required for diagnosis.

Quarantined packages must not be discoverable as installed Plugins or loaded by the Plugin Runtime. Retention and cleanup policies belong to a Technical Specification.

# 5. Local Database Responsibility

PRIDE shall use SQLite as the default local persistence engine for the installed Plugin Registry.

The database file is stored under the PRIDE user data directory, separate from the Plugin artifact directory:

```text
<PRIDE_USER_DATA_DIR>/state/plugin-registry.sqlite
```

The database stores structured information required to manage installed Plugins, including:

- Plugin ID.
- Installed and selected version.
- Relative artifact location under Plugin_Storage.
- Installation source or registry reference.
- Artifact checksum and verification result.
- Installation timestamp.
- Local operational and lifecycle state.
- Compatibility information needed by local discovery and management.
- References to Plugin-owned data locations where required.

The exact relational schema, indexes, migration mechanism, and serialization formats are not defined by this ADR. They belong to the Plugin Module persistence Technical Specification.

The database shall not store Plugin binaries or complete package contents as blobs.

The database shall not become a substitute for the platform Credential Store. Passwords, access tokens, private keys, and other secrets must remain in the designated secure credential boundary.

Permission identifiers and Plugin-declared capabilities may be represented as metadata. The local Plugin Registry must not treat their presence as an authorization grant. Authorization remains owned by the platform Permission Manager and applicable Core security policy.

# 6. Filesystem and Database Authority

The two persistence mechanisms have separate authority:

| Information | Authority |
|---|---|
| Plugin package and artifact bytes | Local filesystem |
| Installed Plugin inventory | Local Plugin Registry database |
| Selected installed version | Local Plugin Registry database |
| Installation and lifecycle metadata | Local Plugin Registry database |
| Plugin-owned persistent files | Plugin data directory |
| Published Plugin metadata | Official Plugin Registry |
| Credentials and secrets | Credential Store |

The Plugin Module must not rely on a full directory scan for every discovery request. Discovery reads the local installed Plugin Registry through its repository boundary.

The filesystem remains authoritative for whether an artifact actually exists and for the bytes that are loaded or verified. A database record alone must never be treated as proof that an artifact is present or trustworthy.

The Module must detect and handle inconsistencies, including:

- A database record referencing a missing artifact.
- An artifact directory with no corresponding installed registry record.
- A checksum mismatch.
- An interrupted installation or update.
- A version directory left behind after a failed database commit.

Reconciliation behavior and recovery states belong in the persistence and installation Technical Specifications.

# 7. Installation and Update Invariants

The installation flow must coordinate filesystem and database changes safely.

The required logical sequence is:

1. Resolve the package source from trusted Registry information.
2. Download the package into temporary storage.
3. Verify package integrity, signature or provenance as required, manifest validity, compatibility, and applicable security policy.
4. Move the verified artifact into its versioned installed location.
5. Commit the corresponding installed Plugin record and selected version to the local database.
6. Expose the Plugin through local discovery only after the installation has reached a consistent installed state.

The implementation must handle failures between filesystem operations and database commits. It must not expose partially installed Plugins as healthy or executable.

Updates must preserve the previous selected version until the replacement has been verified and committed successfully. Rollback and retention behavior must be specified before implementing the installer.

This ADR establishes the consistency requirement; it does not prescribe a specific transaction coordinator or filesystem algorithm.

# 8. Local-First Runtime Expectations

The local installed Plugin inventory and installed artifacts must remain accessible without contacting the Marketplace or Official Plugin Registry.

Network access may still be required for operations such as:

- Discovering remote Plugins.
- Downloading a new Plugin or version.
- Checking remote publication or update information.
- Calling external services required by a Plugin's own functionality.

A Plugin's ability to execute offline depends on its own runtime dependencies and capabilities. Local-first architecture does not imply that every Plugin or every Plugin operation is offline-capable.

# 9. Component Responsibilities

## 9.1 Plugin Module

The Plugin Module owns:

- The local installed Plugin Registry.
- The repository abstraction for installed Plugin records.
- Coordination between Plugin artifact storage and registry persistence.
- Local discovery of installed Plugins.
- Installation consistency and reconciliation contracts.
- Later lifecycle operations over installed Plugin records.

## 9.2 Plugin Storage

Plugin Storage owns the filesystem organization and access to installed artifacts and Plugin-owned data.

It does not decide whether a Plugin is authorized to execute.

## 9.3 Local Persistence Adapter

The persistence adapter implements the Plugin Registry repository using SQLite.

Domain and discovery application packages remain independent of SQLite-specific APIs and schema details.

The in-memory repository remains available as a test double. It is not the production persistence implementation.

## 9.4 Marketplace and Official Plugin Registry

The Marketplace and Official Registry provide published Plugin metadata and distribution references.

They do not manage the local user's installed files or local installed Plugin state.

## 9.5 Core and Permission Manager

Core retains capability matching, candidate selection, workflow orchestration, and applicable platform policy.

The Permission Manager remains the authority for permission grants and authorization decisions.

# 10. Alternatives Considered

## 10.1 Store Plugin Artifacts in SQLite

Store package binaries and Plugin files as database blobs.

**Rejected.**

This mixes large file artifacts with structured metadata, complicates package updates and extraction, and makes filesystem-oriented loading and inspection less direct.

## 10.2 Use Filesystem Scanning as the Only Registry

Discover installed Plugins by scanning Plugin_Storage whenever the application starts or handles a discovery request.

**Rejected.**

Directory contents alone do not provide a reliable lifecycle record, selected-version state, installation provenance, or transactional metadata. Scanning may still be used for reconciliation, not as the primary discovery data source.

## 10.3 Use a Remote Database as the Installed Registry

Store each user's installed Plugin inventory in a centralized service.

**Rejected for the local-first installation model.**

It would make local Plugin management dependent on remote infrastructure and weaken the local ownership boundary.

A future optional synchronization service may be considered separately without replacing the local installed registry.

## 10.4 Use PostgreSQL for the Local Installed Registry

Run or require a PostgreSQL server for each local PRIDE installation.

**Rejected as the default local persistence mechanism.**

It introduces unnecessary server administration and deployment requirements for the local-first use case. SQLite provides transactional local persistence without a separately managed database server.

# 11. Architectural Invariants

The following rules are established:

1. PRIDE uses a local-first model for installed Plugin management.
2. Installed Plugin artifacts are stored in the user's local filesystem under a directory named `Plugin_Storage`.
3. Plugin_Storage is located under the PRIDE user-specific application data directory, not assumed to be beside the executable.
4. Plugin artifacts are organized by Plugin ID and version.
5. Plugin-owned persistent data is separated from installed package artifacts.
6. Temporary and quarantined packages are not treated as installed or executable Plugins.
7. SQLite is the default local persistence engine for the installed Plugin Registry.
8. The local database stores structured Plugin installation and management metadata, not Plugin package binaries.
9. The local Plugin Registry database is the authoritative inventory for installed Plugin discovery and selected-version metadata.
10. The filesystem is authoritative for artifact bytes and artifact existence.
11. The Plugin Module coordinates filesystem and database consistency and must support recovery from interrupted installation operations.
12. Domain and application packages remain independent of SQLite-specific implementation details.
13. The in-memory repository remains a test double, not the production persistence mechanism.
14. The Official Plugin Registry remains separate from the local installed Plugin Registry.
15. Credentials and secrets remain in the designated Credential Store.
16. Permission metadata does not constitute authorization.
17. Local installed Plugin discovery must not require Marketplace or Official Registry availability.
18. The architecture does not guarantee that every Plugin can execute without network access.

# 12. Consequences

## Positive Consequences

### Clear Artifact Ownership

Plugin packages are stored as ordinary local files under a predictable directory.

### Durable Local State

The SQLite database provides a structured record of installed Plugins, selected versions, and lifecycle metadata.

### Independent Package and Data Lifecycles

Plugin updates can replace package artifacts without automatically deleting Plugin-owned persistent data.

### Local Availability

Installed Plugin discovery and local inventory management do not depend on a remote Marketplace or Registry service.

### Replaceable Persistence Implementation

SQLite-specific code remains behind the repository boundary, preserving SDK-neutral domain and application logic.

### Recoverable Installation Flow

Staging, versioned artifact directories, and explicit database records provide a basis for safe installation, update, and recovery.

## Trade-offs

### Dual Persistence Coordination

The installer must coordinate filesystem operations and database transactions because they do not share one atomic transaction.

### Local Data Management

PRIDE must define backup, restore, cleanup, migration, and reconciliation behavior for both filesystem data and the database.

### Platform-Specific Paths

The user data directory must be resolved correctly on each supported operating system.

### SQLite Lifecycle

The application must manage database initialization, schema migrations, file permissions, and concurrent access correctly.

These trade-offs are accepted for the benefits of local ownership, predictable artifact storage, and durable structured state.

# 13. Relationship With Other Decisions

This ADR builds upon:

- **ADR-001 — Plugin-First Architecture**
- **ADR-002 — Plugin SDK as the Official Plugin Communication Boundary**
- **ADR-008 — Marketplace + Registry Architecture**
- **ADR-011 — Plugin Module Foundation Design**

ADR-008 defines the developer-owned Plugin source and Official Registry publication model. This ADR defines the separate local persistence model for a user's installed Plugin artifacts and records.

ADR-011 establishes the Plugin Module repository boundary and defers persistent storage selection. This ADR resolves that persistence decision for the local-first installed Plugin Registry.

This ADR does not redefine the Plugin SDK communication contract, Marketplace publication workflow, Plugin Manifest format, or Plugin Runtime isolation model.

# 14. Decision Outcome

PRIDE shall store installed Plugin artifacts in the user's local filesystem under `Plugin_Storage`, located inside the PRIDE user-specific application data directory.

PRIDE shall use SQLite as the default local persistence engine for the installed Plugin Registry. The database stores structured installation and management metadata, while the filesystem stores Plugin packages and Plugin-owned files.

The Plugin Module owns the coordination boundary between these persistence mechanisms. The local installed Plugin Registry remains independent from the remote Official Plugin Registry and Marketplace.

This decision establishes the local-first persistence architecture. Concrete paths per operating system, database schema, migrations, package layout details, and installation recovery algorithms shall be defined in the relevant Technical Specifications before implementation.
