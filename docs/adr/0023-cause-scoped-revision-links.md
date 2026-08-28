# ADR 0023: Cause-scoped revision links with legacy parent fallback

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-RL-001, C-RL-002, C-RL-003
- Related Evidence: E-RL-001, E-RL-002, E-RL-003
- Related Reviews: Pending
- Supersedes: Portions of ADR 0011 only after this ADR is accepted
- Superseded by: None

## Context

Format 4 stores one optional `parent_revision` directly in every Seal and keeps
that revision edge separate from exact Cause Links. That makes revision
ancestry an intrinsic property of the Seal.

The operator instead wants a Seal to contain no intrinsic parent/child
relation. Revision meaning belongs to the context in which another Seal uses a
target as a Cause. Each Cause Link therefore records the exact previous
revision Seals of its own `target_seal`. The combination of Cause Links, rather
than a top-level Seal parent, supplies revision facts.

The operator supplied this working grammar on 2026-08-28 (E-RL-001):

```text
cause_links := [ cause_link ]
cause_link := {
  target_seal: SEAL,
  previous_revision_seal_of_target_seal:
    [ previous_revision_seal_of_the_target_seal ]
}
```

Existing format-4 objects are immutable and include `parent_revision` in their
canonical bytes and SealID (E-RL-002). Compatibility must preserve those bytes
and their meaning without rewriting historical objects.

ADR 0011, `docs/requirements.md`, `docs/storage-format.md`, and the checked-in
runtime currently require separate intrinsic revision and Cause edges
(E-RL-003). This proposal cannot be implemented as a compatible unknown-field
extension of `sealgraph/seal/v4`.

## Decision

### Confirmed direction

The following decisions are confirmed by the operator:

1. A new-format Seal has no top-level parent or child identity.
2. A Cause Link identifies one exact `target_seal` and records zero or more
   exact previous revision Seals of that target.
3. The old Seal-level `parent_revision` is deprecated but remains the default
   Revision Link for compatibility when a Cause Link has no explicit previous-
   revision member.
4. Historical Seals remain immutable. Compatibility is interpretive; it never
   rewrites an existing Seal or changes its SealID.

These decisions establish:

- C-RL-001: Revision facts are Cause-Link-scoped, not intrinsic Seal state.
- C-RL-002: A Cause Link may commit to an ordered-canonicalized set of exact
  previous revisions of its target.
- C-RL-003: Legacy `parent_revision` supplies a default only when the Cause Link
  lacks an explicit previous-revision statement.

### Compatibility resolution rule

For one Cause Link `l` with target Seal `t`, resolve the link-local previous
revision set as follows:

```text
if l has an explicit previous-revision member:
    previous(l) = the exact set recorded by l
else if t is a legacy Seal and t.parent_revision is non-null:
    previous(l) = { t.parent_revision }
else:
    previous(l) = {}
```

Presence is semantically significant. An explicit empty array means “this Link
records no previous revision for its target” and suppresses the legacy
fallback. A missing member means “use the compatibility default.” New-format
canonical Links should require the array member, including an explicit empty
array, so only legacy Links take the missing-member path.

An explicit Link statement is authoritative for that Link even when its target
is a legacy Seal with a different `parent_revision`. The legacy field remains
canonical identity-bearing data of the target Seal; it is not deleted,
rewritten, or silently copied into the new Link bytes.

Every recorded previous revision is a full exact SealID. Duplicate IDs within
one array are invalid. Canonical encoding sorts IDs bytewise unless a later
accepted semantic rule proves that array order carries meaning. Missing,
noncanonical, self-referential, or cyclic revision targets fail closed.

### Remaining acceptance gates

This ADR remains Proposed because the confirmed data shape does not yet decide
the global derived graph. Before acceptance, the design must fix:

1. which Cause Links participate in one coherent revision observation;
2. whether differing previous-revision sets for the same target are combined,
   kept contextual, or rejected as conflicting claims;
3. how current REF heads, Cause reachability, tags, and detached objects root
   revision observations;
4. how `log`, scoped selectors, stale, impact, frontier, `fsck`, and caches use
   link-scoped revision facts;
