# Sealgraph requirements

Status: normative format-4 contract accepted by ADR 0011. The checked-in Go
runtime implements the canonical/candidate, selector, explicit load, active
revision/Cause graph, history, impact, REF manifest, scoped-tag, and move core.

## 1. Purpose

Sealgraph is a provenance-sealing system for logical content.

It MUST make it possible to answer:

1. What content was sealed?
2. Which exact upstream seal generations were used as its basis?
3. What attachments were part of that sealed state?
4. From which exact parent revision was it derived?
5. Which semantic flags and direct provenance relations were sealed?
6. Which current REF heads became stale because their revision or Cause
   provenance is no longer at an active revision leaf?
7. Through which dependency paths did that impact propagate?

Sealgraph is not a general-purpose VCS.

## 2. Core concepts

### 2.1 REF

A REF is a movable logical lookup/publication name, not immutable Seal
identity and not a Git branch.

Each REF has at most one current HEAD seal.

Multiple REFs MAY point to the same Seal. Moving or renaming a REF MUST NOT
rewrite a Seal or Link.

One canonical REF manifest stores that REF's current HEAD and complete
immutable tag namespace. Path-form REFs, including a REF and another REF for
which it is a slash-prefix, MAY coexist; spelling does not imply hierarchy or
recursive behavior. `mv OLD_REF NEW_REF` moves exactly one manifest to an
absent destination. It MUST NOT move candidate state, retain an old-name alias,
or rewrite a Seal, Link, HEAD, or tag target.

### 2.2 Blob

Content and attachments are stored/read as immutable content-addressed blobs.

Standalone operation MUST NOT require a working file corresponding to a REF.

### 2.3 Seal

A seal is an immutable snapshot of generated material, exact direct Cause
provenance, and one optional revision parent. It may be published as one REF's
HEAD, but the REF name is not part of the Seal.

A seal MUST commit to:

- schema/format version,
- parent revision seal identity or explicit absence,
- content identity,
- attachment identities plus stable attachment metadata,
- dependency links,
- root/draft state.

A core seal MUST NOT persist who, when, or why the seal operation happened.
Seal-level `actor`, `created_at`, event `message`, and equivalent operation
metadata are outside material/provenance identity. When needed, such a claim is
ordinary separately sealed content linked to its exact subject generation.

The required `parent_revision` is `null` for an initial revision and otherwise
one exact full SealID. It means only that the new Seal is a revision derived
from that parent. It MUST NOT imply replacement, invalidation, preference,
truth, trust, approval, or same-REF ownership. One parent MAY have multiple
children, and siblings are equally valid revision tips.

A Seal contains no current or historical REF name. It MAY be reused as the
HEAD, parent, Link target, or tag target in any explicitly valid scope without
changing identity.

One `seal` invocation MUST create at most one new seal for exactly one REF.

There MUST NOT be a `seal --all` or equivalent batch-approval operation in the core product.

### 2.4 Link

A link is a provenance edge from one seal to an exact upstream seal generation.

Links form an N:M directed acyclic graph across seals.

A persisted link MUST contain a concrete target seal identity.

Native format 4 has one domain-independent Cause edge and no persisted link
kind. A link MAY carry an edge-specific message explaining why that exact
upstream generation is a dependency. The link message is part of the seal
identity. It describes the dependency relation and does not assert an actor,
authority, trusted time, or seal-operation event.

`--depend-on UPSTREAM` is command shorthand that resolves the current HEAD at operation time. The persisted seal MUST NOT contain a dynamic HEAD pointer.

The CLI MUST also support explicit historical generation selection.

Format 4 selector forms are `REF`, repository-wide `@SEAL_TOKEN`, and scoped
`REF@TOKEN`. A hexadecimal Seal token is 4 through 64 lower-case hex
characters, resolves uniquely across the native ODB, and must decode as a
canonical Seal. `REF@hex` additionally asserts that the selected Seal is the
REF's current HEAD or a `parent_revision` ancestor of it. `REF@non-hex`
resolves an immutable tag in that REF's UI namespace. Only the resolved full
SealID is persisted.

A tag is an immutable external alias for one exact Seal. It is not part of
Seal or Link bytes and MUST NOT become a dynamic link, movable branch, or
approval claim. Tags are stored in the scoped REF manifest and move atomically
with that REF. Recreating the same binding is idempotent; retarget, delete,
force, and unscoped tag creation are absent.

