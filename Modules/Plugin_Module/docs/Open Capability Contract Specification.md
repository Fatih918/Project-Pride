# PRIDE Open Capability Contract Specification

**Document ID:** POCS-0001  
**Version:** 1.0.0  
**Status:** Proposed for review  
**Date:** 2026-11-27  
**Module:** PRIDE Plugin Module / Core Capability Registry  
**Related decisions:** ADR-004 — Capability Registry; ADR-013 — Unlimited Plugin Extensibility and Composition; PLR-ADR-0004 — Open Capability Contract Architecture Decision Record; PLR-ADR-0003 — Plugin Runtime Execution Architecture

---

## 1. Purpose and normative language

This specification defines the first machine-readable contract for declaring, validating, matching, and safely treating Plugin capabilities. It converts PLR-ADR-0004 into implementable rules for identifiers, namespace ownership, SemVer, signed declaration metadata, JSON Schema input/output contracts, resource bounds, compatibility, and declarative dependencies.

It does not define the business semantics of individual capabilities, the complete Plugin Composition protocol, UI Extension Point / Interface Replacement contract, Agentic planning protocol, or runtime invocation wire protocol.

The terms **MUST**, **MUST NOT**, **REQUIRED**, **SHOULD**, **SHOULD NOT**, and **MAY** are normative.

## 2. Versioning and compatibility baseline

### 2.1 Independent version domains

PRIDE MUST treat the following as independent: Plugin version; developer Manifest schema version; Registry artifact descriptor schema version; Marketplace platform metadata schema version; capability contract version; and this specification version.

A Plugin version MUST NOT substitute for a capability contract version. A descriptor schema upgrade MUST NOT imply that a capability contract changed, and a capability contract upgrade MUST NOT imply that the Plugin Manifest schema changed.

### 2.2 First versioned descriptor increment

The first descriptor format carrying complete capability contracts MUST use:

- artifact descriptor `schema_version: 2.0.0` and `$schema: urn:pride:registry-artifact-descriptor:2`;
- platform metadata `schema_version: 2.0.0` and `$schema: urn:pride:platform-metadata:2`;
- capability declaration schema identifier `urn:pride:capability-declaration:1`;
- JSON Schema Draft 2020-12 for input and output schemas.

Existing artifact descriptor schema 1.1.0 and platform metadata schema 1.0.0 MUST retain their current meaning. They MUST NOT be reinterpreted as schema 2.0.0 by adding inferred defaults or rewriting signed descriptor bytes. Versioned JSON Schema files, examples, and conformance fixtures are a separate follow-up increment after this specification is accepted.

### 2.3 Legacy schema 1.1.0 behavior

A descriptor using schema 1.1.0 MAY continue to be installed under its existing validation and cryptographic rules. Its existing capability entries containing only `id`, `name`, and `description` MUST be classified as **legacy capability metadata**.

Legacy capability metadata:
1. MAY be displayed and returned by discovery.
2. MUST NOT be assigned an inferred contract version.
3. MUST NOT satisfy a versioned capability requirement.
4. MUST NOT be invoked through the versioned capability contract mechanism.
5. MUST NOT be upgraded in place or silently rewritten to schema 2.0.0.
6. MUST remain bound to the exact signed descriptor bytes for recovery verification.

Existing committed installations MUST NOT be removed merely because they contain legacy capability metadata. A PROMOTED operation MUST be recovered according to its persisted descriptor version and the verification rules recorded for that operation. Schema 1.1.0 is not the unsupported schema 1.0.0 legacy case already rejected by the installation implementation.

New schema 2.0.0 installations MUST satisfy this specification. A Plugin with legacy declarations MUST NOT be treated as fully compatible with a versioned capability contract.

## 3. Capability declaration model

A schema 2.0.0 platform metadata document contains at most 256 capability declaration objects. Each declaration MUST contain:

