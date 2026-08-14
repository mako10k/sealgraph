# Initial implementation plan

This plan is descriptive. `PLAN.pert` is the optional perttool projection.

## Phase 0 — lock semantic contracts

- Define canonical seal byte encoding.
- Define REF grammar/path escaping.
- Define ObjectID textual encoding.
- Finalize draft/historical seal admissibility.
- Write fixture-based hash tests before persisting real repositories.

## Phase 1 — native vertical slice

Implement without Git integration:

1. `sealgraph init`
2. native loose blob object write/read
3. one-REF working candidate
4. explicit root
5. `add --content`
6. `add/link --depend-on`
7. one-REF `seal -m`
8. `show`
9. minimal `status`
10. deterministic tests

Success criterion: a root and one derived REF can be sealed, upstream can be superseded, and direct stale is detected without stored stale metadata.

## Dogfood R0 — hermetic native workflow

Before Phase 2, execute the temporary-repository workflow in
[`dogfooding-plan.md`](dogfooding-plan.md): establish a three-REF chain,
supersede its root, observe direct stale, and repair each dependent explicitly.
Do not create the project-root `.sealgraph/` in R0.

## Phase 2 — graph semantics

- N:M DAG validation
- cycle rejection
- transitive stale
- reverse impact
- graph/stale/status
- linklog/log

The tracked R1 dogfood predecessor is the focused graph slice through
`graph`/`stale`/`status`/`impact`. `linklog` and `log` remain later Phase 2
inspection work and do not block the initial tracked manifest exercise.

## Phase 3 — attachments and diff

- attachment blob import
- attachment metadata hashing
- semantic diff across content/link/attachment changes

## Phase 4 — integrity/forensics

- fsck
- ref compare-and-swap
- corruption tests
- low-level Git-compatible object inspection validation

## Phase 5 — Git sidecar

- add stable go-git dependency
- GitObjectReader
- `git sealgraph init/status`
- Git content source bindings
- Git merge/index-stage inspection

After Phase 2 graph behavior is available and the hermetic round has passed,
the separately approved R1 dogfood round may add tracked canonical
`.sealgraph/` state to this repository. R1 is not implicit authorization from a
passing R0.

## Phase 6 — Git conflict assistant

- `git sealgraph conflicts`
- three-way BASE/OURS/THEIRS semantic display
- explicit ours/theirs resolution
- post-resolution stale/impact reporting
- no automatic semantic seal creation
