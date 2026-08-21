# Storage format

Status: the checked-in runtime writes native format 4 and supports the explicit
logical-v1 load boundary. Formats 1 through 3 remain unsupported by the
format-4 runtime.

## 1. Layout

```text
.sealgraph/
├── config
├── objects/
│   └── aa/
│       └── bbbbb...                 # remaining 62 lower-hex characters
├── refs/
│   └── seals/
│       └── <REF>/
│           └── .ref                 # HEAD plus immutable scoped tags
├── index/
│   └── <REF>/
│       ├── .candidate               # mutable, unsealed, runtime
│       └── .track                   # local source binding
├── cache/                           # disposable derived graph index
├── logs/                            # optional/local/rebuildable
└── locks/                           # runtime coordination only
```

Canonical state consists of `config`, immutable objects, and one current REF
manifest per logical REF. Each manifest contains its HEAD and immutable scoped
tag bindings.
Candidate/index, cache, logs, locks, and temporary files are not canonical
provenance and must not be tracked by an outer Git repository.

The `.track` entry accepted by ADR 0019 is versioned local source-binding state.
It records one validated working-directory-relative path for its REF. It is
not included in canonical fsck, logical dump/load, REF movement, candidate or
Seal bytes, and its absence does not affect repository validity. Tracking
readers and writers MUST reject symlinks and non-regular entries. Bind is
expected-absent/same-state idempotent; rebind and unbind require exact observed
old paths. One binding replacement is atomic under the repository writer guard.

An outer checkout may contain canonical paths only. Explicit `sealgraph init`
may recreate missing empty runtime directories after validating the canonical
layout; read commands never bootstrap implicitly.

## 2. Config and experimental boundary

Format 4 fixes at least:

```text
repository_format = 4
object_format     = sha256
ref_format        = manifest-v1
```

The format-4 runtime rejects repository formats 1, 2, and 3 and the interim
format-4 config without `ref_format = manifest-v1`. It has no dual reader,
ignored legacy fields, compatibility mode, in-place conversion, or automatic
repair.

Before the runtime reader changes, the format-3 binary gains a versioned
read-only logical dump. Format-4 load accepts only an empty repository, rebuilds
objects topologically, rewrites old IDs through an explicit mapping, validates
the complete revision/Cause graph, and publishes converted REFs explicitly.
Many old owner-salted SealIDs may map to one format-4 SealID; the complete
mapping is an output receipt, not hidden migration state.

ADR 0012 fixes the format-3 command as `sealgraph dump --format logical-v1`
and the envelope schema as `sealgraph/logical-dump/v1`. Its exact compact JSON
member order is:

```text
schema, source_repository, objects, seals, refs, tags, excluded_objects
```

REF heads and tag targets root the exported parent/Cause closure. `objects`
contains exact base64 payload bytes for referenced content and attachments;
`seals` retains each canonical format-3 payload with its old SealID; and
`excluded_objects` reports valid loose IDs outside both roles without copying
their bytes. Any candidate or corruption rejects the dump.

The load target is stricter than an initialized empty repository:
`.sealgraph` must be absent so a complete sibling staging directory can be
validated and published atomically without replacement. Tag records are
rewritten through the complete SealID map and stored in their scoped REF
manifests. The format-4 runtime still never opens a format-3 repository
directly.

## 3. Native object identity and Git ODB compatibility

All native objects use the Git SHA-256 loose-blob envelope. Given payload `P`:

```text
envelope = "blob " + base10(len(P)) + NUL + P
ObjectID = lower_hex(sha256(envelope))
```

`sha256:` is never part of an ID. The zlib-compressed envelope is stored at:

```text
.sealgraph/objects/<first 2 hex>/<remaining 62 hex>
```

Readers validate path, compression, object type, decimal length, exact payload
length, absence of trailing data, and recomputed ObjectID. Malformed or
hash-mismatched objects are never returned as valid and are never overwritten
or repaired automatically.

