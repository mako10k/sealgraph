# Candidate explicit Seal comparison requirement R2

Status: Requirement Candidate R2 — Awaiting First Owner Review

Date: 2026-09-10

## 1. Objective and source provenance

An operator reviewing mutable Candidate state must be able to compare its
complete prospective Seal projection with one explicitly selected immutable
Seal, rather than being limited to the Candidate's recorded publication
baseline.

This candidate comes from the owner's 2026-09-10 report that Seal-to-Seal-only
comparison is insufficient, followed by the explicit instruction to proceed
with the proposed first increment:

```text
sealgraph candidate compare REF --against SELECTOR
```

R1 at SHA-256
`75680973023dd7b72140ef1b2d5a9bd5a80aad3dee6d056aa3fe701d43abbdd3`
received a completed independent-review verdict of `FAIL`. The review found
that R1's “relevant observed heads” wording narrowed ADR 0023's complete
snapshot discipline. The owner selected `REVISE`. R2 changes that observation
and revalidation wording and its corresponding acceptance criterion; it does
not expand the requested comparison capability.

The conversation establishes the product need and authorizes preparation of
this candidate. Acceptance remains bound to the exact reviewed bytes under the
requirement-authority lifecycle.

## 2. Prior authority disposition

This candidate changes only the following accepted restrictions:

- ADR 0023 currently requires `candidate compare REF` to compare only with
  `candidate.expected_ref_head`.
- `docs/requirements.md` and `docs/cli.md` carry the same publication-baseline
  restriction.
- ADR 0027 defines `sealgraph/candidate-compare/v2` only for that restricted
  meaning.

If accepted, a new ADR must explicitly supersede those restrictions for the
new `--against` mode. It must preserve all other relevant authority, including:

- top-level `sealgraph compare LEFT RIGHT` remains an immutable Seal-to-Seal
  comparison with exactly two explicit selectors;
- the no-implicit-predecessor decision remains unchanged;
- ADR 0023's complete coherent-observation and snapshot-revalidation discipline
  remains unchanged;
- `candidate compare REF` without `--against` retains its exact existing
  publication-baseline semantics and `sealgraph/candidate-compare/v2` output;
- `source compare` remains unchanged;
- comparison remains read-only and does not seal, rebase, relink, repair, or
  mutate Candidate, REF, source, or immutable object state.

No release version, milestone, or external package version is assigned by this
candidate.

## 3. In-scope behavior

The CLI adds the optional form:

```text
sealgraph candidate compare REF --against SELECTOR [--format human|json]
```

`REF` selects exactly one existing Candidate. `SELECTOR` uses the complete
accepted immutable Seal selector grammar and must resolve to exactly one Seal.

The comparison direction is:

- `before`: the explicitly selected immutable Seal; and
- `after`: the Candidate's complete prospective Seal projection.

The comparison must cover the same identity-bearing fields as existing
Candidate and immutable comparison:

- Material ID;
- Provenance ID;
- content Blob ID;
- complete attachment array;
- root;
- draft; and
- complete Cause Link array.

The output must identify the selected Seal as `EXPLICIT_SEAL`, not as a parent,
previous revision, or publication baseline. It must retain the Candidate's
current REF head and expected-head state as separate publication-concurrency
context. The explicit selected Seal does not replace, update, or reinterpret
`candidate.expected_ref_head`.

Selector resolution, Candidate inspection, prospective projection, current
REF-head observation, and result production must follow ADR 0023's complete
coherent-observation and snapshot discipline. The operation must capture `O`,
the complete sorted map of every current REF name to its exact canonical
manifest bytes, and must derive and validate the complete required closure.
Immediately before emitting successful output, it must recapture the complete
manifest map and require byte-for-byte equality, including REF names,
presence, absence, exact HEADs, and complete tag arrays.

When `SELECTOR` uses repository-wide `@SEAL_TOKEN` prefix resolution, the
operation must additionally capture `I_O`, the sorted complete set of every
valid physical loose-object ID, before resolving uniqueness. Immediately before
successful output it must recapture that complete inventory and require exact
equality. An inventory difference must cause the accepted retryable
concurrent-observation failure even if the resolved prefix would remain unique.
An operation that does not use repository-wide Seal prefix resolution must not
scan `I_O` merely for this comparison.

The complete comparison result must remain buffered until all required
recapture and equality checks succeed. Any required observation difference
must fail without emitting an authoritative comparison.

The existing form without `--against` remains byte-for-byte contract-compatible
at the JSON schema level:

```text
sealgraph candidate compare REF [--format human|json]
```

It continues to emit `sealgraph/candidate-compare/v2` and compare only with the
recorded publication baseline.

The new explicit mode emits `sealgraph/candidate-compare/v3`. Its document must
contain:

```text
schema, ref, current_ref_head, expected_head_state,
comparison_target, prospective, changes

comparison_target:
  label = EXPLICIT_SEAL
  selector = the exact user-supplied selector spelling
  seal = the complete resolved seal_view
```