| Property | Required | Meaning |
|---|---:|---|
| `id` | Yes | Immutable namespaced capability identifier |
| `name` | Yes | Human-readable display name |
| `description` | Yes | Human-readable description |
| `contract_version` | Yes | Strict SemVer 2.0.0 version |
| `invocation_mode` | Yes | Platform-defined invocation semantic |
| `input_schema_ref` | Yes | Immutable package-relative input schema reference |
| `output_schema_ref` | Yes | Immutable package-relative output schema reference |
| `platform_compatibility` | No | Typed PRIDE/runtime constraints |
| `dependencies` | No | Declarative capability requirements |

No additional capability declaration properties are defined in version 1. Implementations MUST reject unknown properties in this object. Future extensions MUST use an explicitly versioned schema change. Implementations MUST NOT infer semantics from unknown properties.

The authoritative declaration is Marketplace-owned `platform_metadata` inside the signed artifact descriptor. Developer-owned Manifest data MUST NOT override, replace, or elevate the signed declaration.

### 3.1 Canonical capability identifiers

The canonical identifier form is:

```text
<namespace>/<capability-name>
```

The namespace and capability name each consist of one or more dot-separated segments. The namespace MUST contain at least two segments. Each segment MUST contain only lowercase ASCII letters (`a-z`), digits (`0-9`), and internal hyphens; it MUST begin and end with a lowercase ASCII letter or digit; and it MUST NOT be empty.

Valid examples:
- `io.pride/text.generate`
- `com.example/document.summarize`
- `org.acme/workflow.plan`
- `dev-team.tools/file-index`

Invalid examples:
- `Text.Generate` — uppercase characters;
- `example/summarize` — namespace has fewer than two segments;
- `com..example/summarize` — empty segment;
- `com.example/-summarize` — segment begins with a hyphen;
- `com.example/summarize-` — segment ends with a hyphen;
- `com.example/` — empty capability name;
- `com.example/summarize name` — whitespace.

The complete identifier MUST be at most 255 ASCII bytes. Identifiers are case-sensitive. Producers and consumers MUST NOT lowercase, trim, Unicode-normalize, or otherwise rewrite identifiers before comparison.

The `io.pride` namespace and all descendants are reserved to PRIDE. Third parties MUST NOT claim or publish identifiers in that namespace. Syntactic validity does not prove namespace ownership: the trusted Registry/Marketplace claim process MUST establish ownership before attestation and signing.

Capability identifiers are immutable. A semantic rename MUST allocate a new identifier. Aliases, if introduced later, MUST be explicit signed Registry metadata and MUST NOT rely on implicit normalization.

### 3.2 Human-readable metadata

- `name` MUST be valid UTF-8, non-empty, and at most 128 UTF-8 bytes.
- `description` MUST be valid UTF-8 and at most 1,024 UTF-8 bytes. Empty descriptions are permitted.
- Neither may contain control characters other than tab, line feed, and carriage return.
- Names and descriptions are descriptive only. They MUST NOT be parsed as instructions, used as authorization policy, or used to infer missing semantics.

### 3.3 Contract versions

Every declaration MUST carry a strict SemVer 2.0.0 version:
- It MUST contain `MAJOR.MINOR.PATCH`.
- Numeric identifiers MUST NOT have leading zeroes except the value zero.
- Prerelease and build metadata MUST follow SemVer 2.0.0.
- The complete version MUST be at most 64 ASCII bytes.
- Whitespace, a leading `v`, partial versions, wildcards, and aliases MUST be rejected.
- Precedence MUST follow SemVer 2.0.0. Build metadata does not affect precedence or compatibility matching.

A major version denotes a breaking change to observable input/output semantics or invocation guarantees. A minor version adds backward-compatible optional behavior or optional schema fields. A patch version clarifies implementation-neutral details without changing the accepted input/output contract.

A publisher MUST NOT publish different semantic contracts under the same capability ID and contract version. Marketplace attestation MUST bind the declaration to the immutable Plugin release.

## 4. Deterministic version requirements

Consumers MUST declare an explicit version requirement when requesting or depending on a capability. The grammar is intentionally narrower than package-manager range dialects.

### 4.1 Grammar

A requirement is either one exact SemVer version or a conjunction of one to four comparator clauses separated by ASCII spaces. Each comparator consists of one operator immediately followed by a complete SemVer version:

- `=` exact precedence match;
- `>` greater than;
- `>=` greater than or equal;
- `<` less than;
- `<=` less than or equal.

Without an operator, a complete version means exact match.

Valid examples: `1.2.3`, `=1.2.3`, `>=1.2.0 <2.0.0`, `>1.4.0 <=1.8.2`.

The following MUST be rejected: caret/tilde ranges (`^1.2.3`, `~1.2.3`), wildcards, OR clauses, comma-separated clauses, partial versions, empty clauses, duplicate contradictory comparators, and more than four comparators. The complete requirement MUST be at most 128 ASCII bytes.

### 4.2 Matching semantics

All clauses MUST be true. Comparisons MUST use SemVer precedence, not lexical ordering. Build metadata is ignored for precedence. Exact matching means SemVer precedence equality; byte-identical release identity is a separate Registry constraint.

A prerelease candidate MUST NOT satisfy a requirement composed only of stable-version comparators unless at least one comparator explicitly contains a prerelease version with the same major, minor, and patch tuple. This follows SemVer prerelease exclusion behavior.

A matching version requirement is necessary but not sufficient. Consumers MUST separately validate schema documents, invocation mode, platform/runtime constraints, trust, lifecycle availability, and authorization. Consumers MUST NOT infer compatibility from equal major versions alone.

## 5. Invocation mode and platform compatibility

### 5.1 Invocation mode

Version 1 defines one invocation mode:

```text
io.pride.invocation/request-response
```

It represents one bounded request with one corresponding response over the authenticated host-managed Plugin invocation boundary. A declaration MUST state this value explicitly. Any other value is unsupported in version 1 and MUST make the declaration ineligible for invocation. Unknown modes MUST NOT be guessed or mapped to this mode.

Streaming, events, background jobs, callbacks, UI surfaces, and long-running workflow semantics are outside this initial mode and require separately versioned contracts.

### 5.2 Platform compatibility

The optional `platform_compatibility` object MAY contain:
- `pride_version`: a version requirement using §4 grammar;
- `runtime_profile_ids`: an array of platform-defined runtime profile identifiers.

At least one property MUST be present when the object exists. Unknown properties MUST be rejected.

Each runtime profile ID MUST use the canonical identifier grammar in §3.1, MUST be in the reserved `io.pride.runtime/` namespace, and MUST be at most 128 ASCII bytes. The platform-owned profile registry is authoritative for its meaning. Unknown or unavailable required profiles make the declaration unsupported for execution; they MUST NOT mean unrestricted compatibility.

If `platform_compatibility` is absent, the declaration adds no publisher-specified compatibility constraint. This does not bypass consumer PRIDE version, runtime, operating environment, security policy, or host support checks. Compatibility metadata MUST NOT grant access to runtime features, permissions, credentials, or host resources.

## 6. Input and output JSON Schema

### 6.1 Document and reference requirements

Each capability MUST reference one input schema and one output schema. Both MUST use JSON Schema Draft 2020-12 and be included in the same verified Plugin artifact.

Each reference MUST be a package-relative path under `package/schemas/`, expressed relative to the package root and beginning with `schemas/`. It MUST use forward slashes and MUST NOT contain an absolute path, drive prefix, URI scheme, query, fragment, empty/`.`/`..` segment, backslash, percent-encoded separator, NUL, or control character. The normalized path MUST be at most 240 UTF-8 bytes, consistent with PZIP-0001. The referenced file MUST exist exactly once in the verified archive. PZIP-0001 duplicate-path and case-folding-collision rules also apply.

A schema document MUST:
- be valid UTF-8 JSON without a byte-order mark;
- reject duplicate JSON object keys at every nesting level;
- declare `"$schema": "https://json-schema.org/draft/2020-12/schema"`;
- have an object root and an explicit `type`;
- be at most 65,536 bytes;
- be immutable as part of the verified artifact digest.

Schema bytes MUST be read only from the verified immutable release. Network schema resolution is prohibited during installation, discovery, matching, and invocation.

### 6.2 Supported vocabulary

The initial supported vocabulary is:

