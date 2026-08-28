# ADR 0023: Cause-scoped revision observation

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-CR-001 through C-CR-010
- Related Evidence: E-CR-001 through E-CR-007
- Pending Decision: owner acceptance of the exact candidate after fresh review
- Supersedes on acceptance: the intrinsic-parent revision semantics in ADRs
  0011, 0013, 0016, 0019, and 0020
- Superseded by: None

## Context

Format 4 stores one intrinsic `parent_revision` in every Seal. That makes a
revision edge a global property of the child Seal even when the relationship
being described is meaningful only to one dependent Seal and one Cause Link.
It also gives a Seal two graph roles: immutable material/provenance identity
and a globally asserted position in a revision history.

The required model is different. A Seal has no intrinsic parent or child.
Instead, each Cause Link may state which Seal or Seals are previous revisions
of that Link's exact target. Revision is therefore an assertion made by one
observing Cause Link, not a property stored in the target Seal.

This is an intentional semantic break. Even a successfully migrated format-4
repository changes from global parent meaning to Cause-Link-scoped meaning.
ADR 0026 defines the incompatible migration and reports old parent assertions
that no Cause Link observes; this ADR does not provide a legacy fallback.

## Decision

### Link-scoped assertion

In the format defined by ADR 0025, each Cause Link contains:

```text
cause_link := {
  target_seal: SealID,
  previous_revision_seal_of_target_seal: [SealID],
  messages: [string]
}
```

`target_seal` is one exact immutable Seal. The previous-revision array is the
containing Link's assertion about that exact target and no other target. The
array is required, sorted by full SealID, duplicate-free, and may be empty.
The Link and its assertion are identity-bearing Provenance Blob content.

This establishes C-CR-001: a Seal has no intrinsic parent or child relation.
Revision meaning exists only in an observed Cause Link assertion.

The assertion source is retained as the tuple:

```text
(observer_seal, observer_provenance, target_seal,
 previous_revision_set, messages)
```

Two observer Seals may assert different previous sets for the same target.
That is not canonical corruption and neither assertion overwrites the other.
Inspection must expose both sources.

### Empty, absent, and mixed observations

An explicit empty array means that this observer asserts no previous revision
for this occurrence of the target. It is not missing data and it contributes
no structural revision edge.

For one target `T`, inspection distinguishes:

```text
PREVIOUS_UNOBSERVED
  no observed Cause Link contains target T

PREVIOUS_NONE_ASSERTED
  at least one observed assertion for T is an empty set

PREVIOUS_ASSERTED
  at least one observed assertion for T contains one or more SealIDs
```

The last two states may both apply to `T`. Machine output returns the complete
per-observer assertion records and a derived structural edge union; it must not
collapse mixed empty and non-empty assertions into one purportedly unanimous
answer.

This establishes C-CR-002: absence, explicit no-previous, and asserted
previous revisions remain distinguishable.

### Coherent repository observation

Let `O` be one captured repository observation containing the sorted complete
map of every current REF name to its exact HEAD SealID. From `O`, derive the
least fixed point:

```text
H_O = deduplicated set of all HEAD SealIDs in O

V_0 = H_O
V_n+1 = V_n
        union every target_seal in Cause Links of Seals in V_n
        union every previous-revision SealID asserted by those Cause Links
V_O = the first V_n for which V_n+1 = V_n

C_O = every Cause edge (observer_seal -> target_seal) in V_O

E_O = union of every edge (target_seal -> previous_revision_seal)
      asserted by Cause Links in V_O
A_O = H_O plus every vertex reachable backward from H_O through E_O
```

All referenced Seal, Material, Provenance, content, and attachment Blobs must
validate while deriving the observation. Cause and revision relations are
validated together as one directed graph. A self-edge or cycle in either
relation or their combined traversal is corruption.

A Seal reached only as a Cause target or an unreferenced loose Blob is not
thereby an active revision. `A_O` is rooted only at current REF heads. This
retains the rule that object presence is not publication.

Any assertion in `V_O` can affect structural leafness, staleness, admission,
and history anywhere its edge is reachable. This repository-wide consequence
is deliberate, but every reported or enforced edge must retain its observer
source so an operator can audit who made the assertion.

This establishes:

