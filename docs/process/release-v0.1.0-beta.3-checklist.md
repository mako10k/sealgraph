# v0.1.0-beta.3 release checklist

Status: candidate preparation in progress. This file records the version-specific
gate; it is not publication authority or a release receipt.

## Frozen scope

- Version: `0.1.0-beta.3`.
- Product: standalone `sealgraph`, Linux amd64, MIT License.
- Included change: ADR 0019 local source bindings, content-only file refresh,
  initial REF-as-path import, and status v2 workfile observations.
- Excluded: filesystem watch, automatic add/seal, binding-history restore,
  Git discovery/sidecar, attachment mutation, automatic repair, signatures,
  remote storage, daemon/server behavior, and trust assertions.
- Artifact inventory: standalone `sealgraph`, `LICENSE`, and `README.md` only.

## Exact-source gate

- [ ] Freeze one clean source commit SHA containing runtime, tests, help,
      documentation, CI version, and these release notes.
- [ ] Run `gofmt` clean-tree check, `go vet ./...`, `go test ./...`, and
      `go test -race ./...` on that SHA.
- [ ] Run `npm ci`, clone, complexity, and dead-code checks.
- [ ] Audit all checked-in `.think` decisions required by the implementation.
- [ ] Run `perttool document check PLAN.pert` and
      `perttool dag analyze PLAN.pert`.
- [ ] Run focused source-binding, content-refresh, status-v2, and changed-file
      tests as part of the full Go suite.
- [ ] Build the release twice into separate absent output directories and
      prove byte-identical archives and checksum files.
- [ ] Run extracted-artifact smoke and confirm archive inventory.
- [ ] Push the exact source SHA and require successful remote CI for it.

## Publication record

Fill and obtain explicit operator approval before any tag or Release write:

```text
release version: 0.1.0-beta.3
exact commit SHA: PENDING
artifact: sealgraph_0.1.0-beta.3_linux_amd64.tar.gz
artifact SHA-256: PENDING
checksums artifact: sealgraph_0.1.0-beta.3_checksums.txt
checksums file SHA-256: PENDING
release-note SHA-256: PENDING
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [ ] Receive explicit approval of the completed record.
- [ ] Create immutable `v0.1.0-beta.3` exactly once and push through secdat.
- [ ] Create one GitHub prerelease with only the approved artifacts and notes.
- [ ] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [ ] Record a final beta.3 release receipt without moving the tag.