| Group | Keywords |
|---|---|
| Core | `$schema`, `$ref`, `$defs` |
| Type/value | `type`, `enum`, `const` |
| Object | `properties`, `required`, `additionalProperties`, `minProperties`, `maxProperties` |
| Array | `items`, `minItems`, `maxItems`, `uniqueItems` |
| String | `minLength`, `maxLength` |
| Numeric | `minimum`, `maximum`, `exclusiveMinimum`, `exclusiveMaximum`, `multipleOf` |
| Applicators | `allOf`, `anyOf`, `oneOf`, `not` |

Supported `type` values are `object`, `array`, `string`, `number`, `integer`, `boolean`, and `null`. A type array MAY be used if it contains only these values and no duplicates. A boolean schema MUST NOT be used as a root schema or as the value of any schema-bearing keyword, except that `additionalProperties: false` is permitted and REQUIRED when an object must reject undeclared properties. `additionalProperties: true` is prohibited.

A schema MAY use `$defs` and local JSON Pointer references of the form `#` or `#/$defs/<token>`. References MUST resolve within the same schema document. JSON Pointer tokens MUST follow RFC 6901 escaping rules. External, relative-file, cross-document, anchor, and dynamic references are unsupported. Reference cycles MUST be rejected.

The following keywords are unsupported and MUST cause the schema to be rejected for execution: `$id`, `$anchor`, `$dynamicAnchor`, `$dynamicRef`, `$recursiveRef`, `$recursiveAnchor`, `$vocabulary`, `$comment`, `title`, `description`, `default`, `examples`, `deprecated`, `readOnly`, `writeOnly`, `format`, `contentEncoding`, `contentMediaType`, `contentSchema`, `pattern`, `patternProperties`, `propertyNames`, `unevaluatedItems`, `unevaluatedProperties`, `contains`, `minContains`, `maxContains`, `if`, `then`, `else`, `dependentRequired`, and `dependentSchemas`.

Any unrecognized keyword MUST be rejected. Implementations MUST NOT silently ignore keywords or claim full Draft 2020-12 support. They MUST enforce exactly the vocabulary and semantics defined here.

### 6.3 Schema resource limits

| Limit | Maximum |
|---|---:|
| Capability declarations per Plugin | 256 |
| Schema reference occurrences per Plugin | 512 |
| Distinct schema documents per Plugin | 128 |
| One schema document | 65,536 bytes |
| Aggregate schema bytes per Plugin | 1,048,576 bytes |
| Schema nesting depth (root is level 1) | 32 |
| Schema AST nodes per document | 4,096 |
| Schema AST nodes per Plugin | 32,768 |
| Properties in one object schema | 128 |
| Entries in one `required` array | 128 |
| Values in one `enum` array | 256 |
| Branches in one applicator array | 32 |
| `$defs` entries per document | 64 |
| `$ref` resolutions per validation | 4,096 |
| Validation evaluation steps per input/output | 100,000 |

An AST node is each JSON value in the parsed schema document. Evaluation steps MUST be counted as each supported validation keyword applied to an instance node; each `$ref` resolution also consumes one step. Aggregate counters MUST use overflow-safe arithmetic. Validation MUST terminate when its evaluation budget is exhausted. A timeout alone is not a substitute for the deterministic budget.

String `minLength` and `maxLength` use Unicode code-point counts as defined by JSON Schema. Numeric comparisons and `multipleOf` MUST use exact decimal/rational semantics, not lossy binary floating-point conversion.

Validation MUST be deterministic: identical schema bytes, instance bytes, and validator version MUST produce the same valid/invalid result. A validator unable to enforce this vocabulary or its limits MUST report the contract as unsupported and fail closed.

### 6.4 Invocation instance limits

| Limit | Maximum |
|---|---:|
| Input JSON bytes | 1,048,576 |
| Output JSON bytes | 1,048,576 |
| JSON nesting depth (root is level 1) | 64 |
| Object properties per object | 1,024 |
| Array elements per array | 4,096 |
| UTF-8 bytes per string | 262,144 |
| Total JSON values/nodes per instance | 100,000 |

