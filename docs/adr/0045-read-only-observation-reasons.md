# ADR 0045: Separate observation evidence, reasons and baselines

Status: Accepted within the owner's 2026-11-04 direction to design and implement Accepted Issue18 observation R1. No separate owner review of this ADR or product acceptance is claimed.

## Context and evidence

C-OBS-1: observation is neither history completeness nor publication permission. E-OBS-1: [exact Accepted R1 and owner receipt](../process/issue-18-observation-r1-acceptance-2026-11-04.md), requirement SHA-256 `96e4aba15517472049c5134b5720ef6c4689fafb01ca2d3477fe4fcc45aa1df4`. E-OBS-2: its independent review found no mandatory requirement contradiction, while requiring legacy stale membership, frontier, refs-only and admission to remain unchanged.

C-OBS-2: legacy status intentionally aggregates draft and Candidate presence; candidate compare intentionally uses expected_ref_head. E-OBS-3: existing revision_graph.go and candidate_lifecycle.go and ADRs 0010/0043/0044. Reinterpreting their outputs would break accepted consumers.

## Decision and basic design

Add read-only `observe [SELECTOR] [--inventory] [--max-paths N] [--format human|json]` with new JSON schema `sealgraph/observation/v1`. Existing status, stale, graph, fsck and inspect outputs and all writers remain unchanged. SELECTOR chooses one exact REF or immutable Seal; inventory explicitly requests retained physical Seals and conflicts with SELECTOR. Default scope is current REFs, Candidates and source bindings plus the canonical observed graph. No assertion from an otherwise detached Seal or prospective Candidate is admitted into the current graph merely by displaying it.

Repository observation owns facts and validation. CLI owns selection syntax, escaping, human navigation and buffered output. Observation separates:

- Recorded revision assertions, their exact observer/target/previous identities, and observed successors. No successor evidence means no evidence in the stated observation scope, never proof of nonexistence.
- Active membership derived from current HEADs and revision edges only. Cause reachability does not confer active membership.
- Complete review-target membership for ACTIVE_NON_LEAF and OUTSIDE_ACTIVE, with direct/deeper Cause paths. A positive path limit (default 100, maximum 10000) bounds witnesses per target/reason; explicit incompleteness never suppresses a reason. These diagnostics do not redefine legacy stale labels or frontier.
- Candidate existence, prospective Seal identity versus current HEAD (SAME/DIFFERENT/NO_HEAD), expected_ref_head and publication-baseline state. Candidate reason diagnostics use the current graph, not simulated publication admission. candidate compare keeps its existing expected_ref_head baseline.
- Nullable HEAD draft and separately nullable Candidate facts. Workfile relation retains an explicitly named Candidate-or-HEAD content baseline. Trace and historical review remain separate through inspect/trace/review commands.

No canonical persisted fields, migrations, automatic repair, Candidate generation, Cause update, Seal issuance, background notification or Git inspection are added.

## Detailed design and failure boundaries

`internal/repository/observation*.go` captures the current observation, validates selected and retained Seals using existing readers, constructs deterministic facts and bounded path witnesses, and revalidates context before returning. Candidate exact digests and source state must remain coherent with the captured heads. Read failures or detected concurrent changes fail observation; they cannot become absence or a clean result. Stable reads and revalidation do not promise one simultaneous filesystem snapshot or future reservation.

`internal/cli/observe*.go` calls this API once and emits a completely buffered new-schema JSON document or an escaped human summary with exact identities and read-only follow-up routes. Invalid selection, limits and output failures are nonzero. Existing immutable and admission algorithms are reused without semantic changes. New and materially changed modules keep their R: responsibility comments.

## Alternatives and consequences

Adding fields to old JSON or changing UNSEALED/draft would change a stable contract. A prospective graph would conflate evidence and a publication simulation. Persisting observation state is unnecessary because membership is derived. A path cap is a display witness bound, not a claim that all paths were listed or a performance guarantee.

## Actions, verification and review

A-OBS-1 → C-OBS-1/E-OBS-1/2: implement repository observation with branching, historical/detached, direct/deeper and bounded-path fixtures across formats 5/6/7. A-OBS-2 → C-OBS-2/E-OBS-3: additive CLI/help/completion and baseline/draft/source tests, read-only assertions, legacy output and publication regressions. A-OBS-3: required Go/clone/complexity/deadcode checks and an independent implementation review against the fixed R1/design/diff. Product owner acceptance and actual novice-user testing remain separate, unperformed judgments until recorded.
