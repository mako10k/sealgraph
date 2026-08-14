# ADR 0011: REF-independent seals and branching revision DAG

Status: accepted on 2026-08-14.

This ADR accepts the format-4 design contract. The checked-in Go runtime still
implements native format 3 until the separately sequenced implementation and
dump/load work is completed; acceptance does not authorize silent in-place
migration or partial format-4 publication.

## Context

Native v3 commits an owner REF into every Seal and stores both `target_ref` and
`target_seal` in each Link. Parent traversal, tags, selectors, stale, impact,
and closure admission then require the embedded owner name to match.

That contract conflicts with two approved directions:

1. a Seal identifies frozen generated material and exact Cause provenance, not
   external naming state; and
2. a logical REF path must eventually be movable without rewriting immutable
   Seals or invalidating downstream Links.

The v3 parent also carries an exclusive supersession implication. That is too
strong when an old result can have both an updated child and a same-material
child that intentionally preserves the old result. Both are valid revisions;
neither sibling replaces the other.

Removing only the top-level REF would be insufficient. Link identity,
selectors, candidate CAS state, normal admission, stale, frontier, impact,
history, tags, and migration all currently depend on REF ownership. This ADR
therefore adopts one coordinated experimental format break.

The supporting analysis is:

- [`../decisions/2026-08-14-seal-revision-dag.think`](../decisions/2026-08-14-seal-revision-dag.think)
- [`../process/seal-revision-dag-proposal-2026-08-14.md`](../process/seal-revision-dag-proposal-2026-08-14.md)

## Decision

### Experimental format boundary

The target standalone storage is `repository_format = 4`, with
`sealgraph/seal/v4` and `sealgraph/candidate/v4` schemas.

The format-4 runtime does not read formats 1, 2, or 3. It has no dual reader,
ignored legacy fields, in-place migration, automatic repair, or compatibility
mode. Before changing the runtime, the format-3 binary receives a read-only
logical dump. Format-4 load accepts only an empty repository and rebuilds
content-addressed identities explicitly. The remaining dump collision and tag
conversion policy requires a separate approved contract before conversion is
implemented.

This pre-1.0 break favors one simple resulting model over compatibility
machinery. Persisted and public compatibility must be considered after 1.0.

### Canonical format-4 Seal

The required compact canonical member order is:

```text
schema
parent_revision
content
attachments
links
root
draft
```

Illustrative JSON, with the exact encoder rules retained in
`docs/storage-format.md`:

```json
{"schema":"sealgraph/seal/v4","parent_revision":"<seal-id>","content":{"store":"native","type":"blob","id":"<object-id>"},"attachments":[],"links":[{"target_seal":"<seal-id>","message":"basis"}],"root":false,"draft":false}
```

`parent_revision` is always present. It is `null` for an initial revision and
otherwise one exact full parent SealID. It is hash-committed and is never
inferred from a REF.

Format 4 removes, without replacement:

- the top-level owner `ref`;
- Link `target_ref`;
- seal-level actor, time, event message, or equivalent operation metadata;
- stored stale, preference, supersession, branch, or current-head state.

Content bytes, attachment identity and metadata, optional Link messages, root,
and draft remain identity-bearing. Link messages explain one exact Cause edge;
they do not assert actor, authority, trusted time, or approval.

Links are sorted by `(target_seal, message)` using bytewise string order. There
is at most one Link to one exact target SealID; a second Link to that target is
an error even if its message differs. Attachments keep the existing canonical
sort and unique-name rule. Inputs are rejected rather than silently
deduplicated.

Canonical IDs remain full 64-character lower-case SHA-256 hex. The native
object envelope remains:

```text
envelope = "blob " + decimal-size + NUL + payload
ObjectID = sha256(envelope), rendered as lower-case hex
```

