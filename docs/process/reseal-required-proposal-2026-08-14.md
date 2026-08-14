# Stale review frontier proposal

Status: reviewed and accepted on 2026-08-14. The normative decision is
[`ADR 0010`](../adr/0010-stale-review-frontier.md).

## Conclusion

Do not add a command named `reseal-required`. Extend the factual, read-only
`stale` surface with two orthogonal options:

```text
sealgraph stale
sealgraph stale --refs-only
sealgraph stale --frontier
sealgraph stale --frontier --refs-only
```

The default selects every stale current REF head. `--frontier` selects only the
upstream-most stale current heads whose current direct upstream REFs are not
themselves stale. `--refs-only` changes presentation, not membership.

This output means “current sealed provenance needs explicit freshness review”.
It does **not** mean “this REF is approved”, “a candidate is ready”, “normal
publication will succeed”, or “seal these names automatically”. In particular,
an intentionally historical draft may be stale without carrying an
organizational obligation to promote or reseal it.

The accepted decision preserves the existing boundaries:

- stale is derived from immutable seals and current REF heads and is never
  stored (`docs/requirements.md:117-131`);
- normal publication, unlike this query, validates a HEAD-consistent,
  non-draft complete closure (`docs/requirements.md:133-147`);
- one mutation publishes exactly one REF, and no command automatically relinks,
  reseals, or propagates draft (`docs/adr/0007-linearized-publication-and-draft-closure.md:41-60`);
- read commands are not a multi-file snapshot API
  (`docs/adr/0007-linearized-publication-and-draft-closure.md:61-62`);
- standalone reads only `.sealgraph`; it does not inspect `.git` and does not
  invoke a Git sidecar.

## Confirmed facts and decision boundary

Confirmed from the current normative contract:

- `status [REF]` derives direct and transitive stale state, while `stale` filters
  current heads to either state (`docs/cli.md:195-213`).
- Direct and transitive stale labels can coexist, and candidate state is
  orthogonal (`docs/cli.md:257-270`).
- Draft may preserve historical provenance; historical is observable rather
  than corruption (`docs/cli.md:272-284`).
- Candidate base relation has `INITIAL`, `CURRENT`, `HEAD_ADVANCED`,
  `HEAD_MISSING`, and `UNEXPECTED_HEAD` states (`docs/cli.md:118-142`).
- The current implementation already derives stale through a cycle- and
  ownership-validating graph walk and does not persist it
  (`internal/graph/inspect.go`). `Repository.Stale` derives from current heads
  without loading candidate state (`internal/repository/repository.go`).
- Current `impact` may report multiple paths to one downstream REF; structural
  impact is not stale work (`docs/process/backlog.md:176-205`).

ADR 0010 accepts `--frontier`, `--refs-only`, their exact set rules, and the
output behavior below.

## Set contract

Take one validated observation of the current canonical seal graph:

```text
S = { r | r has a current head and that head is STALE_DIRECT or STALE_TRANSITIVE }

deps(r) = { u | the current seal of r has a direct concrete link naming REF u }

F = { r in S | for every u in deps(r), u is not in S }
```

`sealgraph stale` and `sealgraph stale --refs-only` select `S`.
`sealgraph stale --frontier` and the combined form select `F`.

`F` is a **freshness-review frontier**, not a publish-admissibility frontier.
It says no current direct upstream REF of the selected REF still has stale work.
After any explicit one-REF relink/seal operation, the operator reruns the query;
the old result is not an executable plan or reservation.

The direct-upstream test handles chains and diamonds without inventing a batch:

```text
A <- B <- C, with B and C stale: frontier = B

A <- B <- D
  <- C <- D, with B, C, D stale: frontier = B, C
```

If `D` has two or more stale paths, `D` still appears at most once. When `B`
and `C` become fresh through separate explicit operations, a new observation
may place `D` on the frontier.

## Command and output contract

The set selector and presentation selector are deliberately independent:

| Command | Selected set | Presentation |
|---|---|---|
| `stale` | all `S` | detailed human stale evidence |
| `stale --refs-only` | all `S` | exact REF line stream |
| `stale --frontier` | `F` | detailed human stale evidence |
| `stale --frontier --refs-only` | `F` | exact REF line stream |

All selected logical REFs are deduplicated and ordered by bytewise lexical REF
order. Do not emit a path-order or a purported total topological plan. Detailed
output may retain multiple path records as evidence inside one REF record.

For `--refs-only`, stdout is exactly zero or more records of:

```text
<valid-logical-REF>\n
```

