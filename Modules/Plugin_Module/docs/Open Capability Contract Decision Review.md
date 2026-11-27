# Open Capability Contract Decision Review

**Status:** Proposed — pending architecture approval  
**Date:** 2026-11-27  
**Scope:** Capability identity, declaration, contract compatibility, schema binding, and safe extensibility  
**Related decisions:** ADR-004 — Capability Registry; ADR-013 — Unlimited Plugin Extensibility and Composition; PLR-ADR-0002 — Lifecycle Persistence; PLR-ADR-0003 — Runtime Execution

---

## 1. Purpose

ADR-004 establishes that capability identifiers are open-ended, namespaced, and not a closed Core enum. It also requires a separate contract specification for capability schemas, namespace ownership, versioning, and validation limits.

This review proposes the decisions needed to turn that architectural direction into a stable, implementable contract.

The objective is not to enumerate every capability PRIDE may ever support. The objective is to define a small, durable envelope through which independently developed Plugins can declare capabilities, describe machine-readable contracts, and interoperate without requiring Core source changes solely to recognize new identifiers.

This review does not define the complete Plugin Composition protocol, UI replacement protocol, Agentic orchestration protocol, or runtime IPC protocol. Those remain separate contracts.

## 2. Current Baseline and Gaps

The current implementation and schemas establish the following baseline:

- `domain.Capability` currently contains only `ID`, `Name`, and `Description`.
- `domain.ProtectedMetadata` stores capability declarations alongside permission IDs and compatibility status.
- `registry-artifact-descriptor.schema.json` is version `1.1.0`; its signed `platform_metadata.capabilities` entries currently require only `id`, `name`, and `description`.
- Capability declarations are carried in Marketplace/platform metadata covered by the artifact descriptor's Ed25519 signature.
- `domain.Plugin.Validate` currently checks non-empty and duplicate capability IDs, but does not enforce a canonical namespace grammar or contract semantics.
- Capability declarations are not part of the developer-owned `plugin.json` Manifest.
- Existing artifact verification establishes descriptor signature and package digest integrity; it does not by itself establish that a capability contract is semantically supported or safe to invoke.

These are intentionally narrow primitives, not yet a complete Open Capability Contract. This review proposes additive evolution. It does not authorize silently changing the meaning of already accepted descriptor schema `1.1.0`.

## 3. Decision Questions

### C1 — Capability Identifier Grammar and Namespace Ownership

**Recommendation:**

1. Capability identifiers are canonical, lowercase, globally namespaced identifiers.
2. Use the form:

   `<namespace>/<capability-name>`

3. Both parts consist of dot-separated segments. Each segment uses lowercase ASCII letters and digits, with hyphens allowed internally; a segment must begin with a letter or digit and end with a letter or digit.
4. A namespace must contain at least two segments. Example: `io.pride.platform/ui.replace`, `com.example.files/read`, `dev.acme.agent/plan`.
5. Identifiers are case-sensitive by specification but uppercase is invalid; implementations must not silently lowercase or normalize malformed input.
6. PRIDE reserves namespaces under `io.pride`. Third parties may not publish identifiers in reserved namespaces.
7. Namespace ownership is established by the trusted Registry/Marketplace namespace-claim process. A syntactically valid namespace is not proof of ownership.
8. Capability identifiers are immutable. A semantic rename creates a new identifier; aliases, if ever needed, are explicit Registry metadata and not implicit string normalization.

**Rationale:** A stable grammar prevents collisions and ambiguous matching while keeping the namespace open-ended. Ownership must be established outside the Plugin's self-declared metadata.

**Alternatives considered:**
- Unqualified names: rejected because of collisions.
- Reverse-DNS-only identifiers without a capability separator: rejected because the namespace boundary becomes ambiguous.
- Core-maintained identifier registry: rejected because it would reintroduce a closed catalog.

### C2 — Capability Contract Versioning

**Recommendation:**

