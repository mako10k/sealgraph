# ADR 0023: Cause-scoped revision observation

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-CR-001 through C-CR-013
- Related Evidence: E-CR-001 through E-CR-012
- Pending Decision: owner acceptance of the exact candidate after fresh review
- Supersedes on acceptance: the ancestry-based Link-repoint inference in ADR
  0006; the intrinsic-parent revision semantics and parent-based selector,
  comparison, and impact rules in ADRs 0011, 0013, 0016, 0019, and 0020
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
map of every current REF name to its exact canonical manifest bytes. Each
manifest includes the exact HEAD SealID and complete scoped-tag map. From the
decoded HEAD values in `O`, derive the least fixed point:

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

- C-CR-003: graph answers are derived from one complete canonical REF-manifest
  snapshot, including HEADs and scoped tags;
- C-CR-004: Link-local assertions are preserved individually while structural
  graph operations use their deterministic edge union; and
- C-CR-005: only current REF heads root the active revision graph.

### Snapshot discipline

Every operation whose answer or admissibility depends on `O`, `V_O`, `C_O`,
`E_O`, or `A_O` must:

1. capture the complete sorted REF-name-to-canonical-manifest-byte map;
2. derive and validate the complete required closure without partial output;
3. buffer the result or prospective mutation;
4. recapture that complete manifest map immediately before emitting dependent
   output or committing the candidate/REF mutation;
5. require byte-for-byte equality of both maps, including REF names, presence,
   absence, exact HEADs, and complete tag arrays; and
6. fail with a retryable concurrent-observation error if they differ.

Every supported Candidate or canonical-manifest mutation surface retains ADRs
0011 and 0018's one repository-wide process writer guard. A graph-dependent
mutation acquires that guard before its first capture of `O` and holds it
through the final byte-equality check, journal preparation when applicable,
the one-REF CAS or other already accepted manifest transaction, and the
resulting publication classification. Every native and Git-sidecar manifest
writer must participate in the same guard; a command may not implement a
private per-REF substitute.
The expected-old CAS remains the one-REF publication linearization point, while
the guard prevents two different REF publications from both being admitted
against one old repository graph. Runtime lock bytes remain non-canonical.
Concurrent direct filesystem or outer-VCS rewriting that does not participate
in the guard is outside the supported mutation protocol and must never be
treated as a successful serializable Sealgraph write.

Repository-wide `@SEAL_TOKEN` prefix resolution has one additional observation
component. Before resolving it, the operation captures `I_O`, the sorted
complete set of every valid physical loose-object ID. It resolves uniqueness
across all of `I_O` and then requires the unique object to decode as a canonical
format-5 Seal Blob, preserving ADR 0006's object-name-prefix domain. It buffers
the dependent result, recaptures the same complete loose-object inventory
immediately before output or mutation, and requires exact equality. An object
present in only one of the two captured inventories causes the same retryable
concurrent-observation failure whether or not it changes prefix uniqueness.
Operations that do not use a repository-wide Seal prefix do not scan `I_O`
merely to validate graph facts; `fsck` has the separate complete physical
observation below.

`fsck` captures `P_O`, the complete sorted physical canonical-namespace
observation. It has a fixed `ROOT` slot for the `.sealgraph` repository entry, a
fixed `CONFIG` slot, and a namespace map containing every entry at or below the
canonical `objects` and `refs` roots, including those roots, the `refs/seals`
root, intermediate directories, invalid names, and unexpected entries. `ROOT`
contains its `lstat` kind and mode. `CONFIG` is explicitly `ABSENT` or contains
its `lstat` kind, mode, and, when regular, captured exact-byte length and SHA-256.
Namespace-map keys are raw path bytes relative to `.sealgraph`; values contain
`lstat` kind and mode and, for a regular file, a captured exact-byte length and
SHA-256. Directories contain no byte digest. Symlinks and special entries are
recorded by kind and never followed or opened.

Permission bits are observation fields and explicit init/load/writer
postconditions, not canonical identity or stable integrity authority. A stable
writable mode restored by an outer Git checkout does not by itself make exact
canonical bytes corrupt. During one `fsck` invocation, however, a mode value
that differs between the two `P_O` captures is a concurrent-observation
failure. Wrong entry kinds, symlinks, special files, unreadability, invalid
layout, or invalid bytes remain integrity or operational failures. `fsck`
reports but never changes modes.

