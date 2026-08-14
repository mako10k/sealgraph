# Seal revision DAG and REF rename proposal

Status: accepted design analysis supporting ADR 0011. Runtime implementation,
format-3 dump/load, dogfood conversion, and deferred detail gates remain
separately sequenced.

This document analyzes the coordinated contract change required for:

- Seal identity independent of REF names;
- exact Seal-to-Seal Cause Links;
- branching revision history;
- leaf-based stale derivation;
- a future `sealgraph mv REF1 REF2`;
- explicit preservation of an older result through a sibling revision.

It records the accepted design but does not by itself authorize a partial
runtime format change or tracked dogfood regeneration. The reasoning model is
[`2026-08-14-seal-revision-dag.think`](../decisions/2026-08-14-seal-revision-dag.think).
The consolidated replacement ADR is accepted at
[`ADR 0011`](../adr/0011-ref-independent-seals-and-branching-revisions.md).

## 1. Approved high-level direction

The following direction has operator agreement:

1. A Seal identifies frozen generated material and its exact provenance; it
   contains no current or historical REF name.
2. A Link targets an exact Seal, not a REF. `--depend-on REF` is lookup
   shorthand whose REF identity is discarded after resolution.
3. `parent` means “this is a revision derived from that Seal.” It does not mean
   replacement, invalidation, preference, trust, or same-REF ownership.
4. One parent may have multiple children. Sibling revisions are normal and do
   not make one another stale.
5. A revision that is not an active leaf/tip is stale. Stale remains derived;
   it is never persisted.
6. The canonical product meaning of `supersede` is removed because it is too
   strong for a branching partial order.
7. An older result may be preserved by creating a same-material child from the
   old Seal, then explicitly relinking the downstream result to that sibling
   revision.
8. The active revision DAG is rooted only by current REF HEADs. Object, tag, or
   Cause-Link reachability alone does not publish an active revision.
9. Stale facts need not map one-to-one onto human display labels; machine
   membership and evidence remain exact while presentation may summarize.
10. Stale search normally uses a disposable derived cache, automatically scans
    and rebuilds it after invalidation, and supports explicit `--scan`.
11. `derive` copies the complete identity-bearing material including Cause
    Links; `add --parent` creates new material without inheritance.
12. Selector forms are `REF`, repository-wide `@SEAL_TOKEN`, and scoped
    `REF@TOKEN`; a bare Seal token is not accepted because it can be a REF.
13. Historical, active non-leaf, detached, and draft Seals may be selected
    explicitly as revision parents. Parent admissibility remains separate from
    Cause Link admissibility.
14. Branch-aware frontier ordering uses only the exact Cause closure and stale
    current-head Seals; unselected descendant-tip readiness is irrelevant.
15. `impact` accepts every Seal-resolving selector, displays one deterministic
    shortest path per downstream Seal by default, and bounds explicit all-path
    presentation to 100 paths per downstream Seal unless overridden.
16. Deferred details below remain consultation gates and are not accepted
    implicitly with ADR 0011.

## 2. Current format-3 conflict

Format 3 is name-owned:

- a Seal hashes its `ref` path and treats it as owner identity;
- a Link hashes `target_ref` and `target_seal`;
- parent traversal rejects a Seal whose embedded owner differs;
- stale compares a stored target Seal with the HEAD of stored `target_ref`;
- selectors, tags, closure validation, impact, and linklog repeat owner checks.

Moving only a loose REF file currently breaks dependency lookup and history.
Removing only one owner check would leave the other commands asymmetric. The
change must therefore cross domain, canonical encoding, graph, history,
repository, storage, and CLI boundaries in one coherent experimental format.

A temporary format-3 repository already reproduced the owner mismatch,
missing-target, and log failure. It was removed after read-only observation.

## 3. Two typed relations in the Seal graph

The revised model has two different immutable edge relations.

### Revision edge

```text
child.parent = parent SealID
```

Meaning:

> The child is a revision derived from the parent.

It does not assert that the child is unique, preferred, current, more correct,
or owned by the same REF. Each Seal has zero or one parent; each parent may
have zero or more children.

### Cause edge

```text
dependent.links[] = exact Cause SealIDs
```

Meaning:

> The dependent generated result used these exact sealed results as direct
> causes.

The Link never follows a REF dynamically. Direct Cause identities remain
Merkle-committed into the dependent Seal.

These relations must not be conflated. Parent edges determine revision
ancestry and leafness. Link edges determine Cause closure and transitive stale.
If parent were traversed as an ordinary Cause Link, every current child would
inherit staleness from its own now-non-leaf parent and no branch could become
fresh.

## 4. Active revision DAG

Object existence alone cannot define a new active revision. Seal object bytes
are written before REF CAS publication, so a failed seal can leave a valid
dangling child.

For one validated observation `O`:

```text
H_O = the deduplicated SealIDs stored by all valid current REF HEADs

A_O = H_O plus every Seal reached by following parent edges from H_O
```

`A_O` is the active revision DAG for the observation. This root rule is now the
selected design direction. A child object counts
only when it lies on the parent ancestry of a current REF HEAD. A tag alone,
an unreachable object, or a failed-publication object does not create active
stale state.

The complete REF/head set is captured before derivation and revalidated before
buffered multi-REF output, preserving the accepted coherent-observation
contract. A returned observation remains factual, not a reservation.

A Seal reachable only as a Cause from a current head does not add a root to
`A_O`: Cause reachability proves exact historical use, not current revision
publication.

### Disposable stale/revision cache

Stale and revision queries normally use a local derived index under
`.sealgraph/cache/`. The cache is never canonical provenance, never committed
to an outer Git repository, and may be replaced without migration.

At minimum its validity header binds:

- repository format and cache schema;
- the complete sorted `(REF path, full HEAD SealID)` observation digest;
- its indexed Seal records;
- an internal checksum.

Missing, malformed, checksum-invalid, schema-mismatched, or observation-stale
cache state triggers an automatic canonical scan and atomic cache replacement.
This includes the ordinary case after an outer Git merge or checkout changes
`.sealgraph/refs/`.

“Full scan” means rebuilding from current REF HEADs through validated parent
ancestry and the Cause closure required for stale evidence. It does not mean
enumerating every loose ODB object and treating every child object as active.
That would let a dangling object written before failed publication create
false stale state.

