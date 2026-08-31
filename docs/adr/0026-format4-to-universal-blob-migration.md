# ADR 0026: Incompatible format-4 to Universal Blob migration

- Status: Accepted
- Date: 2026-08-28
- Decision Owner: Operator
- Accepted: 2026-08-31 by explicit Operator acceptance of the exact reviewed
  candidate
- Acceptance Record:
  [`cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md#owner-acceptance-receipt)
- Related Claims: C-MG-001 through C-MG-010
- Related Evidence: E-MG-001 through E-MG-011
- Pending Decision: None
- Execution Authority: Separate; acceptance does not authorize implementation,
  normative-document conversion, repository migration, branch synchronization,
  release, or deployment
- Supersedes: format-4 runtime compatibility as a requirement for
  the format-5 runtime; any legacy-parent fallback proposed by an earlier ADR
  0023 candidate
- Superseded by: None

## Context

ADRs 0023 and 0025 replace the format-4 monolithic Seal and global
`parent_revision` with Universal Blob instances and Cause-Link-scoped revision
assertions. This is not a byte-compatible schema extension. Every format-4 Seal
receives a new identity, and a global parent assertion has no equivalent unless
one or more Cause Links observe the child and can carry that assertion.

Keeping a dual reader or defaulting an omitted Link assertion from the target's
old parent would preserve two authorities indefinitely. The operator instead
selected a migration model and explicitly accepted that an old parent not
observed by any Cause Link loses graph meaning, provided the migration warns
and records the loss.

## Decision

### No runtime compatibility layer

Ordinary format-5 repository operations do not read or write format-4 Seals or
Candidates and never open format-4 repository state. Format-5 typed decoders do
not accept a format-4 schema. The runtime has no dual repository reader, mixed
graph, lazy upgrade, legacy-parent fallback, or in-place rewrite.

Opening a format-4 `.sealgraph` with the format-5 runtime fails before mutation
with a stable `FORMAT4_REQUIRES_MIGRATION` error and names the explicit export
and load boundary. It must not partially validate format 4 and continue.

The final format-4 release exposes the read-only exporter:

```sh
sealgraph dump --format universal-blob-v1
```

The format-5 runtime exposes the importer:

```sh
sealgraph load --format universal-blob-v1 < repository.dump.json
```

The importer parses only the migration document. Inside that isolated command,
a migration-only format-4 payload verifier decodes each exported
`payload_base64`, validates it against the exact format-4 canonical Seal rules,
re-encodes it byte-for-byte, and verifies its old SealID. That verifier has no
repository/store interface and is not callable by normal format-5 operations.
The importer never opens the source format-4 repository or treats an embedded
old payload as a live format-5 Seal.

This establishes C-MG-001: compatibility is an explicit one-way migration,
not a permanent runtime semantic branch.

### Export admission and observation

The exporter is read-only and accepts no positional repository, repair,
ignore, compatibility, or Git option. It operates on the explicitly opened
standalone format-4 repository and never inspects an outer Git repository.

Before producing output it validates:

- the repository format marker and every physical loose object envelope;
- every REF manifest, head, scoped tag, and exact target Seal;
- the complete Seal closure rooted at all REF heads and tag targets through
  both global parent and Cause edges;
- every referenced content and attachment Blob;
- canonical Seal bytes, graph acyclicity over the union of parent and Cause
  dependencies, and one complete repository observation value confirmed by two
  equal captures; and
- complete absence of candidate state, including corrupt or unrecognized
  candidate entries.

The exporter implements that equal-capture observation as an explicit
double-capture transaction. The first capture `S_0` consists of:

```text
config_bytes
  exact source config bytes

ref_manifest_map
  every logical REF -> exact regular manifest bytes
  including its HEAD and complete tag array

loose_object_map
  every relative loose-object path ->
    directory: (DIRECTORY)
    regular:   (REGULAR, byte_length, sha256(exact physical stored bytes))

candidate_namespace_map
  every reserved Candidate path or unrecognized/unsafe Candidate-namespace
  entry ->
    directory: (DIRECTORY)
    regular:   (REGULAR, byte_length, sha256(exact regular-file bytes))
    unsafe:    (SYMLINK or SPECIAL), without opening the entry
```

Directory and regular-file entry kinds are distinguished; symbolic links,
special files, malformed object paths, and unrecognized canonical layout are
validation errors, never followed. Candidate admission requires
`candidate_namespace_map` to be empty. Source-binding `.track` entries are not
Candidate entries and remain in the explicit excluded-state category.

From `S_0`, the exporter validates every physical object envelope, derives the
complete rooted graph and semantic projection, inventories excluded objects,
and buffers the entire migration document and warning set. Immediately before
emitting either warnings or stdout it captures `S_1` using the same path, entry
kind, length, and byte-digest rules. It requires exact equality of all four
members, including additions and removals. A mismatch fails with retryable
`MIGRATION_SOURCE_CHANGED`, emits no document or semantic warning, and does not
repair or lock in either observation. Equal maps establish the recorded source
observation value to which the buffered result is bound. They do not detect a
change-and-restore between captures and therefore do not prove that every
intermediate physical state was identical.

Any candidate blocks export because mutable intent cannot be projected safely.
The operator must explicitly seal or discard it using the format-4 runtime.
Corruption, a dependency cycle, or concurrent change fails nonzero without a
plausible partial document and without repair.

This establishes C-MG-002: migration output is bound to one complete,
coherently observed format-4 logical value that is equal at `S_0` and `S_1`;
continuous source immutability between those captures is not claimed.

### Deterministic migration document

Successful stdout is exactly one compact canonical JSON document followed by
LF with schema `sealgraph/universal-blob-migration/v1`. Unknown members,
noncanonical ordering/escaping, duplicate set members, invalid base64, and a
missing or extra final byte are errors.

JSON string validity, escaping, and byte-equality re-encoding use ADR 0025's
canonical UTF-8 rules. Unicode normalization is never applied. Numbers occur
only in the fixed format fields described below. Their lexical form is part of
the canonical bytes: format 4 is the single ASCII byte `4` and format 5 is the
single ASCII byte `5`. A sign, leading zero, decimal point, fraction, exponent,
or any other JSON number spelling is non-canonical and rejected even when a
generic JSON decoder would assign the same mathematical value.

Its required top-level member order is:

```text
schema, source_repository, objects, seals, refs, tags,
semantic_projection, excluded_objects, excluded_state
```

The exact nested shapes and member orders are:

```text
source_repository: format, object_format
object:            id, bytes_base64
seal:              id, payload_base64
ref:               name, head
tag:               ref, name, target
semantic_projection:
  materialized_parent_assertions,
  unobserved_parent_assertions,
  collapsed_revision_assertions,
  merged_cause_links
materialized:      observer, child, parent
unobserved:        child, parent
collapsed:         observer, child, parent
merged_cause_link: observer, old_targets, new_target
```

JSON types are fixed:

- `schema` is exactly `sealgraph/universal-blob-migration/v1`;
- `source_repository.format` is the integer `4` encoded by exactly the single
  ASCII byte `4`;
- `source_repository.object_format` is exactly the string `sha256`; every
  other value is rejected before object or Seal projection;
- every ID/digest is one 64-character lower-hex string;
- `bytes_base64` and `payload_base64` use canonical RFC 4648 standard padded
  Base64 with no whitespace or line breaks. Unused pad bits must be zero. A
  reader performs strict pad-bit validation and requires standard padded
  Base64 re-encoding of the decoded bytes to equal the original string byte for
  byte; alternate strings for the same decoded bytes are rejected. For example,
  byte `0x66` is `Zg==`; `Zh==` is rejected for nonzero unused pad bits;
- `objects`, `seals`, `refs`, `tags`, every semantic member,
  `excluded_objects`, and `excluded_state` are arrays; and
- `old_targets` is a sorted duplicate-free array containing at least two old
  SealIDs.

The document includes exact bytes for every content or attachment Blob
referenced by exported Seals, every exported old SealID and canonical format-4
payload, every REF, and every scoped tag. `excluded_objects` contains every
valid loose BlobID not otherwise exported.

`excluded_state` is always exactly this constant ordered array, independent of
which local files happen to exist:

```json
["source_bindings","cache","event_logs","recovery_journal","locks","temporary_files"]
```

It declares excluded categories, not paths, counts, contents, or evidence that
any category is present. `event_logs` excludes non-recovery operational logs;
`recovery_journal` explicitly excludes ADR 0018 records even though both live
below the source `logs/` directory. Candidate state is not an exclusion: any
candidate rejects export.

Array order is normative:

```text
objects:        id
seals:          exact ready-set algorithm below
refs:           name
tags:           (ref, name)
materialized:   (observer, child, parent)
unobserved:     (child, parent)
collapsed:      (observer, child, parent)
merged links:   (observer, new_target, old_targets)
excluded IDs:   BlobID
```

All tuple comparisons are bytewise ascending UTF-8. Seal ordering uses Kahn's
algorithm over the exported format-4 dependency graph whose edges point from a
Seal to each global parent and Cause target:

1. `ready` contains every Seal with no not-yet-emitted dependency;
2. remove and emit the lexicographically smallest full old SealID in `ready`;
3. remove that dependency from its direct dependents and insert each newly
   ready dependent; and
4. repeat until every Seal is emitted; a non-empty remainder is a cycle and
   rejects the document.

Thus dependencies always precede dependents and the ready-set full SealID is
the only tie break. No topological layer, map iteration, filesystem order, or
host sort stability affects the result.

The semantic projection is computed by the final format-4 exporter using the
projection below and is recomputed byte-for-byte by the importer. Equal
complete migration observations—including canonical state and the excluded
loose-object inventory—produce equal bytes.

The document contains no source path, timestamp, actor, hostname, tool version,
random value, local binding, cache entry, lock, recovery journal, event log,
temporary filename, or outer Git state. Every REF-scoped tag name is validated
against the accepted format-4 TAGNAME grammar; in particular
`[0-9a-f]{4,64}` remains reserved and is rejected rather than exported as an
unaddressable format-5 tag.

This establishes C-MG-003: the conversion input is a portable, deterministic,
versioned artifact whose omissions and semantic changes are explicit.

### Exact format-4 projection

For every exported format-4 Seal `S`, the importer deterministically creates:

1. one Material Blob containing `S.content`'s BlobID and the same sorted
   attachment names, media types, and BlobIDs;
2. one Provenance Blob containing `S.root`, `S.draft`, and converted Cause
   Links; and
3. one Seal Blob naming those exact Material and Provenance IDs.

The accepted format-4 structural invariant is part of export validation: a
root Seal has no Cause Links and every non-root Seal, including draft, has at
least one Cause Link. A source payload violating it is corrupt and export
fails. Projection therefore cannot create a format-5 non-root Provenance with
an empty Cause Link array. Attachment names are likewise required to be
non-empty, valid UTF-8, and unique before they are copied.

Existing content and attachment Blob bytes are stored unchanged, so their
BlobIDs remain unchanged. Format-4 Seal payload Blobs are not treated as
format-5 Seals and are not copied into the target object store merely because
they appear in the migration document. The target object store contains
exactly the exported content/attachment Blobs plus the unique new Material,
Provenance, and Seal Blobs. New typed IDs are computed solely from their ADR
0025 canonical bytes. Load writes the exact format-5 config bytes defined by
ADR 0025.

For each old Cause Link from observer `D` to target `T`, first form one
provisional record:

```text
target_seal = map(T)
messages = [old Link message]

if T.parent_revision is null:
  previous_revision_seal_of_target_seal = []
else if map(T.parent_revision) = map(T):
  previous_revision_seal_of_target_seal = []
  record COLLAPSED_REVISION_DROPPED(D, T, T.parent_revision)
else:
  previous_revision_seal_of_target_seal = [map(T.parent_revision)]
```

Within each old observer `D`, provisional records are grouped by mapped
`target_seal`. One canonical format-5 Cause Link is emitted per group. Its
previous-revision array is the sorted set union of the provisional arrays and
its `messages` array is the sorted set union of every exact old message,
including an empty string. When a group contains more than one old target, the
group is recorded as `MERGED_CAUSE_LINKS`; no message is discarded and no
duplicate format-5 target is emitted.

The conversion uses one dependency-first pass over the document's topological
Seal order, so every referenced target/parent mapping exists before its
observer is encoded. The importer then validates the complete projected
Cause/revision graph. A projected self-revision edge is impossible after the
explicit collapsed-revision removal above. A remaining Cause self-edge or any
remaining Cause/revision cycle fails before output or target creation with
`FORMAT4_COLLAPSE_GRAPH_CYCLE_UNSUPPORTED`; the migration does not choose
another semantic edge to delete.

Every REF head and tag target is rewritten through the complete old-to-new
SealID map. Multiple old IDs mapping to one new SealID are permitted only if
the canonical conversion proves exact equality and the collapse is reported.

This establishes C-MG-004: material and every distinct Link message are
preserved exactly, collapsed target Links are normalized without duplicate
targets, and revision meaning whose endpoints become one Seal is explicitly
removed rather than converted into corruption.

### Unobserved old parent meaning

For every exported old edge:

```text
child.parent_revision = parent
```

the migration classifies it per observing Link as:

```text
MATERIALIZED
  an exported old Cause Link from observer targets child and
  map(child) != map(parent); map(parent) is copied into that observer's
  converted Link assertion

UNOBSERVED_PARENT_DROPPED
  no exported old Cause Link targets child; no format-5 graph assertion is
  created for the old parent edge

COLLAPSED_REVISION_DROPPED
  an exported old Cause Link from observer targets child but
  map(child) = map(parent); the revision assertion is removed rather than
  creating a self-edge
```

`UNOBSERVED_PARENT_DROPPED` is an intentional semantic change, not corruption
and not an importer failure. The old child and parent are still converted and
listed in the mapping, but their former global edge has no format-5 meaning.
The importer must not invent a Cause Link, observer Seal, synthetic REF, tag,
or fallback field to retain it.

`COLLAPSED_REVISION_DROPPED` is also an intentional semantic change. The old
observer, child, and parent remain in the dump and mapping receipt, and the
complete collapse group remains auditable, but the format-5 Link contains no
self-previous assertion. This is the operator-selected policy for revision
meaning lost through identity collapse.

When a count is nonzero, both exporter and importer emit the corresponding
human-visible stderr warning:

```text
SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=<N>
SEMANTIC_CHANGE_COLLAPSED_REVISION_DROPPED count=<N>
SEMANTIC_CHANGE_MERGED_CAUSE_LINKS count=<N>
```

Each count is exactly the length of the corresponding canonical semantic array;
`MATERIALIZED` has no warning. `MERGED_CAUSE_LINKS` records normalization, not
message loss. Warnings do not change a successfully delivered dump/load exit
status. The deterministic document and load receipt contain the complete sorted
records, not merely the counts. A caller that requires semantic preservation
can treat any non-empty loss array as a policy failure outside Sealgraph;
Sealgraph itself does not silently relabel the migration as lossless.

This establishes:

- C-MG-005: old parent meaning survives only where an actual Cause Link can
  own a non-self new scoped assertion; and
- C-MG-006: every intentional semantic loss is machine-readable and visibly
  warned without fabricating provenance.

### Atomic import

The target `.sealgraph` must be absent. Load validates the complete document,
recomputes its SHA-256 digest, verifies every old object and Seal identity,
repeats the semantic classification, and computes the full old-to-new mapping
before creating target state.

Load constructs a complete format-5 repository in a sibling staging directory,
computes the canonical receipt described below, and writes that exact receipt
at `logs/migration/<source-document-sha256>.receipt` inside staging. Its final
published path is:

```text
.sealgraph/logs/migration/<source-document-sha256>.receipt
```

The receipt is non-canonical local audit state and is excluded from the
repository digest, but it is carried by the same directory publication. The
load durability sequence is exact:

1. create only regular files and directories inside sibling staging, without
   following symbolic links;
2. write each final file completely, set its final mode, verify kind and mode
   through the same open handle, fsync it, and close it;
3. validate the staged config, complete physical object inventory, every typed
   reference, REF/tag manifest, semantic projection, and repository digest with
   the format-5 `fsck` boundary, then separately revalidate the non-canonical
   receipt bytes against those results;
4. set and verify each final directory mode through an open directory handle,
   then fsync every staging directory bottom-up after all child entries exist,
   including the staging root;
5. atomically rename staging to the absent `.sealgraph` target with one
   same-filesystem no-replace operation; and
6. fsync the target's parent directory before claiming durable publication or
   beginning post-publication readback.

Final modes are `0755` for directories, `0644` for config, `0444` for immutable
loose objects, and `0600` for REF manifests and the migration receipt. The
implementation sets and verifies these loader-transaction postconditions
explicitly rather than relying on umask. They are operational hardening, not
canonical identity or stable `fsck` validity; a later outer-Git checkout may
restore writable ordinary-file modes without changing repository bytes. No
Candidate, binding, cache entry, lock entry, or pre-existing recovery/event
record is staged. Empty runtime directories use mode `0755`.

Steps 2 and 4 make every nested file and directory entry durable before the
namespace commit; step 6 makes the destination name durable after it. Load
never merges with or replaces an existing repository. A platform that cannot
prove regular-file fsync, directory fsync, same-filesystem atomic no-replace
rename, and destination-parent fsync rejects load before publication. A
pre-publication staging directory may remain for explicit inspection.

Failure states are explicit:

```text
PRE_PUBLICATION_FAILURE
  this load did not perform the target rename; it did not change a target, but
  a pre-existing or concurrently created target may now exist; retry only after
  correcting the cause and confirming the exact target is absent

LOAD_PUBLISHED_DURABILITY_UNCERTAIN
  the no-replace rename succeeded, but target-parent synchronization failed or
  completion across that boundary is unknown; target may be visible; do not
  retry, delete, or report durable publication automatically

LOAD_PUBLISHED_READBACK_FAILED
  target-parent synchronization succeeded and target exists, but complete
  post-publication verification did not match;
  do not retry load or delete/repair automatically

LOAD_PUBLISHED_RECEIPT_UNDELIVERED
  target and durable receipt are valid, but stdout delivery failed;
  do not retry load; recover the receipt with load-receipt
```

An interruption before atomic publication creates no target from this load.
An interruption at or after rename may leave the complete target and receipt
but cannot claim parent-directory durability without step 6. The operator inspects the
destination and platform durability state rather than retrying load. If the
target survives and validates, `load-receipt` may recover its exact receipt
bytes, but that read-only command does not retroactively turn a durability-
uncertain publication into a successful `load`. A surviving pre-publication
staging directory is reported for explicit inspection and is never adopted or
deleted automatically.

This establishes C-MG-007: canonical repository namespace publication is
atomic, nested and parent-directory durability have an explicit ordered
contract, and every post-rename failure is reported without falsely claiming
that the target is absent or safe to republish. Loader-created modes are
verified within that transaction but do not become canonical integrity state.

### Canonical load receipt

The receipt schema is
`sealgraph/universal-blob-load-receipt/v1`. Encoding uses compact canonical
UTF-8 JSON plus one final LF. Unknown members, unknown enum strings,
noncanonical member/array order, duplicate set members, malformed IDs, and
missing or extra final bytes are errors.

It uses the same canonical string escaping and byte-equality rules as the
migration document. Numbers occur only as the fixed `published_format` value,
encoded by exactly the single ASCII byte `5`; alternate numeric spellings are
rejected.

The required top-level member order is:

```text
schema, source_document_sha256, seal_mappings, collapse_groups,
typed_objects, unchanged_objects, refs, tags, semantic_projection,
excluded_objects, excluded_state, published_format, repository_digest
```

Every nested object has this exact member order:

```text
seal_mapping:       old_seal, new_seal, material, provenance
collapse_group:     new_seal, old_seals
typed_object:       kind, id
ref_mapping:        name, old_head, new_head
tag_mapping:        ref, name, old_target, new_target
semantic_projection:
  materialized_parent_assertions,
  unobserved_parent_assertions,
  collapsed_revision_assertions,
  merged_cause_links
materialized:       old_observer, old_child, old_parent,
                    new_observer, new_target, new_previous
unobserved:         old_child, old_parent, new_child, new_parent
collapsed:          old_observer, old_child, old_parent,
                    new_observer, new_collapsed_seal
merged_cause_link:  old_observer, old_targets, new_observer, new_target
```

JSON types and value domains are fixed:

- `source_document_sha256` and `repository_digest` are 64-character lower-hex
  raw SHA-256 digests without a prefix;
- all other IDs are 64-character lower-hex BlobID/typed-ID strings;
- `kind` is exactly `material`, `provenance`, or `seal`;
- `published_format` is the integer `5` encoded by exactly the single ASCII
  byte `5`;
- `old_seals` and `old_targets` are sorted duplicate-free ID arrays with at
  least two members;
- `excluded_state` is exactly the constant array from the migration document;
  and
- every other plural member is an array that may be empty. Counts are not
  duplicated in the receipt; the canonical array length is authoritative.

Normative array order is:

```text
seal_mappings:      old_seal
collapse_groups:    new_seal
typed_objects:      (kind, id)
unchanged_objects:  BlobID
refs:               name
tags:               (ref, name)
materialized:       (old_observer, old_child, old_parent)
unobserved:         (old_child, old_parent)
collapsed:          (old_observer, old_child, old_parent)
merged links:       (old_observer, new_target, old_targets)
excluded_objects:   BlobID
```

`seal_mappings` contains every old-to-new mapping. A `collapse_group` exists
exactly when at least two old SealIDs map to one new SealID. `typed_objects` is
the duplicate-free role inventory of all format-5 Material, Provenance, and
Seal Blobs. `unchanged_objects` is the sorted duplicate-free inventory of
unchanged content and attachment BlobIDs. The two role inventories may contain
the same BlobID when exact bytes validly serve both roles; physical storage
still contains one Blob. REF and tag mappings contain every old and new target.
Semantic records contain every materialized, unobserved, collapsed, and merged
case. Excluded arrays echo the migration document exactly.

`source_document_sha256` hashes the complete migration document bytes,
including final LF. `repository_digest` is lower-hex SHA-256 of the compact
canonical bytes, without final LF, of:

```text
repository_observation:
  schema, format, object_format, config_sha256, objects, refs, tags
repository_ref: name, head
repository_tag: ref, name, target
```

The schema is `sealgraph/repository-observation/v1`; `format` is integer `5`
encoded by exactly the single ASCII byte `5`, `object_format` is `sha256`, and
`config_sha256` is raw SHA-256 of exact config bytes. `objects` is the sorted
complete physical BlobID inventory. `refs` uses
`name, head` records sorted by name; `tags` uses `ref, name, target` records
sorted by `(ref, name)`. Readers validate every loose envelope and
typed/reference closure before accepting the inventory. This digest therefore
covers the published config, object store, and logical manifest contents. It
deliberately excludes permission bits and physical entry identity: those are
not portable canonical state. Load verifies its own creation-mode postconditions
before publication and readback, while a later stable writable checkout can
retain the same repository digest.

### Receipt delivery and recovery command

After atomic rename and successful target-parent synchronization, `load`
independently reopens the target, validates the complete repository
observation, requires its digest to equal the staged durable receipt, and
separately verifies that entries created by this load still have their final
modes. That mode check is a postcondition of this original load invocation,
not a repository-digest member or a later `load-receipt` prerequisite. Only then
does `load` copy the already stored receipt bytes to stdout. Successful exit
zero means namespace publication, nested and parent durability, original-load
mode postconditions, readback, and complete receipt delivery all succeeded.

Receipt stdout failure does not roll back or relabel the already published
repository. It returns `LOAD_PUBLISHED_RECEIPT_UNDELIVERED` on stderr and a
nonzero exit. The exact receipt is recovered idempotently with:

```sh
sealgraph load-receipt --source-document-sha256 HEX
```

`load-receipt` is read-only. It requires one matching regular receipt file,
revalidates its canonical bytes, verifies the complete current repository
observation against `repository_digest`, and emits those exact receipt bytes
plus their existing LF. It may be retried after output-sink failure and never
creates, repairs, or republishes repository state. A missing, mismatched, or
stale receipt fails without partial stdout.

Receipt generation is evidence, not canonical provenance, approval, or a Seal.
An operator may later seal the receipt only through an ordinary explicit
Candidate workflow.

This establishes C-MG-008: identity rewriting and semantic change have one
fully canonical, publication-bound, recoverable receipt whose validation covers
the complete published object and manifest observation.

### Source retention and rollback boundary

Migration does not delete, rename, edit, or mark the format-4 source. Rollback
means selecting the separately retained source repository with a format-4
runtime; it is not an in-place downgrade of the format-5 repository.

The exporter opens source config, manifests, objects, and Candidate-namespace
entries read-only and has no source mutation capability. Its `S_0`/`S_1`
equality check proves that both complete admitted source captures have the same
recorded value and binds output to that value; it does not prove continuous
absence of source mutation during export. The importer receives only migration-
document bytes and one absent destination; its migration-only format-4 verifier
has no source repository/store interface. Import writes only a sibling staging
tree, the absent destination, and the destination parent synchronization
required by the publication transaction. Neither command accepts a source
cleanup, rename, mark, or delete option.

The migration document and recovered receipt bytes are the portable audit
bridge. The durable repository copy is local recovery evidence, not canonical
state. Excluded objects remain in the source only. Bindings, caches, locks,
pre-existing recovery journals, event logs, and candidates are not portable
canonical provenance and are never recreated by load.

This establishes C-MG-009: auditability relies on retained explicit artifacts,
not on mixed-version runtime behavior.

### Release gate

Format 5 cannot replace format 4 until fixed fixtures prove at least:

- exact migration bytes and SHA-256 for branching Kahn ready-set ordering and
  strings containing control characters, slash, quote, backslash, non-ASCII
  BMP, and supplementary scalars;
- root, draft, linear, branching, and multi-observer graphs;
- attachment-bearing and empty-attachment material;
- REF/tag fan-out and many-old-to-one-new mapping;
- materialized, unobserved, collapsed, and merged classifications;
- collapsed self-revision removal, message-set preservation, and projected
  mixed parent/Cause dependency cycle rejection;
- candidate, corruption, concurrent-observation, output-sink, staging, and
  target-exists failures; and
- exact receipt bytes on two independent loads of identical bytes;
- pre-publication failure, post-publication readback failure, post-publication
  parent-sync/durability uncertainty, stdout failure, interruption before and
  after rename, and idempotent `load-receipt` recovery; and
- full object-inventory plus REF/tag repository-digest readback.

The release must identify the exact last format-4 exporter and first format-5
importer artifacts. Green runtime tests do not waive explicit owner acceptance
of ADRs 0023, 0025, 0026, and 0027.

This establishes C-MG-010: removing compatibility is gated by a proven and
reproducible migration path.

### Decision precedence

With this ADR accepted:

- ADR 0012 remains precedent for deterministic dump, isolated load,
  no-replace publication, and mapping receipts; its `logical-v1` artifact is
  specific to format 3 to format 4.
- ADR 0013's REF/tag inventory and atomic manifest semantics remain and are
  rewritten through the complete mapping.
- ADR 0016's fail-closed, mode-neutral integrity intent remains; its format-4
  parent closure is validated by the exporter and then replaced, not retained
  at runtime.
- ADR 0018 recovery state remains local and is explicitly excluded.
- ADR 0019 and ADR 0024 local source bindings are excluded and recreated
  explicitly after migration when needed.
- ADR 0021's historical attachment preservation requirement is satisfied by
  Material Blob projection.
- ADR 0023 defines new graph meaning; ADR 0025 defines target bytes and the
  mode-neutral physical integrity boundary.
- ADR 0027 defines the format-5 authoring and inspection interface. Migration
  receipts remain the exact schemas in this ADR rather than inspection output.

## Alternatives

### Read format 4 and format 5 indefinitely

Rejected because graph answers would require two revision authorities and
every command would carry a permanent mixed-semantics branch.

### Use old parent as an implicit Link default

Rejected because an omitted scoped assertion would silently depend on legacy
target bytes and continue the global meaning the new model removes.

### Create synthetic Cause Links for every old parent

Rejected because no observer made those Cause claims. It would preserve shape
by fabricating provenance.

### Rewrite the repository in place

Rejected because failure could strand one repository between formats and make
rollback or audit ambiguous.

### Fail migration whenever a parent is unobserved

Rejected because the operator explicitly accepts that meaning loss. Complete
warning and receipt records provide an auditable policy boundary without
blocking the selected model.

### Keep collapse-induced revision self-edges

Rejected because the result would be canonical corruption. The revision
meaning is removed and reported when both endpoints become one SealID.

### Emit the receipt only to stdout

Rejected because stdout cannot be atomic with directory publication. A durable
non-canonical receipt travels with the atomic target and a read-only command
recovers it after delivery failure.

### Treat atomic rename as durable publication

Rejected because namespace atomicity does not persist the nested staging tree
or the renamed destination entry across a crash. File sync, bottom-up directory
sync, rename, and destination-parent sync are distinct ordered obligations.

### Accept any Base64 string that a decoder maps to the same bytes

Rejected because permissive and strict decoders disagree on nonzero unused pad
bits. Strict RFC 4648 decoding plus exact standard re-encoding gives one
portable artifact byte sequence and digest for every payload.

### Require a non-ABA generation or shared lock across export

Rejected because migration identity is bound to the exact complete value seen
at both captures, not to an otherwise unrepresented transition history between
equal endpoints. A continuous guarantee would require one new coordination
authority honored by every format-4 writer and external filesystem actor. The
exporter instead fails every detected capture mismatch and makes the
change-and-restore limit explicit.

## Consequences

Good:

- The format-5 runtime has one schema and one graph authority.
- Migration is deterministic, isolated, reversible by source retention, and
  auditable through exact ID mappings.
- Old attachment/material bytes and every distinct Link message survive.
- Lost parent meaning, including collapse-induced self-revision meaning, is
  explicit instead of hidden behind compatibility logic.
- Receipt delivery failure is recoverable without rerunning publication.
- Strict Base64 pad bits and re-encoding eliminate decoder-dependent accepted
  migration bytes.

Bad / Risk:

- Every SealID changes and all external SealID references need an explicit
  receipt-based rewrite or remain historical references to the source.
- An unobserved old parent edge has no format-5 graph meaning even though both
  endpoint Seals are converted.
- A revision assertion whose endpoints collapse to one new SealID is removed.
- Candidates and corrupt repositories require operator resolution before
  export.
- Double capture cannot detect a source change-and-restore between `S_0` and
  `S_1`; the artifact is bound to their equal recorded value, not to a claim
  about every intermediate physical state.
- Migration artifacts may be large because complete referenced bytes are
  embedded and buffered.
- A post-publication readback failure can leave a target requiring explicit
  inspection; atomic publication cannot be rolled back by stdout or readback.
- A rename followed by parent-directory sync failure leaves publication
  durability explicitly uncertain and cannot be retried automatically.

Neutral:

- Semantic warnings retain success when receipt delivery succeeds; policy
  automation must inspect the structured classification.
- Loader-created modes are verified for the original load transaction but are
  excluded from canonical repository identity and later receipt recovery.
- No automatic source binding, outer-Git state, cache, event, or recovery state
  migration is added.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-MG-001 implement final format-4 exporter | ADRs 0023, 0025, 0026, and 0027 | C-MG-002, C-MG-003, C-MG-005, C-MG-006, C-MG-009 | exact double-capture maps, read-only source capability, candidate/source-structure rejection, fixed artifact bytes/digest, canonical zero-pad-bit Base64 encoding and re-encoding fixtures, alternate-number-spelling rejection, Kahn order, exact object format, tag reservation, constant exclusions including recovery journal, classification warnings, source pre/post equality, explicit change-and-restore observational-boundary fixture, and no-output failure fixtures |
| A-MG-002 implement deterministic projector | ADRs 0023, 0025, 0026, and 0027 | C-MG-004 through C-MG-006 | complete old/new ID, materialized, unobserved, collapsed, merged-message, and remaining-cycle fixtures |
| A-MG-003 implement isolated format-5 load | ADRs 0023, 0025, 0026, and 0027 | C-MG-001, C-MG-003 through C-MG-007, C-MG-009 | migration-only format-4 decode/re-encode and old-ID verification with no source repository interface; malformed/noncanonical document, alternate-number-spelling, nonzero-pad-bit Base64, and Base64 re-encoding mismatch rejection; projection and warning recomputation; destination-only path-scope, operational final-mode verification, file-sync, bottom-up-directory-sync, no-replace, parent-sync, durability-uncertain, target-exists, and staging fault tests |
| A-MG-004 implement receipt/readback/recovery | ADRs 0023, 0025, 0026, and 0027 | C-MG-007 through C-MG-009 | fixed mode-neutral receipt/repository digest bytes, alternate-number-spelling rejection, full-object and stable-writable-mode readback, durability-uncertain/readback/stdout failure separation, and idempotent load-receipt tests |
| A-MG-005 gate format-5 release | ADRs 0023, 0025, 0026, and 0027 | C-MG-010 | exact exporter/importer artifact IDs, public-schema fixtures, normative-document synchronization, and independent fixture reproduction |

Acceptance of this ADR does not by itself authorize implementation or repository
conversion. Migration of tracked dogfood requires a separately reviewed exact
dump, warning set, destination, command, and owner approval.

ADRs 0023, 0025, 0026, and 0027 must be reviewed as one exact decision set and
accepted by the operator before any format-5 implementation action or
normative-document conversion. Action tables in all four ADRs repeat that same
execution prerequisite; narrower Claim ownership never authorizes an earlier
slice.

The acceptance recorded here satisfies only that decision prerequisite. It
does not start any action in this table; execution requires a separately
authorized task.

## Review

The first independent three-scope review of the prior exact candidate failed on
collapse-induced invalid graphs, incomplete receipt bytes, and an impossible
stdout/publication atomicity claim. The operator selected warning-backed removal
of collapsed revision meaning, an exact receipt normal form, and a revised
recoverable command contract. The committed re-review additionally found the
source observation transaction, recovery exclusion, canonical topological and
string normal form, exact object-format domain, migration-only old-payload
reader boundary, full-tree durability, format-4 source invariants, tag
reservation, and importer action trace incomplete. This candidate addresses
each mechanism; no independent review of the new exact bytes is yet recorded.
The subsequent exact review found that C-MG-009 assigned source-retention
responsibility only to receipt implementation. This candidate assigns the
read-only source and destination-only importer obligations to their actual
export/load actions while retaining receipt recovery under A-MG-004.

The 2026-08-31 three-scope review of exact ADR 0026 digest
`a9c8057dd42ad14f3cd17b0750cfec2fbaf7b7e24ac303824b3378f1bf50d4b6`
found that mathematical integer values did not determine the exact JSON number
bytes used by the migration document, receipt, and repository digest, and that
action gates could be read more narrowly than the shared decision-set gate.
This candidate fixes the only allowed numeric lexemes and applies the same
four-ADR execution prerequisite throughout. These are proposed corrections,
not owner acceptance.

The next three-scope review of exact ADR 0026 digest
`a81ac8255271b33026dbb8538804e0f70acfd349811964ceeefeecd9d543ab35`
confirmed the substantive migration contract but found that the shared gate's
`normative-conversion` wording differed from ADR 0027 and could be confused with
repository conversion. This candidate uses the same exact
`normative-document conversion` gate sentence as the other three ADRs while
retaining the separate per-repository migration approval above.

The primary-agent pre-delegation review re-ran the document/receipt byte-domain,
topological projection, durability, readback, excluded-state, source-retention,
and separate repository-approval checks. It found no additional migration
decision change was required before digest freeze.

The next three-scope review of exact ADR 0026 digest
`37a1c485fe6e94e58ba33e2c89dcf9a4bdb9ce65d477411fbee73b89cdfe9a0b`
found that the standard padded Base64 wording still admitted nonzero unused pad
bits to permissive decoders and that loader-created modes had been conflated
with global canonical integrity. This candidate requires strict pad-bit checks
plus exact decode/re-encode equality, retains final modes as loader
postconditions, and makes their exclusion from repository digests explicit.
These corrections do not authorize migration.

The primary correction review then separated the original load invocation's
final-mode readback from later mode-neutral `load-receipt` recovery and added a
concrete nonzero-pad-bit example plus exporter/importer fixtures.

The next three-scope review of exact ADR 0026 digest
`f5abdcc783991356a95b0180a7f55a1a94eb428370294693954daf1d5f6439c2`
confirmed the durable RV-28 provenance correction but found that `S_0`/`S_1`
equality was described as proof that no intermediate source change occurred.
This candidate defines equal endpoint observations as the complete portable
boundary, explicitly excludes change-and-restore detection and continuous
immutability, and requires that boundary as an exporter fixture. It remains
Proposed.

The final exact-candidate review and owner acceptance are recorded in the
linked acceptance record. All three scopes passed with no P0 through P3
finding. On 2026-08-31, the Operator accepted this ADR at pre-transition digest
`b5c10f1890ad1b08c9b18c4c2d440bf0c2e06461939b2bea3812b3c115290feb`
as part of the exact four-ADR decision set.

## Evidence

- E-MG-001: ADR 0012 proves the repository precedent of final-old-format dump,
  new-format isolated load, complete mapping, and atomic no-replace publish.
- E-MG-002: ADR 0011 makes `parent_revision` an intrinsic format-4 field and
  therefore demonstrates the semantic incompatibility.
- E-MG-003: ADR 0021 requires attachment-bearing historical state to receive
  an explicit projection in any future semantic format change.
- E-MG-004: ADR 0023 removes global parent meaning and ADR 0025 changes every
  structured Seal identity.
- E-MG-005: the operator selected migration instead of compatibility after
  distinguishing Sealgraph's operational model from RefGraph theory work;
  recorded as D-MG-001 and D-UB-002 in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md#operator-directions-recorded).
- E-MG-006: the operator explicitly confirmed that an old parent unobserved by
  Cause Links may lose meaning and requested migration warning; recorded as
  D-CR-002.
- E-MG-007: after independent review, the operator directed that
  collapse-lost revision meaning be removed with a warning, receipt bytes be
  fully canonicalized, and the publication/receipt command contract be revised;
  recorded as D-MG-002 through D-MG-004.
- E-MG-008: the committed three-scope re-review at commit `df113d7` identified
  the remaining observation, canonicalization, compatibility-boundary,
  exclusion, durability, tag, source-invariant, and traceability findings; the
  exact finding inventory and reviewed digests are recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md).
- E-MG-009: the subsequent three-scope review of candidate digest
  `6d6331267464e53438a6b445001acbd222254bf9491ffd8533de3809a7b10dda`
  found C-MG-009 action ownership incomplete; the exact target and finding are
  recorded in the same decision/review record.
- E-MG-010: the three-scope review of digest
  `37a1c485fe6e94e58ba33e2c89dcf9a4bdb9ce65d477411fbee73b89cdfe9a0b`
  recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md)
  identified non-canonical Base64 pad-bit ambiguity. Accepted ADR 0016 and the
  retained ADR 0011 Git-sidecar boundary also require mode-neutral integrity,
  while this ADR's explicit final modes remain evidence of loader behavior.
- E-MG-011: the three-scope review of digest
  `f5abdcc783991356a95b0180a7f55a1a94eb428370294693954daf1d5f6439c2`
  recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md)
  identified the double-capture change-and-restore observation limit.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-MG-001 | E-MG-002, E-MG-004, E-MG-005 | A-MG-003 |
| C-MG-002 | E-MG-001, E-MG-002, E-MG-008, E-MG-011 | A-MG-001 |
| C-MG-003 | E-MG-001, E-MG-007, E-MG-008, E-MG-010 | A-MG-001, A-MG-003 |
| C-MG-004 | E-MG-002, E-MG-003, E-MG-004, E-MG-007, E-MG-008 | A-MG-002, A-MG-003 |
| C-MG-005 | E-MG-004, E-MG-006, E-MG-007 | A-MG-001, A-MG-002, A-MG-003 |
| C-MG-006 | E-MG-001, E-MG-006, E-MG-007, E-MG-008 | A-MG-001, A-MG-002, A-MG-003 |
| C-MG-007 | E-MG-001, E-MG-005, E-MG-007, E-MG-008, E-MG-010 | A-MG-003, A-MG-004 |
| C-MG-008 | E-MG-001, E-MG-006, E-MG-007, E-MG-010 | A-MG-004 |
| C-MG-009 | E-MG-001, E-MG-005, E-MG-009, E-MG-010 | A-MG-001, A-MG-003, A-MG-004 |
| C-MG-010 | E-MG-001, E-MG-004, E-MG-005, E-MG-007 | A-MG-005 |

## Follow-ups

- Preserve the completed three-scope and integrated review receipt for this
  exact accepted decision set; materially changing it requires a new review.
- Under a separately authorized implementation task, copy the fixed canonical
  migration and receipt bytes into `docs/storage-format.md`; ADR 0027's accepted
  terminal/JSON behavior is copied into `docs/cli.md` in the same normative
  synchronization gate.
- Inventory external systems that persist SealIDs and define explicit
  receipt-based rewrites before any real repository conversion.
- Prepare a dogfood migration proposal with exact source digest and full
  `UNOBSERVED_PARENT_DROPPED` preview; do not execute it from this ADR alone.
