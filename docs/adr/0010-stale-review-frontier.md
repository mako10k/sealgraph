# ADR 0010: Stale review frontier and stable REF stream

Status: accepted on 2026-08-14 by explicit operator approval.

## Context

`sealgraph stale` already reports every current REF head whose immutable
provenance is directly or transitively stale. Operators also need a deterministic
list of the upstream-most stale REFs that can be reviewed before their stale
downstreams. Calling that list `reseal-required` would incorrectly turn a
derived freshness fact into an organizational obligation, particularly for
intentional draft or historical provenance.

Candidate state is mutable and orthogonal to current sealed provenance. Reading
candidate files while deriving stale membership would allow an unrelated or
corrupt working candidate to hide otherwise valid stale information.

Read-only commands do not acquire the repository writer guard. A multi-REF
observation therefore needs to detect cooperative or external HEAD movement
without claiming to reserve the result.

## Decision

### Command surface

Extend the existing factual `stale` command with two orthogonal flags:

```text
sealgraph stale
sealgraph stale --refs-only
sealgraph stale --frontier
sealgraph stale --frontier --refs-only
```

Let:

```text
S = { r | r has a current head and that head is directly or transitively stale }
deps(r) = { u | the current seal of r has a direct dependency naming REF u }
F = { r in S | every u in deps(r) is outside S }
```

The default selects all of `S`. `--frontier` selects `F`. The frontier is an
upstream-first freshness-review frontier, not a seal-admissibility result,
approval, reservation, or batch plan. Draft current heads use the same factual
membership rule; the command never says that promoting or resealing them is
mandatory.

All selected REFs are deduplicated and ordered by bytewise lexical REF order.
The human presentation keeps the existing detailed stale evidence and prints
`CLEAN` for an empty selection. The selected set is already clear from the
invocation, so `FRONTIER` is not added to the REF status vocabulary.

### Stable REF-only output

`--refs-only` is a deliberately stable, minimal line protocol. Stdout is
exactly zero or more records of:

```text
<valid-logical-REF> LF
```

It has no heading, seal ID, status label, quoting, `CLEAN`, or abbreviated ID.
An empty selection is zero-byte stdout. Successful empty and non-empty queries
both exit zero. Usage and operational/integrity failures remain nonzero and do
not emit a plausible partial list. Future versioned JSON is a separate format;
it does not replace or extend this line protocol.

### Canonical-only observation

Every `stale` mode reads immutable seals and current REF heads only. It does not
list, load, validate, or annotate candidates. `status REF` and `candidate show
REF` remain the explicit surfaces for mutable working state.

The query captures the complete current REF/head set, derives all membership
against that captured set, buffers its output, and re-reads the complete
REF/head set before emitting. If any name or head changed or became unreadable,
the command fails with empty stdout and tells the operator to rerun it. It does
not acquire the exclusive writer guard or retry indefinitely.

A successfully validated result can still become old immediately after the
final check. Help therefore states:

```text
Reports a validated observation of current REF heads.
The result is not a reservation or batch plan.
Re-run after each link or seal operation; seal revalidates dependencies before publication.
```

Normal seal publication remains responsible for writer-guarded closure and
expected-state validation.

### Failure and mutation boundary

Missing dependency heads, invalid REF files, unreadable or corrupt objects,
ownership mismatches, and immutable seal cycles fail closed before stdout is
written. No stale mode creates or modifies objects, REFs, tags, candidates,
caches, logs, locks, or Git state. No mode relinks, reseals, repairs, or invokes
Git behavior.

## Consequences

- Shell workflows receive a simple deterministic REF stream without turning
  query results into commands.
- Chains and diamonds can be reviewed one upstream layer at a time, rerunning
  after every explicit one-REF mutation/publication.
- Candidate corruption cannot suppress canonical stale visibility.
- Detailed stale output no longer includes `UNSEALED` or candidate-derived
  `DRAFT`; those remain available from `status`.
- The observation is coherent when emitted but is not a durable snapshot or
  reservation.
- No persisted schema, object identity, or storage-format field changes.
