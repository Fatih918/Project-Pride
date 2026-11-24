# ADR-0009 — Workspace Architecture

**Status:** Accepted

---

## Context

PRIDE is designed to support projects, conversations, memories, plugins, AI providers, user-generated artifacts, and long-term workflows.

These resources require a consistent organizational model that remains independent of individual Plugins.

Without a unified Workspace model, Plugins would create isolated storage structures, duplicate configuration, and reduce interoperability.

A shared Workspace Architecture is therefore required.

---

## Decision

PRIDE adopts a **Workspace Architecture**.

Every user activity occurs within a Workspace.

A Workspace acts as the logical container for projects, conversations, memories, artifacts, Plugin data, configuration, and user resources.

Plugins interact with the Workspace exclusively through the **Workspace API** provided by the Core.

---

## Principles

The Workspace belongs to the user.

The Core manages the Workspace.

Plugins consume Workspace services.

No Plugin owns the Workspace.

---

## Workspace Responsibilities

A Workspace may contain:

* Projects
* Conversations
* Workspace-specific Memories
* Documents
* Generated files
* Plugin configuration
* AI Provider preferences
* Assets
* Logs
* User preferences

Additional resource types may be introduced without changing the architecture.

---

## Workspace Isolation

Each Workspace remains logically isolated.

Plugins should only access the currently active Workspace unless explicitly authorized by the user.

This improves organization, privacy, predictability, and security.

---

## Workspace API

The Core provides a unified **Workspace API**.

Plugins interact with Workspace resources through this API rather than accessing internal storage directly.

The Workspace API remains provider-independent and storage-independent.

---

## Workspace Lifecycle

The Workspace lifecycle is managed exclusively by the Core.

The Core is responsible for:

* Creating Workspaces
* Opening Workspaces
* Closing Workspaces
* Importing Workspaces
* Exporting Workspaces
* Deleting Workspaces

Plugins never manage the Workspace lifecycle.

Plugins only interact with the currently active Workspace through the Workspace API.

---

## Portability

Workspaces should support migration between devices whenever technically practical.

Platform components should avoid assumptions tied to specific operating systems, storage implementations, or filesystem layouts.

This ensures long-term portability across future environments.

---

## Plugin Integration

Plugins may create resources inside the active Workspace.

Plugins should avoid creating arbitrary directories outside the Workspace unless explicitly approved by the user.

Following shared Workspace conventions improves interoperability between Plugins while maintaining a consistent user experience.

---

## Rationale

A unified Workspace creates a predictable environment for both users and developers.

Projects remain organized while Plugins cooperate through shared Workspace services without requiring knowledge of one another's internal storage structures.

Centralizing Workspace management inside the Core also improves consistency, portability, and long-term maintainability.

---

## Consequences

### Positive

* Consistent project organization
* Simplified Plugin development
* Better interoperability
* Easier migration
* Reduced duplicated storage logic
* Improved long-term maintainability
* Stable Workspace lifecycle

### Trade-offs

* The Core becomes responsible for the Workspace lifecycle.
* Workspace APIs require long-term stability.
* Plugins should follow shared Workspace conventions.

These trade-offs are considered acceptable in exchange for consistency, interoperability, and extensibility.

---

## Alternatives Considered

### Plugin-Owned Storage

Allow every Plugin to manage its own independent storage model.

**Rejected.**

This reduces interoperability, duplicates storage logic, and increases architectural fragmentation.

---

### Global Storage Without Workspaces

Store all resources globally without logical isolation.

**Rejected.**

This reduces organization, portability, scalability, and long-term project management.

---

## Related Foundation

* Stable Foundations
* Open Collaboration
* Unlimited Evolution

---

## Related ADR

* ADR-0002 — Core Infrastructure Principle
* ADR-0003 — Event-Driven Architecture
* ADR-0004 — Permission-Based Security Model

---

This decision establishes the Workspace as the primary organizational unit within PRIDE.

Workspaces belong to users.

The Core manages the Workspace lifecycle.

Plugins consume Workspace services through the Workspace API.

This separation ensures a stable, portable, and extensible platform architecture.