- C-CR-003: graph answers are derived from one complete REF-head snapshot;
- C-CR-004: Link-local assertions are preserved individually while structural
  graph operations use their deterministic edge union; and
- C-CR-005: only current REF heads root the active revision graph.

### Snapshot discipline

Every operation whose answer or admissibility depends on `O`, `V_O`, `C_O`,
`E_O`, or `A_O` must:

1. capture the complete sorted REF-head map;
2. derive and validate the complete required closure without partial output;
3. buffer the result or prospective mutation;
4. recapture the complete sorted REF-name-to-head map immediately before
   emitting dependent output or committing the candidate/REF mutation;
5. require byte-for-byte equality of both maps, including REF names, presence,
   absence, and exact head IDs; and
6. fail with a retryable concurrent-observation error if they differ.

This applies to `show` when it includes derived relations, `log`, `linklog`,
`status`, `stale`, `impact`, `graph`, selectors, `fsck`, candidate mutation,
and seal admission. It is not limited to multi-REF write commands.

This establishes C-CR-006: no operation may combine graph facts from
incompatible repository observations.

### Leafness, history, and staleness

For `S` in `A_O`:

```text
children_O(S) = { T in A_O | (T -> S) is in E_O }
leaf_O(S)     = children_O(S) is empty
```

`log REF` begins at the REF head and follows asserted previous-revision edges.
Branching history is normal. Default deterministic traversal orders siblings
by full SealID. `--all-paths` retains distinct simple paths and their assertion
sources; it does not invent a preferred parent.

`linklog` compares Cause Link records between successive observed revisions.
It reports target, previous-revision assertion, messages, and observer identity
separately so differing assertions remain visible.

A dependent Seal is stale when a direct or transitive Cause target used by its
Provenance is not an active current revision leaf under the same `O`. Staleness
is derived and never persisted. Historical and draft inspection may show a
non-leaf target without repairing or rejecting the existing immutable Seal.

This establishes C-CR-007: history, leafness, and stale are projections of an
explicit coherent observation rather than target-Seal fields.

### Selector and comparison semantics

Define the observed revision closure of a selected Seal under one coherent
observation as:

```text
revision_closure_O(S) = S plus every Seal reachable from S by following E_O
```

`REF@HEX` resolves only within `revision_closure_O(head(REF))`. `HEX` is a
full SealID or an accepted unique lower-hex prefix. No match is not found;
multiple matches are ambiguous. The selector uses the structural assertion
union and returns the complete assertion-source records that justify every
traversed edge. It never chooses a preferred observer or previous revision.

For `REF@TOKEN`, a token of 4 through 64 lower-case hexadecimal characters is
the revision-closure form above; a non-hex token is an exact REF-scoped tag
lookup under ADR 0013. Repository-wide `@SEAL_TOKEN` remains unique canonical
Seal prefix lookup and does not assert revision membership. These forms are
syntactically disjoint and never fall back from one meaning to another.

Immutable comparison requires exactly two explicit selectors:

```text
sealgraph compare LEFT RIGHT
```

It compares the exact selected Seal, Material, Provenance, and derived graph
records. The old one-selector form is removed and fails with
`SECOND_SELECTOR_REQUIRED`; no previous revision is inferred.

`candidate compare REF` compares the prospective Candidate projection with its
exact `expected_ref_head` CAS baseline. Output labels that Seal
`PUBLICATION_BASELINE`, never `parent` or `previous revision`. When
`expected_ref_head` is null, output reports `BASELINE_ABSENT` and the complete
prospective state without inventing empty material. A caller that wants to
compare against a historical Seal must supply that Seal explicitly to the
two-selector immutable comparison surface.

All selector resolution and comparison output follow the complete snapshot
discipline above. This establishes C-CR-010: Cause-scoped revision has no
implicit comparison predecessor; historical membership uses the observed edge
union and publication comparison uses only the explicit CAS baseline.

### Candidate admission scope

Normal non-draft publication evaluates only the prospective candidate's exact
Cause closure:

```text
Q_candidate = the candidate's direct target Seals plus every transitive
              Cause target reached from them
```

The prospective repository observation, including the candidate's Link-local
revision assertions, must be structurally valid. Every target in
`Q_candidate` must be a non-draft leaf in the prospective active graph. A stale
or non-leaf current head outside `Q_candidate` does not block this publication.

