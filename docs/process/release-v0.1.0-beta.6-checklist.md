# v0.1.0-beta.6 release checklist

Status: released and independently read back on 2026-08-24. Final evidence is
recorded in [`release-v0.1.0-beta.6-receipt.md`](release-v0.1.0-beta.6-receipt.md).

## Frozen scope

- Version: `0.1.0-beta.6`.
- Product: standalone `sealgraph`, Linux amd64, MIT License.
- Included: terminal-first aligned and width-aware human output, automatic
  versioned JSON for known non-terminal inspection output, explicit format
  overrides, candidate inspection JSON schemas, terminal mutation receipts,
  ADR 0021 attachment-scope retirement, aligned documentation, and regression
  coverage.
- Retained from beta.5: revision-cache diagnostic classification, native
  comparison vocabulary, Bash completion, explicit local REF recovery,
  recoverable `ref drop`, local source bindings, and content-only refresh.
- Excluded: filesystem watch, automatic add/seal, Git discovery/sidecar,
  attachment mutation, recovery-log retention automation, remote/shared
  recovery, object deletion, reset/reflog/undo, signatures, remote storage,
  daemon/server behavior, and trust assertions.
- Artifact inventory: standalone `sealgraph`, `LICENSE`, and `README.md` only.

## Exact-source gate

- [x] Freeze one clean source commit SHA containing runtime, tests,
      documentation, CI version, release notes, and this checklist.
- [x] Run gofmt clean-tree check, `go vet ./...`, `go test ./...`, and
      `go test -race ./...` on that SHA.
- [x] Run `npm ci`, completion, clone, complexity, and dead-code checks.
- [x] Audit the accepted ADRs and checked-in decisions used by this release.
- [x] Run `perttool document check PLAN.pert` and
      `perttool dag analyze PLAN.pert`.
- [x] Build twice into separate absent directories and prove byte-identical
      archives and checksum files.
- [x] Run extracted-artifact smoke and confirm archive inventory.
- [x] Push the exact source SHA and require successful remote CI for it.

Validated source: `b1166444172e0f257cdd09bd14216ca67b505c15`.
Exact-source GitHub Actions run: `32711251818` (success).
Tag and publication SHA: `13d94fa92c89cd2d29890307e11051a2e813120b`.
Publication GitHub Actions run: `32711506731` (success).

## Publication record

Fill and freeze before any tag or Release write:

```text
release version: 0.1.0-beta.6
validated source SHA: b1166444172e0f257cdd09bd14216ca67b505c15
artifact: sealgraph_0.1.0-beta.6_linux_amd64.tar.gz
artifact SHA-256: 1cf4c8e71e16b3783c7a45275eee25c21dede94211f299ccc2485b9ae404189d
checksums artifact: sealgraph_0.1.0-beta.6_checksums.txt
checksums file SHA-256: 89bc89ca453c7ea8454f25b43483a6a1e22fc4088faf6d50e00cdf71d1667766
release-note SHA-256: fa21c89e8b022c35fab53e50e39bced9335b3b3e6cf5a80bcd9340ce28f10115
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [x] Create immutable `v0.1.0-beta.6` exactly once and push through the
      authenticated release boundary.
- [x] Create one GitHub prerelease with only the approved artifacts and notes.
- [x] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [x] Install the downloaded binary and verify its version and identity.
- [x] Record a final beta.6 release receipt without moving the tag.
