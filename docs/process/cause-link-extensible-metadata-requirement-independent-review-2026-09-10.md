# Cause Link extensible metadata requirement independent review — 2026-09-10

- Status: `completed`
- Review outcome: acceptance blocked pending clarification of acceptance criterion 5
- Owner-selected route: `REVIEW`
- Candidate revision: 1
- Candidate path:
  `docs/process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
- Candidate SHA-256 before and after review:
  `a419872a4971db1d8fcaebe797fde085d9a9ef107b15f93f72eb600138e5cc96`
- Review-input path:
  `docs/process/cause-link-extensible-metadata-requirement-review-input-2026-09-10.md`
- Review-input SHA-256 before and after review:
  `6bb9e8243cb60b1e3a8412912baff1b5736cc7879abb26bc7df76ec480bcb178`
- Mutation boundary: the independent reviewer made no file edits and the
  candidate remained byte-identical throughout review

This review does not accept the candidate or authorize an ADR, normative
conversion, format change, implementation, migration, commit, push, release, or
deployment.

## Result

The independent review completed and found one acceptance-blocking evidence gap.
The generic model otherwise reconciles with the examined accepted authority.
Because the blocking wording has two materially different readings, `ACCEPT` is
not an available next route for this candidate revision. The owner may select
`REVISE`, or may select `REREVIEW` only with an added or changed review question.

## Findings

### F-01 — `evidence gap/unknown` — blocks acceptance

Acceptance criterion 5 requires structural Cause/Revision behavior to be
“byte-for-byte independent of metadata meaning”
([candidate lines 221-222](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L221-L222)).
That phrase permits two materially different readings:

1. Graph-semantic reading: edge membership, staleness, admission, history, and
   related structural results never interpret namespace meaning. This is
   coherent with the candidate's rule that metadata creates no edge and cannot
   weaken structural validation
   ([candidate lines 130-137](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L130-L137))
   and with ADR 0023's exact edge derivation
   ([ADR 0023 lines 101-125](../adr/0023-cause-scoped-revision-links.md#L101-L125)).
2. Output-byte reading: structural inspection or result bytes remain identical
   when metadata bytes change. This cannot hold because metadata changes
   ProvenanceID and SealID
   ([candidate lines 130-131](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L130-L131));
   assertion-source output contains observer Seal and Provenance identities
   ([ADR 0023 lines 61-70](../adr/0023-cause-scoped-revision-links.md#L61-L70),
   [ADR 0027 lines 151-166](../adr/0027-format5-cli-authoring-and-inspection-schemas.md#L151-L166));
   and the candidate requires comparison and `linklog` to expose metadata
   changes
   ([candidate lines 176-188](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L176-L188)).

The narrower graph-semantic reading is internally consistent, so this is not a
contradiction. The criterion is nevertheless not unambiguous or testable as
written. Revision must state the intended invariant without implying unchanged
identities or output bytes.

### F-02 — `evidence gap/unknown` — successor compatibility gate only

The candidate correctly requires a successor decision to explain how unchanged
existing Seals and IDs remain addressable, while leaving mixed historical-schema
reading versus mandatory conversion open
([candidate lines 191-209](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L191-L209),
[candidate lines 232-245](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L232-L245)).
Current format-5 authority uses an isolated migration boundary rather than a
general dual reader
([requirements lines 448-473](../requirements.md#L448-L473)), and public meaning
changes require successor schemas
([ADR 0027 lines 41-70](../adr/0027-format5-cli-authoring-and-inspection-schemas.md#L41-L70)).
The later decision must define whether “addressable” means native typed decoding,
an old-to-new mapping, or retained migration evidence. This does not block the
generic requirement once F-01 is corrected.

### F-03 — `evidence gap/unknown` — successor namespace/storage gate only

The candidate does not yet fix exact non-empty, validity, or reservation rules
for `namespace` and `schema`
([candidate lines 117-128](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L117-L128),
[candidate lines 232-245](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L232-L245)).
The non-normative proposal records one possible policy
([proposal lines 91-120](../proposals/link-metadata-and-sealgraphql.md#L91-L120)),
but the candidate explicitly does not adopt that policy
([candidate lines 50-56](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L50-L56)).
This is a deliberately unresolved successor choice, not a contradiction and not
an independent blocker to the generic requirement.

### F-04 — `optional/future` — does not block acceptance

Project ownership of the revision-context profile, its final namespace and
schema, validator support, exact CLI/query spelling, and query language remain
later choices
([candidate lines 232-245](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L232-L245)).
The proposal likewise separates generic metadata from later validator and query
decisions
([proposal lines 180-200](../proposals/link-metadata-and-sealgraphql.md#L180-L200)).

### F-05 — `out of scope` — does not block acceptance

First-class Cause Link identity, query language, schema registry or network
retrieval, signatures, trusted actor/time, and secret storage are expressly
excluded
([candidate lines 67-79](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L67-L79)).
No examined accepted source requires them for this requirement. ADR 0028 rejects
independently active Links for Sealgraph
([ADR 0028 lines 721-725](../adr/0028-upstream-impact-assessment.md#L721-L725)).

## Material coverage that passed

- The candidate preserves the existing three fields and keeps `messages` a
  sorted, duplicate-free, identity-bearing set, consistent with the current
  requirement
  ([requirements lines 80-97](../requirements.md#L80-L97)).
- Cause Links remain embedded in Provenance without CauseLinkID. Metadata is
  committed transitively through ProvenanceID and SealID without creating a new
  graph edge, consistent with ADR 0025's identity boundary
  ([ADR 0025 lines 188-214](../adr/0025-universal-blob-seal-material-and-provenance.md#L188-L214)).
- The candidate requires a successor format and CLI decision rather than adding
  an unknown format-5 field; current format-5 typed Blobs reject unknown members
  ([storage format lines 103-134](../storage-format.md#L103-L134)).
- Revision-context values remain namespace-owner claims and cannot create core
  preference, truth, trust, approval, or supersession, consistent with the
  normative revision boundary
  ([requirements lines 62-70](../requirements.md#L62-L70)).
- Generic metadata cannot substitute for ADR 0028 Assessment identity, adoption,
  or admission. The inherited later-design consequence is that any exact Cause
  Link metadata change changes ADR 0028's `change_id` and makes assessments for
  the old change inapplicable
  ([ADR 0028 lines 174-197](../adr/0028-upstream-impact-assessment.md#L174-L197)).
- Compatibility text separates preservation of message values from old-reader
  and SealID compatibility and forbids inferred metadata from legacy messages
  ([candidate lines 191-209](cause-link-extensible-metadata-requirement-candidate-2026-09-10.md#L191-L209)).
- ADR 0017 remains proposed and was used only as non-normative comparison
  ([ADR 0017 lines 1-42](../adr/0017-generic-link-metadata-and-query-boundary.md#L1-L42)).

## Review coverage and verification

The independent reviewer examined the exact candidate and review input, Accepted
ADRs 0023, 0025, 0027, and 0028, Proposed ADR 0017 and its detailed proposal, and
relevant portions of `requirements.md`, `architecture.md`, `storage-format.md`,
and `cli.md`. The primary agent independently checked the cited primary sources
for the blocking finding and the reported successor gates before recording this
result. Both bound digests were verified before and after review.

## Next owner decision

- `REVISE`: revise the candidate to remove F-01's ambiguity, create a new digest,
  repeat self-review, and return to first owner review.
- `REREVIEW`: keep this candidate revision unchanged but add or change a review
  question, then run a new independent review.
- `ACCEPT`: unavailable for this revision while F-01 remains acceptance-blocking.
