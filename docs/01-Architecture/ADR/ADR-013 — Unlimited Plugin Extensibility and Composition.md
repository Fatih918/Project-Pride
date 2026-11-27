# ADR-013 — Unlimited Plugin Extensibility and Composition

**Status:** Accepted  
**Date:** 2026-11-27  
**Category:** Platform Architecture  
**Decision ID:** ADR-013  
**Decision owners:** PRIDE project owner  
**Depends on:** PRIDE Foundation; ADR-001 — Plugin-First Architecture; ADR-002 — Plugin SDK as the Official Plugin Communication Boundary; ADR-003 — Official Interface as the Platform Interaction Layer; ADR-004 — Capability Registry; ADR-005 — Execution Architecture; ADR-009 — Stable Core Workflow; ADR-010 — PRIDE SDK Ecosystem Architecture; ADR-011 — Plugin Module Foundation Design; PLR-ADR-0002; PLR-ADR-0003

---

## 1. Context

PRIDE's Foundation already establishes the direction of **Unlimited Evolution**, a stable Core, replaceable implementations, Plugin-first development, and freedom from artificial architectural limits.

In particular:

- The Constitution establishes that the Core remains stable and capabilities should be extended rather than added to the Core whenever practical.
- The Vision states that PRIDE should be a platform without artificial limits.
- The Design Principles establish Stable Core, Unlimited Extensions; Everything Replaceable; Build Once, Extend Forever; and Design for Unknown Futures.
- The project philosophy describes PRIDE as a foundation that enables others to build, improve, replace, and extend what exists.

Those Foundation statements define the purpose and values. This ADR makes their implications for the Plugin architecture explicit and normative.

The current Plugin architecture already provides important foundations for third-party extensions: identity, manifests, signed artifacts, registry and discovery, installation, lifecycle, SDK boundaries, and an out-of-process runtime direction.

However, a narrow interpretation of Plugin as only an integration package or a feature add-on would not fully express the intended platform.

A Plugin must be able to introduce capabilities that PRIDE's original authors did not anticipate. It may provide a feature, replace a platform experience, coordinate other Plugins, orchestrate multiple AI systems, or compose existing capabilities into a larger system.

Examples include, but are not limited to:

- A Passive Plugin that contributes, replaces, or disables the Official Interface implementation.
- An Agentic Plugin that coordinates multiple AI providers or agents, decomposes tasks, creates plans, and manages its own agentic workflow.
- A Plugin that composes capabilities exposed by several other Plugins.
- A Plugin that introduces a new capability category or a new user experience without requiring a Core implementation change.

PRIDE must therefore distinguish between a stable platform kernel and an open, composable Plugin ecosystem.

## 2. Decision

PRIDE adopts **Unlimited Plugin Extensibility and Composition** as a normative architecture principle.

The Core is the platform kernel responsible for the stable operation, coordination, governance, and security of PRIDE.

Plugins are the primary mechanism through which PRIDE gains new functionality, intelligence, experiences, integrations, and higher-level systems.

The architecture shall permit Plugins to extend, compose, and replace eligible non-kernel platform implementations through explicit, versioned contracts. The set of Plugin capabilities and use cases must not be treated as a closed list defined by the current implementation.

The phrase **Unlimited Plugin Extensibility** expresses an architectural direction: PRIDE must not impose artificial product or implementation limits on what a compliant Plugin may build. It does not mean unrestricted access to host resources, implicit trust, or the absence of platform contracts.

## 3. Accepted Decisions

### D1 — Core as the Platform Kernel

**Accepted: Core is the platform kernel, not a container for all platform features.**

Core responsibilities include the stable platform responsibilities required to operate PRIDE, including:

- Canonical workflow orchestration and platform coordination.
- Identity and principal context.
- Authentication and authorization authority.
- Platform policy and trust enforcement.
- Capability registration, resolution, and routing boundaries.
- Plugin ecosystem governance and stable platform contracts.
- Coordination of platform lifecycle and runtime governance.
- Controlled communication between platform components.
- Resource and security boundaries required to protect the platform.

