# Candidate explicit Seal comparison requirement R4

Status: Requirement Candidate R4 — Awaiting First Owner Review

Date: 2026-09-11

## 1. Objective and source provenance

An operator reviewing mutable Candidate state must be able to compare its
complete prospective Seal projection with one explicitly selected immutable
Seal instead of being limited to the Candidate's publication baseline.

The owner selected this explicit command form:

```text
sealgraph candidate compare REF --against SELECTOR
```

R1 and R2 failed independent review and were revised. R3 at SHA-256
`ef97bf8ea62a9c50500c91700c021f0ae5a061048a8362e87d764d36d355b16a`
passed independent review, but was not accepted. Before its second-owner
decision, accepted format-6 work assigned `sealgraph/candidate-compare/v3` to
the existing publication-baseline comparison in format-6 repositories. R3's
assignment of the same identifier to explicit comparison therefore became
stale. The owner authorized a suitable revision on 2026-09-11.

R4 preserves R3's command, direction, fields, coherent-observation discipline,
and scope. It assigns the explicit mode `/v4`, records the accepted format-6
authority, and restarts the requirement lifecycle at step 1.

## 2. Prior authority disposition

This candidate changes only the publication-baseline-only restriction in ADR
0023, `docs/requirements.md`, `docs/architecture.md`, and `docs/cli.md` for the
new `--against` mode.

It preserves:

- top-level `sealgraph compare LEFT RIGHT` as two explicit immutable Seals;
- no implicit predecessor inference;
- ADR 0023's complete coherent observation and revalidation;
- `candidate compare REF` as publication-baseline comparison;
- format-5 default JSON `sealgraph/candidate-compare/v2` from ADR 0027;
- format-6 default JSON `sealgraph/candidate-compare/v3` from accepted ADR 0031;
- `source compare` and every mutation boundary; and
- format-6 Link metadata, typed-generation, and comparison meanings.

If accepted, a new ADR must supersede only the explicit-comparison restriction
and allocate `sealgraph/candidate-compare/v4`. No release or milestone is
assigned.

## 3. In-scope behavior

```text
sealgraph candidate compare REF --against SELECTOR [--format human|json]
```

`REF` selects one existing Candidate. `SELECTOR` uses the accepted immutable
Seal selector grammar and resolves to exactly one Seal permitted by the current
repository format.

Comparison direction is selected Seal as `before` and the Candidate's complete
prospective Seal projection as `after`. It covers Material ID, Provenance ID,
content Blob ID, attachments, root, draft, and complete Cause Links including
format-6 metadata when present.

The result labels the target `EXPLICIT_SEAL`. The exact selector spelling and
complete resolved Seal view are included. Current REF head and expected-head
state remain separate publication-concurrency context and are not replaced or
reinterpreted by the selected Seal.

The operation captures `O`, the complete sorted map from every REF name to its
exact canonical manifest bytes, derives and validates the required closure,
buffers the result, then recaptures `O` and requires byte equality including
names, presence, absence, HEADs, and tag arrays before successful output.

Repository-wide `@SEAL_TOKEN` prefix resolution additionally captures and
recaptures `I_O`, the sorted complete valid loose-object identity set, and
requires exact equality. Other selector forms do not scan `I_O` merely for this
comparison. Any required observation change fails without authoritative output.

Default mode remains unchanged and format-aware:

```text
format 5: candidate compare REF -> sealgraph/candidate-compare/v2
format 6: candidate compare REF -> sealgraph/candidate-compare/v3
```

Explicit mode emits `sealgraph/candidate-compare/v4` in both formats. V4 has:

```text
schema, ref, current_ref_head, expected_head_state,
comparison_target, prospective, changes

comparison_target:
  label = EXPLICIT_SEAL
  selector = exact user-supplied spelling
  seal = complete generation-aware seal_view
```

V4 uses one fixed generation-aware record shape. Format-5 records identify
Seal v5, Provenance v1, and Candidate v5 and expose empty metadata arrays;
format-6 records preserve exact v5/v6 generations and complete metadata.
`prospective` and `changes` use the accepted format-6 generation-aware and
per-namespace comparison shapes. Human output names the explicit target and
retains bounded binary-safe values and abbreviated IDs.

## 4. Out of scope

This candidate adds no Candidate-to-Candidate or workfile-to-arbitrary-Seal
comparison, generalized operands, predecessor inference, textual patch, raw
content output, automatic conflict resolution, mutation, Git discovery, or
Git-derived semantics. These remain optional future candidates.

## 5. Assumptions and unknowns

The explicit comparison is assumed to be the smallest useful increment. A
single generation-aware v4 is assumed safer than different explicit schemas by
repository format. Future operand demand and a generalized syntax remain
unknown and are not acceptance blockers.

## 6. Acceptance criteria

1. ADR, requirements, architecture, CLI, help, and schema documentation define
   explicit mode while preserving both format-specific default schemas.
2. `--against` resolves every permitted Seal selector and rejects missing,
   invalid, or ambiguous selections without comparison output.
3. Selected Seal is complete `before`; prospective Candidate is complete
   `after`; publication concurrency remains separate.
4. Explicit comparison infers no ancestry or publication-baseline meaning.
5. Explicit mode emits only v4 with one stable generation-aware shape; default
   format 5 remains v2 and default format 6 remains v3.
6. Complete `O` revalidation and conditional complete `I_O` revalidation match
   ADR 0023 exactly.
7. Human and JSON tests cover historical and current targets, Candidate
   divergence, both repository formats, invalid and ambiguous selectors, and
   both observation-mismatch paths.
8. Existing comparison tests and the full repository validation suite pass.

## 7. Normative and non-normative inputs

Sections 1 through 6 become normative only if this exact candidate is accepted.
Accepted ADRs 0023, 0027, and 0031 plus current requirements, architecture,
storage-format, and CLI documents are prior authority. Implementation and tests
are non-normative feasibility/current-state evidence.

## 8. Proposed independent-review input

The reviewer must receive this exact path and digest and answer:

1. Does R4 disposition every v2/v3 authority without reusing a schema identity?
2. Is v4 one fixed shape across formats with unambiguous generation/metadata?
3. Are direction, fields, publication context, `O`, and conditional `I_O`
   preserved from R3 and ADR 0023?
4. Is supersession limited to explicit comparison without changing defaults?
5. Did any optional future comparison become an acceptance blocker?

Findings must be classified as contradiction, evidence gap or unknown,
optional/future, or out of scope. Review does not edit or accept the candidate.

## 9. Self-review and differences from R3

Source history, authority, scope, assumptions, unknowns, compatibility,
namespace ownership, acceptance criteria, and review questions are explicit.
R4 changes only: accepted format-6 authority is added; default mode is stated as
v2 in format 5 and v3 in format 6; explicit mode moves from conflicting v3 to
v4 with one generation-aware shape. All other R3 behavior is preserved.
