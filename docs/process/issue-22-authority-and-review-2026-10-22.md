# Issue #22 authority, scope and self-review

Date: 2026-10-22 JST
Base: local main `21195bc273c97c751a0f91085c5b6c7482e7ac9a`; initial working tree clean.
Branch: `codex/issue-22-historical-review`.

## Authority and current requirement

The owner-directed request delivered to this chat asks for implementation of
https://github.com/mako10k/sealgraph/issues/22, including requirements, ADR,
CLI, implementation and tests, and explicitly delegates concrete CLI/storage
design. The Issue body was read through the connected GitHub tool on
2026-10-22; its updated_at was `2026-10-22T12:11:05Z`.

This is a new explicit implementation request after the 2026-10-21
proposal-only closeout. That earlier proposal did not authorize implementation.
The current Issue, rather than the existence of the proposal or a green test,
is the governing source for the change.

The accepted behavior is reviewed exact historical non-draft reliance,
independent current-version/stale facts, observable unknown impact, exact IDs,
and individual decisions. Existing immutable objects, one-REF publication,
coherent observation, no generic force, no automatic relink/reseal, and no
batch publication remain. ADR 0043 maps the delegated design to that behavior.
Its Accepted label records delegated design authority, not a claim that the
owner read or approved each newly authored byte.

This document is a self-review and authority mapping, not an independent
requirement review or a new owner-acceptance record. No independent reviewer
was commissioned and no review status is inferred from file existence.

## Disposition of prior authority

- Requirements §4 and ADR 0023 admission: retain default active-leaf admission;
  ADR 0043 adds only explicit record-gated historical publication.
- ADR 0021 / requirements §5: narrowly supersede the absence of attachment
  authoring for the dedicated review record; arbitrary attachment APIs remain
  absent. The record has no independent logical lifecycle.
- Formats 5/6/7 and ADRs 0025/0030/0033: preserve exact typed schemas and old
  bytes/IDs. Add an ordinary contained JSON evidence schema, without Link
  metadata interpretation or repository migration.
- STALE, observed revision activity, draft, frontier and fsck: preserve their
  meanings. A historical review does not make an old target current-clean.

## Acceptance mapping and self-review

| Issue obligation | Implementation/design evidence | Verification |
| --- | --- | --- |
| Decided historical non-draft reliance | ADR0043; `SealReviewed`; dedicated Candidate record | direct, transitive and detached publication fixtures, formats 5/6/7 |
| Latest differences and pending impact remain separate | unchanged stale derivation; `review show` recorded/current fields and limitations | CLI stale readback; changed-head and unreviewed inspection fixtures |
| Exact dependent, Cause, difference, impact and reason | subject without review attachment; complete Cause entries; exact comparison leaves and other heads | content/Cause/attachment/origin/generation identity checks; canonical codec fixtures |
| Unknown dependencies remain unknown | explicit `unreviewed`, persistable but rejected by historical admission | unreviewed set/show/rejection fixtures |
| Preserve non-draft closure requirement for provisional work | draft Cause always rejected | draft closure fixture |
| One operation, one REF; immutable exact identities | existing publication transaction and expected-old Candidate replacement | old-byte equality, fsck, separate default-path rejection, publication race fixture |
| No automatic repair or proof-of-understanding claim | no generated decisions; explicit file and explicit historical flag; help/AGENTS rules | preparation lacks decisions; usage guards and navigational diagnostic fixture |

The tests use isolated temporary repositories; their fixture loops do not
operate on user provenance. Neither kn's three deferred publications nor its
registry, Cause Links, Seals or earlier automatically published history was
modified by this task.

## Known limitations and non-goals

- Reasons are executor claims and can be false. The product does not prove
  understanding, semantic truth, owner approval, or trusted time.
- The complete other-head observation is deliberately conservative: unrelated
  REF/head changes make a record currently inapplicable. The historical judgment
  remains visible and is not rewritten.
- A detached target may have no asserted current revision lineage. Its automatic
  comparison leaf set is empty; the record still binds the complete other-head
  observation and requires the executor's difference, impact and reason.
- `applicable` describes record/version coverage, not a full publication
  preflight. Draft/root restrictions, origin/content validation, writer guard,
  expected-old checks and final observation validation still apply at Seal time.
- Default strict publication does not interpret the attached review as an
  approval token. The new claim gate is explicitly selected. Native fsck checks
  structure and bytes, not truth of ordinary evidence text.
- Release, installation, push, Issue closure and kn adoption/publication are
  outside this task. No completion claim is made for prior provenance debt.

## Planning observation

Read-only PLAN checks succeeded. Velocity observation returned 16 PTHIS-102
incomplete-history warnings. It offered 19 Points / 607.749722 elapsed hours
from five included tasks and 19 Points / two active dates; neither is an active
implementation-effort measure, and neither was adopted as a forecast for this
slice. Original history and velocity were preserved.

An initial standalone milestone preview failed because it was disconnected
from the existing finish. The subsequent atomic request preview (new Issue22
finish and remaining task, 2/3/5 Points) was valid, but apply was rejected before
write by PTGOV-101 requiring the existing plan-assurance owner `user` for
`goal`/`dag` changes. PLAN.pert remains byte-identical to its initial digest
`sha256:13035ece3306a1ecbf310ebba81387b99ef986f19ddfa1d34d3ac290f39f1c54`.
No alternate plan or rewritten history was used to bypass the gate. These
remaining-scope Points were an agent estimate, not recorded actual effort.
The implementation continues under the separately explicit Issue22 request.

The exact observed velocity and rejected mutation diagnostics are retained in
[the planning observation](issue-22-planning-observation-2026-10-22.json).

## Final review boundary

Current subject: the Issue22 working-tree source/document snapshot listed in
`issue-22-working-tree-manifest-2026-10-22.json`, based on the local commit above.
Current question: does this implementation meet Issue22's historical-reliance
contract while preserving the named invariants? The threshold is the Issue's
behavior and focused scenarios plus this repository's required code checks.
No semantic review of kn's actual premises, release adoption, or independent
review authority is part of this self-review.

INSIDE includes a recorded review that can publish a decided historical
non-draft subject while stale remains visible, missing/unreviewed/draft or
changed-version rejection, coherent current-head observation, and immutable
historical readback. The stale-rejection diagnostic must guide decided
historical reliance to individual review rather than require draft.

OUTSIDE includes a release/installation/kn-publication decision, independent
human approval, cadence monitoring, and guarantees that an executor understood
or honestly described the evidence. Those require separate requests/decisions;
they cannot be silently treated as fulfilled by a passing record check.

BOUNDARY_DISPUTE: none outstanding for the delegated product implementation.
The canonical plan's owner-assurance gate remains a separate unfulfilled plan
mutation prerequisite; it was not treated as implementation authority.
