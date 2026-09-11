# Sealgraph requirements

Status: normative format-5 and format-6 contract. Accepted ADRs 0023, 0025,
0026, and 0027 define the format-5 boundary. Accepted ADRs 0029, 0030, and
0031 add the format-6 Cause Link metadata, storage/migration, and CLI/output
boundary. New repositories initialize as format 5; format 6 is entered only by
the explicit 5-to-6 migration. Both fail closed on format 4 in ordinary use.

## 1. Purpose

Sealgraph is a provenance-sealing system for logical content.

It MUST make it possible to answer:

1. What content was sealed?
2. Which exact upstream seal generations were used as its basis?
3. What attachments were part of that sealed state?
4. Which observers asserted that a target supersedes which exact revisions?
5. Which semantic flags and observer-local provenance claims were sealed?
6. Which current REF heads became stale because their observed revision or Cause
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

Content, attachments, Material, Provenance, and Seal values are stored/read as
immutable content-addressed Blobs.

Standalone operation MUST NOT require a working file corresponding to a REF.

### 2.3 Seal

A Seal is an immutable typed Blob containing exactly one Material BlobID and
one Provenance BlobID. It may be published as one REF's HEAD, but REF names are
not part of Material, Provenance, or Seal identity.

A seal MUST commit to:

- Material identity, which commits to content BlobID and named attachment
  BlobIDs plus stable metadata; and
- Provenance identity, which commits to root/draft and complete Cause Links.

A core seal MUST NOT persist who, when, or why the seal operation happened.
Seal-level `actor`, `created_at`, event `message`, and equivalent operation
metadata are outside material/provenance identity. When needed, such a claim is
ordinary separately sealed content linked to its exact subject generation.

Formats 5 and 6 have no intrinsic parent field. Revision evidence exists only inside a
Cause Link made by one immutable observer. A target MAY have multiple asserted
previous revisions and multiple observers; branching is valid. An assertion
MUST NOT imply preference, truth, trust, approval, or same-REF ownership.

A Seal contains no current or historical REF name. It MAY be reused as the
HEAD, parent, Link target, or tag target in any explicitly valid scope without
changing identity.

One `seal` invocation MUST create at most one new seal for exactly one REF.

There MUST NOT be a `seal --all` or equivalent batch-approval operation in the core product.

### 2.4 Link

A Link is a whole observer-local provenance record from one Seal to one exact
target Seal generation.

Links form an N:M directed acyclic graph across seals.

A persisted link MUST contain a concrete target seal identity.

A format-5 Cause Link contains exactly `target_seal`, the sorted duplicate-free
`previous_revision_seal_of_target_seal` array, and the sorted duplicate-free
`messages` array. The previous array is an assertion about that target made by
the containing observer. Link messages are identity-bearing but do not assert
actor, authority, trusted time, or a seal-operation event.

A format-6 Cause Link adds a required `metadata` array. Each entry contains a
unique non-empty UTF-8 `namespace`, a nullable non-empty UTF-8 `schema`, and one
bounded canonical JSON `value`. Entries sort bytewise by namespace. Metadata is
opaque to core, identity-bearing, and observer-local; it MUST NOT imply a new
edge, authority, validation success, approval, or trusted event. Secret
plaintext remains forbidden. Format-5 Cause Links project to empty metadata
only in shared inspection views; their historical bytes and IDs never change.

`--target TARGET`, `--previous PREVIOUS`, and related selector inputs resolve
to exact full SealIDs before Candidate persistence. Dynamic HEAD pointers and
selector spelling MUST NOT be persisted.

The CLI MUST also support explicit historical generation selection.

Selector forms are `REF`, repository-wide `@SEAL_TOKEN`, and scoped
`REF@TOKEN`. A repository-wide `SEAL_TOKEN` is 4 through 64 lower-case hex
characters, resolves uniquely across the native ODB, and must decode as a
canonical Seal generation permitted by the current repository format. Format 5
therefore accepts Seal v5 only; format 6 accepts valid Seal v5 and Seal v6
objects with their exact required Provenance pairing. `REF@hex` instead resolves
uniquely within the REF's current HEAD plus observed structural revision closure;
unrelated loose objects do not participate in that scoped prefix match.
`REF@non-hex`
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

