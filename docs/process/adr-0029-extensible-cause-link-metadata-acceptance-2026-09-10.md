# ADR 0029 extensible Cause Link metadata acceptance — 2026-09-10

Status: owner-acceptance receipt for ADR 0029. This receipt records an exact
semantic decision; it does not authorize companion decisions, normative
conversion, implementation, migration, repository synchronization, release, or
deployment.

## Accepted technical snapshot

- Decision owner: Operator
- Owner decision: `ACCEPT`
- Decision date: 2026-09-10
- ADR path: `docs/adr/0029-extensible-cause-link-metadata.md`
- Exact reviewed technical candidate SHA-256:
  `3a4886671a44c7b34f8a09ce14da08b1f9445fd9cd97140b4cdf66e36ec8fd8c`
- Independent-review path:
  `docs/process/adr-0029-extensible-cause-link-metadata-independent-review-2026-09-10.md`
- Independent-review SHA-256:
  `cb92147d1f33c64d0105e0ef97ba91f4f8e490338575a7e275aa374936d6d5b4`
- Independent-review verdict: `PASS`; no P0 through P3 findings

The status, acceptance link, completed-review wording, and follow-up wording
added after acceptance naturally produce a new ADR file digest. They record the
lifecycle transition and do not alter the accepted technical decision bytes
identified above.

## Owner decision

On 2026-09-10, after the complete candidate and complete Japanese review support
were presented and the independent review returned `PASS`, the Operator
explicitly selected `ACCEPT` and instructed continuation.

The Operator accepts the exact reviewed technical candidate identified above as
the semantic and responsibility boundary for extensible Cause Link metadata.
In particular, the accepted decision:

- preserves `target_seal`,
  `previous_revision_seal_of_target_seal`, and `messages` with their existing
  meanings and permits messages-only use;
- adds one optional generic `namespace` / `schema` / `value` metadata collection;
- keeps Cause Links embedded in Provenance without CauseLinkID;
- keeps structural Cause and revision behavior independent of metadata meaning
  while committing metadata through ProvenanceID and SealID;
- treats revision context as an external generic profile example rather than a
  core-owned persisted namespace;
- preserves the separate ADR 0028 Assessment boundary;
- rejects inheritance of ADR 0017's format-4, virtual-namespace, SealGraphQL,
  query, numeric-limit, and implementation decisions; and
- requires separately accepted storage/migration and CLI/output companion ADRs
  before normative conversion or implementation.

## Lifecycle effects

- ADR 0029 moves from `Proposed` to `Accepted`.
- Proposed ADR 0017 moves to `Rejected` and links to accepted ADR 0029.
- The storage/migration companion is the first bounded next design step.
- The CLI/output companion remains separately pending.

## Retained decisions and unknowns

Acceptance does not yet select:

- exact repository, Seal, Provenance, Candidate, dump/load, or output versions;
- canonical metadata bytes, integer range, or resource limits;
- old-object readability, mixed-schema operation, or explicit migration policy;
- exact Candidate mutation grammar or exact-target selection;
- human or versioned machine output schemas;
- comparison or `linklog` successor behavior; or
- exact ADR 0028 change-preimage integration.

General query language, query AST, indexes, schema registry, and validator
runtime remain outside the accepted companion set.

## Authority boundary

This acceptance authorizes the decision lifecycle transition and the requested
continuation into companion-ADR drafting. It does not accept an as-yet unwritten
companion, change normative product documents, authorize implementation or
migration, or authorize commit, push, PR, merge, release, deployment, or another
external write.
