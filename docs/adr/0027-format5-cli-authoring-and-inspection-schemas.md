# ADR 0027: Format-5 CLI authoring and inspection schemas

- Status: Proposed
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-IO-001 through C-IO-006
- Related Evidence: E-IO-001 through E-IO-006
- Pending Decision: owner acceptance of the exact candidate after fresh review
- Supersedes on acceptance: for format-5 repositories only, the format-4 Link
  option grouping and parent-based inspection schema meanings in ADRs 0015,
  0020, and 0022
- Superseded by: None

## Context

ADRs 0023 and 0025 remove intrinsic `parent_revision`, make revision history a
branching observed graph, and make format-5 Candidates parentless. Existing
versioned inspection JSON exposes `parent_revision`, linear history, and a
Candidate comparison against a recorded parent. Reusing those schema names for
format 5 would silently change public machine meaning.

The format-5 Cause Link also contains one exact target, a previous-revision
set, and a message set. The format-4 multi-target `--depend-on` invocation and
single shared message cannot express those three arrays without ambiguous
positional grouping or cross-product behavior.

This ADR is the public-interface member of the Proposed format-5 decision set.
ADR 0023 owns graph semantics, ADR 0025 owns Blob and Candidate bytes, ADR 0026
owns migration, and this ADR owns exact format-5 authoring grammar and
successful inspection JSON.

## Decision

### Runtime and schema-version boundary

The format-4 runtime continues to emit its accepted schemas. The format-5
runtime emits only the successor schema in this matrix; it never emits a v1
document under changed format-5 meaning and performs no content negotiation.

| Command | Format 4 | Format 5 |
| --- | --- | --- |
| `show` | `sealgraph/show/v1` | `sealgraph/show/v2` |
| `candidate show` | `sealgraph/candidate-show/v1` | `sealgraph/candidate-show/v2` |
| `candidate compare` | `sealgraph/candidate-compare/v1` | `sealgraph/candidate-compare/v2` |
| `status` | `sealgraph/status/v2` | `sealgraph/status/v3` |
| `stale` | `sealgraph/stale/v1` | `sealgraph/stale/v2` |
| `graph` | `sealgraph/graph/v1` | `sealgraph/graph/v2` |
| `impact` | `sealgraph/impact/v1` | `sealgraph/impact/v2` |
| `log` | `sealgraph/log/v1` | `sealgraph/log/v2` |
| `linklog` | `sealgraph/linklog/v1` | `sealgraph/linklog/v2` |
| `compare` | `sealgraph/compare/v1` | `sealgraph/compare/v2` |
| `fsck` | `sealgraph/fsck/v1` | `sealgraph/fsck/v2` |

`source`, `source compare`, and `recover` retain their accepted v1 schemas
because their local binding, content-byte comparison, and recovery-journal
meanings do not expose an intrinsic parent or branching revision projection.
Mutation narrow receipts, `stale --refs-only`, `show --raw-content`, and
`candidate show --raw-content` retain their exact accepted protocols. ADR
0022's terminal/non-terminal selection and explicit-format precedence remain.

This establishes C-IO-001: a schema identifier has one meaning within one
runtime format and every incompatible format-5 inspection has an explicit
successor.

### Stable JSON encoding and common value rules

Every format-5 inspection document below is compact UTF-8 JSON followed by
exactly one LF. Object members use the exact displayed order. Strings use ADR
0025's canonical scalar escaping; Unicode normalization is not applied.
Booleans are `true` or `false`; non-negative integers use shortest base-10
notation; nullable values use JSON `null`. Unknown members and enum strings are
not part of the schema. IDs are complete 64-character lower-hex strings.

All arrays are present even when empty. Unless a more specific order is stated:

- ID sets use full-ID byte order;
- REF and string sets use raw UTF-8 byte order;
- Cause Links use ADR 0025 canonical order;
- assertion sources use ADR 0023 tuple order; and
- paths use `(edge_count, full SealID sequence)` order.