and loose objects remain Git-compatible SHA-256 blob objects. `sha256:` is not
part of an ID. Same parentless material intentionally produces the same
SealID under different REF paths. A same-material child produces a different
SealID because `parent_revision` differs.

### Two typed immutable edge relations

The revision edge is:

```text
child.parent_revision = parent SealID
```

It asserts only that the child is a revision derived from the parent. It does
not assert invalidation, replacement, truth, trust, preference, approval, or
same-REF ownership. One Seal has zero or one parent; one parent may have zero
or more children. Sibling revisions are normal.

The Cause edge is:

```text
dependent.links[] = exact upstream SealIDs
```

It asserts that those exact Seals were direct causes of the dependent result.
A Link never follows a REF dynamically. Parent ancestry is not a substitute
for Cause provenance, and parent edges are never traversed as Cause edges.

The product vocabulary uses derive revision, publish revision, advance REF,
parent/child, active descendant, and revision leaf/tip. `supersede` is removed
from canonical product semantics because it implies exclusive replacement.

### REFs and active revision observation

A REF is mutable lookup/publication state outside Seal bytes. It points to one
current SealID. More than one REF may point to the same Seal. REF path grammar,
byte-for-byte loose mapping, implicit intermediate directories, and explicit
file/directory conflict rejection remain unchanged.

For one validated current-head observation `O`:

```text
H_O = deduplicated exact SealIDs stored by all current REF heads
A_O = H_O plus every Seal reached from H_O through parent_revision ancestry
```

`A_O` is the active revision DAG. An ODB-only child left before failed REF CAS,
a tag-only Seal, a Cause-only Seal, or another unreachable object is not an
active revision merely because its bytes exist.

Readers that report multi-REF graph facts capture the complete current
REF/head set, derive against it, buffer output, and revalidate the complete set
before emission. A detected change fails without plausible partial stdout. A
successful observation is not a reservation.

Missing, corrupt, non-canonical, or cyclic parent/Cause state fails closed. No
read command repairs or publishes it.

### Seal selectors

The public selector grammar is:

| Form | Meaning |
| --- | --- |
| `REF` | exact current HEAD of REF |
| `@SEAL_TOKEN` | repository-wide unique ODB prefix that decodes as a canonical Seal |
| `REF@TOKEN` | a Seal selected in an explicit REF UI scope |

`SEAL_TOKEN` and a hexadecimal `TOKEN` contain 4 through 64 lower-case hex
characters. Resolution searches all valid ODB names, requires exactly one
match, and then requires a canonical Seal. Prefixes and selector spelling are
never persisted; only the resolved full SealID is stored or emitted as an
identity receipt.

A hexadecimal `REF@TOKEN` additionally requires the selected Seal to equal or
be a `parent_revision` ancestor of the REF's current HEAD. This is an explicit
scope assertion, not canonical ownership. A non-hex `REF@TOKEN` resolves an
immutable tag in that REF's UI namespace. An unscoped sibling or detached Seal
uses `@SEAL_TOKEN`.

Bare hexadecimal Seal tokens are not selectors because a Git-like REF path may
itself be lower hex. `@` is selector syntax, not part of the SealID, and remains
forbidden in REF and TAGNAME. Existing hex-like TAGNAME reservation remains.

Tags are external immutable aliases and never enter Seal or Link bytes. A tag
target must decode as a canonical Seal, but there is no Seal owner to validate.
Tag creation admissibility, rename-safe namespace storage, and the `mv`
transaction remain a follow-up decision.

### Candidate topology and publication CAS

Candidate v4 separates two fields that v3 conflates as `base`:

```text
parent_revision   = hash-committed derivation parent for the next Seal
expected_ref_head = mutable expected old REF value for publication CAS
```

Candidate inspection displays both separately. Candidate diff compares
identity-bearing material with `parent_revision` while reporting current REF
and `expected_ref_head` relation independently.

An ordinary update to an existing REF records its observed current HEAD in
both fields. The first format-4 slice rejects overriding that REF with an
alternate parent; history is not silently discarded.

