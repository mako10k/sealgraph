# ADR 0029: Extensible Cause Link metadata and revision-context boundary

Status: Accepted

Accepted: 2026-09-10 by explicit Operator acceptance of the exact reviewed
technical candidate.

Acceptance record:
[`../process/adr-0029-extensible-cause-link-metadata-acceptance-2026-09-10.md`](../process/adr-0029-extensible-cause-link-metadata-acceptance-2026-09-10.md).

Execution authority: Separate. Acceptance does not authorize normative
conversion, companion acceptance, implementation, migration, commit, push,
release, deployment, or another external write.

Decision owner: Operator.

Requirement authority: exact revision-2 requirement accepted in
[`../process/cause-link-extensible-metadata-requirement-r2-acceptance-2026-09-10.md`](../process/cause-link-extensible-metadata-requirement-r2-acceptance-2026-09-10.md).

This ADR owns only the semantic and responsibility boundary. Exact
persisted bytes, repository and typed-schema versions, numeric resource limits,
migration, CLI grammar, and machine-output schemas require separately accepted
companion ADRs. No implementation or normative conversion is authorized by this
accepted semantic decision.

## Context

Format 5 stores each Cause Link inside immutable Provenance with exactly one
target SealID, an observer-scoped previous-revision SealID set, and a free-form
`messages` set. This preserves structural revision evidence but does not give an
observer a versioned structured place to explain why one exact target is asserted
as a revision of one or more exact previous Seals.

The accepted requirement keeps `messages`, makes structured use optional, adds
one generic extensible area rather than one core field per vocabulary, and keeps
Cause Links embedded in Provenance without CauseLinkID. Its motivating profile
includes progress, correction of a Cause inconsistency, and reconciliation, but
none of those values may become a core graph state.

Accepted ADR 0023 owns structural Cause and revision meaning. Accepted ADR 0025
owns format-5 typed-Blob identity and permits metadata only through a later
accepted schema. Accepted ADR 0027 owns format-5 authoring and inspection
schemas. Accepted ADR 0028 separately owns upstream-impact Assessment identity,
evidence, adoption, disposition, and admission effects.

Proposed ADR 0017 and its detailed proposal contain useful generic-metadata
rationale but couple it to format-4 assumptions and a query-language design.
They are non-normative and do not reflect the accepted Cause-scoped revision,
Universal Blob, or Assessment boundaries.

## Decision

### One embedded generic metadata collection

A successor Cause Link retains its three current fields and adds exactly one
optional logical metadata collection:

```text
cause_link := {
  target_seal: SealID,
  previous_revision_seal_of_target_seal: [SealID],
  messages: [string],
  metadata: [metadata_entry]
}

metadata_entry := {
  namespace: string,
  schema: string | null,
  value: canonical_value
}
```

`messages` retains its current identity-bearing, sorted, duplicate-free string-
set meaning and remains usable without metadata. Existing messages are never
parsed, reclassified, or migrated into structured values.

Cause Links remain records inside Provenance. There is no CauseLinkID,
independent Link Blob, active-Link set, or separate Link publication lifecycle.
Metadata is committed as part of the containing Provenance and transitively by
SealID.

### Namespace and schema responsibility

`namespace` is a non-empty, case-sensitive UTF-8 identifier owned outside
Sealgraph core. One Link contains at most one entry for one exact namespace.
Entries have a deterministic namespace order selected by the storage companion.
Core does not verify namespace ownership, rewrite aliases or case, or assign
truth, trust, approval, preference, or authority to a namespace.

`schema` is either null or a non-empty, case-sensitive, immutable-version
identifier selected by the namespace owner. Core preserves and exposes it but
does not dereference it or imply conformance. Schema registries, network
retrieval, and mandatory namespace-specific validation are outside this ADR.

This ADR reserves no persisted `sealgraph` namespace and adopts no virtual query
namespace. A later core-owned persisted namespace requires its own accepted
decision and cannot be inferred from Proposed ADR 0017.

### Canonical value responsibility

`canonical_value` supports only:

