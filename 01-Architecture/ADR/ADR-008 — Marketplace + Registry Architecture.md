# ADR-008 — Marketplace + Registry Architecture

**Status:** Accepted

**Date:** 2026-08-11

**Category:** Architecture

**Decision ID:** ADR-008

---

# 1. Context

PRIDE requires a standardized mechanism for connecting Plugin developers with Plugin users.

The platform must allow developers to publish Plugins while preserving ownership of their source repositories.

PRIDE therefore shall not operate the Marketplace as a centralized Plugin hosting or Plugin source-code database.

Instead, the Marketplace operates around standardized Plugin Registries.

A Registry describes a Plugin and provides the information required for discovery, verification, and installation while the actual Plugin implementation remains in the developer-controlled repository.

---

# 2. Decision

PRIDE adopts a **Registry-Oriented Marketplace Architecture**.

The Marketplace is an Official Interface of PRIDE responsible for providing the user-facing environment for Plugin discovery and publication.

The Marketplace does not become the storage location for Plugin source code or Plugin binaries.

Instead:

```text
Developer Repository
        │
        │ Publish
        ▼
   Marketplace
        │
        ▼
 Plugin Registry
        │
        ▼
PRIDE Registry Repository
```

For users:

```text
User
  │
  ▼
Marketplace
  │
  ▼
Plugin Registry
  │
  ▼
Repository / Release URL
  │
  ▼
Plugin Download
  │
  ▼
Plugin Installation
```

The Marketplace therefore acts as a bridge between **developer-owned Plugin sources** and **PRIDE users**.

---

# 3. Marketplace as Official Interface

The Marketplace exists as part of the PRIDE Official Interface.

It is presented to users as an official platform interface through which they can:

* Discover Plugins.
* Inspect Plugin information.
* Publish Plugins.
* Review Plugin Registry information.
* Initiate Plugin installation.

The Marketplace is therefore an interface and coordination layer rather than a Plugin database.

The Marketplace Specification defines the Marketplace as the official Plugin publishing, discovery, and distribution platform while explicitly separating it from Plugin implementation hosting.

---

# 4. Registry-Oriented Architecture

The Marketplace shall operate around standardized Plugin Registries rather than Plugin implementations.

A Registry contains standardized information describing a Plugin.

The Registry may contain information such as:

* Plugin identity.
* Plugin description.
* Version.
* Capabilities.
* SDK usage.
* Permissions.
* Compatibility.
* Security information.
* Verification information.
* Repository or release location.

The Registry does not contain the Plugin implementation itself.

This preserves developer ownership and allows the platform to operate without becoming a centralized Plugin hosting service.

---

# 5. Registry Repository

PRIDE shall maintain an official Registry Repository.

The Registry Repository stores standardized Plugin Registry information.

It shall not become a repository for Plugin implementations.

The Registry Repository shall not store:

* Plugin source code.
* Plugin binaries.
* Plugin releases.
* Complete developer repository contents.

Plugin implementations remain inside developer-controlled repositories.

The Registry Repository therefore acts as the official registry source for the PRIDE Plugin ecosystem.

---

# 6. Plugin Publication

Plugin publication begins when a developer provides a Plugin repository to the Marketplace.

The Marketplace analyzes the repository and generates the standardized Registry.

The publication model is:

```text
Developer
    ↓
Submit Repository
    ↓
Repository Analysis
    ↓
Registry Generation
    ↓
Developer Review
    ↓
Security Verification
    ↓
Registry Generated
    ↓
Pull Request
    ↓
PRIDE Registry Repository
    ↓
Plugin Available
```

The Marketplace does not publish the Plugin implementation directly.

It publishes the **Registry information** to the official Registry Repository.

---

# 7. Pull Request Publication Model

Registry publication shall use the repository's Pull Request mechanism.

The Marketplace generates the Registry and submits it to the official PRIDE Registry Repository through a Pull Request.

