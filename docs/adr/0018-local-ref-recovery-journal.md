# ADR 0018: Local REF recovery journal

Status: accepted on 2026-08-17. Implementation remains separately sequenced by
`PLAN.pert`; acceptance of this ADR does not mark runtime recovery implemented.

## Context

Format 4 separates immutable Seal identity from mutable logical REF manifests.
A successful `seal` can advance one manifest, `tag` can add one immutable
scoped binding to a manifest, and `mv` can rename one complete manifest. An
operator may discover immediately afterward that the local operation itself
was accidental.

Semantic correction and operational recovery are different. Meaningful but
wrong or obsolete sealed knowledge is corrected by publishing a new immutable
Seal. A locally accidental REF mutation can instead restore the exact prior
mutable manifest state without changing the accidentally created Seal. The
recovered-away Seal remains a valid historical or detached object.

Manual reconstruction of prior manifests is error-prone, especially because a
format-4 manifest contains both HEAD and its complete scoped tag namespace.
Recovery also needs compare-and-swap protection: it must not overwrite a later
mutation merely because an older local journal entry exists.

The llmthink audit supporting this proposal is
[`../decisions/2026-08-17-local-ref-recovery.think`](../decisions/2026-08-17-local-ref-recovery.think).

## Decision

### Recovery meaning

Recovery restores local mutable REF-manifest state after an explicitly selected
accidental operation. It does not modify Seal bytes, Link bytes,
`parent_revision`, content, attachments, or an existing Seal ID, and it does
not manufacture a corrective Seal.

The operator decides whether an event was a local operational mistake.
Sealgraph does not infer intent or prove that state has not escaped through an
outer Git repository or another copy. A semantic correction remains an
ordinary candidate edit and new Seal publication.

### Version-1 mutation boundary

Recovery v1 records successful canonical REF mutations performed by:

- `seal`, which replaces or creates one REF manifest;
- `tag`, which replaces one REF manifest with an added immutable binding;
- `mv`, which atomically renames one complete manifest from an existing source
  to an absent destination.

`add`, `derive`, `link`, and `unlink` edit mutable candidates rather than REF
manifests and are not journaled by REF recovery. Candidate inspection, editing,
and discard remain their correction path. If an accidental candidate is
published, recovery applies to the resulting `seal` operation.

Failed operations and candidate cleanup do not create recoverable REF events.
A post-publication candidate-cleanup warning does not alter the recorded REF
transition.

### Complete manifest transitions

One operation record contains a fixed operation kind and a sorted set of REF
transitions. Each transition contains:

- one exact logical REF;
- `before`, as absent or exact canonical `sealgraph/ref/v1` bytes;
- `after`, as absent or exact canonical `sealgraph/ref/v1` bytes.

The exact manifest bytes, not only HEAD, are the CAS identity. Before recovery,
every present state is decoded canonically and every HEAD and tag target must
resolve to a canonical Seal. Duplicate REFs and a transition whose before and
after states are equal are invalid.

### Local journal and privacy boundary

Recovery records use a versioned schema under:

```text
.sealgraph/logs/recovery/
```

They are local, non-canonical runtime metadata. They are not included in Seal
identity, REF manifests, canonical configuration, stale/impact derivation,
logical dump/load, or canonical `fsck` validity. They are normally excluded
from outer Git. Their absence, expiration, removal, or corruption reduces only
recovery capability and must not prevent normal repository operation.

The required record does not contain raw argv, content, attachment bytes, cwd,
environment values, actor, hostname, or an assertion that recovery is safe or
authorized. If a timestamp is included for display, it is not identity,
ordering, eligibility, or trusted-time evidence, and its clock is injectable in
tests.

Operation IDs are unpredictable, collision-checked local identifiers. V1 CLI
recovery requires an exact full operation ID; a prefix or implicit most-recent
selection is not accepted.

### Prepare, publication, and crash classification

