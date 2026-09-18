# Issue #17 DIRECTION_COMPARE execution note — 2026-09-18

- Repository branch at instruction: `codex/issue-17-origin-trace-r2-wip`, HEAD `550fc11`; working tree clean.
- Governing scope: Accepted R1/R2/R3 and ADR 0032 §7, ADR 0036 §2.3–2.4, ADR 0038 §§2–4. This note does not amend their semantics or acceptance.
- P4 `SELF_COMPARE` has committed local source binding and Candidate/HEAD/exact Seal own-presence components, including local JSON/human record builders. Its accepted public-output completion condition remains open because `trace compare/v2` requires graph output.
- The user was shown that P4 was active and that the full public `trace compare/v2` output requires P5, then explicitly instructed: 「コミットしてから `DIRECTION_COMPARE` に進めて」. The requested code work proceeds within P5's accepted scope.
- `perttool task start ... DIRECTION_COMPARE` preview failed with `PTACT-108` while P4 occupies `WRITER 1`. Simulated suspension of P4 followed by P5 start failed with `PTDAG-207` because `SELF` is not reached. Neither task was falsely marked complete; PERT continues to show P4 active and P5 planned until a supported truthful lifecycle update is available.
- Execution began after the user instruction at approximately 2026-09-18 16:44 JST. The exact instant was not recorded by PERT. Code commits and this note provide the restart evidence; they do not assert a PERT start event.
- Exit evidence for P5: exact Cause graph scope and budget, independent own/upstream/downstream aggregation over shared local observations, direct and indirect paths, Candidate/HEAD separation, immutable target identity, graph conflict detection, JSON/human representation, and unchanged STALE semantics. The optional `--estimate` branch belongs to P6 and cannot be presented as implemented by P5.
