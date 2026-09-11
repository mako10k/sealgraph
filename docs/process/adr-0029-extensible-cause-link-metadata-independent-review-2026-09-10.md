# Independent ADR review: ADR 0029 extensible Cause Link metadata — 2026-09-10

## Review identity

- Subject: `docs/adr/0029-extensible-cause-link-metadata.md`
- Stage: `Proposed`
- Expected SHA-256:
  `3a4886671a44c7b34f8a09ce14da08b1f9445fd9cd97140b4cdf66e36ec8fd8c`
- Digest before review: matched
- Digest after review: matched
- Reviewer role: one independent, read-only ADR reviewer
- Verdict: `PASS`

This review does not accept ADR 0029, change its status, authorize companion
decisions, or authorize implementation, normative conversion, migration,
commit, push, release, deployment, or another external write.

## Findings

No P0, P1, P2, or P3 findings.

Source-cause phase is not applicable because no defect finding was established.

## Review questions

### 1. Semantic and responsibility slice

**Result: Pass.** ADR 0029 fixes the logical record, namespace and schema
ownership, canonical value categories, graph-independence rule, Candidate
mutation properties, and inspection obligations. It expressly reserves exact
bytes, numeric limits, migration, CLI grammar, and machine schemas for accepted
companion ADRs (ADR 0029 lines 10–14, 43–113, and 158–198).

This matches the accepted requirement's logical model and retained successor
decisions (requirement revision 2 lines 85–141, 195–213, and 240–253). The
semantic slice is independently reviewable and does not preempt the companion
decisions.

### 2. External revision-context profile

**Result: Pass.** The accepted requirement explicitly leaves maintained-profile
versus external-example ownership to an owner decision while requiring the
generic mechanism to support either (requirement revision 2 lines 240–253).

ADR 0029 makes one permitted choice: an external, versioned profile example
with no reserved persisted core namespace, no core schema-conformance promise,
and no graph semantics (ADR 0029 lines 72–87 and 128–156). It supplies the
accepted capability without creating hidden core ownership.

### 3. Structural graph and identity boundary

**Result: Pass.** The accepted structural model derives Cause and revision
relations from exact targets and previous-revision assertions (ADR 0023 lines
43–70 and 99–142). Exact Provenance content is transitively committed by SealID
(ADR 0025 lines 91–200).

ADR 0029 preserves those structural inputs while correctly stating that
metadata changes ProvenanceID, SealID, assertion-source identity, comparisons,
and exact output bytes (ADR 0029 lines 115–126). Semantic non-interpretation is
not confused with byte invariance.

### 4. ADR 0028 Assessment boundary

**Result: Pass.** ADR 0028 requires a separate Assessment Blob, AssessmentID
adoption, exact change binding, evidence, disposition, and admission state. It
explicitly rejects simulating these through messages, attachments, or generic
Cause Link metadata (ADR 0028 lines 55–104, 165–207, and 259–368).

ADR 0029 excludes each of those Assessment properties and requires the storage
companion to account for changed Cause Link bytes in ADR 0028 change identity
(ADR 0029 lines 152–156 and 181–190).

### 5. Disposition of Proposed ADR 0017

**Result: Pass.** ADR 0017 remains Proposed and couples generic metadata to
format-4 Link assumptions, a reserved virtual namespace, and SealGraphQL (ADR
0017 lines 3–42; linked proposal lines 60–178 and 224–240).

ADR 0029 adopts only the separable generic-metadata rationale, expressly
excludes query language and format-4 assumptions, and postpones ADR 0017's
status change until ADR 0029 is accepted (ADR 0029 lines 197–210). Marking the
never-accepted proposal `Rejected` after replacement acceptance is coherent.

## Unresolved questions

The following are mandatory companion decisions, not defects in this semantic
ADR:

- exact repository and typed-Blob versions;
- canonical encoding, integer range, and resource bounds;
- old-object readability and mixed-schema versus explicit migration policy;
- dump/load and historical-ID handling;
- exact Candidate mutation grammar and target selection;
- human and versioned machine output schemas;
- comparison and `linklog` successor behavior; and
- exact integration with ADR 0028 change-preimage bytes.

General query language, query AST, indexes, registries, and validator runtime
remain optional and outside the accepted contract.

## Coverage

The review covered:

- the exact accepted revision-2 requirement and its acceptance record;
- ADR 0029's definitions, alternatives, consequences, gates, and
  Claim/Evidence/Action traceability;
- accepted ADRs 0023, 0025, 0027, and 0028;
- Proposed ADR 0017 and its linked detailed proposal; and
- current normative requirements, architecture, storage-format, and CLI
  boundaries.

The current normative format-5 documents still specify the three-field Cause
Link, as expected. ADR 0029 acknowledges that incompatibility and blocks
normative conversion and implementation until the accepted successor decision
set exists.

## Reviewer conclusion

The exact candidate receives `PASS` with no P0–P3 findings. The candidate
remains `Proposed` pending the Operator's separate `ACCEPT` or `REVISE`
decision.
