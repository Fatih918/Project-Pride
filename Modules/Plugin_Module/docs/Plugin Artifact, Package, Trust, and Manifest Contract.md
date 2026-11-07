# Plugin Artifact, Package, Trust, and Manifest Contract

**Version:** 1.0.0  
**Status:** Accepted  
**Category:** Module Technical Specification  
**Document ID:** PTS-0004  
**Related Documents:** ADR-008, ADR-012, PTS-0001, PTS-0002, PTS-0003, PDR-0001, DCR-0001  
**Module:** PRIDE Plugin Module

## 1. Purpose and scope

This specification establishes the accepted machine-readable contracts for Marketplace artifact descriptors, ZIP packages, SHA-256 integrity, Ed25519 authenticity, trust-root bootstrap, and the developer-owned plugin.json Manifest. It does not implement a Marketplace, downloader, ZIP extractor, signing service, remote trust service, or production cryptographic verifier.

## 2. Ownership and trust boundaries

- Plugin developers own their repositories, package contents, and developer-owned Manifest.
- Marketplace analyzes releases, creates Registry artifact descriptors, computes artifact digests, and signs descriptors.
- Registry Repository stores Registry documents and descriptors only. It does not store Plugin ZIP bytes, source code, or release binaries.
- Artifact hosts serve immutable release bytes referenced by descriptors.
- Plugin Module validates descriptors, acquires bytes, verifies digest and signature, inspects ZIP structure, validates Manifest, and coordinates local installation.
- Trust configuration identifies Marketplace signing keys trusted by a PRIDE distribution's embedded bootstrap key.

A Registry document MUST NOT turn an arbitrary repository page or mutable branch into an installable artifact locator.

## 3. D1 — Versioned Registry artifact descriptor

Every installable release MUST publish an explicit provider-neutral descriptor containing: $schema, schema_version, plugin_id, version, release_id, download_uri, media_type, size_bytes, digest, and signature.

- schema_version is initially 1.0.0; $schema is urn:pride:registry-artifact-descriptor:1.
- plugin_id and version MUST match the Registry identity, package Manifest, and installation request.
- release_id MUST identify the exact immutable Marketplace/source release.
- download_uri MUST use HTTPS and MUST NOT contain URI user-info credentials.
- media_type MUST be application/vnd.pride.plugin+zip.
- size_bytes is the exact ZIP byte length.
- digest.algorithm MUST be sha-256; digest.value MUST be 64 lowercase hexadecimal characters.
- signature.algorithm MUST be Ed25519; signature.key_id identifies a key in accepted trust configuration; signature.value is base64 encoding of a 64-byte signature.

Provider-specific release identifiers may be opaque strings. Provider-specific API fields do not enter the common descriptor. A mutable branch, tag alias, repository homepage, or search page is not an immutable release locator. The Registry Repository stores the descriptor, not the ZIP bytes.

### 3.1 Signature payload

The signature is detached. Construct the descriptor JSON with every required field except signature, canonicalize it using RFC 8785 JSON Canonicalization Scheme (JCS), encode the canonical JSON as UTF-8, and sign those bytes with the Marketplace Ed25519 private key. The verifier MUST reconstruct the same payload. It MUST NOT sign arbitrary pretty-printed or source-order JSON bytes.

The signature binds Plugin identity, version, immutable release ID, download URI, media type, byte length, and digest. SHA-256 is computed over the exact downloaded ZIP bytes, not extracted files.

## 4. D2 — ZIP package format

The first supported package format is ZIP with media type application/vnd.pride.plugin+zip. The archive layout is:

```text
plugin-package.zip
├── plugin.json
└── package/
    ├── <entry-point and plugin files>
    └── ...
```

Exactly one plugin.json MUST exist at the archive root. Plugin payload files MUST be contained under package/. The Manifest entry_point is interpreted relative to package/.

Version 1 permits only regular files and directories. The extractor MUST reject absolute paths, drive-qualified paths, parent traversal, backslash ambiguity, duplicate normalized paths, case-folding collisions, symlink or special-file external attributes, encrypted entries, unsupported compression methods, and entries outside the root Manifest/package layout. ZIP external attributes indicating links, sockets, FIFOs, devices, or other non-regular entries MUST be rejected.

The extractor MUST enforce versioned configurable limits for compressed archive bytes, expanded bytes, entry count, individual file size, path length, nesting depth, compression ratio, and metadata allocation. Numeric limits MUST be published in a versioned installation policy before a production extractor is enabled. Missing limits are a fail-closed configuration error. Extraction MUST stream data and MUST verify actual bytes written rather than trusting ZIP header sizes.