5. how root, draft, normal Cause admission, and revision cycles interact;
6. the new repository, Seal, Candidate, and machine-output schema versions;
7. the explicit migration or mixed-read boundary for format-4 repositories;
8. candidate authoring syntax and the future meaning of `derive` and
   `add --parent`.

No runtime or public CLI implementation is authorized until those gates are
resolved in this ADR or a linked accepted format-transition ADR.

## Alternatives

### Keep intrinsic `parent_revision`

Not selected as the target direction because it contradicts C-RL-001. It
remains the legacy compatibility source defined by C-RL-003.

### Treat the Cause target itself as the previous revision

Rejected because the operator's grammar distinguishes `target_seal` from the
list of previous revisions of that target.

### Treat an empty array as missing

Rejected because it would make an explicit initial/no-previous assertion
indistinguishable from a request to inherit the legacy default.

### Rewrite legacy Seals into the new shape automatically

Rejected because Seals and SealIDs are immutable and because automatic graph
reinterpretation would hide the compatibility boundary.

## Consequences

Good:

- Revision statements become contextual evidence attached to exact Cause use.
- One Cause target can name zero or more exact previous revisions without
  making the target Seal own those relations.
- Existing format-4 Seal bytes and IDs remain valid compatibility evidence.
- Explicit empty state and legacy-default state remain distinguishable.

Bad / Risk:

- Revision ancestry is no longer locally readable from a Seal alone.
- Different Cause Links may make inconsistent claims about one target.
- Existing revision-rooted algorithms and caches require coordinated redesign.
- A mixed legacy/new graph needs strict schema and observation rules to avoid
  order-dependent results.

Neutral:

- REF publication CAS remains mutable coordination outside Seal identity.
- Cause targets remain exact SealIDs and never become dynamic REF HEAD links.
- Source bindings and external Git/filesystem histories follow ADR 0024 and do
  not automatically create Cause-scoped Revision facts.
- Acceptance of this design does not authorize automatic relink, reseal,
  repair, batch publication, or migration.

## Implementation Notes

Implementation Action A-RL-001 is to freeze the derived observation and
conflict rules before changing canonical types.

Implementation Action A-RL-002 is to adopt an explicit new persisted format,
with fixed member order, exact schema identifiers, canonical fixtures,
duplicate/cycle tests, and compatibility tests for legacy fallback.

Implementation Action A-RL-003 is to update domain, canonical, revision,
graph, history, repository, migration, CLI, cache, inspection JSON, and fsck as
one vertical format slice. Partial reinterpretation of format-4 objects is not
acceptable.

The minimum compatibility matrix must cover:

- legacy Link to a legacy target with non-null `parent_revision`;
- legacy Link to a legacy initial target;
- new Link with an explicit empty list targeting a legacy Seal;
- new Link with an explicit list targeting a legacy Seal;
- new Link targeting a new-format Seal;
- duplicate, missing, self, and cyclic previous-revision identities;
- multiple Links making equal and unequal statements about one target.

## Review

- Specialist review: Pending derived-graph and canonical-format review.
- Non-specialist review: Pending explanation of contextual revision behavior.
- Root-chain review: Pending; C-RL-001 through C-RL-003 are owner-confirmed,
  while A-RL-001 through A-RL-003 remain gated.

## Evidence

- E-RL-001: Operator-provided Cause Link grammar and clarification on
  2026-08-28.
- E-RL-002: Format-4 canonical payload and `parent_revision` identity contract
  in `docs/storage-format.md` sections 4 and 5.
- E-RL-003: Accepted separate-edge semantics in ADR 0011 and the checked-in
  `internal/domain.SealPayload` and revision graph implementation.

## Follow-ups

- Resolve the eight remaining acceptance gates.
- Revise the Link Metadata proposal and ADR 0017 if link-scoped Revision data
  becomes a core field rather than opaque application metadata.
- Keep source-adapter selection and portable occurrence evidence governed by
  ADR 0024; neither resolves the remaining Revision aggregation gates.
- On acceptance, update the normative requirements, architecture, storage,
  CLI, integrations, and affected earlier-ADR precedence table together.