For each command, `schema` is exactly the format-5 value in the matrix above.

The following shared records have exact member order:

```text
attachment:
  name, media_type, blob

cause_link:
  target_seal, previous_revision_seal_of_target_seal, messages

assertion_source:
  observer_seal, observer_provenance, target_seal,
  previous_revision_seal_of_target_seal, messages

target_revision_observation:
  target_seal, previous_states, assertions, structural_previous_seals

scoped_target_revision_observation:
  target_seal, in_scope_previous_states, in_scope_assertions,
  structural_previous_seals

revision_edge:
  target_seal, previous_revision_seal, assertion_sources

seal_view:
  seal_id, material_id, provenance_id, content_blob_id, content_bytes,
  attachments, root, draft, cause_links

candidate_view:
  ref, expected_ref_head, prospective_seal_id, prospective_material_id,
  prospective_provenance_id, content_blob_id, content_bytes, attachments,
  root, draft, cause_links

change:
  changed, before, after

cause_link_change:
  target_seal, before, after
```

`attachment.blob`, every `*_id`, every Seal reference, and every path element
are full IDs. `content_bytes` is a non-negative integer. `before` and `after`
have the exact type named by their containing field and may be null only where
specified below. `cause_link_change.before` and `.after` are nullable exact
Cause Link records; at least one is non-null and equal records are omitted.

Repository-wide `previous_states` uses exactly
`PREVIOUS_UNOBSERVED`, `PREVIOUS_NONE_ASSERTED`, and `PREVIOUS_ASSERTED` in
that order under ADR 0023's exclusivity rules. Filter-relative
`in_scope_previous_states` uses exactly `ASSERTION_SCOPE_UNOBSERVED`,
`ASSERTION_SCOPE_NONE_ASSERTED`, and `ASSERTION_SCOPE_ASSERTED` in that order.
`structural_previous_seals` is the sorted union represented by the included
assertions. `revision_edge.assertion_sources` contains every and only source
supporting that exact edge.

This establishes C-IO-002: shared graph and typed-Blob values have one exact
machine representation rather than command-specific approximations.

### Exact single-Seal and comparison documents

The required top-level and nested member orders are:

```text
show/v2:
  schema, seal, current_refs, revision_observation

candidate-show/v2:
  schema, candidate, current_ref_head, expected_head_state

candidate-compare/v2:
  schema, ref, current_ref_head, expected_head_state,
  baseline, prospective, changes
baseline:
  label, state, seal
changes:
  material_id, provenance_id, content_blob_id, attachments, root, draft,
  cause_links

compare/v2:
  schema, from, to, changes
```

`show.seal`, `baseline.seal`, `compare.from`, and `compare.to` are `seal_view`
records. `candidate` and `prospective` are `candidate_view` records.
`current_refs` is a sorted duplicate-free REF array.
`revision_observation` targets `show.seal.seal_id`.

`current_ref_head` is a full SealID or null. `expected_head_state` is exactly
`EXPECTED_ABSENT`, `EXPECTED_CURRENT`, `HEAD_ADVANCED`, `HEAD_MISSING`, or
`UNEXPECTED_HEAD`.

`baseline.label` is exactly `PUBLICATION_BASELINE`. Its `state` is `PRESENT`
or `ABSENT`. `PRESENT` requires one complete `seal_view`; `ABSENT` requires
`seal = null`. No empty material or previous revision is invented.
`current_ref_head` and `expected_head_state` have the same domains as
`candidate-show/v2` and report publication concurrency separately from the
immutable baseline/prospective comparison.

Each member of `changes` is one `change`. For Candidate comparison, `before`
is null for every member when the baseline is absent; otherwise it has the
baseline field's exact type. `after` always has the prospective field's exact
type. For immutable comparison both sides are non-null. `attachments` and
`cause_links` compare the complete canonical arrays. A `changed = false`
record requires exact equality of `before` and `after`.