Cache recovery is permitted because cache state is disposable. Canonical
corruption is different: an invalid REF, missing/corrupt Seal, parent cycle, or
invalid Cause object fails closed and does not get repaired from cache. A cache
hit also does not waive canonical object/hash validation required for emitted
evidence.

The explicit surface is:

```text
sealgraph stale --scan
sealgraph stale --frontier --refs-only --scan
```

`--scan` bypasses cache reads, performs the same canonical reconstruction,
attempts atomic cache refresh, and then uses the unchanged output format. If
only cache persistence fails, the validated query still succeeds and emits a
warning on stderr; a read-only cache directory must not invalidate a correct
provenance observation. Concurrent REF/head movement retains the existing
buffer-and-revalidate failure contract.

## 5. Leaf, REF HEAD, stale, and detached state

REF HEAD and revision leaf are deliberately different terms.

- A REF HEAD is merely the Seal currently named by a REF path.
- A revision leaf/tip is maximal in the active parent partial order.

A REF may still point to an internal revision while another REF points to its
descendant. That REF HEAD is valid but stale.

For Seal `s`:

```text
has_active_descendant_O(s) =
  there exists h in H_O where h != s and s is a parent ancestor of h
```

Recommended internal fact model:

| State | Derived condition |
| --- | --- |
| `CURRENT_LEAF` | `s` is a current REF HEAD and has no active strict descendant |
| `STALE_REVISION` | `s` has an active strict descendant |
| `HISTORICAL_OR_DETACHED` | `s` is not in `A_O` |

If a current REF points to `STALE_REVISION`, its own status has a self-stale
fact. The public CLI does not need a one-to-one label for every internal fact.
`STALE_SELF` is only analysis shorthand; a human view may summarize it as
`STALE` while detailed or machine-readable evidence retains the exact reason.

Absence of a descendant does not make a detached Seal current or normally
admissible. Draft/historical workflows may select detached Seals explicitly,
but normal `CLEAN` must not report them as active leaves.

Multiple REF paths may point to the same SealID. Graph derivation deduplicates
by SealID; REF-oriented output still emits each current path according to the
command's documented line protocol.

## 6. Direct and transitive stale

For a current Seal `D`:

```text
STALE_DIRECT(D) =
  at least one direct Link target is not CURRENT_LEAF

STALE_TRANSITIVE(D) =
  every direct target is CURRENT_LEAF, but a deeper Seal reached through the
  Cause Link closure is not CURRENT_LEAF
```

The implementation may calculate these together, but presentation should
retain the nearest factual reason. Parent edges are used only to decide whether
each encountered Cause Seal is a revision leaf; they are not traversed as
Cause edges.

For current REF status, the orthogonal dimensions become approximately:

- mutable candidate: `UNSEALED`;
- provisional current Seal: `DRAFT`;
- current HEAD revision itself is non-leaf: `STALE_SELF`;
- direct Cause is an active non-leaf or historical/detached: `STALE_DIRECT`;
- deeper Cause is an active non-leaf or historical/detached:
  `STALE_TRANSITIVE`;
- none of the above: `CLEAN`.

Precedence and whether more than one human status is printed remain
presentation decisions, not graph semantics. Stable command membership and
machine-readable evidence must still be deterministic. No stale marker is
added to canonical Seal, REF, candidate, tag, or cache state.

## 7. Same-material sibling as an explicit fixed revision

Let `S` be the result currently used downstream. To preserve that result while
also allowing an updated line, create two children:

```text
                         +-- F  same result material, parent=S
S old result revision --+
                         +-- U  updated result material, parent=S
```

`F` copies all identity-bearing result material from `S` except for its new
parent:

- content identity;
- attachments and attachment metadata;
- direct Cause Links and link messages;
- root;
- draft.

Because `parent=S` is canonical, `F` has a new SealID even though its generated
result and direct causes are unchanged.

Suppose old downstream `D1` links to `S`:

```text
D1 --Link--> S
```

Once either `F` or `U` is an active descendant, `D1` is stale. The operator
explicitly changes the candidate dependency to `F` and reseals only that one
downstream REF:

```text
D2 --Link--> F
```

`U` is a sibling of `F`, not its descendant, so it does not make `F` stale.
The old `D1` remains immutable and stale in history. No pin flag, mutable
acknowledgement, display suppression, recursive relink, or automatic branch
selection is required.

Creating `F` immediately makes `S` non-leaf, so temporary stale state is an
expected and honest observation while downstream REFs are reviewed one at a
time.

This mechanism fixes one revision choice; it does not suppress unrelated
Cause changes below `F`. If one of `F`'s direct Cause targets later gains an
active descendant, `F` becomes stale through its own Cause Links. Preserving
that older Cause also requires an explicit revision choice at that Cause
boundary.

## 8. Publication and fork construction

One `seal` publication still creates exactly one Seal and CAS-updates exactly
one REF. Forking does not authorize batch publication.

The existing default remains useful:

```text
candidate for existing REF
  parent = that REF's observed current HEAD
```

Branch construction additionally needs an explicit way to create a candidate
for an absent destination REF from one exact parent Seal while leaving the
source REF untouched. Recommended first-slice constraints are:

- exact full SealID or resolved immutable selector identifies the parent;
- destination REF is absent;
- source Seal is loaded and validated before candidate creation;
- same-material mode copies the complete material set listed above;
- publication CAS expects destination absent;
- one writer guard covers candidate mutation and seal publication as today;
- no child REF namespace recursion;
- no automatic downstream relink;
- no automatic choice among sibling tips.

Two candidate creation surfaces are selected:

```text
sealgraph derive NEW_REF --from SOURCE_SELECTOR
sealgraph add NEW_REF --parent SOURCE_SELECTOR --content VALUE
```

`derive` is complete material inheritance. `add --parent` is new material with
an explicit revision parent. They are intentionally distinct; no selective
inheritance flag is introduced.

This is a revision-derivation operation, not Git checkout, named-branch merge,
or history rewrite. Its final CLI name and whether candidate creation and
publication are one or two explicit commands remain pending.

An important consequence is that the parent of a newly published Seal need not
be owned by, or previously be HEAD of, the destination REF. Format 4 cannot
retain the format-3 “parent belongs to the same REF” invariant.

Approved parent admissibility is intentionally broader than normal Cause
admissibility:

- an active non-leaf may be selected explicitly; this is required to create a
  sibling revision;