`O` is the validated logical REF-manifest projection of the same captured REF
namespace; it does not replace the physical REF/config portion of `P_O`. The
command captures both `O` and `P_O`, validates config, every physical canonical
layout entry, envelope, BlobID, typed role, reference, REF manifest, tag, and
combined graph, buffers the complete result, then recaptures both observations
immediately before output. Any added or removed entry, or any change to a
captured path, kind, mode, length, or digest for config, object, REF, root, or
intermediate entries, fails retryably without success JSON. A physical
replacement that preserves the complete recorded tuple is observationally
equal and need not be distinguished; `P_O` deliberately contains no inode or
other non-portable entry identity. A change-and-restore between captures that
also preserves the complete recorded tuple is likewise observationally equal;
neither `O`, `I_O`, nor `P_O` claims continuous monitoring. An absent config or
invalid path, entry, manifest, or Blob prevents success; any difference it
contributes between captured tuples must cause failure rather than be omitted.

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

`log REF` begins at exact REF head `H` and follows asserted
previous-revision edges. Branching history is normal. Let:

```text
G_O(H) = every distinct Seal reachable from H through zero or more E_O edges
depth_H(S) = the minimum revision-edge count from H to S
```

Default history emits every member of `G_O(H)` exactly once in
`(depth_H(S), full S SealID)` order. Each entry includes the exact Seal view,
its complete target-revision observation, and every outgoing structural
revision edge with all sorted supporting assertion sources. A shared ancestor
is one entry even when several branches reach it; no traversal algorithm,
filesystem order, or preferred predecessor changes membership or ordering.

Let a history leaf be a member of `G_O(H)` with no outgoing `E_O` edge to
another member. `--all-paths` retains exactly every maximal leaf-terminated
path `[H, ..., L]` whose adjacent pairs are `E_O` edges and whose final member
`L` is a history leaf. When `H` is itself a leaf, the exact path set is the one
zero-edge path `[H]`. Validated acyclicity makes every such path simple. Shared
prefixes remain repeated in their distinct complete paths, and no non-maximal
prefix is a separate path. Assertion sources remain attached to each adjacent
edge; no preferred parent is invented. Exact path order, bounded enumeration,
truncation, and machine representation are defined by ADR 0027.

`linklog REF` also begins at the exact REF head, but it compares each reachable
structural revision edge rather than choosing one predecessor. Let:

```text
L_O(H) = every distinct edge (T -> P) in E_O reachable from head H
depth_H(T) = the minimum revision-edge count from H to T
```

One comparison record is emitted for every edge in `L_O(H)`, ordered by
`(depth_H(T), full T SealID, full P SealID)`. An edge reached through more than
one branch is emitted once. Each record names `T` as the newer side, `P` as the
previous side, and includes every sorted assertion-source tuple supporting the
edge. A child with multiple previous revisions therefore produces one
independent comparison per outgoing revision edge; there is no synthetic
combined baseline and no preferred predecessor.

The edge record also includes every observed assertion record whose exact
target is `T`, including empty sets and assertions of different previous
revisions. Supporting sources and complete target observations are separate
arrays, so mixed observer claims remain visible without treating an empty or
different assertion as support for `(T -> P)`.

For each `(T -> P)`, the command compares the exact Cause Link records in the
Provenance of `P` and `T`, keyed by `target_seal`. Output contains one entry per
target whose record differs, ordered by full target SealID, with nullable exact
`before` and `after` records. An absent `before` is an addition, an absent
`after` is a removal, and two present records expose changes to the exact
previous-revision set and/or messages. A changed target identity is one removal
plus one addition. Format 5 does not infer a repoint from ancestry, because a
branching observed graph does not define a unique old/new target pairing.

Human output displays the structural revision edge, all supporting observers,
and the Cause-record differences. Machine output uses
`sealgraph/linklog/v2`, preserves full IDs and complete assertion sources, and
uses the same edge and target ordering. Empty history succeeds with an empty
record set. Complete graph validation and snapshot revalidation occur before
either output format.

A dependent Seal is stale when a direct or transitive Cause target used by its
Provenance is not an active current revision leaf under the same `O`. Staleness
is derived and never persisted. Historical and draft inspection may show a
non-leaf target without repairing or rejecting the existing immutable Seal.

The exact classification and review frontier are:

