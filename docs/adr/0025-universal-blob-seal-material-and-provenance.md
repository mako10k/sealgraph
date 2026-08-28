# ADR 0025: Universal Blob instances for Seal, Material, and Provenance

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-UB-001 through C-UB-010
- Related Evidence: E-UB-001 through E-UB-007
- Pending Decision: owner acceptance of the exact candidate after fresh review
- Supersedes on acceptance: the format-4 canonical Seal and Candidate shape in
  ADR 0011; the format-4 attachment placement retained by ADR 0021
- Superseded by: None

## Context

Format 4 physically stores content, attachments, and canonical Seal payloads
through the same low-level loose-object mechanism, but the semantic model still
describes a Seal as one large special record. It embeds content references,
attachments, Cause Links, root/draft flags, and a global parent field.

The desired Blob layer is smaller: it should know only immutable exact bytes
and their content address. Seal and Provenance should themselves be Blob
instances rather than privileged storage object kinds. A Seal should commit to
material and provenance with the minimum structural metadata needed to join
them.

This repository uses the model for operational work. RefGraph may explore a
more general theory, but RefGraph concepts are not an authority or runtime
dependency here.

## Decision

### One universal immutable Blob

The physical object layer has one canonical type:

```text
Blob {
  id: BlobID
  bytes: byte[]
}
```

`BlobID` remains the accepted Git-compatible SHA-256 loose-blob identity:

```text
envelope = "blob " + base10(len(bytes)) + NUL + bytes
BlobID   = lower_hex(sha256(envelope))
```

The Blob store contract is limited to immutable put, get, existence, and exact
identity verification. It does not know Seal, Material, Provenance, schema,
Cause, REF, candidate, Git commit, media type, or migration meaning. It never
overwrites or repairs an existing object.

This establishes:

- C-UB-001: all canonical immutable payloads share one physical Blob identity
  and storage path; and
- C-UB-002: the Blob layer contains no domain-type dispatch.

ADR 0003's low-level Git compatibility remains an implementation and forensic
property only. It does not make `.sealgraph` a Git repository and does not give
Git commit/tree semantics to any Blob.

### Typed instances are validation obligations

Domain types are interpretations of Blob bytes:

```text
SealID       := BlobID whose bytes are one canonical Seal Blob
MaterialID   := BlobID whose bytes are one canonical Material Blob
ProvenanceID := BlobID whose bytes are one canonical Provenance Blob
```

The strings have the same 64-character lower-hex representation. A caller may
use an ID in a typed position only after fetching the Blob, validating the
expected schema and semantics, re-encoding it canonically, and requiring exact
byte equality. A syntactically valid BlobID is not automatically a valid
SealID, MaterialID, or ProvenanceID.

This establishes C-UB-003: types live above the universal store and remain
fail-closed at every typed reference boundary.

### Format-5 canonical structured Blobs

Format 5 introduces three canonical UTF-8 JSON Blob schemas.

Seal Blob:

```json
{"schema":"sealgraph/seal/v5","material":"<material-id>","provenance":"<provenance-id>"}
```

Material Blob:

```json
{"schema":"sealgraph/material/v1","content":"<blob-id>","attachments":[{"name":"evidence.bin","media_type":"application/octet-stream","blob":"<blob-id>"}]}
```

Provenance Blob:

```json
{"schema":"sealgraph/provenance/v1","root":false,"draft":false,"cause_links":[{"target_seal":"<seal-id>","previous_revision_seal_of_target_seal":["<seal-id>"],"messages":["basis"]}]}
```

Every displayed member is required. Unknown members are errors. The exact
member order is:

```text
seal:        schema, material, provenance
material:    schema, content, attachments
attachment:  name, media_type, blob
provenance:  schema, root, draft, cause_links
cause_link:  target_seal, previous_revision_seal_of_target_seal, messages
```

Encoding is compact UTF-8 JSON with no insignificant whitespace or trailing
LF. String escaping and byte-equality validation use the accepted format-4
canonical JSON rules. Booleans occur only as `root` and `draft`; numbers do not
occur.

Canonical arrays use bytewise ascending UTF-8 comparison:

```text
attachments: (name, media_type, blob)
cause_links: (target_seal, previous_revision_seal_of_target_seal, messages)
previous_revision_seal_of_target_seal: full SealID
messages: bytewise UTF-8 string
```

Array-valued tuple components compare lexicographically by their ordered
elements; when one is an exact prefix, the shorter array sorts first.

Attachment names are unique. At most one Cause Link may target one exact
SealID within one Provenance Blob. Previous-revision and message arrays are
sorted and duplicate-free. Invalid duplicates are rejected, never silently
deduplicated.

`content` and attachment `blob` may name arbitrary exact Blob bytes. Their
meaning belongs to the Material schema and caller; the universal store does not
infer MIME, encoding, or a special content object kind. Each `messages` member
is identity-bearing valid UTF-8 and may itself be empty. The array may be empty.

This establishes:

- C-UB-004: a Seal Blob contains only schema plus exact Material and
  Provenance identities;
- C-UB-005: reusable material and provenance each have independent immutable
  identities; and
