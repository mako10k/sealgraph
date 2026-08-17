# Format-4 tag contract acceptance — 2026-08-17

## Achieved scope

- ADR 0013 replaces the collision-prone plain REF/tag trees with one canonical
  `sealgraph/ref/v1` manifest at `refs/seals/<REF>/.ref`.
- The exact config now includes `ref_format = manifest-v1`; the interim
  format-4 plain-head layout is rejected without a dual reader or migration.
- A REF manifest contains one full HEAD and its sorted immutable raw-TAGNAME
  bindings. Decode requires strict canonical re-encoding equality.
- Prefix REFs such as `design` and `design/api` coexist. Candidate and per-REF
  lock paths use `.candidate` and `.lock` terminal markers.
- `tag REF [TAGNAME]` lists or creates immutable scoped aliases. Creation uses
  an unchanged-head CAS, is idempotent for the same target, and rejects
  retargeting and unscoped Seal selectors.
- `mv OLD_REF NEW_REF` validates source HEAD/tags, rejects exact source or
  destination candidates, and moves the single manifest with an atomic
  no-replace rename. It creates no old-name alias and moves no candidate.
- Logical-v1 load preserves tag records through the complete old-to-new
  SealID map, validates tag-rooted graph closure, and includes rewritten tags
  in the load receipt.

## Boundary evidence

- Both prefix/tag creation orders succeed without a loose-path collision.
- Existing move destinations are not replaced and failed moves retain source.
- HEAD updates preserve manifest tag bindings; stale observed-head tag
  creation fails without mutation.
- Tag scopes and all bindings move together; the old scope no longer resolves.
- Corrupt/noncanonical manifests, symbolic links, invalid IDs/names, and
  unknown entries fail closed and are never repaired.

## Explicitly deferred

This slice does not convert the tracked format-3 `.sealgraph`, move candidates,
delete or force tags, perform recursive namespace moves, add Git-sidecar
behavior, publish a release, push a commit, or mutate an external tracker. The
next PERT frontier is `FORMAT4_DOGFOOD_LOAD`.

## Validation

The acceptance run passed:

- `gofmt -w .`, `go vet ./...`, `go test ./...`, and `go test -race ./...`;
- `npm ci` with zero reported vulnerabilities;
- jscpd over 44 Go files with zero clones;
- gocyclo with no function above the configured complexity limit 20;
- deadcode with test entrypoints and zero reported unreachable functions;
- PERT document check and analysis; `dag next` selects only
  `FORMAT4_DOGFOOD_LOAD`;
- `git diff --check`.

PERT analysis retains the informational `PTDAG-302` warning because critical
path enumeration is intentionally capped at one representative path.