The Pull Request therefore represents the publication mechanism for adding or updating Registry information in the official repository.

The Marketplace does not directly overwrite the Registry Repository.

This preserves the repository as the authoritative storage location for published Registry information while maintaining a controlled publication process.

---

# 8. Developer Ownership

Plugin developers retain ownership of their Plugin implementations.

The Marketplace does not claim ownership of:

* Source code.
* Developer repositories.
* Plugin releases.
* Documentation.
* Intellectual property.

The developer remains responsible for the actual Plugin implementation.

The Marketplace manages the standardized Registry representation of that Plugin.

---

# 9. Registry Generation

The Marketplace may use the AI SDK to assist in generating Plugin Registry information.

The Marketplace analyzes the submitted repository and collected technical information.

The AI SDK may then assist in generating the standardized Registry.

The developer reviews the generated Registry before publication.

AI assistance therefore supports Registry generation but does not make the AI SDK the owner of the Marketplace or Registry architecture.

---

# 10. Protected Registry Metadata

Registry metadata shall be separated into editable and protected information.

Developers may edit descriptive information such as:

* Plugin Name.
* Description.
* Version.
* Category.
* Documentation.
* Homepage.
* Screenshots.
* Release Notes.

Security and platform-generated information remains protected.

Protected information may include:

* Capability declarations.
* SDK usage.
* Permission analysis.
* Compatibility analysis.
* Security status.
* Risk level.
* Digital signature.
* Registry Identifier.
* Verification status.
* Trust score.

Protected metadata shall not be freely modified by Plugin developers.

---

# 11. Plugin Discovery

Users discover Plugins through the Marketplace.

Discovery operates using Registry information rather than Plugin source code.

The Marketplace may provide:

* Search.
* Categories.
* Tags.
* Recommendations.
* Popular Plugins.
* Recently Updated Plugins.
* Verified Plugins.

The Registry therefore provides the standardized information required by the Marketplace to represent Plugins without hosting their implementations.

---

# 12. Plugin Installation

Plugin installation begins when a user selects a Plugin through the Marketplace.

The Registry provides the location required to obtain the Plugin implementation.

The installation model is:

```text
User
  ↓
Marketplace
  ↓
Plugin Registry
  ↓
Repository / Release URL
  ↓
Download Plugin
  ↓
Verify
  ↓
Plugin SDK
  ↓
Plugin Module
  ↓
Plugin Installed
```

The Marketplace itself does not become the Plugin installation engine.

The Plugin SDK remains responsible for Plugin installation according to the established Plugin architecture.

---

# 13. Marketplace Responsibilities

The Marketplace is responsible for:

1. Plugin discovery.
2. Plugin publication.
3. Repository analysis.
4. Registry generation.
5. Security verification.
6. Registry metadata management.
7. Developer publication assistance.
8. Publishing Registry information to the official Registry Repository.
9. Providing the user-facing Marketplace interface.

These responsibilities define the Marketplace's architectural role.

---

# 14. Marketplace Non-Responsibilities

The Marketplace shall not be responsible for:

* Hosting Plugin source code.
* Hosting Plugin binaries as the official implementation store.
* Executing Plugins.
* Managing Plugin lifecycles after installation.
* Workflow execution.
* Capability execution.
* AI reasoning.
* Memory management.
* Implementing the Plugin SDK.

Those concerns remain owned by their respective platform components.

---

# 15. Architectural Invariants

The following rules are established:

