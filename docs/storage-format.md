# Storage format

Status: native format-4 contract accepted by ADR 0011. The checked-in runtime
still writes format 3 until the explicit dump/load implementation transition.
Formats 1 through 3 remain unsupported by the format-4 runtime.

## 1. Layout

```text
.sealgraph/
├── config
├── objects/
│   └── aa/
│       └── bbbbb...                 # remaining 62 lower-hex characters
├── refs/
│   ├── seals/
│   │   └── <REF>
│   └── tags/                        # exact format-4 mapping is deferred
├── index/
│   └── <REF candidate>              # mutable, unsealed, runtime
├── cache/                           # disposable derived graph index
├── logs/                            # optional/local/rebuildable
└── locks/                           # runtime coordination only
```

Canonical state consists of `config`, immutable objects, current loose Seal
REF files, and immutable tags once their rename-safe mapping is approved.
Candidate/index, cache, logs, locks, and temporary files are not canonical
provenance and must not be tracked by an outer Git repository.

An outer checkout may contain canonical paths only. Explicit `sealgraph init`
may recreate missing empty runtime directories after validating the canonical
layout; read commands never bootstrap implicitly.

## 2. Config and experimental boundary

Format 4 fixes at least:

```text
repository_format = 4
object_format     = sha256
```

The format-4 runtime rejects repository formats 1, 2, and 3. It has no dual
reader, ignored legacy fields, compatibility mode, in-place conversion, or
automatic repair.

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

The future load target is stricter than an initialized empty repository:
`.sealgraph` must be absent so a complete sibling staging directory can be
validated and published atomically without replacement. Tag-bearing load
remains blocked until the rename-safe format-4 tag layout is accepted. The
format-4 runtime still never opens a format-3 repository directly.

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

## 6. REF files and path grammar

A logical current HEAD is stored at:

```text
.sealgraph/refs/seals/<REF>
```

The file contains exactly one full 64-character lower-case SealID plus LF. Its
target must decode as a canonical Seal, but the Seal contains no owner name.
Multiple REF files may point to the same Seal.

REF paths map byte-for-byte and component-for-component with no cleaning,
escaping, Unicode normalization, or case folding. The constructed path follows
`git check-ref-format` rules without branch shorthand, refspec patterns, or
normalization, and additionally forbids `@` for selector syntax. One-level and
hierarchical REFs are valid.

File/directory prefix conflicts such as `design` and `design/api` are rejected
in either creation order. Intermediate directories are implicit.

Updates use validated paths, expected-old value, per-REF lock,
same-directory temporary file, durability steps, and atomic replacement. CAS,
lock, prefix-conflict, or durability failures are reported without repair.

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

The old `refs/tags/<REF>/<ENCODED_TAGNAME>` layout is not accepted as the final
format-4 rename-safe mapping. External stable namespace, manifest, or crash-safe
multi-path transaction details must be approved before format-4 tag creation or
`sealgraph mv` implementation. No namespace is inferred from a Seal.

## 9. Candidate state

Candidate files remain mutable JSON under `.sealgraph/index/<REF>` and use
schema `sealgraph/candidate/v4`. The destination REF may remain in the body as
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

An outer Git repository tracks canonical `.sealgraph/config`, `objects/**`,
`refs/seals/**`, and the eventual accepted tag files as ordinary exact-byte
files. It must not stage `index/**`, `cache/**`, `locks/**`, `logs/**`, or
temporary paths. LFS, clean/smudge filters, working-tree encoding, and
line-ending transformation over canonical paths are unsupported.