- a detached historical Seal may be selected only by an explicit immutable
  selector, never by inferring a likely branch or latest revision;
- a draft Seal may be a parent of a draft or non-draft child because draft is a
  property of each generation, not a property inherited through revision
  ancestry;
- the new child's own root, draft, and Cause closure are validated independently;
- a parent is not a Cause and cannot satisfy the dependency required by a
  non-root child;
- the first slice does not let an existing destination REF fork from a
  non-HEAD parent. The explicit-parent forms require an absent destination.

`derive` copies the source draft flag, so its initial candidate remains draft
when the source is draft. Publishing a reviewed normal child requires an
explicit candidate flag edit; there is no implicit promotion. `add --parent`
uses the draft option supplied for the new material and inherits no flag.

Both `derive NEW --from SOURCE` and `add NEW --parent SOURCE` require the
destination REF and destination candidate to be absent. The candidate records
`expected_ref_head = null`, and publication fails by CAS if another writer has
created the destination. For an ordinary update to an existing REF, the
observed current HEAD supplies both `parent_revision` and
`expected_ref_head`; an alternate-parent override is rejected in the first
slice.

Before writing any candidate, source resolution must produce one canonical
Seal and its parent chain must validate. Missing, ambiguous, non-Seal,
scope-mismatched, or corrupt sources and destination path/file conflicts fail
without leaving a partial candidate. Parent cycles or missing parent objects
also fail closed when historical state is loaded or traversed.

### Exact `derive` material scope

`derive NEW --from SOURCE` copies:

| Field | Copy? | Reason |
| --- | ---: | --- |
| content identity | yes | same generated result |
| attachments and attachment metadata | yes | identity-bearing material |
| direct Cause Links | yes | exact generation basis |
| Cause Link messages | yes | identity-bearing edge rationale |
| root | yes | provenance-boundary property |
| draft | yes | revision-generation property |
| source `parent_revision` | no | the new parent is SOURCE itself |
| REF path | no | lookup state, absent from Seal |
| tags | no | external immutable aliases |
| stale/cache | no | derived state |
| actor/time/event metadata | no | excluded from Seal material |
| candidate metadata | no | mutable orchestration state |

The resulting candidate has:

```text
parent_revision   = exact SOURCE SealID
expected_ref_head = null
```

Cause Links must be copied. Parent revision ancestry is not a substitute for
direct Cause provenance. Omitting Cause Links would create a different
material/provenance state rather than a same-material revision.

Candidate creation itself may copy Causes that have since become non-leaf.
That is inspectable working intent. Normal non-draft publication still rejects
the candidate until its complete Cause closure satisfies active-leaf and draft
admissibility. The operator may instead edit content, Link, attachment, root,
or draft state explicitly; every edit remains visible in candidate diff.

`add --parent` performs no inheritance. It uses the supplied content,
attachments, Links, root, and draft options as a newly constructed material
state. This avoids ambiguous `derive --without-links` or partial-copy modes.

## 9. Normal seal admissibility

Cause Link admissibility and parent admissibility are separate.

Recommended normal non-draft Cause rule:

1. every direct target is an active `CURRENT_LEAF`;
2. every reachable Cause Seal is non-draft;
3. every reachable Cause Link target is an active leaf;
4. Cause and revision cycles or missing objects fail closed;
5. no generic ignore-validation switch exists.

Draft/historical workflows may preserve non-leaf or detached Cause targets,
but remain visibly draft or historical according to the separately approved
contract.

The selected parent does not need to be a leaf. Otherwise the second child of
a revision could never be created. Parent selection is explicit revision
topology, while Links are the generated result's Cause closure.

## 10. Canonical format-4 Seal shape

Illustrative compact JSON only:

```json
{"schema":"sealgraph/seal/v4","parent_revision":"<seal-id>","content":{"store":"native","type":"blob","id":"<object-id>"},"attachments":[],"links":[{"target_seal":"<seal-id>","message":"review basis"}],"root":false,"draft":false}
```

The breaking changes are:

1. remove top-level `ref`; do not add `ref_at_seal` or another owner salt;
2. replace `parent` with the required `parent_revision` member: `null` for an
   initial revision, otherwise one exact full parent Revision Seal ID;
3. remove Link `target_ref` without adding RefID;
4. sort Links by `(target_seal, message)`;
5. reject duplicate exact target SealIDs;
6. allow multiple current REFs to point to the same SealID;
7. allow multiple active children for one parent;
8. remove same-REF parent/tag/selector ownership checks where they no longer
   express a selected UI scope.

The Git SHA-256 blob envelope, full-hex ObjectID, and loose object path do not
change. Same parentless material intentionally produces the same SealID
regardless of REF path. A same-material child produces a new SealID because
its `parent_revision` differs. The member is always present, participates in
canonical ordering, and is hash-committed; it is never inferred from a REF.

Candidate v4 remains mutable and may contain the destination REF path and
expected publication state. Those fields are orchestration state, not Seal
identity.

## 11. Selectors, history, link editing, and tags

### Exact selection

The approved selector contract follows. Once Seals are not REF-owned,
repository-wide exact selection is necessary,
but a bare lower-hex token is ambiguous with a Git-like path REF. Giving either
interpretation precedence would make meaning depend on which REFs currently
exist. The public selector grammar therefore has three forms:

| Form | Meaning |
| --- | --- |
| `REF` | resolve the current HEAD of exactly that REF |
| `@SEAL_TOKEN` | resolve a repository-wide unique ODB prefix, then require a canonical Seal |
| `REF@TOKEN` | resolve a Seal in an explicit REF UI scope |

`SEAL_TOKEN` and a hex `TOKEN` use the accepted 4-to-64 lower-hex prefix
contract. Resolution still searches all valid ODB object names; zero matches,
multiple matches, and a uniquely matched non-Seal object are errors. A bare
Seal token is not accepted. Full IDs are canonical, while prefixes are only
input shorthand and are never persisted or emitted as receipts.

For `REF@TOKEN`:

- a 4-to-64 lower-hex TOKEN resolves repository-wide and must identify a Seal
  equal to or in the parent ancestry of the REF's current HEAD;
- any other TOKEN resolves the immutable tag in that REF's UI namespace;
- the ancestry rule is a selector-scope assertion, not Seal ownership;
- a sibling or detached Seal that is not in the current ancestry remains
  selectable as `@SEAL_TOKEN`, or by an applicable immutable tag;