Seal payloads are native blob objects. `SealID` is the native ObjectID of the
exact canonical Seal payload bytes. Content and attachment blobs use the same
envelope and identity.

This is low-level SHA-256 ODB/forensics compatibility, not a Git repository
contract. `.sealgraph` is not opened as a Git repository, attached as an
alternate to an outer SHA-1 repository, or subjected to Git GC, prune, repack,
refs, maintenance, or porcelain operations.

## 4. Canonical format-4 Seal payload

Encoding is compact UTF-8 JSON with no insignificant whitespace or trailing
LF. The exact required member order is:

```text
seal:       schema, parent_revision, content, attachments, links, root, draft
content:    store, type, id
attachment: name, media_type, blob
link:       target_seal, message
```

Illustrative bytes:

```json
{"schema":"sealgraph/seal/v4","parent_revision":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","content":{"store":"native","type":"blob","id":"abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"},"attachments":[],"links":[{"target_seal":"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210","message":"basis"}],"root":false,"draft":false}
```

Every member is required. `schema` is exactly `sealgraph/seal/v4`.
`parent_revision` is JSON `null` for an initial revision or one exact full
64-character lower-case SealID. All native IDs have that same full-hex form.

The Seal contains no `ref`, `ref_at_seal`, `target_ref`, actor, timestamp,
operation/event message, stale marker, branch, preference, supersession, or
current-head field. Unknown members are errors.

`content.store` is `native` and `content.type` is `blob`. Attachment `blob`
uses the same structure. Attachment names are unique.

Strings are valid UTF-8 without Unicode normalization. JSON uses short escapes
for backspace, tab, LF, form feed, and CR; other U+0000 through U+001F controls
use lower-case `\u00xx`. Numbers do not occur. Decoders parse, validate,
re-encode, and require byte equality.

Canonical array order uses bytewise ascending UTF-8 comparison:

```text
links:       (target_seal, message)
attachments: (name, media_type, blob.store, blob.type, blob.id)
```

At most one Link may target one exact SealID. A duplicate target is an error
even when its message differs. Duplicate attachment names are errors. Inputs
are rejected, never silently deduplicated. Link messages may be empty, are
identity-bearing valid UTF-8, and describe only the exact Cause edge.

Root and draft are identity-bearing booleans. Exact content, attachments,
direct Cause identities/messages, `parent_revision`, root, and draft therefore
all affect SealID.

## 5. Two immutable edge relations

Revision edge:

```text
child.parent_revision = exact parent SealID
```

It asserts derivation only. One Seal has zero or one parent; one parent may
have zero or more children. Parent cycles are corruption. REF ownership or
sibling preference is not validated because neither is canonical Seal state.

Cause edge:

```text
dependent.links[] = exact direct upstream SealIDs
```

Prefixes, tags, REF names, and selector spelling resolve before candidate
persistence. Direct IDs commit transitively Merkle-DAG style; flattened
ancestor lists are not stored. Cause cycles are corruption. Parent edges are
not traversed as Cause edges.

## 6. REF manifests and path grammar

A logical current HEAD and its complete scoped tag namespace are stored at:

```text
.sealgraph/refs/seals/<REF>/.ref
```

The compact canonical JSON schema is `sealgraph/ref/v1`, with no trailing LF.
Exact member order is `schema, head, tags`; each tag uses `name, target`.
`head` and every tag target are full 64-character lower-case SealIDs that must
decode as canonical Seals. Tags are sorted by raw TAGNAME using bytewise UTF-8
order and names are unique. Unknown members, noncanonical bytes/order,
malformed IDs, non-regular entries, and symbolic links fail closed. Multiple
REF manifests may point to the same Seal.

REF paths map byte-for-byte and component-for-component with no cleaning,
escaping, Unicode normalization, or case folding. The constructed path follows
`git check-ref-format` rules without branch shorthand, refspec patterns, or
normalization, and additionally forbids `@` for selector syntax. One-level and
hierarchical REFs are valid.

