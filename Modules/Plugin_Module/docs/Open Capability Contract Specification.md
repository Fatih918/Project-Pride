# PRIDE Open Capability Contract Specification

**Document ID:** POCS-0001  
**Version:** 1.0.0  
**Status:** Proposed for review  
**Date:** 2026-11-27  
**Module:** PRIDE Plugin Module / Core Capability Registry  
**Related decisions:** ADR-004 — Capability Registry; ADR-013 — Unlimited Plugin Extensibility and Composition; PLR-ADR-0004 — Open Capability Contract Architecture Decision Record; PLR-ADR-0003 — Plugin Runtime Execution Architecture

---

## 1. Purpose

This specification defines the first normative, machine-readable contract for declaring, validating, matching, and safely treating Plugin capabilities.

It converts PLR-ADR-0004 into implementable rules for:

- canonical capability identifiers and namespace ownership;
- independent Semantic Versioning and deterministic version requirements;
- signed Marketplace declaration metadata;
- input/output JSON Schema documents and bounded validation;
- platform/runtime compatibility declarations;
- declarative dependencies;
- unknown and legacy capability behavior;
- separation of compatibility, lifecycle, trust, and authorization.

This specification does not define the business semantics of individual capabilities. It does not define the complete Plugin Composition protocol, UI Extension Point / Interface Replacement contract, Agentic planning protocol, or runtime invocation wire protocol.

The terms **MUST**, **MUST NOT**, **REQUIRED**, **SHOULD**, **SHOULD NOT**, and **MAY** are normative.

## 2. Versioning and compatibility baseline

### 2.1 Independent version domains

PRIDE MUST treat these versions as independent:

| Version domain | Purpose |
|---|---|
| Plugin version | Version of a particular Plugin release |
| Manifest schema version | Shape of developer-owned `plugin.json` |
| Artifact descriptor schema version | Shape of the signed Registry descriptor |
| Platform metadata schema version | Shape of Marketplace-owned platform metadata |
| Capability contract version | Observable input/output and invocation semantics of one capability |
| This specification version | Rules governing the open capability contract |

A Plugin version MUST NOT be used as a substitute for a capability contract version. A descriptor schema upgrade MUST NOT imply that a capability contract has changed, and a capability contract upgrade MUST NOT imply that the Plugin's Manifest schema has changed.

### 2.2 First versioned descriptor increment

The first descriptor format carrying complete capability contracts MUST use:

- artifact descriptor `schema_version: 2.0.0` and `$schema: urn:pride:registry-artifact-descriptor:2`;
- platform metadata `schema_version: 2.0.0` and `$schema: urn:pride:platform-metadata:2`;
- capability declaration schema identifier `urn:pride:capability-declaration:1`;
- JSON Schema Draft 2020-12 for input and output schemas.

The existing artifact descriptor schema 1.1.0 and platform metadata schema 1.0.0 MUST retain their existing meaning. They MUST NOT be reinterpreted as schema 2.0.0 by adding inferred defaults or by rewriting signed descriptor bytes.

The versioned JSON Schema files, examples, and conformance fixtures for these identifiers are published in a separate schema-and-fixtures increment after this specification is accepted.

### 2.3 Legacy 1.1.0 behavior

A descriptor using the existing schema 1.1.0 MAY continue to be installed under its existing validation and cryptographic rules. Its existing `id`, `name`, and `description` capability entries MUST be classified as **legacy capability metadata**.

Legacy capability metadata:

1. MAY be displayed and returned by discovery.
2. MUST NOT be assigned an inferred contract version.
3. MUST NOT be treated as satisfying a versioned capability requirement.
4. MUST NOT be invoked through the versioned capability contract mechanism.
5. MUST NOT be upgraded in place or silently rewritten to schema 2.0.0.
6. MUST remain bound to the exact signed descriptor bytes for recovery verification.

