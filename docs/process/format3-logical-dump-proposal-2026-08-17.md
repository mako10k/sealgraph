# Format-3 logical dump and format-4 load-boundary proposal

Status: accepted by the operator on 2026-08-17; recorded by ADR 0012. The
approval authorizes only the format-3 read-only dump implementation slice. It
does not authorize a format-4 repository transition.

This proposal closes the detail gate left by ADR 0011 for the
`FORMAT3_LOGICAL_DUMP` slice. The structured audit model is
[`../decisions/2026-08-17-format3-logical-dump.think`](../decisions/2026-08-17-format3-logical-dump.think).

## 1. Decision summary

The proposed format-3 command is:

```sh
sealgraph dump --format logical-v1
```

It emits one canonical, versioned logical repository document to stdout. It
does not accept an output path and does not write, repair, lock, initialize, or
otherwise change `.sealgraph/`.

The document contains:

- every format-3 Seal reachable from a current REF or immutable tag through
  parent and Cause edges;
- the exact content and attachment blob bytes referenced by those Seals;
- current REF heads and logical `(REF, TAGNAME, target)` tag records;
- a sorted inventory of valid loose objects deliberately excluded because
  they are neither an exported Seal nor referenced material.

Any candidate entry, corrupt canonical state, ambiguous tag attribution,
invalid graph, or changed final observation rejects the dump. No candidate is
serialized or silently discarded.

## 2. Public command and process contract

`--format logical-v1` is required exactly once. `dump` accepts no positional
arguments, no output-file option, no compatibility switch, and no
repair/ignore flag. This keeps migration output explicit and avoids a second
file-publication transaction in the format-3 binary.

On success:

- exit status is `0`;
- stdout is exactly one canonical `sealgraph/logical-dump/v1` JSON document
  followed by one LF;
- stderr is empty.

CLI usage errors return `2`. Repository, integrity, observation, or output
errors return another nonzero status. All repository validation and final
observation checks finish before the first stdout write. A stdout writer can
still fail after accepting a prefix; a consumer MUST require exit `0` and a
complete canonical document before using the result.

The command is standalone. It opens only `WORKDIR/.sealgraph`, requires the
exact format-3 config, and never searches for or reads `.git`.

## 3. Canonical logical-v1 envelope

The top-level member order is fixed:

```text
schema, source_repository, objects, seals, refs, tags, excluded_objects
```

Illustrative shape, with abbreviated IDs and payloads only for readability:

```json
{
  "schema": "sealgraph/logical-dump/v1",
  "source_repository": {
    "repository_format": 3,
    "object_format": "sha256"
  },
  "objects": [
    {
      "id": "<64-lower-hex-object-id>",
      "type": "blob",
      "data_base64": "<RFC-4648-base64-with-padding>"
    }
  ],
  "seals": [
    {
      "old_seal_id": "<64-lower-hex-seal-id>",
      "payload": {
        "schema": "sealgraph/seal/v3",
        "ref": "ROOT",
        "parent": null,
        "content": {
          "store": "native",
          "type": "blob",
          "id": "<64-lower-hex-object-id>"
        },
        "attachments": [],
        "links": [],
        "root": true,
        "draft": false
      }
    }
  ],
  "refs": [
    {"name": "ROOT", "head": "<old-seal-id>"}
  ],
  "tags": [
    {"ref": "ROOT", "name": "accepted", "target": "<old-seal-id>"}
  ],
  "excluded_objects": ["<64-lower-hex-object-id>"]
}
```

The actual encoding is compact UTF-8 JSON with no insignificant whitespace.
It uses the same string-escape rules as canonical Seal JSON. Base64 is the
standard RFC 4648 alphabet with required `=` padding and no embedded
whitespace. The document has exactly one trailing LF; that LF is part of the
logical-v1 byte contract.

All members are required, including empty arrays. Unknown, omitted, duplicate,
or reordered object members are noncanonical. A loader parses, validates,
re-encodes, and requires byte equality. No path, hostname, tool version,
timestamp, actor, or generated nonce appears in the document, so two
successful dumps of the same validated logical observation are byte-identical.
The nested `source_repository` member order is exactly `repository_format`,
`object_format`.

## 4. Exact record contracts and ordering

### 4.1 Objects

