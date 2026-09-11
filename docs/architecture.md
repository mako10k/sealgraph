# Architecture

Status: the checked-in runtime implements the accepted format-5 and format-6
typed-Blob, parentless Candidate, Cause-scoped revision, generic Link metadata,
explicit format-5-to-6 migration, isolated format-4 extraction, atomic universal
migration load, inspection, REF manifest, scoped tag, move, and recovery core.
Git views remain separately sequenced.

## 1. Design center

Sealgraph has one native semantic/storage model:

- immutable content and attachment Blobs;
- typed immutable Material, Provenance, and Seal Blobs;
- exact whole-record Cause Links containing observer-scoped revision evidence;
- movable REF paths outside Seal bytes;
- native SHA-256 loose objects under `.sealgraph`.

Standalone beta implements that model without file synchronization or Git
integration. A separately adopted sidecar would add Git-aware read views
of the same `.sealgraph` files; it is not another Seal schema, ObjectID system,
or Git-commit interpretation of the revision DAG.

ADR 0019 adds a bounded local working-file association. This is explicit
working-file input convenience, not synchronization: it never watches files,
creates candidates automatically, or changes the candidate-only sealing
boundary.

```text
                    domain / canonical / revision / graph / history
                                      ^
                                      |
                         native repository reader
                          exact path + file bytes
                     /           |             \
                    /            |              \
       real .sealgraph     staged/commit tree    merge conflict evidence
       reader + writer       read-only views      stages + source trees
             ^                     ^                       ^
             |                     |                       |
      cmd/sealgraph                    cmd/git-sealgraph
       no Git access                  explicit Git adapter
```

Only the real filesystem view is mutable. Equal complete canonical file trees
have equal native graph meaning regardless of whether bytes came from the
filesystem, prospective staged result tree, or immutable Git commit tree.

## 2. Process surfaces

### `cmd/sealgraph`

- opens only `<workdir>/.sealgraph`;
- never searches for or reads `.git`;
- never changes behavior inside a Git worktree;
- performs native candidate mutation and one-REF publication.

### `cmd/git-sealgraph`

- remains an unreleased source placeholder in the standalone beta;
- is invoked explicitly as `git sealgraph ...`;
- may locate the outer Git repository/worktree;
- uses the same real `.sealgraph` writer for native mutations;
- may expose staged, commit-tree, and merge-stage inspection;
- never treats Git commit/merge success as Sealgraph approval.

The entry points are explicit capabilities, not persisted repository modes.
`sealgraph init` always remains standalone. Any future `git sealgraph init` or
setup behavior creates the same native format and must not install hooks or
modify unrelated Git policy silently.

## 3. Package boundaries

### `internal/domain`, `internal/domain/v5`, and `internal/domain/v6`

Pure semantic types:

- native `ObjectID` and portable REF/tag grammar;
- generation-tagged `Attachment`, `CauseLink`, metadata, `Material`,
  `Provenance`, and `Seal` views;
- parentless Candidate semantic/publication state;
- REF names and derived observation facts.

The format-4 payload type remains reachable only by the isolated migration
document verifier; ordinary repository APIs use only versioned format-5 and
format-6 types.

No filesystem, Git, CLI, clock, environment, or current-REF lookup occurs
here. A Seal contains no owner REF.

### `internal/canonical/v5` and `internal/canonical/v6`

- deterministic Material, Provenance, Seal, and Candidate encoding;
- exact member order and JSON escaping;
- Cause/previous/message/attachment sorting and duplicate rejection;
- canonical decode/re-encode byte equality;
- fixed fixture hashes.

The v6 codec adds bounded namespace-sorted canonical JSON metadata to every
Cause Link and enforces exact Seal v6/Provenance v2 pairing. The v5 codec keeps
historical bytes exact and never accepts v6 members. Material v1 remains shared.
Format-6 Assessment-free change identity uses the generation-specific
`upstream-change/v2` canonical record; Assessment references remain outside this
slice.

Canonical encoding does not resolve selectors, inspect REFs, derive stale, or
perform I/O.

### `internal/migration`

- strict `sealgraph/universal-blob-migration/v1` parser/model and canonical
  round-trip;
- migration-only canonical format-4 payload and old-ID verification;
- deterministic Kahn dependency-first ordering and semantic classifications;
- explicit records for object bytes, old Seals, REF/tag targets, materialized,
  unobserved, collapsed, merged, and excluded state.

The pure migration model and projector do not open repositories, mutate
storage, or read Git. A separate `internal/migration/format4extract` package
owns the only format-4 repository reader. It has no mutation entrypoint, never
inspects Git, and is callable only by `sealgraph migrate extract` before
ordinary format-5 repository dispatch. `internal/repository` consumes the
validated document through a separate absent-target transaction. No ordinary
format-5 operation can call the format-4 source reader.