Existing committed installations MUST NOT be removed merely because they contain legacy capability metadata. A PROMOTED installation operation MUST be recovered according to the descriptor version and verification rules recorded for that operation. A schema 1.1.0 descriptor is not the unsupported schema 1.0.0 legacy case already rejected by the installation implementation.

New installations using schema 2.0.0 MUST satisfy every requirement in this specification. A Plugin with legacy declarations cannot be treated as fully compatible with a versioned capability contract.

## 3. Capability declaration model

A schema 2.0.0 platform metadata document contains a bounded array of capability declaration objects. Each declaration MUST contain:

| Property | Required | Meaning |
|---|---:|---|
| `id` | Yes | Immutable namespaced capability identifier |
| `name` | Yes | Human-readable display name |
| `description` | Yes | Human-readable description |
| `contract_version` | Yes | Strict SemVer 2.0.0 version |
| `invocation_mode` | Yes | Platform-defined invocation semantic |
| `input_schema_ref` | Yes | Immutable package-relative input schema reference |
| `output_schema_ref` | Yes | Immutable package-relative output schema reference |
| `platform_compatibility` | No | Typed PRIDE/runtime compatibility constraints |
| `dependencies` | No | Declarative capability requirements |

No additional capability declaration properties are defined in version 1 of this specification. Implementations MUST reject unknown properties in this object. A future extension MUST use an explicitly versioned schema change; implementations MUST NOT infer semantics from unknown properties.

The authoritative declaration is Marketplace-owned `platform_metadata` inside the signed artifact descriptor. A developer-owned Manifest MAY contain descriptive or proposed information, but it MUST NOT override, replace, or elevate the signed declaration.

### 3.1 Capability identifiers

The canonical identifier form is:

```text
<namespace>/<capability-name>
```

Both parts consist of one or more dot-separated segments. The namespace MUST contain at least two segments. Each segment:

- MUST contain only lowercase ASCII letters (`a-z`), digits (`0-9`), and internal hyphens (`-`);
- MUST begin and end with a lowercase ASCII letter or digit;
- MUST NOT be empty;
- MUST NOT contain consecutive separators or leading/trailing hyphens.

Examples of syntactically valid identifiers:

- `io.pride.text.generate`
- `com.example.document.summarize`
- `org.acme.workflow.plan`
- `dev-team.tools.file-index`

Examples that MUST be rejected:

- `Text.Generate` — uppercase characters;
- `example/summarize` — namespace has fewer than two segments;
- `com..example/summarize` — empty segment;
- `com.example/-summarize` — segment begins with a hyphen;
- `com.example/summarize-` — segment ends with a hyphen;
- `com.example/summarize/` — empty capability name;
- `com.example/summarize name` — whitespace.

The complete identifier MUST be at most 255 ASCII bytes. Identifiers are case-sensitive. Producers and consumers MUST NOT lowercase, trim, Unicode-normalize, or otherwise rewrite an identifier before comparison.

The `io.pride` namespace and all of its descendants are reserved to PRIDE. Third parties MUST NOT claim or publish identifiers in that namespace. A syntactically valid third-party namespace is not trusted merely because it parses: the Registry/Marketplace namespace-claim process MUST establish ownership before attestation and signing.

Capability identifiers are immutable. A semantic rename MUST allocate a new identifier. Aliases, if introduced later, MUST be explicit signed Registry metadata and MUST NOT be implemented as implicit string normalization.

### 3.2 Human-readable metadata

- `name` MUST be valid UTF-8, non-empty, and at most 128 UTF-8 bytes.
- `description` MUST be valid UTF-8 and at most 1,024 UTF-8 bytes. An empty description is permitted.
- Neither property may contain control characters other than tab, line feed, and carriage return.
- Names and descriptions are descriptive only. They MUST NOT be parsed as instructions, used as authorization policy, or used to infer missing contract semantics.

### 3.3 Contract versions

Every declaration MUST carry a strict Semantic Versioning 2.0.0 version.

