# Cause Link extensible metadata requirement candidate — 2026-09-10

- Status: Step 1 requirement candidate; self-review complete; first owner review pending
- Candidate revision: 1
- Decision owner: Operator
- Authority: candidate only; not normative and not implementation authority
- Proposed review input:
  [`cause-link-extensible-metadata-requirement-review-input-2026-09-10.md`](cause-link-extensible-metadata-requirement-review-input-2026-09-10.md)

## Purpose

Preserve the existing Cause Link contract while allowing an observer to add
structured, versioned meaning when free-form `messages` are insufficient. The
motivating profile explains why a target is asserted as a revision of one or
more exact previous Seals, including cases such as progress, correction of a
Cause inconsistency, or reconciliation. The generic extension mechanism must
remain usable for later Link-local vocabularies without adding a new core field
for every vocabulary.

## Source directions

- **D-CLM-001:** Keep `messages` for compatibility.
- **D-CLM-002:** Structured revision context is optional; callers may continue
  to use only `messages`.
- **D-CLM-003:** Add one extensible area for future structured Link-local data.
- **D-CLM-004:** Generalize the revision-context structure through that one
  extension mechanism instead of adding both dedicated revision-reason fields
  and a second generic extension path.
- **D-CLM-005:** A Cause Link remains a record inside an immutable Provenance
  Blob. It does not become a separately addressed Blob and gains no CauseLinkID.

These directions are the current owner-originated requirement source. They do
not by themselves accept this candidate's exact wording or authorize an ADR,
format change, implementation, migration, commit, push, release, or deployment.

## Prior-authority disposition

- Accepted ADR 0023 remains authoritative for observer-scoped revision
  assertions, structural edge union, staleness, history, and admission. Metadata
  does not create or remove a Cause or Revision edge.
- Accepted ADR 0025 remains authoritative for the Universal Blob layer and the
  Seal-to-Material-and-Provenance join. Cause Links and their metadata remain
  Provenance content and are committed transitively by SealID. A successor
  persisted format is required before metadata can be stored.
- Accepted ADR 0027 remains authoritative for format-5 authoring and inspection.
  A later accepted CLI/schema successor must define metadata mutation and output.
- Accepted ADR 0028 remains authoritative for upstream-impact Assessments.
  Generic Link metadata must not simulate an AssessmentID, evidence-bearing
  disposition, adoption reference, or seal-admission result.
- Proposed ADR 0017 and
  [`../proposals/link-metadata-and-sealgraphql.md`](../proposals/link-metadata-and-sealgraphql.md)
  are non-normative design inputs. This candidate reuses only the separable
  namespaced `namespace` / `schema` / `value` concept. It does not accept their
  old Link shape, old revision model, query language, namespace assignment,
  resource limits, or implementation scope. A later ADR must explicitly revise,
  replace, withdraw, or otherwise dispose of ADR 0017.

## In scope

- One optional logical metadata set on each exact Cause Link.
- Namespaced and version-identifiable structured values.
- Deterministic, bounded, identity-bearing canonical representation.
- A versioned revision-context profile expressed through generic metadata.
- Candidate authoring and human/machine inspection requirements.
- Compatibility and migration requirements for existing `messages` and Seals.

## Out of scope

- Making Cause Links separately addressed Blobs or independently active graph
  entities.
- Changing Cause direction, revision-edge derivation, stale propagation,
  frontier membership, or normal Cause-closure admission.
- Replacing ADR 0028 upstream Assessments or satisfying their admission gate.
- Treating progress, correction, reconciliation, trust, approval, truth, or
  supersession as a core graph state.
- A query language, schema registry, network schema retrieval, signatures,
  trusted actor/time, or secret storage.
- Exact successor format numbers, CLI spelling, migration commands, release
  scope, or implementation sequencing.

## Required logical model

The successor contract must preserve the three existing Cause Link fields and
add only one generic metadata collection:

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

The logical metadata collection may be empty. The successor storage decision
must choose one exact canonical representation for the empty set and must not
give omission an environment-dependent meaning.

### Existing fields

`target_seal`, `previous_revision_seal_of_target_seal`, and `messages` retain
their format-5 meanings. In particular:

- `messages` remains an identity-bearing, sorted, duplicate-free string set;
- an empty `messages` array remains valid;
- existing message text is not reclassified or parsed as structured metadata;
  and
- revision edges continue to be derived exclusively from the exact previous
  SealID array, never from metadata.

### Generic metadata

Each metadata entry is owned by one exact, case-sensitive namespace. One Cause
Link may contain at most one entry for one namespace. `schema` is null or one
immutable version-specific identifier selected by the namespace owner. Core
does not dereference the identifier or infer that the value conforms to it.

`canonical_value` must support a bounded deterministic subset sufficient for
null, booleans, integers, strings, arrays, and string-keyed objects. The storage
decision must fix ordering, encoding, nesting, node-count, string-size, entry-
count, and total-byte limits. Floating-point identity, executable values,
environment expansion, and dynamic selectors are prohibited.

Metadata is immutable Provenance content. Changing its namespace, schema, or
value produces a different ProvenanceID and therefore a different SealID.
Metadata creates no additional graph edge and cannot weaken validation of the
three structural Cause Link fields.