There is no heading, seal ID, status label, quoting, `CLEAN`, or abbreviated ID.
An empty selection is zero-byte stdout. This narrow line protocol does not
replace the cross-command versioned JSON work owned by SG-BL-006
(`docs/process/backlog.md:207-236`). ADR 0010 stabilizes only this minimal REF
line stream before that broader work.

Detailed modes retain full 64-character IDs and human-readable stale path
evidence. They may show `DRAFT` only when it belongs to the current sealed
generation. They never list, load, validate, or annotate mutable candidates.
`status REF` and `candidate show REF` own that working-state inspection.

Do not print `RESEAL_REQUIRED`, `READY`, `APPROVED`, or a generated `sealgraph
seal ...` command.

Successful empty and non-empty results both exit 0. Usage errors exit 2. An
operational or integrity failure is nonzero under the eventual common CLI exit
contract. Staleness itself is a query result, not exit 1; a future policy check
mode, if wanted, needs a separate explicit decision. Results are buffered so an
integrity failure never leaves a plausible partial list on stdout.

`stale` accepts no positional selector. `REF`, `REF@full-id`, unique ID prefix,
and REF-scoped tag are all invalid operands here. `status REF` owns one-current-
REF inspection; `show REF@TOKEN` owns historical selection; `impact REF` owns
reverse structural paths. A historical seal or tag has no independent current
stale membership.

## Command x state matrix

“In S/F” below is always determined from the current canonical head, never from
the mutable candidate.

| State at observation | In `S` | In `F` | Required interpretation/action |
|---|---:|---:|---|
| clean non-draft head, no candidate | no | no | no output |
| clean draft head | no | no | draft remains visible in `status`; freshness query says nothing about promotion |
| clean head plus any candidate | no | no | `status`/`candidate show` owns `UNSEALED`; candidate does not make the head stale |
| stale head, no candidate | yes | iff all current direct upstream REFs are outside `S` | operator may create/review one explicitly |
| stale draft head or intentional historical link | yes | same frontier rule | factual stale only; do not claim reseal/promotion is required |
| stale head, candidate base `CURRENT` | yes | same | candidate is not read; inspect it with `status`/`candidate show` |
| stale head, candidate base `HEAD_ADVANCED` | yes | same | candidate is not read; inspect/discard/recreate explicitly before publication |
| stale head, candidate base `UNEXPECTED_HEAD` | yes | same | candidate is not read; its intent predates the unexpected current head |
| draft or historical candidate | membership follows current head | same | candidate is not read; normal/draft publication rules remain unchanged |
| initial candidate, no current head | no | no | `candidate show` reports `INITIAL`; there is no sealed head to reseal |
| candidate base exists but owner HEAD is missing | no | no | `candidate show` reports `HEAD_MISSING`; absence of a current head is not stale membership |
| link names an upstream REF whose current HEAD is missing | error | error | freshness cannot be derived; no partial stdout |
| current head/object or reachable dependency is corrupt, wrong-owner, or unreadable | error | error | fail closed and name explicit inspection/recovery action |
| reachable immutable seal-ID cycle | error | error | fail closed; never choose a frontier from an invalid DAG |
| diamond or multiple stale paths to one REF | yes once | by the same direct-upstream rule | one target record/name; detailed mode may show every evidence path |

Candidate handling is now intentionally absent from every `stale` mode. The
query derives and buffers `S` or `F` from canonical heads only. A corrupt
candidate therefore cannot change membership or suppress detailed/REF-only
stale output. `status` may still combine head and candidate state for its own
broader inspection contract.

## Read consistency and mutation boundary

Every mode is read-only:

- no object, REF, tag, candidate, cache, receipt, or derived stale write;
- no writer guard and no runtime-directory bootstrap;
- no automatic relink, candidate creation/rebase, seal, repair, or `fsck`;
- no `.git` discovery, Git object read, Git command, or sidecar suggestion.

Because readers are not a repository-wide multi-file snapshot, the result means
“validated observation completed successfully”, not “these heads remain fixed”.
The implementation must either validate a coherent observation or fail; it must
not claim a reservation. Normal `seal` remains responsible for revalidating the
candidate version, complete closure heads, draft closure, and expected target
HEAD under the writer protocol before its one REF CAS publication
(`docs/architecture.md:181-199`).

Native v3 seal identity is not changed. The query neither reads nor invents an
actor, timestamp, or seal-event message. Edge messages remain hash-bearing
dependency relation state and may be displayed as safely quoted evidence, but
they do not affect whether an edge is stale except through the enclosing seal
identity (`docs/adr/0009-separate-seal-event-metadata.md:18-38`).