Existing ownership boundaries remain authoritative. For example, the Plugin Module owns Plugin-specific lifecycle invariants and lifecycle orchestration; the host owns process and OS isolation mechanisms; Core remains the authorization authority.

Core must not absorb feature-specific business logic merely because that feature is useful or commonly installed. A capability that can be provided as a Plugin without violating a fundamental platform responsibility should remain outside Core.

### D2 — Open-Ended Plugin Capability

**Accepted: Plugin capabilities are open-ended, not a closed platform enum.**

A Plugin may declare and expose capabilities that were not known when the current Core version was developed.

Capability identifiers must support namespaced ownership, versioned contracts, metadata, and compatibility evaluation. The platform must be able to register, retain, discover, and reason about unknown or newly introduced capability identifiers without requiring a Core source-code change solely to recognize their existence.

Recognition is not equivalent to executable support: a consumer may invoke a capability only when it understands and supports the corresponding contract and the request is authorized. Unknown capability metadata must not be interpreted as executable instructions.

The Core owns platform-level capability governance and authorization. The Plugin Module owns Plugin registry and discovery responsibilities. Plugin implementations own the behavior behind their declared capabilities.

### D3 — Replaceability of Non-Kernel Components

**Accepted: Almost all non-Core platform components must be replaceable or extensible through official contracts.**

Replaceability is a design requirement, not a promise that every internal implementation can be swapped without compatibility work.

Eligible replacement targets include, but are not limited to:

- User interface and interaction experiences.
- AI providers, models, and orchestration engines.
- Agentic and automation systems.
- Feature-specific workflow implementations.
- Integrations and external service adapters.
- Analytics, search, document, and knowledge capabilities.
- Storage and communication integrations.
- Other non-kernel services whose responsibilities can be expressed through stable contracts.

A replacement must preserve the platform contracts and invariants required by its consumers. Registration, activation, priority, conflict resolution, compatibility, authorization, rollback, and fallback behavior must be defined by the relevant extension-point contract.

Core security, identity, trust, authorization, and other fundamental kernel authorities are not replaceable by an ordinary Plugin. A Plugin may contribute implementations or policy inputs where explicitly supported, but it must not silently acquire or replace platform authority.

### D4 — Passive Plugin and Official Interface Replaceability

**Accepted: Passive Plugins may provide, replace, or disable the Official Interface implementation through a governed replacement mechanism.**

The Official Interface is an implementation of the platform interaction experience, not an immutable requirement that prevents alternative experiences.

A Passive Plugin may:

- Contribute interface extensions hosted by the Official Interface.
- Register a complete alternative interface implementation.
- Become the selected primary interface when the user or authorized platform policy activates it.
- Cause the built-in Official Interface implementation to be disabled while the replacement is active, where the replacement satisfies the required interface contract.

The platform must retain a trusted recovery and administration path. A replacement must not be able to remove the user's ability to regain control, manage Plugins, review security decisions, or recover from a broken interface. The exact recovery surface may be a protected platform mode or another separately specified mechanism.

The logical interaction boundary and workflow rules remain platform responsibilities even when the visual interface implementation changes. User-facing components must not bypass the platform workflow or authorization boundaries.

This decision supersedes any interpretation of ADR-003 that makes the built-in Official Interface implementation permanently mandatory or non-replaceable. ADR-003's separation of presentation from workflow and service implementation, its Plugin SDK boundary, and its unified interaction goals remain applicable. ADR-003 must be reconciled in a follow-up documentation change to express this replacement model precisely.

### D5 — Agentic Plugin and Intelligence Extensibility

**Accepted: Agentic systems are valid Plugin implementations, not mandatory Core subsystems.**

An Agentic Plugin may independently implement and operate capabilities such as:

- Multi-agent orchestration.
- Multiple AI provider/model selection and coordination.
- Task decomposition and delegation.
- Planning and plan revision.
- Agent collaboration and result synthesis.
- Tool and Plugin invocation.
- Workflow management and evaluation.
- Domain-specific reasoning and automation.