- an old REF name is not retained as an alias after `mv`.

The tag lookup above does not reintroduce owner identity into the Seal. Whether
tag creation requires current ancestry and how a tag namespace moves with a
REF remain in the later tag/rename consultation gate.

These selector forms apply consistently to `show`, `diff`, parent selection,
and Cause Link editing. For example:

```text
sealgraph derive preserved --from @0123abcd
sealgraph add revised --parent requirements/api@0123abcd --content VALUE
sealgraph link design/api --depend-on @0123abcd
sealgraph show @0123abcd
```

`--depend-on REF` still resolves current HEAD. `--depend-on REF@TOKEN` and
`--depend-on @SEAL_TOKEN` resolve explicit Seals. Persisted Links contain only
the resolved full SealID. Because Links no longer have an upstream REF key,
duplicate detection and `unlink` match the exact target SealID. After an
upstream REF advances, removing its old Link therefore requires the exact
selector printed by candidate inspection, such as `@OLD_SEAL`; resolving the
bare REF would select the new HEAD and correctly fail to match the old Link.

Candidate inspection exposes revision topology and publication state as two
separate facts:

```text
PARENT_REVISION <full-seal-id-or-NONE>
EXPECTED_REF_HEAD <full-seal-id-or-ABSENT>
```

The first is hash-committed derivation. The second is mutable CAS intent. A
single `base` field must not continue to stand for both.

### History

`log REF` resolves the current HEAD and follows parent IDs. It does not compare
embedded names. `diff` may compare any two exact Seals; a same-revision-line
mode can require ancestor relation when that distinction is useful.

`linklog` compares exact target Seal sets between adjacent dependent
revisions. A target move from ancestor `S` to descendant `F` can be presented
as `REPOINT`; ambiguous N:M matching must fail or fall back to explicit
REMOVE/ADD rather than guess.

### Tags and rename

REF-scoped tag names remain a separate UI/storage issue. The accepted
`tags/<REF>/<ENCODED_TAGNAME>` layout does not move atomically with a one-file
REF rename and still has the hierarchical collision SG-BL-010.

Possible solutions include an external stable tag-namespace ID, a one-ref
manifest containing HEAD/tag bindings, or a crash-safe multi-path transaction.
Any namespace ID would remain outside Seal and Link bytes. This detail must be
settled before `mv`; it is not needed to define revision/stale semantics.

Old REF names are not historical aliases. Exact Seal Links remain valid if a
path is moved or reused because they contain no path.

## 12. Impact and branch-aware stale frontier

`impact SELECTOR` resolves any approved selector to exact Seal `h` and finds
current dependent Seals whose Cause closure targets `h` or a
`parent_revision` ancestor of `h`. With siblings, the same old target may be
impacted by more than one active tip; results are deduplicated and no preferred
tip is inferred.

The accepted frontier definition currently assumes every Link names one
upstream REF. That assumption disappears. Looking at all active descendant
tips of a stale target is tempting, but those tips are not yet Causes of the
dependent Seal. A dirty unselected sibling must not block review, and one clean
unselected sibling must not imply that a valid relink choice has been made.

The approved replacement is an exact-Cause structural frontier. For one
validated current-head observation `O`, define:

```text
H_O(r) = exact current HEAD SealID of REF r

S_O = { r |
  H_O(r) has STALE_SELF, STALE_DIRECT, or STALE_TRANSITIVE
}

Q_O = { H_O(r) | r in S_O }

CausePlus(d) = strict transitive closure from Seal d using Link edges only

F_O = { r in S_O | CausePlus(H_O(r)) ∩ Q_O = ∅ }
```

Thus a stale current REF blocks a downstream stale REF only when that exact
stale current-head Seal is present in the downstream Seal's frozen Cause
closure. Parent edges are never traversed for this test. Active descendants
that are not exact Causes, whether clean or stale, do not affect membership.

This retains the original meaning of an upstream-first *review* frontier while
avoiding a readiness claim:

- if `C` links the exact current stale head of `B`, `B` is reviewed before `C`;
- if `C` links historical `B1` while another REF points to stale descendant
  `B2`, `B2` is not in `C`'s Cause closure, so the two reviews are independent;
- if a REF still points to non-leaf `B1`, that REF is `STALE_SELF`; a dependent
  linking exact `B1` is behind it in the frontier;
- if no current REF points to linked `B1`, a dirty sibling tip does not invent
  an upstream ordering edge;
- aliases that point to the same stale Seal are deduplicated for graph
  computation but each selected REF path remains in REF-oriented output.

Being in `F_O` does not mean that the REF is sealable, has a clean descendant
choice, is approved, or requires resealing. It means only that no other stale
current-head Seal occurs earlier in its exact Cause provenance. The operator
still inspects the zero, one, or many active descendant tips and makes an
explicit relink decision.

`--frontier`, `--refs-only`, and the already approved `--scan` behavior remain
orthogonal. Cache hits and canonical scans must produce identical `S_O`,
`Q_O`, `F_O`, ordering, and evidence. `--refs-only` remains a stream of current
REF paths, never a list of automatic relink commands. Candidate files remain
outside all three sets.

### Bounded impact path presentation

Impact membership and path presentation are separate. The query must first
derive every distinct impacted current-head Seal and validate its complete
reachable Cause graph. Limiting displayed paths must never hide an impacted
Seal, skip integrity validation, or turn a corrupt unread path into a
successful partial result.

The approved default is one deterministic shortest path for each distinct
impacted current-head Seal:

1. `impact SELECTOR` accepts `REF`, `@SEAL_TOKEN`, or `REF@TOKEN` and resolves
   it to exact Seal `h`.
2. A Cause path matches when its first encountered terminal is `h` or a
   `parent_revision` ancestor of `h`; traversal stops at that first match.
3. Distance is the number of Link edges.
4. If several paths have minimum distance, compare their full SealID sequences
   bytewise and select the lexically first sequence.
5. Group by downstream current-head SealID. If several current REF paths point
   to it, compute the path once and present the sorted REF paths as aliases.
6. Exclude `h` itself as a downstream result, including all current REF aliases
   that point to `h`.

Selectors that resolve to the same `h` produce identical impact membership and
path evidence. The selector's REF spelling is retained only in the source
receipt when useful; it is not a graph key. Detached, historical, tagged, and
current Seals therefore use the same algorithm after canonical validation.

