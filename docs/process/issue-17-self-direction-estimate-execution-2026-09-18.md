# Issue #17 P4/P5/P6 execution record — 2026-09-18

## Scope and authority

The owner instructed execution of `SELF_COMPARE` and `ESTIMATE_COMPARE` after
the earlier explicit `DIRECTION_COMPARE` instruction. Accepted R2/R3 and ADR
0035/0037/0038 govern the public comparison and optional range estimate.
`status`, stale semantics, and the separate occurrence-list operation were not
changed by these slices.

## Observed result

- `SELF_COMPARE`: the public `trace compare --ref|--seal --max-graph-visits`
  command returns `sealgraph/trace-compare/v2`. Stable current bytes drive
  whole-run exact presence; selection, Candidate/HEAD separation, graph and
  observation identities are included. Without `--estimate`, the declaration
  digest is null and every estimate is `NOT_REQUESTED`.
- `DIRECTION_COMPARE`: exact Cause graph results retain independent own,
  upstream, and downstream facts, path membership, graph budget and scope
  completeness. A public CLI test confirms a downstream Cause and that a
  limited graph preserves completed own facts.
- `ESTIMATE_COMPARE`: only `ABSENT_EXACT` receives automatic deterministic
  single-diff candidates or matching explicit correspondence candidates.
  Candidate and no-candidate cases are tested, as are conflicting declarations,
  fixed current identity, and unchanged exact presence. A synthetic
  `INCOMPLETE` output test confirms that estimate incompleteness does not erase
  `ABSENT_EXACT`; the current in-memory diff has no ordinary recoverable
  resource-error path, so a live resource-error trigger was not exercised.
- Public `trace correspondence put/show/list/remove` uses an ID-sorted local
  declaration set and fixed source/current identities. Its candidate reason
  reports whether declared concatenated bytes equal the old range; differing
  bytes remain valid because the declaration can describe changed content.

Commits: `9816009`, `271ebdc`, `d85b65e`, `1f6297d`, `7960b34`,
`2fbf5c8`, `2f38ed9`, `2355478`, and `c9062d8`.

`go vet ./...`, `go test ./...`, `npm ci`, `npm run clone-check`,
`make complexity-check`, and `make deadcode-check` passed after the public
command and direction tests. The focused incomplete-estimate test passed
afterward. `gofmt -w .` was run before the full suite.

## PERT reconciliation

P5 graph code and part of P6 estimate code were written as authorized early
work while P4 still occupied the single `WRITER` and before the public P4
contract was complete. The prior [P5 execution note](issue-17-direction-compare-execution-2026-09-18.md)
records why `perttool` could not start P5 at that time. The PERT start/finish
events for P5 and P6 now mark their later formal validation and closure, not
the beginning of all code effort. This prevents the recorded 1–2 minute
validation intervals from being mistaken for the full task throughput.

For a remaining-work forecast, the relevant completed sample excludes P5/P6:
`FORMAT_TYPES`, `RANGE_COMPARATOR`, `TRACE_AUTHOR`, and `SELF_COMPARE` total
25 planned Points over 3.9275 recorded elapsed hours (6.365 Points/hour).
This is elapsed throughput, not person-hour effort, and contains four
heterogeneous tasks. The PERT resource schedule after P4/P5/P6 closure is
22.167 Points for the remaining internally executable tasks. At the sample
rate that is 3.48 elapsed working hours; a 2.9–6.4 Points/hour sensitivity
gives about 3.5–7.7 working hours. This is a conditional planning calculation,
not an owner acceptance or release date.

## Remaining gates

`OCCURRENCES_LIST`, `NATIVE_TRANSPORT`, `ENVIRONMENT_REVIEW`, detailed trial,
status decision, normative sync and full AC1–21 verification remain. The
`trace compare` implementation has no accepted numerical performance
threshold; its optional single diff may be costly on large source/current
files. Detailed trial and measurement must evaluate that cost before any new
public resource limit or completion claim is proposed.
