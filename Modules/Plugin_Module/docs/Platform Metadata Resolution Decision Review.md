# Platform Metadata Resolution — Decision Review

**Status:** Proposed for review; no Registry metadata endpoint or production metadata wire format is approved by this document.  
**Scope:** Trusted resolution of Marketplace-owned Plugin type and protected metadata during installation and recovery.  
**Related:** Official Registry Descriptor Resolution Architecture Decision Record, PTS-0004, PRIDE Registry Format Specification, Plugin Module Production Host Integration Contract.

## 1. Problem

The Plugin Module deliberately separates developer-owned Manifest data from Marketplace-owned metadata.

The current application boundary exposes:

`PlatformMetadataResolver.Resolve(ctx, pluginID, version)`

The resolver supplies `PluginType` and `ProtectedMetadata`, but the repository does not yet define:

- the authoritative Registry representation for these fields at one immutable release;
- a wire schema or approved Registry operation for retrieving them;
- how the metadata is cryptographically bound to the artifact descriptor;
- how the resolver binds metadata to the exact namespace, Plugin ID, release ID, and expected version;
- how recovery obtains the same metadata after restart without trusting developer-controlled package content.

The Registry Format Specification establishes that Marketplace-generated metadata is authoritative and protected. The existing signed artifact descriptor, however, currently defines artifact identity, download location, size, digest, and signature; it does not define a typed platform-metadata object.

This is a contract gap, not a reason to invent a live endpoint or accept unsigned metadata.

## 2. Invariants

1. Plugin type, capabilities, permission IDs, and compatibility/security metadata MUST NOT be sourced from `plugin.json` or any other developer-controlled package file.
2. Metadata MUST be bound to the exact immutable Registry release being installed, not merely to a Plugin ID or a mutable version lookup.
3. Metadata that affects Plugin discovery or platform behavior MUST be cryptographically authenticated by Marketplace-controlled signing keys. HTTPS transport alone is insufficient.
4. The metadata identity MUST match the installation request's Plugin ID and expected version.
5. Installation and recovery MUST consume the same authenticated metadata evidence. Recovery must not silently resolve a different release's metadata.
6. The exact signed bytes used as verification evidence MUST remain unchanged in the installation journal.
7. The SDK-neutral Module must not own Marketplace credentials, OAuth flows, or Registry endpoint selection.
8. Metadata parsing and validation must be bounded, deterministic, duplicate-key safe, and fail closed.
9. Permission identifiers remain metadata only. This resolver does not authorize Plugin actions or replace the Core Permission Manager.
10. No production Registry hostname, authentication provider, Marketplace signing key, or metadata endpoint may be fabricated in the Module.

## 3. Candidate designs

### Option A — Embed platform metadata in the signed artifact descriptor

Extend the Registry artifact descriptor with a typed `platform_metadata` object. The Marketplace signs the complete descriptor payload, including this object, with the existing Ed25519 descriptor-signing mechanism.

The metadata object would initially carry only fields already represented by the Module domain:

- `plugin_type`: `active` or `passive`;
- `capabilities`: an array of capability ID, name, and description records;
- `permission_ids`: a unique array of non-empty identifiers;
- `compatibility_status`: a bounded non-empty status string.

The existing `ProtectedMetadata.Attributes` field is intentionally not standardized in this increment. Its semantics and allowed values have not been defined by the Registry Format contract. Unknown top-level descriptor fields may remain forward-compatible, but the known `platform_metadata` object must have a versioned, validated schema.

The descriptor schema should advance to a new minor schema version (proposed `1.1.0`) and require `platform_metadata` for new installations. The verifier must validate the metadata shape and verify the signature over the exact descriptor payload. A descriptor missing required platform metadata must not be accepted for a new installation.

The application and recovery paths parse the candidate metadata from the exact descriptor bytes already persisted in the operation journal. The parsed metadata is not considered trusted until the artifact verifier successfully verifies the descriptor signature. Domain validation and commit happen only after successful verification.

**Advantages**
- One immutable release descriptor binds artifact digest, release identity, and protected metadata.
- No second Registry request or independent metadata cache is required.
- Recovery can reproduce metadata from the durable descriptor evidence.
- Existing Ed25519 trust configuration and signature verification remain the single trust root.
- The metadata is guaranteed to be release-specific.

**Costs**
- The descriptor schema and verifier must be extended.
- Marketplace publishing/signing must emit the new field before new installations can succeed.
- Descriptor size remains bounded by the existing 1 MiB limit.
- Metadata changes require a new immutable release descriptor.

### Option B — Separate signed platform-metadata document

Keep the artifact descriptor focused on artifact delivery and add a separate Registry metadata document. The descriptor must bind the metadata document to the release using an immutable reference and cryptographic digest; the metadata document must also have an authenticated signature or be included in an authenticated signed envelope.

The Module would fetch and verify the metadata document through a separately configured bounded HTTPS transport, or retrieve it from a trusted immutable cache.

**Advantages**
- Artifact delivery and platform metadata can evolve as separate documents.
- Large or independently versioned Marketplace analysis records need not enlarge the artifact descriptor.

