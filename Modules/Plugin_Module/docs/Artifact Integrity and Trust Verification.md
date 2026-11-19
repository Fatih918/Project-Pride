# Artifact Integrity and Trust Verification

**Status:** Implementation increment 10  
**Scope:** Descriptor verification primitives and trust-configuration verification boundary

## Verification sequence

The installation application service follows this order:

1. Acquire the immutable ZIP and its signed Registry descriptor into one operation workspace.
2. Inspect the ZIP and validate the developer-owned Manifest without extracting payload files.
3. Resolve trusted Marketplace signing material from an authenticated trust configuration.
4. Verify the descriptor's Ed25519 signature over the RFC 8785 JCS canonical JSON object with only `signature` omitted.
5. Validate descriptor identity, release metadata, HTTPS download URI, media type, exact byte length, and SHA-256 digest.
6. Extract the ZIP payload into the confined operation workspace.
7. Repeat descriptor signature, trust-key, byte-length, and SHA-256 verification after extraction.
8. Require the second verification evidence to exactly match the first before recording VERIFIED.
9. Promote only after the verified record is persisted.

The second verification prevents a staged artifact changed during extraction from being promoted based solely on an earlier digest check. The staging workspace remains private and is accessed through the confined storage adapter.

## Descriptor verifier

The `installation/artifactverify` package:

- Rejects duplicate JSON keys and malformed/trailing JSON.
- Accepts descriptor schema version 1.0.0 and the agreed ZIP media type.
- Requires the descriptor Plugin ID/version to match the inspected Plugin.
- Requires HTTPS without URI user-info.
- Requires SHA-256 lowercase hexadecimal digest and Ed25519 detached signature.
- Uses the exact acquired ZIP file for byte-length and digest verification.
- Uses RFC 8785 JCS through the pinned `cyberphone/json-canonicalization` implementation.
- Returns a canonical Plugin_Storage-relative final path and verification evidence.

Unknown descriptor properties remain in the signature payload, preserving forward compatibility without allowing unsigned metadata to be introduced.

## Trust configuration verifier

The trust resolver:

- Uses only public bootstrap keys supplied by the PRIDE distribution.
- Verifies the remote trust configuration's detached Ed25519 signature using JCS with `signature` omitted.
- Checks schema version, issue/expiry interval, key IDs, Ed25519 public-key encoding, key validity intervals, and active/revoked status.
- Delegates sequence rollback and same-sequence equivocation protection to an atomic `TrustSequenceStore` port.
- Never allows a remote trust configuration to introduce or replace the bootstrap root.

The sequence store contract requires idempotent acceptance of the same sequence/digest pair, rejection of lower sequences, and rejection of a different digest at an already accepted sequence.

## Explicitly not included

This increment does not create production Registry networking or Marketplace signing infrastructure. It also does not provide the persistent SQLite implementation of `TrustSequenceStore`, distribution bootstrap-key provisioning, application composition-root wiring, atomic filesystem promotion, or startup recovery. The installer must remain unavailable in production until those components are supplied and tested.

The artifact verifier and trust resolver are security components, not evidence that a complete production installation pipeline is ready.
