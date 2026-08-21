# Architecture

Status: the checked-in runtime implements the format-4 canonical/candidate,
selector, atomic load, active revision/Cause graph, history, and impact core.
REF manifests, scoped tags, atomic move, and tracked format-4 dogfood
conversion are also implemented. Explicit path manifests are implemented;
operator-contract usability and Git views remain separately sequenced.

## 1. Design center

Sealgraph has one native semantic/storage model:

- content-only immutable Seals;
- exact Seal-to-Seal Cause Links;
- one optional exact `parent_revision` edge;
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

### `internal/domain`

Pure semantic types:

- native `ObjectID` and `ContentRef`;
- exact-target `Link` and `Attachment`;
- content-only `SealPayload` with `parent_revision`;
- candidate topology/publication state;
- REF names and derived observation facts.

No filesystem, Git, CLI, clock, environment, or current-REF lookup occurs
here. A Seal contains no owner REF.

### `internal/canonical`

- deterministic format-4 Seal encoding;
- exact member order and JSON escaping;
- Link/attachment sort and duplicate rejection;
- canonical decode/re-encode byte equality;
- fixed fixture hashes.

Canonical encoding does not resolve selectors, inspect REFs, derive stale, or
perform I/O.

### `internal/migration`

- strict versioned logical dump parser/model and canonical round-trip;
- deterministic dependency-first conversion ordering;
- explicit records for material, old Seal identity, REF/tag targets, and
  excluded loose objects.

It does not open repositories, mutate storage, read Git, or act as a legacy
repository reader. `internal/repository` consumes the validated document
through a separate absent-target transaction; legacy format-3 payload types
exist only inside this migration-document boundary.

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

### `internal/revision`

- active revision DAG from coherent current REF heads and parent ancestry;
- parent cycle validation;
- ancestor/descendant queries;
- active leaves/tips and detached state;
- explicit parent/fork admissibility.

Parent edges mean revision derivation only. They are not Cause edges and do not
express replacement or preference.

### `internal/graph`

- Link-only Cause traversal and cycle validation;
- direct/transitive stale using revision leaf facts;
- self-stale current heads;
- reverse impact;
- exact-Cause stale review frontier;
- deterministic shortest and bounded all-path evidence.

No stale, impact, frontier, or path result is canonical persisted state.

### `internal/history`

- `parent_revision` traversal independent of REF names;
- Seal-to-Seal semantic diff;
- exact-target Link add/remove/repoint/message events;
- candidate comparison against `parent_revision`;
- separate reporting of `expected_ref_head` relation.

It does not implement Git reflog/history semantics or infer ownership from a
selector spelling.

### `internal/repository`

Coordinates:

- candidate lifecycle;
- content/attachment object writes;
- exact selector resolution;
- explicit parent selection and derivation;
- normal Cause-closure admission;
- canonical Seal creation;
- one-REF CAS publication;
- scoped immutable tag creation and single-manifest REF move;
- coherent multi-REF observations;
- disposable revision/Cause cache orchestration;
- local non-canonical recovery-journal orchestration and operation-specific
  exact-state restoration;
- read-only format-3 logical dump capture, graph closure, and final complete
  observation revalidation during the explicit migration slice.

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

The Bash completion wrapper delegates parsing and candidate selection to a
hidden read-only CLI protocol. Repository-aware completion reads only REF,
candidate, and binding metadata; it does not bootstrap, open bound workfiles,
inspect Git, or update cache state.

### Later Git view adapter

The Git adapter has no Sealgraph domain semantics. It supplies:

- complete exact byte/path views of the worktree, prospective staged result
  tree, and immutable commit tree;
- merge stage 1/2/3 conflict entries associated with corresponding validated
  BASE/OURS/THEIRS complete trees;
- typed physical Git identity internal to the adapter;
- coherent index capture/revalidation.

It delegates config/object/REF decoding and all revision/Cause reasoning to the
native reader and shared domain packages.

## 4. Native object store

Format 4 retains:

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
2. loads one exact candidate version;
3. validates `parent_revision`, `expected_ref_head`, material, and complete
   Cause admissibility;
4. canonicalizes and writes one immutable Seal object;
5. revalidates required state;
6. CAS-updates exactly one destination REF;
7. clears only the unchanged candidate version;
8. releases the writer guard.

Successful expected-old REF CAS is the publication linearization point.
Objects left before failed CAS remain immutable dangling objects and are
reported, not deleted or activated.

## 6. Coherent observations and cache

Multi-REF facts capture the complete current REF/head set, load and validate
the required parent/Cause graph, buffer output, then revalidate the complete
head set before emission. A change fails with no plausible partial stdout.

The active revision DAG is rooted by current REF heads only. Object existence,
tag reachability, or Cause reachability does not publish a revision.

The revision/Cause cache is derived and disposable. Its key binds repository
and schema version plus a digest of the complete sorted REF/head observation.
Missing, snapshot-mismatched, or invalid cache triggers canonical scan and
atomic refresh. Missing and snapshot-mismatched cache are normal misses and do
not warn. Invalid/unsafe cache or refresh failure may warn, but cache failure
never repairs or overrides canonical state, and read-only Git views do not
persist cache.

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

Importing arbitrary Git blobs/trees/commits/tags as generated material is
deferred. Exact blob materialization can be added later without changing Seal
format; zero-copy external references or type-specific projections require a
separate persisted contract.

## 8. Extension discipline

Do not prebuild remote storage, signatures, daemon/server, MCP, arbitrary link
kinds, automatic branch choice, automatic relink/reseal, recursive repair, or
batch publication. New persisted fields require storage-format changes,
deterministic fixtures, compatibility consideration, and an approved ADR.
