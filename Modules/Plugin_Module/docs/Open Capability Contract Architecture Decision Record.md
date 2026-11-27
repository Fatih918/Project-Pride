# Open Capability Contract Architecture Decision Record

**ADR ID:** PLG-ADR-0004  
**Status:** Accepted  
**Decision date:** 2026-11-27  
**Decision owners:** PRIDE Architecture  
**Supersedes:** None  
**Amends:** ADR-004 — Capability Registry; ADR-013 — Unlimited Plugin Extensibility and Composition  
**Related decisions:** PLR-ADR-0002 — Plugin Lifecycle Persistence; PLR-ADR-0003 — Plugin Runtime Execution; PLG-ADR-0003 — Platform Metadata Resolution (if referenced by repository naming)

---

## 1. Context

ADR-004 establishes an open-ended capability model: capability identifiers are namespaced and are not a closed Core enumeration. ADR-013 establishes that Plugins may extend PRIDE in ways that are not known in advance, while Core remains the platform kernel and the authority for workflow, governance, and authorization.

The existing Plugin implementation has a deliberately smaller baseline:

- `domain.Capability` currently contains `ID`, `Name`, and `Description`.
- Capability declarations are part of Marketplace-owned `platform_metadata`, not the developer-owned Plugin Manifest.
- Artifact descriptor schema `1.1.0` requires capability entries containing `id`, `name`, and `description`.
- The descriptor's Ed25519 signature binds platform metadata to the immutable release.
- The current domain validation checks non-empty and duplicate capability identifiers, but does not yet implement the full namespace grammar or contract compatibility model.

These existing semantics remain valid for their schema version. This ADR establishes the normative direction for the next capability-contract increment; it does not silently reinterpret already signed descriptor schema `1.1.0` artifacts.

## 2. Decision

PRIDE adopts an open, versioned, data-described Capability Contract. The contract allows independently developed Plugins to declare interoperable capabilities without requiring Core source changes merely to recognize new identifiers.

The decisions below are normative.

### C1 — Canonical Identifiers and Namespace Ownership

1. A capability identifier MUST use the form `<namespace>/<capability-name>`.
2. Both namespace and capability name consist of dot-separated segments.
3. Each segment MUST use lowercase ASCII letters and digits, with hyphens permitted internally. A segment MUST begin and end with a letter or digit.
4. A namespace MUST contain at least two segments.
5. Identifiers are case-sensitive. Uppercase and malformed identifiers MUST be rejected; implementations MUST NOT silently lowercase or normalize them.
6. PRIDE reserves namespaces under `io.pride`. Third parties MUST NOT publish identifiers in reserved namespaces.
7. Namespace ownership MUST be established by the trusted Registry/Marketplace namespace-claim process. Syntactic validity alone does not prove ownership.
8. Capability identifiers are immutable. A semantic rename MUST create a new identifier. Any future aliasing MUST be explicit Registry metadata and MUST NOT rely on implicit string normalization.

### C2 — Independent Capability Contract Versioning

1. Every capability declaration MUST include an explicit `contract_version`.
2. Capability contract versions MUST use Semantic Versioning 2.0.0.
3. Capability contract versioning is independent of Plugin version, Manifest schema version, artifact descriptor schema version, and platform metadata schema version.
4. A major version denotes a breaking change to observable input/output semantics or invocation guarantees.
5. A minor version adds backward-compatible optional behavior or optional schema fields.
6. A patch version clarifies or tightens implementation-neutral details without changing the accepted input/output contract.
7. Consumers MUST declare an explicit compatible version range or exact version requirement. A Plugin version MUST NOT substitute for a capability contract version.
8. Compatibility MUST be evaluated using the contract version and schemas, not string equality alone. The range grammar and matching algorithm MUST be deterministic and MUST be specified by the normative contract specification before production compatibility decisions are enabled.

### C3 — Declaration Authority and Trust

1. The authoritative capability declaration MUST remain in the signed Registry artifact descriptor's `platform_metadata`.
2. The Marketplace/Registry MUST validate namespace ownership and attest the declaration before signing the descriptor.
3. The descriptor signature MUST bind the capability identifier, contract version, schema references or inline schemas, and compatibility declarations to the immutable Plugin release.
4. A package or Plugin MUST NOT elevate its own trust by declaring a capability. Any developer-supplied proposal is non-authoritative until attested in signed Registry metadata.
5. Capability metadata MUST be treated as trusted only after successful verification of the descriptor signature and artifact digest.
6. The Plugin Module MUST preserve the verified declaration as metadata. Core MUST independently evaluate whether it understands and supports the declared contract.

### C4 — Machine-Readable Input and Output Contracts