```text
direct_stale_O(H) = { D | (H -> D) is in C_O and D is not an active leaf }

transitive_stale_paths_O(H) =
  empty, when direct_stale_O(H) is non-empty
  otherwise every first-stale Cause path [D_0, ..., D_n], n >= 1, where
    (H -> D_0) is in C_O,
    D_0 through D_(n-1) are active leaves,
    every adjacent pair is in C_O, and
    D_n is not an active leaf

S_O = { r | r has current head H and
              (H is an active non-leaf
               or direct_stale_O(H) is non-empty
               or transitive_stale_paths_O(H) is non-empty) }
Q_O = { HEAD(r) | r is in S_O }
CausePlus_O(H) = strict transitive closure from H through C_O
F_O = { r in S_O | CausePlus_O(HEAD(r)) intersects Q_O at no Seal }
```

`stale` selects `S_O`; `stale --frontier` selects `F_O`. Multiple REF aliases
of the same stale head are either all selected or all omitted: an alias does
not block another alias because the closure is strict and the combined graph
is acyclic. A clean or unselected sibling tip, a revision edge, and Candidate
state do not affect frontier membership. Direct and transitive Cause labels are
therefore mutually exclusive for one current head, while self-stale may coexist
with either Cause classification.

This establishes:

- C-CR-007: history, leafness, and stale are projections of an explicit
  coherent observation rather than target-Seal fields; and
- C-CR-012: branching `linklog` compares every unique observed revision edge
  against its exact two endpoint Provenances without preferred-parent or
  repoint inference and retains all assertion sources; and
- C-CR-013: default branching `log` is a unique-node, minimum-depth-ordered
  projection with complete edge and assertion-source evidence.

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

For `REF@TOKEN`, a token matching `[0-9a-f]{4,64}` is the revision-closure form
above. Every other token that is a valid TAGNAME is an exact REF-scoped tag
lookup under ADR 0013. A token that satisfies neither grammar fails with
`INVALID_SELECTOR_TOKEN`. Thus a shorter or longer all-hex token is a tag when
it is otherwise a valid TAGNAME; the two selector branches are exact
complements within the valid input domain and never fall back after lookup.
Repository-wide `@SEAL_TOKEN` accepts only `[0-9a-f]{4,64}`, resolves uniquely
against `I_O`, and does not assert revision membership.

Format 5 retains ADR 0006's TAGNAME reservation: a raw TAGNAME matching
`[0-9a-f]{4,64}` is invalid. Format-4 exporters and format-5 manifest readers
reject such a name as invalid canonical state rather than retaining an
unaddressable tag. No forced-tag selector or hex-to-tag fallback is added.

Immutable comparison requires exactly two explicit selectors:

```text
sealgraph compare LEFT RIGHT
```

It compares the exact selected immutable Seal, Material, and Provenance views,
including each Provenance's complete canonical Cause Link array. Derived graph
state, REF aliases, revision observations, and assertion-source unions are not
comparison fields; commands expose those observation-relative records through
`show`, `graph`, `log`, `linklog`, and `impact`. The old one-selector form is
removed and fails with
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

### Impact semantics and assertion-source filtering

`impact` keeps Cause reachability and revision ancestry as separate relations.
For assertion filtering, define the complete observed assertion records and
their filtered structural edge set:

```text
B_O = every assertion record in Cause Links of observer Seals in V_O

E_O(*) = E_O

E_O(F) = { (target_seal -> previous_revision_seal)
           | an assertion in B_O has observer_seal in F
             and contains previous_revision_seal }

revision_sources_O,F(h) = h plus every Seal reachable from h through E_O(F)
```

`*` means no observer filter. `F` is a non-empty deduplicated set of exact
observer SealIDs. Filtering is by `observer_seal`, not ProvenanceID; one
Provenance Blob may be reused and does not merge the identity of distinct
observer Seals.

The public command is:

```text
sealgraph impact [--asserted-by OBSERVER_SELECTOR]...
                 [--all-paths] [--max-paths N] SELECTOR
```

With no `--asserted-by`, impact uses `E_O(*)`, the structural union of every
observed assertion. Repeated `--asserted-by` options resolve to exact observer
SealIDs under the ordinary unfiltered selector rules and combine as set union.
The observer selectors and the source `SELECTOR` are resolved against the same
coherent observation before filtering. A resolved observer not in `V_O` fails
with `ASSERTION_OBSERVER_NOT_OBSERVED`; the command does not silently degrade
to exact-source-only impact.

The observer filter changes only which revision assertions can extend
`revision_sources_O,F(h)`. It does not filter `C_O`, current REF heads, Cause
paths, or the exact selected source `h`. Reaching `h` therefore never requires
a revision assertion. Restricting the set of downstream current heads would be
a different query and is not implied by `--asserted-by`.

