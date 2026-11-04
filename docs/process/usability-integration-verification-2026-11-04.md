# UX R1 integration verification (2026-11-04)

Scope: Accepted UX R1 / ADR0044, additive read-only inspect, text comparison, original-source read, help/completion and repository-local operator/LLM guide. This is implementation/verification evidence, not owner product acceptance or a novice human trial.

## Observed checks

| Check | Result |
| --- | --- |
| gofmt -w . | applied; final source formatted |
| go vet ./... | pass |
| go test ./... | pass; final CLI rerun 12.044s, repository tests pass |
| npm ci | pass; pinned development tooling only |
| npm run clone-check | pass; 2 pre-existing clone groups, total duplicated lines 0.08%, below 0.1% threshold; no new runtime dependency |
| make complexity-check | pass, no function over 20 |
| make deadcode-check | pass including test executables |
| make build / make completion-check | pass; local development binary only, not installed or released |
| independent implementation review | completed, 5 reported issues fixed/rechecked, no outstanding mandatory core correction |
| independent report snapshot hashes | all current files match the reported hashes |

No Git integration change, so the conditional Git-sidecar integration tests are not applicable. No public existing JSON schema or canonical storage version was changed. Read-only APIs validate exact data and detect observed changes before output; this is not a proof of all race interleavings or a global filesystem snapshot.

## Evidence by required scenario

- S1/S2: inspect tests use real temporary repository state, explicit bindings, changed/missing files, unsupported format, unbound Trace and Candidate/HEAD separation. `inspect` never presents these as an overall all-clear.
- S3: compare tests cover changes after the first 256 bytes, long/binary content, compute/output limits, exact extraction routes and unchanged old JSON. Old/new originals remain separate immutable versions.
- S4: historical-review integration tests and inspect tests cover stored claim/applicability independently, changed other-head observation and mixed subject rejection. Operations guide places review show after review set on the first-use path.
- S5: inspection/source-context tests verify no canonical/Candidate mutation and incomplete source facts. Work is not automatically linked/sealed to clean stale.
- S6: trace-read tests cover snapshot/current, Candidate/HEAD/immutable baseline, original context, representative matches, unknown/absent/read failures, binary raw extraction and a 2MiB source. Excerpt truncation and whole-source reads are explicit.
- Error delivery: new operator human views reject failed/short stdout writes; JSON and raw modes retain explicit protocols.

These deterministic fixtures establish bounded software behavior. They do not prove that a novice user understands a claim or that the original semantically supports every possible assertion. A separate independent LLM-agent operation trial follows; real-human usability evidence remains uncollected.

## Progress-record limitations

The design and six implementation/doc tasks have completed their stated artifacts and focused checks; integration checks passed. Their current start/finish events describe WIP integration/validation recording, not reconstructed times for earlier delegated authoring. Initial author/delegate task-specific effort was not measured accurately and is not fabricated. PERT retains the conservative single integration lane; a second simultaneous lifecycle start was rejected for capacity and did not write, so remaining tasks were recorded sequentially. Work was preserved; dependencies/capacity were not altered to bypass that check.

UX_REQUIREMENTS is completed with the user's exact R1 acceptance receipt. Other milestone criterion records are not automatically owner-verified by a test or this report. The final owner acceptance task remains unstarted. The old plan Seal remains the adopted plan snapshot rather than being reissued for every progress event.

Velocity observation still reports unavailable (Grammar9/history incomplete); no synthetic speed or effort is installed. Remaining forecast will be recalculated from current task state and reported separately from external user/trial waiting.
