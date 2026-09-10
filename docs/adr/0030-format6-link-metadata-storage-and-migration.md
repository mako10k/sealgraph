# ADR 0030: Format-6 Cause Link metadata storage and migration

Status: Accepted

Accepted: 2026-09-10 by explicit Operator acceptance of the exact reviewed
technical candidate.

Acceptance record:
[`../process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-2026-09-10.md`](../process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-2026-09-10.md).

Execution authority: Separate. Acceptance does not authorize normative
conversion, CLI/output companion acceptance, implementation, migration, commit,
push, release, deployment, or another external write.

Decision owner: Operator.

Requirement authority: accepted
[ADR 0029](0029-extensible-cause-link-metadata.md) and its exact accepted
requirement.

This ADR owns only persisted bytes, typed-schema versions, resource limits,
historical-object readability, repository migration, and the ADR 0028
change-preimage consequence. Exact metadata mutation grammar and human/machine
inspection schemas belong to the separately required CLI/output companion ADR.
No implementation or normative conversion is authorized before the complete
companion set is accepted and separate implementation authority is granted.

## Context

Format 5 encodes a Cause Link with exactly `target_seal`,
`previous_revision_seal_of_target_seal`, and `messages`. Its typed readers reject
unknown members. Accepted ADR 0029 adds one identity-bearing `metadata`
collection, so adding that member to an existing format-5 object would violate
the accepted byte contract rather than extend it compatibly.

Historical format-5 Seals and their IDs must remain valid and addressable.
Re-encoding every historical Provenance would change ProvenanceID and SealID,
then require graph-wide replacement of exact references. Unlike the format-4 to
format-5 transition, however, ADR 0029 does not change Cause or revision graph
meaning. A format-6 reader can therefore retain a narrow, exact historical
format-5 typed-object reader without carrying two graph authorities.

ADR 0028 separately requires persisted Assessment references in a future
successor contract. That feature is not part of the accepted Cause Link metadata
scope. This ADR must account for metadata in ADR 0028's assessment-free change
preimage but must not invent Assessment adoption storage.

## Decision

### Repository and typed-schema versions

The successor repository format is 6. Its exact config bytes are:

```text
repository_format = 6
object_format = sha256
ref_format = manifest-v1
```

with one LF after every line and no other bytes.

Format 6 introduces:

- `sealgraph/seal/v6`;
- `sealgraph/provenance/v2`; and
- `sealgraph/candidate/v6`.

`sealgraph/material/v1`, Blob identity, loose-object paths, REF manifest v1,
attachments, root/draft meaning, Cause direction, and revision assertions remain
unchanged.

A format-6 writer emits only Seal v6, Provenance v2, and Candidate v6. A
format-6 repository may also retain and read exact historical Seal v5 and
Provenance v1 objects and Candidate v5 files under the compatibility rules
below. Schema versions are never guessed from member shape.

### Exact successor records

Seal v6 is compact canonical JSON with exact member order:

```text
schema, material, provenance
```

and logical shape:

```json
{"schema":"sealgraph/seal/v6","material":"<material-id>","provenance":"<provenance-id>"}
```

Every Seal v6 references one valid Material v1 and one valid Provenance v2.

Provenance v2 uses exact member order:

```text
schema, root, draft, cause_links
```

Candidate v6 retains Candidate v5's member order:

```text
schema, ref, expected_ref_head, content, attachments,
root, draft, cause_links
```

Its schema value is exactly `sealgraph/candidate/v6`.

In both Provenance v2 and Candidate v6, a Cause Link has exact member order:

```text
target_seal, previous_revision_seal_of_target_seal, messages, metadata
```

Every member is required. Empty metadata is encoded exactly as
`"metadata":[]`; omission and JSON `null` are invalid. Unknown members are
invalid at every successor typed boundary.

A metadata entry has exact member order:

```text
namespace, schema, value
```

`namespace` and non-null `schema` are JSON strings. `schema` may instead be JSON
`null`. Metadata entries sort by decoded namespace UTF-8 bytes and are
duplicate-free by exact namespace bytes. Namespace comparison is case-sensitive
and receives no Unicode normalization.

All non-metadata string encoding, arrays, IDs, Link uniqueness, and typed Blob
validation retain ADR 0025's canonical rules. Cause Links remain unique by exact
`target_seal` and sort by the tuple:

```text
(target_seal, previous_revision_seal_of_target_seal, messages, metadata)
```

The unique target makes later tuple members non-discriminating between two valid
Links, but including them defines a complete canonical comparator.

### Canonical metadata values