Filtering does not weaken validation. The command validates complete `B_O`,
`C_O`, and `E_O` and their combined graph before applying `F` to result
derivation; a cycle, corrupt assertion outside `F`, or unreadable referenced
Blob still fails the complete query without plausible partial output.

For a distinct current-head Seal `d`, an impact path is:

```text
p = (v_0 = d, v_1, ..., v_k = m), k >= 1

for every i < k: (v_i -> v_i+1) is in C_O
m is in revision_sources_O,F(h)
no v_i for 0 < i < k is in revision_sources_O,F(h)
```

The last rule stops one physical Cause route at its first matching source
revision after traversal begins. `v_0` is deliberately excluded from that
test: membership still requires `k >= 1`, so a zero-edge match never creates
impact, while a current head that is itself in the source set may be impacted
when one of its strict Cause paths first reaches another source revision. The
impacted membership is:

```text
impact_O,F(h) = { d in H_O | d != h and at least one impact path exists }
```

Membership is per distinct current-head Seal; sorted REF aliases are
presentation annotations. A different current head that is also in the source
revision set is not included merely by a zero-edge match, but may be included
when one of its strict Cause paths reaches the source set. The selected `h`
itself is always excluded as a downstream result.

Impact summaries are ordered by full downstream SealID and each alias list is
ordered by raw REF bytes. Resolved observer filters are ordered by full SealID.
Assertion records use the tuple order `(observer_seal, observer_provenance,
target_seal, previous_revision_set, messages)`, with both arrays compared
lexicographically by their already canonical element order.

Default presentation emits one Cause path per impacted Seal. The shortest
Cause-edge count wins; equal lengths use the bytewise sequence of full SealIDs.
`--all-paths` enumerates distinct simple first-match Cause paths in that same
order. `--max-paths N` is valid only with `--all-paths`, is positive, defaults
to 100, and limits presentation separately for each impacted Seal. It never
limits membership, complete graph validation, corruption or cycle detection,
or snapshot revalidation.

Every emitted match names its exact `matched_revision` and includes one
deterministic revision proof from `h` to the match using `E_O(F)`: the shortest
revision-edge path wins, with the bytewise full-SealID sequence as the
tie-breaker. A direct match on `h` has the explicit zero-edge proof `[h]`.
Every nonzero proof edge carries all complete assertion-source records that
support that edge within the active scope. For an unfiltered query those are
all observed supporting sources; for a filtered query they are only supporting
sources whose `observer_seal` is in `F`. Assertion records outside `F` are
excluded evidence, not evidence of absence.

Human and machine output always identify the revision assertion scope as
`ALL_OBSERVED` or `FILTERED`. Filtered machine output includes the sorted full
observer SealIDs; human output uses the unambiguous display prefixes required
by ADR 0022. Machine output uses a new `sealgraph/impact/v2` schema and includes:

- the assertion scope and resolved observer filter;
- the exact source Seal and matched revision for every emitted Cause path;
- the Cause path and deterministic revision proof as separate relations;
- the complete in-scope assertion records targeting every Seal on an emitted
  revision proof, including its zero-edge source and empty or non-supporting
  assertions, so mixed observation remains visible; and
- an explicit truncation marker when more Cause paths exist.

For each such target, impact output reports an `in_scope_previous_states`
array using `ASSERTION_SCOPE_UNOBSERVED`, `ASSERTION_SCOPE_NONE_ASSERTED`, and
`ASSERTION_SCOPE_ASSERTED`. Unobserved is exclusive; none-asserted and asserted
may both appear, in that fixed order. These are explicitly filter-relative
annotations and do not redefine the repository-wide `PREVIOUS_UNOBSERVED`,
`PREVIOUS_NONE_ASSERTED`, and `PREVIOUS_ASSERTED` states above. A selected
observed observer with no applicable assertion is therefore a successful
scoped observation, not the same condition as an observer selector outside
`V_O`.

For example, let observed assertion sources contain:

```text
D1 asserts T -> P
D2 asserts P -> Q
```

and consider current heads whose first matching Cause paths are `X -> Q`,
`Y -> P`, and `Z -> T`. Their scoped results are:

```text
no filter:                 revision sources {T, P, Q}; X, Y, and Z match
--asserted-by D1:          revision sources {T, P};    Y and Z match
--asserted-by D2:          revision sources {T};       Z matches
--asserted-by D1 and D2:   revision sources {T, P, Q}; X, Y, and Z match
```