`objects` contains every distinct content or attachment blob referenced by an
exported Seal. The object record members are exactly `id`, `type`, and
`data_base64`, in that order. `type` is exactly `blob`; `id` is the full native
SHA-256 object ID recomputed from the Git-compatible blob envelope; and
`data_base64` encodes the payload bytes, not the zlib file or Git envelope.

Objects are ordered by full `id` using bytewise string order. A physical object
that is both Seal bytes and referenced material appears in `objects` and in
`seals`; the two roles are explicit and are not inferred from its payload.

### 4.2 Seals

`seals` contains every canonical format-3 Seal reached from a REF head or tag
target through zero or more `parent` and Link `target_seal` edges. Record
members are exactly `old_seal_id` and `payload`. The nested payload retains the
exact format-3 semantic members and order:

```text
schema, ref, parent, content, attachments, links, root, draft
```

The exporter re-encodes the payload and requires its native object ID to equal
`old_seal_id`. Parent and Link targets must also be present in `seals`.

Seal records use a deterministic dependency-first topological order over the
union of parent and Cause dependencies: every parent or Link target precedes
the Seal that names it. When more than one Seal is ready, the lower full old
SealID comes first. A cycle in the combined rebuild dependency relation is an
error even if a partial format-3 traversal would otherwise avoid that cycle.

Attachments and Links retain format-3 canonical order. Attachments are ordered
by `(name, media_type, blob.store, blob.type, blob.id)`. Links are ordered by
`(target_ref, target_seal, message)`. Duplicate attachment names, duplicate
format-3 target REFs, noncanonical ordering, or an owner/target mismatch is an
integrity error, never normalized in the dump.

### 4.3 REFs and tags

REF records contain exactly `name` and `head`, sorted by bytewise REF name.
Every head must be an exported canonical format-3 Seal whose embedded `ref`
equals the REF name.

Tag records contain exactly `ref`, `name`, and `target`, sorted by `(ref,
name)` using bytewise UTF-8 string order. `name` is the decoded raw TAGNAME, not
the percent-encoded path component. Every target must be an exported canonical
format-3 Seal owned by the tag's REF scope.

The exporter inventories the complete physical `refs/tags` tree and requires
each regular tag file to be attributable to exactly one current REF plus one
canonical encoded TAGNAME. It rejects:

- a file/directory REF-prefix collision;
- a tag file with no current REF scope;
- a noncanonical or undecodable encoded TAGNAME;
- duplicate logical `(ref, name)` records;
- a non-regular entry or symbolic link;
- an unreadable, non-Seal, or foreign-owner target.

The dump records logical tags only. It does not choose the deferred format-4
rename-safe tag path.

### 4.4 Excluded objects

The exporter validates and inventories every physical loose object. An object
that is neither an exported Seal nor referenced content/attachment material is
listed by full ID in `excluded_objects`, sorted bytewise. Its bytes are not
copied into the logical dump.

This makes dangling or candidate-abandoned objects visible without treating
object-store presence as publication or guessing whether arbitrary blob bytes
were intended to be an unreachable Seal. The source repository remains
unchanged and retains those objects. Format-4 load does not import them and
does not produce an old-to-new Seal mapping for them.

## 5. Candidate policy

Any non-directory entry below `.sealgraph/index/` rejects the dump before
output. This includes a valid candidate, corrupt JSON, a noncanonical
candidate, a regular file at an invalid REF path, a symbolic link, device, or
other non-regular entry. Empty real directories are ignored as runtime
scaffolding.

The exporter does not read candidate content into the document and does not
decide whether a candidate should be sealed, discarded, rebased, relinked, or
translated. For a valid path it reports the candidate REF and directs the
operator to `sealgraph candidate show REF`, followed by an explicit `seal` or
`candidate discard`. Unsafe paths fail with a path-focused inspection error
without printing candidate bytes.

This is stricter than silently omitting candidate state because a candidate
may be the operator's only record of unsealed intent. It is safer than
serializing it because format-3 `base` cannot be translated automatically into
the distinct format-4 `parent_revision` and `expected_ref_head` fields after
REF state changes.

## 6. Validation and read-only observation

Before encoding, the command validates:

1. exact format-3 config and safe repository directory types;
2. the complete loose-object path inventory, envelopes, hashes, and payload
   lengths;
