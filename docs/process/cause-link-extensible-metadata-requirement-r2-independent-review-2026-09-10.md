# Cause Link extensible metadata requirement revision 2 independent review — 2026-09-10

- Status: `completed`
- Review outcome: no acceptance-blocking contradiction or evidence gap found
- Owner-selected first-review route: `REVIEW`
- Candidate revision: 2
- Candidate path:
  `docs/process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`
- Candidate SHA-256 before and after review:
  `83acad3c80f8dadd0f49271829e80efa6bbeb3f5d8adb23c30f928f1f7876ac2`
- Review-input path:
  `docs/process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`
- Review-input SHA-256 before and after review:
  `afa7ce33dcbeaad76aa93cb8c91cdffb6e83ebf2d66b5fd945008147e43e5087`
- Mutation boundary: the independent reviewer made no file edits and both
  review inputs remained byte-identical throughout review

This review does not accept the candidate or authorize an ADR, normative
conversion, format change, implementation, migration, commit, push, release, or
deployment.

## Result

The independent review completed with no contradiction and no
acceptance-blocking evidence gap. Revision 2 resolves revision 1 finding F-01:
acceptance criterion 5 now separates graph-semantic non-interpretation from the
identity and output-byte changes caused by identity-bearing metadata. The exact
candidate is eligible for the owner's Step 4 decision.

## Classified findings

### R2-F01 — `evidence gap/unknown` — non-blocking

The final namespace owner and schema identifier, numeric canonical/resource
limits, successor persisted and output schema versions, historical reading
versus conversion policy, exact CLI, and maintained-profile ownership remain
undecided
([candidate lines 105-107](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L105-L107),
[candidate lines 123-132](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L123-L132),
[candidate lines 195-213](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L195-L213),
[candidate lines 240-253](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L240-L253)).

These choices gate their respective successor design or implementation boundary,
not acceptance of the generic requirement. The candidate requires deterministic
bounds, explicit compatibility treatment, and a separately accepted persisted
and CLI design. This is consistent with the repository's extension discipline
([architecture lines 345-350](../architecture.md#L345-L350)) and accepted schema
authority
([ADR 0025 lines 343-366](../adr/0025-universal-blob-seal-material-and-provenance.md#L343-L366),
[ADR 0027 lines 594-624](../adr/0027-format5-cli-authoring-and-inspection-schemas.md#L594-L624)).

### R2-F02 — `optional/future` — non-blocking

A general metadata query language, schema validator, and Sealgraph-maintained
revision-context vocabulary remain optional later work
([candidate lines 71-83](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L71-L83),
[candidate lines 249-253](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L249-L253)).
The candidate does not require them for canonical reading or the generic metadata
mechanism. Proposed ADR 0017 and its proposal remain non-normative comparison
inputs
([ADR 0017 lines 31-42](../adr/0017-generic-link-metadata-and-query-boundary.md#L31-L42),
[proposal lines 180-200](../proposals/link-metadata-and-sealgraphql.md#L180-L200)).

### R2-F03 — `out of scope` — non-blocking

First-class Cause Link identity or lifecycle, trusted actor or time, signatures,
secret storage, and substitution for ADR 0028 Assessment remain correctly out of
scope
([candidate lines 71-83](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L71-L83),
[candidate lines 167-178](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L167-L178)).
Accepted requirements forbid revision assertions from implying preference,
truth, trust, or approval
([requirements lines 62-70](../requirements.md#L62-L70)), and ADR 0028 separately
owns Assessment identity and admission while rejecting independently active
Links for Sealgraph
([ADR 0028 lines 721-743](../adr/0028-upstream-impact-assessment.md#L721-L743)).

### `contradiction`

None found.

## Review-question coverage

1. `messages` retains its existing free-form, identity-bearing contract, while
   the single metadata collection is the only new structured extension path
   ([candidate lines 85-119](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L85-L119)).
2. Metadata remains Provenance content committed through ProvenanceID and SealID
   without CauseLinkID
   ([candidate lines 33-48](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L33-L48),
   [ADR 0025 lines 188-214](../adr/0025-universal-blob-seal-material-and-provenance.md#L188-L214)).
3. Criterion 5 now states that structural calculations do not interpret
   namespace, schema, or value, while expressly allowing IDs, assertion-source
   identities, comparisons, and exact output bytes to change
   ([candidate lines 225-230](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L225-L230)).
   ADR 0023 derives structural relations only from exact targets and previous
   SealIDs
   ([ADR 0023 lines 101-134](../adr/0023-cause-scoped-revision-links.md#L101-L134));
   ADR 0025 establishes that changed Provenance identity changes SealID
   ([ADR 0025 lines 188-206](../adr/0025-universal-blob-seal-material-and-provenance.md#L188-L206)).
4. Revision context has no trusted actor/time, AssessmentID, change binding,
   evidence adoption, disposition, or admission effect
   ([candidate lines 167-178](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L167-L178)).
5. Compatibility separates field continuity from old-reader support, migration,
   object readability, and SealID preservation
   ([candidate lines 195-213](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L195-L213)).
6. Proposed ADR 0017 is used only for its separable generic model, and a later
   decision must explicitly dispose of it
   ([candidate lines 54-60](cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md#L54-L60)).
7. Each unresolved namespace, format, CLI, migration, or limit choice is stated
   as a successor gate paired with a constraint the later design must satisfy;
   none makes this requirement internally incoherent or untestable.

## Review coverage and verification

The independent reviewer examined the exact candidate and review input, Accepted
ADRs 0023, 0025, 0027, and 0028, Proposed ADR 0017 and its detailed proposal, and
relevant normative sections of `requirements.md`, `architecture.md`,
`storage-format.md`, and `cli.md`. The primary agent independently checked the
material primary-source claims before recording this result. Both frozen digests
were verified before and after review.

## Next owner decision

- `REVISE`: return to Step 1 for any requirement-text change.
- `REREVIEW`: keep the candidate digest unchanged, add or change a review
  question, and repeat Step 3.
- `ACCEPT`: accept only the exact revision-2 candidate bytes identified above.