The unfiltered proof from `T` to `Q` is visibly stitched from the `D1` and `D2`
assertions. `D2` alone cannot use `P -> Q` when its filtered closure from `T`
cannot first reach `P`. An observed `D3` assertion `T -> []` yields source set
`{T}` with `ASSERTION_SCOPE_NONE_ASSERTED`; an observed `D4` with no assertion
about `T` yields the same source set with `ASSERTION_SCOPE_UNOBSERVED`.

The implementation may store disposable adjacency indexes from target to
revision edges and assertion sources and from target to dependent observers.
The index binds to the complete REF-head observation digest and cache checksum;
an observer filter is applied by source-set intersection and does not create a
new canonical or per-filter graph. Cache hit, miss, or bypass must produce the
same membership and evidence. Default membership and one shortest path can be
derived by revision closure plus multi-source reverse Cause traversal. All-path
enumeration remains potentially combinatorial but only its presentation is
bounded.

This establishes C-CR-011: impact is conservative structural Cause
reachability over the all-observer revision union by default, may explicitly
restrict only the revision assertion sources, and always exposes enough scope
and source evidence to audit mixed-observer or stitched revision paths.

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

This establishes C-CR-008: semantic admission is strict for the candidate's
Cause closure without requiring unrelated heads to be clean; the orthogonal
repository writer guard supplies serializable publication rather than changing
that semantic quantifier.

### Authoring contract

Link authoring resolves every selector to an exact SealID before candidate
persistence. One authoring operation names exactly one target and replaces the
complete observer-local Cause Link record for that target. The semantic
operation is:

```text
set Cause Link for one exact target
  with one explicit previous-revision set
  and one explicit message set
```

The public CLI defined by ADR 0027 requires exactly one of repeatable previous-
revision input or an explicit no-previous form. It permits repeatable messages,
uses an empty message array when none are supplied, and does not combine
positions or take a cross-product over multiple targets. No reader or writer
consults a format-4 `parent_revision` as a default.

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

- ADR 0010's factual stale intent and stable `--refs-only` protocol remain. ADR
  0011's later exact-Seal `S_O`/`Q_O`/`F_O` frontier replacement remains
  authoritative, with `CausePlus_O` evaluated over this ADR's `C_O` and the REF
  alias behavior made explicit here.
- ADR 0006 retains full IDs, immutable tags, TAGNAME grammar, and Link
  messages. Its ancestry-based `linklog` repoint inference is superseded;
  format 5 reports exact Cause Link removal and addition instead.
- ADR 0011's REF independence, exact Cause targets, immutable Seals, and
  active-head roots remain; intrinsic `parent_revision`, single-parent
  algorithms, and parent-ancestor impact membership are superseded.
- ADR 0013 retains REF manifests, scoped tags, and atomic move; all validation
  of their targets uses the new Seal/Provenance model, and the reserved
  lower-hex TAGNAME range remains unchanged.
- ADR 0015 retains inspection safety and deterministic output; its history
  projection follows this ADR, while ADR 0027 defines format-5 successor JSON
  schemas.
- ADR 0016 retains fail-closed, mode-neutral integrity behavior; its
  parent-closure checks become observation-derived assertion checks.
- ADR 0017 may add generic metadata only through a later accepted schema; it
  may not duplicate or reinterpret the core revision assertion.
- ADR 0019 retains local source bindings but loses candidate-global parent
  state.
- ADR 0020 retains native operation language except `derive`, one-selector
  compare, parent-based Candidate comparison, format-4 Link option grouping,
  and other intrinsic-parent wording superseded here. The explicit two-
  selector and publication-baseline contracts in this ADR and the one-target
  authoring grammar in ADR 0027 replace them.
- ADR 0022 retains terminal/non-terminal selection, safe human presentation,
  exact-byte modes, and narrow receipts. ADR 0027 defines the format-5 JSON
  version transition and exact successor structures.
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

### Require one observer-specific impact result with no union default

Rejected because Link scope alone does not define how one observer's assertion
should propagate into unrelated downstream Cause closures. It would replace a
conservative repository fact with an implicit viewpoint choice. The default
therefore uses the structural union, while `--asserted-by` makes a narrower
viewpoint explicit and auditable.

### Apply `--asserted-by` to Cause traversal and downstream heads

Rejected because the option selects Revision assertion evidence, not a
downstream subgraph. Filtering `C_O` by the same value would conflate the Seal
that made a revision assertion with the current Seal whose Cause dependency is
being inspected. A separate future query may bound downstream roots if a real
workflow requires it.

### Compare branching `linklog` against one selected predecessor

Rejected because selecting one predecessor would invent a preferred branch
that no observed assertion establishes. Each structural revision edge is a
separate exact comparison boundary.