- null;
- boolean;
- a bounded integer representation;
- UTF-8 string;
- an ordered array of canonical values; and
- a string-keyed object whose keys are unique and canonically ordered.

Floating-point values, executable values, environment expansion, dynamic REF or
HEAD selectors, and implicit type coercion are prohibited. Strings receive no
implicit Unicode normalization. Array order is identity-bearing. Object order is
canonical rather than author-significant.

The storage companion must choose the exact integer range and encoding, member
order, string escaping, depth, node count, string/key size, namespace-entry
count, per-Link metadata bytes, and any containing-Provenance or object limit.
Those limits apply to readers and writers and cannot depend on environment,
presentation destination, or namespace meaning.

Unknown but structurally valid namespaces and schemas remain canonically
readable and exactly inspectable. Core validates structure, bounds, canonical
bytes, and identity commitment only; it does not validate domain meaning.

### Structural graph independence

Cause and revision edges continue to derive exclusively from
`target_seal` and `previous_revision_seal_of_target_seal`. Metadata cannot add,
remove, prefer, invalidate, or weaken validation of an edge. Staleness, active
leafness, history, impact, frontier membership, cycle detection, and Cause-
closure admission do not interpret namespace, schema, or value.

Metadata remains identity-bearing. Changing metadata changes the containing
Provenance identity and therefore changes SealID and may change assertion-source identity,
comparison output, and other exact output bytes. Semantic non-interpretation
does not imply byte-identical identity or inspection output.

### Revision-context profile

The first design documents revision context as a generic profile example, not a
core-owned persisted namespace or schema. A namespace owner may define a
versioned value containing entries such as:

```text
revision_context := {
  revisions: [revision_explanation]
}

revision_explanation := {
  previous_seal: SealID,
  kind: string,
  rationales: [string]
}
```

The profile binds each explanation to one exact previous SealID. A profile may
require `previous_seal` membership in the Link's structural previous-revision
set and may define values such as `progress`, `correction`, or
`reconciliation`. Those are namespace-owner claims. Core does not derive
supersession, preference, truth, approval, or graph behavior from them.

The profile does not contain trusted actor/time or seal-operation event data. It
is not ADR 0028's upstream Assessment and cannot carry or satisfy AssessmentID,
change binding, adopted evidence, compatibility disposition, or admission state.
If a future owner wants Sealgraph to maintain a normative revision-context
namespace or validator, that requires a separate accepted decision.

### Candidate and inspection boundary

Metadata editing occurs only on one exact target Link in one Candidate. Adding,
replacing, and removing one namespace are explicit operations. A metadata-only
operation preserves the target, previous-revision set, `messages`, and every
other namespace unless the same authorized Candidate operation explicitly
changes them. Rejected input leaves the prior Candidate bytes unchanged.

Metadata mutation never creates a Seal or moves a REF implicitly. Publication
continues through the one-Candidate, one-REF, expected-old CAS workflow.

Human inspection identifies namespace and schema and uses bounded unambiguous
escaping for arbitrary values. Versioned machine output returns complete
canonical values. Candidate comparison and `linklog` distinguish `messages`
changes from metadata namespace/value changes. Exact command spelling and output
schemas belong to the CLI companion ADR.

### Compatibility and companion decision set

Keeping `messages` preserves field meaning and authoring continuity only. It does
not make successor bytes readable by a format-5 reader and does not preserve a
SealID after identity-bearing metadata is added.

Before normative conversion or implementation, the Operator must accept one
complete companion decision set containing:

1. a storage/migration ADR that fixes repository and typed-Blob versions,
   canonical bytes and limits, old-object readability, unchanged historical-ID
   addressability, migration or mixed-schema policy, dump/load closure, and the
   effect of changed Cause Link bytes on ADR 0028 change identity; and
2. a CLI/output ADR that fixes Candidate mutation grammar, exact-target
   selection, rejection atomicity, human rendering, versioned machine schemas,
   comparison, and `linklog` behavior.

