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
Booleans are `true` or `false`. A non-negative integer uses exactly the JSON
number lexeme `0` or `[1-9][0-9]*`; a sign, leading zero, decimal point,
fraction, or exponent is invalid even when a generic JSON parser would produce
the same mathematical value. Nullable values use JSON `null`. Unknown object
members and unlisted enum strings are invalid. IDs are complete 64-character
lower-hex strings.

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
are full IDs. `content_bytes` is the exact non-negative byte length of the
referenced content Blob payload, excluding its loose-object envelope. `before`
and `after` have the exact type named by their containing field and may be null
only where specified below. `cause_link_change.before` and `.after` are
nullable exact Cause Link records; at least one is non-null and equal records
are omitted.

`attachment.name`, `attachment.media_type`, and every message are strings with
ADR 0025's exact validity rules. Attachment arrays, Cause Link arrays,
previous-revision arrays, message arrays, and assertion arrays are always
present. `seal_view` contains full IDs, a non-negative `content_bytes`, exact
attachment and Cause Link arrays, and boolean `root` and `draft` fields.
`candidate_view.ref` is a REF; `expected_ref_head` is a full SealID or null;
its three prospective IDs and `content_blob_id` are full IDs; its remaining
fields have the same types as `seal_view`. Successful inspection never uses a
null prospective ID as a placeholder for invalid Candidate state.

`assertion_source.observer_seal` and `.target_seal` are full SealIDs;
`.observer_provenance` is a full ProvenanceID; its previous-revision member is
the exact sorted duplicate-free full-SealID array from that assertion, and its
messages member is the exact canonical string array. A
`target_revision_observation.assertions` member is an `assertion_source` and
contains every repository-wide assertion targeting its `target_seal`. A
`scoped_target_revision_observation.in_scope_assertions` member is the same
record type but contains every and only assertion admitted by the containing
command's declared assertion scope. Each observation's state and structural
arrays are derived from exactly its included assertion array.

Every `revision_edge` names two full SealIDs. Its `assertion_sources` is an
array of `assertion_source` records containing every and only included source
that asserts that exact edge: all observed supporting sources in an unfiltered
document, or all and only in-scope supporting sources in a document with an
explicit assertion scope. It never includes an empty or different assertion.

For every `change`, `changed` is true if and only if `before` and `after` differ
as their exact typed canonical JSON values; JSON `null` differs from every
non-null value. Array equality includes order and every nested member. Thus
equal values with `changed = true` and unequal values with `changed = false`
are both invalid.

Repository-wide `previous_states` uses exactly
`PREVIOUS_UNOBSERVED`, `PREVIOUS_NONE_ASSERTED`, and `PREVIOUS_ASSERTED` in
that order under ADR 0023's exclusivity rules. Filter-relative
`in_scope_previous_states` uses exactly `ASSERTION_SCOPE_UNOBSERVED`,
`ASSERTION_SCOPE_NONE_ASSERTED`, and `ASSERTION_SCOPE_ASSERTED` in that order.
`structural_previous_seals` is the sorted union represented by the included
assertions.

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

It is the following total function of
`candidate.expected_ref_head` (`expected`) and `current_ref_head` (`current`).
No ancestry or assertion relation participates:

| `expected` | `current` | Required state |
| --- | --- | --- |
| null | null | `EXPECTED_ABSENT` |
| full ID `E` | the same full ID `E` | `EXPECTED_CURRENT` |
| full ID | null | `HEAD_MISSING` |
| null | full ID | `UNEXPECTED_HEAD` |
| full ID `E` | a different full ID `C` | `HEAD_ADVANCED` |

`baseline.label` is exactly `PUBLICATION_BASELINE`. Its `state` is `PRESENT`
or `ABSENT`. `PRESENT` requires one complete `seal_view`; `ABSENT` requires
`seal = null`. No empty material or previous revision is invented.
`baseline` is `ABSENT` exactly when `candidate.expected_ref_head` is null;
otherwise it is `PRESENT` and loads that exact immutable Seal even when the
current head is missing or different. `candidate-compare.ref` equals
`prospective.ref`, and `prospective` is the exact projection of `candidate`.
`current_ref_head` and `expected_head_state` have the same domains as
`candidate-show/v2` and report publication concurrency separately from the
immutable baseline/prospective comparison.

