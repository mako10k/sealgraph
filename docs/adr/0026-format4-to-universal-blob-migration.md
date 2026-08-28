# ADR 0026: Incompatible format-4 to Universal Blob migration

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-MG-001 through C-MG-010
- Related Evidence: E-MG-001 through E-MG-007
- Pending Decision: owner acceptance of the exact candidate after fresh review
- Supersedes on acceptance: format-4 runtime compatibility as a requirement for
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

The format-5 runtime does not read or write format-4 Seals or Candidates and
does not open format-4 repository state. It has no dual reader, mixed graph,
lazy upgrade, legacy-parent fallback, or in-place rewrite.

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

The importer parses only the migration document. It never opens the source
format-4 repository.

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
  dependencies, and stable complete repository observation; and
- complete absence of candidate state, including corrupt or unrecognized
  candidate entries.

Any candidate blocks export because mutable intent cannot be projected safely.
The operator must explicitly seal or discard it using the format-4 runtime.
Corruption, a dependency cycle, or concurrent change fails nonzero without a
plausible partial document and without repair.

This establishes C-MG-002: migration begins only from one complete, immutable,
and coherently observed format-4 logical state.

### Deterministic migration document

Successful stdout is exactly one compact canonical JSON document followed by
LF with schema `sealgraph/universal-blob-migration/v1`. Unknown members,
noncanonical ordering/escaping, duplicate set members, invalid base64, and a
missing or extra final byte are errors.

JSON string validity, escaping, and byte-equality re-encoding use ADR 0025's
canonical UTF-8 rules. Unicode normalization is never applied. Numbers occur
only in the fixed format fields described below.

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

- `schema` and `object_format` are strings;
- `source_repository.format` is the integer `4`;
- every ID/digest is one 64-character lower-hex string;
- `bytes_base64` and `payload_base64` use the standard padded base64 alphabet
  with no whitespace or line breaks;
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
["source_bindings","cache","logs","locks","temporary_files"]
```

It declares excluded categories, not paths, counts, contents, or evidence that
any category is present. Candidate state is not an exclusion: any candidate
rejects export.

Array order is normative:

```text
objects:        id
seals:          dependency-first topological order, then old SealID
refs:           name
tags:           (ref, name)
materialized:   (observer, child, parent)
unobserved:     (child, parent)
collapsed:      (observer, child, parent)
merged links:   (observer, new_target, old_targets)
excluded IDs:   BlobID
```

All tuple comparisons are bytewise ascending UTF-8. The semantic projection is
computed by the final format-4 exporter using the projection below and is
recomputed byte-for-byte by the importer. Equal complete migration
observations—including canonical state and the excluded loose-object
inventory—produce equal bytes.

The document contains no source path, timestamp, actor, hostname, tool version,
random value, local binding, cache entry, lock, recovery journal, event log,
temporary filename, or outer Git state.

This establishes C-MG-003: the conversion input is a portable, deterministic,
versioned artifact whose omissions and semantic changes are explicit.

### Exact format-4 projection

For every exported format-4 Seal `S`, the importer deterministically creates:

1. one Material Blob containing `S.content`'s BlobID and the same sorted
   attachment names, media types, and BlobIDs;
2. one Provenance Blob containing `S.root`, `S.draft`, and converted Cause
   Links; and
3. one Seal Blob naming those exact Material and Provenance IDs.

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
repository digest, but it is carried by the same directory publication. Load
then fsyncs staging according to the accepted durability contract, validates
the complete format-5 typed closure with `fsck`, and publishes with one atomic
no-replace directory operation. It never merges with or replaces an existing
repository. A platform without atomic no-replace directory publication rejects
load before publish.

Failure states are explicit:

```text
PRE_PUBLICATION_FAILURE
  target is absent; retry is allowed after correcting the reported cause

LOAD_PUBLISHED_READBACK_FAILED
  target exists, but complete post-publication verification did not match;
  do not retry load or delete/repair automatically

LOAD_PUBLISHED_RECEIPT_UNDELIVERED
  target and durable receipt are valid, but stdout delivery failed;
  do not retry load; recover the receipt with load-receipt
