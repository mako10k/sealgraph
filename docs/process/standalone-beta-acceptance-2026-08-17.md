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

A detached fresh checkout of implementation commit
`cb16a8614aa9832c662b29f55da89d6a79b11903` initially lacked ignored runtime
directories. Explicit init reported `BOOTSTRAPPED_RUNTIME index,locks`, after
which `fsck/v1` reproduced the same inventory above. The checkout remained
free of tracked changes.

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

README, LICENSE, binary inputs, and archive rules were fixed at the detached
implementation commit. Two builds were byte-identical and produced:

```text
archive sha256        09f33f9166c72b318b61d1f16d7ebc6f3843a4bf34c7dcaa62af43c03dfb1688
checksums-file sha256 50bd0ca446ebbf1886c6c41ceb24ca918b73b551a6624b5c1250771f4e5ea744
```

The final clean candidate commit must reproduce these bytes before publication
approval. Remote CI and downloaded-asset readback remain pending.