## 3. Observed revision graph and stale propagation

Seals and Links are immutable. Publishing moves exactly one REF HEAD and never
changes an older Link. The observation set is the fixed point starting from
all current REF heads and following both Cause targets and every previous
revision asserted by those Links. Every included assertion retains its exact
observer Seal and Provenance identity.

The structural revision graph is the union of included target-to-previous
assertions. Its union with Cause edges MUST be acyclic and contain no self
edge. Revision activity starts from current heads and follows structural
revision edges only. Cause reachability does not itself make a Seal active.

An active Seal is `STALE_REVISION` when it has an active structural child. A
current REF that points to such a non-leaf has self-stale state. Sibling leaves
do not make one another stale. A Seal outside the active revision DAG is
historical or detached, not current-clean.

A current Seal is `STALE_DIRECT` when an exact direct Cause Link target is not
an active current revision leaf, including an active non-leaf or a
historical/detached target. It is `STALE_TRANSITIVE` when no direct target is
stale but a deeper Link-only Cause target is not an active current leaf.
Revision assertions decide revision leafness and MUST NOT be traversed as
Cause edges.

Staleness MUST be derived from canonical Seals and one coherent complete REF
head observation. It MUST NOT be authoritative persisted state or depend on
mutable candidates. A changed observation fails without partial stdout.

The product MUST expose the complete stale current-REF set and an upstream-
first exact-Cause review frontier. A stale current REF is blocked by another
stale current head only when that exact head Seal appears in its strict
Link-only Cause closure. Unselected descendant tips and candidates do not
affect frontier membership. These results are factual observations, not
approval, mandatory work, seal admissibility, reservation, or a batch plan.

A disposable cache MAY accelerate graph queries only when it is semantically
equivalent to a canonical scan and bound to the complete observation. Missing,
invalid, unsafe, or mismatched cache state never weakens validation. `--scan`
MUST produce the same answer while bypassing cache reads. Cache state is not
canonical and is not committed to an outer Git repository.

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

Revision evidence is part of each Cause record, not a separate Candidate or
Seal parent. A previous revision may be active, detached, historical, or
draft, but the target selected as the normal non-draft Cause must satisfy the
normal active-leaf rule. There is no generic ignore-validation escape hatch.

## 5. Attachment preservation

Format-5 Material and Candidates may contain zero or more named attachments.

Attachment bytes are immutable blobs.

A seal MUST commit to each attachment's blob identity and stable semantic metadata such as name and media type.

Renaming an attachment changes the seal state even if the attachment bytes are unchanged.

An attachment is contained evidence/artifact data. A link is an external
provenance relation. The two remain semantically distinct and MUST
NOT be silently converted into one another.

As accepted by ADR 0021, the product MUST NOT add attachment mutation commands
for format 5. It MUST continue to decode, validate, inspect, compare, preserve,
dump/load, and include existing attachments in canonical identity. New related
material should use primary content, an independently sealed content result
plus an exact Link, or explicit manifest content according to its semantics.

Machine-local source selection SHOULD remain in explicit local binding state
where practical. Binding is non-canonical and MUST NOT replace portable
content identity, exact provenance, or another semantic claim that a different
repository or machine must verify.

## 6. Working candidate

`add`, `link`, `unlink`, and, in format 6, `link-metadata set/remove` edit the
next Candidate state for one destination REF. Attachment mutation commands are
intentionally absent.

`add` MAY specify dependencies atomically with content creation/update:

```sh
sealgraph add DESIGN-001 --content '...' --non-root \
  --target REQ-001 --no-previous
```

`link` remains necessary for relinking without content changes.

Working candidate state is not a seal and is not authoritative history.

Format-5 Candidates are parentless. `expected_ref_head` is publication CAS
state only and never supplies graph meaning. A Candidate stores content BlobID,
attachments, root/draft, and complete Cause Link records.

A new root requires `--root --clear-cause-links`. A new non-root requires
`--non-root` and one complete target group containing exactly one of at least
one `--previous` or `--no-previous`. Existing edits preserve omitted root mode
and omitted Cause group. `--root --clear-cause-links` atomically changes the
mode and clears Links; `--non-root` must leave at least one Link.