Instances MUST be valid UTF-8 JSON and reject duplicate object keys. Numbers MUST be parsed without lossy conversion before numeric validation. Invalid or over-limit input MUST be rejected before invocation. Invalid or over-limit output MUST be treated as a failed invocation, never a successful contract-conforming response.

These limits do not replace Runtime Host IPC frame, memory, CPU, deadline, or process limits.

## 7. Declarative dependencies

A capability MAY declare dependencies. Each dependency MUST contain:
- `capability_id`: a canonical identifier from §3.1;
- `version_requirement`: a requirement from §4;
- `kind`: exactly `required` or `optional`.

A dependency MUST NOT identify a concrete Plugin ID, Registry URL, process, endpoint, or executable. Namespace ownership is established by the Registry; Core later resolves eligible providers.

There MAY be at most 64 dependencies per capability and 1,024 across one Plugin release. Duplicate tuples of `capability_id` and `kind` within one declaration MUST be rejected. Identifier and version requirement limits in §§3–4 apply.

Dependencies are declarative only. They MUST NOT cause discovery to invoke another Plugin, start another runtime, create a permission grant, or inherit permissions. Until a separate Plugin Composition Contract is accepted and implemented, consumers MAY preserve/display dependencies but MUST NOT automatically invoke other Plugins or execute a dependency graph. Cycle handling, conflict resolution, provider selection, execution ordering, partial failure, and rollback belong to that separate contract.

## 8. Trust, compatibility, lifecycle, and authorization

Capability eligibility MUST be evaluated through distinct stages; they MUST NOT be collapsed into one capability-match boolean:

1. Descriptor identity: Plugin ID, version, namespace, release ID, and artifact identity are consistent.
2. Cryptographic trust: exact descriptor signature is valid under accepted trust configuration and artifact digest matches exact package bytes.
3. Declaration validity: platform metadata satisfies its versioned schema and bounds.
4. Identifier match: requested ID exactly matches.
5. Contract version: declared version satisfies the consumer's explicit requirement.
6. Schema support: the consumer supports the vocabulary and validates both schemas from the verified release within limits.
7. Platform/runtime compatibility: PRIDE version, runtime profile, host, and platform policy permit it.
8. Lifecycle/runtime availability: selected immutable Plugin version is installed and in the required lifecycle/runtime state. Discovery alone does not prove runtime activity.
9. Core authorization: Core evaluates the concrete principal, task, Plugin identity/version, capability, and invocation context. Denial or inability to obtain an authoritative decision MUST fail closed.
10. Host enforcement: Runtime Host enforces the decision, authenticated Plugin identity, scoped resources, IPC limits, and isolation.

Passing one stage MUST NOT imply passing a later stage. A matching ID does not prove version compatibility; version compatibility does not prove schema compatibility; trusted metadata does not prove runtime availability; trust does not grant permission; dependencies do not inherit authority.

Core remains the sole authorization authority. Plugin Module owns bounded metadata validation and discovery, not platform authorization or provider selection. Runtime Host enforces Core decisions and isolation.

## 9. Unknown and legacy metadata

Unknown capability IDs and unsupported contract versions MAY be stored, displayed, and discovered as inert metadata. Consumers MUST NOT invoke them or infer semantics from IDs, names, descriptions, or dependencies. Unknown schema keywords make a schema unsupported for execution.

Unknown properties in schema 2.0.0 capability declarations and platform metadata MUST be rejected unless a future schema explicitly defines an extension point. Unknown developer Manifest properties remain governed by the separate Manifest contract and MUST NOT affect capability trust, matching, permissions, or authorization. Implementations MUST NOT claim to preserve unknown fields if they discard them.

Future extensions MUST define ownership, validation, compatibility, and resource bounds before acceptance. Unknown data remains inert.

## 10. Namespace and Marketplace governance

Registry/Marketplace owns namespace claims, ownership evidence, declaration review, attestation, and signing. Before signing, it MUST verify:
1. namespace ownership by the publishing principal or explicit delegation;
2. canonical, non-reserved capability ID;
3. valid SemVer contract version;
4. existence and validity of both schema references in the release package;
5. schema vocabulary and resource-limit conformance;
6. binding to the immutable Plugin release;
7. syntactic validity and bounds of dependencies.

