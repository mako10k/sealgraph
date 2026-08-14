# Storage format

Status: native v2 byte contract frozen by ADR 0006. Native v1 is intentionally
unsupported during the pre-1.0 experimental phase.

## 1. Layout

```text
.sealgraph/
├── config
├── objects/
│   └── aa/
│       └── bbbbb...
├── refs/
│   ├── seals/
│   │   └── <REF>
│   └── tags/
│       └── <REF>/
│           └── <ENCODED_TAGNAME>
├── index/
│   └── <candidate representation>
├── logs/       # optional/local/rebuildable, not canonical
├── cache/      # disposable
└── locks/      # runtime only
```

Canonical state is immutable objects, current seal refs, and immutable
REF-scoped tags. Candidate/index state is mutable but unsealed. Caches, logs,
and locks are not canonical provenance.

An outer VCS checkout may contain only `config`, `objects`, and `refs`. Explicit
`sealgraph init` may recreate missing empty `index` and `locks` after validating
canonical layout; read commands never bootstrap implicitly.

## 2. Native object identity and Git ODB compatibility

The config fixes `object_format = sha256`. All native objects are Git-compatible
SHA-256 loose blob objects. Given payload `P`, the uncompressed bytes are:

```text
blob SP <base-10 byte length of P> NUL P
```

The ObjectID is the lower-case SHA-256 hex of those bytes with no algorithm
prefix. The zlib-compressed envelope is stored at:

```text
.sealgraph/objects/<first 2 hex>/<remaining 62 hex>
```

Readers recompute the digest, validate type/length/compression, and reject
malformed, truncated, trailing, or hash-mismatched objects. Existing corruption
is never overwritten automatically. Seal payloads remain blob objects; Git
commit/tag/branch semantics are not introduced.

## 3. Canonical seal payload

Native v2 uses `sealgraph-canonical-json-v2`: compact UTF-8 JSON with no
insignificant whitespace or trailing LF. Illustrative semantic form:

```json
{"schema":"sealgraph/seal/v2","ref":"DESIGN-001","parent":"<64-hex-seal-id>","content":{"store":"native","type":"blob","id":"<64-hex-object-id>"},"attachments":[],"links":[{"target_ref":"REQ-001","target_seal":"<64-hex-seal-id>","message":"Reviewed requirement basis"}],"message":"Reviewed design","root":false,"draft":false,"created_at":"2026-08-14T00:00:00Z"}
```

Member order is exact:

```text
seal:       schema, ref, parent, content, attachments, links,
            message, root, draft, created_at
content:    store, type, id
attachment: name, media_type, blob
link:       target_ref, target_seal, message
```

Every member is required. `parent` is JSON `null` only for the first seal. The
schema is exactly `sealgraph/seal/v2`. Native IDs are JSON strings containing
exactly 64 lower-case hex characters; `sha256:` and per-ID algorithm members are
not accepted or persisted.

Strings are valid UTF-8 without Unicode normalization. JSON escaping follows
v1: short escapes for backspace, tab, LF, form feed, and CR; other U+0000 through
U+001F controls use lower-case `\u00xx`. Numbers do not occur. Decoders parse,
validate, re-encode, and require byte equality.

Canonical set order uses bytewise ascending UTF-8 comparison:

```text
links:       (target_ref, target_seal, message)
attachments: (name, media_type, blob.store, blob.type, blob.id)
```

There is one dependency link per target REF. Duplicate target REFs and duplicate
attachment names are errors, never silently deduplicated. V2 has no link kind.
Link messages may be empty, are valid UTF-8, and are part of seal identity.

Content bytes are exact. `created_at` is identity-bearing UTC whole-second time
formatted `YYYY-MM-DDTHH:MM:SSZ`.

## 4. Merkle provenance

A seal stores only concrete full IDs of direct upstream seals. Prefixes and tags
resolve before candidate persistence and never appear in a seal. Direct IDs
commit transitively to the provenance DAG; ancestor lists are not flattened.

## 5. Refs

A logical REF head is stored at `.sealgraph/refs/seals/<REF>` as exactly 64
lower-case hex characters plus LF. REFs map byte-for-byte and component-for-
component with no cleaning, escaping, normalization, or case folding.

The constructed `refs/seals/<REF>` follows `git check-ref-format` rules without
normalization, branch shorthand, or refspec patterns. V2 additionally forbids
`@`, reserving it as selector delimiter. One-level and hierarchical REFs are
valid. File/directory prefix conflicts such as `design` and `design/api` are
rejected in either creation order; intermediate directories are implicit.

Ref updates use a per-REF lock, expected old value, same-directory temporary
file, and atomic rename. CAS or lock failures are reported without repair.

## 6. Unique object prefixes

User seal tokens may be 4 through 64 lower-case hex characters. Resolution
matches valid loose object names across the entire ODB. Exactly one match is
required. The resolved object must be a canonical seal owned by the selected
REF. Zero, ambiguous, non-seal, and foreign-REF results are errors.

Canonical refs, tags, candidates, links, and command output use full IDs.

## 7. Tags

An immutable lightweight tag scoped by REF is stored at:

```text
.sealgraph/refs/tags/<REF>/<ENCODED_TAGNAME>
```

The file contains one full seal ID plus LF and must target a canonical seal
owned by `<REF>`. Recreating the same tag/target is idempotent; retargeting is an
error. Tags are aliases only and never persisted inside dependency links.

A raw TAGNAME is non-empty valid UTF-8 without ASCII control/DEL or `@`; `/` is
allowed. Encoding operates on UTF-8 bytes: ASCII letters, digits, `-`, and `_`
remain literal; every other byte becomes `%` plus two upper-case hex digits.
Thus `release/1` becomes `release%2F1` and `v1.0` becomes `v1%2E0`. Raw lower-
case hex names of length 4 through 64 are reserved for object prefixes.

Because hierarchical REFs and tag leaves share this deliberately simple loose
path, a tag leaf that equals the next REF path component can conflict with tags
for a child REF. For example, tag `api` on REF `design` conflicts with the tag
namespace for REF `design/api`. Either creation order is rejected explicitly;
no escaping, relocation, or automatic repair is performed.

## 8. Candidate state

V2 candidate files use schema `sealgraph/candidate/v2`, full-hex JSON string
IDs, and the same attachment/link representation as seals. Candidate dependency
inputs are resolved to concrete full seal IDs immediately. Candidate base
records the observed REF head for CAS sealing.

## 9. Seal admissibility

- An explicitly declared root has no dependencies.
- Every non-root, including a draft, has at least one dependency.
- A draft may retain a concrete non-HEAD dependency.
- A normal non-draft seal requires complete reachable dependency closure HEAD
  consistency.
- Every v2 link participates in freshness, impact, and the Merkle DAG.
- There is no generic validation bypass.

Stale remains derived from immutable seals and current REF heads and is never
stored canonically.

## 10. Experimental format boundary

`repository_format = 2` rejects format 1. There is no dual reader, migration,
or compatibility switch. V2 remains loose-only: no canonical packs, packed
refs, alternates, or object-format translation maps.