`link` creates or replaces the complete record for one exact target. `unlink`
removes exactly one target record. Neither operation unions omitted values,
associates positional arguments, changes another target, or creates a Seal.
Multiple targets require separate reviewed Candidate mutations. `derive` and
`add --parent` are absent.

In format 6, legacy `add`/`link` authoring preserves existing metadata on the
same exact target and creates empty metadata for a new target. Only
`link-metadata set` adds or replaces one complete namespace entry, and only
`link-metadata remove` removes one existing namespace. Both operations require
an existing Candidate or REF baseline and exact Cause target, preserve every
unselected field, validate the complete successor graph, and replace one
expected-old Candidate version. They never migrate, seal, move a REF, interpret
metadata meaning, or perform a batch mutation.

In a format-6 repository, reading Candidate v5 is side-effect free and projects
empty Link metadata in memory. A successful separately authorized Candidate
mutation writes the complete successor as Candidate v6. Publication never
publishes Candidate v5 bytes directly or rewrites that file as a hidden
preliminary action: it constructs and validates the exact in-memory Candidate-v6
projection, creates Provenance v2 and Seal v6, and then follows the existing
expected-old observation and one-REF CAS workflow.

Candidate inspection MUST remain distinct from immutable `REF@TOKEN`
selection. The standalone CLI MUST allow one candidate to be shown, compared
with its explicit publication baseline (`expected_ref_head`), and explicitly
discarded. The current REF relation is reported separately. Candidate
inspection and comparison MUST NOT automatically rebase, relink, repair, or
seal it.

Discard removes only one exact candidate state. It MUST NOT move a REF, delete
immutable objects, recurse through hierarchical REFs, or report a missing or
unsafe target as successful. Explicit discard MUST remain possible when the
candidate representation is corrupt.

An authoring mutation MUST resolve every selector against one coherent
repository observation, validate the complete prospective combined graph, and
revalidate before Candidate persistence. Failure MUST leave the prior exact
Candidate bytes unchanged.

Standalone mutations MUST use repository-wide writer coordination. Cooperative
writers execute serially. A seal publishes at the successful expected-old CAS
update of its one target REF and MUST NOT clear a candidate version other than
the one it sealed.

## 7. Required inspection commands

The product is expected to provide:

- `show`
- `compare`
- `status`
- `log`
- `linklog`
- `impact`
- `graph`
- `stale`
- `fsck`

`compare` requires exactly two explicit selectors and MUST represent Material,
Provenance, content, attachment, root/draft, and complete Cause-record
differences without inferring a predecessor.

`status` MUST distinguish at least candidate modifications/unsealed state,
draft, self-stale revision, and direct/transitive Cause staleness.

`stale` MUST offer a stable REF-only line form for shell composition. It emits
one valid logical REF plus LF per selected current head in bytewise lexical
order, with no header or status fields, and emits zero bytes for an empty set.
Candidate inspection remains outside this command.

`impact` accepts every Seal-resolving selector. Revision proof uses either all
observed assertions or the explicit repeatable `--asserted-by` observer set.
Each result carries the Cause path, matched revision, exact revision edges,
supporting assertion sources, and scope-relative target observations. Default
output emits one deterministic first-match path per impacted head; explicit
`--all-paths` is bounded by positive `--max-paths N`, default 100 per head.
Presentation limits MUST NOT limit membership, validation, or revalidation.

`log` traverses branching Cause-scoped revision assertions and reports each
reachable Seal once by minimum depth unless bounded complete paths are
explicitly requested. `linklog` compares exact Cause records across each
structural revision edge and never infers a repoint.

Successful format-5 machine output uses `show/v2`, `candidate-show/v2`,
`candidate-compare/v2`, `status/v3`, `stale/v2`, `graph/v2`, `impact/v2`,
`log/v2`, `linklog/v2`, `compare/v2`, and `fsck/v2`. In format 6, every listed
schema containing Links, assertion sources, typed schema generations, or
comparisons advances to `/v3`; `status/v3` and `stale/v2` remain unchanged.
Format-6 shared Link records include complete metadata, typed Seal/Provenance
generations, and detailed per-target/per-namespace comparison records as fixed
by ADR 0031. A changed meaning MUST NOT be emitted under an older schema.