Each member of `changes` is one `change`. For Candidate comparison, `before`
is null for every member when the baseline is absent; otherwise it has the
baseline field's exact type. `after` always has the prospective field's exact
type. For immutable comparison both sides are non-null. `attachments` and
`cause_links` compare the complete canonical arrays under the shared
if-and-only-if `changed` rule.

Within `changes`, `material_id`, `provenance_id`, and `content_blob_id` compare
full IDs; `attachments` compares complete `attachment` arrays; `root` and
`draft` compare booleans; and `cause_links` compares complete `cause_link`
arrays. `compare/v2` contains no revision observation, REF alias, assertion
source, or other derived graph field.

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
stale_status_record:
  ref, head_seal_id, draft, stale, sealed_state_labels
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

Every `status/v3.statuses` member is exactly one `status_record`; every
`stale/v2.statuses` member is exactly one `stale_status_record`. Both arrays use
REF order. For `status/v3`, membership is the union of current REF-manifest
names, Candidate names, and local source-binding REF names; an explicit
`status REF` selects exactly that member or fails when the union does not
contain it. For `stale/v2`, membership is only current REF heads having at least
one self, direct, or transitive stale fact, optionally reduced from ADR 0023's
exact `S_O` to its exact-Seal `F_O` frontier. `frontier` and `scan` are booleans
recording the requested modes.

For a `status_record`, `head_seal_id` is null exactly when the named REF has no
current manifest. `candidate_to_head` is `UNSEALED` exactly when a Candidate
exists and otherwise `NO_CANDIDATE`. Its `draft` is true exactly when the
present Candidate is draft, the present current head is draft, or both. For a
`stale_status_record`, `head_seal_id` is always the selected current head and
`draft` is exactly that head's draft flag. In both records, `stale` is always a
`stale_record`, never null: `self` is a boolean; `direct_target_seal_ids` and
`transitive_paths` are always arrays. With no current head in a `status_record`,
`self` is false and both arrays are empty.

Every `transitive_paths` member is a non-empty full-SealID array. Its first
member is a direct Cause target and its last member is the first non-active-leaf
target on that Cause route.

For a current head `H`, `self` is true exactly when `H` is an active non-leaf.
`direct_target_seal_ids` is the sorted duplicate-free set of exact direct Cause
targets of `H` that are not active leaves. One `transitive_paths` member is
emitted for every distinct first-stale Cause path `[D_0, ..., D_n]`, where
`D_0` is a direct Cause target of `H`, `n >= 1`, every `D_0` through
`D_(n-1)` is an active leaf, every adjacent pair is a Cause edge, and `D_n` is
not an active leaf. The same first-stale target may occur through more than one
path. Paths use common path order. These paths are emitted only after direct
classification: when `direct_target_seal_ids` is non-empty,
`transitive_paths` is empty. Self-stale is independent and may coexist with
either Cause classification.

For both record types, `sealed_state_labels` is never empty and its members
occur only in this order: `SEALED_STATE_CLEAN`, `UNSEALED`, `DRAFT`,
`STALE_SELF`, `STALE_DIRECT`, `STALE_TRANSITIVE`. In `status_record`,
`UNSEALED` is present exactly for a Candidate. It is never present in
`stale_status_record`. `DRAFT` matches that record's `draft`; each stale label
is present exactly when its corresponding boolean or array is non-empty.
`SEALED_STATE_CLEAN` is the sole member exactly when none of the other labels
permitted for that record applies. Because stale output membership requires a
stale fact, it never emits `SEALED_STATE_CLEAN`.

`local_source` is null exactly when no binding exists. Otherwise its `path` is
ADR 0019's exact local relative path and `baseline` is exactly `CANDIDATE` when
a Candidate exists, `HEAD` when no Candidate and a current head exists, or
`NONE` when neither exists. Its `relation` is exactly
`WORKFILE_MATCHES_CANDIDATE`, `WORKFILE_DIFFERS_FROM_CANDIDATE`,
`WORKFILE_MATCHES_HEAD`, `WORKFILE_DIFFERS_FROM_HEAD`,
`WORKFILE_DIFFERS_FROM_NONE`, `SOURCE_MISSING`, or `SOURCE_UNREADABLE`, and the
named baseline suffix must equal `baseline`. A missing or unreadable source has
the corresponding relation independently of baseline. Local source remains a
non-canonical observation.