### Infer `linklog` repoints from revision ancestry

Rejected because branching and mixed-observer revision assertions can produce
more than one plausible old/new target pairing. Exact removal and addition are
auditable and deterministic; a domain-specific pairing belongs in a separate
query layer.

### Filter assertions by ProvenanceID instead of observer SealID

Rejected because one immutable Provenance Blob may be reused by more than one
Seal. Observer identity is the exact Seal occurrence that places the
Provenance in the observed graph; Provenance identity remains a source
annotation, not the filter key.

### Give empty and absent assertions the same meaning

Rejected because an explicit no-previous claim is evidence and must remain
distinguishable from no observer.

### Activate every Cause-reachable Seal as a revision head

Rejected because a dependency occurrence is not publication. Only current REF
heads establish active revision roots.

### Add inode or filesystem generation identity to `P_O`

Rejected because such identity is not portable across filesystems, outer-Git
checkouts, or staged/commit-tree views and is unnecessary for deterministic
Sealgraph output. Equality of the complete path/kind/mode/length/digest tuple is
the supported observation boundary; a physically replaced but tuple-identical
entry is the same observation.

## Consequences

Good:

- Seal identity no longer carries an intrinsic place in one global history.
- Different dependent contexts can express different revision assertions.
- Exact observer provenance remains available when unioned graph edges affect
  repository-wide results.
- Candidate admission has an explicit, bounded quantifier.
- External source history and Sealgraph revision meaning stay separated.
- Historical selection and comparison no longer depend on an implicit parent.
- Impact defaults to the conservative all-observer revision union while an
  explicit observer filter can inspect a narrower asserted history without
  changing the Cause traversal domain.
- Physical observation detects every difference in its portable tuple without
  claiming non-portable entry identity or treating stable mode as corruption.

Bad / Risk:

- The same target may have conflicting assertions; tools must retain and show
  all sources rather than provide a single parent field.
- One observed assertion can change leafness or stale results for other REFs.
- All graph-dependent operations must validate and revalidate a complete head
  snapshot, increasing read cost.
- Format-4 parent semantics cannot survive automatically when no Cause Link
  observes the child.
- Callers of one-selector `compare` must choose an explicit second Seal.
- Default impact may stitch a revision path from assertions made by different
  observer Seals; output must make every supporting observer explicit.
- Branching histories can produce more `linklog` comparison records than the
  old single-parent history, because every distinct revision edge is retained.

Neutral:

- This ADR defines semantic graph rules, not the physical Blob schema or
  migration encoding.
- Acceptance does not mutate a repository, implement a CLI, or accept ADRs
  0025, 0026, or 0027.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-CR-001 implement assertion-aware graph model | ADRs 0023, 0025, 0026, and 0027 | C-CR-001 through C-CR-005 | canonical mixed-observer, empty/non-empty, branch, cycle, and inactive-object fixtures |
| A-CR-002 add observation transaction | ADRs 0023, 0025, 0026, and 0027 | C-CR-003, C-CR-006 | deterministic concurrent-HEAD, tag-only, manifest-presence, cross-REF write-skew, writer-guard participation, complete repository-root/config/object/REF physical-namespace tuple mutation, stable writable-mode acceptance, observationally equal replacement and change-and-restore, and repository-wide loose-object-prefix inventory mutation tests for each dependent command family |
| A-CR-003 replace history/stale/admission algorithms | ADRs 0023, 0025, 0026, and 0027 | C-CR-007, C-CR-008, C-CR-013 | worked multi-REF tests proving candidate-scoped admission, repository-wide edge effects, direct/transitive exclusivity, exact-Seal frontier aliases/self-stale/branch/downstream membership, unique-node minimum-depth log order, maximal leaf-path membership including the zero-edge path, shared-ancestor deduplication, and complete source evidence |
| A-CR-004 replace parent authoring surfaces | ADRs 0023, 0025, 0026, and 0027 | C-CR-009 | one-target whole-record CLI contract, candidate canonicalization fixtures, and rejection tests for removed parent vocabulary and ambiguous option grouping |
| A-CR-005 expose assertion sources | ADRs 0023, 0025, 0026, and 0027 | C-CR-002, C-CR-004, C-CR-007, C-CR-011, C-CR-012 | stable human and JSON fixtures for unobserved, empty, asserted, and mixed states |
| A-CR-006 replace selector and comparison contracts | ADRs 0023, 0025, 0026, and 0027 | C-CR-006, C-CR-007, C-CR-010 | scoped closure, regex-complement tag/hex disambiguation, short/long all-hex tags, ambiguity, two-selector, absent-baseline, concurrent-manifest, and prefix-inventory fixtures |
| A-CR-007 redefine impact and filtered evidence | ADRs 0023, 0025, 0026, and 0027 | C-CR-004, C-CR-006, C-CR-011 | all-observer, single/multiple-observer, stitched-chain, empty/unobserved, deterministic proof, cache-hit/canonical-scan equivalence, and bounded-path fixtures for human and `sealgraph/impact/v2` output |
| A-CR-008 replace branching linklog | ADRs 0023, 0025, 0026, and 0027 | C-CR-004, C-CR-006, C-CR-012 | multi-previous, shared-edge deduplication, observer-source, exact Cause-record before/after, no-repoint, deterministic order, target-filter membership, and `sealgraph/linklog/v2` fixtures |