Format-6 Assessment-free change identity uses
`sealgraph/upstream-change/v2`. It retains ADR 0028's established member order,
and `after_cause_links` contains complete canonical format-6 Cause Links,
including metadata. Format-6 operations emit and compare only v2 change IDs,
even when every metadata array is empty; any metadata addition, replacement, or
removal therefore changes `change_id`. Existing immutable v1 change and
Assessment Blobs are not rewritten, and Assessment-reference persistence
remains separately gated.

Default human inspection MUST NOT emit arbitrary content bytes directly. It
MUST use bounded, unambiguous escaping. In format 6, selected Link metadata is
shown completely as bounded canonical JSON with namespace and schema; control
characters remain escaped and format-5 projected emptiness is labeled. Exact
content extraction MAY be provided only by an explicit bytes-only mode whose
stdout contains no mixed metadata or added newline.

Inspection output MUST default to width-aware human presentation when stdout
is a terminal and to a versioned structured machine document when stdout is a
known non-terminal destination. An explicit supported format option overrides
destination detection. Human object identities MAY be abbreviated for display;
machine documents and exact identity receipts retain complete identities.
Bytes-only and stable narrow line protocols remain explicit separate formats.

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

Git sidecar is a deferred, separately decided product surface. It is not part
of the standalone beta product or release artifacts. If adopted, it
is exposed as `git sealgraph ...` through a `git-sealgraph` executable.

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

## 13. Explicit format-4 migration boundary

Ordinary format-5 operations MUST reject an existing exact format-4 config
before mutation with stable code `FORMAT4_REQUIRES_MIGRATION`. They MUST name
the isolated extract/load sequence and MUST NOT provide a dual reader,
in-place rewrite, lazy upgrade, or legacy-parent fallback.

The format-5 binary provides one migration-only read-only format-4 extractor:

```sh
sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json
```

Ordinary format-5 operations still reject format 4. The format-5 runtime imports
only that isolated document into an absent target:

```sh
sealgraph load --format universal-blob-v1 < repository.dump.json
```

The canonical document schema is `sealgraph/universal-blob-migration/v1`.
It contains exact referenced object bytes, canonical format-4 Seal payloads,
complete REF/tag mappings, semantic projection records, excluded object IDs,
and the constant excluded-state categories. The importer MUST verify old
payload bytes and old IDs using a migration-only format-4 codec with no live
repository interface.

Projection creates Material, Provenance, and Seal Blobs. Old global parent
meaning is materialized only in actual observer Cause Links. Unobserved parent,
collapse-dropped revision, and merged Cause records MUST be retained in the
document and receipt and warned with exact counts. No synthetic observer,
Cause Link, REF, tag, or fallback field may be invented.

Any Candidate entry, corrupt object, invalid REF/tag, graph cycle, or changed
extract observation MUST reject extraction. The extractor MUST expose no source
mutation operation and MUST NOT inspect Git. Load MUST fully validate and
project before target creation, stage a complete format-5 repository and canonical
`sealgraph/universal-blob-load-receipt/v1`, fsync nested state, publish with an
atomic no-replace rename, fsync the parent, and read back fsck, repository
digest, and receipt. Pre-publication, durability-uncertain, readback-failed,
and receipt-undelivered states MUST be distinguished and MUST NOT authorize an
automatic retry, delete, or repair.

Migration MUST NOT modify or mark the retained format-4 source. Rollback means
explicitly selecting that retained source with its format-4 runtime, not an
in-place downgrade.

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

### 14.1 Local source binding

As accepted by ADR 0019, a logical REF MAY have one non-canonical local source
binding to one working-directory-relative regular file. Binding state
MUST NOT enter a Seal, candidate, canonical REF manifest, logical dump, or
SealID, and its absence MUST NOT affect canonical repository validity.

When `add` has no explicit content source, it MUST use the REF's bound path
when present. It MAY use the exact REF spelling as a path only when both REF
and candidate are absent and that spelling satisfies the portable relative-path grammar. An existing REF or candidate without a binding MUST fail rather
than select a coincidentally named path. Source selection
MUST be deterministic and MUST NOT search, clean, expand globs, walk
directories, inspect Git, or follow symbolic links. A file changed or replaced
during reading MUST fail without a plausible candidate update.