### `internal/store`

Native storage capabilities:

- immutable `ObjectReader` / `ObjectWriter`;
- one-manifest-per-REF `RefStore` with expected-old CAS and atomic move;
- immutable tag bindings stored in the scoped REF manifest;
- exact repository path/file reading needed by native validation.

The real filesystem implementation owns atomic write, lock, no-clobber, fsync,
path-safety, corruption, and hash-mismatch handling. A later read-only tree
view exposes exact path existence, enumeration, and file bytes; it does not
expose Git hash types or mutation.

### Observed revision graph

The repository observation builder, rather than an intrinsic-parent package,
owns format-5 revision meaning:

- fixed-point loading from all heads through Cause targets and asserted
  previous revisions;
- exact observer/Provenance assertion-source retention;
- structural union, active leaves, branching reachability, and detached state;
- combined Cause/revision self-edge and cycle validation; and
- repository-wide and assertion-scope-relative observations.

### `internal/graph`

- Link-only Cause traversal and cycle validation;
- direct/transitive stale using revision leaf facts;
- self-stale current heads;
- reverse impact;
- exact-Cause stale review frontier;
- deterministic first-match and bounded all-path evidence with revision proof.

No stale, impact, frontier, or path result is canonical persisted state.

### History and comparison

- branching traversal over observed Cause-scoped revision assertions;
- exact Material/Provenance Seal-to-Seal comparison with two explicit inputs;
- exact whole Cause-record before/after changes on every structural edge;
- Candidate comparison against its explicit publication baseline; and
- separate reporting of current and `expected_ref_head` relation.

It does not implement Git reflog/history semantics or infer ownership from a
selector spelling.

### `internal/repository`

Coordinates:

- candidate lifecycle;
- content object writes and preservation of existing attachment objects;
- exact selector resolution;
- format-5 whole-record and format-6 metadata-preserving legacy Cause authoring
  with one coherent selector observation;
- single-namespace Link metadata set/remove with expected-old Candidate
  replacement and legacy authoring preservation;
- normal Cause-closure admission;
- canonical Material, Provenance, and Seal creation;
- one-REF CAS publication;
- scoped immutable tag creation and single-manifest REF move;
- coherent multi-REF observations;
- canonical-scan graph orchestration with optional disposable cache semantics;
- local non-canonical recovery-journal orchestration and operation-specific
  exact-state restoration;
- absent-target universal-blob import, typed projection, semantic-loss receipt,
  fsck/digest readback, and atomic namespace publication;
- exact config-only format-5-to-6 migration with retained-state readback.

It never probes Git. A Git entry point passes the real worktree root explicitly
when native mutation is requested.

### `internal/recovery`

- strict versioned PREPARED/COMMITTED operation records;
- exact present/absent REF-manifest before/after transitions;
- safe local journal inventory and crash-state classification;
- no canonical repository, provenance, graph, or intent semantics.

The repository package owns eligibility and mutation sequencing. The recovery
package does not write canonical REFs directly or select an operation for the
operator.

### `internal/cli`

Parsing and presentation only:

- REF and Seal selector grammar;
- binary-safe preview and raw bytes-only output;
- deterministic status/stale/impact rendering;
- stable narrow line protocols;
- explicit error and next-action text.
- registry-backed completion candidates and Git-shaped misuse navigation.
- terminal-width-aware aligned human rendering and destination-based selection
  of existing versioned machine output.

### `internal/pathmanifest`

- validates explicit portable relative semantic paths;
- reads only caller-named regular files without following symlink components;
- sorts entries and emits `sealgraph/path-manifest/v1` canonical bytes;
- computes exact file SHA-256 and the canonical-entry aggregate.

It does not open `.sealgraph`, inspect Git, expand globs, walk directories,
write objects/candidates, or approve/seal its output.

### Local source adapter

- stores one versioned non-canonical REF-to-relative-path association below
  the runtime index;
- safely reads and revalidates exact regular-file bytes without following
  symbolic links;
- reports working-file state relative to candidate content or current HEAD;
- supplies bytes to an explicit `add`, never directly to `seal`.

It does not change canonical REF manifests, candidate/Seal schemas, discover
Git, watch directories, expand globs, or perform automatic add/seal. The
repository package coordinates binding changes with candidate mutation under
the native writer guard.

ADR 0024 generalizes source selection as a non-canonical adapter boundary.
`WorktreePath` remains the standalone/default binding. A future
`GitTreeEntry` binding is available only to the explicit Git entry point and
records exact object format, commit, path, blob, and file-mode identity. Both
adapters materialize exact bytes through `add`; neither is visible to `seal`,
which remains Candidate-only.