ADRs 0023, 0025, 0026, and 0027 must be reviewed as one exact decision set and
accepted by the operator before any format-5 implementation action or
normative-document conversion. Action tables in all four ADRs repeat that same
execution prerequisite; narrower Claim ownership never authorizes an earlier
slice.

## Review

Earlier ADR 0023 candidates failed three-scope review because they did not
define the candidate-admission quantifier, mixed assertion output, complete
snapshot discipline, contextual assertion consequences, supersession scope,
migration collision behavior, or the successors to parent-based selector,
comparison, and impact contracts precisely enough. This revision addresses
those design mechanisms, removes the rejected legacy fallback, and defines
explicit revision-closure selection, two-selector immutable comparison, the
publication CAS comparison baseline, and assertion-filtered impact with
auditable revision proofs. The committed re-review also found branching
`linklog` and hex-tag precedence incomplete; this candidate defines edge-wise
comparison and binds the existing reserved TAGNAME range explicitly.

The next exact re-review found an internally contradictory impact first-match
quantifier, a REF-head snapshot narrower than scoped-tag selector state,
incomplete branching `log`, a non-total selector lexical partition, and
format-5 public-schema/authoring gaps. This candidate corrects the semantic and
observation mechanisms; ADR 0027 owns their exact CLI and machine-output
representation.

The 2026-08-31 three-scope review of exact ADR 0023 digest
`17287daf9b82a087a82215a48008e4598abaf2a82d6e04fe87f4c1645ff6affa`
found that graph revalidation plus one-REF CAS did not state the retained
repository-writer serialization needed to prevent cross-REF write skew, that
`fsck` lacked a complete physical namespace transaction, that
`log --all-paths` did not choose its terminal path set, and that action tables
could be read as narrower gates than the four-ADR acceptance boundary. This
candidate explicitly retains the shared writer guard through publication,
defines the complete `fsck` observation and maximal leaf-path set, and applies
one four-ADR execution gate. These are proposed corrections, not finding
closure or owner acceptance.

The next three-scope review of exact ADR 0023 digest
`74178fecb58f5d018f1286b2f1a5d575b96b14b58fd306e5c4a7f138c79ae299`
found that the physical observation omitted REF/config metadata, the frontier
reference pointed to ADR 0010's superseded named-REF predicate, immutable
comparison claimed an unrepresented derived-graph view, and writer/CAS
authority was not traced to its accepted source. It also confirmed that
direct/transitive stale coexistence would contradict the retained accepted
classification. This candidate defines the complete physical canonical
namespace, carries forward ADR 0011's exact-Seal frontier, narrows immutable
comparison to represented immutable fields, restores Cause-classification
exclusivity, and adds the accepted writer/CAS evidence below. These remain
proposed corrections.

The primary-agent pre-delegation review then exercised empty, alias, mixed-
branch, concurrent-write, invalid-namespace, and permission-boundary cases. It
found that the physical map still needed an unambiguous repository-root slot
and initially treated exact writer-created modes as successful `fsck` rules;
the independent correction below narrows those modes to observation and writer
postconditions. That review also made `S_O` grouping explicit and found
asymmetric action ownership by reverse-comparing every action's Claim column
with the exact trace table.

The next three-scope review of exact ADR 0023 digest
`0d007265c006e8e6bc4f0599cd4c8b205684dbbca7167f9f0bdcd4b59f146f9a`
found that exact loader-created modes had been incorrectly promoted above ADR
0016's mode-neutral integrity boundary and that `P_O` promised to detect a
byte-, kind-, mode-, and path-identical physical replacement absent from its
tuple. This candidate keeps mode changes observable within one command, makes
stable mode values non-canonical, and defines tuple equality as the complete
portable observation boundary. These remain proposed corrections.