1. Marketplace is an Official Interface of PRIDE.
2. Marketplace operates as a Registry-oriented system.
3. Marketplace is not the Plugin source-code database.
4. Marketplace is not the official Plugin implementation host.
5. Plugin implementations remain in developer-controlled repositories.
6. Plugin Registries are the standardized representation of published Plugins.
7. PRIDE maintains an official Registry Repository.
8. The Registry Repository stores Registry information rather than Plugin implementations.
9. Plugin publication produces Registry information.
10. Registry publication uses a Pull Request to the official Registry Repository.
11. Users discover Plugins through the Marketplace using Registry information.
12. Registry information provides the location required to obtain the Plugin implementation.
13. Plugin installation remains governed by the Plugin SDK architecture.
14. Marketplace does not execute Plugins.
15. Marketplace does not become responsible for Workflow, Capability, Memory, or AI execution.
16. AI may assist Registry generation through the AI SDK.
17. Developer-controlled descriptive metadata and Marketplace-generated protected metadata remain separated.
18. The Marketplace acts as a bridge between developer-owned Plugin sources and PRIDE users.

---

# 16. Alternatives Considered

## 16.1 Centralized Plugin Hosting

PRIDE hosts all Plugin source code and releases inside the Marketplace.

**Rejected.**

This would make PRIDE responsible for centralized Plugin hosting and weaken developer ownership.

---

## 16.2 Marketplace as Plugin Database

The Marketplace stores Plugin implementations directly and functions as the primary Plugin database.

**Rejected.**

The architecture is Registry-oriented. Plugin implementations remain outside the Marketplace.

---

## 16.3 Manual Registry Creation

Developers manually create and maintain Registry files.

**Rejected.**

The Marketplace is designed to analyze repositories and assist Registry generation, reducing unnecessary publication complexity.

---

## 16.4 Direct Registry Repository Modification

The Marketplace directly modifies the official Registry Repository.

**Rejected.**

Registry publication uses a Pull Request model, keeping publication controlled through the repository workflow.

---

# 17. Consequences

## Positive Consequences

### Developer Ownership

Developers retain control over Plugin source repositories.

### Minimal Infrastructure

PRIDE does not need to host every Plugin implementation.

### Ecosystem Scalability

New Plugins can be added without requiring centralized Plugin storage.

### Standardized Discovery

Users interact with a consistent Registry-based discovery system.

### Clear Installation Boundary

Marketplace discovery remains separate from Plugin installation and execution.

### Repository-Based Publication

The Registry Repository provides a transparent and version-controlled source for published Registry information.

---

## Negative Consequences

### External Repository Dependency

Plugin availability depends on developer-controlled repositories or release locations.

### Registry Maintenance

Registry information must remain synchronized with the corresponding Plugin implementation.

### Verification Complexity

Marketplace publication requires repository analysis and security verification before Registry publication.

### Multiple Architectural Boundaries

Marketplace, Registry, Plugin SDK, and Plugin Module must remain clearly separated.

---

# 18. Relationship With Other Decisions

This ADR follows:

* **ADR-000 — Software Design Record Concept**
* **ADR-001 — Plugin-First Architecture**
* **ADR-002 — Plugin SDK Boundary**
* **ADR-003 — Official Interface**
* **ADR-004 — Capability Registry**
* **ADR-005 — Execution Architecture**
* **ADR-006 — Unified Memory Architecture**
* **ADR-007 — AI SDK Boundary**

This ADR establishes the Marketplace and Registry architecture that later participates in:

* **ADR-009 — Workflow**

The Marketplace does not replace the Plugin SDK, Capability Registry, AI SDK, Memory SDK, or Workflow.

---

# 19. Decision Outcome

PRIDE shall implement the Marketplace as an **Official Interface and Registry-oriented bridge** between Plugin developers and Plugin users.

The Marketplace shall not become a Plugin source-code database.

Plugin implementations remain in developer-controlled repositories.

The Marketplace generates standardized Plugin Registries.

Published Registry information is submitted to the official PRIDE Registry Repository through a Pull Request.

Users discover Plugins through the Marketplace and obtain Plugin implementations using the locations specified by the Registry.

The Plugin SDK remains responsible for Plugin installation.

This architecture allows PRIDE to provide a unified Plugin Marketplace experience without taking ownership of Plugin implementations or requiring centralized Plugin hosting.