Two explicit absent-destination operations are provided:

```text
sealgraph derive NEW_REF --from SOURCE_SELECTOR
sealgraph add NEW_REF --parent SOURCE_SELECTOR --content VALUE
```

Both require the destination REF and destination candidate to be absent and
record `expected_ref_head = null`. A competing publication causes CAS failure.
Source resolution, canonical Seal loading, and parent-chain validation finish
before one candidate file is written; missing, ambiguous, non-Seal, corrupt,
scope-mismatched, and destination-conflict cases leave no partial candidate.

`derive` creates a same-material child. It copies exactly:

- content identity;
- attachments and attachment metadata;
- direct Cause Links and Link messages;
- root;
- draft.

It sets `parent_revision` to the source Seal and does not copy the source's
parent, REF names, tags, stale/cache state, actor/time/event state, or candidate
metadata. Cause Links are retained because parent ancestry does not prove the
new result's direct causes.

`add --parent` creates new material and inherits none of those fields. There
are no selective inheritance flags. Later explicit candidate edits remain
visible in candidate diff.

An active non-leaf, detached historical, or draft Seal may be selected
explicitly as a parent. Parent draft does not automatically propagate to the
child because draft is a per-generation property and parent is not a Cause.
`derive` nevertheless copies the source draft flag; publishing it as a normal
child requires an explicit visible candidate edit. A parent never satisfies
the Cause requirement of a non-root child.

Every standalone mutation remains repository-writer serialized. One `seal`
creates one Seal for one REF. Successful expected-old REF CAS is the
publication linearization point. Candidate cleanup removes only the exact
candidate version that was sealed. Dangling immutable objects after failed
publication are reported and not deleted.

### Root, draft, and normal Cause admission

Root and draft remain hash-committed properties of each Seal generation, not
permanent REF types. Root is an explicit provenance boundary, not truth,
trust, or approval. A root has no Cause Links. A non-root, including a draft,
has at least one Cause Link.

A normal non-draft publication requires:

1. every direct Cause target is an active current revision leaf;
2. every Seal reachable through Cause Links is non-draft;
3. every reachable Cause target is an active current revision leaf;
4. every required immutable object and parent/Cause chain validates; and
5. no generic ignore-validation switch is used.

A draft candidate may preserve active, historical, detached, draft, or
non-draft exact Cause targets, but all referenced immutable objects and graph
structure still validate. Candidate creation may contain stale Causes; normal
publication, not mutable editing, enforces active-leaf closure.

No operation automatically changes draft, chooses a sibling tip, relinks a
Cause, reseals a downstream REF, or repairs a graph.

### Revision and stale facts

For Seal `s` in observation `O`:

- `CURRENT_LEAF`: `s` is a maximal Seal in `A_O`;
- `STALE_REVISION`: `s` is a strict parent ancestor of a current HEAD;
- `HISTORICAL_OR_DETACHED`: `s` is outside `A_O`.

A current REF that points to a non-leaf has a self-stale fact. A current Seal
has direct stale evidence when a direct Cause is not an active current leaf,
and transitive stale evidence when such a Cause occurs deeper in its Link-only
Cause closure. Exact evidence distinguishes an active non-leaf from a
historical/detached target. Human labels may summarize these facts, but must
not report a detached or non-leaf Cause as clean.

Parent edges determine revision ancestry and leafness. Only Link edges
propagate Cause staleness. Otherwise every current child would inherit stale
from its own parent and could never become fresh.

Stale remains derived. No stale marker is added to a Seal, REF, candidate,
tag, or other canonical state.

### Disposable stale index and explicit scan

`.sealgraph/cache/` may hold a disposable revision/Cause index. It is not
canonical, is not committed by an outer Git repository, and never repairs or
overrides canonical state.

