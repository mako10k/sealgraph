# Issue18 observation integration verification

Implementation: `f7bdc72` + complexity-only test split `090e7b0` + review corrections `b3c4fda81828ae6e20360f7494ca8a4fb74fe709`. Governing artifacts: exact Accepted R1, acceptance receipt, ADR0045 and basic/detailed design. No persisted schema or existing observation/admission algorithm changed.

## Scope and results

Repository fixtures cover formats5/6/7, exact observer/target/previous successor evidence, active non-leaf and outside-active reasons, branching sibling leaves without choosing a preferred/latest leaf, retained historical inventory and immutable selection, Candidate assertions excluded from the canonical graph, complete target membership with bounded path witnesses, Candidate SAME/DIFFERENT/NO_HEAD, HEAD/Candidate draft matrix and missing versions, no canonical writes, unchanged status and strict historical-publication rejection. Added HEAD_ADVANCED coverage verifies that current comparison never replaces the publication baseline.

CLI fixtures cover selected REF/immutable Seal/inventory, current/Candidate/workfile baselines, new-schema limitation, human/help/completion, invalid inputs/no bootstrap, output writer errors, and exact unchanged existing status/stale/frontier/refs-only/graph/fsck/candidate-compare JSON outputs on the same fixture. Full existing regression tests continue to protect their broader semantics and old Trace/review contracts.

The initial complexity check found one new test at complexity24. Its assertions were extracted into a responsibility-specific helper; subsequent complexity check passes without changing the threshold. npm ci reported zero vulnerabilities. Clone check reports two existing within-threshold clones (0.08% duplicated lines); no new runtime dependencies were introduced.

Required checks on the corrected code all succeeded: `gofmt -w .`, `go vet ./...`, `go test ./...`, `npm ci`, `npm run clone-check`, `make complexity-check`, `make deadcode-check`, and `make completion-check`. Independent focused reexecution is recorded in the separate implementation review. The local repository's new `observe --format json` also returned the versioned schema successfully, without Candidates or any write.

## Limits and unperformed work

Observation scans retained canonical objects for revalidation and may traverse many Cause paths. The witness cap bounds displayed paths, not traversal cost. No general performance claim, simultaneous filesystem snapshot, future reservation, truth/semantic support/approval attestation or periodic notification is made. Registered workfile unavailability remains explicit. Actual novice-user testing, owner product acceptance, installation, publication and Issue closure have not occurred. No new Seal/Cause operations were performed for this increment.

PERT preserves prior UX acceptance and history, records the Accepted Issue18 increment separately, and leaves OBS_OWNER_ACCEPTANCE to the user. Task completion does not assert measured active effort. Velocity observation remains unavailable (no selected tasks/history incomplete); no guessed time was adopted as actual effort or velocity. Review lifecycle records the actual checkpoint start, with waiting/parallel checks not converted into effort.

Final `perttool document check`, `dag analyze --schedule both`, and `dag next` succeed without errors. The only remaining task is `OBS_OWNER_ACCEPTANCE` (planned); internal design/runtime/docs/check/review tasks are done. Owner response duration is unknown; recording a received decision is provisionally 5–15 minutes of internal work, low confidence, not measured actual effort. The shared work-time check requires no action. Local commits preserve the delivered increment; no remote push or integration is claimed.
