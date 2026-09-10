# Cause Link extensible metadata requirement review input — 2026-09-10

- Status: independent review requested by the owner
- Candidate revision: 1
- Candidate path:
  `docs/process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
- Candidate SHA-256:
  `a419872a4971db1d8fcaebe797fde085d9a9ef107b15f93f72eb600138e5cc96`
- Owner route after first review: `REVIEW`
- Route selected: 2026-09-10, in the current Sealgraph requirement-review thread

This input is bound only to the exact candidate bytes above. A change to the
candidate requires a new revision, digest, self-review, and first owner review.
This document does not accept the candidate or authorize an ADR, format change,
implementation, migration, commit, push, release, or deployment.

## Source provenance and prior authority

The requirement source is the Operator direction recorded as D-CLM-001 through
D-CLM-005 in the candidate: retain `messages`; make structured revision context
optional; provide one generic extension area; generalize the concrete use case
through that area; and keep Cause Links embedded in Provenance without a
CauseLinkID.

Review the candidate against Accepted ADRs 0023, 0025, 0027, and 0028. Treat
Proposed ADR 0017 and its detailed proposal only as non-normative comparison
material. In particular, do not infer acceptance of ADR 0017's old Link shape,
revision assumptions, query language, namespace reservation, limits, or scope.

## Review scope

In scope:

- preservation of the existing three Cause Link fields and their semantics;
- one namespaced `namespace` / `schema` / `value` metadata mechanism;
- deterministic identity commitment, bounds, and safe inspection;
- the revision-context profile as an optional use of generic metadata;
- separation from structural Cause/Revision behavior and ADR 0028 Assessment;
- Candidate mutation, output, compatibility, and migration requirements; and
- the completeness and testability of the ten acceptance criteria.

Out of scope:

- implementing or accepting a successor format or CLI;
- making Cause Links first-class Blobs;
- designing a query language, schema registry, validator runtime, signatures,
  trusted time/actor, or secret storage;
- selecting final namespace names, version numbers, resource limits, command
  spelling, migration commands, or release sequencing; and
- proposing useful adjacent features as acceptance blockers.

## Acceptance criteria under review

The reviewer must determine whether the candidate requires all of the following
without contradiction:

1. Provenance-embedded Cause Links with no CauseLinkID.
2. Unchanged format-5 field meanings and continued `messages`-only use.
3. One generic extension path capable of the revision-context example.
4. Identity-bearing, deterministic, bounded, safely rendered metadata.
5. Structural graph behavior independent of namespace meaning.
6. No substitution for ADR 0028 Assessment identity or admission effects.
7. Explicit, namespace-isolated Candidate mutations with no implicit publish.
8. Separate human and machine visibility for `messages` and metadata.
9. Precise compatibility claims for fields, readers, migration, and SealIDs.
10. Exact legacy-message preservation without inferred structured meaning.

## Known unknowns

- Final namespace owner and revision-context schema identifier.
- Exact canonical value and resource limits.
- Successor repository and typed-Blob schema versions.
- Mixed historical-schema reading versus mandatory conversion.
- Exact CLI and initial query scope.
- Whether the revision-context profile is project-maintained or only an
  external example.

These are evidence gaps or later owner decisions. The reviewer should report a
contradiction only when leaving one open makes the stated requirement internally
incoherent, incompatible with accepted authority, or untestable. Optional design
preferences must remain optional findings.

## Review questions

1. Does preserving `messages` while adding one metadata collection satisfy the
   owner directions without creating two competing extension mechanisms?
2. Is metadata located and committed correctly through the Provenance Blob and
   SealID without making the Cause Link independently addressed?
3. Can all structural Cause/Revision results remain independent of metadata
   content, including unknown or contradictory namespaces?
4. Is the revision-context profile clearly separated from seal-operation
   metadata and ADR 0028 upstream Assessment?
5. Are the compatibility and migration statements accurate and sufficient to
   prevent claims that old readers or SealIDs remain compatible automatically?
6. Does the candidate reuse only independently valid parts of Proposed ADR 0017
   and leave an explicit later disposition for that proposal?
7. Are any acceptance criteria dependent on an unresolved namespace, schema,
   format, CLI, or limit decision that must instead be fixed before acceptance?

## Required reviewer result

Classify every material result as exactly one of:

- contradiction with the candidate or an authoritative source;
- evidence gap or unresolved unknown;
- optional or future candidate; or
- out of scope.

Return `completed` or `not-reviewable`, cite exact primary-source locations,
preserve uncertainties, and do not edit the candidate. The route after review
must be supplied by the Operator during first owner review before independent
review begins.