**Costs**
- Requires another wire schema, endpoint or retrieval mechanism, and verification path.
- Requires explicit descriptor-to-metadata binding and independent replay/rollback rules.
- Recovery must persist or reliably re-fetch the exact metadata evidence.
- Adds network and operational dependencies to installation and startup recovery.
- A plain HTTPS response without a signature and descriptor-bound digest is not sufficient.

### Option C — Resolve metadata from a trusted HTTPS endpoint without a signed metadata binding

The host configures a Registry endpoint and the resolver trusts the response because it was fetched over HTTPS.

**Rejected.** TLS authenticates the transport endpoint, but this design does not bind metadata to the Marketplace-signed immutable artifact release. It also creates a mutable lookup/recovery risk and weakens the existing signed-descriptor trust model.

### Option D — Derive platform metadata from the Plugin Manifest or package

**Rejected.** This violates the developer/Marketplace ownership boundary and permits package authors to self-assert protected capabilities, permissions, compatibility, or Plugin type.

## 4. Recommended direction

**Recommend Option A: include typed platform metadata in the signed artifact descriptor.**

The descriptor already represents one immutable Registry release and is verified using the Marketplace Ed25519 trust chain. Its signing payload is canonicalized after removing only the signature field, so a typed metadata member can be covered by the same signature without creating a second trust system.

This recommendation does not mean the current verifier already validates platform metadata. It does not. The verifier currently decodes only artifact fields, while its signing-payload construction includes all descriptor fields other than `signature`. Explicit schema validation and typed decoding are required before the recommendation can be implemented safely.

The implementation should use a typed metadata extraction port over the exact descriptor bytes, not a lookup keyed only by Plugin ID/version. The application and recovery coordinators must verify the descriptor signature before treating the extracted metadata as trusted or committing the assembled Plugin.

## 5. Decisions requested

| ID | Decision | Recommended option | Notes |
|---|---|---|---|
| M1 | Metadata source | A — embedded in signed artifact descriptor | One signed immutable release envelope |
| M2 | Descriptor schema version | 1.1.0 minor revision | New installations require the metadata object |
| M3 | Initial metadata fields | Plugin type, capabilities, permission IDs, compatibility status | Match existing domain model only |
| M4 | Arbitrary attributes | Defer | Do not publish unstandardized security fields through `map[string]any` |
| M5 | Release binding | Exact descriptor identity: namespace/reference, Plugin ID, release ID, expected version | No mutable Plugin ID-only lookup |
| M6 | Trust semantics | Metadata is untrusted until existing Ed25519 descriptor verification succeeds | TLS is not metadata authenticity |
| M7 | Legacy descriptor behavior | Reject missing metadata for new installs; define explicit recovery compatibility before production enablement | No silent metadata defaults |
| M8 | Permission semantics | Metadata only; authorization remains Core-owned | No authorization logic in Plugin Module |

## 6. Proposed metadata shape

Illustrative shape only; it becomes normative only after M1–M4 are approved:

```json
{
  "platform_metadata": {
    "schema_version": "1.0.0",
    "plugin_type": "active",
    "capabilities": [
      {
        "id": "text.generate",
        "name": "Text generation",
        "description": "Generates text from an input request."
      }
    ],
    "permission_ids": [
      "network.http"
    ],
    "compatibility_status": "verified"
  }
}
```

Validation must at minimum enforce:
- supported metadata schema version;
- Plugin type is exactly `active` or `passive`;
- capability IDs are non-empty and unique;
- permission IDs are non-empty and unique;
- bounded strings and collection sizes;
- no duplicate JSON keys at any nesting level;
- no unknown fields inside the versioned metadata object unless a later schema version explicitly permits them.

Capability identifiers and permission identifiers must be interpreted as opaque platform identifiers here. This contract does not define capability matching or permission authorization.

## 7. Required implementation sequence after approval

1. Update PTS-0004 and the descriptor JSON schema for the selected metadata contract.
2. Extend descriptor decoding and strict validation; preserve the exact raw descriptor bytes for JCS signature verification.
3. Replace the Plugin ID/version-only metadata resolver input with a typed descriptor-backed request or extraction port.
4. Reorder installation and recovery so extracted metadata cannot be treated as trusted until descriptor signature verification succeeds.
5. Add tests for missing/malformed metadata, duplicate IDs, duplicate JSON keys, identity mismatch, unsupported metadata schema, invalid signatures, and recovery from persisted descriptor evidence.
6. Update the production host integration contract and Module README to reflect the new boundary.
7. Keep installation disabled until the Marketplace signer actually publishes the approved metadata shape and the host supplies the real trust configuration.

## 8. Explicit non-goals

This decision review does not approve:
- a deployed Registry metadata endpoint;
- Marketplace publishing implementation or signer changes;
- capability matching or permission authorization;
- Plugin loading, lifecycle, or execution;
- arbitrary security attributes or trust-score semantics;
- production host composition or real bootstrap key material.

The purpose is to settle the source, integrity, and recovery contract before writing a production adapter.
