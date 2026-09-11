# Format 6 normative synchronization independent review — 2026-09-11

Status: completed

This is the Step 3 independent requirement review for the exact snapshot routed
by the Operator as `REVIEW`. It does not modify or accept the candidate.

## Reviewed snapshot

- `docs/requirements.md`
  - SHA-256: `02bece52cfcc140d1c671530b9f29971886f05062aae52151bc80fa60683260e`
- `docs/architecture.md`
  - SHA-256: `47d652157b82076aa48984fe9de3e159b2e71663aea6af9030fa04dd6c96e896`
- `docs/storage-format.md`
  - SHA-256: `b53bad2c02729b5290d79db2a00b8255f665387111b44403af22816a38087dc0`
- `docs/cli.md`
  - SHA-256: `0e36fa4ec191a63194187bc28baaa7a481389319535b934e260198fd08cb5970`
- Review input: `docs/process/format6-normative-sync-review-ja-2026-09-11.md`
  - SHA-256: `7530b1ce32e7a77e640d028a0dc9fc5807ee08970f410ffb2ba614b99e21ffe3`

All five pre-review digests matched the owner-reviewed identities. The same
digests were recomputed after review and remained unchanged.

## Authority and method

The authoritative sources were Accepted ADR 0029, Accepted ADR 0030, Accepted
ADR 0031, and ADR 0029's accepted revision-2 requirement authority. Accepted
format-5 ADRs 0023, 0025, and 0027 were used to distinguish preserved format-5
contracts from format-6 successors.

The review compared the candidate against:

- the generic, opaque, identity-bearing, observer-local metadata boundary;
- exact format-6 typed records, limits, historical format-5 readability,
  mixed-generation graph behavior, Candidate projection, migration, and
  `upstream-change/v2` consequences;
- metadata mutation, legacy authoring preservation, human and machine output,
  comparison, `linklog`, and migration receipt contracts;
- format-5 compatibility and external namespace ownership;
- excluded query, registry, validator, Assessment-reference, release,
  repository-migration execution, and Git-sidecar implementation scope; and
- the eight concrete help-navigation routes added to `docs/cli.md`.

Existing implementation and tests were used only as non-normative comparison
evidence. All eight documented help routes executed successfully, matched the
registered parent/leaf command hierarchy, and did not change repository status.

Before relying on the review disposition, the reasoning was audited with the
command-line `llmthink dsl audit` as
`format6-normative-sync-step3-2026-09-11`. The relied-upon audit result was
`fatal=0`, `error=0`, `warning=0`; its 17 hints concerned only long-line DSL
readability and do not limit the substantive review.

## Classified findings

### F-01 — contradiction — P1/high

`docs/requirements.md` lines 111–114 and `docs/cli.md` lines 47–52 require a
repository-wide `@SEAL_TOKEN` to decode as a format-5 Seal. In a format-6
repository, Accepted ADR 0030 lines 193–202 permits Cause targets and asserted
previous revisions to name either valid Seal generation, and format-6 writers
produce Seal v6. The candidate wording therefore makes valid format-6 Seals
unaddressable through the accepted repository-wide selector while
`docs/storage-format.md` correctly describes the target as a canonical Seal
without the format-5 restriction.

This is a format-6 contract contradiction and a format-5 wording carryover, not
a request for broader selector behavior. The earliest correction point is the
Step 1 normative synchronization candidate. Existing implementation happens to
decode both supported Seal generations, but that does not repair the normative
conflict.

### F-02 — evidence gap or unresolved unknown — P2/medium

Accepted ADR 0030 lines 283–302 requires the assessment-free change identity to
advance to `sealgraph/upstream-change/v2`, with complete format-6 Cause Links in
`after_cause_links` and metadata changes affecting `change_id`. None of the four
candidate documents names or disposes this accepted successor schema.

The Accepted ADR remains authoritative, so this is not evidence that the schema
was rejected. It is a missing mandatory normative-synchronization contract that
could allow a later reader to retain `upstream-change/v1` under changed nested
meaning.

### F-03 — evidence gap or unresolved unknown — P2/medium

Accepted ADR 0030 lines 214–231 distinguishes three format-6 Candidate-v5
behaviors: side-effect-free read projection, upgrade only on an authorized
Candidate mutation, and publication by constructing and validating an in-memory
Candidate-v6 projection without first rewriting the Candidate file. The
candidate records the first two only at a high level and does not state the
direct-publication rule.

This omission matters to format-5 compatibility because an implementation or
operator could otherwise infer that a Candidate-v5 file must be rewritten as a
precondition or hidden preliminary migration before `seal`.

### F-04 — evidence gap or unresolved unknown — P3/low

The Step 2 review input states source provenance, scope, exclusions,
compatibility, one acceptance unknown, self-review claims, and the requested
review questions, but it does not explicitly enumerate the candidate's
acceptance criteria as required by the requirement-review lifecycle. The
authoritative ADRs and exact candidate snapshot still supplied enough evidence
to complete this review, so the report is not `not-reviewable`.

On the next Step 1 revision, the review input should state the exact criteria
whose satisfaction would make the four-document synchronization acceptable,
rather than relying on self-review claims as their substitute.

### F-05 — optional or future candidate — P3/informational

No optional or future candidate was promoted into the reviewed acceptance
boundary. Metadata query/filter/AST/index work, schema registries and validators,
core-owned revision-context namespaces, and Assessment-reference commands
remain unintroduced. The existing future Git-sidecar constraints do not promise
implementation in this candidate.

### F-06 — out of scope — P3/informational

Release, actual repository migration, commit, push, deployment, Git-sidecar
implementation, and ADR 0028 Assessment adoption/storage remain outside this
review. No evidence from those areas was used as requirement authority.

## Passed review areas

- Metadata remains one generic required-on-disk collection whose use is
  optional, and it remains embedded in the containing Cause Link and
  Provenance; no CauseLinkID or separate publication lifecycle is introduced.
- Namespace/schema/value records remain bounded, canonical, identity-bearing,
  opaque to core, and independent of Cause/revision graph semantics.
- Historical format-5 object bytes and IDs remain unchanged, and exact
  Seal-v5/Provenance-v1 versus Seal-v6/Provenance-v2 pairing is preserved.
- Legacy format-6 `add` and `link` preserve unmentioned metadata; explicit
  whole-Link deletion still deletes it.
- Changed machine records advance to `/v3`, while `status/v3` and `stale/v2`
  remain unchanged; historical projected empty metadata is distinguished from
  stored v2 emptiness.
- The config-only 5-to-6 migration remains explicit, one-way, atomic, and
  non-retryable after possible commit uncertainty.
- No Sealgraph-owned metadata namespace or external schema ownership was
  assigned.
- Every concrete help-navigation spelling in the candidate matches the current
  command registry and succeeds repository-independently.

## Step 3 disposition

The independent review is `completed`. The unchanged candidate proceeds under
the selected `REVIEW` route to Step 4, but F-01 means this exact snapshot is not
safe to accept. The reviewer recommendation is `REVISE`, returning to Step 1 to
correct the selector contradiction and resolve the mandatory synchronization
gaps. This recommendation is review input only and does not edit requirement
text or make the owner decision.

This review does not accept the requirement and does not authorize a candidate
edit, ADR change, implementation change, commit, push, PR, merge, release,
deployment, repository migration, or runtime mutation.