PRIDE provides the platform foundations that make these implementations possible, such as the AI SDK boundary, Plugin runtime, capability contracts, authorization, and controlled communication.

PRIDE Core is not required to contain a universal agent orchestrator, planner, multi-agent scheduler, or domain-specific agent framework. Those may be developed as Plugins and may themselves be composed with other Plugins.

Agentic Plugins remain subject to user authority, platform authorization, runtime isolation, and applicable confirmation policies. Their autonomy does not grant them authority to bypass Core governance.

### D6 — Plugin Composition

**Accepted: Plugins must be composable with other Plugins through explicit platform contracts.**

A Plugin may consume capabilities exposed by other Plugins and may expose higher-level capabilities that combine them.

The architecture must support a Plugin composition model with:

- Explicit Plugin and capability dependencies.
- Immutable version constraints and compatibility checks.
- Dependency resolution and conflict reporting.
- Authenticated, authorized Plugin-to-Plugin invocation.
- Contract and data-version compatibility.
- Cancellation, deadlines, bounded payloads, and typed failures.
- Failure containment and partial-failure handling.
- Dependency-cycle detection and deterministic resolution behavior.
- Independent lifecycle and version identity for each Plugin.

Composition does not imply shared memory, direct database access, filesystem access, process access, or unrestricted calls between Plugin implementations. Communication must use supported contracts and must be authorized by the platform.

A composed Plugin must not silently inherit the permissions of its dependencies. Each capability invocation is evaluated under the applicable principal, policy, and explicit grants.

### D7 — Plugin Categories Are Non-Exhaustive

**Accepted: Plugin categories describe common roles and must not become a closed type system.**

Integration, Feature, Passive, and Agentic are useful examples, not an exhaustive taxonomy or mandatory mutually exclusive types.

A single Plugin may provide multiple roles and capabilities. Future Plugin roles may emerge without changing the fundamental Plugin identity or requiring a new hard-coded Plugin type in Core.

The architecture must distinguish descriptive categorization from executable capability contracts, runtime requirements, and authorization metadata. A category label alone must never grant access or determine execution behavior.

### D8 — Open Implementation, Controlled Authority

**Accepted: Freedom of implementation is broad; access to privileged platform resources remains explicitly governed.**

Plugins may choose their internal architecture, libraries, AI providers, frameworks, algorithms, and domain-specific workflows, subject to the supported runtime and compatibility contracts.

Plugins do not receive implicit access to:

- Core databases or internal repository handles.
- Credential stores, raw secrets, signing keys, or trust roots.
- Other Plugin processes or private data.
- Unrestricted host filesystem or network access.
- Platform identity or authorization internals.
- Privileged runtime and operating-system controls.

Privileged capabilities must be explicit, scoped, revocable, auditable, and authorized by Core. The host enforces the granted boundary. Permission declarations in a Manifest or signed descriptor remain metadata and are not themselves grants.

This is the security interpretation of **No Lock-in**: no artificial lock-in to a particular feature, provider, implementation, framework, or ecosystem, while preserving explicit platform authority and isolation.

### D9 — Stable Contracts and Independent Evolution

**Accepted: Extensibility must be achieved through stable, versioned, implementation-neutral contracts.**

Plugin-facing contracts must be independent from internal Core implementation details and must support:

- Explicit contract and schema versioning.
- Backward and forward compatibility rules.
- Namespaced identifiers.
- Typed requests, results, and failures.
- Bounded inputs and outputs.
- Capability and dependency compatibility evaluation.
- Safe behavior when optional or unknown metadata is encountered.

A new Plugin must not require a Core release merely because it introduces a new implementation or capability identifier. A Core change is required only when a genuinely new kernel-level contract, authority, or invariant is needed.

### D10 — Developer and Ecosystem Independence

**Accepted: Independent developers and organizations must be able to build meaningful PRIDE extensions without depending on Core internals.**

