# Candidate explicit Seal comparison R2 — independent review

Status: COMPLETED

Date: 2026-09-10

Verdict: **FAIL**

## Exact reviewed snapshot

- Candidate path: `docs/process/candidate-explicit-seal-comparison-requirement-r2.md`
- Candidate SHA-256 before review: `50b6b1b69869abdc3b761d35670bbecde3faf1c9fccaa0bb5197531e7bc16a27`
- Candidate SHA-256 before reviewer report: `50b6b1b69869abdc3b761d35670bbecde3faf1c9fccaa0bb5197531e7bc16a27`
- Review-input path: `docs/process/candidate-explicit-seal-comparison-r2-review-input-2026-09-10.md`
- Review-input SHA-256: `0799c51d290d7257283b696291329160ba8c13bd568abbf6f616bde0eaeed0ce`
- First-owner route: `REVIEW`
- Owner-confirmed syntax: `sealgraph candidate compare REF --against SELECTOR`

The independent reviewer edited no repository file. The primary agent verified
the blocking claim against the cited accepted source before recording it.

## Finding

### P1 — Prior-authority disposition omits the accepted architecture boundary

Classification: contradiction with authoritative source.

Claim:

R2 says that it changes only the publication-baseline restrictions identified
in ADR 0023, requirements, CLI documentation, and ADR 0027. The accepted
architecture independently defines Candidate comparison as comparison against
its explicit publication baseline. R2 neither dispositions that architecture
boundary nor includes architecture in its synchronization acceptance criterion.

Primary evidence:

- R2 lines 34–46 list the accepted restrictions proposed for change but omit
  `docs/architecture.md`.
- `docs/architecture.md` lines 153–159 define the history and comparison
  boundary, including Candidate comparison against its explicit publication
  baseline.
- R2 lines 180–185 require synchronization of ADR, requirements, CLI, help, and
  schema documentation but omit architecture.

Reasoning:

The explicit `--against SELECTOR` mode extends Candidate comparison beyond the
publication baseline. Accepting R2 unchanged would leave one accepted primary
source describing the package boundary as publication-baseline-only while the
new requirement permits another target. This contradicts R2's self-review
claim that every directly affected accepted restriction is identified.

Earliest causal phase:

Requirement lifecycle step 1. R2 authoring and self-review omitted architecture
from prior-authority disposition and synchronization acceptance.

Uncertainty:

Low. The R2 review input itself lists `docs/architecture.md` as an accepted
primary source. Even if the passage were later classified as descriptive, that
classification would require explicit disposition before relying on it.

Non-normative corrective suggestion:

Create R3 that explicitly dispositions and synchronizes the architecture
comparison boundary. The selected CLI syntax, comparison behavior, snapshot
discipline, schemas, and out-of-scope behavior need not change. Changed bytes
restart the requirement lifecycle at step 1.

## Answers to review questions

1. **Contradiction found.** The accepted architecture boundary is outside the
   restrictions R2 explicitly proposes to supersede.
2. **No schema-identity contradiction found.** The unchanged form retains v2
   while the explicit form uses v3.
3. **Snapshot discipline passes.** R2 preserves complete `O` recapture and
   equality, conditional complete `I_O` recapture and equality, and the
   no-unnecessary-inventory-scan boundary.
4. **Comparison semantics pass.** Fields, direction, explicit-target identity,
   and publication-concurrency separation are complete and consistent.
5. **No scope expansion found.** Future operand pairs remain unknown,
   out-of-scope, and non-blocking.

## Unresolved questions

None beyond the owner/author disposition required by the P1 finding.

## Primary-agent verification and causal audit

The primary agent read back the unchanged R2 and the cited architecture
section. The P1 claim and earliest causal phase were represented and audited
with command-line `llmthink` under thought ID
`sealgraph-candidate-compare-r2-finding`. The audit result was zero fatal,
error, warning, info, and hint findings. LLMThink output is reasoning evidence
only and does not accept the finding or requirement by itself.

## Coverage

The reviewer checked the exact R2 and review input against all seven accepted
sources listed in the input, including requirements, architecture, storage
format, CLI, and ADRs 0020, 0023, and 0027. The review covered selector and
comparison semantics, v2/v3 compatibility, complete `O` and conditional `I_O`
discipline, comparison direction and publication context, mutation and Git
boundaries, and out-of-scope future comparisons. R1 material was not used to
derive the verdict.

This completed review follows the recorded route to second owner review of the
unchanged R2. It does not accept R2 or authorize an ADR, implementation, commit,
push, release, or deployment.