A valid cache binds at least repository/schema version, the complete sorted
REF/head snapshot digest, and a cache checksum. Missing, invalid, or
snapshot-mismatched cache state triggers a canonical scan rooted at the
current REF heads and atomic cache replacement. The scan follows parent and
Cause edges needed for the observation; it does not treat unrelated ODB
objects as active children.

Canonical corruption fails closed instead of falling back to cached claims. A
cache write failure alone may return the fully validated result with a stderr
warning. Cached lookup still validates canonical objects required by emitted
evidence.

The orthogonal command forms include:

```text
sealgraph stale [--frontier] [--refs-only] [--scan]
```

`--scan` bypasses cache reads, performs the canonical scan, refreshes cache,
and retains the same stdout contract and membership as a valid-cache query.

### Branch-aware stale frontier

For one observation `O`:

```text
S_O = current REFs whose heads have self, direct, or transitive stale facts
Q_O = { HEAD(r) | r in S_O }
CausePlus(d) = strict transitive closure from d using Link edges only
F_O = { r in S_O | CausePlus(HEAD(r)) intersects Q_O at no Seal }
```

`stale` selects `S_O`; `stale --frontier` selects `F_O`. A stale current REF
blocks a downstream stale REF only when that exact stale current-head Seal is
already in the downstream's frozen Cause closure. Unselected clean or dirty
sibling tips do not alter frontier membership and do not imply a relink choice.

The frontier is an upstream-first review boundary, not readiness, approval,
reservation, obligation, or a batch plan. Candidates do not participate.

The accepted `--refs-only` line protocol remains unchanged: zero or more
bytewise sorted current logical REF paths, each followed by LF, with no
heading, IDs, labels, `CLEAN`, quoting, or plausible partial output. Its set
now also includes self-stale current REF heads according to `S_O`.

### Impact selection and bounded path evidence

`impact` accepts any Seal-resolving selector:

```text
sealgraph impact [--all-paths] [--max-paths N] SELECTOR
```

The selector resolves to exact Seal `h`. Impact reports distinct current-head
Seals whose exact Cause closure first reaches `h` or a `parent_revision`
ancestor of `h`. The selected Seal `h` itself is excluded as a downstream
result. Different selectors resolving to the same `h` produce the same
membership and path evidence; REF spelling is presentation only.

Default presentation emits one path per distinct impacted downstream Seal.
Path distance is Link-edge count. The shortest path wins; equal-length paths
are ordered by the bytewise sequence of full SealIDs. Traversal stops at the
first matching source-ancestry Seal so one physical route is not repeated for
every older matching ancestor.

Multiple current REFs pointing to the same downstream Seal share one graph
calculation and are displayed as sorted aliases.

`--all-paths` enumerates distinct simple Cause paths in `(edge count, full
SealID sequence)` order. `--max-paths N` is valid only with `--all-paths`, is a
positive integer, defaults to 100, and applies separately to each downstream
Seal. Finding another path emits an explicit truncation marker and exits zero;
the result is not presented as complete. Every impacted Seal summary remains
visible.

Path limits never limit membership derivation, complete reachable graph
validation, corruption/cycle detection, or current-head snapshot revalidation.
Impact is structural provenance, not a stale-only work list.

### History and explicit revision choice

`log` follows `parent_revision` IDs and does not compare embedded names.
`diff` may compare any explicitly selected canonical Seals. A mode that claims
one revision line must validate parent ancestry rather than REF ownership.

Link editing and unlink match exact target SealIDs. `--depend-on REF` remains
current-HEAD shorthand; after an upstream moves, removing an old Link uses the
exact selector displayed by candidate inspection. Duplicate detection is by
exact target SealID.

`linklog` compares exact Link target sets between adjacent revisions. A move
from ancestor `S` to descendant `F` may be presented as a repoint. Ambiguous
N:M matching is explicit add/remove rather than an inferred semantic pairing.

