# ADR 0030 format-6 Link metadata storage and migration acceptance — 2026-09-10

Status: owner-acceptance receipt for ADR 0030. This receipt records an exact
storage and migration decision; it does not authorize normative conversion,
implementation, migration, repository synchronization, release, or deployment.

## Accepted technical snapshot

- Decision owner: Operator
- Owner decision: `ACCEPT`
- Decision date: 2026-09-10
- ADR path: `docs/adr/0030-format6-link-metadata-storage-and-migration.md`
- Exact reviewed technical candidate SHA-256:
  `b879af866be94106c8a7675a1477e9ccfabfdc823058c4b3bb7157aa757f198e`
- Independent-review path:
  `docs/process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-2026-09-10.md`
- Independent-review SHA-256:
  `720cfae1198455366c49e897568f931a4da05b153afa54c0f70c8a1dfdd03ed8`
- Independent-review verdict: `PASS`; no P0 through P3 findings

The status, acceptance link, completed-review wording, and follow-up wording
added after acceptance naturally produce a new ADR file digest. They record the
lifecycle transition and do not alter the accepted technical decision bytes
identified above.

## Owner decision

On 2026-09-10, after the complete candidate and complete Japanese review support
were presented and independent review returned `PASS`, the Operator explicitly
selected `ACCEPT` and instructed continuation.

The accepted decision:

- selects repository format 6, Seal v6, Provenance v2, and Candidate v6;
- retains Material v1, Blob identity, REF manifests, and graph meaning;
- preserves exact historical Seal v5, Provenance v1, Candidate v5, messages,
  IDs, REFs, and tags;
- permits exact read-only v5 compatibility inside format 6 while writing only
  v6 successor objects;
- fixes canonical metadata bytes and resource limits;
- selects an explicit config-only format-5-to-format-6 migration;
- selects `sealgraph/upstream-change/v2` for metadata-bearing assessment-free
  change identity; and
- keeps Assessment-reference storage and CLI/output contracts separately gated.

## Retained decisions and unknowns

Acceptance does not yet select:

- exact migration CLI spelling;
- metadata mutation grammar and input source;
- human and versioned machine inspection schemas;
- comparison and `linklog` successor schemas;
- Assessment-reference persistence; or
- implementation or normative-conversion sequencing.

## Authority boundary

This acceptance authorizes the ADR lifecycle transition and the requested
continuation into CLI/output companion drafting. It does not accept that
as-yet-unwritten companion, change normative product documents, authorize
implementation or migration, or authorize commit, push, PR, merge, release,
deployment, or another external write.
