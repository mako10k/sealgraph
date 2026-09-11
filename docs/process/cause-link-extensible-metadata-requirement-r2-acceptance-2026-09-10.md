# Cause Link extensible metadata requirement revision 2 acceptance — 2026-09-10

- Status: accepted requirement
- Decision owner: Operator
- Owner decision: `ACCEPT`
- Decision date: 2026-09-10
- Accepted requirement revision: 2
- Accepted candidate path:
  `docs/process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`
- Accepted candidate SHA-256:
  `83acad3c80f8dadd0f49271829e80efa6bbeb3f5d8adb23c30f928f1f7876ac2`
- First-owner route: `REVIEW`
- Review-input path:
  `docs/process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`
- Review-input SHA-256:
  `afa7ce33dcbeaad76aa93cb8c91cdffb6e83ebf2d66b5fd945008147e43e5087`
- Independent-review path:
  `docs/process/cause-link-extensible-metadata-requirement-r2-independent-review-2026-09-10.md`
- Independent-review SHA-256:
  `de76431a8e1a0dccf6cca977b84dd771f61964b63d86500c77b6dd7b93cdf03a`
- Independent-review result: `completed`; no acceptance-blocking
  contradiction or evidence gap

## Decision

The Operator accepts only the exact revision-2 candidate bytes identified above
as the governing requirement for a future Cause Link extensible-metadata design.
The separate acceptance record preserves the reviewed candidate bytes unchanged;
the candidate's historical header is not rewritten after review.

The accepted requirement:

- preserves `target_seal`,
  `previous_revision_seal_of_target_seal`, and `messages` with their existing
  meanings, including continued messages-only use;
- adds one optional, generic, namespaced `namespace` / `schema` / `value`
  metadata mechanism for structured Link-local meaning;
- requires deterministic, bounded, identity-bearing metadata committed through
  ProvenanceID and SealID;
- supports revision-context explanations such as progress, correction, and
  reconciliation without making those values core graph states;
- keeps Cause Links embedded in Provenance and creates no CauseLinkID;
- requires structural Cause/Revision calculations not to interpret metadata
  meaning, while allowing identity and exact output bytes to change when
  identity-bearing metadata changes;
- keeps revision context distinct from seal-operation metadata and ADR 0028
  upstream Assessments; and
- requires explicit successor storage, CLI, compatibility, and migration
  decisions before persisted implementation.

## Review lineage

1. Revision 1 completed self-review and first owner review.
2. The Operator selected `REVIEW`; independent review finding F-01 identified an
   ambiguity between semantic non-interpretation and output-byte invariance.
3. At Step 4 the Operator selected `REVISE`.
4. Revision 2 returned to Step 1 and changed only candidate identity/review
   routing and acceptance criterion 5. Findings F-02 through F-05 were not added
   as requirements.
5. The Operator selected `REVIEW` for revision 2. Independent review completed
   with no acceptance-blocking contradiction or evidence gap and verified the
   candidate and review-input digests before and after review.
6. At Step 4 the Operator selected `ACCEPT` for the exact revision-2 candidate.

## Retained non-blocking decisions

Acceptance does not select:

- a final namespace owner or revision-context schema identifier;
- exact canonical value and resource limits;
- successor repository, Seal, Provenance, Candidate, dump/load, or output schema
  versions;
- mixed historical-schema reading versus mandatory repository conversion;
- exact CLI syntax or initial query scope; or
- whether Sealgraph maintains the revision-context profile or documents only an
  external example.

These remain mandatory successor decisions where the accepted requirement says
they gate design or implementation. Optional and out-of-scope review findings do
not become acceptance criteria through this record.

## Authority boundary

This acceptance establishes requirement authority only for the exact candidate
bytes identified above. It does not itself create or accept an ADR, revise
normative product documents, choose a persisted format or external namespace,
authorize implementation or migration, or authorize commit, push, PR, merge,
release, deployment, or any other external write. Each downstream artifact and
effect retains its own review and authorization boundary.