- C-UB-006: canonical bytes and ordering determine all typed identities.

### Identity and graph meaning

The SealID commits transitively to exact material and provenance:

```text
SealID = BlobID(canonical Seal Blob bytes)
           -> MaterialID -> content and attachment BlobIDs
           -> ProvenanceID -> root, draft, and exact Cause Link assertions
```

Equal Material and equal Provenance produce the same SealID regardless of REF,
machine, candidate, binding, clock, or publication event. A change to either
typed identity produces a different SealID. One Material or Provenance Blob may
be reused by multiple Seals without copying its bytes.

Provenance is a Blob instance, not a mutable side record. It contains the graph
claims defined by ADR 0023. It contains no REF head, current/stale flag, local
path binding, source adapter selection, Git branch/commit inference, actor,
clock, publication event, or approval assertion.

Root and draft remain provenance claims. A root Provenance has no Cause Links.
Every non-root Provenance has at least one Cause Link, including draft and
explicit historical workflows. Draft changes admission policy, not structural
validity. Previous-revision references do not count as Cause targets.

This establishes C-UB-007: mutable repository observation and local acquisition
state never enter immutable Seal, Material, or Provenance identity.

### Candidate and publication boundary

Candidate state remains mutable orchestration state and is not a Blob instance.
A format-5 Candidate contains at least:

```text
schema, ref, expected_ref_head,
prospective material fields,
prospective provenance fields
```

It contains no intrinsic parent. Exact Candidate encoding and CLI spelling are
specified with the implementation contract, but they may not add hidden
identity fields or bypass canonical structured-Blob validation.

A successful seal operation:

1. validates the complete Candidate and coherent graph observation;
2. writes or verifies content/attachment Blobs selected by explicit input;
3. writes or verifies one canonical Material Blob;
4. writes or verifies one canonical Provenance Blob;
5. writes or verifies one canonical Seal Blob; and
6. performs exactly one expected-old REF-head CAS for exactly one logical REF.

Several immutable Blob puts still produce exactly one prospective logical Seal
identity. Failure before the REF CAS may leave unreachable immutable Blobs;
their presence is not publication and does not authorize automatic deletion or
repair.

This establishes C-UB-008: the existing one-Seal/one-REF publication invariant
is preserved above the simpler Blob layer.

The prospective Seal Blob may already exist because identities are
content-addressed and REF-independent. When `expected_ref_head` differs from
the prospective SealID, publication may verify and reuse that existing Blob and
perform the one REF CAS; it publishes exactly one Seal identity without
claiming that new immutable bytes were allocated. When `expected_ref_head`
already equals the prospective SealID, sealing fails with `SEAL_ID_UNCHANGED`,
does not CAS the REF, and leaves the Candidate for explicit inspection or
discard.

This establishes C-UB-010: one logical publication does not require duplicate
Blob allocation, while an unchanged HEAD is never reported as a new Seal.

### Repository and adapter boundary

The standalone repository continues to use `.sealgraph/objects/` for universal
loose Blobs and one movable manifest per REF. REF heads and tag targets are
BlobID strings that must validate as canonical Seal Blobs. Cache, candidate,
binding, lock, recovery, and event state remain non-canonical namespaces.

The exact format-5 config bytes are:

```text
repository_format = 5
object_format = sha256
ref_format = manifest-v1
```

The file has one LF after every displayed line and no other bytes. Format 5
retains the manifest-v1 REF/tag layout; the repository format number selects
the new structured Blob validation contract.

ADR 0024 source adapters terminate at exact content bytes. They do not create
special Git-backed Blob kinds, make external object availability a read
dependency, or bypass Candidate review.

This establishes C-UB-009: source acquisition, immutable byte storage, domain
validation, and REF publication remain separate responsibilities.

### Decision precedence

On acceptance:

- ADR 0003 remains authoritative for the loose-blob envelope and non-porcelain
  boundary.
- ADR 0011's format-4 Seal shape is superseded; its immutability, REF
  independence, exact Cause identity, and Merkle-DAG intent remain.
- ADR 0013's REF-manifest and tag semantics remain, with targets validated as
  format-5 Seal Blobs.
- ADR 0017 may extend metadata only through another accepted typed schema and
  may not introduce unvalidated generic fields into these Blobs.
- ADR 0019 and ADR 0024 retain non-canonical source bindings and exact-byte
  adapter ingestion.
- ADR 0021's refusal to add attachment mutation remains. Its requirement to
  keep attachments inside format-4 Seal bytes is superseded only for converted
  format-5 repositories; Material Blobs preserve their identities and metadata.
- ADR 0023 is authoritative for Cause-scoped revision observation.
- ADR 0026 is authoritative for the format-4 to format-5 conversion boundary.

## Alternatives

### Keep one monolithic special Seal payload

Rejected because storage and domain concerns remain coupled and independently
reusable material/provenance cannot be addressed directly.

### Add distinct physical object kinds and ID formats

Rejected because it would complicate the Blob layer and object paths without
adding semantic safety beyond typed validation.

### Hash raw payload bytes instead of the accepted Blob envelope