- The version MUST contain `MAJOR.MINOR.PATCH`.
- Leading zeroes in numeric identifiers are prohibited except for the number zero itself.
- Prerelease and build metadata MUST follow SemVer 2.0.0.
- The complete version string MUST be at most 64 ASCII bytes.
- Whitespace, a leading `v`, partial versions, wildcards, and non-SemVer aliases MUST be rejected.
- SemVer precedence MUST follow SemVer 2.0.0. Build metadata does not affect precedence or compatibility matching.

A major version denotes a breaking change to observable input/output semantics or invocation guarantees. A minor version adds backward-compatible optional behavior or optional schema fields. A patch version clarifies implementation-neutral details without changing the accepted input/output contract.

A publisher MUST NOT publish two different semantic contracts under the same capability identifier and contract version. Marketplace attestation MUST bind the declaration to the immutable Plugin release.

## 4. Deterministic version requirements

A consumer MUST declare an explicit version requirement when it depends on or requests a capability. The requirement grammar is intentionally narrower than package-manager range dialects.

### 4.1 Grammar

A requirement is either:

1. one exact SemVer version; or
2. a conjunction of one to four comparator clauses separated by exactly one or more ASCII spaces.

Each comparator consists of one operator followed immediately by a complete SemVer version:

- `=` exact precedence match;
- `>` greater than;
- `>=` greater than or equal;
- `<` less than;
- `<=` less than or equal.

If no operator is present, the complete version is an exact match.

Examples:

- `1.2.3`
- `=1.2.3`
- `>=1.2.0 <2.0.0`
- `>1.4.0 <=1.8.2`

The following MUST be rejected:

- `^1.2.3`, `~1.2.3`, `1.x`, `*`;
- OR clauses such as `1.2.3 || 2.0.0`;
- comma-separated clauses;
- partial versions such as `1.2`;
- empty clauses, duplicate contradictory operators, or more than four comparators.

The complete requirement MUST be at most 128 ASCII bytes.

### 4.2 Matching semantics

All comparator clauses MUST be true for a version to match. Comparisons MUST use SemVer 2.0.0 precedence, not lexical string ordering.

A prerelease candidate MUST NOT satisfy a requirement composed only of stable-version comparators unless at least one comparator in that requirement explicitly contains a prerelease version with the same major, minor, and patch tuple. This follows SemVer prerelease exclusion behavior and prevents accidental adoption of prerelease contracts.

Build metadata is ignored for precedence. Exact matching compares SemVer precedence; a consumer that needs byte-identical version text MUST express that as a separate Registry/release identity constraint.

A matching version requirement is necessary but not sufficient for compatibility. Consumers MUST separately validate the schema documents, invocation mode, platform/runtime constraints, trust, lifecycle availability, and authorization.

A consumer MUST NOT infer compatibility from equal major versions alone. The declared requirement MUST be evaluated exactly as written.

## 5. Invocation mode and platform compatibility

### 5.1 Invocation mode

Version 1 defines one invocation mode:

```text
io.pride.invocation.request-response
```

It represents one bounded request with one corresponding response over the authenticated host-managed Plugin invocation boundary.

A declaration MUST state this value explicitly. Any other value is unsupported in version 1 and MUST make the declaration ineligible for invocation. Unknown modes MAY be preserved as inert metadata only if the owning versioned model explicitly permits preservation; they MUST NOT be guessed or mapped to the request-response mode.

Streaming, event delivery, background jobs, callbacks, UI surfaces, and long-running workflow semantics are outside this initial invocation mode and require separately versioned contracts.

### 5.2 Platform compatibility

The optional `platform_compatibility` object MAY contain:

- `pride_version`: a version requirement using §4 grammar;
- `runtime_profile_ids`: an array of platform-defined runtime profile identifiers.

The object MUST contain at least one of these properties when present. Unknown properties MUST be rejected.

Each runtime profile identifier MUST be a canonical platform-owned identifier with a maximum length of 128 ASCII bytes. The platform profile registry is authoritative for the identifier's meaning. Unknown or unavailable required profiles make the declaration unsupported for execution; they MUST NOT be treated as unrestricted compatibility.

