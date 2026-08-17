# Local REF recovery contract acceptance

Status: accepted on 2026-08-17. This receipt closes only the design and
acceptance-matrix task. No recovery journal, canonical mutation integration, or
CLI implementation is claimed.

## Accepted boundary

- Semantic correction remains an immutable new-Seal operation.
- Local operational recovery restores complete exact REF-manifest state and
  never changes immutable objects.
- V1 records successful `seal`, `tag`, and `mv`; candidate-only commands are
  outside REF recovery.
- Records are versioned non-canonical local metadata with no raw command,
  content, path, environment, actor, or trusted-time payload.
- PREPARED precedes canonical publication; COMMITTED follows it. Exact current
  before/after comparison, not log status alone, decides recovery state.
- V1 atomic execution is limited to one-manifest restoration and inverse
  no-replace move. General multi-manifest rollback requires a separate format
  decision.
- CLI selection is one exact operation ID. Git reset/reflog vocabulary,
  implicit last/REF selection, undo/redo, deletion, and GC are absent.

## Acceptance matrix for implementation

| Area | Required evidence |
| --- | --- |
| Journal codec | Strict schema/order/member/limit fixtures; duplicate REF and equal transition rejection |
| Safety | Symlink/non-regular/path traversal rejection; no sensitive argv/content/environment fields |
| Crash states | PREPARED-before, PREPARED-after, COMMITTED-after, already-restored, and intervened fixtures |
| Seal | Existing-REF and initial-REF recovery; recovered-away Seal remains valid and inactive |
| Link/unlink | Accidental candidate relation is recovered only through its successful Seal publication |
| Tag | Whole-manifest restoration removes only the accidental local binding and preserves prior tags/HEAD |
| Move | One inverse atomic no-replace rename; occupied source/destination fails with neither state changed |
| Concurrency | Later seal/tag/move/manual manifest change rejects the complete recovery |
| Optionality | Missing/removed/corrupt journal does not block open, normal commands, or canonical `fsck` |
| Graph | Active revision/Cause, stale, frontier, and impact use restored current heads only |
| CLI | Exact operation ID, deterministic human/JSON, no partial stdout, safe next-action diagnostics |
| Boundaries | No Git discovery, corrective Seal, candidate undo, general multi-file transaction, object deletion, or GC |

## Audit and plan evidence

`llmthink dsl audit docs/decisions/2026-08-17-local-ref-recovery.think
--pretty` reports zero fatal, error, or warning findings. `PLAN.pert` sequences
implementation after this contract as `RECOVERY_CORE`, parallel
`RECOVERY_SEAL_TAG` / `RECOVERY_MOVE`, then `RECOVERY_ACCEPTANCE`.
