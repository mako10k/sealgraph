# v0.1.0-beta.5 release checklist

Status: release candidate preparation in progress. Publication is separately
approval-gated.

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

- [ ] Freeze one clean source commit SHA containing runtime, tests,
      documentation, CI version, release notes, and this checklist.
- [ ] Run gofmt clean-tree check, `go vet ./...`, `go test ./...`, and
      `go test -race ./...` on that SHA.
- [ ] Run `npm ci`, completion, clone, complexity, and dead-code checks.
- [ ] Audit checked-in llmthink decisions used by this implementation.
- [ ] Run `perttool document check PLAN.pert` and
      `perttool dag analyze PLAN.pert`.
- [ ] Build twice into separate absent directories and prove byte-identical
      archives and checksum files.
- [ ] Run extracted-artifact smoke and confirm archive inventory.
- [ ] Push the exact source SHA and require successful remote CI for it.

Validated source: pending.
GitHub Actions run: pending.

## Publication record

Fill and freeze before any tag or Release write:

```text
release version: 0.1.0-beta.5
validated source SHA: pending
artifact: sealgraph_0.1.0-beta.5_linux_amd64.tar.gz
artifact SHA-256: pending
checksums artifact: sealgraph_0.1.0-beta.5_checksums.txt
checksums file SHA-256: pending
release-note SHA-256: pending
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [ ] Create immutable `v0.1.0-beta.5` exactly once and push through secdat.
- [ ] Create one GitHub prerelease with only the approved artifacts and notes.
- [ ] Independently read back remote tag, prerelease metadata, asset identities,
      downloaded hashes, and extracted-artifact smoke.
- [ ] Record a final beta.5 release receipt without moving the tag.