```

An interruption before atomic publication leaves no target. An interruption
at or after publication may leave the complete target and durable receipt; the
operator inspects the destination and uses the read-only receipt command rather
than retrying load. A surviving pre-publication staging directory is reported
for explicit inspection and is never adopted or deleted automatically.

This establishes C-MG-007: canonical repository publication is atomic and
never rewrites the source, while post-publication readback/output failure is
recoverable without falsely claiming that the target is absent.

### Canonical load receipt

The receipt schema is
`sealgraph/universal-blob-load-receipt/v1`. Encoding uses compact canonical
UTF-8 JSON plus one final LF. Unknown members, unknown enum strings,
noncanonical member/array order, duplicate set members, malformed IDs, and
missing or extra final bytes are errors.

It uses the same canonical string escaping and byte-equality rules as the
migration document. Numbers occur only as the fixed `published_format` value.

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
- `published_format` is the integer `5`;
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

The schema is `sealgraph/repository-observation/v1`; `format` is integer `5`,
`object_format` is `sha256`, and `config_sha256` is raw SHA-256 of exact config
bytes. `objects` is the sorted complete physical BlobID inventory. `refs` uses
`name, head` records sorted by name; `tags` uses `ref, name, target` records
sorted by `(ref, name)`. Readers validate every loose envelope and
typed/reference closure before accepting the inventory. This digest therefore
covers the published config, object store, and logical manifest contents.

### Receipt delivery and recovery command

After atomic publication, `load` independently reopens the target, validates
the complete repository observation, and requires its digest to equal the
durable receipt. Only then does it copy the already stored receipt bytes to
stdout. Successful exit zero means publication, readback, and complete receipt
delivery all succeeded.

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

The migration document and recovered receipt bytes are the portable audit
bridge. The durable repository copy is local recovery evidence, not canonical
state. Excluded objects remain in the source only. Bindings, caches, locks,
pre-existing recovery state, event logs, and candidates are not portable
canonical provenance and are never recreated by load.

This establishes C-MG-009: auditability relies on retained explicit artifacts,
not on mixed-version runtime behavior.

### Release gate

Format 5 cannot replace format 4 until fixed fixtures prove at least:

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
  stdout failure, interruption at the publication boundary, and idempotent
  `load-receipt` recovery; and
- full object-inventory plus REF/tag repository-digest readback.

The release must identify the exact last format-4 exporter and first format-5
importer artifacts. Green runtime tests do not waive explicit owner acceptance
of ADRs 0023, 0025, and 0026.

This establishes C-MG-010: removing compatibility is gated by a proven and
reproducible migration path.

### Decision precedence

On acceptance:

- ADR 0012 remains precedent for deterministic dump, isolated load,
  no-replace publication, and mapping receipts; its `logical-v1` artifact is
  specific to format 3 to format 4.
- ADR 0013's REF/tag inventory and atomic manifest semantics remain and are
  rewritten through the complete mapping.
- ADR 0016's fail-closed integrity intent remains; its format-4 parent closure
  is validated by the exporter and then replaced, not retained at runtime.
- ADR 0018 recovery state remains local and is explicitly excluded.
- ADR 0019 and ADR 0024 local source bindings are excluded and recreated
  explicitly after migration when needed.
- ADR 0021's historical attachment preservation requirement is satisfied by
  Material Blob projection.
- ADR 0023 defines new graph meaning and ADR 0025 defines target bytes.

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

## Consequences

Good:

- The format-5 runtime has one schema and one graph authority.
- Migration is deterministic, isolated, reversible by source retention, and
  auditable through exact ID mappings.
- Old attachment/material bytes and every distinct Link message survive.
- Lost parent meaning, including collapse-induced self-revision meaning, is
  explicit instead of hidden behind compatibility logic.
- Receipt delivery failure is recoverable without rerunning publication.

Bad / Risk:

- Every SealID changes and all external SealID references need an explicit
  receipt-based rewrite or remain historical references to the source.
- An unobserved old parent edge has no format-5 graph meaning even though both
  endpoint Seals are converted.
- A revision assertion whose endpoints collapse to one new SealID is removed.
- Candidates and corrupt repositories require operator resolution before
  export.
- Migration artifacts may be large because complete referenced bytes are
  embedded and buffered.
- A post-publication readback failure can leave a target requiring explicit
  inspection; atomic publication cannot be rolled back by stdout or readback.

Neutral:

- Semantic warnings retain success when receipt delivery succeeds; policy
  automation must inspect the structured classification.
- No automatic source binding, outer-Git state, cache, event, or recovery state
  migration is added.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-MG-001 implement final format-4 exporter | ADR 0026 | C-MG-002, C-MG-003, C-MG-005, C-MG-006 | fixed artifact bytes/digest, constant exclusions, classification warnings, and no-output failure fixtures |
| A-MG-002 implement deterministic projector | ADRs 0023, 0025, 0026 | C-MG-004 through C-MG-006 | complete old/new ID, materialized, unobserved, collapsed, merged-message, and remaining-cycle fixtures |
| A-MG-003 implement isolated format-5 load | ADRs 0025 and 0026 | C-MG-001, C-MG-007 | target-exists, fault-injection, no-replace, publication-boundary, staging, and platform capability tests |
| A-MG-004 implement receipt/readback/recovery | ADR 0026 | C-MG-007 through C-MG-009 | fixed receipt bytes/digest, full-object readback, stdout-failure state, and idempotent load-receipt tests |
| A-MG-005 gate format-5 release | ADRs 0023, 0025, 0026 | C-MG-010 | exact exporter/importer artifact IDs and independent fixture reproduction |

No implementation or repository conversion is authorized by this Proposed
record. Migration of tracked dogfood requires a separately reviewed exact dump,
warning set, destination, command, and owner approval.

## Review

The first independent three-scope review of the prior exact candidate failed on
collapse-induced invalid graphs, incomplete receipt bytes, and an impossible
stdout/publication atomicity claim. The operator selected warning-backed removal
of collapsed revision meaning, an exact receipt normal form, and a revised
recoverable command contract. No independent review of these revised exact
bytes is yet recorded.

A three-scope review must check the artifact byte contract, closure and cycle
rules, semantic classification completeness, atomic publication feasibility,
and every precedence claim before owner acceptance.

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
  distinguishing Sealgraph's operational model from RefGraph theory work.
- E-MG-006: the operator explicitly confirmed that an old parent unobserved by
  Cause Links may lose meaning and requested migration warning.
- E-MG-007: after independent review, the operator directed that
  collapse-lost revision meaning be removed with a warning, receipt bytes be
  fully canonicalized, and the publication/receipt command contract be revised.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-MG-001 | E-MG-002, E-MG-004, E-MG-005 | A-MG-003 |
| C-MG-002 | E-MG-001, E-MG-002 | A-MG-001 |
| C-MG-003 | E-MG-001, E-MG-007 | A-MG-001 |
| C-MG-004 | E-MG-002, E-MG-003, E-MG-004, E-MG-007 | A-MG-002 |
| C-MG-005 | E-MG-004, E-MG-006, E-MG-007 | A-MG-001, A-MG-002 |
| C-MG-006 | E-MG-001, E-MG-006, E-MG-007 | A-MG-001, A-MG-002 |
| C-MG-007 | E-MG-001, E-MG-005, E-MG-007 | A-MG-003, A-MG-004 |
| C-MG-008 | E-MG-001, E-MG-006, E-MG-007 | A-MG-004 |
| C-MG-009 | E-MG-001, E-MG-005 | A-MG-004 |
| C-MG-010 | E-MG-001, E-MG-004, E-MG-005, E-MG-007 | A-MG-005 |

## Follow-ups

- Run one three-scope review for each exact ADR candidate and an integrated
  cross-ADR consistency pass.
- After acceptance, copy the fixed canonical migration and receipt bytes into
  `docs/storage-format.md` and exact terminal/JSON behavior into `docs/cli.md`.
- Inventory external systems that persist SealIDs and define explicit
  receipt-based rewrites before any real repository conversion.
- Prepare a dogfood migration proposal with exact source digest and full
  `UNOBSERVED_PARENT_DROPPED` preview; do not execute it from this ADR alone.