Deferred. It would change every existing content BlobID without being required
to simplify the semantic layer. Such a change needs its own evidence and
migration decision.

### Put REF, event, actor, time, or source path in Provenance

Rejected because those values are mutable, local, or operational observations
rather than minimum portable graph claims.

### Remove attachments during the format change

Rejected because format-4 attachment bytes, names, and media types can be
projected losslessly into Material Blob content. Removal would cause unrelated
data loss.

## Consequences

Good:

- One small immutable store serves arbitrary bytes and every structured domain
  object.
- Seal identity is a minimal join of Material and Provenance identity.
- Structured types remain explicit and independently verifiable above storage.
- Existing content and attachment BlobIDs can survive migration unchanged.
- RefGraph remains a separate theoretical tool rather than a production
  dependency.

Bad / Risk:

- Reading one Seal normally requires fetching and validating at least three
  structured Blobs plus referenced material.
- Typed validation cannot rely on ID syntax and must be performed at every
  boundary.
- More unreachable immutable Blobs may remain after failed publication.
- The schema split changes all format-4 SealIDs even when material is unchanged.

Neutral:

- This ADR does not add attachment mutation, garbage collection, pack files,
  remote fetching, or Git repository semantics.
- This ADR does not authorize runtime dual-read compatibility.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-UB-001 isolate universal Blob store | ADR 0025 | C-UB-001 through C-UB-003 | byte-envelope, idempotent put, corrupt-object, and typed-mismatch tests |
| A-UB-002 implement three canonical schemas | ADR 0025 | C-UB-004 through C-UB-007 | fixed byte/ID fixtures, ordering, duplicate, unknown-field, and transitive identity tests |
| A-UB-003 update candidate publication | ADRs 0023 and 0025 | C-UB-007 through C-UB-010 | fault-injection, pre-existing-ID, unchanged-ID, and REF-CAS tests proving one logical publication and no partial publication |
| A-UB-004 update inspection and fsck | ADRs 0023 and 0025 | C-UB-003, C-UB-006, C-UB-007 | missing/wrong-schema/cyclic-reference fixtures and deterministic output tests |

No implementation action is authorized by this Proposed record. Normative
documentation changes follow acceptance of the exact reviewed candidate.

## Review

The prior three-scope review confirmed the layer separation but found that
identity collapse could create duplicate Cause targets or self-revision edges,
that draft non-root Cause validity was ambiguous, and that existing-ID
publication behavior was unspecified. This revision preserves one Link per
mapped target with canonical previous/message set union, makes the non-root
invariant unconditional, and distinguishes Blob reuse from an unchanged-HEAD
seal attempt. No independent review of these revised exact bytes is yet
recorded.

A three-scope review must verify canonical implementability, interaction with
ADRs 0003/0011/0021/0023/0024/0026, and repository-wide effects before owner
acceptance.

## Evidence

- E-UB-001: ADR 0003 and `docs/storage-format.md` already use one SHA-256
  Git-compatible blob envelope for Seal, content, and attachment bytes.
- E-UB-002: format 4 embeds material, graph, lifecycle, and attachment state in
  one special Seal record, creating the coupling described in Context.
- E-UB-003: ADR 0021 explicitly deferred attachment placement changes to a
  semantic-format decision with migration behavior.
- E-UB-004: ADR 0024 requires all source adapters to converge on exact bytes and
  keeps local bindings outside canonical identity.
- E-UB-005: the operator directed that Blob storage be simplified and that
  Seal and Provenance be Blob instances with minimal Seal metadata.
- E-UB-006: the operator distinguished RefGraph as theory validation and
  Sealgraph as the operational model.
- E-UB-007: the independent three-scope review identified collapse target
  uniqueness, draft/non-root validity, and existing-ID publication as missing
  decisions; the operator selected warning-backed collapse semantic loss and
  new-semantics contract revision.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-UB-001 | E-UB-001, E-UB-005 | A-UB-001 |
| C-UB-002 | E-UB-001, E-UB-005 | A-UB-001 |
| C-UB-003 | E-UB-001, E-UB-005 | A-UB-001, A-UB-004 |
| C-UB-004 | E-UB-002, E-UB-005 | A-UB-002 |
| C-UB-005 | E-UB-002, E-UB-005 | A-UB-002 |
| C-UB-006 | E-UB-001, E-UB-005, E-UB-007 | A-UB-002, A-UB-004 |
| C-UB-007 | E-UB-004, E-UB-005, E-UB-007 | A-UB-002, A-UB-003 |
| C-UB-008 | E-UB-002, E-UB-005 | A-UB-003 |
| C-UB-009 | E-UB-004, E-UB-006 | A-UB-003 |
| C-UB-010 | E-UB-001, E-UB-002, E-UB-007 | A-UB-003 |

## Follow-ups

- Review ADR 0023, this ADR, and ADR 0026 as one decision set while preserving
  their separate authority boundaries.
- After acceptance, replace format-4 normative schema text with the exact
  format-5 byte contract and fixed fixture digests.
- Decide garbage collection only in a separate ADR with reachability,
  concurrency, recovery, and outer-Git safety evidence.
