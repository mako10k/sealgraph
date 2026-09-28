# Sealgraph implementation plan

Status: the format-6 delivery history below is complete. Issue #17 R1–R3 has
separate internal verification in [`issue-17-origin-trace.pert`](issue-17-origin-trace.pert);
R4-c1 has accepted design and internally verified implementation, while Issue
#17 integration and release remain open. `PLAN.pert`
projects the earlier format-6 delivery only. Neither completed PERT projection
asserts that Issue #17 is merged, released, or accepted for use. Attachment
mutation and the separately gated Git sidecar remain future work.

## Issue #17 planning

The [Issue #17 WIP handoff](issue-17-wip-handoff-2026-09-18.md) and its linked
Accepted P3 plan describe the current R1–R3 implementation and verification
boundary. [R4-c1](issue-17-direct-substring-r4-acceptance-2026-09-18.md) adds
an Accepted direct-substring Trace authoring requirement. Its exact bytes now
have a [requirement Seal](issue-17-direct-substring-r4-seal-2026-09-28.md) with
Cause to Accepted R3. [ADR 0042](../adr/0042-direct-substring-trace-authoring.md)
has an [owner acceptance record](issue-17-r4-adr-0042-acceptance-2026-09-28.md),
and its [implementation](issue-17-r4-c1-direct-trace-implementation-2026-09-28.md)
and [verification](issue-17-r4-c1-direct-trace-verification-2026-09-28.md)
are recorded with non-draft Seals. The ADR's `Proposed` label is its adopted
pre-acceptance snapshot; the acceptance record establishes its current status.
The [earlier independent review](issue-17-r4-adr-0042-independent-review-2026-09-28.md)
covers only the earlier candidate, before inline `--content` was added.
The [earlier P1](issue-17-origin-trace-implementation-plan-2026-09-17.md)
and [P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) plans are
historical, not current execution guides.

**Issue #17 goal (IG-17):** preserve Accepted R4-c1 as governing authority and
integrate the reviewed R1–R4 increment against its exact accepted scope.
R1–R3 and R4-c1 have separate internal verification; the original GitHub Issue
also describes broader unresolved fragment behavior, so integration scope and
any issue-wide completion claim require an explicit review. A
worktree or release-scope decision does not revoke or reclassify the Accepted
requirement; a narrower release cannot be called complete Issue #17. The next
public release identity and publication remain separate owner decisions.

## Local worktree minimization and integration plan — 2026-09-28

**Owner goal (WM-0):** retain only worktrees needed for actual concurrent
development in this repository. There is no fixed target count. Merge work
that meets its governing acceptance and review conditions; preserve work that
must stay separate on a remote branch before removing its worktree. A branch
may remain without a worktree. This plan records decision gates and does not
authorize a merge, push, branch deletion, or release.

**Branch disposition (pre-integration snapshot on 2026-09-28; refresh before execution):**

| Local branch | Current relation | Planned disposition |
| --- | --- | --- |
| `main` (`c23aa34`) | Matches remote; integration target | Keep. |
| `codex/issue-17-origin-trace-r2-wip` (`c83fe90`) | Seven commits ahead of remote `9c58308`; primary checkout has this plan edit and Accepted ADR 0042 with R4-c1 implementation evidence | Keep as the IG-17 development line. Reconcile this plan, review the exact R1–R4 integration scope, and push the active branch for review. Merge claims follow the reviewed scope. |
| `plan/normalize-recovery-completion` (`b116644`) | Ancestor of `main`; matches remote | No merge or new push. Remove redundant local pointer after final ancestry and remote readback; retain remote history. |
| `release/v0.1.0-beta.5` (`a18c2f9`) | Ancestor of `main`; remote branch and published tag exist | No merge or new push. Remove redundant local pointer after tag, ancestry, and remote readback; retain tag and remote history. |
| `release/v0.1.0-beta.6` (`2143304`) | Ancestor of `main`; matches remote and published tag exists | No merge or new push. Remove redundant local pointer after tag, ancestry, and remote readback; retain tag and remote history. |

The only remaining local worktree is `/home/katsumata-m/sealgraph` on IG-17.
The performance experiment stash was dropped at the owner's direction; its
idle managed worktree was archived and its redundant local branch deleted.
The clean Candidate comparison worktree and local branch were also removed
after GitHub readback confirmed the remote branch at `17cfa7a`. That separate
requirement packet remains unmerged on the remote branch. The primary checkout
is the active Issue #17 workspace. Accepted R4-c1, published tags, and remote
branches remain in scope for their separate decisions.

1. **WM-1 — Reconfirm current fit.** Before cleanup, read back every worktree's
   branch, exact HEAD, dirty state, stash, remote SHA, and whether a live task or
   process still uses it. Identify unique work and its owner. Exit: each
   worktree is classified as active parallel development, merge candidate, or
   preservation-only, with no unaccounted changes. This inventory is a
   prerequisite to the remaining steps, not permission to remove anything.
2. **WM-2 — Decide merge eligibility.** The committed status-performance change
   is already in the Issue #17 branch; no second merge is needed. The separate
   Candidate-comparison R4 packet needs its own requirement lifecycle decision
   before any normative integration. IG-17 has completed R4-c1 internal
   verification; compare the accepted R1–R4 increment with the broader Issue
   body before calling Issue #17 complete. Review the exact candidate
   SHA and validate the integration scope. Merge eligible work through the
   chosen review route only after its acceptance boundary and validation are
   satisfied. Exit: each proposed merge has an explicit scope, reviewed source
   SHA, target, and readback; unresolved work remains separate.
3. **WM-3 — Preserve independent work before closing a worktree.** The
   Candidate comparison branch's exact remote SHA was verified before its
   worktree was removed. Verify the primary branch's exact remote continuation
   after each integration-ready push. Exit: every
   worktree proposed for removal has a verified remote continuation for all
   unique work, or an explicit owner disposition of material not retained.
4. **WM-4 — Remove only idle worktrees, then verify.** For each worktree, apply
   WM-1 and its own WM-3 preservation gate; WM-2 can continue independently.
   The performance and Candidate comparison worktrees are closed. Keep a
   worktree when real concurrent development needs it; reassess it when that
   work stops. Read back `git worktree list`, branch/stash state, and remote
   continuation SHAs. WM-0 is met: the primary checkout is the only remaining
   worktree, and Candidate comparison's unique commits remain on its remote
   branch. Historical branch cleanup is a separate local-pointer decision.

The applicable existing PERTs stop at format-6 delivery and Issue #17 R1–R3
internal verification respectively. This operational goal is mapped here to
those existing boundaries; neither PERT's zero remaining tasks closes WM-0.
Effort and calendar duration for WM-2 and the remaining IG-17 continuation
depend on integration-scope review and the selected PR/merge route. The next
integration checkpoint is an exact R1–R4 scope comparison and branch review;
historical local-pointer cleanup follows its own ancestry and remote-ref check.
External review and
publication waits remain separate. The next release version, tag, artifacts,
real-repository migration, and GitHub publication each require their own
frozen scope and authorization.

## Current format-6 Link metadata frontier

Proceed in this order; do not partially mix formats:

1. [x] deliver and synchronize the accepted format-5 Cause-scoped revision,
   Universal Blob, migration, CLI, inspection, and tracked-dogfood baseline;
2. [x] accept the extensible Cause Link metadata requirement and ADRs 0029,
   0030, and 0031;
3. [x] implement Seal v6, Provenance v2, Candidate v6, canonical bounded
   metadata, exact historical v5 reading, v6-only writes, change identity, and
   the config-only migration primitive;
4. [x] implement one-namespace Candidate mutation, legacy authoring
   preservation, successor human/JSON inspection, comparison, typed `linklog`,
   and the exact public migration command and receipt; and
5. [x] synchronize requirements, architecture, storage format, CLI,
   integrations, help, completion, fixtures, and full validation.

The implementation, accepted normative synchronization, public navigation, and
full validation are complete. Migrating the project repository, releasing,
deploying, or starting the Git sidecar remains separately authorized. The
format-4 and earlier sections below are historical implementation context, not
the current runtime frontier.

## Historical implementation phases

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

## Attachment expansion — intentionally not planned

ADR 0021 retires the planned format-4 attachment mutation phase. The runtime
continues to read, validate, preserve, compare, and migrate existing
attachment-bearing state, but it does not add `attach` or `detach` commands.
New workflows use primary content, independently sealed related content plus
exact Links, explicit manifests, and non-canonical local source bindings
according to their distinct semantics.

Removing the canonical field is deferred to a separately accepted
Content/Link/Context semantic-format decision with representative projection
evidence. No format-4 attachment is converted automatically.

## Phase 4 — integrity/forensics

- fsck
- ref compare-and-swap
- corruption tests
- low-level Git-compatible object inspection validation

## Standalone beta preparation

Prepare `v0.1.0-beta.1` as an explicitly prerelease standalone-only preview
after the usability, tag-collision, read-only `fsck`, and recurring-dogfood
blockers in [`release-checklist.md`](release-checklist.md) are satisfied. The
first artifact scope is Linux amd64 and excludes the unimplemented
`git-sealgraph` executable. Preparation does not authorize a tag or GitHub
Release; publication requires a separately approved exact-SHA gate.

The beta reaches `READY` without passing through `GIT`. Cross-command JSON and
link-message contracts are complete; attachment mutation is intentionally not
planned, while file synchronization and Git sidecar remain outside the beta as
listed in the checklist.

## Separately gated future — Git sidecar

- select a maintained Git SDK only after the native format-4 boundary is stable;
- expose commit-tree, index, and merge-stage views of the exact native
  `.sealgraph/` paths and bytes through a read-only Git view adapter;
- add `git sealgraph init/status` without defining a second Seal schema or
  repository mode;
- validate staged/commit views against the same native reader and domain
  invariants;
- keep hooks validation-only and make merge/index-stage inspection explicit;
- defer importing arbitrary Git worktree content as Sealgraph material until a
  separate source-import contract is accepted.

Passing the standalone beta does not adopt or start sidecar implementation.
That work requires a separate product decision after the beta gate.

## Phase 6 — Git conflict assistant

- `git sealgraph conflicts`
- three-way BASE/OURS/THEIRS semantic display
- explicit ours/theirs resolution
- post-resolution stale/impact reporting
- no automatic semantic seal creation