Portable source-occurrence provenance is separately sealed application content
and may be named by an exact Cause Link. Local binding fields, Git commit
ancestry, file-history heuristics, and working-file timestamps do not enter
Seal identity or create Revision facts automatically.

The Bash completion wrapper delegates parsing and candidate selection to a
hidden read-only CLI protocol. Repository-aware completion reads only REF,
candidate, and binding metadata; it does not bootstrap, open bound workfiles,
inspect Git, or update cache state.

### Later Git view adapter

The Git adapter has no Sealgraph domain semantics. It supplies:

- complete exact byte/path views of the worktree, prospective staged result
  tree, and immutable commit tree;
- exact regular-blob bytes for an explicitly configured non-canonical
  `GitTreeEntry` source binding;
- merge stage 1/2/3 conflict entries associated with corresponding validated
  BASE/OURS/THEIRS complete trees;
- typed physical Git identity internal to the adapter;
- coherent index capture/revalidation.

It delegates config/object/REF decoding and all revision/Cause reasoning to the
native reader and shared domain packages.

## 4. Native object store

Formats 5 and 6 retain:

- immutable loose objects;
- Git-compatible SHA-256 blob envelope and path where practical;
- full 64-character lower-case native IDs;
- user-input unique prefixes only;
- one loose mutable manifest per REF, containing HEAD and immutable tag
  bindings;
- no canonical packs or packed refs.

Low-level Git compatibility is forensic/storage compatibility only. An
explicitly configured Git SHA-256 API may read native loose blobs, but
`.sealgraph` is not a Git repository or an alternate for an outer SHA-1
repository and must not receive Git maintenance or porcelain operations.

## 5. Publication transaction

One Seal publication:

1. acquires the repository-wide native writer guard;
2. loads one exact candidate version and, for Candidate v5 in a format-6
   repository, constructs its exact Candidate-v6 projection in memory without
   rewriting the Candidate file;
3. validates `expected_ref_head`, Material, Provenance, complete Cause
   admissibility, and the prospective combined graph;
4. canonicalizes and writes Material, Provenance, and Seal Blobs;
5. revalidates required state;
6. CAS-updates exactly one destination REF;
7. clears only the unchanged candidate version;
8. releases the writer guard.

Successful expected-old REF CAS is the publication linearization point.
Objects left before failed CAS remain immutable dangling objects and are
reported, not deleted or activated.

## 6. Coherent observations and cache

Multi-REF facts capture exact REF-manifest bytes and the complete current
REF/head set, build the fixed-point Cause/revision observation, buffer output,
then revalidate the exact manifests before emission. Graph-dependent Candidate
mutations apply the same capture/build/prospective-validate/revalidate pattern.
A change fails with no plausible output or Candidate replacement.

The active revision DAG is rooted by current REF heads only. Object existence,
tag reachability, or Cause reachability does not publish a revision.

Any revision/Cause cache is derived and disposable. It may answer only when
bound to the complete observation and exactly equivalent to canonical scan.
Missing, mismatched, or invalid cache state never repairs, overrides, or
weakens canonical validation, and read-only Git views do not persist cache.

## 7. Git sidecar boundary

This is a constraint on any separately approved future sidecar, not a beta
implementation commitment. Its first potential value is `.sealgraph` file integration:

- prospective staged-tree validation;
- historical read-only validation/inspection;
- merge conflict evidence;
- explicit validation-only hook dispatch.

A staged validator builds the prospective commit tree from the base plus
stage-zero index and validates unchanged as well as changed canonical paths.
Nonzero merge stages are a separate conflict state. Concurrent index change,
missing partial-clone object, unsupported Git/native format, or canonical-byte
filter transformation fails explicitly without native mutation, implicit
network fetch, dual reader, or automatic migration.

The selected Git SDK must prove the released binary's supported SHA-1/SHA-256,
worktree, linked-worktree, index, tree, pack, and alternate matrix. No SDK type
crosses into native domain APIs; there is no hand-written pack reader or silent
Git CLI fallback.

Outside the exact `GitTreeEntry` source-binding path accepted by ADR 0024,
importing arbitrary Git blobs/trees/commits/tags as generated material remains
deferred. Zero-copy external references or type-specific projections require a
separate persisted contract.

## 8. Extension discipline

Do not prebuild remote storage, signatures, daemon/server, MCP, arbitrary link
kinds, automatic branch choice, automatic relink/reseal, recursive repair, or
batch publication. New persisted fields require storage-format changes,
deterministic fixtures, compatibility consideration, and an approved ADR.