`.ref` is a reserved terminal marker rather than a REF component. A REF and a
slash-prefixed REF such as `design` and `design/api` can therefore coexist.
The spelling implies no hierarchy, inheritance, or recursive operation.

HEAD and tag updates use validated paths, expected-old state, a shared per-REF
lock, same-directory temporary file, durability steps, and atomic replacement.
CAS, lock, corruption, or durability failures are reported without repair.

## 7. Selectors and unique prefixes

Public selector forms are:

| Form | Resolution |
| --- | --- |
| `REF` | exact current HEAD of REF |
| `@SEAL_TOKEN` | repository-wide unique native ODB prefix that decodes as a canonical Seal |
| `REF@TOKEN` | explicit Seal in a REF UI scope |

A hexadecimal token is 4 through 64 lower-case hex characters. Prefix lookup
matches valid loose object names repository-wide, requires exactly one match,
and then requires canonical Seal decoding. Zero, ambiguous, and uniquely
matched non-Seal objects are errors.

For `REF@hex`, the selected Seal must be the REF's current HEAD or a
`parent_revision` ancestor of it. This is a UI scope assertion, not ownership.
An unscoped sibling or detached Seal uses `@SEAL_TOKEN`. `REF@non-hex` resolves
an immutable tag in that REF's UI namespace.

Bare hexadecimal Seal tokens are not selectors because a REF may itself be
lower hex. Prefixes and selector spelling are never persisted. REFs, tags,
candidates, Links, and identity receipts store or emit full IDs.

## 8. Tags

Tags remain immutable external aliases to exact canonical Seals. Recreating the
same tag/target is idempotent; retargeting is an error. Tags never enter Seal or
Link bytes.

Raw TAGNAME remains non-empty valid UTF-8 without ASCII control/DEL or `@`;
`/` is allowed. Encoding operates on UTF-8 bytes: ASCII letters, digits, `-`,
and `_` remain literal; every other byte becomes `%` plus two upper-case hex
digits. Raw lower-case hex names of length 4 through 64 remain reserved for
object prefixes.

Raw TAGNAMEs and full targets are stored in the REF manifest. Percent encoding
remains the injective interchange/display contract but is not a canonical path
leaf. Creating a tag requires an observed unchanged REF HEAD, so a concurrent
HEAD update cannot silently lose a binding. CLI creation requires a current or
REF-scoped selector; unscoped `@SEAL_TOKEN` has no tag scope.

`sealgraph mv OLD_REF NEW_REF` validates the source manifest, HEAD, and every
tag target, rejects an existing destination or exact source/destination
candidate, and commits with one same-filesystem atomic no-replace rename of the
`.ref` file. HEAD and tags move together. The old name is not retained as an
alias. Candidate state is never moved or rewritten. A post-commit directory
sync error reports that the move may already be visible and requires explicit
inspection before retry.

## 9. Candidate state

Candidate files remain mutable JSON under `.sealgraph/index/<REF>/.candidate`
and use schema `sealgraph/candidate/v4`. The destination REF may remain in the body as
path-validation/orchestration state; it is never copied into a Seal.

Required candidate members are:

```text
schema, ref, parent_revision, expected_ref_head,
content, attachments, links, root, draft
```

`parent_revision` is `null` or the exact parent to hash into the next Seal.
`expected_ref_head` is `null` for expected-absent publication or one exact old
HEAD for CAS. They are distinct even when an ordinary update records the same
SealID in both.

Candidate Link inputs resolve immediately to exact full target SealIDs and use
the format-4 Link representation/order/duplicate rules. Candidate files are not
Seal objects and have no ObjectID, but writers serialize them deterministically
and cleanup compares their exact persisted bytes with the version loaded for
sealing.

The format-4 writer emits compact JSON in the required candidate member order,
uses the same nested content/attachment/Link member order as Seal encoding,
and appends one LF. Readers reject unknown/trailing members and semantic
invalidity but may accept insignificant JSON whitespace because candidates are
mutable orchestration state rather than content-addressed objects. The fixed
writer-byte fixture hash, including LF, is recorded in the format-4 native-core
acceptance receipt.