## Alternatives considered

1. **Top-level `reseal-required`. Rejected.** “Required” adds organizational
   policy that cannot be derived for intentional draft/history, and “reseal”
   compresses review, candidate edit, relink, and publication into one implied
   action.
2. **Top-level `review-required`. Rejected for now.** Better wording, but still
   duplicates the same stale set and can imply policy. Existing `stale` is the
   factual source of truth.
3. **Only `stale --refs-only`, no frontier. Valid minimal alternative.** It
   avoids a new graph view but makes every operator independently reconstruct
   safe upstream-first progress, especially in diamonds.
4. **One topologically sorted all-stale list. Rejected.** A DAG has multiple
   valid total orders; the list becomes stale after the first operation and
   looks like an automatic batch plan.
5. **Normal-only frontier excluding draft heads/ancestry. Rejected.** That
   turns freshness inspection into a domain policy for promotion. Draft
   admissibility stays with explicit `seal` validation.
6. **Candidate-ready frontier. Rejected.** Candidate presence/base equality
   neither proves review nor complete closure admissibility and would make
   mutable state alter a canonical stale fact.
7. **Exit 1 when results exist. Deferred.** Query state and command failure stay
   separate; a future `--check` would require a cross-command policy contract.

## Accepted decisions

1. Extend `stale`, rather than add a `reseal-required` command.
2. Adopt the exact `S` and `F` definitions and the term “freshness review
   frontier”.
3. Stabilize `--refs-only` now as the exact minimal line protocol; future JSON
   remains separate.
4. Use bytewise lexical order and one-record-per-REF deduplication rather than
   a topological total order.
5. Exclude candidate reads and annotations from every stale mode.
6. Exit 0 for both empty and non-empty successful query results.
7. Describe the result as a validated observation, not a reservation or batch
   plan, and require rerunning after each explicit link or seal operation.

## Acceptance tests

1. Empty all/frontier detailed output is `CLEAN`; empty refs-only output is zero
   bytes; all successful queries exit 0.
2. Direct-stale and transitive-stale current heads are both members of `S`, and
   a head bearing both labels appears once.
3. Chain `A <- B <- C` produces `B` alone in `F` while `B` and `C` are stale;
   after explicit repair of `B`, rerunning produces `C` if still stale.
4. A diamond produces each frontier REF exactly once, in lexical order,
   regardless of link canonical/input order or number of stale paths.
5. `--refs-only` emits only valid logical REF bytes plus LF, no IDs, labels,
   headers, `CLEAN`, quoting, or partial output.
6. A stale draft/historical head is included under the same `S/F` rules and is
   never described as mandatory promotion or ready-to-seal.
7. Clean draft, candidate-only initial, and candidate `HEAD_MISSING` states are
   excluded. A stale current head remains included regardless of candidate
   absence, base relation, or candidate draft/historical intent.
8. No stale mode lists, loads, validates, or annotates candidates. A corrupt
   candidate cannot change `S/F` or cause either detailed or refs-only failure.
9. Missing dependency HEAD, missing/corrupt/hash-mismatched object, wrong seal
   owner, invalid ref file, and cycle all fail nonzero with empty stdout and an
   actionable stderr diagnostic.
10. Positional REF, REF@full-ID, unique prefix, tag, repeated/unknown flags, and
    forbidden combinations produce usage exit 2 without reading `.git`.
11. The command does not create `.sealgraph` runtime directories or modify any
    object, REF, tag, candidate, lock, receipt, canonical bytes, or directory
    entries.
12. A concurrent/cooperative mutation may make a completed observation old, but
    no result is called a snapshot/reservation and later normal seal publication
    revalidates closure/current-head conditions.
13. Native v3 output and identity tests confirm no actor, created_at, or
    seal-event message is introduced; link messages remain relation evidence.
14. Help explicitly states that `--frontier` proposes no batch operation,
    automatic relink/reseal/repair, Git behavior, truth, or trust.

## llmthink audit

The companion design model is
`docs/decisions/2026-08-14-reseal-required.think`. It was audited with:

```sh
llmthink dsl audit docs/decisions/2026-08-14-reseal-required.think --pretty
```

The accepted companion model is re-audited after every contract edit. Audit
diagnostics are development evidence; the normative acceptance remains ADR
0010 and the executable tests. The accepted revision audits with
`fatal=0 error=0 warning=0 info=0 hint=12`; the hints are the explicit
preferred/rejected comparison pairs plus style guidance, not an undisclosed
contract contradiction.