Metadata `value` is one recursively canonical JSON value from this closed set:

- `null`;
- `true` or `false`;
- a signed 64-bit integer from `-9223372036854775808` through
  `9223372036854775807`;
- a UTF-8 string;
- an ordered array of canonical metadata values; or
- an object mapping UTF-8 string keys to canonical metadata values.

An integer uses its shortest base-10 JSON spelling. Leading plus, leading zero,
negative zero, decimal point, fraction, exponent, and whitespace are invalid.
Floating point and integers outside the signed 64-bit range are invalid.

Strings use ADR 0025's canonical JSON scalar escaping and receive no Unicode
normalization. Array order is identity-bearing. Object keys are unique after
JSON decoding and appear in bytewise ascending decoded UTF-8 order. Object order
is canonical, not author-significant. Duplicate keys, even if different escape
spellings decode to the same string, are invalid.

Decoding, semantic validation, canonical re-encoding, and exact byte equality
are required. A generic host JSON serializer or map iteration order is not a
canonical encoder.

### Exact metadata resource limits

The following limits are canonical validity rules for both readers and writers:

- at most 64 metadata entries per Cause Link;
- namespace length from 1 through 255 decoded UTF-8 bytes;
- non-null schema length from 1 through 1,024 decoded UTF-8 bytes;
- at most 16 nested value levels, counting the entry's root `value` as level 1;
- at most 4,096 total value nodes per Cause Link, counting every scalar, array,
  and object as one node;
- at most 4,096 decoded UTF-8 bytes per object key or string value; and
- at most 65,536 canonical bytes for the complete metadata array value,
  including `[` and `]` but excluding the member name, colon, and surrounding
  Cause Link bytes.

All limits must be checked without unbounded recursion or allocation. Exceeding
any limit is invalid; readers do not truncate and writers do not partially
persist.

This ADR adds no new total size limit for a containing Provenance, Candidate, or
Blob. Existing `messages` and Link counts had no such successor limit, and
adding one here could retroactively prevent exact preservation of a valid
format-5 record. The new feature is bounded per Link; any future whole-object
limit requires a separate compatibility decision.

### Historical typed-object compatibility

After explicit repository migration to format 6, ordinary reads accept exactly:

- Seal v5 paired with Material v1 and Provenance v1 under the accepted format-5
  rules; or
- Seal v6 paired with Material v1 and Provenance v2 under this ADR.

Cross-pairing is invalid: Seal v5 cannot reference Provenance v2, and Seal v6
cannot reference Provenance v1. Cause targets and previous-revision IDs may name
either valid Seal generation. Graph derivation treats both generations under
ADR 0023 and never interprets metadata.

The historical reader is schema-exact and read-only. It does not accept unknown
v5 members, synthesize stored metadata, rewrite an object, or emit new v5
objects. When exposed through a successor inspection schema, a historical v5
Link has the logical metadata value `[]`, explicitly identified as a historical
projection rather than bytes claimed to occur in Provenance v1.

Format-5 runtimes reject repository format 6 and do not read Seal v6,
Provenance v2, or Candidate v6. Keeping `messages` preserves field meaning, not
old-reader compatibility.

### Candidate compatibility

A format-6 repository reader accepts exact Candidate v5 and Candidate v6 files.
Reading Candidate v5 is side-effect free and projects each Link's logical
metadata to `[]`. The first separately authorized Candidate mutation writes the
whole resulting Candidate as v6; it preserves every v5 field and every message
exactly and adds empty metadata unless that same authorized operation supplies
metadata for one selected Link.

Opening, inspecting, comparing, or validating Candidate v5 never upgrades it.
Rejected mutation input leaves the original Candidate bytes unchanged. New
format-6 Candidates are always v6.

Publishing a Candidate v5 directly is forbidden. Publication first constructs
and validates the exact in-memory v6 projection, creates Provenance v2 and Seal
v6, performs the existing expected-old and observation revalidation, and only
then follows the one-REF CAS workflow. It never rewrites the old Candidate in
place as a hidden preliminary action.

### Repository migration

Migration from repository format 5 to 6 is one explicit, separately authorized
repository transaction. Exact CLI spelling belongs to the CLI/output companion.
There is no automatic migration during open, inspection, Candidate mutation, or
publication.

The migration operation:

1. acquires the accepted repository-wide process writer guard;
2. verifies the exact format-5 config and a complete successful format-5 fsck;
3. captures the complete canonical REF-manifest map, Candidate-file byte map,
   and physical loose-object identity set;
4. verifies that every Candidate is exact Candidate v5 and every reachable typed
   object is valid under format 5;
