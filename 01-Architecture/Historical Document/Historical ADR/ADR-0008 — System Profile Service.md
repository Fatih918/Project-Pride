# ADR-0008 — System Profile Service

**Status:** Accepted

---

## Context

Many platform decisions depend on the capabilities of the host system.

Examples include:

* Initial platform setup
* Local model compatibility
* Plugin compatibility
* Marketplace compatibility
* Planning decisions
* Resource estimation

Allowing every Plugin to independently inspect the operating system would create duplicated logic, inconsistent results, and unnecessary complexity.

A unified System Profile is therefore required.

---

## Decision

PRIDE adopts a **System Profile Service** within the Core.

The Core is responsible for collecting and exposing a standardized, read-only representation of the host system.

Platform components and Plugins consume this information through the **System Profile API**.

---

## Principles

The System Profile represents the capabilities of the current host environment.

It does not make decisions.

It does not install software.

It does not recommend Providers.

It does not recommend Plugins.

Its sole responsibility is providing accurate, consistent, and authoritative system information.

---

## Responsibilities

The System Profile may expose information such as:

* Operating System
* CPU
* Memory (RAM)
* GPU
* GPU Memory (VRAM)
* Storage Capacity
* Available Storage
* System Architecture
* Installed Runtimes
* Hardware Acceleration Support
* AI Acceleration Technologies
* Network Availability

Additional properties may be introduced without breaking compatibility.

---

## Read-Only Design

The System Profile is read-only.

Platform components and Plugins may consume the System Profile.

They must never modify it.

The Core remains the single authoritative source of system information.

---

## Consumers

Examples of components that may consume the System Profile include:

* Setup Wizard
* Planning
* Capability Registry
* AI SDK
* Marketplace
* Plugin Installer

Each consumer remains responsible for making its own decisions.

The System Profile only supplies information.

---

## Rationale

A centralized System Profile eliminates duplicated hardware detection while ensuring every component operates using the same understanding of the host environment.

This improves consistency, maintainability, and future extensibility across the entire platform.

---

## Consequences

### Positive

* Single source of truth
* Consistent hardware detection
* Reduced Plugin complexity
* Better setup experience
* Improved compatibility detection
* Simplified marketplace filtering
* Stable platform architecture

### Trade-offs

* The Core becomes responsible for hardware detection.
* Platform-specific detection requires maintenance.
* The System Profile API should remain stable over time.

These trade-offs are considered acceptable in exchange for consistency and long-term maintainability.

---

## Alternatives Considered

### Plugin-Based Detection

Allow every Plugin to detect system information independently.

**Rejected.**

This creates duplicated implementation, inconsistent behavior, and unnecessary maintenance.

---

### External Detection Service

Delegating system detection to an external application.

**Rejected.**

System information is fundamental to platform operation and should remain part of the Core.

---

## Related Foundation

* Stable Foundations
* Unlimited Evolution
* Human First

---

## Related ADR

* ADR-0002 — Core Infrastructure Principle
* ADR-0005 — Provider Abstraction Layer
* ADR-0006 — User-Approved Task Execution

---

This decision establishes a unified **System Profile Service** for PRIDE.

The Core provides a consistent, authoritative, and read-only view of the host environment.

Platform components and Plugins remain responsible for deciding how that information is used.

The System Profile provides facts.

The rest of the platform provides decisions.