Every native mutation holds one repository-wide writer guard. Explicit discard
removes only the exact validated regular candidate file. It never moves a REF,
deletes an immutable object, recursively removes a namespace, or repairs state.

## 10. Seal admissibility

- Root is a per-generation identity-bearing property. A root has no Cause
  Links.
- Every non-root, including a draft, has at least one Cause Link.
- A normal non-draft publication requires every direct and reachable Cause
  target to be a non-draft active revision leaf in one coherent current-head
  observation.
- A draft may preserve active, historical, detached, draft, or non-draft exact
  Causes, but immutable graph integrity still validates.
- Revision-parent selection is separate from Cause admission. Active non-leaf,
  detached historical, and draft parents are allowed when selected explicitly.
- Parent draft does not propagate automatically; `derive` copies source draft
  as visible candidate material.
- There is no generic validation bypass or automatic relink/reseal/repair.

Stale, active-leaf, impact, frontier, and preference are derived and never
stored in Seal, Link, REF, candidate, tag, or canonical config state.

## 11. Disposable cache and Git tracking

`.sealgraph/cache/` may contain a derived revision/Cause index bound to
repository/schema version, complete sorted REF/head snapshot digest, and its
own checksum. Cache miss, corruption, or digest mismatch triggers canonical
scan and atomic refresh. Cache write failure may warn without invalidating an
already validated result. Cache never repairs canonical state.

The current disposable file is `.sealgraph/cache/revision-v1.json`, schema
`sealgraph/revision-cache/v1`. It records the sorted active Seal IDs and exact
`parent_revision` values, repository format 4, the complete observation
SHA-256, and a checksum over those derived fields. A cache hit re-reads every
recorded canonical Seal and verifies its parent before graph results are used.
`stale --scan` bypasses the file. Unsafe cache symlinks/non-files are ignored
and never followed; failure to refresh is a warning, not canonical repair.

An outer Git repository tracks canonical `.sealgraph/config`, `objects/**`,
and `refs/seals/**/.ref` manifests as ordinary exact-byte files. It must not
stage `index/**`, `cache/**`, `locks/**`, `logs/**`, or
temporary paths. LFS, clean/smudge filters, working-tree encoding, and
line-ending transformation over canonical paths are unsupported.

## 12. Non-canonical local recovery journal

The recovery boundary uses `.sealgraph/logs/recovery/` for versioned local
operation records. This directory is not canonical repository
state and is excluded from outer Git, logical dump/load, Seal identity, REF
identity, graph derivation, and canonical `fsck` validity.

One record stores a fixed operation kind plus bytewise-REF-sorted transitions.
Each transition stores one exact logical REF and `before`/`after` states, where
each state is either absent or the exact canonical `sealgraph/ref/v1` bytes.
Readers reject duplicate REFs, equal before/after states, malformed present
manifests, and unknown schema members.

A durable `PREPARED` record precedes canonical mutation and an atomic record
replacement marks `COMMITTED` afterward. Exact current state equal to before,
after, or neither classifies not-applied/already-restored, recoverable, or
intervened state. Journal status alone is never sufficient to mutate a REF.

The runtime schema is canonical JSON plus LF with schema
`sealgraph/recovery/v1`. Operation IDs are exactly 32 lowercase hexadecimal
characters generated from 16 random bytes and filenames are
`OPERATION_ID.json`. Byte slices use JSON base64 strings; absence uses `null`.
Unknown members, non-canonical encoding, invalid REF names, operation-shape
mismatches, and records over 64 MiB are rejected. Each present before/after
manifest is limited to 16 MiB. V1 permits exactly one transition for `seal`,
`tag`, or `ref-drop` and exactly two sorted transitions for `mv`; `seal` ends
present, `tag` is present-to-present, `ref-drop` is present-to-absent, and `mv`
contains one present-to-absent plus one absent-to-present transition.