An older result is preserved without pin metadata by creating a same-material
sibling child, explicitly relinking one downstream candidate to that child,
and publishing one new downstream Seal. The old downstream Seal remains
immutable and stale in history. No automatic target choice or repair occurs.

### Git sidecar stability boundary

Standalone and Git-sidecar entry points use one
native SHA-256 Seal, Link, REF, object-store, and repository-format contract.
There is no sidecar Seal schema, mode field, Git-backed SealID, or alternate
Cause identity.

The first sidecar slice is a Git-aware view adapter over exact `.sealgraph`
files, not a Git-object content store. It presents worktree, prospective staged
tree, and immutable commit tree as complete read-only byte/path views to the
same native config/object/REF decoders and domain/graph validators used by
standalone. Merge index stages are conflict entries, not complete repository
views; the sidecar relates them to the corresponding BASE/OURS/THEIRS complete
trees before making graph claims. Mutations still target the real worktree
`.sealgraph` through the normal one-REF writer/CAS protocol.

The outer Git repository versions canonical `.sealgraph/config`, immutable
object files, and loose Sealgraph REF/tag files as ordinary tracked files.
Outer Git OIDs identify those tracked file versions only; they are never Seal
IDs or native ObjectIDs. Runtime candidates, locks, caches, and logs are not
canonical tracked state. Git filters, LFS, line-ending conversion, or another
attribute-driven transformation must not rewrite staged canonical bytes.

Sidecar validation of a prospective commit reads the staged result tree,
including unchanged paths inherited from its base, rather than validating the
possibly different worktree. It validates repository layout, every staged
native object by its native ID, REF/tag targets, complete reachable Seal/Cause
state, and the absence of forbidden runtime paths. A concurrent index change
invalidates the observation and fails explicitly.

Hook integration is explicit and validation-only. It may provide a command for
a user-managed pre-commit or pre-push hook, but it does not install itself,
overwrite an existing hook, stage files, create a Seal, advance a REF, relink,
repair, commit, or push. Exact hook installation/dispatch policy remains a
separate public CLI decision.

Merge assistance may inspect `.sealgraph` conflict entries through index
stages and validate the corresponding BASE/OURS/THEIRS complete trees before
explaining revision/Cause relations. Different bytes at one immutable native
object path are corruption; divergent mutable REF/tag targets remain an
explicit conflict. The sidecar does not choose a sibling, manufacture a child,
or treat Git merge success as semantic approval.

Inside the Git view adapter, Git identity remains typed by repository object
format, object type, and full OID. SDK types do not enter native `ObjectID`,
Seal bytes, domain graph, or history. SDK selection occurs only after one
released binary proves its supported SHA-1/SHA-256, worktree, index, tree,
pack, alternate, and linked-worktree capability matrix. Unsupported or missing
objects fail explicitly without a hand-written pack reader, silent Git CLI
fallback, or implicit network fetch.

A historical Git tree whose `.sealgraph/config` declares an unsupported native
repository format fails explicitly. Sidecar history does not reintroduce a
format-3 reader, ignored fields, or automatic migration into the format-4
binary; the operator uses the matching old binary or the approved dump/load
boundary.

Importing an arbitrary Git blob/tree/commit/tag as generated Seal material is
not part of the initial sidecar boundary. Git-blob materialization may be added
later if a concrete workflow justifies it; direct external Git object
references or type-specific projections would require a separate persisted
contract and approval.

Its evidence and deferred implementation details are expanded in the
supporting design proposal.

### Git compatibility boundary

Format 4 retains native Git SHA-256 loose-blob envelope, hash, zlib file, and
object-path compatibility. Explicit Git low-level APIs may read those objects
when configured for the same object format.

This does not import Git commit, tag-object, branch, merge, rebase, checkout,
reflog, pack maintenance, or garbage-collection semantics. Standalone code
still does not discover or read `.git`. A Git SDK remains appropriate later
for physical sidecar Git reading, not for defining Sealgraph revision or
standalone mutation semantics.

