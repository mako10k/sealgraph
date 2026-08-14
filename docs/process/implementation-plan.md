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
7. one-REF `seal`
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

## Stale review frontier slice — complete

- extend `stale` with orthogonal `--frontier` and `--refs-only` flags;
- derive all/frontier membership from current sealed provenance without reading
  candidates;
- capture and revalidate the complete REF/head observation before buffered
  output;
- keep the REF-only line protocol deterministic and stable;
- add chain, diamond, candidate-corruption, concurrent-head-change, and
  read-only tests.

The accepted contract is recorded in
[ADR 0010](../adr/0010-stale-review-frontier.md). It adds no automatic relink,
reseal, repair, or batch publication operation.

## Experimental native v2 and decision dogfood

- replace algorithm-tagged native IDs with full 64-character hex IDs
- resolve user selectors through repository-wide unique prefixes or REF-scoped
  immutable tags
- keep Git-compatible SHA-256 loose blob objects
- remove the redundant persisted link kind and add optional hash-committed link
  rationale
- reject format 1 rather than add a compatibility reader or automatic migration
- regenerate tracked dogfood state and seal ADR 0006 after validation

## Material-identity native v3

- remove seal-level event `message` and `created_at` from canonical identity
- do not add `actor` or an unauthenticated mutable event log
- retain edge-specific link messages as dependency relation state
- represent actor/time/approval evidence as separately sealed content with an
  explicit concrete link
- reject format 2 rather than add a compatibility reader or migration
- regenerate tracked dogfood state explicitly after validation

## External-spec consistency gate — complete

Before another product slice or recurring dogfood is treated as routine:

- [x] linearize seal publication and serialize cooperative standalone writers;
- [x] prevent a sealed candidate version from deleting a newer candidate edit;
- [x] reject draft seals anywhere in a normal dependency closure;
- [x] add an explicit candidate inspection/unlink/discard lifecycle;
- [x] make exact binary content inspection safe by default;
- [x] document native seal REF ownership and the standalone Git low-level
  conformance boundary.

The accepted publication and draft-closure contracts are recorded in
[ADR 0007](../adr/0007-linearized-publication-and-draft-closure.md). The review
analysis and remaining work are recorded in
[`external-spec-review-2026-08-14.md`](external-spec-review-2026-08-14.md).
The candidate lifecycle and safe-output CLI choices are analyzed in
[`candidate-lifecycle-proposal-2026-08-14.md`](candidate-lifecycle-proposal-2026-08-14.md);
the accepted contract is recorded in
[ADR 0008](../adr/0008-candidate-lifecycle-and-safe-output.md) and its focused
dogfood receipt is
[`2026-08-14-candidate-lifecycle.md`](dogfooding-receipts/2026-08-14-candidate-lifecycle.md).

## Phase 3 — attachments

- attachment blob import
- attachment metadata hashing
- attachment CLI integration with the existing semantic diff model

## Phase 4 — integrity/forensics

- fsck
- ref compare-and-swap
- corruption tests
- low-level Git-compatible object inspection validation

## Standalone alpha preparation

Prepare `v0.1.0-alpha.1` as an explicitly experimental standalone-only preview
after the usability, tag-collision, read-only `fsck`, and recurring-dogfood
blockers in [`release-checklist.md`](release-checklist.md) are satisfied. The
first artifact scope is Linux amd64 and excludes the unimplemented
`git-sealgraph` executable. Preparation does not authorize a tag or GitHub
Release; publication requires a separately approved exact-SHA gate.

The alpha does not reach the plan's `GIT` or `READY` milestones. Cross-command
JSON, link-message ergonomics, attachments, and Git sidecar may remain open as
listed in the checklist.

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