3. an empty candidate file inventory;
4. every REF and tag path/value plus its format-3 ownership rule;
5. every REF/tag-rooted canonical Seal, parent edge, Cause edge, content blob,
   attachment blob, and both graph cycle classes;
6. root/non-root, draft, attachment, Link, and canonical-byte invariants;
7. the complete object/REF/tag/candidate inventory and mutable REF/tag bytes a
   second time after the output has been buffered.

If the final observation differs, the command fails with zero stdout and asks
the operator to stop concurrent writes and rerun. As with other coherent read
commands, successful output is a validated observation, not a reservation.
External/manual ABA filesystem changes outside the Sealgraph writer protocol
are not claimed to be excluded.

The dump operation never:

- acquires the exclusive mutation guard;
- creates a lock, cache, log, temporary file, object, REF, tag, or candidate in
  `.sealgraph/`;
- changes permissions or timestamps intentionally;
- repairs, removes, deduplicates, relinks, or reseals state;
- reads or writes Git state.

## 7. Future format-4 load and identity receipt

The paired future command is proposed as:

```sh
sealgraph load --format logical-v1 < repository.dump.json
```

It belongs to the format-4 binary and is not part of the first
`FORMAT3_LOGICAL_DUMP` implementation slice. Parsing
`sealgraph/logical-dump/v1` is a migration-document parser, not a format-3
repository reader. The format-4 runtime still rejects a format-3 `.sealgraph`
directory.

The load command requires `WORKDIR/.sealgraph` to be absent. It does not load
into, merge with, replace, or repair an initialized repository, even one that
appears empty. It builds a complete format-4 repository in a sibling staging
directory, validates it through the format-4 reader, and publishes it with one
atomic no-replace directory operation. The target appearing concurrently is an
error. A platform without the required no-replace publication primitive
refuses load rather than weakening the boundary.

Before publication, load:

1. validates and canonical-round-trips the complete logical-v1 input;
2. validates all old object IDs and the complete old parent/Cause graph;
3. writes referenced material blobs, whose IDs remain unchanged;
4. rebuilds format-4 Seals in the document's dependency-first order, removing
   owner `ref` and Link `target_ref` only as specified by ADR 0011;
5. rewrites every parent, Link, REF, and tag target through the generated
   mapping;
6. validates the complete resulting format-4 revision and Cause DAG;
7. creates no candidate, cache, log, or stale state.

Successful stdout is one canonical compact JSON document plus LF with schema
`sealgraph/logical-load-receipt/v1`. Its fixed members are:

```text
schema, source_dump_sha256, old_to_new_seals, collapsed_seals, refs, tags,
published_repository_format
```

`source_dump_sha256` is the lower-case SHA-256 digest of the exact canonical
dump bytes including its trailing LF. `old_to_new_seals` contains every
exported old SealID exactly once, sorted by old ID, with members
`old_seal_id`, `new_seal_id`. `collapsed_seals` contains only new SealIDs
having two or more old inputs, sorted by new ID, with each sorted complete
`old_seal_ids` group; its member order is `new_seal_id`, `old_seal_ids`. REF
receipt records use `name`, `head`; tag receipt records use `ref`, `name`,
`target`. REF and tag records retain the dump ordering and contain rewritten
full format-4 targets. `published_repository_format` is the integer `4`. The
mapping is output evidence and is not hidden mutable repository state.

On validation or staging failure, `.sealgraph` remains absent. Normal cleanup
may remove only the command's known staging directory. A crash may leave that
noncanonical staging directory; a later command reports it for explicit
inspection and never adopts or deletes it automatically.

## 8. Tag publication conflict and required sequencing decision

The current tracked format-3 dogfood contains tags, while ADR 0011 explicitly
defers the format-4 rename-safe tag namespace. A complete load cannot both
preserve those tags and publish them canonically before that namespace is
approved.

This proposal recommends the fail-closed choice:

- logical-v1 dump always preserves and validates logical tag records;
- format-4 load rejects a non-empty `tags` array until the rename-safe
  format-4 tag contract is accepted and implemented;
- load never drops tags, writes the old collision-prone layout, or stores a
  private deferred-tag manifest;
- `TAG_CONTRACT` must therefore move before the tag-bearing
  `FORMAT4_DOGFOOD_LOAD` execution gate.