Stopping at the first matching source-ancestry Seal prevents one physical
route from being emitted again for every older matching ancestor below it. A
direct path always wins over a transitive path. Default result ordering is by
downstream full SealID, with aliases and evidence ordered bytewise.

Explicit exhaustive presentation uses:

```text
sealgraph impact [--all-paths] [--max-paths N] SELECTOR
```

- without `--all-paths`, exactly one shortest path is displayed per impacted
  downstream Seal;
- `--all-paths` enumerates distinct simple Link-edge paths in `(edge count,
  full SealID sequence)` order;
- `--max-paths N` is valid only with `--all-paths`, requires a positive integer,
  defaults to 100, and applies separately to each impacted downstream Seal;
- when another path exists beyond the limit, output includes an explicit
  truncation marker with the SealID and limit; expected truncation exits zero;
- a truncated group is never presented as complete, but every impacted Seal
  still receives its membership summary;
- `--max-paths` is a presentation/enumeration bound, not a traversal,
  validation, impact-membership, or current-head snapshot bound.

A per-downstream limit matches the actual explosion boundary and prevents an
early lexical result from consuming a global budget and hiding path evidence
for later impacted Seals. The number of impacted Seal summaries is not reduced
by this option. A separate result-set limit would be a different future
contract.

The full-path enumerator may stop after finding `N+1` ordered paths for one
downstream Seal so it can prove truncation without counting every omitted
path. It still validates the complete reachable immutable graph independently.
The command buffers output and revalidates the captured complete current-head
snapshot before emission, like other multi-REF factual queries.

This path presentation contract is approved. It does not make impact a
stale-only query: impact continues to report structural Cause reachability
even when all current heads are clean.

## 13. Removing `supersede` from product semantics

The canonical vocabulary becomes:

- derive a revision;
- publish a revision;
- advance a REF;
- parent/child revision;
- active descendant;
- revision leaf or tip;
- stale Cause.

`supersede` is removed from normative product meaning because it implies
exclusive replacement or invalidation. Existing accepted ADR files remain
historical records and are not rewritten. A later approved replacement ADR
will state which earlier decisions no longer govern format 4.

Error and help text should avoid claiming that a stale Seal is invalid, false,
untrusted, or forbidden. Stale says only that a newer active revision exists
within the parent partial order selected by current REF heads.

## 14. Explicit Dump -> Load migration

The pre-1.0 runtime should remain simple while preserving inspectable dogfood
conversion:

1. add a versioned read-only logical dump to the format-3 binary;
2. reject corruption and unresolved candidates instead of guessing repairs;
3. emit content, attachments, parent edges, Cause Links, flags, tags, and
   current REF heads in deterministic order;
4. load only into an empty format-4 repository;
5. rebuild Seals topologically after removing owner REF fields;
6. rewrite every target SealID through the generated mapping;
7. allow several old format-3 SealIDs to map to one format-4 SealID when owner
   name was their only difference;
8. emit the complete old-to-new mapping, including many-to-one entries;
9. verify the active revision and Cause DAGs before publishing converted refs.

This is not a runtime dual reader, in-place auto migration, ignored legacy
field, or automatic repair.

## 15. Acceptance scenarios

The detail contract should cover at least:

1. Two parentless Seals with identical material under different REF paths have
   the same format-4 SealID.
2. A same-material child differs from its parent because `parent_revision` is
   hashed.
3. Two children of one parent are valid sibling leaves.
4. Publishing either child makes the parent `STALE_REVISION`.
5. One sibling never makes the other sibling stale.
6. A current REF may point to a non-leaf and reports self-stale.
7. A direct Link to a non-leaf reports `STALE_DIRECT`.
8. A deeper stale Cause reports `STALE_TRANSITIVE` without traversing parent as
   a Cause edge.
9. A dangling child left before failed REF CAS does not make its parent stale.
10. A same-material sibling can be selected by explicit relink and the new
    downstream generation is clean.
11. The old downstream generation remains immutable and stale.
12. Multiple active descendant tips are reported without automatic preference.
13. A detached target is not misreported as current/clean.
14. Normal Link closure accepts only active leaf, non-draft Causes.
15. Explicit parent selection can create a sibling from an active non-leaf.
16. One derivation publication updates exactly one destination REF.
17. History follows parent IDs across REF creation and rename without owner
    checks.
18. `impact` finds old Cause targets through each selected tip's ancestry.
19. Moving a REF path without changing its SealID does not alter revision or
    stale results.
20. Format-3 dump/load handles many-to-one SealID conversion deterministically.
21. Cache hit and full scan produce the same stale membership and evidence.
22. REF/head digest mismatch after an outer Git merge triggers scan and atomic
    cache replacement.
23. A failed-publication dangling child is excluded during scan.
24. `--scan` bypasses a valid cache without changing stdout format.
25. Canonical corruption fails instead of being hidden or repaired by cache.
26. `derive` copies content, attachments, Cause Links/messages, root, and draft
    exactly, but not tags, REF, cache, stale, or event state.
27. `add --parent` inherits no material and still commits the exact parent
    Revision Seal ID.
28. Git SHA-256 loose-object bidirectional conformance remains unchanged.
29. Standalone operations do not inspect `.git`.
30. A lower-hex REF cannot capture the meaning of `@SEAL_TOKEN`, and adding a
    new REF cannot change an existing selector's interpretation.
31. `REF@SEAL_TOKEN` accepts an ancestor but rejects a sibling or detached
    Seal; the same Seal remains selectable explicitly as `@SEAL_TOKEN`.
32. An active non-leaf can be an explicit parent of a new sibling.
33. A detached historical Seal can be an explicit parent without any owner
    check.
34. A draft parent does not automatically make its child draft, while
    `derive` initially copies the source draft flag and exposes its removal in
    candidate diff.
35. An existing destination REF or candidate rejects `derive` and
    `add --parent` without overwriting either state.
36. Candidate inspection distinguishes `parent_revision` from publication
    `expected_ref_head`.
37. If `C` links the exact stale current HEAD of `B`, the frontier contains
    `B` and excludes `C` until `B` is no longer stale or no longer its exact
    Cause.
38. A dirty sibling descendant that is not in `C`'s exact Cause closure does
    not remove `C` from the frontier.
39. A clean sibling descendant does not by itself add `C` to the frontier or
    select a relink target.
