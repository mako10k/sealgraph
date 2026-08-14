# Storage format

Status: native v1 byte contract frozen by ADR 0005.

## 1. Layout

Target standalone layout:

```text
.sealgraph/
├── config
├── objects/
│   └── aa/
│       └── bbbbb...
├── refs/
│   └── seals/
│       └── <REF>
├── index/
│   └── <candidate representation>
├── logs/       # optional/local/rebuildable, not canonical
├── cache/      # disposable
└── locks/      # runtime only
```

Canonical state is primarily:

- immutable objects,
- current seal refs.

Candidate/index state is mutable but unsealed.

Caches/logs/locks are not canonical provenance.

## 2. Object compatibility goal

Native objects should use a low-level Git-compatible loose-object envelope/layout where doing so does not distort sealgraph semantics.

Compatibility is for:

- shared object read/write primitives,
- mature libraries,
- forensic inspection,
- recovery/debugging.

Compatibility is NOT a promise that high-level Git commands implement sealgraph semantics.

In particular, standalone does not define branch/HEAD/commit/tree meaning for seals.

## 3. Hash algorithms

Native v1 target:

- native object IDs: SHA-256,
- seal IDs: the native object ID of the canonical seal payload object.

Git-sidecar content references preserve their source Git object identity and algorithm.

Persisted identities MUST be algorithm-tagged in semantic data so SHA-1 and SHA-256 Git repositories can be represented unambiguously.

Example semantic form:

```json
{
  "algorithm": "sha256",
  "hex": "..."
}
```

## 4. Canonical seal payload

Illustrative semantic form:

```json
{
  "schema": "sealgraph/seal/v1",
  "ref": "DESIGN-001",
  "parent": {"algorithm":"sha256","hex":"..."},
  "content": {
    "store": "native",
    "type": "blob",
    "id": {"algorithm":"sha256","hex":"..."}
  },
  "attachments": [
    {
      "name": "benchmark.csv",
      "media_type": "text/csv",
      "blob": {
        "store": "native",
        "type": "blob",
        "id": {"algorithm":"sha256","hex":"..."}
      }
    }
  ],
  "links": [
    {
      "relation": "depend-on",
      "target_ref": "REQ-001",
      "target_seal": {"algorithm":"sha256","hex":"..."}
    }
  ],
  "message": "Reviewed against REQ-001 update",
  "root": false,
  "draft": false,
  "created_at": "2026-08-13T22:00:00Z"
}
```

Native v1 uses `sealgraph-canonical-json-v1`, a compact UTF-8 JSON byte
encoding. It has no insignificant whitespace and no trailing LF. Object members
appear in exactly this order:

```text
seal:       schema, ref, parent, content, attachments, links,
            message, root, draft, created_at
object id:  algorithm, hex
content:    store, type, id
attachment: name, media_type, blob
link:       relation, target_ref, target_seal
```

Every member is required. `parent` is JSON `null` for the first seal; no other
member is nullable. The schema value is exactly `sealgraph/seal/v1`.

Strings must be valid UTF-8. They are emitted without Unicode normalization.
`"`, `\`, and U+0000 through U+001F are escaped; the short JSON escapes are
used for backspace, tab, LF, form feed, and CR, and other controls use lower-case
`\u00xx`. Other Unicode scalar values are emitted as their original UTF-8.
Numbers do not occur in the v1 seal payload. Booleans are `true` or `false`.

Decoders MUST parse the payload, validate its semantics, re-encode it, and
require byte-for-byte equality before accepting it as a canonical seal.

## 5. Canonicalization requirements

At minimum:

- UTF-8,
- deterministic map/key encoding,
- deterministic boolean/null representation,
- deterministic timestamp representation,
- deterministic normalization of links,
- deterministic normalization of attachments,
- duplicate-link policy defined explicitly,
- duplicate-attachment-name policy defined explicitly.

For semantically unordered dependency sets, order MUST NOT change the seal identity.

Canonical ordering:

```text
links: sort by (relation, target_ref, target_seal.algorithm, target_seal.hex)
attachments: sort by (name, media_type, blob.store, blob.type,
                      blob.id.algorithm, blob.id.hex)