Extraction occurs only inside the operation-scoped staging directory. Every output path MUST be resolved through the Plugin storage adapter and checked for containment. Promotion is forbidden until descriptor signature, byte length, SHA-256 digest, ZIP structure, extraction policy, Manifest validity, and identity/version consistency all pass.

## 5. D3 — Integrity, authenticity, and trust roots

SHA-256 detects byte changes relative to a signed expected digest; it does not establish publisher identity. Ed25519 verifies that a Marketplace signing key authenticated the descriptor. A digest without a trusted signature is insufficient, and a signature without a matching digest is insufficient. Verification MUST fail closed when required evidence, trust configuration, or metadata is absent, invalid, expired, revoked, or inconsistent.

### 5.1 Hybrid bootstrap trust

1. A PRIDE distribution embeds one or more public bootstrap trust keys. These are public keys, never private signing material.
2. The client obtains a remote trust configuration.
3. Every remote trust configuration MUST be authenticated directly by an embedded bootstrap key. Marketplace descriptor keys are rotated inside that authenticated configuration; they cannot replace the embedded bootstrap root.
4. Only keys listed as active and valid in an accepted trust configuration may verify Marketplace descriptors.
5. A remote configuration cannot make a new root trusted merely by declaring it trusted.

The bootstrap public key is provisioned through the PRIDE release/build process. It MUST NOT be downloaded for the first time from the same unauthenticated endpoint as the trust configuration.

### 5.2 Trust configuration

A trust configuration MUST contain $schema, schema_version, monotonically increasing sequence, issued_at, expires_at, unique key_id records, algorithm, public key, validity interval, status, and a detached signature envelope. Its schema is `Modules/Plugin_Module/schemas/trust-configuration.schema.json`. Version 1 accepts Ed25519 Marketplace descriptor keys only. Public keys are base64-encoded raw Ed25519 public keys (32 bytes). Private keys MUST never be distributed in trust configuration or stored by Plugin Module.

The client MUST persist the highest accepted sequence and reject lower values. Invalid signatures and expired configurations MUST be rejected. Revoked keys MUST NOT verify new installations. Rotation MUST be represented by a higher-sequence configuration authenticated by an already trusted bootstrap/authorized key. Replacing a compromised embedded bootstrap key requires a PRIDE application/distribution update or separately reviewed recovery mechanism; remote metadata cannot silently replace the embedded root.

If no currently valid trust configuration is available, new install/update operations MUST fail closed. Existing installed Plugins are not automatically removed when trust configuration expires. The client clock is part of expiry enforcement; sequence-based rollback protection remains mandatory.

The Marketplace signature on the artifact descriptor is the required installation attestation. A Registry-level signature, if present, protects a distinct payload and is not interchangeable with the descriptor signature.

## 6. D4 — Developer Manifest JSON

Every package MUST contain exactly one root-level plugin.json. The Manifest uses snake_case properties, schema_version initially 1.0.0, and $schema set to urn:pride:plugin-manifest:1. The bundled schema uses JSON Schema Draft 2020-12.

Required properties: $schema, schema_version, plugin_id, name, version, description, author, license, entry_point, and minimum_pride_version. Optional properties: homepage, documentation_url, repository_url, issue_tracker_url, keywords, category, screenshots, icon, and changelog_url. These map the semantic fields of the stable PRIDE Plugin Manifest specification.

Unknown properties MUST be accepted for forward compatibility. They MUST NOT affect identity, capability matching, permissions, authorization, trust, or execution policy. Implementations may ignore unknown properties and MUST preserve the original Manifest snapshot where required for audit/recovery. Known properties with invalid types or values still fail validation. A future semantic change requires a new schema version and compatibility rules.

The Manifest remains developer-owned identity/descriptive metadata. It MUST NOT contain authoritative Marketplace security analysis, permission grants, trust scores, capability authorization, or verification decisions.

## 7. Compatibility and versioning

Descriptor, Manifest, and trust configuration have independent schema_version values and stable $schema identifiers. Clients MUST reject unsupported major versions. Compatible additive optional properties may be accepted under the forward-compatibility rules. Existing property meaning and type cannot change within the same major version. Canonicalization is for signature input and does not authorize discarding the original metadata snapshot.

## 8. Implementation boundary

The application service remains SDK-neutral. Acquisition, ZIP inspection/extraction, digest/signature verification, and trust configuration are injected adapters/components. Filesystem paths are resolved only by the storage adapter. This specification does not introduce public SDK lifecycle RPCs.

Production installation remains disabled until required adapters and the numeric ZIP installation policy are implemented and tested.