Unknown but structurally valid namespaces remain readable and inspectable. Core
must preserve their exact canonical values and must not assign domain meaning,
truth, approval, or authority to them.

## Revision-context profile

The extension mechanism must be capable of representing a versioned profile
with the following logical information. This is an illustrative shape, not an
accepted namespace, schema identifier, or canonical byte contract:

```json
{
  "namespace": "example.revision-context",
  "schema": "example.revision-context/v1",
  "value": {
    "revisions": [
      {
        "previous_seal": "<full-seal-id>",
        "kind": "correction",
        "rationales": [
          "The previous revision used an inconsistent Cause."
        ]
      }
    ]
  }
}
```

The profile binds each explanation to one exact previous Seal rather than to
the Cause Link's complete previous set. A conforming profile may require every
`previous_seal` to occur in the structural previous array and may define values
such as `progress`, `correction`, `reconciliation`, or an extensible equivalent.
Those values remain namespace-owner claims and do not invalidate, supersede, or
prefer any Seal in core graph semantics.

This profile explains an observer's revision assertion. It is not seal-operation
metadata and does not record a trusted actor or time. It also is not ADR 0028's
upstream Assessment: it has no AssessmentID, change binding, evidence adoption,
compatibility disposition, or admission effect. When either of those stronger
claims is required, its separately accepted mechanism remains required.

## Authoring and inspection requirements

- Metadata mutation operates on one exact target Link in one Candidate.
- Adding, replacing, and removing one namespace are explicit operations.
- A metadata-only edit preserves the target, previous array, `messages`, and all
  other namespaces unless the same authorized Candidate operation explicitly
  changes them.
- Rejected metadata input leaves the Candidate unchanged.
- Human inspection identifies namespace and schema and renders arbitrary values
  with escaping and output bounds.
- Versioned machine output returns complete canonical values without truncation.
- Cause Link comparison and `linklog` distinguish `messages` changes from
  metadata namespace/value changes.
- Metadata mutation never publishes a Seal or moves a REF implicitly.

## Compatibility and migration requirements

Keeping `messages` provides semantic and authoring continuity; it does not make
new canonical bytes readable by an old format-5 reader and does not preserve a
SealID after identity-bearing metadata is added.

A successor decision must therefore state separately:

1. which old typed Blob schemas remain readable;
2. whether repositories require an explicit one-way format migration;
3. how unchanged existing Seals and their IDs remain addressable;
4. how an edited old Cause Link becomes a new-format Provenance and Seal; and
5. how dump/load and migration preserve object closure and exact metadata bytes.

Migration must preserve every existing `messages` member exactly. It must not
invent a namespace, schema, revision kind, rationale, upstream Assessment, or
other structured interpretation from legacy message text. Absence of metadata
must not make an existing historical Seal corrupt, unapproved, or semantically
reclassified.

## Acceptance criteria

This requirement is satisfied only when a later reviewed design demonstrates:

1. Cause Links remain Provenance-embedded records without CauseLinkID.
2. The three format-5 fields retain their meanings and `messages` remains usable
   without metadata.
3. One generic namespaced metadata mechanism supports the revision-context
   example without a dedicated second extension path.
4. Metadata is identity-bearing, deterministic, bounded, and safe to inspect.
5. Structural Cause/Revision behavior is byte-for-byte independent of metadata
   meaning and cannot be weakened by an unknown namespace.
6. Revision-context data is distinguishable from ADR 0028 Assessment evidence
   and cannot satisfy its admission states.
7. Candidate mutations are explicit, isolated by namespace, and do not publish.
8. Human and versioned machine inspection expose `messages` and metadata
   separately.
9. Compatibility claims distinguish field continuity, old-object readability,
   old-reader support, migration, and SealID preservation.
10. Migration preserves legacy messages without inferred structured meaning.

## Assumptions and unresolved decisions

- The final namespace owner and immutable revision-context schema identifier are
  not yet assigned.
- Exact canonical value limits are not yet selected.
- The successor repository, Seal, Provenance, Candidate, dump/load, and output
  schema versions are not yet selected.
- Mixed historical-schema reading versus mandatory repository conversion needs
  an explicit compatibility decision.
- Exact CLI syntax and whether a general metadata query is included in the first
  delivery remain open; neither is implied by this requirement candidate.
- Whether Sealgraph ships a maintained revision-context profile or documents an
  external example remains an owner decision. The generic mechanism must support
  either choice.

## Step 1 self-review

- **Source provenance:** D-CLM-001 through D-CLM-005 above are owner-originated;
  the existing ADRs and proposal are comparison and constraint sources.
- **Prior authority:** Accepted ADRs 0023, 0025, 0027, and 0028 are preserved
  except that any persisted implementation requires a separately accepted
  successor storage/CLI decision. Proposed ADR 0017 is not silently accepted.
- **Compatibility:** The candidate expressly separates `messages` continuity
  from old-reader and SealID compatibility.
- **Namespace:** No real namespace or schema identifier is assigned by the
  illustrative example.
- **Optional/future work:** Query language, validators, and built-in profile
  ownership remain non-blocking later decisions.
- **Unknowns:** The unresolved decisions above are material review questions and
  have not been filled by implementation assumptions.
- **Later effects:** ADR creation or revision, normative conversion,
  implementation, migration, commit, push, release, and deployment remain
  unauthorized by this candidate.