A root has no Cause Links. Mentioning a Seal only in
`previous_revision_seal_of_target_seal` does not satisfy the Cause requirement
and does not make that Seal a dependency.

Draft and explicit historical workflows may preserve non-leaf dependencies,
but they must label that policy in output and may not present it as normal
HEAD-consistent publication.

This establishes C-CR-008: admission is strict for the candidate's Cause
closure without turning unrelated repository state into a global write lock.

### Authoring contract

Link authoring resolves every selector to an exact SealID before candidate
persistence. The semantic operations are:

```text
add Cause Link with one exact target
set zero or more previous revisions for that exact target
set zero or more Link messages
```

The public CLI must provide an explicit way to repeat previous-revision input
and an explicit no-previous form. If neither is supplied for a new Link, the
candidate stores the required empty array. No reader or writer consults a
format-4 `parent_revision` as a default.

One Provenance contains at most one Cause Link for one exact target. Repeated
message inputs form the sorted duplicate-free `messages` array; repeated
previous-revision inputs form the corresponding sorted duplicate-free SealID
array. An exact target therefore has one canonical observer-local assertion.

Format 5 removes intrinsic-parent authoring. `derive` and `add --parent` are
not accepted as deprecated aliases because they would suggest a global parent.
A copy-only candidate seed operation may copy Material and Provenance for
editing, but it creates no revision assertion by itself.

This establishes C-CR-009: authoring requires an exact Link target and makes
the scope of every revision assertion explicit.

### Decision precedence

On acceptance:

- ADR 0010's stale/frontier intent remains, but its revision edges are derived
  from the observation defined here.
- ADR 0011's REF independence, exact Cause targets, immutable Seals, and
  active-head roots remain; intrinsic `parent_revision` and single-parent
  algorithms are superseded.
- ADR 0013 retains REF manifests, scoped tags, and atomic move; all validation
  of their targets uses the new Seal/Provenance model.
- ADR 0015 retains inspection safety and deterministic output; its history
  projection follows this ADR.
- ADR 0016 retains integrity and fail-closed behavior; its parent-closure
  checks become observation-derived assertion checks.
- ADR 0017 may add generic metadata only through a later accepted schema; it
  may not duplicate or reinterpret the core revision assertion.
- ADR 0019 retains local source bindings but loses candidate-global parent
  state.
- ADR 0020 retains native operation language except `derive`, one-selector
  compare, parent-based Candidate comparison, and other intrinsic-parent
  wording superseded here. The explicit two-selector and publication-baseline
  contracts in this ADR replace them.
- ADR 0021's no-attachment-mutation decision remains; ADR 0025 decides the
  new Material Blob representation and ADR 0026 decides migration.
- ADR 0024 remains unchanged: external Git or filesystem history is evidence,
  never an automatic Cause or revision assertion.

## Alternatives

### Keep intrinsic `parent_revision`

Rejected because the relation would remain global to the child Seal and could
not express observer-specific revision meaning.

### Retain format-4 parent as a default when a Cause Link omits previous data

Rejected. It creates two authorities and makes an omitted Link field depend on
the target's storage generation. The incompatible conversion belongs solely to
ADR 0026.

### Require one authoritative previous set per target across the repository

Rejected because it would restore a global property indirectly and make two
valid independent observations mutually corrupting.

### Give empty and absent assertions the same meaning

Rejected because an explicit no-previous claim is evidence and must remain
distinguishable from no observer.

### Activate every Cause-reachable Seal as a revision head

Rejected because a dependency occurrence is not publication. Only current REF
heads establish active revision roots.

## Consequences

Good:

- Seal identity no longer carries an intrinsic place in one global history.
- Different dependent contexts can express different revision assertions.
- Exact observer provenance remains available when unioned graph edges affect
  repository-wide results.
- Candidate admission has an explicit, bounded quantifier.
- External source history and Sealgraph revision meaning stay separated.
- Historical selection and comparison no longer depend on an implicit parent.

Bad / Risk:

- The same target may have conflicting assertions; tools must retain and show
  all sources rather than provide a single parent field.
- One observed assertion can change leafness or stale results for other REFs.
- All graph-dependent operations must validate and revalidate a complete head
  snapshot, increasing read cost.
- Format-4 parent semantics cannot survive automatically when no Cause Link
  observes the child.
