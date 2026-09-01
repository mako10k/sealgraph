# Format-5 tracked dogfood migration review — 2026-09-01

Status: `READY_FOR_OWNER_DECISION`. This record preserves self-review and three-
scope independent review of the exact proposal. It does not authorize migration,
commit, push, PR, merge, tag, release, or publication.

## Exact review subject

```text
proposal  docs/process/format5-dogfood-migration-proposal-2026-09-01.md
proposal sha256  1b02d9f39c3102b8510ae76fa19655a70e93264a4f2f73fb9b2611030afe920b
dump      /tmp/sealgraph-dogfood-format5-proposal.7E9DIU/dogfood.fixed.json
dump sha256      9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048
binary sha256    80779c759f41247f9f5e134ab22273f2e29f190f3753a5bfc8e55f43927f2c5c
receipt sha256   739d68ed690d8f28f0eab1282835f20818826e8fe41f265eb1cf45491241fd19
fsck sha256      358646536024f3ea6295cfcdcd1f92163b404662e32420edd9837d3e9160faa5
```

Every final reviewer hashed the proposal and dump before and after review and
reported the expected values.

## Primary-agent self-review

Before the first independent review, self-review found and corrected:

- a normal Go Build ID made whole-binary SHA-256 depend on output path; the
  reviewed binary uses an empty Build ID and was reproduced at two paths;
- external-consumer absence could not be inferred from repository-local search;
- plain `mv` admitted accidental nesting/overwrite semantics;
- dump, warning, load receipt, recovered receipt, fsck, status, graph, all Seal/
  REF/tag mappings, and the exact shell syntax required independent checks; and
- failure must stop without inferred retry, restore, delete, or downgrade.

The self-review ran the exact extractor twice, two isolated loads, receipt
recovery, source pre/post checks, mapping/projection assertions, `bash -n`,
`git diff --check`, `gofmt -l`, `go vet ./...`, `go test ./...`,
`go test -race ./...`, `npm ci`, `npm run clone-check`,
`make complexity-check`, `make deadcode-check`, `perttool document check
PLAN.pert`, and `perttool dag analyze PLAN.pert --schedule both`. All final
invocations passed.

That review was still incomplete. The first independent round found material
issues that should have been caught before delegation: the retained path was not
an executable `<workdir>/.sealgraph`, cutover was not rebound to a final source
observation, external inventory did not satisfy ADR 0026, and the proposed
checkout move still conflicted with the accepted no-rename source boundary.

## Corrective review history

| Reviewed proposal SHA-256 | Result | Mechanisms corrected before next round |
| --- | --- | --- |
| `6f562b5497f5a305913e66a296727f3550732eb845fa29b2a9863668640b0acd` | FAIL | retained layout; cutover observation window; external inventory gate; rename-only/device boundary; audit/final durability claims; ambient Go/Git writes; filesystem versus release publication wording; dump versus receipt collapse attribution |
| `2225287bce471e941e5aea26480fb70f1def2d2e7e2f632a0d6a0e387fcc23a9` | FAIL | source-retention no-rename boundary; all live remote heads; beta.1–beta.6 occurrence definition; Wiki classification; external-actor failure state; final-namespace durability wording |
| `1b02d9f39c3102b8510ae76fa19655a70e93264a4f2f73fb9b2611030afe920b` | PASS | no P0–P3 finding in semantic/ADR, execution-safety, or audit-completeness scope |

The final candidate establishes a separate physical-snapshot/dump-identical
format-4 source copy without renaming, editing, or marking it. Only the tracked-
checkout copy is relocated. The final install remains gated by another exact
extract from the formal retained source.

## AntiPattern analysis and next pre-delegation gate

Evidence: the earlier format-4 dogfood receipt was inspected and its move-based
cutover influenced the first procedure, while accepted ADR 0026 introduced a
stricter no-rename retained-source contract. The first self-review concentrated
on canonical artifact identity and happy-path validation; it did not first
enumerate every path's role and every transition/failure state. It also counted
Git grep hit lines while labeling the value as textual occurrences.

Classification:

- `AP-006 Procedure Reinvention` applies as partial-fit reuse: a prior successful
  cutover invariant was reused without separating the parts superseded by the
  current ADR.
- `AP-005 Helpful Scope Expansion` applies to the discarded draft that allowed
  ordinary migration approval to accept an uncompleted external-inventory
  follow-up; that would have broadened approval into an implicit ADR waiver.
- `AP-008 Escape Cause Substitution` does not describe the correction. Excessive
  independent findings were the escape signal, not the source cause; merely
  adding more review would not correct the procedure.

Replacement: before freezing another real-repository migration procedure for
independent review, the primary self-review must first complete one bounded
pre-delegation table covering:

1. every normative MUST and release/follow-up gate in the accepted decision;
2. every source, retained, staged, audit, cutover, and final path's exact shape,
   owner, permitted writes, and direct runtime selectability;
3. precondition, successful postcondition, and every partial/failure state for
   each state-changing command;
4. all configured/documented external consumers, classified as active rewrite,
   historical retention, or no reference; and
5. the exact unit for each count or digest claim, such as files, lines, fields,
   or textual occurrences.

This is a migration-procedure gate, not a new universal review layer. Each item
maps to an observed failure path above; ordinary documentation changes do not
inherit it.

Verification: the final three reviewers independently reproduced the complete
mapping/projection evidence, all five remote heads, all six beta tags with 20
textual occurrences each, GitHub tracker/comment/Wiki classification, source-
copy/cutover command semantics, and final failure/durability boundaries. No
persistent guidance or AntiPattern catalog was changed, so no catalog scenario
or structural-validator update was required.

## Final independent verdicts

| Scope | Verdict | P0–P3 findings |
| --- | --- | --- |
| Semantic and accepted-ADR consistency | PASS | none |
| Execution safety and failure boundary | PASS | none |
| Audit evidence and inventory completeness | PASS | none |

Residual risks remain explicit rather than resolved by review:

- the maintenance window is cooperative, not a system-wide filesystem lock;
- extractor double capture does not disprove an intermediate change-and-restore;
- audit redirects and the two-move final cutover make no final-namespace crash-
  durability claim;
- unconfigured or undocumented external consumers may exist; discovery before
  execution invalidates the candidate and requires fresh classification/review;
  and
- GitHub and remote state can change after review and must be refreshed at the
  later synchronization/release gates.

## Owner decision boundary

The exact candidate is ready for an Operator accept/reject decision. Acceptance
must bind to both SHA-256 values in the Exact review subject and the eight
decision items in the proposal, including the exclusive maintenance window. It
authorizes only the proposal's local migration sequence. Commit, push, PR,
merge, tag, release, and external artifact publication remain separate actions.