If `platform_compatibility` is absent, the declaration adds no publisher-specified compatibility constraint. This absence does not bypass the consumer's own PRIDE version, runtime, operating environment, security policy, or host support checks.

A compatibility declaration is descriptive input to Core policy. It MUST NOT grant access to runtime features, permissions, credentials, or host resources.

## 6. Input and output JSON Schema

### 6.1 Document requirements

Each capability MUST reference one input schema and one output schema. Both documents MUST use JSON Schema Draft 2020-12 and MUST be included in the same verified Plugin artifact.

The schema references MUST be package-relative paths under `package/schemas/`. The reference value is relative to the package root and MUST begin with `schemas/`. It MUST use forward slashes and MUST NOT contain:

- an absolute path, drive prefix, URI scheme, query, or fragment;
- empty, `.`, or `..` path segments;
- backslashes, percent-encoded separators, or NUL/control characters;
- a path outside the normalized `package/` payload.

The normalized path MUST be at most 240 UTF-8 bytes, consistent with PZIP-0001. The referenced file MUST exist exactly once in the verified archive. Duplicate normalized paths and case-folding collisions remain prohibited by PZIP-0001.

A schema document MUST:

- be valid UTF-8 JSON without a byte-order mark;
- reject duplicate JSON object keys at every nesting level;
- declare `"$schema": "https://json-schema.org/draft/2020-12/schema"`;
- have an object root and an explicit `type`;
- be at most 65,536 bytes;
- be immutable as part of the verified artifact digest.

A schema document MUST NOT be downloaded, resolved, or refreshed from a network location during installation, discovery, matching, or invocation. Schema bytes MUST be read only from the verified immutable release.

### 6.2 Supported vocabulary

The initial supported vocabulary is deliberately bounded. The following keywords are supported:

| Group | Keywords |
|---|---|
| Core | `$schema`, `$ref`, `$defs` |
| Type and value | `type`, `enum`, `const` |
| Object validation | `properties`, `required`, `additionalProperties`, `minProperties`, `maxProperties` |
| Array validation | `items`, `minItems`, `maxItems`, `uniqueItems` |
| String validation | `minLength`, `maxLength` |
| Numeric validation | `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf` |
| Applicators | `allOf`, `anyOf`, `oneOf`, `not` |

The supported `type` values are `object`, `array`, `string`, `number`, `integer`, `boolean`, and `null`. A type array MAY be used if it contains only these values and no duplicates.

A schema MAY use `$defs` and local JSON Pointer references of the form `#` or `#/$defs/<token>`. A reference MUST resolve within the same schema document. External references, relative-file references inside `$ref`, anchors, dynamic references, and cross-document reference graphs are not supported.

The following keywords are unsupported in version 1 and MUST cause the schema to be rejected for execution: `$id`, `$anchor`, `$dynamicAnchor`, `$dynamicRef`, `$recursiveRef`, `$recursiveAnchor`, `$vocabulary`, `$comment`, `title`, `description`, `default`, `examples`, `deprecated`, `readOnly`, `writeOnly`, `format`, `contentEncoding`, `contentMediaType`, `contentSchema`, `pattern`, `patternProperties`, `propertyNames`, `unevaluatedItems`, `unevaluatedProperties`, `contains`, `minContains`, `maxContains`, `if`, `then`, `else`, `dependentRequired`, and `dependentSchemas`.

Any unrecognized keyword MUST be rejected. Implementations MUST NOT silently ignore a keyword that may affect validation. An implementation MUST NOT claim Draft 2020-12 support merely because it accepts the `$schema` URI; it must enforce exactly the vocabulary and semantics defined here.

### 6.3 Reference and recursion rules

- `$ref` MUST use a local fragment reference only.
- JSON Pointer tokens MUST follow RFC 6901 escaping rules.
- Every reference target MUST exist.
- Reference cycles MUST be rejected in version 1, including cycles through `$defs`.
- A schema MUST NOT use a reference to escape its own document.
- Unreferenced `$defs` MAY be present but count toward all schema size and node limits.