`prospective` and `changes` retain the v2 field meanings and types. For every
change, `before` is the selected Seal field and `after` is the prospective
Candidate field. Human output must name the explicit comparison target and
must continue to use bounded, binary-safe presentation and abbreviated IDs.

## 4. Out of scope

This candidate does not add:

- Candidate-to-Candidate comparison;
- workfile-to-arbitrary-Seal comparison;
- source-binding changes;
- a generalized cross-kind operand grammar for top-level `compare`;
- one-selector predecessor inference;
- textual patches or arbitrary raw-content output;
- automatic conflict resolution after `HEAD_ADVANCED`;
- any Candidate, REF, Seal, object, source, or graph mutation; or
- Git discovery or Git-derived comparison semantics.

These remain optional future candidates and are not acceptance blockers for
this requirement.

## 5. Assumptions and unknowns

Assumptions:

- An explicit Candidate-to-Seal comparison is the smallest useful increment
  that addresses the owner's reported insufficiency without conflating state
  layers.
- Preserving v2 for the existing no-option form is preferable to forcing
  existing machine consumers onto a new schema when they did not request the
  new behavior.

Unknowns:

- Frequency and priority of Candidate-to-Candidate and workfile-to-explicit-Seal
  workflows have not been established.
- A future generalized operand syntax may supersede the CLI spelling proposed
  here, but it must not be treated as required by this candidate.

## 6. Acceptance criteria

The requirement is satisfied only when all of the following hold:

1. The accepted ADR, requirements, CLI contract, help, and schema documentation
   describe the explicit mode without contradicting the preserved default mode.
2. `candidate compare REF --against SELECTOR` resolves every accepted Seal
   selector form and rejects missing, invalid, or ambiguous selections without
   comparison output.
3. The result compares the selected complete Seal view to the complete
   prospective Candidate view in the documented direction.
4. `current_ref_head` and `expected_head_state` remain derived solely from the
   Candidate's publication expectation and observed current head.
5. Explicit comparison neither infers revision ancestry nor labels the selected
   Seal as the Candidate's parent, predecessor, or publication baseline.
6. The explicit mode emits `sealgraph/candidate-compare/v3`; the unchanged mode
   continues to emit its existing `sealgraph/candidate-compare/v2` contract.
7. Human and JSON tests cover an explicit historical Seal, current HEAD after
   Candidate divergence, invalid or ambiguous selectors, complete REF-manifest
   observation mismatch, and conditional loose-object-inventory mismatch for a
   repository-wide Seal prefix.
8. Existing Candidate comparison tests and the full required repository
   validation suite continue to pass.

## 7. Normative and non-normative inputs

Normative if this exact candidate is accepted:

- Sections 1 through 6 of this document.

Non-normative review and feasibility evidence:

- the current implementation of `candidate compare`, `compare`, and
  `source compare`;
- existing tests and help output; and
- future syntax ideas outside the explicit `--against` form.

Implementation evidence cannot accept or expand this requirement.

## 8. Proposed independent-review input

The independent reviewer must receive this exact candidate path and SHA-256
digest and must answer only these questions:

1. Does the candidate contradict an accepted invariant outside the restrictions
   it explicitly proposes to supersede?
2. Does the v2-preservation/v3-explicit split avoid changing existing machine
   output under an old schema identifier?
3. Does R2 preserve ADR 0023's complete `O` recapture and equality contract and
   conditional complete `I_O` recapture and equality contract without adding
   unnecessary inventory scans?
4. Are the in-scope comparison fields, direction, and publication-concurrency
   separation complete and internally consistent?
5. Has any optional future operand pair become an implicit acceptance blocker?

The reviewer must classify findings as contradiction, evidence gap or unknown,
optional or future candidate, or out of scope. The reviewer must not edit the
candidate or turn suggestions into requirement text.

After review, follow exactly the first-owner route selected for this snapshot.

## 9. Self-review record

- Source provenance states the R1 review result and owner `REVISE` route and
  does not treat existing implementation as requirement authority.
- Every directly affected accepted restriction is identified; unrelated
  top-level, source, revision, mutation, and Git boundaries are preserved.
- R2 explicitly preserves complete `O` revalidation and conditional complete
  `I_O` revalidation instead of narrowing them to observed heads.
- In-scope and out-of-scope behavior are explicit.
- The v3 namespace is new and the existing v2 meaning is preserved.
- No release version or delivery milestone is claimed.
- Acceptance criteria cover documentation, semantics, complete observation,
  schemas, errors, focused tests, and regression validation.
- Unknown future operand pairs remain non-blocking.
- The proposed independent-review input is bounded to this requirement.

## 10. Material differences from R1

- Records the completed R1 `FAIL` review and owner `REVISE` disposition.
- Replaces “relevant observed heads” revalidation with complete `O` recapture
  and byte equality.
- Adds conditional complete `I_O` recapture and equality for repository-wide
  `@SEAL_TOKEN` prefix resolution and preserves the no-unnecessary-scan rule.
- Updates the concurrency acceptance test and review question accordingly.
- Makes no other functional or schema change.