The Plugin ecosystem must provide public, documented, testable contracts and developer tooling sufficient to build, validate, package, sign or obtain trusted publication, install, compose, operate, and update Plugins.

Official Marketplace distribution must not be the only architectural source of Plugins. Official Registry, private registries, and other explicitly supported distribution channels may provide artifacts under the same trust and compatibility requirements.

The platform must avoid unnecessary dependence on PRIDE-maintained implementations. A community Plugin must be able to replace an official non-kernel implementation when it satisfies the applicable contract and policy.

## 4. Normative Architectural Invariants

The following invariants are established by this ADR:

1. Core is the platform kernel and remains responsible for stable platform operation, workflow coordination, governance, and security authority.
2. Plugins are the primary mechanism for extending PRIDE's functionality and intelligence.
3. The set of Plugin capabilities, categories, and use cases is open-ended.
4. Almost all non-kernel platform implementations must be designed for replacement or extension through explicit contracts.
5. The built-in Official Interface implementation is replaceable by a compliant Passive Plugin through a governed mechanism.
6. PRIDE must preserve a trusted recovery and administration path when an alternative interface is active.
7. Agentic orchestration and multi-AI coordination may be implemented by Plugins; they are not mandatory Core subsystems.
8. Plugins must be composable through explicit, versioned, authorized contracts.
9. A Plugin does not inherit the permissions or authority of another Plugin merely by depending on it.
10. Plugin categories are descriptive and non-exhaustive; they must not become a closed Core enum.
11. Plugin implementation freedom does not grant unrestricted access to privileged host or Core resources.
12. Core authorization remains authoritative; the host enforces runtime isolation and granted capabilities.
13. New non-kernel capabilities should not require Core source changes solely to register or recognize them.
14. Existing Plugin identity, immutable versioning, artifact verification, lifecycle, and runtime isolation invariants remain in force.
15. Future architectural decisions must be evaluated against the Foundation and this ADR.

## 5. Relationship With Existing Architecture Decisions

This ADR strengthens and operationalizes the existing Foundation and Plugin-first direction. It does not replace the Foundation.

- **ADR-001 — Plugin-First Architecture:** remains the foundational Plugin-first decision. This ADR expands the intended scope of Plugin extensibility and composition.
- **ADR-002 — Plugin SDK:** remains the official communication boundary. Composition and replacement must use official contracts rather than internal implementation access.
- **ADR-003 — Official Interface:** its presentation/workflow separation and Plugin hosting model remain applicable. Its assumption that the built-in Official Interface is permanently mandatory is superseded by D4 and requires reconciliation.
- **ADR-004 — Capability Registry:** remains the capability governance foundation. Its implementation and contract must be checked against the open-ended capability requirement in D2.
- **ADR-005 — Execution Architecture:** remains applicable to platform execution boundaries. Plugin composition does not authorize bypassing the execution and authorization model.
- **ADR-009 — Stable Core Workflow:** remains the canonical workflow authority. Plugins may implement and compose capabilities within that workflow; they do not silently replace or bypass the Core workflow authority.
- **ADR-010 — PRIDE SDK Ecosystem Architecture:** remains the SDK ecosystem foundation. Public SDK contracts should evolve to support the open Plugin ecosystem.
- **ADR-011 — Plugin Module Foundation Design:** remains the implementation foundation, subject to follow-up reconciliation where its initial scope or assumptions are narrower than this ADR.
- **PLR-ADR-0002 — Lifecycle Persistence:** remains authoritative for durable Plugin lifecycle state, operation idempotency, leases, fencing, and recovery.
- **PLR-ADR-0003 — Runtime Execution:** remains authoritative for out-of-process execution, RuntimePort ownership, immutable runtime identity, host-enforced isolation, and runtime supervision.

Where an earlier document conflicts with this ADR's explicit decisions on Plugin replaceability, open-ended capabilities, or composition, this ADR is authoritative for the Plugin ecosystem's long-term architecture. Existing security, lifecycle, runtime, and ownership contracts remain authoritative unless explicitly amended by a later accepted decision.