### 6.4 Schema resource limits

The following limits are mandatory for version 1:

| Limit | Maximum |
|---|---:|
| Capability declarations per Plugin | 256 |
| Schema reference occurrences per Plugin | 512 |
| Distinct schema documents per Plugin | 128 |
| One schema document | 65,536 bytes |
| Aggregate schema document bytes per Plugin | 1,048,576 bytes |
| Schema nesting depth | 32 levels |
| Schema AST nodes per document | 4,096 |
| Schema AST nodes per Plugin, aggregate | 32,768 |
| Properties in one object schema | 128 |
| Entries in one `required` array | 128 |
| Values in one `enum` array | 256 |
| Branches in one applicator array | 32 |
| `$defs` entries per document | 64 |
| `$ref` resolutions per schema evaluation | 4,096 |
| Validation evaluation steps per input or output | 100,000 |

All byte counts are inclusive upper bounds. Aggregate counts MUST use overflow-safe arithmetic. The validator MUST enforce an evaluation budget and MUST terminate validation when the budget is exhausted. A timeout alone is not a substitute for a deterministic evaluation budget.

Schema validation MUST be deterministic: identical schema bytes, JSON instance bytes, and validator version MUST produce the same valid/invalid result. A validator that cannot enforce the required vocabulary and limits MUST report the contract as unsupported and MUST fail closed.

### 6.5 Invocation instance limits

The first request-response contract version also bounds JSON instances:

| Limit | Maximum |
|---|---:|
| Input JSON bytes | 1,048,576 |
| Output JSON bytes | 1,048,576 |
| JSON nesting depth | 64 levels |
| Object properties per object | 1,024 |
| Array elements per array | 4,096 |
| UTF-8 bytes per string | 262,144 |
| Total JSON values/nodes per instance | 100,000 |

Instances MUST be valid UTF-8 JSON and MUST reject duplicate object keys. Numbers MUST be parsed without lossy conversion before applying numeric constraints. Invalid or over-limit input MUST be rejected before invocation. Invalid or over-limit output MUST be treated as a failed invocation result; it MUST NOT be returned as a successful contract-conforming response.

These are contract-validation limits, not replacements for the Runtime Host's independent IPC frame, memory, CPU, deadline, and process limits.

## 7. Declarative dependencies

A capability MAY declare dependencies on other capabilities. A dependency object MUST contain:

- `capability_id`: a canonical identifier from §3.1;
- `version_requirement`: a requirement from §4;
- `kind`: exactly `required` or `optional`.

A dependency MUST NOT identify a concrete Plugin ID, Registry URL, process, endpoint, or executable. The Registry resolves namespace ownership; Core later resolves eligible providers.

Limits:

- at most 64 dependency entries per capability;
- at most 1,024 dependency entries across one Plugin release;
- no duplicate tuple of `capability_id` and `kind` within one capability declaration;
- each dependency identifier and version requirement obeys the limits in §§3–4.

Dependencies are declarative requirements only. They MUST NOT cause discovery to invoke another Plugin, start another runtime, create a permission grant, or inherit the dependency's permissions.

Until a separate Plugin Composition Contract is accepted and implemented, consumers MAY preserve and display dependency declarations but MUST NOT perform automatic cross-Plugin invocation, dependency graph execution, or privilege propagation. Cycle handling, conflict resolution, provider selection, execution ordering, partial failure, and rollback are reserved for that composition contract.

## 8. Trust, compatibility, lifecycle, and authorization pipeline

A capability declaration is eligible for invocation only after all applicable stages succeed. The stages are logically distinct and MUST NOT be collapsed into one capability-match boolean.

