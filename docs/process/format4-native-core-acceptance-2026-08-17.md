# Format-4 native core acceptance — 2026-08-17

## Achieved scope

The checked-in standalone runtime now uses `repository_format = 4`,
`sealgraph/seal/v4`, and `sealgraph/candidate/v4` without a format-3 repository
reader or ignored legacy fields.

- Seal bytes contain `parent_revision` and exact Cause `target_seal` only;
  owner `ref` and Link `target_ref` are absent.
- Candidate bytes separate `parent_revision` from `expected_ref_head` and use
  deterministic persisted JSON.
- Selectors implement exact REF HEAD, repository-wide `@hex`, and ancestry-
  checked `REF@hex`. Scoped non-hex tags fail closed at `TAG_CONTRACT`.
- `sealgraph load --format logical-v1` parses the migration document rather
  than a source repository, accepts only an absent target, rebuilds identities,
  validates the staged graph, and publishes with Linux amd64
  `renameat2(RENAME_NOREPLACE)`.
- The canonical load receipt contains the exact source digest, every sorted
  old-to-new pair, complete many-to-one groups, rewritten REFs, no silently
  dropped tags, and published format 4.
- Prior `.sealgraph-load-*` crash evidence is reported and left untouched.

Fixture identities are:

- canonical Seal payload: `d73988845debf3a426e92b33b3269f6dcca41f5dce265f8630c80f88911364ec`;
- deterministic candidate bytes including LF: `2400bb778e6275421fe1c4651cddd500dfb7ec86c3d3dc4552b6ad0b34149fb9`.

## Explicitly deferred

`FORMAT4_NATIVE_CORE` does not claim the active revision index. Normal
non-root publication, `status`, `stale`, `graph`, `impact`, `log`, `linklog`,
and `diff` fail explicitly rather than applying format-3 owner checks or
claiming a false clean result. `derive`, `add --parent`, active-leaf admission,
cache/`--scan`, and branch-aware graph presentation belong to
`FORMAT4_REVISION_GRAPH`.

Tags, REF move, tracked dogfood conversion, Git sidecar, release, publication,
and external tracker mutation remain unstarted. The project-root `.sealgraph`
still records format 3 and is intentionally not changed or opened by the
format-4 runtime.

## Validation evidence

Focused tests cover REF-independent identity, candidate field separation,
exact/scoped selectors, legacy-runtime rejection, normal-publication gating,
binary migration material, many-to-one mapping, tag rejection, existing target
preservation, abandoned staging evidence, and atomic no-replace behavior.

The actual format-3 binary at commit
`5b24d47cb66e2ff2be000d4c9cb32cb59e7957fa` created a tagless two-REF
repository and emitted a 1,343-byte logical dump with SHA-256
`9ba23a562dd405ef742f45a15bb08aa0e0b7ddd66e2fa824a34e793042a541ee`.
The current format-4 binary loaded it into an absent target, emitted a 737-byte
receipt, and showed the converted CHILD content as exact `child-v3` bytes. The
temporary binaries/repositories were moved to desktop trash and remain
recoverable.

Repository-wide checks passed:

```text
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
npm ci                         # 0 vulnerabilities
npm run clone-check            # 0 clones
perttool document check PLAN.pert
perttool dag analyze PLAN.pert # PTDAG-302 display truncation warning only
perttool dag next PLAN.pert     # FORMAT4_REVISION_GRAPH only
git diff --check
```

The accepted ADR 0011 and ADR 0012 decision models also audit at
`fatal=0 error=0 warning=0`; their remaining info/hints are existing explicit
pending alternatives and style hints, not implementation blockers.