This is a planning correction, not part of the format-3 dump code. Approval of
this proposal authorizes updating `PLAN.pert` and the implementation plan to
remove the current ordering conflict before format-4 load is implemented. An
alternative explicit operator decision to abandon and later recreate existing
tags would require a separate recorded migration exception; it is not assumed.

## 9. Old-to-new mapping and duplicate semantics

Every `seals` record receives exactly one mapping entry. More than one old ID
may map to one new ID only when the fully rewritten format-4 canonical payload
bytes are identical. Such convergence is normal when owner REF names or Link
`target_ref` names were the only format-3 differences.

The loader never silently selects one old record as primary. It emits every
old-to-new pair and every many-to-one group. A duplicate old ID, conflicting
old payload, missing dependency mapping, or two records that claim the same
old ID rejects the input. Two distinct new IDs never replace one another based
on REF order, tag reachability, or current-head preference.

REF aliases that rewrite to the same new SealID remain separate REF records.
Tags that rewrite to the same new SealID remain separate logical tag records.
Neither affects Seal identity.

## 10. Acceptance and boundary examples

The accepted implementation must prove at least these cases:

1. Two dumps of an unchanged repository are byte-for-byte identical.
2. NUL, CRLF, non-UTF-8, and missing-final-LF material round-trips through
   padded base64 without text normalization.
3. A parent and every Cause target precede its dependent Seal; a diamond uses
   old SealID to break a ready-set tie.
4. A current REF, a tag-only historical Seal, their ancestry/Cause closure,
   and all referenced material are included exactly once by role.
5. A valid but unreachable blob appears only in `excluded_objects`; corrupt or
   malformed unreachable object storage rejects the entire dump.
6. A valid, corrupt, unsafe-path, or concurrently appearing candidate rejects
   the dump without candidate bytes or stdout.
7. A hierarchical REF/tag prefix collision, orphan tag scope, foreign-owner
   tag, or noncanonical TAGNAME path rejects the dump.
8. A REF, tag, object inventory, or head value change during collection
   rejects the buffered result with zero stdout.
9. Same logical content under different format-3 owner REFs may produce two
   complete old-to-new entries, one collapse group, and one format-4 SealID.
10. A missing mapping dependency, conflicting old record, or combined rebuild
    cycle rejects load before target publication.
11. An existing `.sealgraph`, including an initialized empty one, is unchanged
    and causes load to fail.
12. A non-empty tag array causes format-4 load to fail until the rename-safe
    tag contract is accepted; no tag is dropped or privately deferred.
13. Successful load publishes the complete staged repository once and emits a
    receipt whose source digest matches the exact accepted dump bytes.
14. Neither dump nor its tests inspect `.git` or mutate the source repository.

## 11. Implementation slice after approval

The first implementation remains format-3 dump only:

1. add a migration-document model and canonical logical-v1 encoder in a
   package independent from CLI parsing;
2. add strict native loose-object and complete tag-tree inventory operations;
3. add repository orchestration for candidate rejection, graph closure,
   dependency-first ordering, exclusion reporting, and final revalidation;
4. add `dump --format logical-v1` parsing and buffered stdout emission;
5. add deterministic fixtures for empty, chain, diamond, history, attachment,
   binary material, tag, and unreachable-object cases;
6. add corruption, collision, candidate, concurrent-observation, no-stdout,
   no-mutation, and no-Git-detection tests;
7. keep the runtime config, Seal schema, tracked `.sealgraph`, PERT task state,
   and release state at format 3.

Format-4 Seal/candidate code, load, tracked dogfood conversion, tag namespace,
REF move, cache/scan, Git sidecar, release, and external Issue changes remain
outside this implementation slice.

## 12. Approval questions

Operator approval is requested for the proposal as one contract, with special
attention to these choices:

1. required `dump --format logical-v1` with canonical JSON on stdout only;
2. reject every candidate rather than omit or translate it;
3. export only REF/tag-rooted logical Seals/material while explicitly listing
   excluded loose-object IDs;
4. strict logical tag preservation and collision rejection;
5. future load into an absent target via staged atomic no-replace publication;
6. complete mapping plus explicit many-to-one collapse groups;
7. move `TAG_CONTRACT` before tag-bearing dogfood load instead of dropping or
   privately deferring tags.
