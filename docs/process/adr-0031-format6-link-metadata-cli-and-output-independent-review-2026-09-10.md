# Independent ADR review: ADR 0031 format-6 Link metadata CLI and output — 2026-09-10

## Review identity

- Subject: `docs/adr/0031-format6-link-metadata-cli-and-output.md`
- Stage: `Proposed`
- Expected SHA-256:
  `f3082d68aad0638e3af5c28f5199a4644490af8b3d12a846281d1091185eff9f`
- Digest before review: matched
- Digest after review: matched
- Reviewer role: one independent, read-only repository-wide ADR reviewer
- Verdict: `FAIL`

This review does not change the candidate, accept ADR 0031, authorize normative
conversion or implementation, perform migration, commit, push, release, deploy,
or authorize another external write.

## Findings

### P1 — `linklog/v3` cannot identify projected format-5 metadata

**Acceptance blocking: yes.**

#### Claim

The proposed `linklog/v3` record cannot distinguish a historical format-5 Cause
Link projected as `metadata: []` from a format-6 Cause Link that actually stores
an empty metadata array.

#### Evidence

- Accepted ADR 0030 lines 204–208 require a historical v5 Link exposed through
  a successor inspection schema to be explicitly identified as a historical
  projection, rather than as bytes claimed to exist in Provenance v1.
- ADR 0031 lines 177–181 repeats that requirement and says enclosing schema
  fields identify the projection.
- ADR 0031 lines 193–199 defines `cause_link_change` without the schema
  generation of the owning before/after Seal or Provenance, and lines 217–219
  makes `linklog` use that same record.
- ADR 0031 lines 245–247 retains ADR 0027's existing top-level and entry shape
  except for substituted shared records.
- Accepted ADR 0027 lines 465–470 defines a `linklog_entry` with newer and
  previous Seal IDs but no newer/previous Seal or Provenance schema fields.
- Current implementation confirms the ownership relation: `history.go` lines
  180–197 compares Cause Links owned by the newer and previous endpoint Seals,
  while `inspection_json.go` lines 540–575 emits only their IDs around the
  change records.

#### Reasoning

The `linklog/v3` top-level schema identifies the output-document version, not
the storage generation of either compared endpoint. Supporting assertions
identify assertion observers, and the target revision observation describes
assertions targeting the newer Seal; neither reliably identifies the
Provenance generation that owns each compared Cause Link.

Consequently, a consumer seeing `before.metadata: []` or
`after.metadata: []` cannot tell whether the value is stored format-6 data or a
logical format-5 projection. This violates ADR 0030's explicit projection
contract and ADR 0031's own statement that enclosing schema fields provide that
distinction.

#### Causal classification and corrective boundary

- **Root cause:** the CLI machine-schema design reuses `cause_link_change`
  outside an enclosing typed Seal or Candidate view without carrying the
  endpoint generation context on which projection provenance depends.
- **Contributing cause:** the shared change record was treated as
  self-sufficient even though the meaning of an empty metadata array is
  contextual to the owning Provenance generation.
- **Escape or detection cause:** author self-review checked schema-version bumps
  and historical projection separately, but did not trace projection provenance
  through the exact `linklog_entry` topology. Missing tests did not create the
  defect.
- **Corrective action:** revise the candidate so each `linklog/v3` before/after
  side is unambiguously associated with its exact endpoint Seal or Provenance
  generation, and specify the resulting exact member order.
- **Recurrence-prevention action:** require mixed v5-to-v6 and v6-to-v5
  `linklog` fixtures in which `[]` is projected on one side and stored on the
  other, plus a topology check for every projected record.

The command-line LLMThink causal audit completed with zero fatal, error, or
warning findings. Uncertainty is low.

#### Coordinator disposition after `REVISE`

`Current correction required` was adopted. The revised candidate adds complete
newer and previous Seal/Provenance identities and schema generations once per
`linklog/v3` entry, defines `before` as owned by previous and `after` as owned by
newer, and requires mixed-generation projection fixtures. This disposition does
not change the original finding or verdict. The revised candidate SHA-256 is
`82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47` and requires
focused rereview.

## Reviewed areas without additional findings

### Legacy authoring continuity

**Result: Pass.** Preserving metadata across existing format-6 `add` and `link`
syntax is a coherent successor refinement. It retains structural
previous/message replacement while preventing omission from deleting a newly
introduced extension area. Explicit `unlink` and root clearing still remove the
named whole-Link scope.

### Metadata mutation

**Result: Pass.** `link-metadata set` and `remove` provide exact one-target,
one-namespace Candidate mutation, complete validation before expected-old
replacement, and rejection atomicity. They neither publish nor migrate.

### Schema matrix and comparison

**Result: Pass except for the P1 `linklog` defect.** The schema matrix otherwise
covers changed shared records. Retaining `status/v3`, `stale/v2`, and narrow
protocols is consistent with their unchanged meanings. Candidate and immutable
comparison records carry enclosing Candidate or Seal schema fields, so they do
not share the `linklog` ambiguity.

### Human output

**Result: Pass with an implementation limitation.** A terminal-safety violation
is not established. The candidate requires canonical JSON escaping, never emits
control characters literally, bounds the complete metadata array to 65,536
canonical bytes through ADR 0030, and retains ADR 0022 width-safe behavior.
Implementation must still demonstrate width-safe rendering at the maximum value
size; that verification need is not a second design finding.

### Migration and excluded scope

**Result: Pass.** Migration syntax, config-only transaction, receipt, and
no-automatic-retry behavior are consistent with ADR 0030. Query language,
registry/validation, and ADR 0028 Assessment commands remain correctly excluded.
Downstream `impact/v3` remains distinct from ADR 0028's separate
`upstream-impact/v1` contract.

## Unresolved question

The corrective design must choose whether endpoint generation is carried once
per `linklog_entry`, independently on each `cause_link_change` side, or through
another self-contained typed wrapper. That is an ADR revision decision, not a
reviewer edit.

## Coverage

The review covered:

- ADR 0031's definitions, alternatives, consequences, review claims, and
  Claim/Evidence/Action traceability;
- current requirements, architecture, storage-format, and CLI documents;
- accepted ADRs 0022, 0023, 0027, 0028, 0029, and 0030;
- the accepted revision-2 requirement and its acceptance record; and
- current Candidate lifecycle, expected-old persistence, history comparison,
  machine-output, and human-output implementation for current-fit evidence.

## Reviewer conclusion

The exact candidate receives `FAIL` with one acceptance-blocking P1 finding and
no P0, P2, or P3 findings. ADR 0031 remains `Proposed`. The appropriate next
lifecycle action is `REVISE`, followed by a new review of the revised exact
candidate.