1. Capability input and output contracts MUST use JSON Schema Draft 2020-12.
2. PRIDE MUST define and implement a documented, bounded subset of that vocabulary for the initial production contract.
3. Unsupported schema keywords that affect validation semantics MUST NOT be silently ignored.
4. Schema validation MUST be deterministic and MUST fail closed when a required schema feature cannot be validated.
5. Schema references MUST be immutable and cryptographically bound to the verified Plugin release. Mutable external URLs and arbitrary network schema resolution are prohibited.
6. The preferred initial representation is package-relative schema references within the verified artifact, subject to normalized-path containment and the package digest. Inline schemas MAY be used only within explicit descriptor-size bounds.
7. Schema resolution MUST NOT perform network access during installation, discovery, matching, or invocation.
8. Human-readable names and descriptions are descriptive metadata and MUST NOT substitute for machine-readable input/output semantics.

### C5 — Separate Compatibility and Invocation Checks

A capability declaration MAY include:

- Capability identifier and contract version.
- Input and output schema references.
- A bounded invocation-mode identifier from a platform-defined vocabulary, where that mode has normative semantics.
- Platform/runtime compatibility constraints expressed through a versioned, typed structure.
- Optional descriptive metadata.

Compatibility and invocation eligibility MUST be evaluated as distinct stages:

1. Identifier match.
2. Contract-version compatibility.
3. Input/output schema compatibility.
4. Platform and runtime compatibility.
5. Plugin trust validity.
6. Plugin lifecycle and runtime availability.
7. Core authorization for the concrete principal, task, and invocation context.

Passing one stage MUST NOT imply passing any later stage. A matching identifier is not proof that a capability is executable.

### C6 — Validation and Resource Bounds

1. Capability identifiers, names, descriptions, versions, schema references, and compatibility structures MUST have explicit byte or length limits.
2. The maximum number of capability declarations per Plugin MUST remain bounded. The existing limit of 256 is retained for the first contract increment.
3. Schema document size, reference count, nesting depth, schema evaluation complexity, and aggregate schema bytes MUST be bounded.
4. Duplicate capability identifiers within one Plugin release MUST be rejected.
5. Duplicate JSON object keys MUST be rejected.
6. Invalid, ambiguous, unsupported, or over-limit declarations MUST fail closed. Implementations MUST NOT silently truncate or partially accept a declaration.
7. Exact numeric limits MUST be published in the normative contract specification and versioned schemas. They MUST NOT be left to implementation discretion.

### C7 — Forward Compatibility and Unknown Contracts

1. Unknown fields MAY be preserved only where the owning versioned data model explicitly permits them and the implementation can preserve them safely.
2. Unknown fields MUST remain data and MUST NOT be interpreted as instructions.
3. Unknown capability identifiers and unsupported contract versions MAY be stored, displayed, and discovered as inert metadata.
4. A consumer that does not understand a capability contract MUST NOT invoke it, synthesize its input contract, or infer semantics from its identifier or description.
5. Unknown required schema keywords or invocation semantics MUST make a declaration unsupported for execution, not implicitly compatible.
6. An implementation MUST NOT claim lossless unknown-field preservation if it discards those fields.

### C8 — Permission Metadata and Core Authorization

1. Capability declarations and permission declarations MUST remain separate metadata concepts.
2. Permission IDs identify requested or Marketplace-attested permission categories; they MUST NOT grant access.
3. Core MUST remain the sole platform authorization authority.
4. Authorization MUST be evaluated for the concrete invocation context and relevant principal.
5. A Plugin MUST NOT inherit permissions from another Plugin or from a declared dependency.
6. The Runtime host MUST enforce authorization decisions and scoped resource boundaries.
7. Capability schemas, names, descriptions, and dependency metadata MUST NOT be interpreted as authorization policy.

### C9 — Declarative Dependencies; Composition Deferred

1. A capability contract MAY declare required or optional dependency capabilities using namespaced identifiers and explicit version constraints.
2. Dependency declarations are declarative requirements. They MUST NOT be interpreted as direct Plugin references, execution instructions, or authorization grants.
3. Dependency graph format, cycle handling, conflict resolution, invocation routing, partial failure, and execution ordering MUST be defined by a separate Plugin Composition Contract.
4. Until that contract is accepted and implemented, dependency metadata MAY be preserved but MUST NOT trigger automatic cross-Plugin invocation.
5. Composition MUST remain Core-planned and host-executed through explicit, versioned, authorized contracts. Plugins MUST NOT bypass the Plugin Module discovery boundary.

### C10 — Capability Contracts Are Independent of Plugin Categories