Plugin Module MUST treat descriptor metadata as untrusted until Ed25519 signature and SHA-256 artifact digest verification succeeds. It MUST preserve exact signed descriptor bytes for installation and recovery. Package self-declarations are not authoritative. Namespace attestation does not replace Core authorization or Runtime Host enforcement.

## 11. Ownership boundaries

| Responsibility | Owner |
|---|---|
| Namespace claims, review, attestation, signed declarations | Marketplace / Registry |
| Descriptor/package verification, bounded parsing, schema containment, metadata preservation, discovery | Plugin Module |
| Capability matching, version/schema compatibility, platform policy, provider selection, dependency planning, deterministic Execution Plan | Core Capability Registry |
| Per-invocation authorization decision | Core Authorization |
| Authenticated invocation, isolation, scoped resources, IPC and host limits | Runtime Host |
| Developer-facing declarations and typed client interfaces | Plugin SDK |

No component MAY infer executable semantics from arbitrary Plugin metadata. Plugins MUST NOT bypass Core workflow or the Plugin Module discovery boundary.

## 12. Security requirements

Implementations MUST reject malformed, duplicate-key, ambiguous, or over-limit metadata; use strict UTF-8 and canonical identifier validation; resolve schemas only from the verified artifact; prohibit network schema resolution; bound parsing, evaluation, and invocation instances; reject unsupported semantics; keep capability and permission metadata separate; require Core authorization per invocation; and enforce decisions/isolation in the Runtime Host.

Implementations SHOULD avoid logging raw payloads, secrets, or unbounded attacker-controlled metadata. Errors MUST be bounded and typed and MUST NOT leak credentials, absolute paths, or internal policy details.

This contract does not itself provide sandboxing. Process isolation and host resource enforcement remain governed by PLR-ADR-0003 and the Runtime Host deployment contract.

## 13. Conformance requirements

An implementation claiming POCS-0001 conformance MUST test:
- valid/invalid ID grammar, reserved namespaces, and case sensitivity;
- strict SemVer and all requirement operators, including prerelease/build behavior;
- descriptor/platform version separation and legacy 1.1.0 behavior;
- duplicate JSON keys and duplicate capability/dependency declarations;
- missing, malformed, duplicate, traversal, external, and cross-document schema references;
- every supported and explicitly unsupported schema keyword;
- schema/instance size, depth, node, evaluation, and integer-overflow boundaries;
- deterministic validation;
- unknown capabilities/contracts remaining inert;
- separation of compatibility stages;
- permission metadata never granting access;
- authorization denial and authorization-service failure both failing closed;
- no automatic cross-Plugin invocation before the Composition Contract is implemented.

Versioned schemas MUST ship with positive and negative conformance fixtures. Happy-path tests alone are insufficient for production readiness.

## 14. Explicit non-goals

This specification does not define individual capability business semantics; full Plugin Composition; dependency graph execution; UI Extension Points or primary-interface replacement; Agentic planning, multi-agent coordination, or AI-provider routing; streaming/events/background jobs/callbacks; Core capability matching/provider selection/Execution Plan implementation; JSON Schema validator implementation; SDK RPCs, Protobuf, runtime IPC, SQLite migrations, or lifecycle transitions. It does not enable production capability execution by itself.

## 15. Required implementation sequence

After acceptance:
1. Publish versioned descriptor/platform metadata/capability declaration JSON Schemas and positive/negative fixtures.
2. Implement strict schema-reference loading and bounded JSON Schema validation against verified packages.
3. Add provider-neutral domain types and parsing/validation.
4. Add persistence and discovery compatibility without changing signed schema 1.1.0 meaning.
5. Add Core compatibility evaluation and deterministic candidate selection.
6. Define and implement Plugin Composition before automatic dependency invocation.
7. Define and implement Extension Point / Interface Replacement before Passive Plugin primary-interface replacement.
8. Audit RuntimePort and SDK boundaries against accepted contracts.
9. Enable production invocation only after conformance, security, and end-to-end tests pass.

Acceptance establishes this normative specification; it does not claim that schemas, validators, Core resolution, composition, UI replacement, or production execution are implemented.
