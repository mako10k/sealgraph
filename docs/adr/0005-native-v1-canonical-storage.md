# ADR 0005: Native v1 canonical storage contract

Status: accepted for native v1; superseded by ADR 0006 for experimental native
v2. This ADR remains the historical record of the v1 decision.

## Context

ADR 0003 selected a Git-compatible loose-object direction but intentionally
left the byte contract pending. Native seals cannot be persisted safely until
payload bytes, object identities, REF paths, ordering, duplicates, event time,
and historical dependency admissibility are deterministic.

## Decision

### Canonical seal payload

Native v1 uses a compact UTF-8 JSON encoding named
`sealgraph-canonical-json-v1`. It is not generic JSON canonicalization. The
member order, required members, string escaping, and array ordering are fixed
by `docs/storage-format.md`. Unknown, omitted, duplicate, or reordered members
are not canonical seal bytes.

User content is stored as an exact byte blob and is never Unicode-normalized or
line-ending-normalized.

### Native object identity and loose envelope

Every native v1 object, including a canonical seal payload, uses the Git blob
envelope:

```text
blob SP <decimal payload byte length> NUL <payload bytes>
```

Its ObjectID is `sha256:` followed by the lower-case SHA-256 digest of the
uncompressed envelope. The loose file contains the zlib-compressed envelope at
`objects/<first-two-hex>/<remaining-hex>`. Consequently, a seal ID is exactly
the native blob ObjectID of its canonical seal payload bytes. This preserves
low-level Git SHA-256 blob compatibility without introducing Git commit,
branch, or repository semantics.

### REF names and files

A native v1 logical REF is a slash-separated path. The constructed full name
`refs/seals/<REF>` must satisfy the same rules as `git check-ref-format` without
normalization or refspec patterns. Thus a logical REF may be one level such as
`ROOT-001` or hierarchical such as `requirements/external/ROOT-001`.

The REF path maps byte-for-byte and component-for-component to
`.sealgraph/refs/seals/<REF>`. Sealgraph does not case-fold, escape, clean, or
otherwise rewrite it. File/directory prefix conflicts such as `design` versus
`design/api` are rejected explicitly in either creation order, as they are
incompatible loose ref namespaces. Intermediate directories are implicit
namespace only; they are not separately declared REF or directory entities.

Each ref file contains exactly one tagged ObjectID and LF. Ref updates use an
expected old value, a per-REF exclusive lock, and an atomic rename. A mismatch
is reported; it is never repaired automatically.

### Sets, sorting, and duplicates

Links are sorted by `(relation, target_ref, target_seal.algorithm,
target_seal.hex)` using bytewise ascending UTF-8 order. V1 supports only the
`depend-on` relation and permits at most one such link per `target_ref`.
Repeated targets are rejected rather than silently deduplicated.

Attachments are sorted by `(name, media_type, blob.store, blob.type,
blob.id.algorithm, blob.id.hex)`. Attachment names are unique; duplicate names
are rejected even when all other fields match.

### Event time and identity

`created_at` is required, normalized to UTC with whole-second precision in the
exact form `YYYY-MM-DDTHH:MM:SSZ`, and included in canonical seal bytes and the
seal identity. Implementations inject a clock for deterministic fixtures.

### Root, draft, and historical dependencies

A root seal must be declared explicitly and has no dependency links. Every
non-root seal, including a draft, has at least one dependency.

An explicit `REF@SEAL` may be recorded in a working candidate. A draft may seal
that historical relation. A normal non-draft seal requires HEAD consistency
for the complete reachable dependency closure, not only its direct links. V1
has no generic force/ignore-validation switch. A non-draft historical candidate
must be made draft or explicitly relinked before sealing.

## Consequences

- Link input order cannot alter a seal ID.
- Changing a message, event time, parent, content, root/draft state, attachment,
  or direct upstream seal identity changes the seal ID.
- Stale remains derived and is absent from canonical payloads.
- REF names and their loose-file paths remain directly usable by a future Git
  sidecar ref namespace without flattening or an incompatible escaping scheme.
- Full transitive status/impact presentation may arrive later, but normal seal
  validation already refuses a stale dependency closure.