This establishes C-IO-003: Candidate comparison is against only the explicit
publication baseline, and immutable comparison never infers a predecessor.

### Exact status, stale, graph, and fsck documents

The required member orders are:

```text
status/v3:
  schema, statuses

stale/v2:
  schema, frontier, scan, statuses

status_record:
  ref, head_seal_id, candidate_to_head, draft, stale,
  sealed_state_labels, local_source
stale_record:
  self, direct_target_seal_ids, transitive_paths
local_source:
  path, baseline, relation

graph/v2:
  schema, nodes
graph_node:
  seal_id, revision_state, refs, revision_observation, causes
graph_cause:
  target_seal_id, revision_state

fsck/v2:
  schema, result, blobs, seals, materials, provenances, refs, tags,
  active_seals, historical_or_detached_seal_ids, unreferenced_blob_ids
```

`statuses` and graph `nodes` use REF and full SealID order respectively.
`head_seal_id` and `local_source` may be null. `candidate_to_head` is
`NO_CANDIDATE` or `UNSEALED`. `sealed_state_labels` is the fixed-order subset
of `SEALED_STATE_CLEAN`, `UNSEALED`, `DRAFT`, `STALE_SELF`, `STALE_DIRECT`,
and `STALE_TRANSITIVE`. `local_source` retains ADR 0019's exact local relative
path and baseline relation; it remains non-canonical observation.

`local_source.baseline` is exactly `CANDIDATE`, `HEAD`, or `NONE`.
`local_source.relation` is exactly `WORKFILE_MATCHES_CANDIDATE`,
`WORKFILE_DIFFERS_FROM_CANDIDATE`, `WORKFILE_MATCHES_HEAD`,
`WORKFILE_DIFFERS_FROM_HEAD`, `WORKFILE_DIFFERS_FROM_NONE`, `SOURCE_MISSING`,
or `SOURCE_UNREADABLE`, consistent with the named baseline.

`direct_target_seal_ids` is sorted. `transitive_paths` uses path order.
`frontier` and `scan` are booleans. `stale` never reads Candidate state, so
each stale status record has `candidate_to_head = NO_CANDIDATE` and
`local_source = null`.

`revision_state` is exactly `ACTIVE_LEAF`, `ACTIVE_NON_LEAF`, or
`HISTORICAL_OR_DETACHED`. Graph `refs` is sorted. Causes are ordered by target
SealID. Every graph node carries the complete revision observation for that
target, so structural union and mixed assertion sources are both visible.

`fsck.result` is exactly `ok` on successful JSON output. The seven count fields
`blobs`, `seals`, `materials`, `provenances`, `refs`, `tags`, and
`active_seals` are non-negative integers. The two ID inventories are sorted
and duplicate-free. `blobs` counts distinct physical valid loose BlobIDs;
`seals`, `materials`, and `provenances` count distinct IDs valid in that typed
role; `refs`, `tags`, and `active_seals` count their distinct validated
records/IDs. Operational or integrity failure remains stderr plus nonzero exit
and does not emit a plausible success document.

### Exact impact document

The exact shapes are:

```text
impact/v2:
  schema, source_seal_id, assertion_scope,
  asserted_by_observer_seal_ids, all_paths, max_paths, impacts

impact_record:
  head_seal_id, refs, paths, paths_truncated

impact_path:
  cause_seal_ids, matched_revision_seal_id, revision_proof

revision_proof:
  seal_ids, edges, target_observations
```

`assertion_scope` is `ALL_OBSERVED` or `FILTERED`. The all-observed form
requires an empty observer array; the filtered form requires the sorted
non-empty resolved observer SealID set. `max_paths` is null when `all_paths` is
false and is the positive requested/default bound when true.