40. A current REF pointing to a non-leaf exact Cause is `STALE_SELF` and blocks
    its exact downstream in structural frontier ordering.
41. Multiple current REF aliases at one stale Seal produce identical graph
    membership and deterministic per-REF output without blocking one another.
42. `stale --frontier` and `stale --frontier --scan` return identical selected
    REFs and evidence for an unchanged canonical snapshot.
43. A diamond with two paths to one impacted Seal emits one deterministic
    shortest path by default.
44. A direct and a transitive route to the source ancestry emits the direct
    route by default.
45. Equal-length shortest paths choose the lexically first full-SealID
    sequence regardless of link input, REF name, or map iteration order.
46. `--all-paths` emits all distinct paths up to 100 per impacted Seal by
    default and marks a 101st path as truncation without enumerating it.
47. `--max-paths N` without `--all-paths`, zero, negative, or malformed N is a
    usage error before repository traversal.
48. Path truncation never removes an impacted Seal summary and never hides a
    missing, corrupt, or cyclic object.
49. Two REF aliases at one impacted current-head Seal share one path
    computation and are presented as sorted annotations.
50. `impact REF`, `impact @SEAL_TOKEN`, and `impact REF@TOKEN` produce identical
    membership and path evidence when they resolve to the same Seal.
51. A detached or historical exact Seal is a valid impact source, and the
    source Seal itself is not reported as its own downstream impact.
52. Equal canonical `.sealgraph` file trees produce equal native validation,
    graph, and inspection results through standalone worktree, sidecar staged,
    and sidecar commit-tree views.
53. The outer Git OID, repository object format, commit, branch, and path do not
    enter native candidate, Seal, Link, REF, or ObjectID bytes.
54. Staged validation reads the prospective commit tree, including unchanged
    base paths, and is unaffected by a different unstaged worktree file.
55. A concurrent index change, unsupported repository format, missing
    promisor/partial-clone object, or transformed canonical file fails
    explicitly without a native mutation or network fetch.
56. A validation hook cannot stage, seal, advance a REF, relink, repair,
    commit, push, overwrite another hook, or treat hook success as approval.
57. A merge-stage conflict on one mutable REF remains explicit; different
    bytes at one immutable native object path fail as corruption.
58. Runtime candidate, lock, cache, or log paths staged below `.sealgraph` are
    rejected by prospective-tree validation.
59. Importing a source Git blob/tree/commit/tag as generated material remains
    unavailable until its separate content contract is approved.
60. A historical Git tree containing an unsupported native repository format
    fails explicitly and does not activate a dual reader or automatic
    migration.

## 16. Future Git sidecar compatibility seam

Format 4 should not make Seal identity depend on the outer Git repository's
object format or on the SDK selected later. Git sidecar does not need a second
Sealgraph storage mode; it needs Git-aware views of the same native files:

```text
real .sealgraph filesystem ----------> native reader / writer
                                           |
Git worktree view -------------------------|
Git prospective staged-tree view ----------+--> same config/object/REF decoders
Git immutable commit-tree view ------------+--> same domain/revision/Cause graph
Git merge stage 1/2/3 entries --------------> conflict evidence joined to
                                               BASE/OURS/THEIRS complete trees

Only the real filesystem view is mutable.
Every Git tree/index/stage view is inspection input.
```

### One native format, several read views

The real `.sealgraph` filesystem remains the only native mutation target.
Standalone opens it without inspecting `.git`. `git-sealgraph` explicitly
locates the outer repository and may construct these additional read views:

| View | Source | Mutation | Snapshot rule |
| --- | --- | --- | --- |
| native filesystem | current `.sealgraph` paths | native writer/CAS only | existing coherent REF capture/revalidation |
| Git worktree | outer worktree files | none through the view | diagnostic only; do not confuse with index |
| prospective staged tree | base tree plus index stages-zero changes | none | capture and revalidate the complete index state |
| immutable commit tree | one exact commit/tree | none | tree OID fixes the view |
| merge conflict evidence | index stages 1/2/3 plus corresponding complete source trees | explicit conflict workflow only | capture/revalidate the index; graph claims use a validated complete tree |

Every complete view exposes exact path existence, directory enumeration, and
file bytes. Merge stages expose only conflict entries and must not be passed
off as a complete repository. The native layer still decompresses and hashes
`.sealgraph/objects/xx/...`, parses canonical Seal payloads, validates REF/tag
targets, and derives the revision/Cause graph. Outer Git blob OIDs are merely
physical locators for view bytes and never replace a native identity.

Two byte-identical complete canonical trees must produce the same validation,
current-head revision/Cause, stale, impact, and history facts regardless of
view. Runtime candidate and cache facts are outside that comparison: they are
not canonical tracked tree state, and their absence must not be reported as a
candidate `CLEAN` claim. Intentional asymmetry is limited to runtime/authority:
immutable Git views cannot inspect an untracked candidate, create candidates,
write objects, advance a REF, refresh cache, or repair state.

### Tracked and runtime path policy

The outer Git repository versions canonical files as ordinary worktree files:

- `.sealgraph/config`;
- immutable `.sealgraph/objects/**` files;
- loose `.sealgraph/refs/seals/**` files;
- loose `.sealgraph/refs/tags/**` files.

It must not version mutable candidate index, locks, derived cache, logs, or
temporary files. Prospective-tree validation rejects those paths rather than
silently ignoring them.

Canonical bytes must reach the staged tree unchanged. Git LFS, clean/smudge
filters, working-tree encoding, line-ending normalization, or another
attribute transformation over canonical `.sealgraph` paths is unsupported.
The eventual sidecar setup contract must verify or install a non-transforming
attribute policy without overwriting unrelated user rules. A transformed path
fails native hash/canonical validation; it is not translated back.

### Prospective staged-tree validation and hooks

A pre-commit check must validate what Git will commit, not whichever bytes are
currently in the worktree. The adapter constructs the prospective result tree
from the base commit and stage-zero index, so unchanged canonical objects remain
visible and unstaged edits do not contaminate the observation.

The validator checks at least:

- repository-format/config consistency;
- exact native object path, envelope, decompression, hash, and canonical Seal
  decoding;
- every REF/tag target and complete reachable parent/Cause closure;
- canonical path grammar and hierarchical conflicts;
- exclusion of runtime/candidate/cache/lock/log/temp paths;
- one coherent captured index observation before emitting success.