1. Every capability declaration has an explicit `contract_version`, independent of Plugin version, Manifest schema version, artifact descriptor version, and platform metadata schema version.
2. Use Semantic Versioning 2.0.0 for capability contracts.
3. A major version denotes a breaking change to observable input/output semantics or invocation guarantees.
4. A minor version adds backward-compatible optional behavior or optional schema fields.
5. A patch version clarifies or tightens implementation-neutral details without changing the accepted input/output contract.
6. Consumers declare an explicit compatible version range or exact version requirement. A provider's Plugin version must never be used as a substitute for its capability contract version.
7. Compatibility is evaluated against the contract version and schemas, not by string equality alone. The exact range grammar and matching algorithm must be deterministic and specified by the contract implementation.

**Rationale:** Plugin release cadence and capability semantics evolve independently. Separating versions allows a Plugin to update without forcing a capability contract break, and allows multiple Plugins to implement the same contract version.

**Alternatives considered:**
- Reuse Plugin SemVer: rejected because Plugin release and capability semantics are different lifecycles.
- Date-based versions: rejected because they do not express compatibility as clearly.
- Unversioned capabilities: rejected because consumers cannot safely reason about contract compatibility.

### C3 — Capability Declaration Ownership and Trust

**Recommendation:**

1. The authoritative capability declaration remains in the signed Registry artifact descriptor's `platform_metadata`, consistent with the accepted platform-metadata architecture.
2. The Marketplace/Registry is responsible for validating namespace ownership and attesting the declaration before signing the descriptor.
3. The descriptor signature binds the capability identifier, contract version, schemas/references, and compatibility declarations to the immutable Plugin release.
4. A package or Plugin may not elevate its own trust by declaring a capability. A developer-supplied proposal, if introduced later, is input to Marketplace review only and is not authoritative until attested in signed metadata.
5. Capability metadata becomes trusted only after the existing descriptor signature and artifact digest verification succeeds.
6. The Plugin Module preserves the verified declaration as metadata. Core independently evaluates whether it understands and supports the declared contract.

**Rationale:** This preserves PR #73's signed platform metadata boundary and prevents a mutable or self-asserted declaration from becoming trusted merely because it appears in a package.

### C4 — Input and Output Contract Representation

**Recommendation:**

1. A capability contract declares machine-readable input and output schemas using JSON Schema Draft 2020-12.
2. The contract defines the expected top-level data shape and semantic constraints; human-readable name and description remain descriptive only.
3. The initial supported schema vocabulary is a documented, bounded subset of Draft 2020-12. Unsupported keywords must not be silently ignored when they affect validation semantics.
4. Schema validation must be deterministic and fail closed when the consumer cannot validate a required schema feature.
5. Schema references, if used, must be immutable and cryptographically bound to the verified Plugin release. Mutable external URLs and arbitrary network schema resolution are prohibited.
6. The preferred initial representation is package-relative schema references within the verified artifact, with strict normalized-path rules and content covered by the descriptor's package digest. Inline schemas may be used only where explicitly bounded by the descriptor-size policy.
7. Schema resolution must not perform network access during installation, discovery, matching, or invocation.

**Rationale:** JSON Schema provides a language-neutral data contract while the verified artifact digest binds referenced schemas to the same immutable release. Prohibiting network resolution avoids SSRF, mutable-schema, and availability risks.

**Alternatives considered:**
- Arbitrary HTTPS schema URLs: rejected because they are mutable and introduce network trust/SSRF concerns.
- Opaque prose-only input/output descriptions: rejected because they are not machine-verifiable.
- Arbitrary executable validation scripts: rejected because declarations must remain data, not code.

### C5 — Compatibility and Invocation Semantics

**Recommendation:**

A capability declaration may include:

- Capability identifier and contract version.
- Input and output schema references.
- A bounded invocation mode identifier from a platform-defined vocabulary, where the mode has normative semantics.
- Platform/runtime compatibility constraints expressed through a versioned, typed structure.
- Optional descriptive metadata.

Compatibility must be evaluated in distinct stages:

1. Identifier match.
2. Contract version compatibility.
3. Input/output schema compatibility.
4. Platform and runtime compatibility.
5. Plugin trust validity.
6. Plugin lifecycle and runtime availability.
7. Core authorization for the concrete principal, task, and invocation context.