1. Capability contracts MUST be independent of descriptive Plugin categories such as Active, Passive, or Agentic.
2. UI extension and primary-interface replacement MUST be governed by a dedicated Extension Point / Interface Replacement Contract. Arbitrary UI capabilities MUST NOT be treated as ordinary task-execution capabilities by default.
3. Agentic orchestration, planning, multi-agent coordination, and AI-provider routing MAY be exposed through ordinary versioned capabilities and composition contracts.
4. No capability category MUST receive implicit access to Core workflow, AI credentials, other Plugins, user data, or host resources.
5. The capability model MUST NOT hard-code a closed taxonomy of future Plugin types or use cases.

### C11 — Versioned Schema Evolution and Legacy Compatibility

1. Capability contract versioning MUST remain independent from artifact descriptor and platform metadata schema versioning.
2. Adding required capability contract fields MUST use a new platform metadata schema version and a new artifact descriptor schema version.
3. Existing signed descriptor schema `1.1.0` semantics MUST remain unchanged.
4. Legacy declarations containing only `id`, `name`, and `description` MUST NOT be treated as fully compatible with the new capability contract.
5. The normative implementation specification MUST explicitly define whether legacy descriptors remain installable, discoverable, or only recoverable. No implicit reinterpretation is permitted.
6. Migration MUST preserve exact signed descriptor bytes for recovery verification and MUST NOT rewrite signed metadata in place.
7. Versioned schemas, examples, and conformance fixtures MUST be published alongside each supported schema version.

## 3. Ownership Boundaries

| Responsibility | Owner |
|---|---|
| Namespace claims, declaration review, attestation, signed descriptor publication | Marketplace / Registry |
| Bounded parsing, signature and digest verification, schema-reference containment, verified metadata preservation, immutable release binding, discovery | Plugin Module |
| Identifier matching, contract compatibility, platform policy, provider selection, dependency planning, deterministic Execution Plan | Core Capability Registry |
| Authoritative per-invocation authorization decisions | Core Authorization |
| Process isolation, scoped resource enforcement, authenticated invocation, enforcement of authorization decisions | Runtime Host |
| Developer-facing typed declarations and invocation contracts without host/Core internals | Plugin SDK |

No component MAY infer executable semantics from arbitrary Plugin metadata.

## 4. Security and Compatibility Consequences

- Open-ended identifiers enable ecosystem growth without a Core enum update for every new capability.
- Namespace ownership prevents syntactically valid but unauthorized namespace claims from being treated as legitimate.
- Signed declarations bind capability metadata to immutable Plugin releases.
- Schema references cannot introduce mutable external dependencies or network resolution during sensitive operations.
- Unknown capabilities remain discoverable but inert until a consumer explicitly supports their contract.
- Capability compatibility, lifecycle availability, trust, and authorization remain separate checks.
- Declarative dependencies do not create implicit cross-Plugin privilege inheritance or invocation.
- Existing schema 1.1.0 descriptors retain their original signed meaning and require an explicit compatibility policy.

## 5. Explicit Non-Goals

This ADR does not:

- Enumerate all current or future PRIDE capabilities.
- Define the business semantics of individual capabilities.
- Implement Core capability resolution or provider selection.
- Implement JSON Schema parsing, validation, or loading.
- Define the complete Plugin Composition protocol.
- Define UI extension points or primary-interface replacement.
- Define Agentic planning, tool-use, or multi-agent protocols.
- Add SDK lifecycle or execution RPCs.
- Change runtime isolation, authorization ownership, or Core workflow invariants.
- Modify the existing descriptor schema or runtime behavior by itself.

## 6. Required Follow-Up Specifications

The following work is required before production capability execution:

1. Publish the normative Open Capability Contract and its supported JSON Schema vocabulary.
2. Define exact resource limits, schema-reference rules, and deterministic contract-version range matching.
3. Publish a new versioned platform metadata schema and artifact descriptor schema without changing schema 1.1.0 semantics.
4. Define explicit legacy descriptor install/discovery/recovery compatibility.
5. Update domain types, descriptor parsing, validation, persistence, and conformance tests.
6. Define the Plugin Composition Contract.
7. Define the Extension Point / Interface Replacement Contract.
8. Audit Core Capability Registry and RuntimePort against the accepted contracts.
9. Implement conformance tests before enabling capability-based execution in production.

## 7. Acceptance

This ADR records the explicit approval of recommendations C1–C11 from the Open Capability Contract Decision Review. All eleven recommendations are accepted without modification as of 2026-11-27.

Acceptance establishes normative architectural decisions. It does not claim that the corresponding runtime, schemas, SDK, Core resolution, or production execution features are already implemented.
