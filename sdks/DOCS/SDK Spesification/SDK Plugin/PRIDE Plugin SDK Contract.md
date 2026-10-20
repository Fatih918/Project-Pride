# PRIDE Plugin SDK Contract

**Version:** 1.0.0
**Status:** Draft
**Category:** SDK Contract
**Document ID:** PRIDE-PLUGIN-SDK-CONTRACT
**Related Specification:** PRIDE SDK Specification
**Related Standard:** PRIDE SDK Code Standard

---

# 1. Purpose

This document defines the public contract of the PRIDE Plugin SDK.

The Plugin SDK provides the standardized programming interface through which PRIDE platform components communicate with the Plugin ecosystem.

The contract defines the standardized representation of Plugin operations, requests, responses, errors, metadata, lifecycle operations, execution operations, and Plugin state information.

---

# 2. Contract Boundary

The Plugin SDK establishes the official communication boundary between PRIDE platform components and the Plugin ecosystem.

```text
Platform Component
        │
        ▼
    Plugin SDK
        │
        ▼
   Plugin Module
        │
        ▼
Plugin Implementation
```

Platform components shall not bypass the Plugin SDK to communicate directly with Plugin implementations, Plugin runtimes, or Plugin internals.

---

# 3. Contract Principles

The Plugin SDK follows these principles:

1. One official communication boundary.
2. Implementation independence.
3. Standardized Plugin requests.
4. Standardized Plugin responses.
5. Standardized Plugin errors.
6. Capability-oriented Plugin interaction.
7. Explicit lifecycle operations.
8. Controlled Plugin state access.
9. No Plugin implementation logic in the SDK.
10. No direct platform-to-Plugin invocation.

---

# 4. Plugin SDK Operations

The Plugin SDK provides standardized interfaces for:

```text
Plugin SDK

├── Discovery
├── Lifecycle
├── Execution
├── Metadata
└── State
```

These responsibilities are established by the Plugin architecture.

---

# 5. Plugin Request

A Plugin Request represents an operation requested by a PRIDE platform component.

Logical structure:

```text
PluginRequest

├── request_id
├── operation
├── plugin_id
├── input
└── metadata
```

---

## 5.1 request_id

`request_id` uniquely identifies the requested operation.

Requirements:

* Required.
* Unique within the applicable execution context.
* Preserved throughout the operation.
* Returned in the corresponding response or error.

---

## 5.2 operation

`operation` identifies the standardized Plugin SDK operation being requested.

The initial operation categories are:

```text
DISCOVER
LIFECYCLE
EXECUTE
METADATA
STATE
```

The detailed payload for each operation is defined by the corresponding operation contract.

---

## 5.3 plugin_id

`plugin_id` identifies the Plugin involved in the operation.

The SDK shall treat the Plugin identity as an identifier, not as an implementation reference.

The public contract shall not require consumers to understand the Plugin's internal runtime, package structure, or implementation technology.

---

## 5.4 input

`input` contains operation-specific data.

Its structure depends on the requested operation.

The SDK shall not expose implementation-specific input structures.

---

## 5.5 metadata

`metadata` contains standardized contextual information required by the operation.

Metadata shall not be used to bypass the Plugin SDK contract.

---

# 6. Discovery Contract

Discovery allows platform components to obtain standardized information about available Plugins.

A discovery request shall not execute a Plugin.

Logical result:

```text
Discovery Request
        ↓
Plugin SDK
        ↓
Plugin Module
        ↓
Plugin Information
```

The discovery result may contain:

* Plugin identity.
* Plugin availability.
* Declared capabilities.
* Version information.
* Compatibility information.
* Other standardized Plugin metadata.

The exact Plugin Manifest and Registry structures are outside this contract unless explicitly referenced by the applicable specification.

---

# 7. Lifecycle Contract

The Plugin SDK provides standardized lifecycle operations.

Lifecycle operations may include:

```text
INSTALL
ENABLE
DISABLE
START
STOP
UNINSTALL
```

The exact lifecycle state machine belongs to the Plugin Module specification.

The SDK provides the communication contract for requesting lifecycle operations.

The SDK shall not implement lifecycle behavior itself.

---

# 8. Execution Contract

Execution allows an authorized platform component to request execution of a Plugin capability.

Logical structure:

```text
Execution Request

├── request_id
├── plugin_id
├── capability
├── input
└── metadata
```

