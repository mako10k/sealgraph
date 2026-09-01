# ADR 0028: Upstream impact assessment without reverse Cause edges

- Status: Accepted
- Date: 2026-08-31
- Decision Owner: Operator
- Accepted: 2026-08-31 by explicit Operator acceptance of the exact reviewed
  candidate
- Accepted Proposal SHA-256:
  `e4bb0376f9ca4273a0cfec008476d5082e6a52bffb9c46019533bf42e119fcde`
- Acceptance Record:
  [`upstream-impact-assessment-acceptance-2026-08-31.md`](../process/upstream-impact-assessment-acceptance-2026-08-31.md#owner-acceptance-receipt)
- Related Claims: C-UR-001 through C-UR-013
- Related Evidence: E-UR-001 through E-UR-007
- Pending Decision: None
- Execution Authority: Separate; acceptance does not authorize implementation,
  a persisted-format or CLI successor, normative synchronization, migration,
  branch synchronization, commit, push, release, or deployment
- Extends: ADR 0023 impact and Candidate-admission semantics, ADR
  0025 typed-Blob separation, and ADR 0027 format-5 CLI/schema boundaries
- Supersedes: None
- Superseded by: None

## Context

Cause Links are directional claims committed by the dependent observer Seal's
Provenance. If detailed design depends on basic design, the detailed-design
Seal commits to the exact basic-design Seal. Reverse `impact` can already be
derived by reading those same links in the opposite query direction; a reverse
Cause Link or independently active Link registry is unnecessary.

Structural freshness is not semantic consistency. Suppose detailed design is
changed while its basic-design target remains the same current HEAD. A new
detailed-design Seal can satisfy ordinary active-leaf admission without proving
that anyone considered whether basic design also needed revision. Treating the
new dependent Seal itself as proof of review would allow formal resealing to
launder an omitted upstream correction.

Marking the basic-design Seal structurally stale whenever any dependent changes
is not sufficient. Resealing basic design would make its dependents structurally
stale; relinking a dependent could then mark basic design reverse-stale again.
That ping-pong has no stable fixed point and encourages meaningless reseals.

The product therefore needs two separate facts:

1. conservative upstream impact derived from the existing dependency DAG; and
2. an exact immutable assessment that disposes one direct upstream impact for
   one exact dependent change.

The operator selected this operational Sealgraph model and left a fully
symmetric independently versioned relation graph to RefGraph. The same design
discussion selected a compatibility transition for `impact`: omission of a
direction initially retains the current downstream behavior but warns the
caller to specify the direction explicitly.

## Decision

### Dependency ownership remains directional

A Cause Link remains identity-bearing content of the dependent observer Seal's
Provenance. The upstream target does not own or approve its dependents, and no
reverse Cause edge is created. Given a Seal and its immutable closure, the
direct dependency claim remains self-contained and portable.

An upstream assessment is a separate immutable typed Blob rather than a mutable
field owned by either endpoint Seal. Its identity is nevertheless bound to the
exact dependent change and upstream Seal described below. A dependent Seal
adopts an assessment by committing to its exact AssessmentID through a
non-Cause reference. Assessment references:

- are not Cause Links;
- are not Revision assertions;
- are not traversed as Cause or Revision edges by structural graph, stale, or
  downstream-impact queries;
- are followed as typed object references by upstream-assessment inspection,
  the seal-admission gate, integrity validation, and the persisted object
  closure defined by the successor storage decision;
- do not activate the Assessment Blob as a Seal revision; and
- do not make the assessment truthful, approved, or authorized merely because
  its bytes exist.

That object closure follows an adopted AssessmentID to the assessment's
upstream-change Blob, exact upstream Seal, and evidence Seals. It follows the
change Blob's non-null before Seal, after Material, and after Cause targets as
typed references. Dump/load, migration, and continuation therefore cannot
preserve a claim while dropping the exact compared state or named evidence.
Following those references for object availability does not add Cause
membership, structural impact, or current REF authority. The successor storage
decision MUST define exact traversal and validation order and MUST reject a
missing or mistyped member before successful publication or export.

The accepted format-5 Provenance and Candidate schemas have no such non-Cause
reference. This ADR does not overload `messages`, attachments, or generic Cause
Link metadata to simulate one. Before implementation, a separate accepted
storage-format decision MUST define the exact successor Seal, Provenance,
Candidate, dump/load, migration, and canonical-byte contracts. Until then, all
actions below remain design targets.

This establishes:

- C-UR-001: dependency authority remains with the dependent observer without a
  reverse edge or external active-Link authority;
- C-UR-002: assessment authorship and dependent adoption are distinct exact
  identities; and
- C-UR-003: assessment evidence cannot alter structural graph semantics.

### Explicit impact direction with a warning transition

The conceptual command surface is:

```text
sealgraph impact [--downstream]
                 [--asserted-by OBSERVER_SELECTOR]...
                 [--all-paths] [--max-paths N]
                 SELECTOR [--format human|json]

sealgraph impact --upstream REF
                 [--all-paths] [--max-paths N]
                 [--format human|json]

sealgraph impact --upstream
                 --from LEFT_SELECTOR --to RIGHT_SELECTOR
                 [--all-paths] [--max-paths N]
                 [--format human|json]
```

The upstream REF form is explicitly a Candidate-change query and fails with
`CANDIDATE_NOT_FOUND` when the named REF has no Candidate. It does not accept an
arbitrary Seal selector because one immutable state does not identify a before/
after change. The selector-pair form is the immutable historical comparison.
For both directions, `--max-paths` is invalid without `--all-paths`, must be
positive, and defaults to 100 when all paths are requested.
`--format human|json` is accepted once. Omission retains ADR 0022's current
terminal-human and redirected/piped-JSON selection rule.

`--downstream` and `--upstream` are mutually exclusive. Supplying both fails
with `DIRECTION_CONFLICT`. The existing `--asserted-by` option is valid only
for downstream impact; using it with `--upstream` fails with
`OPTION_NOT_APPLICABLE`.

For the initial transition, omitting both direction options is exactly
equivalent to explicit `--downstream`, except that the command emits this one
line to stderr after successful argument validation and before repository
observation:

```text
warning: impact direction omitted; defaulting to --downstream; specify --downstream or --upstream explicitly
```

The warning is emitted for terminal, redirected, and JSON output. It never
appears on stdout and therefore cannot corrupt existing human bytes or the
`sealgraph/impact/v2` JSON document. Explicit `--downstream` emits no warning
and retains the exact accepted downstream membership, ordering, path, filter,
and output schema. Explicit `--upstream` emits the new schema defined below.
One executable impact invocation emits the line at most once. Help, completion,
version, and argument-validation failure paths emit no direction warning
because they do not begin an impact query.

This ADR does not authorize a later hard error when direction is omitted. Making
direction mandatory requires another accepted CLI transition after callers have
had an explicit migration period.

This establishes C-UR-004: direction omission remains operationally compatible
while becoming observable and mechanically removable from callers.

### Upstream change identity

Candidate-mode upstream impact compares the candidate's exact
`expected_ref_head` with its assessment-free prospective semantic projection.
The accepted `expected_head_state` must be `EXPECTED_ABSENT` or
`EXPECTED_CURRENT`. `HEAD_ADVANCED`, `HEAD_MISSING`, or `UNEXPECTED_HEAD` fails
the query, names that exact state in the diagnostic, and emits no stdout
document.

The exact logical change preimage is:

```json
{"schema":"sealgraph/upstream-change/v1","before_seal":null,"after_material":"<material-id>","after_root":false,"after_draft":false,"after_cause_links":[]}
```

Member order is exactly:

```text
schema, before_seal, after_material, after_root, after_draft, after_cause_links
```

`before_seal` is the full expected-head SealID or JSON `null` for expected
absence. `after_material` is the full prospective MaterialID. Root, draft, and
Cause Links use ADR 0025's exact canonical values, member order, array sorting,
and validation. Assessment references are deliberately absent. Compact scalar-
exact JSON without a trailing LF is hashed with the Universal Blob envelope;
the resulting BlobID is the `change_id`.

Adding, replacing, or removing an assessment therefore does not change the
`change_id` and cannot create a self-referential hash. Changing material, root,
draft, or any Cause Link does change it and makes every assessment for the old
change inapplicable. The immutable old Assessment Blob is retained; no command
rewrites or deletes it automatically.

The historical form with `--from` and `--to` accepts two immutable Seal-
resolving selectors, reads no Candidate, and computes the same preimage from
the left SealID and the right Seal's assessment-free semantic projection. The
two options are required together. The REF form and the selector-pair form are
mutually exclusive. `--from` and `--to` are valid only with `--upstream`; using
either with downstream mode fails with `OPTION_NOT_APPLICABLE`.

This establishes C-UR-005: an assessment binds to the changed dependent
semantics without including or being invalidated by its own adoption reference.

### Direct obligations and transitive information

Let `B` be the set of exact direct Cause target SealIDs in the before state and
`A` the set in the after state. Upstream impact evaluates every member of:

```text
U = B union A
```

For `u` in `U`, `edge_change(u)` is:

```text
ADDED     when u is in A and not B
RETAINED  when u is in A and B
REMOVED   when u is in B and not A
```

Using the union prevents removal of a Cause Link from erasing the obligation to
consider the former upstream. Retargeting from one exact upstream Seal to
another creates one `REMOVED` and one `ADDED` obligation; it is not collapsed
into an implicit replacement.

Every `u` is a direct review obligation. Traversing Cause Links from `u` toward
its own upstream closure yields transitive impact information, but does not
immediately create review obligations for those transitive ancestors. If an
assessment says that `u` must change and a new revision of `u` is prepared,
that separate change creates its own direct upstream obligations. Review thus
progresses one dependency layer at a time rather than exploding recursively.

Default upstream presentation emits one shortest Cause path to each distinct
transitive upstream Seal. Equal lengths use the bytewise full-SealID path.
`--all-paths` and positive `--max-paths N` retain ADR 0023's per-target default
of 100, deterministic simple-path ordering, explicit truncation, and separation
between membership and presentation bounds. Complete graph validation and one
coherent observation precede all output.

Candidate mode captures the exact Candidate version and complete current
manifest set; selector-pair mode captures the complete manifest set used for
aliases and assessment validation. Each mode derives and buffers the full
result, then revalidates every captured mutable identity before stdout. Change
or unreadability fails nonzero with empty stdout; it never emits a plausible
partial impact document.

This establishes:

- C-UR-006: direct obligations cover added, retained, and removed exact
  dependencies; and
- C-UR-007: transitive upstreams are visible without prematurely imposing a
  repository-wide batch review.

### Immutable Assessment Blob

The logical Assessment Blob is exact compact JSON:

```json
{"schema":"sealgraph/upstream-assessment/v1","change":"<change-id>","upstream_seal":"<seal-id>","edge_change":"RETAINED","disposition":"COMPATIBLE_AS_IS","evidence_seals":["<seal-id>"]}
```

Member order is exactly:

```text
schema, change, upstream_seal, edge_change, disposition, evidence_seals
```

The fields are:

- `change`: one full typed upstream-change BlobID;
- `upstream_seal`: one full SealID in the exact direct obligation set;
- `edge_change`: exactly `ADDED`, `RETAINED`, or `REMOVED`, matching the
  recomputed obligation;
- `disposition`: exactly `COMPATIBLE_AS_IS` or
  `UPSTREAM_CHANGE_REQUIRED`; and
- `evidence_seals`: a non-empty, sorted, duplicate-free array of full SealIDs.

AssessmentID is the Universal Blob ID of the canonical bytes. Evidence Seals
are exact immutable review material but are non-Cause references from the
Assessment Blob. Sealgraph validates their identities and availability; it does
not infer reviewer identity, authority, signature validity, truth, or approval.
An external policy may require evidence issued by a particular reviewer or
signature system, but that policy is outside this core schema.

The Assessment Blob uses ADR 0025's canonical UTF-8 JSON string rules, exact
member order, no insignificant whitespace, and no trailing LF. Its
`evidence_seals` array sorts by full SealID bytes; resolving two inputs to the
same Seal is a duplicate error, not silent deduplication. The typed reader
rejects unknown members, invalid enums, noncanonical order or encoding, and any
referenced upstream-change Blob that fails its exact schema and byte check.

Within one Candidate or Seal, at most one adopted AssessmentID may name one
exact `upstream_seal`, irrespective of the assessment's `change`. Creating a
second assessment does not overwrite the first. Candidate adoption uses
explicit expected-old replacement or removal.

This establishes C-UR-008: every disposition is exact, independently immutable,
evidence-bearing, and non-structural.

### Review states and seal admission

After all adopted references pass integrity validation, each direct obligation
is evaluated by this ordered, exhaustive, and mutually exclusive function:

```text
1. no adopted assessment names the exact upstream
   -> UPSTREAM_REVIEW_REQUIRED

2. an adopted assessment names the upstream but its change or edge_change
   differs from the current obligation
   -> UPSTREAM_ASSESSMENT_OUTDATED

3. change and edge_change both match, and disposition is COMPATIBLE_AS_IS
   -> UPSTREAM_COMPATIBLE_AS_IS

4. change and edge_change both match, and disposition is
   UPSTREAM_CHANGE_REQUIRED
   -> UPSTREAM_CHANGE_REQUIRED
```

Disposition is consulted only after applicability matches; an old
`COMPATIBLE_AS_IS` assessment can therefore never satisfy a new change. After
all direct obligations are classified, each adopted, type-valid assessment
whose `upstream_seal` is outside the before/after union is classified once as
an `UPSTREAM_ASSESSMENT_OUTDATED` adoption rather than a direct obligation. It
blocks sealing until explicit removal. An unknown, missing, corrupt, duplicate,
or type-invalid assessment is an integrity or Candidate-validation error, not
`REVIEW_REQUIRED` or `OUTDATED`.

Normal non-draft seal admission retains every ADR 0023 Cause-closure rule and
additionally requires every direct upstream obligation to be exactly
`UPSTREAM_COMPATIBLE_AS_IS` and requires no outdated adoption outside the
obligation union. `UPSTREAM_REVIEW_REQUIRED`, `UPSTREAM_CHANGE_REQUIRED`, or
`UPSTREAM_ASSESSMENT_OUTDATED` blocks normal publication without changing any
REF.

Draft publication is also blocked while any direct obligation is unresolved or
an outdated adoption exists outside the obligation union. The Candidate
remains the only provisional state. This deliberately avoids losing an
unresolved obligation when successful publication cleans up the Candidate: the
accepted format-5 Seal does not commit the expected before Seal or a complete
unresolved-obligation set. A future accepted decision may permit unresolved
draft publication only by defining exact persisted change identity,
unresolved-obligation carry-forward, and later-discharge rules; that is not an
implementation choice under this ADR.

The new gate applies only when publishing a Candidate under the accepted
successor contract. Existing format-5 Seals are not retroactively corrupt,
stale, unapproved, or unpublished because they lack Assessment references, and
migration MUST NOT fabricate compatible dispositions. The storage successor
must define their exact migration projection. A historical comparison may
truthfully report that no matching assessment exists without changing the
validity of either historical Seal.

This gate is only the upstream-review subgate. `SATISFIED` does not mean that a
Candidate is publishable: expected-head state, Candidate and object integrity,
ADR 0023 structural Cause-closure admission, writer revalidation, and the final
REF CAS remain separate blockers. Human or machine output MUST NOT label this
subgate as aggregate publication readiness.

This establishes C-UR-009: structural freshness and upstream semantic review
are independent gates, no reseal alone discharges a review obligation, and the
new admission rule does not retroactively invalidate or approve old Seals.

### Assessment authoring and inspection commands

The proposed command surface is:

```text
sealgraph assess upstream REF
                 --against UPSTREAM_SELECTOR
                 --result compatible|change-required
                 --evidence EVIDENCE_SELECTOR...
                 [--replace-assessment EXPECTED_ASSESSMENT_ID]
                 [--format human|json]

sealgraph assessment show REF [--format human|json]

sealgraph assessment remove REF
                 --upstream UPSTREAM_SELECTOR
                 --expected-assessment ASSESSMENT_ID
                 [--format human|json]

sealgraph review-required [REF] [--format human|json]
```

For every command above, explicit `--format human|json` is accepted once. When
omitted, ADR 0022's current terminal-first rule applies: terminal stdout selects
human output and redirected or piped stdout selects JSON. The warning for an
omitted impact direction is independent of that choice and remains stderr-only.

`assess upstream` acquires the ordinary repository-wide writer guard before its
first repository observation and holds it through Candidate replacement. It
requires a valid Candidate, computes its current `change_id`, resolves
`--against` to one exact member of the direct obligation union, and resolves
every repeated `--evidence` selector to a full SealID before writing anything.
At least one evidence selector is required. Two selectors resolving to the same
evidence Seal fail with `DUPLICATE_EVIDENCE`. The command then stores or
verifies the canonical upstream-change Blob, stores or verifies the immutable
Assessment Blob, and adopts the exact AssessmentID with exact Candidate-version
comparison. A failure before Candidate replacement may leave unreachable
immutable Blobs; their existence is not assessment adoption or publication.

`--against` is resolved exactly, not by logical document identity. In
particular, a removed obligation whose old target is no longer a current REF
head normally requires an explicit historical Seal selector. Resolving a REF
to a different current Seal fails with `UPSTREAM_NOT_IN_OBLIGATION_SET` and
identifies the exact outstanding upstream SealID; it does not retarget the
obligation.

If no assessment is currently adopted for that upstream,
supplying `--replace-assessment` fails with `ASSESSMENT_NOT_FOUND`. If one
exists, omission of
`--replace-assessment` fails with `ASSESSMENT_EXISTS`; supplying a value other
than the exact current ID fails with `ASSESSMENT_CHANGED`. Removal likewise
requires the exact expected AssessmentID; absence fails with
`ASSESSMENT_NOT_FOUND` and mismatch fails with `ASSESSMENT_CHANGED`. Expected
AssessmentIDs are full 64-character lower-hex IDs, not selectors or display
prefixes. No operation deletes an immutable Blob.

`assessment remove` uses the same writer guard and exact Candidate-version
comparison as `assess upstream`, but deliberately does not require union
membership. Under the guard, `--upstream` must resolve to the exact
`upstream_seal` named by one Assessment currently adopted by the Candidate;
that target may be outside the current obligation union. Absence fails with
`ASSESSMENT_NOT_FOUND`, and the full expected AssessmentID must still match.
Both mutation commands fail with `CANDIDATE_NOT_FOUND` before any write when the
REF has no Candidate. Successful removal changes only Candidate adoption; the
Assessment, change, evidence, and other immutable Blobs remain untouched.

Candidate mutations that change `change_id` preserve adopted AssessmentIDs so
the mismatch remains visible as `UPSTREAM_ASSESSMENT_OUTDATED`; they never
silently relabel, remove, or apply an old assessment to new bytes. The successor
Candidate schema and inspection schema must represent an outdated assessment
without fabricating a valid prospective SealID. Explicit replacement or removal
is required before normal sealing.

`review-required` reads Candidate review states. Without `REF`, it returns all
logical REFs having at least one unresolved direct obligation or outdated
adoption. It does not promote transitive informational impact into an
obligation.

`assessment show REF` and `review-required REF` fail with
`CANDIDATE_NOT_FOUND` when that REF has no Candidate. The unscoped
`review-required` command examines only existing Candidates. Its first capture
is the sorted complete Candidate-namespace inventory: each decodable logical
REF, its candidate-designated relative path, file type and safety
classification, and exact Candidate bytes. The inventory also records every
unsafe, unreadable, or undecodable entry that could occupy a candidate-
designated path; established non-Candidate index entries are classified and
excluded by the successor storage contract. Unsafe, unexpected, duplicate-
decoding, unreadable, or invalid Candidate entries fail the query rather than
disappearing from membership. After deriving and buffering the complete
result, the command repeats the same full inventory and requires exact
equality. Addition, removal, replacement, byte change, type change, or
unreadability fails nonzero with empty stdout. At both boundaries it also
captures and requires equality of the complete current manifest-head set used
for selector resolution and `current_refs`. Scoped inspection similarly
revalidates the exact Candidate bytes and captured manifest heads before
output. As with ADR 0023 observations, equal endpoint captures establish only
the recorded observation; they do not prove continuous immutability, detect a
change-and-restore ABA sequence, reserve the result, or promise freshness after
return.

This establishes C-UR-010: assessment mutation is explicit, expected-old,
single-subject, and never automatic approval or repair.

### Successful human output

Candidate-mode upstream impact is presented as:

```text
UPSTREAM IMPACT

subject       DETAIL-001
change        d152a31c890e -> candidate
change id     09b15e12c7ae
candidate     non-draft
upstream review  blocked

DIRECT REVIEW OBLIGATIONS

UPSTREAM    SEAL          EDGE      ASSESSMENT
BASIC-001   b183ce32d319  RETAINED  REVIEW REQUIRED

TRANSITIVE IMPACT

UPSTREAM    VIA         DEPTH
REQ-001     BASIC-001   2
```

A successful compatible assessment receipt is:

```text
UPSTREAM ASSESSMENT RECORDED

subject       DETAIL-001
change id     09b15e12c7ae
upstream      BASIC-001 @ b183ce32d319
edge          RETAINED
result        COMPATIBLE AS-IS
assessment    a8713c142e8a
evidence      REVIEW-042 @ 71a26f8e1290
candidate     non-draft
upstream review  satisfied
```

A change-required receipt substitutes:

```text
result        UPSTREAM CHANGE REQUIRED
candidate     non-draft
upstream review  blocked
```

Successful removal is presented as:

```text
UPSTREAM ASSESSMENT REMOVED

subject          DETAIL-001
candidate change 6c1123e37d4a
upstream         BASIC-001 @ b183ce32d319
assessment       a8713c142e8a
assessed change  09b15e12c7ae
upstream review  blocked
```

Human IDs use ADR 0022's unambiguous presentation prefixes. Every successful
JSON document contains full IDs.

### Exact successful JSON documents

Upstream impact uses `sealgraph/upstream-impact/v1` with exact top-level order:

```text
schema, subject, all_paths, max_paths, direct_impacts,
outdated_assessments, transitive_impacts, upstream_review_gate
```

Its record shapes are:

```text
subject:
  ref, change_id, before_seal_id, after_kind, after_seal_id

direct_impact:
  upstream_seal_id, current_refs, current_edge_change, assessment_state,
  assessment_id, assessment_change_id, assessment_edge_change,
  disposition, evidence_seal_ids

outdated_assessment:
  assessment_id, upstream_seal_id, assessment_change_id,
  assessment_edge_change,
  disposition, evidence_seal_ids, state

transitive_impact:
  upstream_seal_id, current_refs, paths, paths_truncated

impact_path:
  seal_ids, depth

upstream_review_gate:
  after_draft, status, blocking_states
```

`subject.ref` is a REF or JSON `null` for the selector-pair form.
`before_seal_id` is a full SealID or null. `after_kind` is `CANDIDATE` or
`SEAL`; `after_seal_id` is null for Candidate and a full SealID for Seal.
`current_edge_change` is the direct obligation's exact enum. `assessment_id`,
`assessment_change_id`, `assessment_edge_change`, and `disposition` are all
null exactly for `UPSTREAM_REVIEW_REQUIRED`; otherwise they contain the exact
adopted Assessment fields. Evidence arrays are present and empty only when an
assessment is absent. `current_refs` are all sorted REF aliases of the exact
upstream Seal.
Every `outdated_assessments` member is one adopted, type-valid assessment whose
upstream is outside the current obligation union; its `state` is exactly
`UPSTREAM_ASSESSMENT_OUTDATED`. The array is sorted by full AssessmentID.

For the selector-pair form, assessment states are evaluated against assessment
references actually adopted by the right Seal. A format-5 right Seal has none
and therefore reports `UPSTREAM_REVIEW_REQUIRED`; a successor-format right
Seal is validated under its accepted typed-reference contract.
`upstream_review_gate` is
the result of applying this ADR's admission policy to the hypothetical
left-to-right comparison. It does not retroactively declare that an already
published historical right Seal was valid or invalid under a policy that did
not yet exist.

`all_paths` is false and `max_paths` is null by default. With `--all-paths`,
`all_paths` is true and `max_paths` is the positive requested bound or 100 when
omitted. Direct impacts sort by full upstream SealID. Transitive impacts sort
by full upstream SealID. Each path starts at one direct upstream and ends at
the record's `upstream_seal_id`; direct obligations do not also appear as
transitive records. A path's `seal_ids` omit the unsealed subject and begin at
one direct upstream; `depth` is one plus that path's Cause-edge count so it is
the conceptual Cause distance from the subject and is at least two. Paths sort
by `(depth, full seal_ids sequence)`. Default output contains exactly the first
path and sets `paths_truncated` exactly when another path exists. With
`--all-paths`, it contains the first `min(max_paths, path_count)` paths and sets
`paths_truncated` exactly when more exist. Path bounds never change transitive
membership or validation.

`upstream_review_gate.after_draft` is the exact Candidate `draft` value, or the
right Seal's `draft` value for the selector-pair form.
`upstream_review_gate.status` is `SATISFIED` exactly when every direct obligation is
`UPSTREAM_COMPATIBLE_AS_IS` and `outdated_assessments` is empty; otherwise it is
`BLOCKED`. `blocking_states` is the bytewise sorted distinct set of direct
non-compatible assessment states plus `UPSTREAM_ASSESSMENT_OUTDATED` when an
outdated adoption exists outside the obligation union. It is empty exactly when
`status` is `SATISFIED` and non-empty exactly when `status` is `BLOCKED`.

The assessment mutation receipt uses
`sealgraph/upstream-assessment-update/v1`:

```text
schema, action, ref, candidate_change_id, upstream_seal_id,
current_edge_change, assessment_change_id, assessment_edge_change,
disposition, assessment_id, evidence_seal_ids, upstream_review_gate
```

`action` is `RECORDED` or `REMOVED`. `candidate_change_id` is the full current
Candidate change after the mutation. The assessment-prefixed fields,
`disposition`, `assessment_id`, and evidence are the exact immutable Assessment
Blob that was adopted or removed. For `RECORDED`, `current_edge_change` is
non-null and equals `assessment_edge_change`, and the two change IDs are equal.
For `REMOVED`, `current_edge_change` is the current direct-obligation enum or
null when the removed assessment's upstream is outside the union; the stored
assessment fields are never rewritten to match it.

`upstream_review_gate` has the same exact `after_draft`, `status`, and
`blocking_states` record as upstream impact, computed across the complete
Candidate after mutation.

Assessment inspection uses `sealgraph/upstream-assessment-show/v1`:

```text
schema, ref, change_id, assessments, unresolved_upstream_seal_ids,
upstream_review_gate
```

Each `assessments` member has the stored Assessment Blob fields plus
`assessment_id` and derived `state`. Its exact member order is:

```text
assessment_id, assessment_change_id, upstream_seal_id,
current_edge_change, assessment_edge_change, disposition,
evidence_seal_ids, state
```

`current_edge_change` is the current direct-obligation enum or null when the
assessment's upstream is outside the union. The assessment-prefixed values are
the immutable stored values. Assessments sort by
`(upstream_seal_id, assessment_id)`.
`unresolved_upstream_seal_ids` is the sorted duplicate-free set of direct
upstream SealIDs whose derived state is not `UPSTREAM_COMPATIBLE_AS_IS`; it does
not include an outdated adoption outside the obligation union, which remains
visible in `assessments` and `upstream_review_gate`.

Pending review uses `sealgraph/upstream-review-required/v1`:

```text
schema, records

record:
  subject_ref, change_id, kind, upstream_seal_id, current_refs,
  current_edge_change, state, assessment_id, assessment_change_id,
  assessment_edge_change
```

`kind` is `DIRECT_OBLIGATION` for a non-compatible direct state and
`OUTDATED_ADOPTION` for an adopted assessment outside the obligation union.
`current_edge_change` is null only for `OUTDATED_ADOPTION`; otherwise it is the
direct obligation enum. `assessment_id`, `assessment_change_id`, and
`assessment_edge_change` are null only for `UPSTREAM_REVIEW_REQUIRED`;
otherwise they are the full adopted AssessmentID, full stored change BlobID,
and stored edge enum. Every `current_refs` array is sorted by REF bytes. Records
sort by
`(upstream_seal_id, subject_ref, change_id, kind)`. These schemas report review
state; they do not claim that a human review was competent, truthful, approved,
or complete outside the exact recorded disposition.

All four successful JSON documents use ADR 0027's compact canonical JSON
encoding and exactly one final LF. Unknown members and noncanonical input are
never accepted as typed Assessment or upstream-change Blobs. The direction
warning remains stderr-only and is not part of any JSON document.

This establishes:

- C-UR-011: direction, impact, review, and mutation output have separate stable
  machine contracts; and
- C-UR-012: JSON exposes the exact change, upstream, assessment, evidence, and
  upstream-review subgate facts needed to audit a formal reseal.

### Composite execution authority

Acceptance of this semantic ADR alone authorizes no runtime or persisted-format
change. For each A-UR action, every ADR named in its `Accepted ADR gate` cell
below must be Accepted before that action starts. A missing storage or CLI
successor is a closed gate, not an implementation detail that may be completed
inside the action. Acceptance of one action's gate does not open another action
or authorize migration, release, or deployment.

This establishes C-UR-013: implementation authority is action-specific and
composite; no semantic, storage, migration, or CLI decision substitutes for a
missing companion acceptance.

## Alternatives

### Add reverse Cause Links

Rejected because the upstream document does not own its dependents, and a
reverse edge creates cycle and stale-propagation ambiguity in the dependency
DAG.

### Make Links independently active symmetric entities

Rejected for Sealgraph because it requires a separately versioned active-Link
set, publication authority, replacement lifecycle, and graph snapshot. RefGraph
may explore that theory without becoming a Sealgraph runtime dependency.

### Treat resealing against the same upstream HEAD as review

Rejected because it permits an unchanged formal dependency selection to erase
the fact that upstream semantic impact was never assessed.

### Mark upstream Seals structurally stale

Rejected because structural stale currently means that an exact dependency
target is no longer an active leaf. Reusing the term for an unreviewed reverse
impact conflates different creating mechanisms and can cause reseal ping-pong.

### Store AssessmentID in a Cause Link message or attachment

Rejected because a reserved message encoding lacks a typed reference boundary,
and an attachment makes governance evidence part of Material rather than
Provenance. Both avoid the required schema decision by hiding semantics in an
existing field.

### Make impact direction mandatory immediately

Rejected for the first transition because current callers omit direction and
already mean downstream. The explicit stderr warning provides discoverability
without corrupting stdout or silently changing query meaning.

### Keep direction omission silent indefinitely

Rejected because `impact` becomes genuinely bidirectional and a positional
selector alone no longer communicates operator intent clearly.

## Consequences

Good:

- Existing Cause ownership and DAG acyclicity remain intact.
- One graph supports downstream and upstream impact queries.
- Removing a dependency cannot erase its upstream review obligation.
- A compatible-as-is decision is exact, evidence-bearing, and bound to one
  assessment-free dependent change.
- Assessment adoption does not create a self-referential identity.
- Direct-only obligations avoid immediate transitive review explosions.
- Omitted direction remains compatible while producing a precise migration
  signal on stderr.

Bad / Risk:

- Semantic consistency still depends on human or external-policy competence;
  core can prove only exact recorded identities and dispositions.
- A new non-Cause Provenance reference requires a successor persisted format,
  migration decision, canonical fixtures, and inspection schemas.
- Candidate mutation becomes more complex because old immutable assessments
  must remain visible without applying to new changes.
- Unresolved upstream review cannot be published even as draft in this initial
  contract; the Candidate is the resumable provisional state.
- Every normal dependent change with direct Cause targets requires at least one
  sealed evidence item per distinct assessment, increasing workflow cost.
- Upstream impact is conservative and may request review for changes that are
  semantically irrelevant to the upstream document.
- Direction warnings on stderr may expose callers that incorrectly require
  empty stderr on successful commands; explicit `--downstream` is the remedy.

Neutral:

- Assessment does not mutate, invalidate, or approve either endpoint Seal.
- The absence of an Assessment Blob is not corruption; its absence becomes a
  Candidate review obligation only in the context of one exact change.
- This ADR does not authorize format implementation, migration, automatic
  review, recursive sealing, batch publication, push, release, or deployment.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-UR-001 implement explicit direction parsing and warning | ADR 0028 plus accepted CLI successor | C-UR-004, C-UR-011, C-UR-013 | exact stderr bytes, no-warning explicit directions/help/parser failures, at-most-once warning, JSON stdout isolation, conflict/not-applicable errors, terminal/pipe/file fixtures |
| A-UR-002 implement assessment-free change identity | ADR 0028 plus accepted storage successor | C-UR-005, C-UR-013 | fixed before-null/non-null, material/root/draft/added/retained/removed Link bytes and IDs; proof that adopted assessment changes do not change `change_id` |
| A-UR-003 implement upstream impact | ADR 0028 plus ADRs 0023 and 0027 and accepted storage and CLI successors | C-UR-003, C-UR-006, C-UR-007, C-UR-011 through C-UR-013 | union, retarget, removal, direct/transitive, aliases, path bounds, cycle/corruption, Candidate expected-head, historical pair, ordering, and coherent-observation fixtures |
| A-UR-004 implement Assessment Blob and Candidate adoption | ADR 0028 plus accepted storage successor | C-UR-001 through C-UR-003, C-UR-005, C-UR-008, C-UR-010, C-UR-013 | exact typed bytes/IDs, evidence resolution, missing/duplicate/outside-union/outdated, expected-old create/replace/remove, outside-union removal, no-delete, and non-Cause traversal fixtures |
| A-UR-005 add review state and upstream-review subgate | ADR 0028 plus accepted storage and CLI successors | C-UR-009, C-UR-011 through C-UR-013 | exhaustive ordered state truth table across missing/compatible/change-required/outdated/corrupt, explicit non-aggregate subgate output, complete Candidate-namespace double-capture, original structural admission coexistence, normal and draft no-write rejection, unchanged Candidate preservation, and no retroactive format-5 invalidity or fabricated assessment |
| A-UR-006 synchronize normative/public contracts | ADR 0028 plus every accepted successor above | C-UR-001 through C-UR-013 | requirements, architecture, storage-format, CLI, integrations, index, help, completion, fixed JSON, migration, and cross-reference checks |

Acceptance of this ADR does not start any implementation action. In particular,
the storage successor must decide exact non-Cause reference bytes, Candidate
invalid/outdated representation, repository format number, migration
projection, and receipt behavior before A-UR-002, A-UR-004, or A-UR-005 can
begin.

## Review

The primary design review separated dependency ownership, reverse traversal,
semantic assessment, and approval authority. It rejected independent active
Links and reverse Cause edges because those change the source of graph
authority rather than merely adding a query direction.

The pre-delegation self-review found and corrected these design traps:

1. binding an assessment to a Candidate identity that includes the assessment
   creates a self-reference, so `change_id` excludes assessment references;
2. examining only after-state dependencies lets Link removal erase an
   obligation, so direct impact uses the before/after union;
3. marking upstream structurally stale creates reseal ping-pong, so review
   states remain separate from stale;
4. allowing unresolved draft publication without persistence loses obligations
   at Candidate cleanup, so the initial contract blocks draft publication and
   preserves the Candidate;
5. representing `--all-paths` as repeated singular path records loses per-target
   truncation truth, so each transitive target owns its ordered path set and
   explicit truncation bit;
6. presenting normal and draft gates for one Candidate contradicts the exact
   `after_draft` value included in its change identity, so one exact upstream-
   review subgate is reported;
7. preserving an assessment after its upstream leaves the before/after union
   can otherwise hide it outside direct impacts, so outdated adoptions are
   separately visible and blocking;
8. excluding Assessment references from all reachability can preserve a claim
   while dropping its change or evidence, so typed object closure is separated
   from structural graph traversal;
9. accepting one immutable operand for upstream assessment leaves the compared
   change unknown, so upstream mode requires either a Candidate REF or an exact
   left/right Seal pair;
10. silent evidence deduplication, prefix-based expected-old mutation, or
    unlocked inspection weakens exactness, so duplicate evidence fails,
    expected AssessmentIDs are full, and mutable observations are revalidated;
11. reusing a recording receipt for removal conflates current Candidate state
    with the removed Assessment's older claim, so the action, current change,
    assessed change, and both edge values are reported separately; and
12. applying the new gate retroactively would turn missing historical evidence
    into fabricated invalidity or migration approval, so it governs only new
    successor publications and forbids synthetic assessments.

The first frozen candidate (`sha256:3bbf78f1520fed6a98c4551d393dea171156dd6eb4139ba1d73c2265ab4bd200`)
then received independent ADR-internal `FAIL`, related-ADR
`PASS_WITH_FINDINGS`, and repository-wide `FAIL` verdicts. The primary agent
discarded that candidate and corrected every supported unique finding before a
new review: ordered mutually exclusive state evaluation, outside-union removal,
complete Candidate-namespace double capture, the non-aggregate upstream-review
subgate, explicit mutation format selection, exact Action/Claim transpose with
E-UR-007 closure, and D-UR-006 scenario provenance. No verdict on the discarded
bytes transfers to this candidate.

The final exact candidate at
`sha256:e4bb0376f9ca4273a0cfec008476d5082e6a52bffb9c46019533bf42e119fcde`
received independent ADR-internal, related-ADR, and repository-wide `PASS`
verdicts with no P0 through P3 finding. Each scope verified that digest before
and after review. The integrated verdict was `READY_FOR_OWNER_DECISION`, not
acceptance by itself. On 2026-08-31, the Operator then explicitly accepted
that exact candidate. The review and owner-acceptance readback are preserved
in the linked Acceptance Record.

## Evidence

- E-UR-001: accepted ADR 0023 defines Cause Links as identity-bearing content
  of the observer Provenance and retains assertion sources while deriving
  structural graph unions.
- E-UR-002: accepted ADR 0025 commits exact Cause Links through Provenance and
  contains no non-Cause assessment-reference field.
- E-UR-003: accepted ADR 0027 versions incompatible output shapes and separates
  Candidate/status records from factual stale records.
- E-UR-004: accepted ADRs 0010 and 0023 make stale and impact derived factual
  observations rather than approval or mandatory work.
- E-UR-005: Proposed ADR 0017 records non-authoritative supporting rationale
  against core-defined application approval semantics and query mutation
  disguised as read-only graph inspection.
- E-UR-006: the operator's basic-design/detailed-design scenario identified
  formal resealing against an unchanged upstream HEAD as insufficient semantic
  review; the selected ownership, Assessment Blob, RefGraph boundary, and
  direction-warning decisions are recorded in
  [`../process/upstream-impact-assessment-decision-2026-08-31.md`](../process/upstream-impact-assessment-decision-2026-08-31.md).
- E-UR-007: the accepted four-ADR execution gate recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md)
  demonstrates that an exact semantic decision does not authorize storage or
  runtime work before every required companion contract is accepted.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-UR-001 | E-UR-001, E-UR-006 | A-UR-004, A-UR-006 |