All journal and REF work occurs inside the existing repository-wide writer
guard. The mutation sequence is:

1. read and validate all `before` states;
2. construct and validate all `after` states;
3. durably publish a `PREPARED` operation record;
4. execute the existing atomic canonical REF mutation;
5. atomically replace the record with `COMMITTED` state.

On inspection or recovery, exact current state classifies each transition:

- current equals `before`: the mutation was not applied or has already been
  recovered;
- current equals `after`: the mutation is recoverable;
- current equals neither: a later or external mutation intervened and automatic
  recovery is rejected.

A durability error after either publication point reports the possible visible
state and requires explicit inspection. The implementation does not blindly
retry or guess from the journal state alone.

Recovery revalidates every transition before writing and emits no partial
success output. A recovered operation is not a redo entry; repeating it cannot
reapply the accidental state.

### Atomicity limit

The journal model can group multiple logical REF transitions under one
operation ID. Recovery v1 executes only transition sets having an existing
format-4 atomic implementation:

- one-manifest exact-state restoration; or
- inverse same-filesystem no-replace rename for one `mv` operation, covering
  source-present/destination-absent as one atomic path transition.

It does not implement arbitrary replacement of two or more independent
manifest files. Format 4 has no canonical transaction root or generation
pointer that could make such writes atomic. A future command needing general
multi-manifest recovery requires a separate canonical storage-format decision,
ADR, fixtures, and migration consideration.

### CLI surface

The initial explicit surface is:

```text
sealgraph recover show
sealgraph recover show OPERATION_ID
sealgraph recover OPERATION_ID
```

Inspection distinguishes at least prepared-not-applied, recoverable,
intervened, already-recovered, and corrupt local-record states. Recovery errors
identify each mismatched REF and direct the operator to inspect current sealed
state. Human and versioned JSON output use recovery terminology.

The surface does not introduce `reset`, `reflog`, `checkout`, `undo`, `redo`,
implicit `last`, REF-based automatic selection, or object deletion.

## Consequences

- Accidental local `seal`, `tag`, and `mv` operations can be reversed without
  rewriting immutable provenance.
- Tag recovery is possible because the entire prior manifest is restored; it
  is not a general tag delete or retarget API.
- Exact after-state comparison prevents recovery from discarding subsequent
  cooperative or detectable external REF work.
- Recovered-away immutable objects remain valid and can appear as historical,
  detached, or unreferenced inventory without affecting the active graph.
- Journal preparation adds a required local durable write before canonical REF
  publication. Journal write failure prevents the REF mutation.
- Removing local logs intentionally removes convenience recovery history but
  changes no canonical meaning.
- General-purpose VCS history manipulation and arbitrary multi-file rollback
  remain absent.

## Acceptance gate

Implementation requires deterministic and fault-injected tests for:

1. accidental existing-REF and initial-REF `seal` recovery;
2. an accidental link/unlink candidate after it is published by `seal`;
3. complete tag-manifest restoration;
4. inverse atomic `mv` recovery;
5. intervening seal, tag, move, and external manifest mutation rejection;
6. all-or-nothing failure with unchanged REF manifests;
7. PREPARED-before, PREPARED-after, COMMITTED-after, and already-restored crash
   classifications;
8. missing, expired, corrupt, symlink, and non-regular journal entries without
   impact on normal repository operations or canonical `fsck`;
9. recovered-away Seal validity and exclusion from active revision/Cause facts;
10. deterministic human/JSON inspection, exact-ID selection, safe diagnostics,
    and absence of Git discovery or Git reset terminology.

Approval of this ADR would cover the local journal schema/store, exact manifest
snapshot and CAS primitives, `seal`/`tag`/`mv` integration, operation-specific
recovery executor, CLI inspection/recovery, and focused tests. It would not
cover implementation before acceptance, arbitrary multi-manifest transactions,
candidate undo, remote/shared recovery, log retention policy, object deletion,
garbage collection, Git sidecar behavior, release, or publication.