5. buffers the exact format-6 config;
6. recaptures all three maps and requires byte-for-byte equality;
7. atomically replaces only the config file with the exact format-6 bytes and
   synchronizes the file and containing directory; and
8. reopens the repository as format 6, validates all retained v5 objects,
   Candidate projections, REFs, and complete graph, and emits success only after
   that readback.

The transaction does not rewrite or copy an immutable Blob, move a REF, change a
tag, rewrite a Candidate, fabricate metadata, or create a Seal. Because the only
canonical mutation is one atomic config replacement, interruption leaves either
the exact format-5 config or exact format-6 config. An ambiguous durability or
readback result is reported as uncertain and must not trigger an automatic
second migration.

Format 6 has no downgrade operation. A format-6 repository may contain v6
objects and Candidate files that format 5 cannot interpret.

### Dump, load, and closure

The existing `universal-blob-v1` document remains exclusively the accepted
format-4-to-format-5 migration boundary. It is not extended to source or produce
format 6.

The initial format-5-to-format-6 transition uses the config-only migration above
and introduces no second dump/load protocol. To recover a format-4 repository,
an operator first performs the accepted format-4-to-format-5 extract/load into
an absent target, verifies that format-5 result, and then separately authorizes
the format-5-to-format-6 migration.

Repository copies, outer-Git synchronization, and any future logical
dump/load surface must preserve every retained v5 and v6 immutable Blob byte and
ID exactly, plus current REF manifests and Candidate bytes. No closure operation
may re-encode a historical v5 object as v6 or infer metadata from messages.

### ADR 0028 change identity

Format-6 Assessment-free change identity uses the successor schema
`sealgraph/upstream-change/v2`. It retains ADR 0028's exact member order:

```text
schema, before_seal, after_material, after_root, after_draft, after_cause_links
```

`after_cause_links` contains complete canonical format-6 Cause Link values,
including the required metadata array. All other field meanings remain those of
ADR 0028. Format-6 operations emit and compare only v2 change IDs, even when
every metadata array is empty. A metadata addition, replacement, or removal
therefore changes `change_id`.

Existing immutable v1 upstream-change or Assessment Blobs are not rewritten.
Migration fabricates no Assessment and adopts none. The separate ADR 0028
storage successor must decide how Assessment references are persisted and how a
historical v1 assessment is represented or rejected; this metadata companion
does not create that reference path.

## Alternatives

### Rewrite every format-5 object as format 6

Rejected because schema bytes would change ProvenanceID and SealID, requiring a
graph-wide reference rewrite and defeating unchanged historical-ID
addressability.

### Add metadata to format 5 without a version change

Rejected because format-5 readers reject unknown members and exact bytes define
typed identity. Reusing the version would create two meanings for one schema.

### Continue to reject all historical objects after migration

Rejected because it would strand REF heads and exact Cause references or force
the prohibited identity rewrite. The narrow v5 reader retains one graph meaning
and is materially different from format 4's incompatible parent semantics.

### Rewrite all Candidate files during repository migration

Rejected because it turns a one-file format transition into a multi-file
transaction and changes mutable WIP even when the operator requested only the
repository transition. Side-effect-free v5 reading and write-on-authorized-edit
preserve the Candidate boundary.

### Add Assessment references in format 6

Rejected for this ADR because it would absorb ADR 0028's separately gated
storage feature. Only the assessment-free change preimage is updated here.

### Reuse upstream-change/v1 with a larger Cause Link

Rejected because the exact nested value meaning would change under the same
schema identifier. Version 2 makes the identity transition explicit.

## Consequences

- Historical format-5 SealIDs, ProvenanceIDs, messages, REF heads, tags, and
  exact Cause references remain unchanged and addressable.
- New metadata-bearing or metadata-empty publications use format-6 identities;
  a semantically unchanged v5-to-v6 reseal does not preserve SealID.
- Format-6 readers carry a narrow exact v5 typed-object and Candidate reader,
  but writers have only one successor format.
- Repository migration is small and atomic because immutable objects, REFs, and
  Candidates are not rewritten.
- Metadata has deterministic resource bounds without imposing a new retroactive
  total size limit on legacy messages or Provenance.
- Format-5 binaries fail closed on format-6 repositories.
- ADR 0028 change IDs move to v2 and change when metadata changes.
- Assessment adoption storage, CLI grammar, public inspection schemas, query,
  registries, and validator runtimes remain unavailable until separately
  decided and authorized.

## Implementation notes

No implementation action is authorized before the CLI/output companion is
accepted.