1. **Descriptor identity:** descriptor Plugin ID, version, namespace, release ID, and artifact identity are internally consistent.
2. **Cryptographic trust:** the exact descriptor signature is valid under accepted trust configuration and the artifact digest matches the exact package bytes.
3. **Declaration validity:** the capability declaration satisfies the versioned platform metadata schema and this specification's bounds.
4. **Identifier match:** the requested capability identifier exactly matches the declaration.
5. **Contract version:** the declaration's contract version satisfies the consumer's explicit requirement.
6. **Schema support and validation:** the consumer supports the schema vocabulary and both input/output schema documents are resolved from the verified release and validated within limits.
7. **Platform/runtime compatibility:** the PRIDE version, runtime profile, host, and platform policies permit the declaration.
8. **Lifecycle/runtime availability:** the selected immutable Plugin version is installed and in the required lifecycle/runtime state. A discovery result alone is not proof that a runtime is active.
9. **Core authorization:** Core evaluates the concrete principal, task, Plugin identity/version, capability, and invocation context. Denial or inability to obtain an authoritative decision MUST fail closed.
10. **Host enforcement:** the Runtime Host enforces the authorization decision, authenticated Plugin identity, scoped resources, IPC limits, and runtime isolation.

Passing a stage MUST NOT imply that a later stage will pass. In particular:

- a matching identifier does not prove version compatibility;
- a compatible version does not prove schema compatibility;
- valid metadata does not prove runtime availability;
- a trusted Plugin does not automatically have permission to execute;
- permission IDs and capability declarations are not grants;
- a dependency does not inherit another Plugin's authority.

Core remains the sole authorization authority. The Plugin Module owns bounded metadata validation and discovery, not platform authorization or provider selection. The Runtime Host enforces Core decisions and isolation.

## 9. Unknown metadata and forward compatibility

- Unknown capability identifiers MAY be stored, displayed, and discovered as inert metadata.
- Unsupported contract versions MAY be stored and displayed as inert metadata.
- Consumers MUST NOT invoke unknown identifiers or unsupported contracts.
- Consumers MUST NOT infer semantics from identifiers, names, descriptions, or dependency declarations.
- Unknown schema keywords MUST cause the schema to be unsupported for execution.
- Unknown properties in the version 2.0.0 capability declaration and platform metadata objects MUST be rejected unless a future schema explicitly defines an extension point.
- Unknown properties in developer-owned Manifest data remain governed by the separate Manifest contract and MUST NOT affect capability trust, matching, permissions, or authorization.
- Implementations MUST NOT claim to preserve unknown fields if their data model or persistence layer discards them.

A future extension MUST define its ownership, validation, compatibility, and resource bounds before it is accepted. Unknown data MUST remain inert.

## 10. Namespace and Marketplace governance

The Registry/Marketplace is responsible for namespace claims, namespace ownership evidence, declaration review, attestation, and signing.

Before signing a declaration, the Marketplace MUST verify:

1. the namespace is owned by the publishing principal or explicitly delegated to it;
2. the capability identifier is canonical and not reserved;
3. the contract version is valid SemVer;
4. both schema references exist in the release package and satisfy this specification;
5. the schemas pass the supported-vocabulary and resource-limit checks;
6. the declaration is bound to the immutable Plugin release;
7. dependency declarations are syntactically valid and bounded.

The Plugin Module MUST treat descriptor metadata as untrusted until the existing Ed25519 signature and SHA-256 artifact digest verification succeeds. It MUST preserve the exact signed descriptor bytes used for installation and recovery. It MUST NOT accept a package's self-declared capability metadata as authoritative.

The Registry's namespace attestation is an identity/governance assertion. It does not replace Core authorization or Runtime Host enforcement.

## 11. Ownership boundaries

| Responsibility | Owner |
|---|---|
| Namespace claims, review, attestation, signed release declarations | Marketplace / Registry |
| Descriptor and package verification; bounded parsing; schema-reference containment; metadata preservation; discovery | Plugin Module |
| Capability matching; version/schema compatibility; platform policy; provider selection; dependency planning; deterministic Execution Plan | Core Capability Registry |
| Per-invocation authorization decision | Core Authorization |
| Authenticated invocation, process isolation, scoped resource enforcement, IPC and host limits | Runtime Host |
| Developer-facing declaration and typed client interfaces | Plugin SDK |