If the index changes concurrently, an object is missing from a partial clone,
or the selected SDK cannot prove the repository format, validation fails before
native mutation. It does not fetch from a promisor remote implicitly.

The selected Git tree may contain an older Sealgraph repository format. That is
not authority to restore backward-compatibility machinery: the current binary
fails with the exact unsupported format and directs the operator to a matching
old binary or the approved logical dump/load path.

Hook integration is opt-in and validation-only. The narrow first surface can be
a command such as `git sealgraph hook run pre-commit` called by a user-managed
hook. It never installs itself, overwrites an existing hook, stages paths,
creates a Seal, advances a REF, relinks, repairs, commits, or pushes. An
installer/dispatcher, if later desired, requires a separate collision-safe
contract.

### Merge-stage assistance

The sidecar may inspect `.sealgraph` conflict entries from index stages. Stage
1/2/3 alone are not complete repository views, so it must associate them with
the corresponding BASE/OURS/THEIRS complete trees and validate those trees
through the same native graph code before it may explain:

- the exact SealID at each conflicting REF;
- parent ancestry and sibling relations;
- Cause Link differences and affected current REFs;
- missing or corrupt objects in any side.

The following remain hard boundaries:

- different bytes at one native immutable-object path are corruption;
- divergent targets at one movable REF remain an explicit choice;
- divergent immutable tags are not retargeted automatically;
- no sibling is preferred, child Seal manufactured, Cause relinked, or approval
  inferred from Git's merge result.

Git normally handles disjoint REF-file additions and unilateral changes without
calling semantic assistance. The sidecar is for inspection and an explicitly
selected resolution, not a second merge engine inside Sealgraph core.

### Typed SDK boundary

Inside the Git view adapter, physical Git identity is typed by at least:

```text
object_format = sha1 | sha256
object_type   = blob | tree | commit | tag
full_oid      = 40 or 64 lower-hex according to object_format
```

This type locates tracked `.sealgraph` bytes but does not enter native
`ObjectID`, candidate or Seal bytes, graph, history, or output receipts that
claim Seal identity. The adapter returns a byte/path view, not Git hash types.

As of 2026-08-14, go-git v5.19 demonstrates SHA-256 repositories only with its
`sha256` build tag, while the official v6 migration guide describes
simultaneous SHA-1/SHA-256 repository support as planned and not yet finalized.
Therefore ADR 0011 does not pin a go-git major version or shape core APIs
around the current `plumbing.Hash` type.

At the sidecar implementation gate, pin one SDK version and prove in temporary
repositories which object formats the same released binary can open and read
across worktrees, linked worktrees, stage-zero and conflicted indexes, immutable
trees, loose objects, packs, and alternates. Unsupported, unknown, or mixed
formats fail explicitly. There is no hand-written pack reader, silent Git CLI
fallback, or implicit network fetch.

The product entry points remain explicit:

- `sealgraph init` and all standalone lifecycle code never discover `.git`;
- `git-sealgraph` may locate and open Git because the user invoked
  `git sealgraph ...`;
- sidecar mutations use the same real `.sealgraph` native writer and one-REF
  CAS protocol, not the outer Git ODB or Git refs;
- the outer Git repository versions `.sealgraph/objects` and loose REF files as
  ordinary worktree files; its own OIDs for those files are transport/history
  identities and never Seal IDs or native ObjectIDs;
- immutable native object paths need no Git merge semantics beyond ordinary
  byte identity; conflicting movable REF files require explicit sidecar
  inspection and never authorize an automatic Seal;
- outer Git merge-stage inspection may read BASE/OURS/THEIRS `.sealgraph` REF
  files, but it never manufactures a Seal or semantic approval.

Rejected alternatives are storing Seals in the outer Git ODB using that
repository's hash, storing an untyped Git OID as a native ObjectID, or giving
staged/history views a different Seal decoder. They split identity or command
meaning across entry points.

Importing Git source blobs as generated material is deferred, not rejected.
If demanded later, exact payload materialization can still write the same
native ContentRef without changing format 4. A zero-copy external reference or
tree/commit/tag projection would be a new availability/identity contract and
requires separate approval. It is not an ADR 0011 acceptance blocker.

This accepted seam preserves a later SDK choice without requiring format 4 to
change direction.

Primary evidence:

- [Git hash-function transition](https://git-scm.com/docs/hash-function-transition)
- [Git loose-object format](https://git-scm.com/docs/gitformat-loose)
- [go-git v6 migration status](https://go-git.github.io/docs/tutorials/migrating-from-v5-to-v6/)
- [go-git v5.19 SHA-256 example](https://github.com/go-git/go-git/blob/v5.19.0/_examples/sha256/main.go)

## 17. Symmetry, consistency, and completeness review

### Symmetry

| Question | Result |
| --- | --- |
| Do equal complete native file trees mean the same thing in standalone, staged, and commit-tree views? | Yes. All use the same native decoders and domain/graph validation. |
| Are merge index stages complete views? | No. They are conflict entries joined to validated BASE/OURS/THEIRS complete trees. |
| Do all selectors resolve to full native SealIDs? | Yes. Outer Git OIDs remain view locators only. |
| Can every view mutate? | Deliberately no. Only the real filesystem uses native writer/CAS; immutable/staged views are observational. |
| Does sidecar publication differ from standalone publication? | No. One candidate, one Seal, one REF CAS, no batch or Git commit implication. |
| Are candidate/cache facts symmetric? | They are deliberately excluded. Read-only Git views neither infer candidate cleanliness nor persist/refresh runtime cache. |

The only asymmetry is explicit authority, not meaning. A historical Git tree
cannot be made writable merely to imitate the standalone filesystem.

### Consistency

The accepted boundary is consistent with the main Sealgraph philosophy:

- external Git commit, branch, actor, time, path, and OID state does not enter
  Seal identity;
- parent revision and exact Cause Link semantics are unchanged by entry point;
- standalone still never probes `.git`;
- sidecar uses Git only because its executable was invoked explicitly;
- hooks validate but do not approve or publish;
- merge assistance exposes alternatives but never chooses a semantic result;
- outer Git file tracking does not turn Git branches or commits into Sealgraph
  REFs or revisions.

It also preserves the accepted merge-friendly layout: immutable objects remain
additive, independent REF paths merge independently, and divergent updates to
one loose REF produce a desirable conflict.

### Completeness

The review covers identity, filesystem layout, worktree/index/tree/stage views,
snapshot consistency, native object validation, REF/tag reachability, runtime
path exclusion, Git attributes/filters, partial-clone failure, hooks, merge
conflicts, SDK object-format capability, and explicit mutation authority.

The following remain named pre-implementation gates rather than hidden gaps:

1. exact `git sealgraph init/setup` behavior and non-destructive
   `.gitignore`/`.gitattributes` policy;
2. exact staged-view construction for unborn HEAD, multiple worktrees,
   submodules, sparse index, and split index;
3. hook command names, output/exit contract, and coexistence with an existing
   hook manager;
4. explicit merge resolution command and index-lock transaction;
5. whether historical commands expose `--at REV` directly or a narrower
   read-only `verify` surface first;
6. the SDK version and the proven SHA-1/SHA-256 capability matrix;
7. any future Git source-material import contract.

None of these gates requires a new Seal schema. They affect only Git view,
setup, or presentation contracts.

The post-acceptance llmthink re-audit on 2026-08-14 completed with exit 0 for
both decision models:

- `sealgraph-design.think`: `fatal=0 error=0 warning=0`;
- `2026-08-14-seal-revision-dag.think`: `fatal=0 error=0 warning=0`.

The revision model's contradiction candidates were manually checked. They are
complementary decisions intentionally sharing one evidence set—for example,
semantic equality across views versus read-only authority, and staged
validation versus rejection of automatic sealing—not conflicting outcomes.
The remaining `pending` findings correspond to the explicit tag/move,
format-3 dump/load, and Git setup/merge detail gates below.

The accepted normative documents now describe format 4 while the checked-in
runtime still implements format 3. That is an explicit implementation frontier,
not permission for a partial reader or mixed-format publication. The normative
update includes `requirements.md`, `architecture.md`, `integrations.md`,
`cli.md`, `storage-format.md`, and the original `sealgraph-design.think`.

## 18. Implementation order after ADR acceptance

1. Add deterministic format-3 logical dump before changing the runtime reader.
2. Introduce content-only format-4 Seal/Link bytes and fixture hashes.
3. Add one shared revision index derived from coherent current REF heads.
4. Add cache invalidation, automatic canonical rebuild, and `stale --scan`
   equivalence tests.
5. Convert stale, normal closure, status, log, diff, linklog, graph, and impact
   away from REF ownership.
6. Implement `derive` complete clone and `add --parent` new-material candidate
   creation.
7. Add empty-repository format-4 load and mapping verification.
8. Regenerate tracked dogfood through dump/load and exercise a same-material
    sibling freeze.
9. Resolve tag namespace and add narrow crash-safe `mv`.
10. Implement the approved branch-aware stale frontier and bounded impact path
    projection.
11. Add a read-only `.sealgraph` tree-view abstraction and staged-tree
    validator using deterministic in-memory fixtures.
12. Add the Git SDK only at the explicit sidecar capability gate, then prove
    index/tree/stage behavior in temporary SHA-1/SHA-256 repositories.
13. Add validation-only hook dispatch and merge-stage inspection after their
    public CLI contracts are approved.
14. Resume alpha freeze only after the revised fsck and dogfood gates pass.

This should precede Git-sidecar implementation. A Git SDK remains appropriate
for physical Git object/ref reading after format 4, not for defining Sealgraph
revision or standalone mutation semantics.

### Package responsibility sketch

| Package | Proposed responsibility |
| --- | --- |
| `internal/domain` | content-only Seal, exact Cause Link, parent revision identity |
| `internal/canonical` | deterministic format-4 bytes and exact-target duplicate checks |
| `internal/revision` | active parent index, ancestor/descendant, leaves, detached state, fork validation |
| `internal/graph` | direct/transitive stale and impact over typed revision/Cause relations |
| `internal/history` | parent-chain log/diff and exact-target linklog |
| `internal/store` | immutable objects, current REF observation/CAS, later tag/mv transaction |
| `internal/repository` | candidates, explicit parent selection, closure admission, one-REF publication |
| `internal/cli` | Seal-first selectors, fork/mv parsing, safe branch-aware presentation |
| later Git view adapter | exact path/byte views for staged/commit trees and merge evidence; no domain semantics |

The revision index is derived and disposable. It does not become canonical
state, a named-branch system, a reflog, or a repair engine.

## 19. Documents and accepted decisions affected

ADR acceptance required, and this change applies, coordinated normative
updates:

- ADR 0001: replace REF-relative supersession/stale with revision-leaf stale;
- ADR 0006: remove Link `target_ref` ownership and redesign tag scope;
- ADR 0007: replace HEAD-consistent Cause closure with active-leaf consistency
  while retaining writer/CAS publication;
- ADR 0008: update candidate inspection/link/unlink ownership matching;
- ADR 0009: remove owner REF from material identity while retaining event
  metadata separation;
- ADR 0010: replace named-upstream frontier with a branch-aware definition and
  replace its no-cache-mutation rule with disposable automatic cache rebuild
  plus explicit `--scan`;
- `requirements.md`: define typed revision/Cause relations and remove exclusive
  supersession language;
- `architecture.md`: add derived revision indexing, remove Seal owner checks,
  and replace the speculative Git content reader with native-tree views;
- `storage-format.md`: format-4 payload and parent/link validation;
- `cli.md`: status taxonomy, exact selectors, fork construction, stale/frontier,
  `mv`, and later validation-only sidecar surfaces;
- `integrations.md`: move the first SDK seam from Git content ingest to
  staged/tree/merge-stage `.sealgraph` views;
- `decisions/sealgraph-design.think`: replace the Git-content-source premise
  with same-native-tree view integration;
- `PLAN.pert`, backlog, and alpha checklist: make format-4 revision semantics a
  pre-freeze blocker.

ADR acceptance updates the named normative documents together. Runtime and
tracked dogfood remain unchanged until the separately validated dump/load and
format-4 implementation sequence reaches them.

## 20. Detail consultation gates

Before code reaches each deferred boundary, confirm:

1. Git-sidecar native-tree views, staged validation, validation-only hooks, and
   typed SDK capability gate;
2. tag namespace and REF rename transaction;
3. format-3 dump candidate/tag collision policy;
4. whether `fsck` treats multiple current REFs at one Seal as normal aliases
    and how it reports non-leaf REF heads.
