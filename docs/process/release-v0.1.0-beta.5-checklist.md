# v0.1.0-beta.5 release checklist

Status: released and independently read back on 2026-08-21. Final evidence is
recorded in [`release-v0.1.0-beta.5-receipt.md`](release-v0.1.0-beta.5-receipt.md).

## Frozen scope

- Version: `0.1.0-beta.5`.
- Product: standalone `sealgraph`, Linux amd64, MIT License.
- Included: correct warning classification for routine revision-cache
  observation mismatch, aligned documentation, and regression coverage.
- Retained from beta.4: native comparison vocabulary and Bash completion;
  explicit local REF recovery and recoverable `ref drop`; local source bindings
  and content-only refresh.
- Excluded: filesystem watch, automatic add/seal, Git discovery/sidecar,
  recovery-log retention automation, remote/shared recovery, object deletion,
  reset/reflog/undo, signatures, remote storage, daemon/server behavior, and
  trust assertions.
- Artifact inventory: standalone `sealgraph`, `LICENSE`, and `README.md` only.

## Exact-source gate

- [x] Freeze one clean source commit SHA containing runtime, tests,
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

Validated source: `6cfd6cfde869366e65fffb1e8c3d48cd22d0fafb`.
GitHub Actions run: `32468221350` (success).

## Publication record

Fill and freeze before any tag or Release write:

```text
release version: 0.1.0-beta.5
validated source SHA: 6cfd6cfde869366e65fffb1e8c3d48cd22d0fafb
artifact: sealgraph_0.1.0-beta.5_linux_amd64.tar.gz
artifact SHA-256: 45e2832b67a62d2302c062f9a54826b0df3805ab49e34a3992d97712838f97f1
checksums artifact: sealgraph_0.1.0-beta.5_checksums.txt
checksums file SHA-256: 8540ea3ae680b81b070f28e004d266263128ee28faff29b9c000c6c9a3b9fa5f
release-note SHA-256: 4957b6c7c1f797afa38275b43dd97d2740d57ec5570c501ef9340972466a7747
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [x] Create immutable `v0.1.0-beta.5` exactly once and push through secdat.
- [x] Create one GitHub prerelease with only the approved artifacts and notes.
- [x] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [x] Record a final beta.5 release receipt without moving the tag.
