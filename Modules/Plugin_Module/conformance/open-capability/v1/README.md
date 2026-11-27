# Open Capability Contract v1 Conformance Fixtures

These fixtures accompany POCS-0001 and the versioned schema documents.

## Layout

- `positive/descriptor-v2.json`: structurally valid Descriptor 2.0.0 and Platform Metadata 2.0.0 example.
- `positive/request.schema.json` and `positive/response.schema.json`: package-local input/output schemas using only the initial supported vocabulary.
- `negative/*.json`: one intentionally invalid case per file.

## Important test boundary

The positive descriptor uses deterministic placeholder digest/signature values. It is a **structural schema fixture only**. It is not a cryptographically valid Marketplace release and MUST NOT be used for installation or signature-verification tests.

The JSON Schema documents validate structural shape and bounded string/array sizes. They do not replace semantic validators for:
- canonical capability identifier grammar and reserved namespace ownership;
- strict SemVer parsing and version-requirement matching;
- path normalization and verified-package containment;
- duplicate capability IDs and duplicate JSON object keys;
- supported JSON Schema vocabulary, cycle detection, and resource budgets;
- signature and artifact digest verification;
- Registry namespace ownership and Marketplace attestation.

A conformance runner MUST classify each negative fixture by the semantic or schema rule it violates. It MUST NOT accept a fixture merely because the structural JSON Schema cannot express a rule.

## Expected negative cases

| Fixture | Expected rejection |
|---|---|
| `descriptor-wrong-schema-version.json` | Descriptor schema version mismatch |
| `capability-id-uppercase.json` | Non-canonical capability identifier |
| `capability-schema-reference-traversal.json` | Package-relative schema path traversal |
| `capability-unsupported-invocation-mode.json` | Unsupported invocation mode |
| `schema-unsupported-pattern-keyword.json` | Unsupported JSON Schema keyword |
| `duplicate-capability-id.json` | Duplicate capability identifier within one declaration set |
| `invalid-contract-version.json` | Non-SemVer capability contract version |
| `duplicate-json-keys.json` | Duplicate JSON object key; reject before schema validation |
| `capability-unknown-property.json` | Unknown property in strict capability declaration |

## Determinism

Fixture bytes are normative test inputs. A conformance runner MUST NOT rewrite, normalize, or reserialize a fixture before tests that concern duplicate keys, exact bytes, signatures, or digests.