- Callers of one-selector `compare` must choose an explicit second Seal.

Neutral:

- This ADR defines semantic graph rules, not the physical Blob schema or
  migration encoding.
- Acceptance does not mutate a repository, implement a CLI, or accept ADRs
  0025 and 0026.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-CR-001 implement assertion-aware graph model | ADR 0023 | C-CR-001 through C-CR-005 | canonical mixed-observer, empty/non-empty, branch, cycle, and inactive-object fixtures |
| A-CR-002 add observation transaction | ADR 0023 | C-CR-003, C-CR-006 | deterministic concurrent-head mutation tests for every graph-dependent command family |
| A-CR-003 replace history/stale/admission algorithms | ADR 0023 | C-CR-007, C-CR-008 | worked multi-REF tests proving candidate-scoped admission and repository-wide edge effects |
| A-CR-004 replace parent authoring surfaces | ADR 0023 and ADR 0025 | C-CR-009 | CLI contract, candidate canonicalization fixtures, and rejection tests for removed parent vocabulary |
| A-CR-005 expose assertion sources | ADR 0023 | C-CR-002, C-CR-004, C-CR-007 | stable human and JSON fixtures for unobserved, empty, asserted, and mixed states |
| A-CR-006 replace selector and comparison contracts | ADR 0023 | C-CR-006, C-CR-007, C-CR-010 | scoped closure, tag/hex disambiguation, ambiguity, two-selector, absent-baseline, and concurrent-REF-map fixtures |

No action may start from this Proposed ADR alone. Acceptance of the exact ADR,
then the corresponding storage and migration ADRs, is the implementation gate.

## Review

Earlier ADR 0023 candidates failed three-scope review because they did not
define the candidate-admission quantifier, mixed assertion output, complete
snapshot discipline, contextual assertion consequences, supersession scope,
migration collision behavior, or the successor to parent-based selector and
comparison contracts precisely enough. This revision addresses those design
mechanisms, removes the rejected legacy fallback, and defines explicit
revision-closure selection, two-selector immutable comparison, and the
publication CAS comparison baseline.

A fresh three-scope review of these exact bytes is still required. Review PASS
does not itself change `Proposed` to `Accepted`.

## Evidence

- E-CR-001: ADR 0011 and format 4 establish the current global
  `parent_revision` mechanism being replaced.
- E-CR-002: ADRs 0010, 0015, and 0016 show that revision edges affect stale,
  inspection, and repository integrity across command families.
- E-CR-003: ADR 0024 establishes that Git commit/file history is not native
  Seal revision authority.
- E-CR-004: the operator specified the Cause Link record as a target Seal plus
  a target-scoped previous-revision Seal array.
- E-CR-005: the operator confirmed that old parent meaning not observed from a
  Cause Link may be lost, provided migration warns about the change.
- E-CR-006: the prior independent three-scope review identified the quantified
  admission, mixed assertion, observation, precedence, and migration gaps
  summarized in the Review section.
- E-CR-007: the operator directed that selector and comparison behavior be
  redefined for the new semantics instead of retaining parent-based defaults.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-CR-001 | E-CR-001, E-CR-004 | A-CR-001 |
| C-CR-002 | E-CR-004, E-CR-006 | A-CR-005 |
| C-CR-003 | E-CR-002, E-CR-006 | A-CR-001, A-CR-002 |
| C-CR-004 | E-CR-004, E-CR-006 | A-CR-001, A-CR-005 |
| C-CR-005 | E-CR-001, E-CR-002 | A-CR-001 |
| C-CR-006 | E-CR-002, E-CR-006 | A-CR-002 |
| C-CR-007 | E-CR-001, E-CR-002 | A-CR-003, A-CR-005 |
| C-CR-008 | E-CR-002, E-CR-006 | A-CR-003 |
| C-CR-009 | E-CR-003, E-CR-004 | A-CR-004 |
| C-CR-010 | E-CR-002, E-CR-007 | A-CR-006 |

## Follow-ups

- Accept or revise ADR 0025's exact Universal Blob canonical schema.
- Accept or revise ADR 0026's deterministic incompatible migration contract.
- Run a fresh three-scope ADR review against this exact candidate digest.
- After acceptance, align normative requirements, storage, CLI, architecture,
  integration documentation, and implementation plan in one change set.
