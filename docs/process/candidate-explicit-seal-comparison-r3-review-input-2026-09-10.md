# Candidate explicit Seal comparison R3 — independent-review input

Status: OWNER_ROUTE_RECORDED

Date: 2026-09-10

## Exact subject

- Candidate path: `docs/process/candidate-explicit-seal-comparison-requirement-r3.md`
- Candidate SHA-256: `ef97bf8ea62a9c50500c91700c021f0ae5a061048a8362e87d764d36d355b16a`
- Requirement revision: R3
- First-owner route: `REVIEW`
- Owner-confirmed syntax: `sealgraph candidate compare REF --against SELECTOR`
- Owner-added review questions: none
- Route after independent review: second owner review of the unchanged R3

## Authority and review scope

R3 proposes one explicit Candidate-to-immutable-Seal comparison mode. It
dispositions the publication-baseline-only restrictions in ADR 0023,
requirements, architecture, CLI, and ADR 0027. It preserves the existing
no-option v2 mode, top-level immutable comparison, source comparison,
no-implicit-predecessor rule, read-only boundary, and complete ADR 0023
snapshot discipline.

Test the exact R3 against:

- `docs/requirements.md`;
- `docs/architecture.md`;
- `docs/storage-format.md`;
- `docs/cli.md`;
- `docs/adr/0020-sealgraph-native-operation-vocabulary.md`;
- `docs/adr/0023-cause-scoped-revision-links.md`; and
- `docs/adr/0027-format5-cli-authoring-and-inspection-schemas.md`.

R1, R2, and their reviews are historical lifecycle evidence only. Existing
implementation and tests are non-normative feasibility evidence only.

## In-scope, out-of-scope, and unknowns

Use R3 sections 3 through 6 exactly. Do not add acceptance criteria.
Candidate-to-Candidate, workfile-to-explicit-Seal, generalized operands,
predecessor inference, textual patches, automatic resolution, mutation, and
Git-derived semantics remain out of scope and non-blocking. Their frequency and
priority and a future generalized syntax remain unknown.

## Review questions

1. Does R3 identify and synchronize every accepted publication-baseline-only
   restriction, including architecture, without superseding unrelated
   authority?
2. Does the v2-preservation/v3-explicit split avoid changing existing machine
   output under an old schema identifier?
3. Does R3 preserve complete `O` recapture and equality and conditional
   complete `I_O` recapture and equality without unnecessary inventory scans?
4. Are comparison fields, direction, explicit-target identity, and
   publication-concurrency separation complete and internally consistent?
5. Has any optional future operand pair become an implicit acceptance blocker?

## Reviewer contract

Verify the exact candidate and review-input digests before review and verify
the candidate again immediately before reporting. Do not edit files. Return:

- verdict `PASS`, `PASS_WITH_FINDINGS`, or `FAIL`;
- findings ordered P0 through P3, each classified as contradiction, evidence
  gap or unresolved unknown, optional or future candidate, or out of scope;
- exact primary-source evidence, reasoning, earliest causal phase when known,
  and uncertainty;
- optional suggestions explicitly marked non-normative;
- unresolved questions; and
- concise coverage.

Independent review cannot accept R3 or authorize an ADR, implementation,
commit, push, release, or deployment.

