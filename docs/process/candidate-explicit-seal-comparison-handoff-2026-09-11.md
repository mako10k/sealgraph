# Candidate explicit Seal comparison — preservation handoff

Status: WIP_PENDING_R4_FIRST_OWNER_ROUTE

Date: 2026-09-11

## Purpose

This non-normative handoff records the end-of-day lifecycle and Git state of the
Candidate explicit Seal comparison requirement packet. Preservation does not
advance requirement authority.

## Current lifecycle state

- R1 received an independent-review verdict of `FAIL`; the second owner selected
  `REVISE`. R1 remains historical evidence.
- R2 received an independent-review verdict of `FAIL`; the second owner selected
  `REVISE`. R2 remains historical evidence.
- R3 received an independent-review verdict of `PASS` over unchanged bytes but
  did not receive a second-owner disposition and was never accepted.
- Accepted format-6 work subsequently assigned
  `sealgraph/candidate-compare/v3` to the existing format-6 publication-baseline
  comparison. That made R3's proposed reuse of v3 for explicit comparison
  stale.
- On 2026-09-11, the owner authorized a suitable requirement change. R4 was
  authored and completed lifecycle step 1 self-review. It has not received its
  first-owner route, independent review, second-owner decision, or acceptance.

The exact R3 subject is
`docs/process/candidate-explicit-seal-comparison-requirement-r3.md` at SHA-256
`ef97bf8ea62a9c50500c91700c021f0ae5a061048a8362e87d764d36d355b16a`.
Its independent review is
`docs/process/candidate-explicit-seal-comparison-r3-independent-review-2026-09-10.md`
at SHA-256
`1def070c59833bf40dd89d089ad6d700f478a0aeaa63ced0547ce0fc6850b611`.

The exact current R4 subject is
`docs/process/candidate-explicit-seal-comparison-requirement-r4.md` at SHA-256
`bac327c78482e9663ac052368266adec95f2a416094979eee1d9b19a334a9430`.
Its complete non-normative Japanese review-support translation is
`docs/process/candidate-explicit-seal-comparison-requirement-r4-ja.md` at
SHA-256
`9a97a082cd6e39b9701ef107ed6af17278df043c506cfb0346a4476ad31cabb3`.

R4 preserves R3's selected `--against` command, direction, comparison fields,
coherent observation, and scope. It records the accepted format-6 authority,
keeps default format-5 comparison at v2 and default format-6 comparison at v3,
and assigns explicit comparison a fixed generation-aware v4 in both formats.

## Preserved packet

| Artifact | SHA-256 |
| --- | --- |
| `candidate-explicit-seal-comparison-requirement-r1.md` | `75680973023dd7b72140ef1b2d5a9bd5a80aad3dee6d056aa3fe701d43abbdd3` |
| `candidate-explicit-seal-comparison-review-input-2026-09-10.md` | `8e2c4a9794637e343abfcc51cf5e3e91c16a4f3118d4829c35cb894d96305afb` |
| `candidate-explicit-seal-comparison-independent-review-2026-09-10.md` | `6dbbea0c8572a055c4ec6c253c58c7adae1f4a84ee94d0905c594f4a2faaa5b7` |
| `candidate-explicit-seal-comparison-owner-disposition-2026-09-10.md` | `fdf4bd4e790e19cb184311ebc64a14d348301410bd015fa2864a456285b25def` |
| `candidate-explicit-seal-comparison-requirement-r2.md` | `50b6b1b69869abdc3b761d35670bbecde3faf1c9fccaa0bb5197531e7bc16a27` |
| `candidate-explicit-seal-comparison-r2-review-input-2026-09-10.md` | `0799c51d290d7257283b696291329160ba8c13bd568abbf6f616bde0eaeed0ce` |
| `candidate-explicit-seal-comparison-r2-independent-review-2026-09-10.md` | `fec7c03f23a001ae5f0b72227c0a1eeb3cae1fa592565a2b668e74ecc475c8e0` |
| `candidate-explicit-seal-comparison-r2-owner-disposition-2026-09-10.md` | `e52db877ff44e3952ba8d39d4842fc361a785a399551d49e1528f554ca64f9d7` |
| `candidate-explicit-seal-comparison-requirement-r3.md` | `ef97bf8ea62a9c50500c91700c021f0ae5a061048a8362e87d764d36d355b16a` |
| `candidate-explicit-seal-comparison-r3-review-input-2026-09-10.md` | `88c45a5b31e6d2f67075e592af4bcb26d981a4dfd1601531a87b2436866e6e12` |
| `candidate-explicit-seal-comparison-r3-independent-review-2026-09-10.md` | `1def070c59833bf40dd89d089ad6d700f478a0aeaa63ced0547ce0fc6850b611` |
| `candidate-explicit-seal-comparison-requirement-r4.md` | `bac327c78482e9663ac052368266adec95f2a416094979eee1d9b19a334a9430` |
| `candidate-explicit-seal-comparison-requirement-r4-ja.md` | `9a97a082cd6e39b9701ef107ed6af17278df043c506cfb0346a4476ad31cabb3` |

## Preservation validation

All thirteen packet digests matched the values above before the WIP commit.
The R4 authoring-decision audit and exact-candidate self-review audit each
reported zero fatal findings, zero errors, and zero warnings. `git diff --check`
reported no finding for the current change.

The active linked worktree is
`/home/katsumata-m/sealgraph-worktrees/candidate-explicit-compare` on branch
`codex/candidate-explicit-compare`. Accepted `origin/main` at
`c23aa34cea273d7aa2beb94b9c9d69a924560fe6` was merged locally before R4 was
authored. The WIP commit containing this handoff is intended to be the pushed
HEAD of `origin/codex/candidate-explicit-compare`.

## Authority boundary and restart point

Committing or pushing this packet does not accept R3 or R4 and does not
authorize independent review, an ADR, normative-document synchronization,
implementation, PR creation, merge, release, or deployment.

The exact restart point is R4 lifecycle step 2. Present the exact R4 subject,
its complete Japanese translation, material differences from R3, alternatives,
risks, and unknowns, then obtain exactly one first-owner route:
`REVISE`, `REVIEW_THEN_REVISE`, `REVIEW_THEN_DECIDE`, or `REVIEW`. Only after
that route is selected may the corresponding exact-digest independent-review
input be created and routed.