### 2.5 Root

A root seal generation explicitly declares a provenance boundary for that
immutable generation. Root is an identity-bearing seal attribute, not a
permanent type of the logical REF.

Root MUST NOT be inferred merely from an empty dependency list.

Root does not mean “true”, “trusted”, or “approved by an external authority”.

Successive generations of the same REF MAY explicitly change between root and
non-root. Such a change creates a new seal identity and MUST remain visible in
history/diff; it never changes the root state of an older seal. Changing the
root attribute MUST NOT add or remove dependency links automatically.

A non-root sealed candidate normally requires at least one upstream dependency.

### 2.6 Draft

Draft is an explicit semantic state for provisional sealing.

A draft may intentionally depend on a non-HEAD upstream seal.

Draft MUST remain visible in status/show/log output.

## 3. Revision DAG and stale propagation

Seals and Links are immutable. Publishing a new revision moves exactly one REF
HEAD and never changes an older downstream Link.

For one coherent current-head observation, the active revision DAG is the
deduplicated set of current REF HEAD SealIDs plus every Seal reached through
`parent_revision` ancestry. An object-store-only child, tag-only Seal,
Cause-only Seal, or failed-publication dangling object is not active merely
because its bytes exist.

An active Seal is `STALE_REVISION` when it has an active strict descendant. A
current REF that points to such a non-leaf has self-stale state. Sibling leaves
do not make one another stale. A Seal outside the active revision DAG is
historical or detached, not current-clean.

A current Seal is `STALE_DIRECT` when an exact direct Cause Link target is not
an active current revision leaf, including an active non-leaf or a
historical/detached target. It is `STALE_TRANSITIVE` when no direct target is
stale but a deeper Link-only Cause target is not an active current leaf.
Parent edges decide revision leafness and MUST NOT be traversed as Cause edges.

Staleness MUST be derived from canonical Seals and one coherent complete REF
head observation. It MUST NOT be authoritative persisted state or depend on
mutable candidates. A changed observation fails without partial stdout.

The product MUST expose the complete stale current-REF set and an upstream-
first exact-Cause review frontier. A stale current REF is blocked by another
stale current head only when that exact head Seal appears in its strict
Link-only Cause closure. Unselected descendant tips and candidates do not
affect frontier membership. These results are factual observations, not
approval, mandatory work, seal admissibility, reservation, or a batch plan.

A disposable derived cache MAY accelerate stale queries when bound to the
repository/schema version and complete sorted REF/head snapshot. Missing,
invalid, or mismatched cache state triggers canonical full scan and atomic
refresh; canonical corruption fails closed. `--scan` MUST bypass cache reads.
Cache state is not canonical and is not committed to an outer Git repository.

## 4. Seal admissibility

A normal non-draft seal MUST reject a candidate unless every direct and
reachable Cause target is a non-draft active revision leaf in one coherent
current-head observation.

This rule exists to force unresolved upstream review to progress explicitly from upstream to downstream.

Explicit draft/historical workflows MAY seal against active, historical,
detached, draft, or non-draft exact Cause targets, but those relations remain
observable and MUST NOT be reported as normal-clean.

A draft candidate MAY depend on current or historical draft/non-draft seals.
Draft is distinct from stale and MUST NOT be propagated, relinked, or resealed
automatically. To depend on provisional provenance, the operator explicitly
keeps the dependent candidate draft.

Revision-parent admissibility is separate from Cause admissibility. An active
non-leaf, detached historical, or draft Seal MAY be selected explicitly as a
revision parent. A parent does not satisfy the required Cause of a non-root
Seal and parent draft state does not automatically propagate. There is no
generic ignore-validation escape hatch.

## 5. Attachments

Content may include zero or more named attachments.

Attachment bytes are immutable blobs.

A seal MUST commit to each attachment's blob identity and stable semantic metadata such as name and media type.

Renaming an attachment changes the seal state even if the attachment bytes are unchanged.

An attachment is contained evidence/artifact data. A link is an external provenance relation. The two MUST remain semantically distinct.

## 6. Working candidate

`add`, `derive`, `link`, `unlink`, `attach`, and `detach` edit the next
candidate state for one destination REF.

`add` MAY specify dependencies atomically with content creation/update:

```sh
sealgraph add DESIGN-001 \
  --content '...' \
  --depend-on REQ-001 \
  --depend-on POLICY-001@<seal-id>
```

