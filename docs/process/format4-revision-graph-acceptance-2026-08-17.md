# Format-4 revision graph acceptance — 2026-08-17

## Achieved scope

The standalone format-4 runtime now derives one active revision DAG from a
coherent complete current REF/head observation. Parent edges and Link-only
Cause edges remain typed and are never interchanged.

- `internal/revision` validates branching `parent_revision` ancestry,
  deduplicates aliases, classifies active leaves/non-leaves, and treats
  unreachable ODB objects as historical/detached.
- `derive NEW_REF --from SOURCE` copies exact identity-bearing material into an
  absent-destination candidate with SOURCE as parent. `add NEW_REF --parent
  SOURCE` creates new material with the same explicit absent-destination/CAS
  boundary.
- Normal non-draft publication requires every direct and reachable Cause to be
  a non-draft active revision leaf. Rejection retains the candidate; no stale
  target, sibling, or repair is chosen automatically.
- `status` distinguishes `UNSEALED`, `DRAFT`, `STALE_SELF`, `STALE_DIRECT`, and
  `STALE_TRANSITIVE`. `stale --frontier --refs-only` uses only exact Link
  closure and current stale-head identities.
- `.sealgraph/cache/revision-v1.json` is checksum- and complete-observation-
  bound disposable state. Invalid/mismatched cache scans and refreshes;
  `--scan` bypasses reads; refresh failure only warns after canonical
  validation. Cache symlinks are never followed.
- `log`, `diff`, and `linklog` follow ownerless parent identities. Link history
  presents an ancestry-related repoint only when the removed/added match is
  unambiguous; N:M ambiguity remains add/remove.
- `impact` accepts every Seal selector, matches the selected Seal or its
  revision ancestors, groups current REF aliases by downstream Seal, chooses
  deterministic shortest paths, and bounds explicit all-path presentation
  without limiting membership or graph validation.
- All multi-REF results are buffered and the complete REF/head set is
  revalidated before the repository method returns.

## Static-analysis gate added with this slice

`go vet` remains the standard suspicious-construct analyzer. The development
and CI gate now also pins and runs:

- jscpd `5.0.12`, threshold `0.1%`, for Go clone detection;
- gocyclo `v0.6.0`, maximum cyclomatic complexity `20` per function;
- `golang.org/x/tools/cmd/deadcode` `v0.49.0` with `-test` for RTA-unreachable
  functions across production and test entrypoints.

The tools are invoked only as development checks and are not `go.mod` or
runtime dependencies. Existing functions over the new complexity ceiling were
split by responsibility; dormant format-4 gate scaffolding was either
activated or removed. The accepted baseline has zero clones and zero reported
dead functions.

## Explicitly deferred

The rename-safe tag namespace and REF `mv` transaction remain `TAG_CONTRACT`.
Tracked project dogfood conversion, content manifest/usability work, recurring
dogfood, Git sidecar, release, push, and external tracker mutation remain
separately gated. The project-root `.sealgraph` remains format 3 and was not
opened or converted by the format-4 runtime.

## Validation evidence

Focused tests cover branching/aliases, direct and transitive stale, exact-Cause
frontier, active-leaf publication rejection with candidate retention,
same-material derive, explicit add-parent, revision-aware impact, deterministic
bounded diamond paths, cache corruption/rebuild, unsafe cache symlink refusal,
complete-head revalidation, history/diff/repoint, and CLI line/flag contracts.

Repository-wide checks passed:

```text
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
npm ci                         # 0 vulnerabilities
npm run clone-check            # 0 clones
make complexity-check          # no function above 20
make deadcode-check            # no unreachable function
perttool document check PLAN.pert
perttool dag analyze PLAN.pert # PTDAG-302 display truncation warning only
perttool dag next PLAN.pert     # TAG_CONTRACT only
git diff --check
```

`FORMAT4_REVISION_GRAPH` reaches `REVISION_DAG`; this acceptance does not
authorize `TAG_CONTRACT`, tracked dogfood conversion, release, or publication.
