# ADR 0031 format-6 Link metadata CLI and output acceptance — 2026-09-10

Status: owner-acceptance receipt for ADR 0031. This receipt records an exact
CLI and output decision; it does not authorize normative conversion,
implementation, migration, repository synchronization, release, or deployment.

## Accepted technical snapshot

- Decision owner: Operator
- Owner decision: ACCEPT
- Decision date: 2026-09-10
- ADR path: docs/adr/0031-format6-link-metadata-cli-and-output.md
- Exact reviewed technical candidate SHA-256:
  82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47
- Initial independent-review path:
  docs/process/adr-0031-format6-link-metadata-cli-and-output-independent-review-2026-09-10.md
- Initial review verdict: FAIL with one P1 finding
- Focused-rereview path:
  docs/process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-2026-09-10.md
- Focused-rereview SHA-256:
  6dac3b056fa32388521eb4309534409ec42432650e34409c4aa9983599b8625d
- Focused-rereview verdict: PASS; no P0 through P3 findings; prior P1 RESOLVED

The status, acceptance link, completed-review wording, and follow-up wording
added after acceptance naturally produce a new ADR file digest. They record the
lifecycle transition and do not alter the accepted technical decision bytes
identified above.

## Owner decision

On 2026-09-10, after the revised complete candidate and complete Japanese review
support were presented and focused independent rereview returned PASS, the
Operator explicitly selected ACCEPT.

The accepted decision:

- introduces dedicated link-metadata set and remove commands for one exact
  target and namespace in one Candidate;
- preserves messages and all unnamed metadata during metadata-only edits;
- preserves metadata when legacy add or link syntax changes only the structural
  previous-revision and message arrays;
- defines complete format-6 shared machine records, comparison records, and
  successor schema versions;
- makes linklog/v3 carry exact newer and previous Seal/Provenance identities and
  generations so projected format-5 empty metadata is distinguishable from
  stored format-6 empty metadata;
- defines complete escaped human inspection with explicit historical-projection
  labels;
- selects the exact format-5-to-format-6 migration command and receipt while
  prohibiting automatic retry after an ambiguous committed result; and
- keeps query language, schema registry/validation, and ADR 0028 Assessment
  commands outside this decision.

## Retained gates and unknowns

Acceptance does not authorize or complete:

- normative requirements, architecture, storage-format, or CLI conversion;
- implementation and exact code fixtures;
- execution of repository migration;
- ADR 0028 Assessment-reference storage or command work;
- metadata query or validation facilities; or
- commit, push, PR, merge, release, deployment, or publication.

## Authority boundary

This acceptance authorizes only the ADR lifecycle transition and its durable
receipt. Any normative conversion, implementation, migration, synchronization,
or subsequent project task requires separate user authority.