The `capability` identifies what functionality is requested.

The SDK does not define how the Plugin performs the capability internally.

This preserves the separation:

```text
Plugin SDK
    │
    └── Communication

Plugin Module
    │
    └── Plugin management

Plugin
    │
    └── Functionality
```

The SDK shall not become responsible for executing Plugin internals.

---

# 9. Capability Boundary

Plugin capabilities are declared by Plugins and consumed by platform components through standardized SDK operations.

A platform component shall request a capability rather than depending on the internal implementation of a Plugin.

The Capability Registry is responsible for capability selection and Execution Plan generation.

The Plugin SDK is responsible for communication with the selected Plugin.

Therefore:

```text
Capability Registry
        │
        │ selects capability / Plugin
        ▼
Execution Plan
        │
        ▼
Plugin SDK
        │
        ▼
Plugin Module
        │
        ▼
Plugin
```

The Capability Registry does not execute Plugins.

---

# 10. Metadata Contract

The Plugin SDK may expose standardized Plugin metadata.

Metadata may include:

* Plugin identity.
* Plugin name.
* Plugin version.
* Declared capabilities.
* Compatibility information.
* Availability information.
* State information.
* Other contract-defined metadata.

Metadata shall describe the Plugin rather than expose its internal implementation.

---

# 11. State Contract

Plugin state remains the responsibility of the Plugin Module.

The Plugin SDK may expose standardized state information to platform components.

Platform components shall not directly manipulate Plugin state through implementation-specific mechanisms.

State transitions remain controlled by the Plugin Module.

The SDK therefore distinguishes between:

```text
State Information
        ≠
State Ownership
```

The SDK may communicate state information without becoming the owner of Plugin state.

---

# 12. Plugin Response

A successful SDK operation shall return a standardized Plugin Response.

Logical structure:

```text
PluginResponse

├── request_id
├── operation
├── status
├── output
└── metadata
```

---

## 12.1 request_id

The response shall contain the identifier of the originating request.

---

## 12.2 operation

The response shall identify the operation that produced the result.

---

## 12.3 status

`status` identifies whether the requested operation completed successfully.

The SDK shall not require consumers to interpret implementation-specific status values.

---

## 12.4 output

`output` contains the standardized result of the requested operation.

The output structure is operation-specific.

---

## 12.5 metadata

Response metadata may provide standardized information relevant to the completed operation.

Implementation-specific runtime objects shall not be exposed through the public response.

---

# 13. Plugin Error

A failed operation shall return a standardized Plugin SDK Error.

Logical structure:

```text
PluginError

├── request_id
├── error_code
├── message
└── metadata
```

---

# 14. Standard Error Categories

The Plugin SDK shall normalize implementation failures into contract-level errors.

Initial error categories include:

```text
PLUGIN_NOT_FOUND
PLUGIN_UNAVAILABLE
PLUGIN_DISABLED
PLUGIN_NOT_READY
CAPABILITY_NOT_FOUND
INVALID_REQUEST
INVALID_INPUT
PERMISSION_DENIED
INCOMPATIBLE_PLUGIN
LIFECYCLE_ERROR
EXECUTION_ERROR
TIMEOUT
INTERNAL_ERROR
UNKNOWN_ERROR
```

The exact error taxonomy may be extended while preserving compatibility.

---

# 15. Error Message

The `message` field provides a human-readable description of the failure.

Consumers shall use `error_code` for programmatic handling.

Error messages shall not expose unnecessary:

* Internal implementation details.
* Runtime internals.
* File paths.
* Credentials.
* Secrets.
* Private Plugin infrastructure.

---

# 16. Plugin Identity

The Plugin SDK shall use stable Plugin identity.

A Plugin identity shall not depend on:

* Runtime process identifiers.
* Memory addresses.
* File paths.
* Installation locations.
* Framework-specific identifiers.

Plugin identity shall remain stable across compatible Plugin implementation changes.

---

# 17. Implementation Independence

The Plugin SDK shall remain independent from:

* Programming language.
* Runtime environment.
* Plugin framework.
* Packaging implementation.
* Storage mechanism.
* Execution technology.
* Vendor-specific implementation.

A Plugin may change its internal implementation without requiring platform components to change, provided that the Plugin remains compatible with the SDK contract.

---

# 18. Plugin Categories

The SDK boundary applies to both Plugin categories defined by the architecture.