`link` remains necessary for relinking without content changes.

Working candidate state is not a seal and is not authoritative history.

Format-4 candidates keep revision topology and publication coordination
separate:

- `parent_revision` is the hash-committed parent of the next Seal;
- `expected_ref_head` is mutable expected-old state for destination REF CAS.

An ordinary update of an existing REF records its observed current HEAD in both
fields. `derive NEW_REF --from SOURCE` creates a same-material candidate for an
absent destination, copies content, attachments, direct Cause Links/messages,
root, and draft, and sets `parent_revision` to SOURCE.
`add NEW_REF --parent SOURCE --content ...` creates new material with no
inheritance. Both require destination REF and candidate absence and record
`expected_ref_head = null`.

Candidate inspection MUST remain distinct from immutable `REF@TOKEN`
selection. The standalone CLI MUST allow one candidate to be shown, compared
with its recorded `parent_revision`, and explicitly discarded. Current REF and
`expected_ref_head` relation is reported separately. Candidate inspection and
diff MUST NOT automatically rebase, relink, repair, or seal it.

Discard removes only one exact candidate state. It MUST NOT move a REF, delete
immutable objects, recurse through hierarchical REFs, or report a missing or
unsafe target as successful. Explicit discard MUST remain possible when the
candidate representation is corrupt.

`unlink` removes exactly one dependency edge identified by its resolved exact
target SealID. A bare REF is current-HEAD lookup shorthand and therefore does
not match an older stored target after that REF advances. It MUST NOT change
content, root/draft state, another dependency, or create a seal.

Standalone mutations MUST use repository-wide writer coordination. Cooperative
writers execute serially. A seal publishes at the successful expected-old CAS
update of its one target REF and MUST NOT clear a candidate version other than
the one it sealed.

## 7. Required inspection commands

The product is expected to provide:

- `show`
- `diff`
- `status`
- `log`
- `linklog`
- `impact`
- `graph`
- `stale`
- `fsck`

`diff` MUST be capable of representing content, attachment, link, and material metadata differences.

`status` MUST distinguish at least candidate modifications/unsealed state,
draft, self-stale revision, and direct/transitive Cause staleness.

`stale` MUST offer a stable REF-only line form for shell composition. It emits
one valid logical REF plus LF per selected current head in bytewise lexical
order, with no header or status fields, and emits zero bytes for an empty set.
Candidate inspection remains outside this command.

`impact` accepts every Seal-resolving selector. By default it emits one
deterministic shortest Link-edge path per distinct impacted current-head Seal;
equal lengths use the bytewise lexical full-SealID sequence. Explicit
`--all-paths` is bounded by positive `--max-paths N`, default 100 per impacted
Seal, with an explicit truncation marker. Presentation limits MUST NOT limit
membership derivation, integrity validation, or snapshot revalidation.

Default human inspection MUST NOT emit arbitrary content or metadata bytes
directly. It MUST use bounded, unambiguous escaping. Exact content extraction
MAY be provided only by an explicit bytes-only mode whose stdout contains no
mixed metadata or added newline.

## 8. Intentionally absent VCS semantics

Core sealgraph MUST NOT implement Git-like:

- merge
- rebase
- branch
- checkout
- cherry-pick

Multiple direct causes are expressed through provenance Links, not merge
commits.

## 9. Standalone initialization

`sealgraph init` MUST always initialize standalone mode.

It MUST NOT:

- detect `.git`,
- change behavior because it runs inside a Git working tree,
- suggest or activate Git sidecar automatically.

Standalone canonical reads MUST use `.sealgraph` only.

### 9.1 Standalone Git low-level compatibility

Standalone Git compatibility is limited to object identity/envelope
conformance and safe read-only low-level forensic interoperability. Native
objects MUST retain the documented Git SHA-256 loose-blob envelope and identity
so an explicitly configured Git SHA-256 low-level object API can read them
without identity disagreement, silent translation, or mutation.

This compatibility MUST NOT make `.sealgraph` a Git repository or import Git
commit, branch, merge, checkout, reflog, garbage-collection, maintenance, or
porcelain semantics. A sealgraph adapter or conformance tool used in an
incompatible object-format context MUST reject it rather than guess or
translate. In particular, the native SHA-256 object directory is not an
alternate object directory for a SHA-1 repository.

