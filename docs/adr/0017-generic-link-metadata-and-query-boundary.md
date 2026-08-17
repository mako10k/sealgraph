# ADR 0017: Generic Link Metadata and read-only query boundary

Status: proposed on 2026-08-17; implementation requires explicit acceptance.

## Context

Format 4 has one domain-independent Cause Link. Each Link commits to one exact
upstream SealID and one optional message, but it cannot carry structured
domain-specific relation data. Adding core-defined link kinds would make the
provenance engine own application semantics such as approval, role, evidence
class, or confidence.

Users instead need a generic, identity-bearing structure whose meaning belongs
to an external namespace. They also need deterministic queries over that
structure and the immutable revision/Cause graph.

## Proposed decision

Adopt the detailed contract in
`docs/proposals/link-metadata-and-sealgraphql.md` as the design target.

In summary:

- a Cause Link remains a Cause Link and still targets one exact SealID;
- a Link gains a sorted set of namespaced metadata entries;
- metadata values use a bounded canonical JSON value model without floating
  point numbers;
- namespace and optional schema identifiers are opaque identifiers to core;
- `sealgraph` is a reserved query namespace exposing core Link fields as
  virtual values; it is never duplicated in persisted metadata;
- core validates structure and canonical bytes, but not domain meaning or
  schema conformance;
- every metadata byte is immutable and participates in Candidate/Seal identity;
- the initial SealGraphQL surface is deterministic and read-only;
- querying never changes a Candidate, creates a Seal, moves a REF, repairs a
  Link, or treats a match as approval;
- query mutation and schema validation remain separately gated extensions.

This is a persisted format change. Implementation therefore requires a new
repository/Seal/Candidate format, canonical fixtures, an explicit migration or
empty-target load boundary, compatibility tests, and a separately accepted
ADR. It must not be added as an unknown field to format 4.

## Consequences

- Applications can define relation vocabularies without adding core Link
  kinds.
- Exact namespace/schema/value bytes are committed transitively through the
  existing Merkle DAG.
- Two metadata entries cannot silently collide or be reordered into different
  identities.
- A schema declaration is discoverable but does not imply that SealGraph
  validated, trusts, or can retrieve that schema.
- Core Link fields and user metadata can use one query predicate model without
  risking disagreement between duplicated stored values.
- Existing graph, stale, impact, and admissibility semantics remain based on
  every Cause Link regardless of metadata.
- The first query implementation can evolve independently from mutation
  syntax, while sharing the same domain reader and coherent observation rules.

## Rejected alternatives

### Core-defined Link kinds

Rejected because a kind would either acquire hidden built-in graph semantics
or be only a less extensible metadata key. Cause/staleness behavior must not
depend on application vocabulary.

### Unnamespaced flat keys

Rejected because independently authored vocabularies collide and cannot state
which schema, if any, applies.

### Mandatory schema validation

Rejected for the initial design because it creates resolution, availability,
versioning, trust, and validator-runtime dependencies. Optional validation may
later be an explicit pre-seal operation; canonical decoding remains independent
of it.

### Query mutations in the first surface

Rejected because changing immutable Link Metadata means constructing and
publishing a new Seal under the existing one-REF Candidate/CAS workflow. A
query language must not create a second or implicit publication protocol.
