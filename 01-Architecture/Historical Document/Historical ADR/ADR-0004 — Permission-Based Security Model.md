# ADR-0004 — Permission-Based Security Model

**Status:** Accepted

---

## Context

PRIDE is designed as an extensible platform where functionality is primarily provided through plugins.

Plugins may require access to sensitive resources such as the file system, network, clipboard, microphone, camera, local storage, or external providers.

Granting unrestricted access by default conflicts with PRIDE's principles of privacy, transparency, and user control.

A consistent permission model is therefore required.

---

## Decision

PRIDE adopts a **Permission-Based Security Model**.

Plugins receive no sensitive capabilities by default.

Capabilities must be explicitly requested by the plugin and explicitly granted by the user.

Permission requests should be transparent, understandable, and reviewable before installation.

---

## Principles

Permissions are **granted**, not assumed.

Users remain in control of every capability provided to a plugin.

Permissions may be revoked at any time.

Plugins must continue operating with reduced functionality whenever reasonably practical after permission revocation.

Core services must verify permissions before exposing protected functionality through the SDK.

Plugins should not access protected resources outside the SDK.

---

## Permission Declaration

Every plugin should declare the permissions it requires within its manifest.

Examples include, but are not limited to:

* Internet
* Filesystem Read
* Filesystem Write
* Notifications
* Clipboard Read
* Clipboard Write
* Microphone
* Camera
* Workspace Access
* Provider Access

The permission declaration exists to inform users before installation.

---

## Installation Experience

Before installation, PRIDE should present:

* Plugin identity
* Publisher information
* Requested permissions
* Permission descriptions
* Verification status when available

The installation process should enable informed decision-making rather than requiring blind trust.

---

## Permission Changes

If a plugin update introduces new permissions, PRIDE should request user approval before those permissions become active.

Previously granted permissions remain unchanged unless explicitly modified by the user.

---

## Permission Revocation

Users should be able to review, grant, or revoke permissions after installation.

Permission changes should take effect without requiring plugin reinstallation whenever technically practical.

---

## Rationale

User control is a permanent principle of PRIDE.

Providing transparent permission management enables users to understand what plugins can access without unnecessarily restricting platform extensibility.

This model also encourages plugin developers to request only the permissions genuinely required.

---

## Consequences

### Positive

* Greater user control
* Improved transparency
* Better privacy protection
* Reduced plugin privileges
* More trustworthy plugin ecosystem
* Clear security boundaries

### Trade-offs

* Plugin development requires permission planning.
* SDK permission enforcement becomes critical.
* Permission documentation must remain accurate.
* Users may occasionally experience additional permission prompts.

These trade-offs are considered acceptable in exchange for stronger security and greater user autonomy.

---

## Alternatives Considered

### Full Trust by Default

Granting all permissions during installation.

**Rejected.**

This unnecessarily increases security risk and reduces user control.

---

### Permanent Denial

Preventing plugins from requesting sensitive capabilities.

**Rejected.**

This would significantly reduce platform extensibility and limit legitimate plugin functionality.

---

## Related Foundation

* Privacy
* Transparency
* Ethics
* Constitution (Human Authority)

## Related ADR

* ADR-0001 — Plugin-First Architecture
* ADR-0002 — Core Infrastructure Principle
* ADR-0003 — Event-Driven Architecture

---

This decision establishes the security model for PRIDE.

Capabilities are never assumed.

They are explicitly granted by the user and continuously remain under the user's control.