## 6. Consequences

### Positive Consequences

- **Long-term extensibility:** PRIDE can support capabilities and Plugin use cases not anticipated by the current team.
- **Reduced Core coupling:** feature development does not automatically expand the Core.
- **Replaceable experiences:** users and organizations can choose alternative interfaces and non-kernel implementations.
- **Composable ecosystem:** developers can build higher-level systems from existing Plugins.
- **AI independence:** agentic and multi-provider systems can evolve outside Core.
- **Developer freedom:** external contributors can create substantial platform functionality without depending on Core internals.
- **Sustainable evolution:** the platform can replace technologies while preserving stable contracts and authority boundaries.

### Costs and Risks

- **Contract complexity:** extension points, capability contracts, and compatibility rules require careful design.
- **Composition complexity:** dependency resolution, cycles, authorization propagation, and failure isolation require dedicated architecture.
- **Replacement safety:** interface and service replacement require conflict handling, fallback, rollback, and recovery mechanisms.
- **Security burden:** broader extensibility increases the importance of isolation, least privilege, supply-chain verification, and host enforcement.
- **Compatibility governance:** open-ended capability registration must not imply that arbitrary unknown operations can be executed.
- **Developer experience:** SDKs, testing tools, documentation, and compatibility tooling become essential platform deliverables.

These costs are accepted because extensibility and ecosystem independence are central PRIDE goals. They must be addressed through explicit contracts and staged implementation rather than by weakening the platform boundaries.

## 7. Implementation Guidance and Required Follow-up

This ADR establishes direction and invariants. It does not claim that the following capabilities are already implemented.

The following work must be specified in separate contracts and implementation increments:

1. Reconcile ADR-003 with replaceable Official Interface selection, protected recovery access, activation, fallback, and rollback.
2. Audit ADR-004 and the Capability Registry implementation for open-ended identifiers, namespacing, versioning, and compatibility.
3. Define Plugin dependency and composition contracts, including dependency graphs, cycle detection, invocation, authorization, and failure semantics.
4. Define general extension-point registration and replacement contracts, including priority, conflict resolution, activation, compatibility, and recovery.
5. Define Passive Plugin UI hosting and full-interface replacement contracts.
6. Reconcile the Plugin SDK public contracts with cross-Plugin composition and the intended developer ecosystem.
7. Audit the RuntimePort and runtime host design for composable Plugins and different Plugin roles without weakening PLR-ADR-0003 isolation requirements.
8. Define capability grant and host capability brokering semantics for Plugin-to-Plugin and Plugin-to-platform interactions.
9. Define developer tooling and conformance tests for third-party Plugin authors.
10. Revisit the implementation roadmap only after the above contracts identify concrete gaps.

Implementation must remain incremental. This ADR does not authorize a broad rewrite of existing Plugin Module code or an immediate expansion of the Core.

## 8. Explicit Non-Goals

This ADR does not:

- Implement a universal Agentic AI framework inside Core.
- Require PRIDE to ship a replacement Official Interface.
- Grant Plugins unrestricted access to Core or the host.
- Make every unknown capability automatically executable.
- Require all Plugins to depend on one another.
- Mandate a specific UI framework, AI provider, runtime engine, IPC protocol, database, or programming language.
- Change the existing Plugin artifact format, signature scheme, lifecycle schema, or RuntimePort signatures by itself.
- Declare the current Plugin Module production-ready.
- Replace the PRIDE Foundation or alter its constitutional principles.

## 9. Decision Outcome

PRIDE shall be built as a stable platform kernel surrounded by an open, replaceable, and composable Plugin ecosystem.

**Core defines how the platform operates. Plugins define what the platform can become.**

PRIDE must not limit the future to the capabilities, categories, interfaces, or implementations known today. Its responsibility is to provide the stable contracts, governance, security, and execution foundations that allow independent developers and future generations to build beyond the original platform.

This ADR makes the Foundation's principle of Unlimited Evolution concrete for the Plugin architecture.