| C-UR-002 | E-UR-002, E-UR-006 | A-UR-004, A-UR-006 |
| C-UR-003 | E-UR-001, E-UR-004, E-UR-005 | A-UR-003, A-UR-004, A-UR-006 |
| C-UR-004 | E-UR-003, E-UR-006 | A-UR-001, A-UR-006 |
| C-UR-005 | E-UR-002, E-UR-006 | A-UR-002, A-UR-004, A-UR-006 |
| C-UR-006 | E-UR-001, E-UR-006 | A-UR-003, A-UR-006 |
| C-UR-007 | E-UR-001, E-UR-004, E-UR-006 | A-UR-003, A-UR-006 |
| C-UR-008 | E-UR-002, E-UR-005, E-UR-006 | A-UR-004, A-UR-006 |
| C-UR-009 | E-UR-001, E-UR-003, E-UR-004, E-UR-006 | A-UR-005, A-UR-006 |
| C-UR-010 | E-UR-002, E-UR-005 | A-UR-004, A-UR-006 |
| C-UR-011 | E-UR-003, E-UR-006 | A-UR-001, A-UR-003, A-UR-005, A-UR-006 |
| C-UR-012 | E-UR-003, E-UR-004, E-UR-006 | A-UR-003, A-UR-005, A-UR-006 |
| C-UR-013 | E-UR-007 | A-UR-001 through A-UR-006 |

## Follow-ups

- Preserve the final exact-candidate review and owner-acceptance receipt; any
  material change to the accepted technical decision requires a new exact-
  candidate review and owner decision.
- Under a separately authorized design task, decide the exact successor
  persisted format, migration, and Candidate invalid/outdated representation
  in a separate ADR before implementation.
- After the successor decision is accepted and the work is separately
  authorized, update the active PERT and normative/public contracts before
  beginning runtime work.
- Let RefGraph explore independently active symmetric Links or graph snapshots;
  do not add that authority to Sealgraph as an implementation convenience.
