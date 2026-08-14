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

The tracked R1 dogfood predecessor is the focused graph slice through
`graph`/`stale`/`status`/`impact`. `linklog` and `log` remain later Phase 2
inspection work and do not block the initial tracked manifest exercise.

## History inspection slice

- validated one-REF parent-chain traversal
- `log`
- derived `linklog` add/remove/repoint events
- semantic `diff` for all canonical seal fields
- focused R1 history dogfood

This slice adds no persisted fields and no Git history/reflog semantics. Content
diff is identity-based and does not print arbitrary blob bytes.

## Experimental native v2 and decision dogfood

- replace algorithm-tagged native IDs with full 64-character hex IDs
- resolve user selectors through repository-wide unique prefixes or REF-scoped
  immutable tags
- keep Git-compatible SHA-256 loose blob objects
- remove the redundant persisted link kind and add optional hash-committed link
  rationale
- reject format 1 rather than add a compatibility reader or automatic migration
- regenerate tracked dogfood state and seal ADR 0006 after validation

## Phase 3 — attachments

- attachment blob import
- attachment metadata hashing
- attachment CLI integration with the existing semantic diff model

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
