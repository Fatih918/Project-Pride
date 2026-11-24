# ADR-0011 — Plugin Lifecycle

**Status:** Accepted

---

# Context

Plugins are the primary mechanism for extending PRIDE.

Every capability beyond the Core should be delivered through Plugins.

To maintain consistency, security, and predictability, every Plugin follows a standardized lifecycle managed by the Core.

A unified lifecycle ensures users always understand what happens before, during, and after a Plugin executes.

---

# Decision

PRIDE adopts a standardized **Plugin Lifecycle**.

Every Plugin progresses through a defined lifecycle regardless of its purpose or implementation.

The Core exclusively manages lifecycle transitions.

Plugins interact with the lifecycle through the Plugin SDK.

---

# Plugin Lifecycle

A Plugin may progress through the following stages:

1. Discovery
2. Installation
3. Verification
4. Permission Approval
5. Activation
6. Execution
7. Suspension
8. Update
9. Deactivation
10. Uninstallation

Not every Plugin experiences every stage during a single session.

---

# Discovery

Plugins may originate from:

* PRIDE Marketplace
* Local packages
* Repository integrations
* Organization repositories
* Other trusted Plugin sources

Discovery does not imply trust.

Every Plugin must complete Verification before execution.

---

# Installation

Installing a Plugin copies the required files into the local Plugin environment.

Installation alone does not activate the Plugin.

---

# Verification

Before activation, PRIDE verifies:

* Package integrity
* Manifest validity
* SDK compatibility
* Dependency availability
* Platform compatibility

Plugins failing verification must never execute.

---

# Permission Approval

Plugins requesting protected capabilities require explicit user approval.

Permissions should clearly describe:

* Requested capability
* Purpose
* Potential impact

Permission approval remains independent from installation.

---

# Activation

Activation prepares the Plugin for execution.

Plugins should initialize only the resources necessary for operation.

Expensive background work should be deferred whenever practical.

Activation does not imply execution.

---

# Execution

Once activated,

Plugins execute only when requested through the Core workflow.

Plugins communicate exclusively through published SDK interfaces.

Plugins must never communicate directly with:

* Core internals
* AI Providers
* Internal platform components outside their published SDK interfaces

---

# Suspension

Inactive Plugins may enter a suspended state.

Suspended Plugins remain installed while consuming minimal resources.

Execution may resume without requiring reinstallation.

---

# Update

Plugin updates should preserve compatibility whenever reasonably practical.

Updated Plugins must complete Verification again before becoming Active.

If new permissions become necessary,

PRIDE must request additional user approval before enabling those capabilities.

---

# Deactivation

Users may disable Plugins without uninstalling them.

Disabled Plugins retain their configuration unless the user explicitly requests removal.

---

# Uninstallation

Removing a Plugin removes its executable components.

User-created data should remain unless complete removal is explicitly requested.

---

# Lifecycle Ownership

The Core exclusively manages Plugin lifecycle transitions.

Plugins may react to lifecycle events,

but Plugins must never control their own lifecycle state.

This guarantees predictable platform behavior.

---

# Design Principles

The Plugin Lifecycle should remain:

* Predictable
* Transparent
* Permission-driven
* Recoverable
* Consistent

Users should always understand the current state of every installed Plugin.

---

# Rationale

A shared lifecycle creates a consistent experience across the entire ecosystem.

Users benefit from predictable behavior.

Developers benefit from standardized implementation patterns.

The Core remains responsible for orchestration.

Plugins remain responsible for functionality.

---

# Consequences

## Positive

* Consistent Plugin behavior
* Better resource management
* Improved security
* Easier debugging
* Predictable developer experience
* Scalable Plugin ecosystem
* Stable lifecycle management

## Trade-offs

* Plugins must follow lifecycle conventions.
* Lifecycle management increases Core responsibilities.
* SDK implementations become more structured.

These trade-offs are considered acceptable in exchange for long-term platform stability.

---

# Alternatives Considered

## Plugin-defined Lifecycle

Allow every Plugin to define its own lifecycle.

**Rejected.**

This creates inconsistent behavior across the ecosystem.

---

## Immediate Execution After Installation

Automatically execute Plugins immediately after installation.

**Rejected.**

Verification and Permission Approval must always occur before execution.

---

# Related Foundation

* Stable Foundations
* Unlimited Evolution
* Human Authority

---

# Related ADR

* ADR-0001 — Plugin-First Architecture
* ADR-0004 — Permission-Based Security Model
* ADR-0006 — User-Approved Task Execution
* ADR-0010 — Extensible SDK Architecture

---

This decision establishes a unified lifecycle for every PRIDE Plugin.

The Core owns the Plugin lifecycle.

Plugins respond to lifecycle events through the Plugin SDK.

This separation provides predictable behavior, stronger security, and a consistent developer experience while preserving the extensibility of the PRIDE platform.
