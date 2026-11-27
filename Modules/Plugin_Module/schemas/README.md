# Versioned Plugin Schemas

This directory contains the structural JSON Schema documents for the first open-capability contract increment.

| File | Contract |
|---|---|
| `registry-artifact-descriptor-v2.schema.json` | Registry artifact descriptor 2.0.0 |
| `platform-metadata-v2.schema.json` | Marketplace platform metadata 2.0.0 |
| `capability-declaration.schema.json` | Standalone structural schema for one capability declaration |

All schemas target JSON Schema Draft 2020-12 and intentionally use a bounded vocabulary. Semantic constraints that JSON Schema cannot express within POCS-0001—such as strict SemVer parsing, canonical identifier grammar, duplicate IDs, path containment, and cryptographic verification—remain mandatory in the owning validators.

The platform metadata and descriptor schemas contain equivalent inline capability declaration definitions so each complete document can be validated without external or network schema resolution. The standalone capability declaration schema is provided for SDK/tooling validation and conformance tests.

These files do not replace the existing schema 1.1.0 descriptor schema. Existing signed descriptor bytes and their validation path remain unchanged.
