# Standalone beta preparation acceptance — 2026-08-17

Status: implementation prepared for `v0.1.0-beta.1`; exact final SHA, remote
CI, tag, push, and GitHub prerelease remain unapproved and incomplete.

## Frozen scope

- License: MIT.
- Artifact: Linux amd64 standalone `sealgraph` only.
- No file synchronization, watcher, implicit source import, or working-tree
  comparison. `manifest` remains an explicit path/digest claim.
- No Git discovery, hook, or `git-sealgraph` artifact. Sidecar adoption is a
  separate future product decision.
- Attachment-bearing state is read and preserved; `attach` and `detach`
  mutations are deferred.

## Full-inventory integrity

`fsck` validates every loose object path, Git-compatible envelope and native
SHA-256 identity; every REF manifest and tag target; all canonical Seal bytes
and material references; and every parent-revision and Cause graph. It
re-observes complete REF heads, tags, and object bytes before emitting success.
Valid historical/detached Seals and unreferenced blobs are reported separately
and are never deleted or repaired.

The tracked repository observation was:

```text
FSCK_OK objects=13 seals=7 material_objects=6 refs=6 tags=4 active_seals=7 historical_or_detached=0 unreferenced_objects=0
```

Focused fixtures cover clean inventory, detached Seal reporting, unreferenced
blob reporting, corrupt envelopes/hashes, missing material, human output, and
`sealgraph/fsck/v1` JSON. Existing lower-level REF/object and graph fixtures
cover unsafe canonical paths, malformed manifests, missing targets, and both
DAG cycle errors.

## Artifact preparation

Release builds inject `0.1.0-beta.1`; development builds remain
`0.1.0-dev`. Two independent local builds produced byte-identical archives and
checksum files. The archive contains one top-level directory with exactly
`sealgraph`, `LICENSE`, and `README.md`. The extracted-artifact smoke covers
version, init, root/dependent publication, stale frontier, successful fsck, and
corruption refusal.

Artifact digests are intentionally not frozen here because README and
acceptance metadata are still changing. They must be recomputed from the final
clean exact SHA before publication approval.