After the complete decision set is accepted under separate implementation
authority:

- keep v5 readers isolated from v6 writers;
- use iterative validation for metadata depth, nodes, and byte limits;
- add exact boundary fixtures for every canonical value and resource limit;
- prove v5 object and Candidate reads have no write path;
- prove migration changes only config and read back both the transition and the
  complete graph;
- test v5/v6 mixed graph closure, cycles, staleness, history, and admission;
- test Candidate v5 projection and rejection atomicity;
- test upstream-change/v2 IDs across empty, added, replaced, and removed
  metadata; and
- synchronize normative documents only in the separately authorized
  implementation change set.

## Review

Author self-review checked:

- exact schema separation and old-reader failure;
- historical ID and message preservation;
- mixed-generation graph behavior without mixed graph semantics;
- Candidate read versus mutation boundaries;
- config-only migration atomicity and readback;
- canonical value and limit completeness;
- dump/load non-repurposing; and
- ADR 0028 change-preimage scope without Assessment-reference expansion.

Independent review returned `PASS` with no P0 through P3 findings, and the
Operator accepted the exact reviewed technical candidate on 2026-09-10. The
review answered:

1. Is exact v5 reading inside format 6 a safe compatibility boundary rather
   than a return to the rejected format-4 mixed-semantics reader?
2. Do Seal v6, Provenance v2, Candidate v6, and upstream-change/v2 prevent every
   schema-meaning collision?
3. Does config-only migration preserve every immutable ID, REF, message, and
   Candidate byte without creating an unsafe half-migrated state?
4. Are canonical values and all new resource dimensions exact and bounded?
5. Does the ADR satisfy ADR 0028 change identity without preempting Assessment
   storage?

## Evidence

- **E-SM-001:** accepted ADR 0029 defines the semantic boundary, required
  metadata model, compatibility distinctions, companion set, and implementation
  gate.
- **E-SM-002:** accepted ADR 0025 fixes format-5 schema bytes, unknown-member
  rejection, canonical strings, Provenance identity, and SealID commitment.
- **E-SM-003:** accepted ADR 0023 fixes graph derivation exclusively from exact
  Cause targets and previous-revision assertions.
- **E-SM-004:** accepted ADR 0026 shows why the format-4 semantic break requires
  isolated migration rather than an ordinary mixed reader; that rejected premise
  does not apply to ADR 0029's graph-preserving extension.
- **E-SM-005:** accepted ADR 0028 fixes assessment-free change identity and
  retains Assessment-reference storage as a separate successor decision.
- **E-SM-006:** the checked-in format-5 runtime and normative documents use
  repository format 5, Seal v5, Provenance v1, Candidate v5, and the three-field
  Cause Link.

| Claim | Evidence | Required implementation action |
| --- | --- | --- |
| C-SM-001: use explicit repository and typed-schema successors | E-SM-001, E-SM-002, E-SM-006 | A-SM-001, A-SM-002 |
| C-SM-002: preserve v5 objects and exact historical IDs | E-SM-001, E-SM-002 | A-SM-001, A-SM-003 |
| C-SM-003: write only v6 while reading exact v5 history | E-SM-002, E-SM-003, E-SM-004 | A-SM-001, A-SM-003 |
| C-SM-004: define deterministic bounded metadata bytes | E-SM-001, E-SM-002 | A-SM-001, A-SM-004 |
| C-SM-005: preserve Candidate bytes until authorized mutation | E-SM-001, E-SM-006 | A-SM-002, A-SM-004 |
| C-SM-006: migrate only config with coherent readback | E-SM-001, E-SM-004, E-SM-006 | A-SM-003, A-SM-004 |
| C-SM-007: move assessment-free change identity to v2 | E-SM-001, E-SM-005 | A-SM-001, A-SM-004 |
| C-SM-008: exclude Assessment-reference storage | E-SM-001, E-SM-005 | A-SM-005 |

Implementation action identities are:

- **A-SM-001:** implement exact v6 domain and canonical typed-object codecs.
- **A-SM-002:** implement exact Candidate v5 projection and v6 persistence.
- **A-SM-003:** implement explicit config-only repository migration and
  historical v5 reading.
- **A-SM-004:** add canonical, limit, identity, migration, graph, and atomicity
  fixtures.
- **A-SM-005:** keep Assessment-reference storage outside this implementation
  slice and synchronize only the accepted assessment-free change-preimage rule.

## Follow-ups

- Draft the separate CLI/output companion against this exact storage contract.
- Do not begin normative conversion or implementation until both companion ADRs
  are independently reviewed, accepted, and separately authorized.
