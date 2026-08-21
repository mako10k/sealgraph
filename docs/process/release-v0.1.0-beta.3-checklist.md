# v0.1.0-beta.3 release checklist

Status: exact source and artifacts validated; publication approval pending.
This file records the version-specific gate; it is not publication authority
or a release receipt.

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

- [x] Freeze one clean source commit SHA containing runtime, tests, help,
      documentation, CI version, and these release notes.
- [x] Run `gofmt` clean-tree check, `go vet ./...`, `go test ./...`, and
      `go test -race ./...` on that SHA.
- [x] Run `npm ci`, clone, complexity, and dead-code checks.
- [x] Audit all checked-in `.think` decisions required by the implementation.
- [x] Run `perttool document check PLAN.pert` and
      `perttool dag analyze PLAN.pert`.
- [x] Run focused source-binding, content-refresh, status-v2, and changed-file
      tests as part of the full Go suite.
- [x] Build the release twice into separate absent output directories and
      prove byte-identical archives and checksum files.
- [x] Run extracted-artifact smoke and confirm archive inventory.
- [x] Push the exact source SHA and require successful remote CI for it.

Validated source: `9c80756490b63baa4641f3b5680cfd6e9b065816`.
GitHub Actions run: `32451570635` (success).

## Publication record

Fill and obtain explicit operator approval before any tag or Release write:

```text
release version: 0.1.0-beta.3
validated source SHA: 9c80756490b63baa4641f3b5680cfd6e9b065816
artifact: sealgraph_0.1.0-beta.3_linux_amd64.tar.gz
artifact SHA-256: 4421d8627b892a829b4dc4011ab0f9faabcc0d53494544cd537a0f962e5921c8
checksums artifact: sealgraph_0.1.0-beta.3_checksums.txt
checksums file SHA-256: c768b430581b8492265b38f78972d98380319483d870f0214d02bfaeac551b50
release-note SHA-256: 2c1a84f4f6228986b389a02d3e2cee0e68da3ef8823e273839420ade114610bf
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [ ] Receive explicit approval of the completed record.
- [ ] Create immutable `v0.1.0-beta.3` exactly once and push through secdat.
- [ ] Create one GitHub prerelease with only the approved artifacts and notes.
- [ ] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [ ] Record a final beta.3 release receipt without moving the tag.