`stale` never reads Candidate or source-binding state. Its separate record has
no Candidate or source member and always contains at least one stale label/fact.
For `status`, canonical head and stale facts use ADR 0023's coherent `O`, while
Candidate, binding, and workfile relations retain ADR 0019's stable per-file
observation contract. They do not claim one atomic filesystem-wide snapshot and
do not enter `O` or canonical graph identity.

Graph `nodes` is exactly `V_O` from ADR 0023, ordered by full SealID. A node's
`revision_state` is `ACTIVE_LEAF` when it is a leaf in `A_O`,
`ACTIVE_NON_LEAF` when it is a non-leaf in `A_O`, and
`HISTORICAL_OR_DETACHED` exactly when it is in `V_O` but not `A_O`. Its `refs`
is the sorted set of current REFs whose HEAD is that Seal. `causes` contains
one `graph_cause` per exact direct Cause Link target, ordered by target SealID;
each cause state is derived by the same `A_O` membership/leaf rule. Every graph
node carries the complete revision observation for its own Seal, so structural
union and mixed assertion sources remain visible. A valid physical Seal outside
`V_O` belongs only to complete `fsck` inventory, not graph output.

`graph_node.revision_observation` is a `target_revision_observation` whose
`target_seal` equals that node's `seal_id`. Its `causes` is an array of
`graph_cause`; each cause carries one full target SealID and the target's exact
revision-state enum.

For successful `fsck/v2`, let:

```text
B_F = every distinct valid physical BlobID in the object-namespace portion of
      ADR 0023's captured P_O
S_F = IDs in B_F that validate as format-5 Seal Blobs
M_F = IDs in B_F that validate as Material Blobs
P_F = IDs in B_F that validate as Provenance Blobs
R_F = the least BlobID set rooted at every S_F and closed over every typed
      Seal, Material, Provenance, content, attachment, Cause-target, and
      previous-revision reference
```

One BlobID may belong to more than one typed-role set when the exact bytes
validly satisfy those roles. `fsck.result` is exactly `ok`. The seven
non-negative counts are `|B_F|`, `|S_F|`, `|M_F|`, `|P_F|`, the number of REF
manifests in `O`, the total number of scoped tag records in those manifests,
and `|A_O|`, respectively. `historical_or_detached_seal_ids` is exactly the
sorted members of `S_F` that are not in `A_O`. `unreferenced_blob_ids` is
exactly the sorted members of `B_F` that are not in `R_F`; every Seal is
therefore classified through active versus historical/detached rather than
also being called an unreferenced Blob.

The seven count members are non-negative integers. Both trailing ID arrays are
sorted duplicate-free full-BlobID arrays; every member of
`historical_or_detached_seal_ids` is additionally a valid SealID.

Every physical canonical config/object/REF entry, Blob, typed role, reference,
REF, tag, and combined graph must validate under ADR 0023's complete `O`/`P_O`
capture, ADR 0025's mode-neutral integrity boundary, and revalidation. Entry
kind, readability, layout, and bytes are validity conditions; a mode difference
between captures is a concurrent-observation failure, but stable permission
bits alone are not corruption.
Operational, integrity, or concurrent-observation failure remains stderr plus
nonzero exit and emits no plausible success document.

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
at least two IDs. `revision_proof.seal_ids` begins with `source_seal_id` and ends
with the matched revision. Its `edges` are the ordered exact adjacent revision
edges. `target_observations` has exactly one scoped observation for each Seal in
`seal_ids`, in that same source-to-match sequence order. The validated acyclic
proof path contains no duplicate Seal. A direct match has one proof SealID, its
one source observation, and no edge.

Every `impact_path.revision_proof` is a `revision_proof`; its `edges` members
are `revision_edge` records in proof sequence and its `target_observations`
members are `scoped_target_revision_observation` records. Each edge's assertion
sources and each observation use the top-level `assertion_scope`.