No component MAY infer executable semantics from arbitrary Plugin metadata. No Plugin may bypass the Core workflow or Plugin Module discovery boundary.

## 12. Security requirements

Implementations MUST:

- reject malformed, duplicate-key, ambiguous, or over-limit metadata;
- use strict UTF-8 and canonical identifier validation;
- use the exact verified artifact for schema resolution;
- prohibit network schema resolution;
- bound parsing, schema evaluation, and invocation instances;
- reject unsupported schema semantics instead of silently accepting them;
- keep capability metadata separate from permission metadata;
- require Core authorization for each concrete invocation;
- enforce authorization and isolation in the Runtime Host;
- avoid logging raw input/output payloads, secrets, or unbounded attacker-controlled metadata by default;
- return bounded, typed failure categories without leaking credentials, filesystem paths, or internal policy details.

The capability contract does not itself provide sandboxing. Process isolation and host resource enforcement remain governed by PLR-ADR-0003 and the Runtime Host deployment contract.

## 13. Conformance requirements

An implementation claiming conformance to POCS-0001 MUST pass tests for:

- valid and invalid identifier grammar, reserved namespace handling, and exact case sensitivity;
- strict SemVer parsing and every supported version-requirement operator;
- prerelease exclusion and build-metadata precedence behavior;
- descriptor/platform metadata version separation and legacy 1.1.0 treatment;
- duplicate JSON keys and duplicate capability/dependency declarations;
- missing, malformed, duplicate, traversal, external, and cross-document schema references;
- every supported JSON Schema keyword and every explicitly unsupported keyword;
- schema and instance size/depth/node/evaluation limits, including integer-overflow boundaries;
- deterministic validation outcomes;
- unsupported contract versions and unknown capabilities remaining inert;
- compatibility stages remaining separate;
- permission metadata never granting access;
- Core authorization denial and authorization-service failure both failing closed;
- no automatic cross-Plugin invocation before the Composition Contract is implemented.

Conformance fixtures MUST be versioned with the schemas and MUST include both positive and negative cases. A validator MUST NOT be considered production-ready solely because it passes a small set of happy-path examples.

## 14. Explicit non-goals

POCS-0001 does not:

- define business semantics for any specific capability;
- define the full Plugin Composition Contract;
- define dependency graph resolution, cycle handling, provider conflicts, execution order, or partial failure;
- define UI Extension Points or primary-interface replacement;
- define Agentic planning, multi-agent coordination, or AI-provider routing protocols;
- define streaming, events, background jobs, callbacks, or long-running invocation modes;
- implement Core capability matching, provider selection, or Execution Plan generation;
- implement the JSON Schema validator or schema loader;
- change SDK RPCs, Protobuf messages, runtime IPC, SQLite schema, or lifecycle transitions;
- enable production capability execution by itself.

## 15. Required implementation sequence

After acceptance of this specification:

1. Publish versioned descriptor/platform metadata/capability declaration JSON Schemas and positive/negative conformance fixtures.
2. Implement strict schema-reference loading and bounded JSON Schema validation against the verified package.
3. Add provider-neutral domain types and parsing/validation for versioned capability declarations.
4. Add persistence and discovery compatibility without changing the meaning of signed schema 1.1.0 records.
5. Add Core Capability Registry compatibility evaluation and deterministic candidate selection.
6. Define and implement the Plugin Composition Contract before automatic dependency invocation.
7. Define and implement the Extension Point / Interface Replacement Contract before allowing a Passive Plugin to replace the primary interface.
8. Audit RuntimePort and SDK boundaries against the accepted contracts.
9. Enable production invocation only after conformance, security, and end-to-end tests pass.

Acceptance of this document establishes the normative specification. It does not claim that schemas, validators, Core resolution, composition, UI replacement, or production execution are implemented.
