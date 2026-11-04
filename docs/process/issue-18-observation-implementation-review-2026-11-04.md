# Issue18 observation independent implementation review

Input and boundary: [fixed review input](issue-18-observation-implementation-review-input-2026-11-04.md), exact implementation `090e7b020acfb46118496efa6a9ab7c2682dfb55`; accepted R1 SHA-256 `96e4aba15517472049c5134b5720ef6c4689fafb01ca2d3477fe4fcc45aa1df4`. Independent read-only agent `/root/issue18_observation_impl_review` inspected source/design and ran focused tests. Root transcribes the reported result here; the reviewer did not edit implementation or acceptance state.

## Findings and disposition

1. INSIDE / P2 / R1 §2: human/JSON originally exposed active=false without explaining that current evidence cannot distinguish historical from detached. Fixed by new-schema `membership_limitation` and the same human introduction, with CLI coverage. This changes no old output or membership rule.
2. INSIDE / P2 / documentation / R1 §6: ADR's blanket read-failure statement exceeded the actual source behavior. Fixed by distinguishing canonical integrity/read failure and detected concurrent change (nonzero) from an explicit SOURCE_MISSING/SOURCE_UNREADABLE workfile observation (success with unavailable relation). No workfile failure is presented as unchanged.

Correction snapshot: `b3c4fda81828ae6e20360f7494ca8a4fb74fe709`. Also added a focused HEAD_ADVANCED test: current HEAD remains distinct from expected_ref_head, Candidate identity compares against current HEAD, and normal publication still rejects the stale expectation.

The same reviewer read the correction diff, retained the same R1/boundary, confirmed both findings resolved, and independently reran `go test ./internal/repository ./internal/cli -run TestObserve -count=1`: success (repository 19.419s, CLI 2.016s). No unresolved mandatory INSIDE finding. No BOUNDARY_DISPUTE. Recommendation: implementation review complete. This recommendation is not owner product acceptance.

OUTSIDE: actual novice-user trials, owner acceptance of delivered product, push/merge/release/install, general performance guarantees and excluded Issue18 alternatives remain separate. No vote or number of agents is used as evidence of product correctness.