The complete set must preserve every legacy `messages` member exactly and must
not infer namespace, schema, revision kind, rationale, Assessment, or other
structured meaning from message text. Existing historical Seals without
metadata must not become corrupt, unapproved, or semantically reclassified.

General metadata query language, query AST, indexes, query mutation, schema
registry, and validator runtime are not members of this decision set.

### Disposition of Proposed ADR 0017

ADR 0029 replaces Proposed ADR 0017 as the current generic Cause
Link metadata semantic direction. ADR 0017's format-4 Link shape, singular
message model, reserved virtual query namespace, SealGraphQL design, query
language, namespace assignment, suggested numeric limits, and implementation
scope are not adopted.

The 2026-09-10 acceptance follow-up marks ADR 0017 Rejected and links it to this
ADR. ADR 0017 was not rewritten while this replacement remained Proposed.

## Alternatives

### Add dedicated revision-kind and rationale fields

Rejected because each later Link-local vocabulary would require another core
field and because core could accidentally acquire semantics for progress,
correction, or supersession.

### Replace `messages` with structured metadata

Rejected because it breaks compatibility, forces structured authoring, and
would invite lossy interpretation of existing free-form text.

### Make Cause Links first-class Blobs

Rejected because the accepted requirement keeps Links embedded and Sealgraph has
no independently active Link set, publication authority, or replacement
lifecycle.

### Adopt ADR 0017 and SealGraphQL together

Rejected because query language is not required for the accepted metadata use
case and would couple semantic, storage, observation, cost, and public CLI
decisions that can be reviewed independently.

### Define a core revision-context namespace now

Deferred because the accepted requirement needs capability, not core ownership.
Assigning a normative namespace without selecting its validator and evolution
policy would create an external compatibility promise unnecessarily.

### Put exact storage and CLI bytes in this ADR

Rejected because accepted ADRs already separate graph semantics, typed-Blob
bytes, migration, and CLI schemas. Companion ADRs make each authority and
compatibility transition explicit.

## Consequences

- Callers may keep using `messages` only or add structured Link-local meaning
  through one generic mechanism.
- Revision explanations can name exact previous Seal generations without
  becoming graph edges or core supersession state.
- Unknown namespaces remain portable and inspectable but are not trusted or
  semantically validated by core.
- Metadata changes immutable Provenance and Seal identities.
- Old readers cannot silently accept the successor Link shape.
- A profile consumer must know and validate its namespace-specific schema
  independently of core canonical reading.
- At least two companion ADRs are required before implementation; this increases
  review work but prevents semantic, storage, migration, and CLI authority from
  collapsing into one oversized decision.
- General metadata query remains unavailable unless separately required and
  accepted later.

## Implementation notes

No implementation action is authorized while any required companion ADR is
unaccepted.

When the complete decision set is accepted under a separate implementation
instruction:

- keep domain semantics independent from CLI parsing and storage;
- implement bounded canonical values without floating point;
- add deterministic canonicalization and boundary fixtures;
- preserve one-target Candidate mutation and rejection atomicity;
- expose `messages` and metadata separately in human and machine inspection;
- update ADR 0028 change-preimage handling for the exact successor Cause Link;
- provide compatibility and migration fixtures for existing messages and IDs;
  and
- synchronize requirements, architecture, storage format, CLI, integrations,
  and planning documents in the authorized change set.

## Review

Author self-review on 2026-09-10 checked:

- the accepted requirement's ten acceptance criteria;
- internal separation of structural fields, opaque metadata, and profile meaning;
- precedence against Accepted ADRs 0023, 0025, 0027, and 0028;
- explicit non-adoption and future disposition of Proposed ADR 0017;
- compatibility boundaries for messages, old readers, migration, and SealIDs;
- non-assignment of a core revision-context namespace; and
- implementation gates and unauthorized downstream effects.

Independent ADR review returned `PASS` with no P0 through P3 findings, and the
Operator accepted the exact reviewed technical candidate on 2026-09-10. The
review answered:

1. Does this semantic slice decide enough to review independently without
   preempting storage/migration or CLI/output companions?
2. Does the external-profile choice satisfy the accepted revision-context
   capability without creating a hidden core schema promise?
