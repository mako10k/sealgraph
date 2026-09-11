# Focused ADR rereview: ADR 0031 format-6 Link metadata CLI and output — 2026-09-10

## Review identity

- Subject: `docs/adr/0031-format6-link-metadata-cli-and-output.md`
- Stage: `Proposed; revised on 2026-09-10`
- Expected SHA-256:
  `82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47`
- Digest before rereview: matched
- Digest after rereview: matched
- Candidate unchanged during review: yes
- Reviewer role: one independent, read-only focused ADR reviewer
- Verdict: `PASS`

This rereview does not change or accept ADR 0031, authorize normative conversion
or implementation, perform migration, commit, push, release, deploy, or
authorize another external write.

## Findings

No P0, P1, P2, or P3 findings in the focused scope.

No new defect-producing source-cause phase was established.

## Prior P1 disposition

`RESOLVED`.

The revision adds complete entry-level identities and schema generations for
both owning endpoints:

- `newer_seal_id`, `newer_seal_schema`;
- `newer_provenance_id`, `newer_provenance_schema`;
- `previous_seal_id`, `previous_seal_schema`; and
- `previous_provenance_id`, `previous_provenance_schema`.

ADR 0031 lines 232–240 fixes the ownership mapping explicitly: `before` belongs
to the previous endpoint, `after` belongs to the newer endpoint, and a null Link
side still retains its endpoint generation.

This is sufficient to distinguish a Provenance v1 `metadata: []` projection
from Provenance v2 stored empty metadata for additions, removals, and two-sided
changes. It satisfies ADR 0030 lines 191–208 without repeating generation fields
in every changed Link or namespace. The prior defect-producing phase was CLI
machine-schema design; the revised entry schema corrects that omission directly.

## Focused review questions

### 1. Endpoint classification

**Result: Pass.** Every `before` and `after` has an unambiguous owning endpoint
and a valid Seal/Provenance generation pairing.

### 2. Orientation and null cases

**Result: Pass.** `before = previous` and `after = newer` is explicit. Additions,
removals, v5-to-v6, and v6-to-v5 orientations remain classifiable.

### 3. Member ordering

**Result: Pass.** `linklog/v3` retains ADR 0027's top-level order while
explicitly replacing only the entry member order. Existing entry sorting by
`(minimum_newer_depth, newer SealID, previous SealID)` is unaffected.

### 4. Human output and fixtures

**Result: Pass.** Human entries display both endpoint generations, and the
implementation notes require mixed-generation projection-versus-storage
fixtures.

### 5. New contradictions

**Result: Pass.** None were found. The endpoint fields remain contained in the
already-successor `linklog/v3` contract and do not alter graph membership,
filtering, entry ordering, or Cause Link comparison meaning.

## Unresolved questions

None in the focused rereview scope.

## Coverage

The focused rereview covered:

- revised ADR 0031 status, comparison and `linklog` records, schema matrix,
  human inspection, alternatives, implementation fixtures, review, and
  follow-ups;
- accepted ADR 0030 historical typed-object compatibility, lines 191–208;
- accepted ADR 0027 `linklog` topology and ordering, lines 455–532;
- current `internal/repository/history.go` ownership and orientation; and
- current `internal/cli/inspection_json.go` format-5 topology.

Previously passed, unchanged areas were not reopened because the revision did
not introduce a concrete contradiction in their assumptions.

## Reviewer conclusion

The exact revised candidate receives `PASS` with no P0 through P3 findings. The
prior P1 is resolved. ADR 0031 remains `Proposed` pending the Operator's separate
`ACCEPT` or `REVISE` decision.