For one impact let `P_d` be its complete first-match Cause-path set in common
path order. Default output contains exactly the first member of `P_d` and sets
`paths_truncated` to `|P_d| > 1`. With `--all-paths` and bound `N`, `paths`
contains exactly the first `min(N, |P_d|)` members and `paths_truncated` is true
if and only if `|P_d| > N`. Limits do not change membership or validation.

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

For both `log/v2` and `linklog/v2`, `ref` is the requested valid current REF and
`head_seal_id` is its non-null exact head. `minimum_depth` and
`minimum_newer_depth` are non-negative integers.

Every `log_entry.seal` is a `seal_view`; `revision_observation` is a
`target_revision_observation` for that Seal; and `outgoing_revision_edges` is a
previous-SealID-ordered `revision_edge` array. Every `log_path.edges` member is
the exact `revision_edge` for its adjacent SealID pair, in path sequence.

With `--all-paths`, `max_paths` is a positive integer `N` defaulting to 100. Let
`P_H` be the complete ADR 0023 maximal leaf-terminated path set from the head in
common path order. `paths` contains exactly the first `min(N, |P_H|)` members of
`P_H`, and `paths_truncated` is true if and only if `|P_H| > N`. Each path
contains its ordered SealIDs and exact adjacent edges. A leaf head emits the one
path `[H]` with an empty `edges` array; a linear `H -> P -> Q` history emits
only `[H,P,Q]`, not its prefixes; and a branch `H -> P`, `H -> Q` with leaf `P`
and `Q` emits exactly those two complete paths in common path order. Complete
entry membership, graph validation, and snapshot revalidation are independent
of the path bound.

The public history command is:

```text
sealgraph log [--all-paths] [--max-paths N] REF
```

`--max-paths` is invalid without `--all-paths`, must be positive, and defaults
to 100 when all paths are requested.

The public Cause-history command is:

```text
sealgraph linklog [--upstream TARGET_SELECTOR] REF
```

`--upstream` is optional and may occur once. `TARGET_SELECTOR` uses ADR 0023's
selector grammar and resolves under the same coherent observation as the
history. Missing, ambiguous, or invalid selection fails without partial output.
`linklog.upstream_seal_id` is null when omitted and otherwise is the resolved
full ID.

Without the filter, each structural revision-edge entry contains every changed
Cause Link target. With the filter, an entry retains in its `changes` array only
the member whose `target_seal` equals `upstream_seal_id` and is omitted entirely
when that change is absent. The filter does not change structural edge
membership, minimum depth, supporting assertions, or target revision
observation before the final empty-entry omission. A resolved target with no
change on any edge succeeds with an empty `entries` array.

Retained entries use ADR 0023 `(minimum_newer_depth, newer SealID, previous
SealID)` order.
`supporting_assertions` is the complete assertion-source array for that
structural edge; `target_revision_observation` is the repository-wide
observation for the newer Seal; and `changes` is a target-SealID-ordered
`cause_link_change` array. It never infers a repoint.

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
- ADR 0025 remains authoritative for Candidate and typed-Blob bytes and for the
  mode-neutral physical integrity boundary.
- ADR 0026 remains authoritative for format selection and one-way migration.

ADRs 0023, 0025, 0026, and 0027 must be reviewed as one exact decision set and
accepted by the operator before any format-5 implementation action or
normative-document conversion. Action tables in all four ADRs repeat that same
execution prerequisite; narrower Claim ownership never authorizes an earlier
slice.

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
| A-IO-001 implement shared format-5 JSON records | ADRs 0023, 0025, 0026, and 0027 | C-IO-001 through C-IO-003 | fixed complete bytes for every successor, null/empty/enum rejection, expected-head truth-table, separate status/stale records, direct/transitive exclusivity, exact-Seal frontier alias/self-stale/branch fixtures, exact graph/fsck set-population, stable writable-mode acceptance, concurrent mode-change rejection, full-ID, and order fixtures, and proof that no format-5 command emits an incompatible old schema |
| A-IO-002 implement branching history output | ADRs 0023, 0025, 0026, and 0027 | C-IO-002, C-IO-004 | branch, shared-ancestor, mixed-observer, default unique-entry, maximal leaf-path including zero-edge, exact ordered-prefix bounds, proof-observation order, target-filter, no-match, truncation, and snapshot-change fixtures |
| A-IO-003 implement one-target authoring | ADRs 0023, 0025, 0026, and 0027 | C-IO-005 | add/set/replace/remove, selector-resolution, repeat ordering, no-previous, root/non-root, ambiguity rejection, and exact Candidate-byte fixtures |
| A-IO-004 synchronize normative/public contracts | ADRs 0023, 0025, 0026, and 0027 | C-IO-001 through C-IO-006 | requirements, architecture, storage-format, CLI, integrations, index, help, completion, and cross-reference checks in the accepted implementation candidate |

