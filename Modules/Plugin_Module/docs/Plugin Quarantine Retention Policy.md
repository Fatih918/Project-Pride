# Plugin Quarantine Retention Policy

**Status:** Implemented as an SDK-neutral cleanup service and confined storage adapter. Production scheduling remains the host's responsibility.

## Default policy

| Limit | Default |
|---|---:|
| Maximum age | 30 days |
| Maximum quarantine directories | 100 |
| Maximum aggregate regular-file size | 1 GiB (1,073,741,824 bytes) |

All limits must be positive. The policy can be injected by the host; the defaults are the baseline, not hard-coded limits in the storage adapter.

## Eligibility and safety

A quarantine directory is named by its installation operation ID. Before any deletion, the retention service resolves every directory against the durable installation operation journal.

Only operations in `FAILED` or `ROLLED_BACK` are eligible. Directories whose journal record is missing, whose phase is non-terminal (including `PREPARING`, `STAGED`, `VERIFIED`, or `PROMOTED`), or whose operation ID does not match the journal record are retained.

The service resolves the complete inventory before deleting anything. A journal lookup error aborts the run without deleting entries. This avoids treating a database outage as permission to delete unclassified artifacts.

## Eviction order

1. Inventory all quarantine directories and their recursive regular-file sizes.
2. Remove eligible entries whose directory modification time is at least the configured maximum age.
3. If the remaining inventory exceeds the directory-count or aggregate-byte limit, remove the oldest remaining eligible entries first. Ties are ordered by operation ID.
4. Entries that are not eligible remain, even if that means the configured limits cannot be met. Safety takes precedence over quotas.

The age clock uses the quarantine directory's filesystem modification time. This timestamp is a local retention signal, not a cryptographic or audit timestamp. Directory size includes all regular files recursively and excludes directory metadata.

## Filesystem constraints

The storage adapter inventories and deletes through the confined `os.Root` handle rooted at `Plugin_Storage`. Directory enumeration uses opened directory handles; file metadata is inspected without following links. It rejects symlinks and special files instead of following them. It validates operation IDs and removes only the corresponding `quarantine/<operation-id>` directory.

The cleanup API does not accept arbitrary paths. It is deliberately separate from recovery and installation orchestration.

## Invocation and observability

The host may invoke `installation/quarantine.Retainer.Prune` after successful recovery or from a controlled maintenance task. It must not invoke cleanup before storage and the durable operation repository are available. The host should record the returned scanned/removed/retained counts and any error without exposing artifact contents.

This increment provides the policy and cleanup primitive; it does not add a background scheduler or wire cleanup into a production composition root. A cleanup failure must not cause the host to bypass startup recovery or expose an unreconciled Plugin registry.