Standalone product code continues to avoid `.git`. Explicit temporary Git
conformance tests do not change that lifecycle boundary.

## 10. Git sidecar

Git sidecar is a separate product surface exposed as `git sealgraph ...` through a `git-sealgraph` executable.

Sidecar uses the same native `.sealgraph` Seal, Link, REF, object-store, and
repository-format contract. It MUST NOT define a sidecar Seal schema or use an
outer Git OID as a native identity.

Sidecar MAY present outer-Git worktree, prospective staged tree, and immutable
commit tree as complete read-only exact path/byte views to the same native
decoders and domain/graph validators. Merge index stages are conflict entries,
not complete repository views; graph claims require the corresponding
validated BASE/OURS/THEIRS complete trees.

Sidecar publication writes the real worktree `.sealgraph` using the same
one-REF writer/CAS protocol. Git tree/index/history views are read-only and
MUST NOT create candidates, objects, REFs, cache, or repairs.

Sealgraph provenance semantics remain independent from Git commit semantics.

Git commits MUST NOT automatically create seals.

Git merge MUST NOT automatically repair stale provenance.

Git-sidecar MAY provide three-way conflict inspection/resolution assistance for sealgraph REF conflicts.

Automatic semantic merging or fabricated approval is forbidden.

Hook integration MUST be explicit, opt-in, and validation-only. A validation
hook MUST inspect the prospective staged result tree rather than a potentially
different worktree, and MUST NOT install itself, overwrite an existing hook,
stage, seal, advance a REF, relink, repair, commit, push, or treat success as
approval.

Canonical `.sealgraph` files tracked by outer Git MUST reach the staged tree
without LFS, clean/smudge filtering, working-tree encoding, or line-ending
transformation. Runtime candidates, locks, cache, logs, and temporary paths
MUST NOT be staged. Missing partial-clone objects and unsupported Git or native
repository formats fail explicitly without implicit network fetch, dual
reader, or automatic migration.

## 11. Merge-friendly metadata

A `.sealgraph` directory tracked by an outer Git repository SHOULD merge predictably:

- immutable objects should be additive,
- one logical REF should use one small mutable ref file,
- canonical native storage should avoid pack/repack churn,
- canonical native refs should avoid packed-refs-like aggregation.

When the same logical REF advances differently on two Git branches, an outer Git merge conflict on that REF file is desirable.

When different REFs advance independently, Git should normally merge them without conflict.

## 12. Security

Sealgraph MUST NOT treat secret plaintext as a normal metadata field.

Repository docs/tests MUST NOT include real credentials.

Integration with secdat is optional and explicit; core operation does not depend on secdat.

## 13. Explicit experimental migration boundary

The final format-3 binary before the runtime transition provides:

```sh
sealgraph dump --format logical-v1
```

The command emits one deterministic canonical
`sealgraph/logical-dump/v1` document and MUST NOT change repository or Git
state. Current REF heads and immutable tag targets root the exported parent and
Cause closure. Referenced content/attachment bytes are included exactly;
valid loose objects outside that logical graph are reported by identity and
not copied.

Any candidate entry, corrupt object, invalid REF/tag attribution, invalid
Seal/graph, or changed final observation MUST reject the dump without
plausible stdout. Candidate state is neither omitted nor translated.

The format-4 load MUST use only an absent `.sealgraph` target, complete
staging validation, atomic no-replace publication, and an explicit complete
old-to-new SealID receipt. It MUST NOT merge, replace, repair, or silently drop
tags. Every logical tag record is rewritten through the same complete SealID
mapping and published inside its REF manifest.

## 14. Exact content input and explicit path manifests

`add --content-file PATH|-` MUST preserve exact bytes. File input accepts only
a regular non-symlink file; stdin is read exactly. Conflicting content sources,
missing paths, directories, symlinks, devices, sockets, and FIFOs MUST fail
before candidate mutation. The source path is not persisted automatically and
`add` never seals automatically.

The standalone manifest builder accepts only explicit caller-supplied relative
paths and one explicit source identity. It MUST NOT infer Git identity,
repository root, file sets, globs, directory recursion, or environment-derived
metadata. It emits a versioned deterministic path/size/SHA-256 claim, not an
attachment or proof that the named files were imported. Input order MUST NOT
affect bytes; file bytes, semantic paths, and source identity MUST affect the
resulting manifest blob identity.