The primary correction review then checked transient changes and
outer-Git/staged-tree portability. It narrowed concurrency failure to differing
captured tuples, rejected non-portable inode identity explicitly, and added
tuple-equal replacement fixtures before the next digest freeze.

The next primary pre-delegation horizontal audit applied ADR 0026's RV-29
change-and-restore finding to every two-capture contract. It clarified that
`O`, `I_O`, and `P_O` compare only their two complete captured tuples, claim no
continuous monitoring, and treat a change-and-restore to the same tuple as
observationally equal. A-CR-002 carries that boundary into implementation
fixtures without expanding the portable tuple.

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
  a target-scoped previous-revision Seal array; recorded as D-CR-001 in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md#operator-directions-recorded).
- E-CR-005: the operator confirmed that old parent meaning not observed from a
  Cause Link may be lost, provided migration warns about the change; recorded
  as D-CR-002 in the same decision record.
- E-CR-006: ADR 0015 requires deterministic inspection with complete IDs and
  distinct graph relations; it supports retaining complete assertion sources
  instead of collapsing mixed observations.
- E-CR-007: the operator directed that selector and comparison behavior be
  redefined for the new semantics instead of retaining parent-based defaults;
  recorded as D-CR-003.
- E-CR-008: the operator confirmed that impact defaults to the complete
  observed assertion union, may restrict revision assertions by observer
  through an explicit filter, and annotates the active scope and supporting
  assertion sources; recorded as D-CR-004.
- E-CR-009: the committed three-scope re-review at commit `df113d7` identified
  missing impact and branching `linklog` successor contracts and a possible
  hex-tag selector collision; the exact finding inventory and target digests
  are recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md).
- E-CR-010: the subsequent three-scope review of candidate digest
  `2afbe7c5a62550b310772748916bcd5265313813024f03e0bb131ed030609a0e`
  identified the impact-quantifier, manifest-observation, branching-log,
  selector-partition, and public-schema gaps recorded in the same review
  record. These are review findings, not owner acceptance evidence.
- E-CR-011: accepted ADR 0011 requires every standalone mutation to retain one
  repository-wide writer guard and makes the successful expected-old REF CAS
  the publication linearization point; accepted ADR 0018 requires journal and
  REF work to remain inside that same guard through mutation classification.
- E-CR-012: accepted ADR 0016 explicitly rejects writable checkout modes as
  integrity authority, and accepted ADR 0011 requires Git-sidecar views of
  ordinary tracked `.sealgraph` files to use the same native byte/path
  validators. A portable observation therefore records kind, mode, length, and
  digest changes without inventing a stable inode identity or canonicalizing
  permission bits.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-CR-001 | E-CR-001, E-CR-004, E-CR-005 | A-CR-001 |
| C-CR-002 | E-CR-004, E-CR-006 | A-CR-001, A-CR-005 |
| C-CR-003 | E-CR-002, E-CR-010 | A-CR-001, A-CR-002 |
| C-CR-004 | E-CR-004, E-CR-006 | A-CR-001, A-CR-005, A-CR-007, A-CR-008 |
| C-CR-005 | E-CR-001, E-CR-002 | A-CR-001 |
| C-CR-006 | E-CR-002, E-CR-010, E-CR-011, E-CR-012 | A-CR-002, A-CR-006, A-CR-007, A-CR-008 |
| C-CR-007 | E-CR-001, E-CR-002 | A-CR-003, A-CR-005, A-CR-006 |
| C-CR-008 | E-CR-002, E-CR-010, E-CR-011 | A-CR-003 |
| C-CR-009 | E-CR-003, E-CR-004 | A-CR-004 |
| C-CR-010 | E-CR-002, E-CR-007 | A-CR-006 |
| C-CR-011 | E-CR-002, E-CR-006, E-CR-008 | A-CR-005, A-CR-007 |
| C-CR-012 | E-CR-002, E-CR-009 | A-CR-005, A-CR-008 |
| C-CR-013 | E-CR-002, E-CR-010 | A-CR-003 |

## Follow-ups

- Accept or revise ADR 0025's exact Universal Blob canonical schema.
- Accept or revise ADR 0026's deterministic incompatible migration contract.
- Accept or revise ADR 0027's format-5 CLI and inspection schema contract.
- Run a fresh three-scope ADR review against this exact candidate digest.
- After acceptance, align normative requirements, storage, CLI, architecture,
  integration documentation, and implementation plan in one change set.