```

All comparisons are bytewise ascending UTF-8 comparisons. V1 permits only one
`depend-on` link per `target_ref`; a repeated target REF is an error rather than
an implicit deduplication. Attachment names are unique and a repeated name is
an error.

Do not normalize user content bytes. Blob identity preserves exact bytes.

`created_at` is part of the canonical payload and therefore the seal identity.
It is UTC, truncated to a whole second, and encoded exactly as
`YYYY-MM-DDTHH:MM:SSZ`.

## 6. Native loose object bytes

All native v1 objects are Git-compatible SHA-256 blob objects. Given payload
bytes `P`, the uncompressed envelope is:

```text
blob SP <base-10 byte length of P> NUL P
```

The tagged ObjectID is:

```text
sha256:<lower-case SHA-256 of the uncompressed envelope>
```

The envelope is zlib-compressed into:

```text
.sealgraph/objects/<digest bytes 0..1>/<remaining digest>
```

Object reads recompute the digest, validate the header and declared byte
length, and reject malformed, truncated, trailing, or hash-mismatched data.
An existing mismatched object is corruption and MUST NOT be overwritten as an
automatic repair.

A seal payload is stored through the same blob envelope. Its seal ID is that
blob ObjectID; the `sealgraph/seal/v1` schema distinguishes its semantic use.

## 7. Merkle provenance

A seal stores direct upstream seal identities only.

```text
A seal id = HA

B payload links -> A@HA
B seal id = HB

C payload links -> B@HB
C seal id = HC
```

`HC` therefore commits transitively to `HB`, which commits to `HA`.

Do not flatten all ancestors into every downstream payload.

## 8. Refs

A sealgraph logical REF is a movable name whose file contains one current seal ID.

Target path:

```text
.sealgraph/refs/seals/<REF>
```

Do not map logical REFs onto Git `refs/heads/*`; they are not Git branches.

One REF per file is deliberate to make an outer Git three-way merge produce narrow, meaningful conflicts.

The native v1 logical REF grammar is defined by validating the
constructed full refname:

```text
git-check-ref-format-compatible("refs/seals/" + REF)
```

Validation is implemented in standalone domain code and does not execute Git or
inspect `.git`. It applies the documented `git check-ref-format` rules without
normalization, `--branch` shorthand expansion, or refspec patterns:

- `/` separates non-empty hierarchy components;
- no component begins with `.`, ends with `.lock`, or creates `.`/`..` path
  traversal;
- the name has no `..`, ASCII control/DEL, space, `~`, `^`, `:`, `?`, `*`, `[`,
  backslash, or `@{`;
- it does not begin or end with `/`, contain `//`, end with `.`, or equal `@`.

One-level logical names are valid because `refs/seals/` supplies the required
Git ref namespace components. The logical REF maps byte-for-byte and
component-for-component below `.sealgraph/refs/seals/`; there is no escaping,
path cleaning, Unicode normalization, or case folding. File/directory prefix
conflicts such as `design` and `design/api` are rejected in either creation
order. Intermediate directories are implicit namespace only and do not need a
separate declaration. Each leaf ref file contains exactly
`sha256:<64 lower-case hex bytes>` followed by LF.

Ref mutation is compare-and-swap: the caller supplies the expected current
value (or absence), takes a per-REF exclusive runtime lock, verifies the value,
writes a same-directory temporary file, and atomically renames it. Lock or CAS
failure is reported without repair.

## 9. Candidate state

Candidate files under `.sealgraph/index/` are mutable working state, not
canonical provenance. They store concrete dependency seal IDs; plain
`--depend-on REF` input is resolved before candidate persistence. Candidate
state also records the expected base REF head so sealing can refuse a
concurrent head change.

## 10. Seal admissibility

- An explicitly declared root has no dependencies.
- Every non-root, including a draft, has at least one dependency.
- A draft may retain a concrete non-HEAD dependency.
- A normal non-draft seal requires every link in its complete reachable
  dependency closure to match that target REF's current HEAD.
- There is no v1 generic validation bypass.

Historical links are valid immutable data. Their non-HEAD state is derived from
current refs and is never stored as `stale` in a seal.

## 11. No canonical pack/packed refs in v1

The v1 canonical representation should avoid:

- pack/repack-driven file churn,
- packed ref aggregation,
- storage rewrites unrelated to semantic changes.

Optimization may later exist as disposable cache or a versioned format change.
