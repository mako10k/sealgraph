# Upstream impact assessment decision record — 2026-08-31

Status: operator-direction evidence for Accepted ADR 0028. This record did not
itself accept the ADR and does not authorize implementation, migration,
publication, or external writes. The later exact owner acceptance is recorded
in
[`upstream-impact-assessment-acceptance-2026-08-31.md`](upstream-impact-assessment-acceptance-2026-08-31.md#owner-acceptance-receipt).

## Operator directions recorded

- **D-UR-001:** Keep Cause Links committed by the dependent observer Seal's
  Provenance. Do not introduce reverse Cause edges merely to support upstream
  impact.
- **D-UR-002:** Derive upstream and downstream impact from the same dependency
  DAG. Keep semantic upstream-review state separate from structural stale.
- **D-UR-003:** Make an upstream Assessment an independent immutable entity and
  let the dependent Seal commit to its exact identity through a non-Cause
  reference.
- **D-UR-004:** Leave fully symmetric independently active relation graphs to
  RefGraph; Sealgraph remains the operational dependency and evidence model.
- **D-UR-005:** During the first CLI transition, omission of impact direction
  retains the current downstream behavior but emits a warning asking callers to
  specify `--downstream` or `--upstream` explicitly.
- **D-UR-006:** The motivating case is a detailed-design change that retains the
  same basic-design HEAD. A formal reseal proves which exact HEAD was selected
  but does not prove that basic/detail semantic consistency was reviewed; the
  unchanged selection alone must not discharge upstream review.

## Authority boundary

These directions authorized preparation and review of an exact Proposed ADR.
They did not accept its exact bytes or authorize a persisted-format change,
runtime implementation, migration, commit, push, release, or deployment. The
fresh exact-candidate review and explicit owner decision were subsequently
completed and are preserved in the linked Acceptance Record; those later steps
do not broaden this record's authority boundary.