3. Can every structural graph result remain independent of metadata meaning
   while identity and output bytes change?
4. Does the ADR preserve ADR 0028 Assessment identity and admission boundaries?
5. Is Proposed ADR 0017 disposed of without silently inheriting its query or
   format-4 assumptions?

## Evidence

- **E-LM-001:** the exact accepted requirement and acceptance record establish
  the owner-approved outcome, compatibility distinctions, open successor
  decisions, and downstream authority boundary.
- **E-LM-002:** Accepted ADR 0023 defines the three-field format-5 Cause Link,
  observer-scoped previous assertions, structural edge union, staleness,
  history, and admission.
- **E-LM-003:** Accepted ADR 0025 commits exact Cause Links through ProvenanceID
  and SealID and requires any metadata extension to use another accepted typed
  schema.
- **E-LM-004:** Accepted ADR 0027 owns format-5 Candidate authoring, inspection,
  comparison, and `linklog` schemas and requires successor schemas for changed
  meaning.
- **E-LM-005:** Accepted ADR 0028 owns upstream Assessment identity, evidence,
  adoption, disposition, and admission and rejects hiding AssessmentID in Link
  messages or attachments.
- **E-LM-006:** Proposed ADR 0017 and its proposal supply non-normative rationale
  for one namespaced metadata set and canonical values, but also carry obsolete
  Link and query assumptions that are not accepted here.
- **E-LM-007:** `docs/architecture.md` requires every new persisted field to have
  a storage-format change, deterministic fixtures, compatibility consideration,
  and an approved ADR.

Exact traceability is:

| Claim | Evidence | Required implementation action |
| --- | --- | --- |
| C-LM-001: preserve current fields and embedded Link identity | E-LM-001, E-LM-002, E-LM-003 | A-LM-001, A-LM-003 |
| C-LM-002: add one optional generic metadata collection | E-LM-001, E-LM-006 | A-LM-001, A-LM-003 |
| C-LM-003: keep namespaces opaque and canonical values bounded | E-LM-001, E-LM-003, E-LM-006, E-LM-007 | A-LM-001, A-LM-003, A-LM-005 |
| C-LM-004: keep graph semantics independent of metadata meaning | E-LM-001, E-LM-002 | A-LM-003, A-LM-005 |
| C-LM-005: support revision context without core graph state | E-LM-001, E-LM-002 | A-LM-002, A-LM-003 |
| C-LM-006: do not substitute for upstream Assessment | E-LM-001, E-LM-005 | A-LM-001, A-LM-002, A-LM-005 |
| C-LM-007: preserve explicit Candidate and publication boundaries | E-LM-001, E-LM-004 | A-LM-002, A-LM-003, A-LM-005 |
| C-LM-008: make compatibility and migration explicit | E-LM-001, E-LM-003, E-LM-004, E-LM-007 | A-LM-001, A-LM-005 |
| C-LM-009: separate metadata from ADR 0017 query scope | E-LM-001, E-LM-006 | A-LM-004 |
| C-LM-010: require one complete accepted companion set | E-LM-001, E-LM-003, E-LM-004, E-LM-007 | A-LM-001 through A-LM-005 |

Implementation action identities are:

- **A-LM-001:** under separate implementation authority, implement the accepted
  storage/migration companion.
- **A-LM-002:** under separate implementation authority, implement the accepted
  CLI/output companion.
- **A-LM-003:** implement domain, repository, inspection, and graph behavior.
- **A-LM-004:** dispose of Proposed ADR 0017 and keep query work separate.
- **A-LM-005:** add canonical, compatibility, migration, graph-independence,
  rejection-atomicity, and output fixtures and synchronize normative documents.

## Follow-ups

- Draft the storage/migration and CLI/output companion ADRs without implementing
  them.
- Keep Rejected ADR 0017 linked to ADR 0029 as its accepted replacement; do not
  inherit ADR 0017's query or format-4 decisions.
- Do not begin normative conversion or implementation until the complete
  companion set is independently reviewed, accepted, and separately authorized.