Impacts use head SealID order and `refs` uses REF order. `cause_seal_ids` begins
with the downstream head and ends with `matched_revision_seal_id`; it contains
at least two IDs. `revision_proof.seal_ids` begins with `source_seal_id` and
ends with the matched revision. Its `edges` are the ordered exact adjacent
revision edges and its `target_observations` contains the scoped observation
for every distinct proof target, including the zero-edge source. A direct
match has one proof SealID and no edge.

Default output contains exactly one shortest Cause path per impact.
`--all-paths` contains at most `max_paths` paths per impact in path order.
`paths_truncated` is true exactly when another valid first-match path exists.
Limits do not change membership or validation.

### Exact log and linklog documents

The exact shapes are:

```text
log/v2:
  schema, ref, head_seal_id, all_paths, max_paths,
  entries, paths, paths_truncated

log_entry:
  minimum_depth, seal, revision_observation, outgoing_revision_edges

log_path:
  seal_ids, edges

linklog/v2:
  schema, ref, head_seal_id, upstream_seal_id, entries

linklog_entry:
  minimum_newer_depth, newer_seal_id, previous_seal_id,
  supporting_assertions, target_revision_observation, changes
```

Default `log` emits every reachable Seal once in ADR 0023
`(minimum_depth, full SealID)` order. `outgoing_revision_edges` uses previous
SealID order. Default `all_paths` is false, `max_paths` is null, `paths` is
empty, and `paths_truncated` is false.

With `--all-paths`, `max_paths` is a positive integer defaulting to 100 and
`paths` contains at most that many distinct simple paths from the head in path
order. Each path contains its ordered SealIDs and exact adjacent edges.
`paths_truncated` is true exactly when another valid path exists. Complete
entry membership, graph validation, and snapshot revalidation are independent
of the path bound.

The public history command is:

```text
sealgraph log [--all-paths] [--max-paths N] REF
```

`--max-paths` is invalid without `--all-paths`, must be positive, and defaults
to 100 when all paths are requested.

`linklog.upstream_seal_id` is null without the optional exact Cause-target
filter and otherwise is its resolved full ID. Entries use ADR 0023
`(minimum_newer_depth, newer SealID, previous SealID)` order.
`supporting_assertions` contains every source of that structural edge;
`target_revision_observation` contains all observed assertions for the newer
Seal. `changes` contains Cause Link changes ordered by target SealID. It never
infers a repoint.

This establishes C-IO-004: branching history has bounded deterministic
machine output, shared-node handling, and complete assertion evidence.

### Format-5 Cause Link authoring grammar

Format 5 uses one-target whole-record operations:

```text
CAUSE_GROUP := --target TARGET PREVIOUS_CHOICE MESSAGE_INPUT...
PREVIOUS_CHOICE := --no-previous
                 | --previous PREVIOUS [--previous PREVIOUS]...
MESSAGE_INPUT := -m MESSAGE
ROOT_MODE := --root --clear-cause-links | --non-root

sealgraph add REF ADD_CONTENT_SOURCE_DRAFT_OPTIONS [ROOT_MODE] [CAUSE_GROUP]

sealgraph link REF --target TARGET
  (--previous PREVIOUS ... | --no-previous)
  [-m MESSAGE ...]

sealgraph unlink REF --target TARGET
```

`ADD_CONTENT_SOURCE_DRAFT_OPTIONS` is not a literal token. It denotes the
accepted `add` content, content-file, source-binding, and draft options in
`docs/cli.md`; format 5 removes `--parent`, `--depend-on`, and the old implicit
root default. The `ROOT_MODE` and `CAUSE_GROUP` productions above are the only
format-5 root/Cause input grouping.

`--target` occurs exactly once when present. `--previous` and `-m` are
repeatable. Exactly one of at least one `--previous` or `--no-previous` is
required for a target. Messages may be omitted and then form the explicit
empty array. All selectors resolve against one ADR 0023 observation before
candidate persistence.

`link` replaces the complete Cause Link record for the exact resolved target,
or creates it when absent. It preserves content, attachments, root/draft,
publication expectation, and every other Cause Link. `unlink` removes exactly
that target record and changes nothing else. Neither command may leave an
invalid root-with-Cause or non-root-without-Cause Candidate.

