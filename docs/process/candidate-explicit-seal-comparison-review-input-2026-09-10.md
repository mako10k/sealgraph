# Candidate explicit Seal comparison R1 — independent-review input

Status: OWNER_ROUTE_RECORDED

Date: 2026-09-10

## Exact subject

- Candidate path: `docs/process/candidate-explicit-seal-comparison-requirement-r1.md`
- Candidate SHA-256: `75680973023dd7b72140ef1b2d5a9bd5a80aad3dee6d056aa3fe701d43abbdd3`
- Requirement revision: R1
- First-owner route: `REVIEW`
- Owner-added review questions: none
- Route after independent review: second owner review of the unchanged R1

## Source provenance and prior-authority disposition

The owner reported that Seal-to-Seal-only comparison was insufficient and then
instructed the author to proceed with the proposed first increment,
`sealgraph candidate compare REF --against SELECTOR`. R1 proposes to supersede
only the publication-baseline-only restriction for this new explicit mode. It
preserves the existing no-option Candidate comparison, top-level immutable
comparison, source comparison, no-implicit-predecessor rule, and read-only
boundary.

The accepted sources to test are:

- `docs/requirements.md`;
- `docs/architecture.md`;
- `docs/storage-format.md`;
- `docs/cli.md`;
- `docs/adr/0020-sealgraph-native-operation-vocabulary.md`;
- `docs/adr/0023-cause-scoped-revision-links.md`; and
- `docs/adr/0027-format5-cli-authoring-and-inspection-schemas.md`.

Existing implementation and tests are non-normative feasibility and
compatibility evidence only.

## In-scope and out-of-scope behavior

In scope is one explicit Candidate-to-immutable-Seal comparison mode, with a
new v3 JSON schema, complete identity-bearing comparison fields, preserved
publication-concurrency context, coherent observation, and head revalidation.
The no-option mode and its v2 schema remain unchanged.

Candidate-to-Candidate, workfile-to-explicit-Seal, generalized cross-kind
operands, predecessor inference, textual patches, automatic conflict
resolution, mutation, and Git-derived semantics are out of scope and must not
become blockers.

## Acceptance criteria and unknowns

Use the eight acceptance criteria in R1 section 6 without adding criteria.
The frequency and priority of other cross-kind comparisons are unknown. A
future generalized syntax is also unknown and non-blocking.

## Review questions

1. Does R1 contradict an accepted invariant outside the restrictions it
   explicitly proposes to supersede?
2. Does the v2-preservation/v3-explicit split avoid changing existing machine
   output under an old schema identifier?
3. Are snapshot and revalidation requirements sufficient to prevent a mixed or
   falsely authoritative comparison observation?
4. Are the in-scope comparison fields, direction, and publication-concurrency
   separation complete and internally consistent?
5. Has any optional future operand pair become an implicit acceptance blocker?

## Reviewer contract

Verify the exact candidate digest before review and again before reporting.
Do not edit the candidate or this input. Return:

- verdict: `PASS`, `PASS_WITH_FINDINGS`, or `FAIL`;
- findings ordered P0 through P3;
- classification of each finding as contradiction, evidence gap or unresolved
  unknown, optional or future candidate, or out of scope;
- exact primary-source evidence, reasoning, earliest causal phase when known,
  and uncertainty for each finding;
- optional corrective suggestions marked non-normative;
- unresolved questions; and
- a concise coverage statement.

A review finding cannot create a requirement. Independent review cannot accept
R1 or authorize an ADR, implementation, commit, push, release, or deployment.