## Consequences

- Seal and Link identities survive REF path rename because neither contains a
  current or historical REF name.
- Two REFs may intentionally point to the same Seal without making aliases part
  of immutable identity.
- All format-3 SealIDs change, and different name-owned v3 Seals may collapse to
  one format-4 SealID.
- Parent commits revision derivation while permitting forks and same-material
  siblings.
- A new active child makes its parent non-leaf but never invalidates a sibling.
- Normal Cause closure stays conservative without preventing explicit
  historical parent selection.
- Stale/frontier/impact are derived from exact Seal edges and coherent current
  REF observations rather than stored upstream names.
- Disposable caching improves repeated graph queries without becoming truth or
  repair state.
- Candidate schema and inspection become clearer because semantic parent and
  publication CAS expectation are different fields.
- Git sidecar adds read views and validation around the same native files; it
  does not create a second Sealgraph storage or identity mode.
- Implementation must change domain, canonical fixtures, graph algorithms,
  history, repository orchestration, CLI, dogfood, and migration together; a
  partial owner-check removal is invalid.

## Decisions intentionally deferred

This ADR does not decide:

- rename-safe tag namespace storage, tag creation ancestry, or the atomic
  `sealgraph mv REF1 REF2` transaction;
- format-3 dump candidate policy, tag collisions, non-injective mapping output,
  or exact format-4 load publication boundary;
- final `fsck` presentation for multiple current REF aliases and non-leaf REF
  heads;
- signatures, trusted timestamps, remote storage, daemon/server, or MCP;
- Git source-material import/projections, exact hook/setup CLI, existing-hook
  coexistence, historical selector presentation, or explicit Git index merge
  resolution transaction;
- link kinds, automatic branch preference, automatic relink/reseal, recursive
  repair, or batch seal.

Each deferred persisted or public decision above requires its own approval
before format-4 implementation reaches that boundary.

## Effect on earlier ADRs

Accepted ADR files remain immutable historical records. This ADR's conflicting
decisions take precedence as follows:

| ADR | Effect |
| --- | --- |
| 0001 | Retain immutable exact-Cause Merkle provenance, one-REF seal, and explicit repair; replace owner-relative supersession and stale with branching revision/leaf semantics. |
| 0002 | Unchanged: standalone remains the default and never detects Git. |
| 0003 | Unchanged: low-level Git-compatible native loose storage remains. |
| 0004 | Retain the separate executable/product surface; define it as Git-aware read views and validation over the same native `.sealgraph` format. |
| 0005 | Historical only; already superseded for experimental formats. |
| 0006 | Retain full-hex SHA-256 IDs, ODB prefixes, immutable tags, TAGNAME encoding, Link messages, and Git conformance; replace owner validation, `target_ref`, REF-keyed duplicate/order rules, and selector scope. |
| 0007 | Retain repository-wide writer serialization, one-REF CAS publication, unchanged-candidate cleanup, and non-draft closure; replace REF-HEAD consistency with active-leaf Cause consistency and keep parent admission separate. |
| 0008 | Retain the separate candidate namespace, explicit discard, and binary-safe output; replace `base` with parent/CAS fields and REF-keyed unlink with exact-Seal unlink. |
| 0009 | Retain exclusion of actor/time/event metadata and explicit separately sealed attestations; remove owner REF, rename parent to `parent_revision`, and introduce format 4. |
| 0010 | Retain factual stale, coherent observations, candidate exclusion, and stable `--refs-only`; add self-stale membership, replace named-upstream frontier, permit disposable cache refresh, and add `--scan`. |

## Acceptance record

The operator explicitly accepted this complete ADR, including the Git sidecar
stability boundary, on 2026-08-14 after llmthink and
symmetry/consistency/completeness review. Implementation remains separately
sequenced and must not partially mix format 3 and format 4.
