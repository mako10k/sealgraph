# ADR 0016: Standalone-only beta integrity and release boundary

Status: accepted on 2026-08-17 by explicit operator direction.

## Decision

Close the initial alpha development phase without publishing an alpha and
prepare `v0.1.0-beta.1` as a standalone-only Linux amd64 prerelease under the
MIT License. The beta has no file synchronization and no Git sidecar. Sidecar
adoption remains a separate discussion and cannot block the standalone beta.

The beta includes a read-only full-inventory `fsck` with human output and
`sealgraph/fsck/v1` JSON. It validates every physical loose object's path,
envelope and hash; every REF manifest and tag target; every canonical Seal and
referenced content/attachment object; and the complete parent-revision and
Cause DAGs. Valid historical/detached Seals and unreferenced blobs are reported
separately from corruption. It never writes cache, repairs, deletes, repacks,
changes file modes, or treats writable checkout modes as integrity authority.

The beta reads and preserves attachment-bearing state but defers `attach` and
`detach` mutation commands. `manifest` remains an explicit deterministic
path/digest claim and is not file synchronization.

Release binaries receive `0.1.0-beta.1` through deterministic link-time
injection; ordinary builds report `0.1.0-dev`. The deterministic archive
contains only the standalone binary, MIT license, and README. Tagging, pushing,
and GitHub Release creation remain separately approved non-idempotent actions.

## Consequences

PERT release readiness no longer depends on Git sidecar. Exact-SHA validation,
remote CI, artifact checksum/readback, and explicit publication approval still
gate release. No persisted format changes.
