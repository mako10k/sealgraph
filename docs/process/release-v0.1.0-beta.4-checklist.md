# v0.1.0-beta.4 release checklist

Status: released and independently read back on 2026-08-21. Final evidence is
recorded in [`release-v0.1.0-beta.4-receipt.md`](release-v0.1.0-beta.4-receipt.md).

## Frozen scope

- Version: `0.1.0-beta.4`.
- Product: standalone `sealgraph`, Linux amd64, MIT License.
- Included: ADR 0020 native operation vocabulary and Bash completion; ADR 0018
  explicit local REF recovery; recoverable `ref drop`.
- Retained from beta.3: explicit local source bindings and content-only refresh.
- Excluded: filesystem watch, automatic add/seal, Git discovery/sidecar,
  recovery-log retention automation, remote/shared recovery, object deletion,
  reset/reflog/undo, signatures, remote storage, daemon/server behavior, and
  trust assertions.
- Artifact inventory: standalone `sealgraph`, `LICENSE`, and `README.md` only.

## Exact-source gate

- [x] Freeze one clean source commit SHA containing runtime, tests, help,
      documentation, CI version, release notes, and this checklist.
- [x] Run gofmt clean-tree check, `go vet ./...`, `go test ./...`, and
      `go test -race ./...` on that SHA.
- [x] Run `npm ci`, completion, clone, complexity, and dead-code checks.
- [x] Audit checked-in llmthink decisions used by this implementation.
- [x] Run `perttool document check PLAN.pert` and
      `perttool dag analyze PLAN.pert`.
- [x] Build twice into separate absent directories and prove byte-identical
      archives and checksum files.
- [x] Run extracted-artifact smoke and confirm archive inventory.
- [x] Push the exact source SHA and require successful remote CI for it.

Validated source: `8e7ca4bc9a20261cd1f09447849b9bd6ba9f796d`.
GitHub Actions run: `32462204170` (success).

## Publication record

Fill and obtain explicit operator approval before any tag or Release write:

```text
release version: 0.1.0-beta.4
validated source SHA: 8e7ca4bc9a20261cd1f09447849b9bd6ba9f796d
artifact: sealgraph_0.1.0-beta.4_linux_amd64.tar.gz
artifact SHA-256: eff9136499c24e71ec909448591f0c4ae268afd157cc4aaf58844407b253e1b0
checksums artifact: sealgraph_0.1.0-beta.4_checksums.txt
checksums file SHA-256: 481aa74c040eedd89d31bbd86a5015da80d37b153f741db87a51757c70c81c71
release-note SHA-256: cae36b1cc9187effca5cac97e8bd578d5f31748fb0602ec86bfc832a7a0e65ab
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [x] Receive explicit approval of the completed record.
- [x] Create immutable `v0.1.0-beta.4` exactly once and push through secdat.
- [x] Create one GitHub prerelease with only the approved artifacts and notes.
- [x] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [x] Record a final beta.4 release receipt without moving the tag.
