# Plugin ZIP Installation Policy

**Version:** 1.0.0  
**Status:** Accepted  
**Document ID:** PZIP-0001  
**Policy ID:** `pride.plugin.zip-installation`  
**Related Specifications:** PTS-0002, PTS-0004  
**Module:** PRIDE Plugin Module

## 1. Purpose and authority

This policy defines the mandatory resource limits for version 1 PRIDE Plugin ZIP inspection and extraction. It is normative for every implementation that processes a package using media type `application/vnd.pride.plugin+zip`.

The machine-readable policy is `schemas/zip-installation-policy.v1.json` and is validated by `schemas/zip-installation-policy.schema.json`. The schema and policy instance are versioned with this document. Implementations MUST load a supported, valid policy; missing, malformed, or unsupported policy configuration MUST fail closed. Implementations MUST NOT silently substitute larger defaults.

## 2. Accepted limits

| Limit | Value | Enforcement meaning |
|---|---:|---|
| Compressed archive size | 100 MiB (104,857,600 bytes) | Maximum exact ZIP artifact byte length |
| Total expanded size | 512 MiB (536,870,912 bytes) | Maximum cumulative uncompressed bytes written for all regular files |
| Individual file size | 128 MiB (134,217,728 bytes) | Maximum uncompressed size of any one regular file |
| Entry count | 10,000 | Maximum total ZIP entries, including files and directory entries |
| Directory depth | 32 | Maximum number of directory components from archive root to an entry |
| Compression ratio | 100:1 | Maximum per-entry uncompressed-to-compressed size ratio |
| Normalized path length | 240 bytes | Maximum UTF-8 byte length of an entry's normalized, slash-separated relative path |
| Metadata allocation budget | 16 MiB (16,777,216 bytes) | Maximum parser-owned ZIP central-directory and normalized per-entry metadata budget |

MiB uses binary units: 1 MiB = 1,048,576 bytes. All byte limits are inclusive upper bounds; a value greater than the configured maximum MUST be rejected.

## 3. Enforcement semantics

### 3.1 Archive and expanded bytes

- Compressed archive size is measured from the exact acquired artifact bytes before ZIP parsing.
- Expanded size is measured from bytes actually emitted while streaming each regular file to staging. ZIP header sizes are advisory and MUST NOT be trusted as the enforcement counter.
- The cumulative expanded-byte counter MUST use overflow-safe arithmetic and MUST stop extraction before writing beyond the limit.
- The individual-file limit and total-expanded limit apply simultaneously.
- Directory entries do not contribute expanded file bytes.

### 3.2 Entries and paths

- Entry count includes every central-directory entry, including explicit directory entries.
- Paths MUST first pass the PTS-0004 ZIP path normalization and rejection rules. The path-length limit applies to the resulting canonical relative path encoded as UTF-8, without a leading slash.
- Directory depth is the number of directory components in the normalized entry path. For a regular file, this is the number of path components preceding its filename. For a directory entry, its final directory component is included.
- Duplicate normalized paths, case-folding collisions, absolute paths, drive-qualified paths, parent traversal, backslash ambiguity, symlinks, and special files remain rejected independently of these numeric limits.

### 3.3 Compression ratio

For each regular file entry, the ratio is the uncompressed size divided by the compressed size. The comparison MUST avoid floating-point rounding and integer overflow. An entry whose ratio exceeds 100:1 MUST be rejected.

An entry with zero compressed bytes and nonzero uncompressed bytes MUST be rejected. An empty entry with both sizes zero is permitted. The ratio limit is per entry; the archive-size and total-expanded-size limits independently constrain aggregate resource use.

### 3.4 Metadata allocation

The metadata budget covers parser-owned central-directory bytes and normalized per-entry metadata retained during inspection/extraction. Implementations MUST bound the central-directory input before parsing and MUST account for normalized entry metadata before retaining it. They MUST NOT rely solely on entry count as a proxy for metadata memory.

The budget is an implementation resource ceiling, not a claim that every Go runtime allocation can be measured exactly. A parser that cannot enforce a bounded pre-parse and retained-metadata budget MUST NOT be used for production package processing.

### 3.5 ZIP64 compatibility\n\nZIP64 is not supported by PZIP-0001 v1.0.0. Inspectors MUST reject ZIP64 end-of-central-directory records and locators, ZIP64 sentinel values in classic EOCD or entry headers, and ZIP64 extra fields in central or local headers. Packages that require ZIP64 MUST be republished using the classic ZIP format within the accepted size and entry limits.\n\n### 3.5 ZIP64 compatibility\n\nZIP64 is not supported by PZIP-0001 v1.0.0. Inspectors MUST reject ZIP64 end-of-central-directory records and locators, ZIP64 sentinel values in classic EOCD or entry headers, and ZIP64 extra fields in central or local headers. Packages that require ZIP64 MUST be republished using the classic ZIP format within the accepted size and entry limits.\n\n## 4. Failure behavior

Exceeding any limit MUST reject the package before promotion. Partial extraction MUST remain confined to the operation-scoped staging workspace and MUST never be marked verified or installed. The caller may clean or quarantine the failed workspace according to the separately approved recovery/retention policy; it MUST NOT expose partial content to discovery or runtime.

Limit failures MUST map to stable, non-secret failure categories. Error messages MUST NOT include absolute filesystem paths, credentials, raw package contents, or unbounded attacker-controlled strings.

## 5. Compatibility and changes

The policy is immutable once published as version 1.0.0. Changing any numeric limit or its enforcement semantics requires a new policy version and corresponding schema/fixture updates. A client MUST NOT reinterpret an existing policy version with different values.

A future policy version may raise or lower limits only through an explicit reviewed change. Installers MUST identify the policy version used for verification in their installation evidence so recovery can apply the same policy or a stricter explicitly approved policy.

## 6. Implementation boundary

This policy does not implement a ZIP parser, extractor, Manifest decoder, artifact verifier, trust loader, filesystem promoter, or startup recovery coordinator. It authorizes implementation work against these limits; it does not enable production installation by itself.

PTS-0004 remains normative for archive layout, path safety, accepted entry types, Manifest location, digest/signature requirements, and trust boundaries. PTS-0002 remains normative for installation orchestration and recovery.
