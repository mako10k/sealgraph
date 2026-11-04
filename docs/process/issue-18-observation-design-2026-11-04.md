# Issue18 observation: basic and detailed design

Authority: exact Accepted R1 and [acceptance receipt](issue-18-observation-r1-acceptance-2026-11-04.md). [ADR0045](../adr/0045-read-only-observation-reasons.md) fixes the additive contract. Owner acceptance of the later implementation remains separate.

| R1 | Basic design | Detailed implementation / evidence |
| --- | --- | --- |
| 1 | Exact current revision evidence, independent of absence claims | Observe selected Seal, Previous and successor AssertionSource identities; canonical graph only; no Candidate/detached assertion import |
| 2 | Active is heads plus revision traversal | Active/ActiveLeaf fields; explicit current vs retained/selected scope; outside-active cannot independently distinguish historical from detached |
| 3 | Explain targets without changing legacy membership | Reason target/kind/direct/paths/paths_complete; complete target membership and bounded witnesses; continue Cause traversal behind a direct stale node |
| 4 | Candidate presence/identity/publication baseline separately | nullable Candidate; prospective Seal/current HEAD comparison; exact Candidate digest; expected_ref_head/state kept distinct; existing candidate compare untouched |
| 5 | Version-specific draft and missing values | nullable HeadDraft and Candidate pointer; present draft bool per version |
| 6 | Workfile baseline and provenance independent | SourceStatus Path/Baseline/Relation plus source comparison revalidation; Trace/review remain inspect/trace/review responsibilities |
| 7 | Preserve existing protocols | only new observe schema; old commands and admission unchanged; registry help/completion and documented selection/limits |
| 8 | Read only | physical namespace before/after equality and normal readers; no writer calls; tests compare state and strict admission |

Repository owns capture, selection, deterministic evidence/reasons and final revalidation. CLI parser accepts at most one exact selector or explicit inventory, bound 1–10000 and human/json. Buffered output uses the operator writer, reporting short writes/errors. Human output escapes untrusted names/paths/assertion text and includes immutable show commands, expected-baseline compare and inspect routes. No mutation suggestions are generated.

`MaxPaths` bounds the number of paths per target, not the number of targets or traversal cost. No general performance guarantee or simultaneous filesystem snapshot is claimed. Missing bindings/files and absent versions are distinct from unchanged/false. Failed or changing observations cannot become a complete clean result. Existing domain evidence records retain their PascalCase inner JSON fields while new observation envelope fields use documented snake_case; this is one fixed new schema, not an old-schema extension.

Verification uses repository fixtures across formats5/6/7, multiple branches/path limits, exact evidence, retained inventory, Candidate canonical identity, draft matrix, immutable selectors, source observations, no-write assertions, old JSON/human/frontier/refs-only preservation and strict historical admission. All required repository checks and independent implementation review precede completion of the implementation tasks. Actual novice-user testing, owner product acceptance and release/install/push remain separate.
