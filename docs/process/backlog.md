# Sealgraph dogfood backlog

Status: SG-BL-001 through SG-BL-011 were mirrored to public GitHub Issues on
2026-08-14. ADR 0011 now adds a release-blocking format-4 tranche in
`PLAN.pert`; creating or rewriting external Issues remains a separate explicit
tracker operation.

This document preserves the dogfood evidence, design constraints, and proposed
execution order behind the Issues in
[`mako10k/sealgraph`](https://github.com/mako10k/sealgraph). GitHub tracks Issue
state; the `SG-BL-*` identifiers remain stable across planning documents and
external Issues.

| Backlog ID | GitHub Issue |
| --- | --- |
| SG-BL-001 | [#1](https://github.com/mako10k/sealgraph/issues/1) |
| SG-BL-002 | [#2](https://github.com/mako10k/sealgraph/issues/2) |
| SG-BL-003 | [#3](https://github.com/mako10k/sealgraph/issues/3) |
| SG-BL-004 | [#4](https://github.com/mako10k/sealgraph/issues/4) |
| SG-BL-005 | [#5](https://github.com/mako10k/sealgraph/issues/5) |
| SG-BL-006 | [#6](https://github.com/mako10k/sealgraph/issues/6) |
| SG-BL-007 | [#7](https://github.com/mako10k/sealgraph/issues/7) |
| SG-BL-008 | [#8](https://github.com/mako10k/sealgraph/issues/8) |
| SG-BL-009 | [#9](https://github.com/mako10k/sealgraph/issues/9) |
| SG-BL-010 | [#10](https://github.com/mako10k/sealgraph/issues/10) |
| SG-BL-011 | [#11](https://github.com/mako10k/sealgraph/issues/11) |

Priority meanings:

- P1: complete before expanding the product surface to Git sidecar;
- P2: complete before recurring dogfood is treated as routine automation;
- P3: release/forensics backlog that does not block the next usability slice.

Issue IDs are stable local identifiers. Preserve them in commits, PERT tasks,
and any future tracker migration.

## SG-BL-001 — Exact content input from a file or stdin

- Status: completed locally on 2026-08-17; external Issue state unchanged
- Priority: P1
- PERT: `CONTENT_INGEST`
- Implementation: `--content-file PATH|-`, exact-byte ingestion, mutual
  exclusion, binary round-trip, and explicit missing/directory/symlink/FIFO/
  device no-mutation fixtures are complete.

### Dogfood observation

R1 had to place every manifest in a shell variable and pass it through
`--content`. This made exact trailing-LF behavior, quoting, command length, and
argv exposure part of the operator workflow. It is unsuitable for substantial
documents or arbitrary blob bytes.

### Required design

- Add an exact-byte input such as `add REF --content-file PATH` and decide
  whether `--content-file -` means stdin.
- Make it mutually exclusive with `--content`.
- Do not normalize text, line endings, Unicode, or a final LF.
- Do not persist the source path unless a separately specified manifest says
  that the path is semantic content.
- Define safe handling for regular files, symlinks, devices, and binary
  content. Human `show` must not dump unsafe control bytes accidentally.
- Preserve the rule that standalone operation does not inspect `.git`.

### Acceptance

- A fixture containing NUL, CRLF, and a missing final LF receives the expected
  native blob ID and reads back byte-for-byte.
- File and stdin input produce the same ObjectID for the same bytes.
- Conflicting content flags and unsafe input types fail without creating or
  changing a candidate.
- No automatic seal is performed.

## SG-BL-002 — Deterministic multi-file manifest builder

- Status: completed locally on 2026-08-17; external Issue state unchanged
- Priority: P1
- PERT: `CONTENT_INGEST`
- Depends on: SG-BL-001 for direct file-to-candidate flow

### Dogfood observation

The architecture REF committed to nine files. Their path/digest lines, source
identity, ordering, and aggregate digest were assembled manually with shell
commands and copied into the receipt. `--content-file` alone does not remove
this error-prone step.

### Required design

- Provide a deterministic manifest builder that accepts explicit paths and an
  explicit source identity.
- Sort path entries bytewise and state the digest/aggregate algorithm in the
  output.
- Never infer a Git commit, repository root, or `.git` location in standalone
  mode. A Git SHA, when desired, is an explicit caller-supplied value.
- Keep manifest construction separate from approval: output bytes may feed
  `add`, but the command must not link or seal automatically.
- Define duplicate path, missing path, symlink, directory, and path traversal
  behavior.

### Acceptance

- Input order cannot change manifest bytes or their blob ObjectID.
- Rebuilding from unchanged files is deterministic.
- Changing one byte, semantic path, or explicit source identity changes the
  manifest identity.
- The output says that it is a path/digest claim, not a claim that sealgraph
  stored the named files as attachments.

### Resolution

ADR 0014 defines `sealgraph manifest --source SOURCE --file PATH...` and the
canonical `sealgraph/path-manifest/v1` document. Paths are explicit portable
relative semantic/read paths; no Git discovery, glob, recursion, mapping, or
automatic candidate/seal action occurs. Entries are bytewise path-sorted,
each exact file uses SHA-256, and the aggregate hashes the canonical entries
array.

## SG-BL-003 — Implement `log` and `linklog` as seal history, not Git history

- Status: completed; GitHub Issue #3 closed on 2026-08-14
- Priority: P1, highest inspection priority
- PERT: `HISTORY_INSPECTION`
- Implementation: complete in the native history slice with parent
  ownership/cycle validation, upstream filtering, read-only tests, and safe
  presentation. Remote CI passed for
  `46d9ea2cc496a3807f600e8b4a58c2e95891d163` before tracker closure.

### Dogfood observation

R1 had to record seal IDs externally and issue repeated `show REF@SEAL`
commands to prove parent history and old link immutability. The missing history
commands made the receipt longer and made an omitted generation easy to miss.

### Required design

- `log REF` walks the immutable `parent` chain for the selected logical REF,
  newest first.
- It validates parent object integrity, REF ownership, and parent-cycle
  absence. It does not consult Git commits.
- `linklog REF` derives link add/remove/repoint events by comparing adjacent
  seal generations. It should support filtering by upstream REF.
- Document that `linklog` is not a Git reflog. It describes immutable seal
  state changes; it is not an audit log of every mutable REF-file movement.
- Historical outer checkouts may select an older current head without
  rewriting or fabricating the seal parent chain.

### Acceptance

- The R1 storage, implementation, and validation two-generation histories are
  shown without the operator supplying historical seal IDs.
- Link changes identify the exact old and new upstream seal IDs.
- Corrupt parents, foreign-REF parents, and cycles fail with an explicit next
  inspection action and no repair.
- Read-only history commands do not mutate canonical or runtime state.

## SG-BL-004 — Semantic `diff` between exact seal generations

- Status: completed; GitHub Issue #4 closed on 2026-08-14
- Priority: P1
- PERT: `HISTORY_INSPECTION`
- Implementation: complete for current-parent and exact same-REF generation
  comparison, including content identity, attachments, links, root/draft,
  parent, canonical-order invariance, and read-only behavior. Tracker closure
  followed passing remote CI for
  `46d9ea2cc496a3807f600e8b4a58c2e95891d163`.

### Dogfood observation

The historical format-2 storage supersession intentionally kept content and
links unchanged while changing event message, time, parent, and seal ID.
ADR 0009 removes those event fields in format 3. Downstream repair still keeps
content unchanged while changing a concrete link.

### Required design

- Compare two explicit seals, or a current head with its parent.
- Separate changes in content identity, attachments, direct links, root/draft
  state, and parent. Link-message changes remain edge changes.
- Report link add/remove/repoint distinctly; do not call it a Git merge diff.
- Define safe textual/binary content presentation and size limits.

### Acceptance

- Historical R1 storage v1→v2 remains readable in its receipt; native v3 diff
  has no seal event metadata fields.
- R1 implementation v1→v2 reports unchanged content and a storage link
  repoint.
- Input order or canonical link order does not create false changes.
- The command is read-only and works with explicit historical selectors.

## SG-BL-005 — Clarify `CLEAN`, REF, impact, root, and Git boundaries

- Status: complete (ADR 0015 and implementation, 2026-08-17)
- Priority: P1
- PERT: `OPERATOR_CONTRACT`

### Dogfood observation

Several familiar Git terms invite a materially wrong interpretation:

- `CLEAN` currently means no unsealed candidate and no derived stale relation;
  it does not mean that named working files still match a manifest.
- A sealgraph REF is a logical identity with one current seal, not a Git branch
  and not a checkout target.
- `impact REF` reports structural downstream provenance paths even when every
  current head is clean; it is not a list of currently stale work.
- `root` is an explicit provenance boundary, not a trust anchor.
- standalone `init` inside a Git repository does not attach to or inspect Git.
- an outer Git checkout may make loose files writable; immutability is enforced
  by identity/hash validation, not by relying on filesystem mode alone.

### Required design

- Put a compact semantic legend in help and operator documentation.
- Decide whether human `status`, `impact`, and `graph` need explicit headings
  such as `SEALED_STATE` and `STRUCTURAL_IMPACT` without breaking stable status
  labels.
- Ensure error/help text never suggests branch, checkout, reflog, trust, or
  working-tree synchronization semantics that core sealgraph does not have.
- Add tests that standalone help does not imply Git discovery.

### Acceptance

- A new operator can explain why a REF may be `CLEAN` after a tracked file has
  changed but before a new manifest candidate is added.
- Help distinguishes structural impact from stale-only output.
- Documentation explicitly contrasts seal history/link history with Git
  commit/reflog history.

## SG-BL-006 — Versioned machine-readable inspection output

- Status: complete (ADR 0015 and implementation, 2026-08-17)
- Priority: P1
- PERT: `OPERATOR_CONTRACT`
- Depends on: SG-BL-003 and SG-BL-004 for final history/diff shapes

### Dogfood observation

R0/R1 receipts had to parse or copy human `show`, `status`, `graph`, and
`impact` text. Paths were preformatted strings, making automation vulnerable
to whitespace and presentation changes.

### Required design

- Add a versioned format such as `--format json` for `show`, `status`, `stale`,
  `graph`, `impact`, `log`, `linklog`, and `diff`.
- Preserve ADR 0010's already stable `stale --refs-only` one-REF-per-line stream
  as a narrow shell-composition format; JSON remains separate and structured.
- Represent ObjectIDs and graph paths as structured values, not formatted
  strings.
- Preserve orthogonal direct/transitive stale states.
- Version schemas before receipts or external tools rely on them.
- Define exit codes separately from result state; a clean/stale result should
  not be confused with integrity or usage failure.

### Acceptance

- R1 status and impact evidence can be captured without scraping human text.
- Reordering presentation-only fields does not affect consumers.
- JSON never includes secret environment values or implicit filesystem/Git
  discovery results.

## SG-BL-007 — Report runtime bootstrap distinctly from idempotent init

- Status: complete (ADR 0015 and implementation, 2026-08-17)
- Priority: P2, small
- PERT: `OPERATOR_CONTRACT`

### Dogfood observation

After a canonical-only checkout, explicit `sealgraph init` created missing
`index` and `locks` directories but printed "already initialized". The action
was safe and correct, but the message concealed a real local mutation.

### Required design and acceptance

- Distinguish `initialized`, `bootstrapped runtime directories`, and `already
  complete` outcomes.
- Name the created runtime directories without printing sensitive paths or
  environment data.
- Repeated init remains idempotent.
- Object and REF bytes/digests are identical before and after bootstrap.

## SG-BL-008 — Recurring dogfood workflow and self-reference runbook

- Status: open
- Priority: P2
- PERT: `DOGFOOD_RECURRING`
- Depends on: SG-BL-001 through SG-BL-007

### Dogfood observation

R1 required a predecessor source commit, then a separate commit containing the
sealgraph objects and receipt. The manifest's `source_git` therefore names the
predecessor, not the dogfood metadata commit. Treating the final commit as
sealing itself would be a self-referential and misleading claim.

### Required design

- Document and exercise the two-commit workflow explicitly.
- Use explicit source identity and deterministic manifests; do not inspect Git
  from standalone core.
- Automate evidence collection only after versioned machine output exists.
- Never introduce a Git hook, automatic commit→seal, recursive repair, or
  batch seal.
- Each changed REF still requires a reviewed one-REF `add`/`link` plus `seal`.

### Acceptance

- One controlled documentation update is manifested, sealed, inspected with
  history/diff, committed, and reproduced from a fresh checkout.
- The receipt distinguishes source commit, seal metadata commit, and current
  outer checkout.
- No candidate, lock, cache, or log file enters the commit.

## SG-BL-009 — Full `fsck` and checkout-mode integrity explanation

- Status: open
- Priority: P3, required by `RELEASE_GATE`
- PERT: `RELEASE_GATE`

### Dogfood observation

R0 tested one known corrupted object through `show`, while R1 validated only
objects reachable from the five current heads. Outer Git stores only an
executable bit, so checkout did not preserve the original `0444` loose-object
mode. Hash validation still enforced semantic immutability, but there is no
complete repository inventory yet.

### Required design

- Verify every loose object envelope, hash, canonical Seal payload, REF value,
  parent-revision chain, exact Cause target, active/detached state, and both DAG
  cycle classes.
- Report unreachable/dangling immutable objects separately from corruption;
  non-leaf and detached history is expected, not garbage.
- Explain that writable mode does not authorize overwriting a content-addressed
  object and is not the integrity boundary.
- Remain read-only by default. Any future recovery action must be explicit and
  separately specified.

### Acceptance

- Clean R1 state passes from a fresh canonical checkout after runtime
  bootstrap.
- Corrupt object, missing object, bad REF, foreign target, and cycle fixtures
  fail with exact object/REF context.
- `fsck` does not repair, delete, repack, or rewrite canonical state.

## SG-BL-010 — Resolve REF-scoped tag loose-path namespace collisions

- Status: done by ADR 0013 and `TAG_CONTRACT`
- Priority: P2, resolve before tags become routine
- PERT: `TAG_CONTRACT`

### Dogfood observation

The approved `.sealgraph/refs/tags/<REF>/<ENCODED_TAGNAME>` mapping has a
file/directory conflict when a prefix-REF tag leaf equals the next component of
a child REF. `design` tag `api` and tags scoped to `design/api` cannot coexist.
Native v2 rejects either creation order without repair, but tag availability
therefore depends on hierarchical namespace use.

### Acceptance

- Decide explicitly whether to retain the limitation or adopt a deliberately
  breaking collision-free format before 1.0.
- Preserve path-form REFs, immutable UI-scoped tags, injective TAGNAME
  encoding, REF rename, and future Git-sidecar forensic usability.
- Do not add dual lookup, hidden fallback storage, or automatic migration only
  for experimental compatibility.
- Test both creation orders and make any retained error name the conflicting
  scopes and next explicit action.

### Resolution

ADR 0013 adopts `.sealgraph/refs/seals/<REF>/.ref`, one canonical manifest
containing HEAD and sorted immutable tag bindings. The reserved terminal marker
allows prefix REFs to coexist and makes `mv` one atomic no-replace manifest
rename. Candidate and lock files use their own runtime terminal markers. There
is no dual lookup, automatic migration, old-name alias, tag retarget/delete, or
candidate move.

## SG-BL-011 — Make distinct dependency messages atomic and ergonomic

- Status: complete (repeated-command contract retained by ADR 0015, 2026-08-17)
- Priority: P2
- PERT: `OPERATOR_CONTRACT`

### Dogfood observation

ADR 0006 needed two dependencies with different edge rationales. One
`link ... -m MESSAGE` invocation applies the same message to every dependency,
so dogfood used two candidate updates. This is semantically safe but verbose
and not atomic as a two-edge candidate edit.

### Acceptance

- Either define an unambiguous atomic dependency/message-pair syntax or retain
  the repeated-command contract with documented rationale.
- Keep one domain-independent dependency semantic; do not add ADR-specific
  kinds or a free-form kind taxonomy without a separate accepted need.
- Resolve every selector before candidate mutation, preserve one Link per exact
  target SealID, and keep deterministic canonical order.
- Continue to distinguish Link-message changes from target repoints. Format 4
  has no whole-Seal event message.

## Proposed execution order

Before Git sidecar:

1. `FORMAT3_LOGICAL_DUMP`: preserve an explicit export boundary before the
   runtime reader changes.
2. `FORMAT4_NATIVE_CORE` and `FORMAT4_REVISION_GRAPH` are complete without a
   dual reader or owner-relative graph checks.
3. `TAG_CONTRACT` is complete: ADR 0013 selects a rename-safe format-4
   manifest and narrow `mv` transaction without compatibility scaffolding.
4. `FORMAT4_DOGFOOD_LOAD` is complete: tracked provenance was explicitly
   converted and sibling revision behavior was exercised without dropping
   format-3 tags.
5. `CONTENT_INGEST` is complete: SG-BL-001 and SG-BL-002 provide exact
   file/stdin ingestion and deterministic explicit-path manifests.
6. `HISTORY_INSPECTION`: revalidate SG-BL-003 and SG-BL-004 against
   `parent_revision` and exact target SealIDs.
7. `OPERATOR_CONTRACT`: SG-BL-005 through SG-BL-007 and SG-BL-011, using the
   final history and diff data shapes.
8. `DOGFOOD_RECURRING`: SG-BL-008 validates the combined operator workflow.

SG-BL-009 remains in the existing release gate. Git sidecar follows the
recurring dogfood gate so it can reuse one established inspection vocabulary
instead of introducing Git-shaped semantics into an ambiguous core surface.