Passing an earlier stage does not imply passing a later stage. Capability metadata is never an authorization grant.

**Rationale:** This formalizes the separation already required by ADR-004 and avoids treating a matching identifier as proof of executability.

### C6 — Validation and Resource Bounds

**Recommendation:**

1. Capability identifiers, names, descriptions, versions, schema references, and compatibility structures have explicit byte/length limits.
2. The maximum number of capability declarations per Plugin remains bounded; the current descriptor limit of 256 is retained for the first contract increment.
3. Schema document size, reference count, nesting depth, schema evaluation complexity, and total aggregate schema bytes are bounded.
4. Duplicate capability identifiers within one Plugin release are rejected.
5. Duplicate JSON object keys are rejected, consistent with existing descriptor parsing rules.
6. Invalid, ambiguous, unsupported, or over-limit declarations reject the relevant Plugin installation or descriptor validation; implementations must not silently truncate or partially accept them.
7. Exact numeric limits are finalized in the accepted contract specification and schema version, not left to implementation discretion.

**Rationale:** Open-ended extensibility must not become unbounded parser or validator resource consumption.

### C7 — Unknown Fields and Unknown Capability Contracts

**Recommendation:**

1. Preserve the existing forward-compatible unknown-field policy for versioned metadata objects where the schema version explicitly permits it.
2. Unknown fields are retained only when the owning data model can preserve them safely; they are never interpreted as instructions.
3. Unknown capability identifiers and unsupported contract versions may be stored, displayed, and discovered as inert metadata.
4. A consumer that does not understand a capability contract must not invoke it, synthesize an input contract, or infer semantics from its name or description.
5. Unknown required schema keywords or invocation semantics make that declaration unsupported for execution, not implicitly compatible.
6. A future schema revision may define how unknown optional fields are preserved across round trips; no implementation may claim lossless preservation if it discards them.

**Rationale:** Forward compatibility supports ecosystem growth, while inert treatment prevents unknown metadata from becoming executable behavior.

### C8 — Permission Metadata and Authorization

**Recommendation:**

1. Capability declarations and permission declarations remain separate metadata concepts.
2. Permission IDs identify requested or Marketplace-attested permission categories; they do not grant access.
3. Core remains the sole platform authorization authority, as established by the lifecycle and runtime ADRs.
4. Authorization is evaluated per concrete invocation context and relevant principal; a Plugin does not inherit permissions from another Plugin or declared dependency.
5. The Runtime host enforces the authorization decision and scoped resource boundary.
6. Capability schema fields, names, descriptions, and dependency metadata must never be interpreted as authorization policy.

**Rationale:** This preserves the existing Core authorization boundary and prevents capability composition from becoming privilege inheritance.

### C9 — Capability Dependencies and Composition

**Recommendation:**

1. The Open Capability Contract may identify a capability's required or optional dependency capabilities using namespaced identifiers and explicit version constraints.
2. Dependency declarations are declarative requirements, not direct Plugin references, execution instructions, or authorization grants.
3. The detailed dependency graph format, cycle handling, conflict resolution, invocation routing, partial failure, and execution ordering are deferred to a separate Plugin Composition Contract.
4. Until that contract is accepted, dependency metadata may be preserved but must not trigger automatic cross-Plugin invocation.
5. Composition remains Core-planned and host-executed through explicit versioned contracts; Plugins may not bypass the Plugin Module discovery boundary.

**Rationale:** Capability contracts need a stable way to describe dependencies, but defining graph execution here would duplicate and prematurely freeze the separate composition architecture.

### C10 — Passive, Active, and Agentic Plugin Semantics

**Recommendation:**

1. Capability contracts are independent from Plugin category labels such as Active or Passive.
2. UI extension and primary-interface replacement are governed by a dedicated Extension Point / Interface Replacement Contract, not by treating arbitrary UI capabilities as executable task capabilities.
3. Agentic orchestration, planning, multi-agent coordination, and AI-provider routing may be exposed through ordinary versioned capabilities and composition contracts.
4. No capability category receives implicit access to Core workflow, AI credentials, other Plugins, user data, or host resources.
5. The capability model must not hard-code a closed taxonomy of future Plugin types or use cases.