For a newly created non-root Candidate, `add` requires one complete target
group and explicit `--non-root` in the invocation. A new root requires
`--root --clear-cause-links` and forbids a target group. For an existing edit
baseline, omitted `ROOT_MODE` preserves root and omitted target group preserves
the complete Cause Link array; a supplied group sets that one whole record.
`--root --clear-cause-links` explicitly and atomically sets root and removes
every Cause Link. `--non-root` explicitly sets non-root and must leave at least
one Cause Link. This makes root transitions possible without an invalid
intermediate Candidate or an automatic Link edit. There is no positional association,
multi-target invocation, message cross-product, union-with-existing record,
implicit current-parent assertion, or format-4 `--depend-on` compatibility
alias. Multiple targets require reviewed separate candidate mutations.

`derive` and `add --parent` are absent. A later copy-only seed command requires
its own explicit CLI decision and creates no revision assertion automatically.

This establishes C-IO-005: every successful authoring invocation maps to one
deterministic Candidate transition and one exact observer-local assertion.

### Decision precedence and acceptance boundary

On acceptance:

- ADR 0015 retains safety, complete IDs, result/diagnostic exit separation,
  and human terminology; this ADR replaces only incompatible format-5 JSON
  schema versions and structures.
- ADR 0019 retains local source baselines, relations, and explicit relative
  path inspection; `status/v3` carries that orthogonal local observation
  without putting it in canonical identity.
- ADR 0020 retains the native `compare`, `candidate compare`, `source compare`,
  `link`, and `unlink` vocabulary. Its one-selector/parent comparison and
  format-4 multi-target Link grouping are replaced for format 5.
- ADR 0022 retains output destination selection, exact byte modes, narrow
  receipts, and human presentation; this ADR supplies format-5 successor
  documents instead of silently changing an existing schema.
- ADR 0023 remains authoritative for observation, revision, impact, history,
  and comparison meaning.
- ADR 0025 remains authoritative for Candidate and typed-Blob bytes.
- ADR 0026 remains authoritative for format selection and one-way migration.

This ADR is Proposed and does not authorize CLI implementation. ADRs 0023,
0025, 0026, and 0027 must be reviewed as one exact decision set and accepted by
the operator before the format-5 runtime or normative-document conversion.

This establishes C-IO-006: semantic, storage, migration, and public-interface
authority remain separate but share one acceptance gate.

## Alternatives

### Reuse v1 schema identifiers with changed fields

Rejected because automation could not distinguish intrinsic-parent output from
Cause-scoped revision output.

### Bump only impact and linklog

Rejected because show, Candidate inspection, compare, graph, log, status,
stale, and fsck also expose or derive parent-based facts.

### Group multiple targets and previous revisions by option position

Rejected because parsers and users could disagree about group boundaries and
produce different Candidate bytes from the same apparent intent.

### Merge repeated Link input into an existing record

Rejected because omission would become an implicit preservation rule for one
part of an identity-bearing assertion. Whole-record replacement makes the
complete new assertion visible before persistence.

### Leave exact JSON shapes to implementation

Rejected because version identifiers would be fixed while independent
implementations remained structurally incompatible.

## Consequences

Good:

- Format and schema identifiers expose the semantic break explicitly.
- Parentless Candidate and comparison output cannot fabricate an old parent.
- Branching histories are deterministic and bounded without hiding graph
  membership.
- One-target Link operations produce unambiguous canonical Candidate bytes.

Bad / Risk:

- Every consumer of affected format-4 JSON must deliberately adopt a format-5
  successor.
- Complete assertion sources and typed views can make inspection documents
  substantially larger.
- Whole-record Link replacement requires the operator to resupply messages
  and previous revisions intentionally.
- All-path history remains combinatorial even though presentation is bounded.

Neutral:

- Human output is presentation and may remain compact under ADR 0022.
- Source-binding and recovery schemas remain unchanged.
- This ADR does not add machine diagnostics or mutation JSON receipts.

## Implementation Notes

| Action | Accepted ADR gate | Claim | Evidence required before completion |
| --- | --- | --- | --- |
| A-IO-001 implement shared format-5 JSON records | ADRs 0023, 0025, 0027 | C-IO-001 through C-IO-003 | fixed complete bytes for every successor, null/empty/enum rejection, full-ID and order fixtures, and proof that no format-5 command emits an incompatible old schema |
| A-IO-002 implement branching history output | ADRs 0023 and 0027 | C-IO-002, C-IO-004 | branch, shared-ancestor, mixed-observer, default unique-entry, bounded all-path, truncation, and snapshot-change fixtures |
| A-IO-003 implement one-target authoring | ADRs 0023, 0025, 0027 | C-IO-005 | add/set/replace/remove, selector-resolution, repeat ordering, no-previous, root/non-root, ambiguity rejection, and exact Candidate-byte fixtures |
| A-IO-004 synchronize normative/public contracts | ADRs 0023, 0025, 0026, 0027 | C-IO-001 through C-IO-006 | requirements, architecture, storage-format, CLI, integrations, index, help, completion, and cross-reference checks in the accepted implementation candidate |

No implementation action starts from this Proposed ADR alone.

## Review

The three-scope review of ADRs 0023, 0025, and 0026 at digests recorded in the
2026-08-28 decision/review record found that format-5 semantics had no complete
machine-schema transition, branching `log` had no bounded normal form, and
Cause Link CLI grouping was ambiguous. This ADR is the proposed correction and
has not yet received an independent review of its exact bytes.

## Evidence

- E-IO-001: accepted ADR 0015 requires a new version for incompatible JSON
  changes.
- E-IO-002: accepted ADR 0022 makes Candidate inspection and existing JSON
  schemas public compatibility contracts.
- E-IO-003: Proposed ADRs 0023 and 0025 remove intrinsic parents and define
  branching revision and parentless Candidate semantics.
- E-IO-004: the current format-4 JSON implementation exposes parent fields in
  show, Candidate inspection, graph, log, linklog, and compare, demonstrating
  that unchanged schema identifiers cannot represent format 5 faithfully.
- E-IO-005: the subsequent three-scope review identified the schema-transition,
  exact-v2-structure, branching-log, and authoring-grouping gaps; its target
  digests and integrated findings are recorded in
  [`../process/cause-scoped-revision-decision-review-2026-08-28.md`](../process/cause-scoped-revision-decision-review-2026-08-28.md).
- E-IO-006: after that review, the operator directed this separated ADR and the
  proposed schema-transition/whole-record correction shape to proceed, recorded
  as D-IO-001. This authorizes the Proposed draft, not acceptance of its exact
  bytes.

Exact traceability is:

| Claim | Evidence | Implementation action |
| --- | --- | --- |
| C-IO-001 | E-IO-001, E-IO-002, E-IO-003, E-IO-004, E-IO-006 | A-IO-001, A-IO-004 |
| C-IO-002 | E-IO-001, E-IO-003, E-IO-005 | A-IO-001, A-IO-002 |
| C-IO-003 | E-IO-002, E-IO-003, E-IO-004 | A-IO-001 |
| C-IO-004 | E-IO-003, E-IO-005 | A-IO-002 |
| C-IO-005 | E-IO-003, E-IO-005, E-IO-006 | A-IO-003 |
| C-IO-006 | E-IO-001, E-IO-003, E-IO-006 | A-IO-004 |

## Follow-ups

- Run the same three-scope read-only review over ADRs 0023, 0025, 0026, and
  0027 after the correction set is committed.
- After acceptance, replace the format-4 normative/help schema text and add
  fixed format-5 JSON fixture bytes.
- Add a machine diagnostic schema only under a separate consumer-driven ADR.