## 18.1 Active Plugins

Active Plugins provide executable functionality and participate in workflow execution.

Examples include:

* File System.
* Browser.
* Git.
* Calculator.
* Python Runtime.
* Terminal.

Active Plugins are executed through the Plugin SDK.

---

## 18.2 Passive Plugins

Passive Plugins extend platform behavior without directly participating in workflow execution.

Examples include:

* Custom User Interface.
* Memory Explorer.
* Dashboard.
* Sidebar.
* Theme.
* Settings Panel.

Passive Plugins are managed through the Official Interface and Plugin SDK and are not executed by the Executor.

---

# 19. Direct Invocation Prohibition

The following architecture is prohibited:

```text
Platform Component
        │
        ▼
      Plugin
```

The required architecture is:

```text
Platform Component
        │
        ▼
    Plugin SDK
        │
        ▼
   Plugin Module
        │
        ▼
      Plugin
```

Every Plugin operation must pass through the Plugin SDK.

---

# 20. SDK Responsibility Boundary

The Plugin SDK is responsible for:

* Standardized Plugin communication.
* Request validation.
* Request representation.
* Response representation.
* Error representation.
* Plugin discovery communication.
* Lifecycle operation communication.
* Execution operation communication.
* Metadata access.
* State information access.

The Plugin SDK is not responsible for:

* Plugin functionality.
* Plugin internal execution.
* Plugin implementation management.
* Plugin storage.
* Plugin packaging.
* Marketplace behavior.
* Registry architecture.
* Capability selection.
* Workflow planning.
* Workflow execution strategy.

These responsibilities belong to their respective platform components and specifications.

---

# 21. Marketplace Boundary

The Plugin SDK does not define Marketplace behavior.

Marketplace publication, discovery, Registry generation, installation, and distribution remain governed by the Marketplace and Registry architecture.

The Marketplace operates around Plugin Registries while Plugin implementations remain in developer-controlled repositories.

The Plugin SDK therefore does not become:

* A Marketplace.
* A Plugin Registry.
* A Plugin source repository.
* A Plugin distribution service.

---

# 22. Capability Registry Boundary

The Capability Registry selects compatible capabilities and Plugins and produces an Execution Plan.

The Plugin SDK does not perform capability selection.

The separation is:

```text
Capability Registry
        │
        └── Selects

Plugin SDK
        │
        └── Communicates

Plugin Module
        │
        └── Manages

Plugin
        │
        └── Provides functionality
```

This preserves the separation between selection, communication, management, and functionality.

---

# 23. Compatibility

A Plugin implementation is SDK-compatible when it satisfies the applicable Plugin SDK Contract.

Internal implementation changes shall not require platform changes when public SDK behavior remains compatible.

Contract changes shall follow the applicable SDK versioning rules.

---

# 24. Conformance

A Plugin SDK implementation conforms to this contract when it:

1. Provides the defined public SDK boundary.
2. Accepts standardized Plugin requests.
3. Produces standardized Plugin responses.
4. Produces standardized Plugin errors.
5. Supports the defined Plugin operations.
6. Does not expose Plugin implementation details.
7. Prevents direct platform-to-Plugin communication outside the SDK boundary.
8. Preserves Plugin implementation independence.
9. Maintains compatibility with compliant Plugins.

---

# 25. Contract Model

The complete communication model is:

```text
                         PRIDE PLATFORM
                              │
                              ▼
                     ┌─────────────────┐
                     │   Plugin SDK    │
                     └────────┬────────┘
                              │
              ┌───────────────┼───────────────┐
              │               │               │
              ▼               ▼               ▼
          Discovery       Lifecycle       Execution
              │               │               │
              └───────────────┼───────────────┘
                              ▼
                     ┌─────────────────┐
                     │  Plugin Module  │
                     └────────┬────────┘
                              │
                              ▼
                     Plugin Implementation
```

The SDK provides the stable communication contract.

The Plugin Module manages Plugin lifecycle and implementation boundaries.

The Plugin provides the actual functionality.

---

# 26. Contract Principle

The Plugin SDK follows one fundamental rule:

> **The platform requests Plugin capabilities through the SDK; the Plugin implementation remains behind the boundary.**

The SDK defines **how PRIDE communicates with Plugins**.

It does not define **how Plugins implement their functionality**.

This preserves the Plugin-First architecture, implementation independence, and long-term extensibility of PRIDE.