**Rationale:** This supports ADR-013's open-ended Plugin model without conflating task execution, user-interface extension, and platform-level interaction surfaces.

### C11 — Schema Evolution and Migration

**Recommendation:**

1. Capability contract versioning is independent from artifact descriptor and platform metadata schema versioning.
2. Adding the required capability contract fields is an additive metadata change and must use a new platform metadata schema version and a new artifact descriptor schema version; existing schema `1.1.0` semantics remain unchanged.
3. The implementation must explicitly define whether older descriptors remain installable, discoverable, or only recoverable. No implicit reinterpretation of legacy descriptors is allowed.
4. New installations must not treat legacy declarations containing only `id`, `name`, and `description` as fully compatible with the new contract.
5. Migration must preserve exact signed descriptor bytes for recovery verification and must not rewrite signed metadata in place.
6. Contract schema examples and conformance fixtures must be published alongside the versioned schema.

**Rationale:** The current descriptor parser is strict for capability object properties. Additive fields require a deliberate schema increment and compatibility policy, not an in-place mutation of accepted signed data.

---

## 4. Proposed Implementation Boundaries

If these recommendations are accepted:

- **Marketplace/Registry:** namespace ownership, declaration review/attestation, signed descriptor publication.
- **Plugin Module:** bounded parsing, signature/digest verification, schema-reference containment, metadata preservation, immutable release binding, discovery.
- **Core Capability Registry:** identifier matching, contract compatibility, platform policy, provider selection, dependency planning, deterministic Execution Plan.
- **Core Authorization:** authoritative per-invocation decisions.
- **Runtime Host:** process isolation, scoped resource enforcement, authenticated invocation, and enforcement of authorization decisions.
- **Plugin SDK:** developer-facing typed declarations and invocation contracts, without exposing host internals or Core implementation details.

No component may infer executable semantics from arbitrary Plugin metadata.

## 5. Explicit Non-Goals

This review does not:

- Enumerate all PRIDE capabilities.
- Define every capability's business semantics.
- Implement capability resolution in Core.
- Implement JSON Schema validation or schema loading.
- Define the complete Plugin Composition protocol.
- Define UI extension points or primary-interface replacement.
- Define Agentic planning, tool-use, or multi-agent protocols.
- Add SDK lifecycle or execution RPCs.
- Change runtime isolation, authorization ownership, or Core workflow invariants.
- Modify the existing descriptor schema or runtime behavior.

## 6. Approval Checklist

The following recommendations require explicit acceptance or revision:

- C1 — Identifier grammar and namespace ownership.
- C2 — Independent SemVer capability contract versions.
- C3 — Signed Registry descriptor as authoritative capability declaration.
- C4 — JSON Schema Draft 2020-12 bounded subset and immutable package-relative schema references.
- C5 — Separate compatibility stages and typed invocation/platform constraints.
- C6 — Explicit validation/resource bounds and fail-closed handling.
- C7 — Forward-compatible unknown fields with unknown capabilities remaining inert.
- C8 — Permission metadata remains separate from Core authorization.
- C9 — Dependency declarations allowed, full composition semantics deferred.
- C10 — Capability contracts remain independent from Plugin categories; UI replacement gets a separate contract.
- C11 — New descriptor/platform-metadata schema versions; no reinterpretation of existing signed schema 1.1.0 artifacts.

## 7. Expected Follow-Up Sequence

After approval:

1. Record accepted decisions in a Capability Contract Architecture Decision Record.
2. Publish the normative Open Capability Contract and versioned JSON schemas/examples.
3. Audit and update descriptor parsing, domain types, and validation with backward-compatibility tests.
4. Define the Plugin Composition Contract.
5. Define the Extension Point / Interface Replacement Contract.
6. Audit the Core Capability Registry and RuntimePort boundaries against the accepted contracts.
7. Implement conformance tests before enabling capability-based execution in production.
