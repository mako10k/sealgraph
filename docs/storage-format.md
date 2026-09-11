# Storage format

Status: the checked-in runtime writes native format 5 and supports only the
explicit `universal-blob-v1` extract/load boundary for format 4. Ordinary
runtime readers never interpret formats 1 through 4.

## 1. Layout

```text
.sealgraph/
├── .gitignore                       # recommended outer-Git policy, non-canonical
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

## 2. Config and migration boundary

Format 5 config bytes are exactly:

```text
repository_format = 5
object_format = sha256
ref_format = manifest-v1
```

An exact format-4 config fails before mutation with
`FORMAT4_REQUIRES_MIGRATION` and the commands fixed by ADR 0026. Every other
config is unsupported or malformed. There is no dual reader, ignored legacy
field, compatibility mode, in-place conversion, legacy-parent fallback, or
automatic repair.

`sealgraph migrate extract --source-format 4 --format universal-blob-v1`
is the sole format-5 command that reads a format-4 repository. Its source
access is read-only and isolated from ordinary repository APIs.

`sealgraph load --format universal-blob-v1` accepts one exact canonical
`sealgraph/universal-blob-migration/v1` document from stdin and only an absent
`.sealgraph` target. A migration-only codec verifies every embedded canonical
format-4 payload and old SealID. It has no on-disk format-4 repository API.
Projection, semantic classification, complete staged fsck, receipt creation,
bottom-up durability, atomic no-replace publication, parent-directory sync,
and full digest/readback precede successful receipt delivery.

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

Every content, attachment, Material, Provenance, and Seal is a native Blob.
Each typed ID is the native ObjectID of its exact canonical payload bytes.

This is low-level SHA-256 ODB/forensics compatibility, not a Git repository
contract. `.sealgraph` is not opened as a Git repository, attached as an
alternate to an outer SHA-1 repository, or subjected to Git GC, prune, repack,
refs, maintenance, or porcelain operations.

## 4. Canonical format-5 typed Blobs

Structured Blob encoding is compact UTF-8 JSON with no insignificant
whitespace or trailing LF. Exact required member order is:

```text
material:   schema, content, attachments
attachment: name, media_type, blob
provenance: schema, root, draft, cause_links
cause_link: target_seal, previous_revision_seal_of_target_seal, messages
seal:       schema, material, provenance
```

Schemas are exactly `sealgraph/material/v1`, `sealgraph/provenance/v1`, and
`sealgraph/seal/v5`. Every member is required and every ID is a full
64-character lower-hex native BlobID. Typed Blobs contain no REF, tag, actor,
timestamp, operation event, stale marker, branch, preference, supersession, or
current-head field. Seal contains no intrinsic parent. Unknown members fail.

Strings are valid UTF-8 without Unicode normalization. JSON uses short escapes
for backspace, tab, LF, form feed, and CR; other controls use lower-case
`\u00xx`. Numbers do not occur. Decoders parse, validate, re-encode, and
require byte equality.

Canonical arrays use bytewise order. Attachments sort by
`(name, media_type, blob)` and names are unique. Cause Links sort by target and
have unique targets. Previous SealIDs and messages within each Link are sorted
and duplicate-free. Empty messages are valid and identity-bearing. Invalid or
duplicate inputs are rejected rather than deduplicated.

Root and draft are Provenance booleans. Exact content, attachments, targets,
previous arrays, messages, root, and draft affect the joined SealID.

## 5. Two immutable edge relations

Revision assertion:

```text
observer.provenance.cause_links[target].previous_revision_seal_of_target_seal[]
```

Each element asserts that the exact target has that exact previous revision,
as observed by the containing Seal. Structural revision edges are the union of
all included observer assertions and may branch.

Cause edge:

```text
dependent.provenance.cause_links[].target_seal = exact direct upstream SealID
```

Prefixes, tags, REF names, and selector spelling resolve before Candidate
persistence. Direct IDs commit transitively Merkle-DAG style; flattened
ancestor lists are not stored. Cause self-edges, revision self-edges, and
cycles in the combined Cause/revision graph are corruption. Revision edges are
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

A hexadecimal token is 4 through 64 lower-case hex characters.
`@SEAL_TOKEN` prefix lookup matches valid loose object names repository-wide,
requires exactly one match, and then requires canonical Seal decoding. Zero,
ambiguous, and uniquely matched non-Seal objects are errors.

`REF@hex` resolves uniquely only among the REF's current HEAD and Seals
reachable through its observed structural revision closure. Unrelated loose
objects do not participate. This is a UI scope assertion, not ownership.
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
and use schema `sealgraph/candidate/v5`. The destination REF remains
orchestration state and is never copied into a typed Blob.

Required candidate members are:

```text
schema, ref, expected_ref_head, content, attachments,
root, draft, cause_links
```

`expected_ref_head` is `null` for expected-absent publication or one exact old
HEAD for CAS. It is concurrency state and has no revision meaning.

Candidate Cause inputs resolve under one coherent observation to complete
format-5 Link records. Candidate files are not Blob objects and have no
ObjectID, but writers serialize them deterministically and cleanup compares
their exact persisted bytes with the version loaded for sealing.

The format-5 writer emits compact canonical JSON in required member order and
appends exactly one LF. Readers decode, normalize, re-encode, and require exact
byte equality. Root requires no Cause Links; non-root requires at least one.

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
- Previous-revision assertions are scoped to Cause records and do not satisfy
  the non-root Cause requirement independently.
- There is no generic validation bypass or automatic relink/reseal/repair.

Stale, active-leaf, impact, frontier, and preference are derived and never
stored in Seal, Link, REF, candidate, tag, or canonical config state.

## 11. Disposable cache and Git tracking

`.sealgraph/cache/` may contain a derived revision/Cause index only when bound
to repository/schema version and the complete observation. Cache results MUST
be equivalent to canonical scan, and a hit MUST NOT skip Blob, graph, selector,
or observation revalidation. The current format-5 runtime does not persist a
graph cache; `stale --scan` is therefore semantically identical. Future cache
state remains disposable and never repairs canonical state.

An outer Git repository tracks canonical `.sealgraph/config`, `objects/**`,
and `refs/seals/**/.ref` manifests as ordinary exact-byte files. It must not
stage `index/**`, `cache/**`, `locks/**`, `logs/**`, or
temporary paths. LFS, clean/smudge filters, working-tree encoding, and
line-ending transformation over canonical paths are unsupported.

New standalone `init` repositories include this recommended `.gitignore`:

```gitignore
/index/
/cache/
/locks/
/logs/
/objects/*/.tmp-object-*
/refs/seals/**/.tmp-ref-*
```

The file itself may be tracked by outer Git but is not canonical provenance.
It is created without detecting or inspecting Git. Existing repositories keep
their current policy: re-running `init` does not create or overwrite this file.

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