Binding never means automatic import or publication. A contentless refresh
MUST preserve identity-bearing candidate/HEAD state except for fields named by
explicit mutation options. `seal` MUST use only its
validated candidate and MUST NOT reread a working file. Status MUST distinguish
working-file/baseline relations from candidate, draft, and stale facts.

The product MUST expose explicit create, read-one, read-all, compare-and-replace,
and compare-and-remove operations for local bindings. Binding inspection MUST
NOT open source files. There is no implicit retarget, restore-last, binding
reflog, deletion staging, or automatic cross-machine import.

As accepted by ADR 0024, standalone path binding is the default source adapter.
A future Git tree-entry binding exists only through explicit `git sealgraph`
operation and remains non-canonical local input configuration. It resolves a
movable Git spelling to one full commit OID, exact path, blob OID, file mode,
and object format before publication of the binding. It MUST NOT add Git
discovery to standalone commands or put Git identity into Candidate, Seal,
SealID, or canonical REF state.

Every adapter supplies exact bytes to an explicit `add`; `seal` reads only the
validated Candidate. Durable Git or non-Git source-occurrence evidence is
ordinary separately sealed content related by an exact Cause Link. Git commit
parents, file-history/rename inference, timestamps, and watcher events MUST NOT
automatically create a Seal, Cause Link, or Revision Link.

As accepted by ADR 0020, source comparison MAY open exactly the selected bound
file and MUST compare it with candidate content when present, otherwise current
HEAD content. It MUST identify the baseline, use the same stable non-symlink
read boundary as add, and MUST NOT write an object or candidate. Canonical
comparison names are `compare`, `candidate compare`, and `source compare`.
Git-shaped names are diagnostics only and MUST require an explicit retry with
the canonical vocabulary.

## 15. Local operational recovery

Sealgraph MUST distinguish semantic correction from local operational
recovery. A semantic correction creates a new immutable Seal. Recovery MAY
restore the exact prior mutable REF-manifest state of an explicitly selected
locally recorded operation, but MUST NOT modify or reinterpret any Seal, Link,
revision assertion, content, attachment, or Seal ID, and MUST NOT create a
corrective Seal implicitly.

Recovery records are versioned non-canonical local metadata. Their absence,
expiration, removal, or corruption MUST NOT change repository validity,
canonical `fsck`, Seal identity, provenance semantics, or derived active,
stale, frontier, or impact results. Records MUST NOT retain raw argv, content,
cwd, environment values, credentials, or an actor/trusted-time assertion.

Recovery eligibility requires exact current state to equal the complete logged
post-operation REF-manifest state. Any intervening mutation rejects automatic
recovery without partial writes. V1 covers successful `seal`, `tag`, `mv`, and
`ref drop` REF mutations using only an existing atomic shape: one-manifest
restoration or inverse no-replace rename for one move. Candidate edits, arbitrary multi-file
rollback, implicit most-recent selection, reset/reflog/undo semantics, object
deletion, and garbage collection remain absent.

`ref drop` removes exactly one complete current REF manifest from the active
namespace. An exact candidate or local source binding blocks it. It MUST NOT
remove or modify a workfile, source binding, candidate, immutable object, Seal,
Link, or downstream Seal, and it has no recursive, prefix, batch, or force form.

## 16. Format-6 migration boundary

The only format-5-to-format-6 transition is:

```sh
sealgraph migrate repository --from 5 --to 6 [--format human|json]
```

The transaction validates exact format-5 state, snapshots canonical objects,
REF manifests, and Candidate bytes, atomically replaces only `config`, reopens
under format 6, runs complete fsck/readback, and proves retained state unchanged.
It MUST NOT rewrite a historical Seal, Provenance, Candidate, Material, REF, tag,
or content Blob. A committed durability/readback/output uncertainty MUST state
that migration may already be committed and MUST NOT authorize automatic retry.

Format 6 reads historical Seal v5/Provenance v1 pairs and Candidate v5 files
strictly and writes only Seal v6/Provenance v2 and Candidate v6 successors.
Seal v6 MUST pair only with Provenance v2; Seal v5 MUST pair only with
Provenance v1. Material remains v1. Mixed-generation observations are valid
only through these exact pairings and never project metadata into historical
identity.