No implementation action starts from this Proposed ADR alone or from a proper
subset of the four accepted ADRs.

## Review

The three-scope review of ADRs 0023, 0025, and 0026 at digests recorded in the
2026-08-28 decision/review record found that format-5 semantics had no complete
machine-schema transition, branching `log` had no bounded normal form, and
Cause Link CLI grouping was ambiguous. This ADR supplied the first proposed
correction.

The 2026-08-31 three-scope review of exact ADR 0027 digest
`5b4d6a9826c6aab7820703a56696802b50abbe5128826a799ebd08ce3b541a95`
then found incomplete `status/v3`, `stale/v2`, `graph/v2`, and `fsck/v2`
populations and types; no physical `fsck` observation transaction; ambiguous
maximal-path and `linklog` filter membership; non-total Candidate state/change
derivations; and conflicting action-specific versus four-ADR gates. This
candidate defines every affected population, null/enum invariant, set equation,
path terminal, filter rule, truth table, and shared gate. It still requires a
fresh review of its new exact bytes and remains Proposed.

The next three-scope review of exact ADR 0027 digest
`ea62105ee6fa0cd3dc4bc50c5c02747d877e6ef2e356c1856fc31b5a4e96debc`
found that bounded path membership was only an upper bound, direct and
transitive stale labels contradicted retained semantics, the frontier cited a
superseded predicate, status and stale reused one record with incompatible
observation rules, comparison promised an absent graph projection, physical
fsck coverage omitted REF/config metadata, proof-observation order was missing,
and the shared gate used two terms. This candidate defines ordered-prefix
functions, restores stale exclusivity and the exact-Seal frontier, separates
status/stale records, aligns compare with its immutable projection, uses the
complete physical canonical observation, orders proof observations, and uses
one exact gate sentence. These corrections remain Proposed.

The primary-agent pre-delegation review then enumerated every top-level and
nested record, null/empty case, enum, array order, numeric boundary, path bound,
status/stale combination, and format-4 precedence boundary. It found and fixed
implicit nested-record typing, non-exact general integer wording, the accepted
non-atomic status-local-observation boundary, and a stale commit-dependent
review follow-up before digest freeze. Its action-to-Claim reverse comparison
also made A-IO-004's normative synchronization ownership explicit for every
claim named by that action.

The next three-scope review of exact ADR 0027 digest
`3f7181baa58b88b2a9c7fab5882a68c6fb9cb6abce07a716bb1eec9db1126b69`
confirmed the repaired output schemas but found that its `fsck/v2` validity
sentence imported exact permission bits contrary to Accepted ADR 0016 and the
ordinary-file Git-sidecar boundary. This candidate cites ADR 0025's explicit
mode-neutral integrity Claim and distinguishes stable permission values from
mode changes during observation. It remains Proposed.

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
| C-IO-002 | E-IO-001, E-IO-003, E-IO-005 | A-IO-001, A-IO-002, A-IO-004 |
| C-IO-003 | E-IO-002, E-IO-003, E-IO-004 | A-IO-001, A-IO-004 |
| C-IO-004 | E-IO-003, E-IO-005 | A-IO-002, A-IO-004 |
| C-IO-005 | E-IO-003, E-IO-005, E-IO-006 | A-IO-003, A-IO-004 |
| C-IO-006 | E-IO-001, E-IO-003, E-IO-006 | A-IO-004 |

## Follow-ups

- Freeze the exact candidate bytes and run the same three-scope read-only review
  over ADRs 0023, 0025, 0026, and 0027 before owner acceptance.
- After acceptance, replace the format-4 normative/help schema text and add
  fixed format-5 JSON fixture bytes.
- Add a machine diagnostic schema only under a separate consumer-driven ADR.
