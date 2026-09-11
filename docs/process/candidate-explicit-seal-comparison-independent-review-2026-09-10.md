# Candidate explicit Seal comparison R1 — independent review

Status: COMPLETED

Date: 2026-09-10

Verdict: **FAIL**

## Exact reviewed snapshot

- Candidate path: `docs/process/candidate-explicit-seal-comparison-requirement-r1.md`
- Candidate SHA-256 before review: `75680973023dd7b72140ef1b2d5a9bd5a80aad3dee6d056aa3fe701d43abbdd3`
- Candidate SHA-256 before reviewer report: `75680973023dd7b72140ef1b2d5a9bd5a80aad3dee6d056aa3fe701d43abbdd3`
- Review-input path: `docs/process/candidate-explicit-seal-comparison-review-input-2026-09-10.md`
- Review-input SHA-256: `8e2c4a9794637e343abfcc51cf5e3e91c16a4f3118d4829c35cb894d96305afb`
- First-owner route: `REVIEW`

The independent reviewer edited no repository file. The primary agent verified
the blocking claim against the cited accepted sources before recording it.

## Finding

### P1 — Explicit-mode revalidation is narrower than the accepted selector snapshot contract

Classification: contradiction with authoritative source.

Claim:

R1 accepts the complete immutable selector grammar but requires only one
coherent observation and revalidation of “the relevant observed heads.” That
wording does not preserve the complete observation and revalidation contract
which R1 says remains authoritative outside its stated supersession.

Primary evidence:

- R1 lines 60–61 accept the complete immutable Seal selector grammar.
- R1 lines 85–89 require revalidation only of relevant observed heads.
- Accepted ADR 0023 lines 99–104 define `O` as the complete map of every current
  REF name to exact canonical manifest bytes, including HEAD and scoped tags.
- Accepted ADR 0023 lines 144–156 require recapture and byte equality of that
  complete map, including REF names, presence, absence, HEADs, and complete tag
  arrays.
- Accepted ADR 0023 lines 173–182 additionally require complete loose-object
  inventory `I_O` capture and equality when repository-wide `@SEAL_TOKEN`
  resolution is used.
- Accepted ADR 0023 lines 398–401 apply the complete snapshot discipline to all
  selector resolution and comparison output.

Reasoning:

A newly added loose object can make a repository-wide Seal prefix ambiguous
without changing a REF head. Head-only revalidation could therefore emit a
successful result after the user-supplied selector ceased to be unique. It also
does not establish equality of scoped tags or REF presence and absence.

Earliest causal phase:

Requirement lifecycle step 1. R1 authoring replaced the complete accepted
observation components with narrower head-only language while claiming to
preserve the underlying authority.

Uncertainty:

“One coherent repository observation” could be read as importing complete
`O`, but the explicit final revalidation requirement still says only heads and
does not import conditional `I_O`. The contradiction therefore remains.

Non-normative corrective suggestion:

Create R2 that explicitly imports ADR 0023's complete snapshot discipline:
complete `O` recapture and byte equality, plus conditional complete `I_O`
recapture and equality for repository-wide Seal prefixes. Changed bytes restart
the lifecycle at step 1.

## Answers to review questions

1. **Contradiction found.** R1 narrows an accepted complete snapshot invariant
   outside its stated supersession.
2. **No schema-identity contradiction found.** Preserving the existing
   no-option v2 meaning and assigning v3 only to explicit mode avoids changing
   old machine output under the old identifier.
3. **Insufficient.** Complete manifest and conditional loose-object-inventory
   revalidation are missing or narrowed.
4. **Otherwise internally consistent.** Comparison direction, complete
   identity-bearing fields, and separation of publication concurrency from the
   explicit target are consistent with accepted v2 record semantics.
5. **No scope expansion found.** Future Candidate-to-Candidate,
   workfile-to-explicit-Seal, and generalized operands remain non-blocking.

## Unresolved questions

- “Relevant observed heads” does not state whether it means all REF heads,
  selector-reachable heads, or directly named heads. None is equivalent to the
  accepted exact-manifest-map requirement.
- The frequency and priority of other operand pairs and a generalized future
  syntax remain unknown and non-blocking.

## Primary-agent verification and causal audit

The primary agent read back the unchanged candidate and the cited ADR 0023 and
ADR 0027 sections. The P1 claim and earliest causal phase were then represented
and audited with command-line `llmthink` under thought ID
`sealgraph-candidate-compare-finding-verification`. The audit result was zero
fatal, error, warning, info, and hint findings. LLMThink output is reasoning
evidence only and does not accept the finding or requirement by itself.

## Coverage

The reviewer checked the exact candidate and review input against all seven
accepted sources listed in the input: requirements, architecture, storage
format, CLI contract, and ADRs 0020, 0023, and 0027. Existing implementation
and tests were not needed to establish the normative contradiction and were not
used as requirement authority.

This completed review follows the recorded route to second owner review of the
unchanged R1. It does not accept R1 or authorize an ADR, implementation, commit,
push, release, or deployment.

