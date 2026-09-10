# ADR 0031: Format-6 Cause Link metadata CLI and output

Status: Accepted

Accepted: 2026-09-10 by explicit Operator acceptance of the exact reviewed
technical candidate.

Acceptance record:
[`../process/adr-0031-format6-link-metadata-cli-and-output-acceptance-2026-09-10.md`](../process/adr-0031-format6-link-metadata-cli-and-output-acceptance-2026-09-10.md).

Execution authority: Separate. Acceptance does not authorize normative
conversion, implementation, migration, commit, push, release, deployment, or
another external write.

Decision owner: Operator.

Requirement authority: accepted
[ADR 0029](0029-extensible-cause-link-metadata.md) and accepted storage companion
[ADR 0030](0030-format6-link-metadata-storage-and-migration.md).

This ADR owns exact metadata mutation commands, format-5 authoring continuity,
format-6 human inspection, versioned machine schemas, comparison, `linklog`, and
the format-5-to-format-6 migration command and receipt. It does not define a
metadata query language, schema registry, validator runtime, or ADR 0028
Assessment-reference commands. No implementation or normative conversion is
authorized by this acceptance; each requires separate execution authority.

## Context

The format-5 `link` command replaces one complete three-field Cause Link. In
format 6, treating omitted metadata as an empty replacement would let a
messages-only caller delete structured data it never named. Adding repeatable
metadata flags to the same structural group would instead create positional
grouping and shell-quoting ambiguity.

Format-5 inspection shares exact Cause Link and assertion-source records across
`show`, Candidate inspection, comparison, graph, impact, log, and `linklog`.
Metadata changes those nested record meanings and exact output bytes. Reusing
their schema identifiers would violate the accepted rule that one machine schema
has one meaning.

Format 6 also reads exact historical format-5 Seals, Provenances, and Candidates.
Inspection must expose their logical empty metadata projection without claiming
that a `metadata` member existed in historical Provenance v1 or Candidate v5
bytes.

## Decision

### Dedicated one-namespace mutation commands

The public metadata mutation commands are:

```text
sealgraph link-metadata set REF --target TARGET --namespace NAMESPACE
  (--schema SCHEMA | --no-schema)
  (--value-json JSON | --value-file PATH_OR_DASH)
  [--format human|json]

sealgraph link-metadata remove REF --target TARGET --namespace NAMESPACE
  [--format human|json]
```

`REF` is one logical Candidate REF, not a Seal selector. If no Candidate exists,
the command creates the ordinary mutable Candidate baseline from the current
REF head. A missing REF, root baseline, absent exact target Link, corrupt
Candidate, or invalid prospective graph fails without creating or changing a
Candidate.

`--target` occurs exactly once and uses the accepted selector grammar. It
resolves under the operation's coherent repository observation to one full
SealID. Mutation then requires one Cause Link in the Candidate whose stored
`target_seal` equals that exact ID. A bare target REF therefore does not match a
historical Link after its head advances; diagnostics identify the resolved full
ID and direct the operator to inspect the Candidate and use an exact selector.

`--namespace` occurs exactly once and must satisfy ADR 0030. `set` requires
exactly one of `--schema` or `--no-schema`. `--schema` supplies the non-null
identifier; `--no-schema` supplies JSON null. Neither form dereferences or
validates a namespace schema.

`set` requires exactly one value source:

- `--value-json JSON` supplies one complete JSON value as the option argument;
  or
- `--value-file PATH_OR_DASH` reads one complete JSON value from a named regular
  non-symlink file or from stdin when the value is `-`.

Input may use insignificant JSON whitespace and non-canonical but semantically
equivalent string escaping. The command rejects duplicate object keys before
lossy host-map decoding, validates the closed value model and every ADR 0030
limit, canonicalizes the value, and requires no trailing non-whitespace JSON
token. Dynamic selectors, environment expansion, interpolation, and executable
values do not occur. A JSON string that resembles a REF remains only a string.

`set` adds the namespace when absent or replaces its complete schema/value entry
when present. `remove` requires that exact namespace to exist and removes only
that entry. Same-entry `set` succeeds idempotently without changing Candidate
bytes. Removing an absent namespace fails rather than reporting a false change.

Both operations preserve target, previous-revision IDs, `messages`, every other
namespace, content, attachments, root/draft, publication expectation, and every
other Link. They validate and buffer the complete successor Candidate before one
expected-old Candidate-file replacement. Rejected input or concurrent Candidate
change leaves the original bytes unchanged. Neither command creates a Seal,
moves a REF, writes an immutable Blob, performs migration, or interprets metadata
meaning.

### Existing Link authoring continuity

In format 6, existing `add ... --target ...` and `link REF --target ...` grammar
retains its format-5 spelling and continues to replace the selected Link's
previous-revision and `messages` arrays. For an existing exact-target Link, it
preserves the complete metadata collection. For a newly created target Link, it
creates exact empty metadata.

This is a deliberate successor refinement of format 5's whole-record wording:
a legacy invocation cannot erase an extension area it does not name. Metadata
changes occur only through `link-metadata set` or `link-metadata remove`.

`unlink REF --target TARGET` still removes the complete exact-target Link,
including all its metadata. `add REF --root --clear-cause-links` still removes
every Link and therefore every contained metadata entry. Those commands already
name whole-Link deletion explicitly. No clear-all-metadata option, multi-target
metadata mutation, wildcard namespace, or implicit migration is added.

### Mutation receipt

`link-metadata set` and `remove` follow ADR 0022 destination selection: terminal
stdout defaults to human output, non-terminal stdout defaults to JSON, and
`--format` overrides it.

The JSON receipt schema is `sealgraph/link-metadata-mutation/v1` with exact
member order:

```text
schema, action, ref, target_seal, namespace, before, after,
candidate_schema, prospective_provenance_id, prospective_seal_id
```

`action` is `SET` or `REMOVED`. `ref`, `target_seal`, and `namespace` identify
the exact mutation. `before` and `after` are nullable complete metadata-entry
records. `SET` requires non-null `after`; `REMOVED` requires non-null `before`
and null `after`. An idempotent same-entry `SET` has equal non-null values.
Candidate schema is exactly `sealgraph/candidate/v6`; both prospective IDs are
full lower-hex IDs computed from the resulting valid Candidate.

Human receipts identify action, REF, full target ID, namespace, schema or
`none`, and abbreviated prospective IDs. They do not print the arbitrary value;
the next explicit inspection command is shown. Failure emits no receipt.

### Format-6 shared machine records

Format-6 machine documents use ADR 0027's compact UTF-8 JSON plus LF, exact
member order, full IDs, arrays-present rule, and no-partial-output failure. The
following shared records replace their format-5 counterparts:

```text
metadata_entry:
  namespace, schema, value

cause_link:
  target_seal, previous_revision_seal_of_target_seal, messages, metadata

assertion_source:
  observer_seal, observer_seal_schema,
  observer_provenance, observer_provenance_schema,
  target_seal, previous_revision_seal_of_target_seal, messages, metadata

seal_view:
  seal_id, seal_schema, material_id, provenance_id, provenance_schema,
  content_blob_id, content_bytes, attachments, root, draft, cause_links

candidate_view:
  ref, candidate_schema, expected_ref_head,
  prospective_seal_id, prospective_material_id, prospective_provenance_id,
  content_blob_id, content_bytes, attachments, root, draft, cause_links
```

`seal_schema` is exactly `sealgraph/seal/v5` or `sealgraph/seal/v6`;
`provenance_schema` and `observer_provenance_schema` are respectively the exact
paired `sealgraph/provenance/v1` or `sealgraph/provenance/v2` value.
`observer_seal_schema` identifies the observer's exact Seal schema.
`candidate_schema` is exactly `sealgraph/candidate/v5` or
`sealgraph/candidate/v6`.

Every machine `cause_link` contains `metadata`. For historical Provenance v1 or
Candidate v5 it is `[]`, and the enclosing schema fields identify that value as
the ADR 0030 historical projection. For v2/v6 it is the complete stored canonical
array. Machine output never truncates a metadata value or omits an unknown but
structurally valid namespace.

### Comparison records

Format-6 Candidate and immutable comparison retain ADR 0027's explicit baseline
and no-inferred-predecessor rules. Their `changes.cause_links` member becomes a
`cause_links_change`:

```text
cause_links_change:
  changed, before, after, records

cause_link_change:
  target_seal, before, after,
  previous_revision_seal_of_target_seal, messages, metadata

metadata_change:
  namespace, before, after
```

`cause_links_change.before` and `.after` are complete cause-link arrays and
`changed` follows the exact if-and-only-if equality rule. `records` contains one
entry for every target whose complete Link differs, ordered by full target
SealID.

For a `cause_link_change`, `before` and `after` are nullable complete Cause Link
records and at least one is non-null. The previous-revision and `messages`
members are ordinary `change` records with nullable arrays for Link addition or
removal. `metadata` is a namespace-ordered array containing every and only
changed namespace.

Each `metadata_change.before` and `.after` is a nullable complete metadata entry;
at least one is non-null and equal entries are omitted. Thus schema-only,
value-only, add, and remove changes are distinguishable from message and
structural previous-revision changes without assigning domain meaning.

`linklog` uses the same `cause_link_change` record. Its `--upstream` filter still
filters only by exact `target_seal`; metadata does not affect structural history
edge membership, supporting assertions, depth, or entry ordering.

Every `linklog/v3` entry has exact member order:

```text
minimum_newer_depth,
newer_seal_id, newer_seal_schema,
newer_provenance_id, newer_provenance_schema,
previous_seal_id, previous_seal_schema,
previous_provenance_id, previous_provenance_schema,
supporting_assertions, target_revision_observation, changes
```

The newer and previous endpoint identities are complete. Their Seal and
Provenance schema values use the exact generation identifiers defined for
`seal_view`, and each Seal/Provenance pair must be one of ADR 0030's valid typed
pairings. In every contained `cause_link_change`, `before` is owned by the
previous endpoint and `after` is owned by the newer endpoint. A null side has no
Cause Link, but its endpoint generation remains present. The endpoint Provenance
schemas therefore distinguish a Provenance v1 projected `metadata: []` from a
Provenance v2 stored empty array without repeating generation fields on every
changed namespace or Link.

### Format-6 schema matrix

Commands whose documents embed changed Cause Link, assertion-source, Seal-view,
Candidate-view, comparison, or mixed typed-role meaning receive successor
schemas:

| Command | Format 5 | Format 6 |
| --- | --- | --- |
| `show` | `sealgraph/show/v2` | `sealgraph/show/v3` |
| `candidate show` | `sealgraph/candidate-show/v2` | `sealgraph/candidate-show/v3` |
| `candidate compare` | `sealgraph/candidate-compare/v2` | `sealgraph/candidate-compare/v3` |
| `graph` | `sealgraph/graph/v2` | `sealgraph/graph/v3` |
| `impact` | `sealgraph/impact/v2` | `sealgraph/impact/v3` |
| `log` | `sealgraph/log/v2` | `sealgraph/log/v3` |
| `linklog` | `sealgraph/linklog/v2` | `sealgraph/linklog/v3` |
| `compare` | `sealgraph/compare/v2` | `sealgraph/compare/v3` |
| `fsck` | `sealgraph/fsck/v2` | `sealgraph/fsck/v3` |

`status` retains `sealgraph/status/v3` and `stale` retains
`sealgraph/stale/v2` because neither document contains Link, assertion-source,
typed schema, or comparison values and their meanings do not change. Source,
manifest, tag, REF, recovery, raw-content, and narrow publication receipts retain
their accepted schemas or exact protocols for the same reason.

All successor top-level member orders remain those of ADR 0027 after substituting
the shared records above. `candidate-compare/v3` and `compare/v3` use the new
`cause_links_change`. `linklog/v3` retains its top-level order, uses the new
`cause_link_change` array, and replaces its entry order with the exact typed
endpoint order specified above.

`fsck/v3` has exact member order:

```text
schema, result, blobs,
seals, seals_v5, seals_v6,
materials,
provenances, provenances_v1, provenances_v2,
candidates_v5, candidates_v6,
refs, tags, active_seals,
historical_or_detached_seal_ids, unreferenced_blob_ids
```

All count members are non-negative integers. Aggregate `seals` equals
`seals_v5 + seals_v6`; aggregate `provenances` equals
`provenances_v1 + provenances_v2`. Candidate counts cover the complete valid
Candidate namespace. The two ID arrays and all physical/coherent validation
rules remain those of ADRs 0023, 0025, and 0030.

### Human inspection

Human `show`, `candidate show`, comparison, graph, impact, log, and `linklog`
keep ADR 0022's terminal-first layout and safe width behavior. Where a Seal,
Provenance, or Candidate is displayed, the format/schema generation is visible.

Within each Cause Link, output separates:

- target;
- previous revisions;
- messages; and
- metadata entries.

Each metadata entry displays its namespace, schema or `none`, and complete value
as compact canonical JSON using ADR 0030 escaping. Control characters are never
emitted literally. Empty metadata is explicitly `none`. Historical v5 projection
is labeled `projected empty from v5`; it is not presented as stored v1 content.

Comparison and `linklog` use separate labels for target addition/removal,
previous-revision changes, message changes, and per-namespace metadata changes.
They do not infer progress, correction, reconciliation, supersession, approval,
or Assessment meaning.

Each human `linklog` entry displays the newer and previous endpoint Seal and
Provenance generations beside their abbreviated identities. A projected empty
metadata value is labeled against its owning v1 endpoint; a stored empty value
is labeled against its owning v2 endpoint.

The maximum value and per-Link metadata bytes are bounded by ADR 0030. Human
output does not silently truncate within those canonical limits. Existing
destination-safe escaping and stdout/stderr separation still apply.

### Repository migration command and receipt

The explicit ADR 0030 transaction is invoked only as:

```text
sealgraph migrate repository --from 5 --to 6 [--format human|json]
```

`--from 5` and `--to 6` are each required exactly once. No other pair, inferred
current format, batch path, target directory, downgrade, automatic continuation,
or force option exists. The command operates only on `.sealgraph` below the
current directory, never detects Git, and performs exactly the config-only
transaction in ADR 0030.

The success JSON schema is `sealgraph/repository-migrate/v1` with exact member
order:

```text
schema, from_format, to_format, result,
retained_seals_v5, retained_provenances_v1, retained_candidates_v5
```

Formats are JSON integers 5 and 6; `result` is exactly `MIGRATED`; retained
counts are the complete read-back format-6 inventories. Human success identifies
the same transition and counts.

If the config transition commits but success output cannot be delivered, the
command reports `MIGRATION_COMMITTED_OUTPUT_UNDELIVERED` on stderr when possible
and instructs the operator not to rerun migration. The bounded read-only
verification is `sealgraph fsck --format json`, whose successful format-6 result
uses `sealgraph/fsck/v3`. An already-format-6 repository fails the mutation
command without rewriting config or claiming a second migration.

### Query and Assessment boundary

No metadata filter, search, query AST, index, namespace registry, schema fetch,
or validator command is introduced. Callers inspect exact selected Seals or
Candidates and may perform domain-specific validation outside core.

ADR 0028 Assessment authoring, adoption, removal, review-state, and impact-output
successors remain separately gated. This ADR exposes only the assessment-free
`upstream-change/v2` consequence already accepted in ADR 0030 and does not add an
Assessment reference to a Link or metadata entry.

## Alternatives

### Put metadata flags on `link`

Rejected because structural previous/message replacement and namespace
add/replace/remove have different preservation rules. Combining them would make
omission ambiguous and invite multi-value grouping errors.

### Let existing `link` clear omitted metadata

Rejected because a legacy messages-only invocation could destroy structured
state it did not name. Preserving metadata makes compatibility an authoring
property rather than only a parse property.

### Provide a generic metadata query language now

Rejected because exact selected-object inspection satisfies the accepted
contract and query syntax, indexes, cost, and observation scope require a
separate decision.

### Reuse format-5 JSON schemas

Rejected because changed nested records and mixed-generation labels would give
one schema identifier two meanings.

### Show only metadata digests in human output

Rejected because the accepted requirement calls for inspectable arbitrary
values. Canonical JSON escaping and ADR 0030 bounds make complete display safe
without claiming domain interpretation.

### Hide historical projection labels

Rejected because an empty projected value is not a member stored in Provenance
v1 or Candidate v5 bytes.

### Repeat endpoint generation on every `cause_link_change`

Rejected because all Link changes in one `linklog` entry share the same previous
and newer owning Provenances. Entry-level typed endpoint identity is complete
and avoids repeating identical schema context for every changed Link and
namespace.

## Consequences

- Messages-only callers retain their syntax and cannot erase metadata through
  omission.
- Metadata mutation is explicit, one-target, one-namespace, and Candidate-only.
- JSON value files avoid mandatory shell quoting for nested data while inline
  values remain available.
- Machine consumers must adopt successor schemas for documents containing
  changed records; unchanged schemas remain stable.
- Comparison and `linklog` distinguish structural, message, and metadata changes.
- Historical v5 projections are visible rather than fabricated as stored bytes;
  mixed-generation `linklog` entries carry both owning endpoint generations.
- Complete human values can be large but remain bounded to ADR 0030's maximum.
- Migration has one exact command and versioned receipt but no retry-on-ambiguity
  behavior.
- Query, schema validation, and Assessment commands remain unavailable.

## Implementation notes

Acceptance does not itself authorize any implementation action.

After this complete companion set is accepted under separate implementation
authority:

- parse JSON with duplicate-key detection before host-map construction;
- buffer and validate the complete Candidate before expected-old replacement;
- reuse one shared format-6 view and diff model across human and JSON output;
- test every schema member order and exact fixture;
- test v5 projection labels and v6 stored empty metadata separately;
- test preservation across legacy `add` and `link`, and deletion through
  `unlink` and root clearing;
- test same-entry idempotence, absent remove, invalid values, input-source
  conflicts, concurrent Candidate change, and output failure;
- test every comparison change class and `linklog --upstream` independence;
- test mixed v5-to-v6 and v6-to-v5 `linklog` entries where one empty metadata
  array is projected and the other is stored;
- test migration success, already-migrated failure, pre-commit failure,
  committed-output-undelivered handling, and fsck readback; and
- update help, completion, requirements, architecture, storage format, CLI,
  integrations, and planning documents only in the authorized implementation
  change set.

## Review

Author self-review verified:

1. old `link` and `add` syntax cannot erase metadata by omission;
2. set/remove selects exactly one Candidate Link and namespace and has no
   publication or migration side effect;
3. every machine document with changed nested meaning receives a successor
   schema while truly unchanged documents retain theirs;
4. comparison and `linklog` distinguish previous, message, and namespace changes;
5. historical projections are explicit in human and machine output, including
   typed previous/newer endpoint context in `linklog`;
6. migration output ambiguity does not cause an automatic second mutation; and
7. query and Assessment scope remains excluded.

The initial author self-review LLMThink audit completed with no fatal, error, or
warning finding. Independent review then identified one P1 `linklog` endpoint-
generation omission. The revision reasoning and causal audits also completed
with no fatal, error, or warning finding; the revision added exact typed endpoint
context and mixed-generation fixtures. Focused independent rereview returned
`PASS` with no P0 through P3 findings and resolved the prior P1. The Operator
then explicitly accepted the exact reviewed technical candidate.

## Evidence

- **E-CO-001:** accepted ADR 0029 requires one-namespace explicit Candidate
  mutation, separate message/metadata inspection, and detailed comparison and
  `linklog` behavior.
- **E-CO-002:** accepted ADR 0030 fixes format-6 bytes, historical v5 projection,
  resource bounds, config-only migration, and upstream-change/v2.
- **E-CO-003:** accepted ADR 0027 fixes format-5 authoring grammar, schema
  versioning practice, shared records, and comparison/history meaning.
- **E-CO-004:** accepted ADR 0022 fixes terminal-first destination selection,
  safe rendering, stdout/stderr separation, and narrow receipt behavior.
- **E-CO-005:** current implementation confirms that format-5 `link` replaces
  one complete three-field Link and that non-terminal mutation receipts are not
  complete inspection documents.

| Claim | Evidence | Required implementation action |
| --- | --- | --- |
| C-CO-001: use dedicated one-namespace commands | E-CO-001, E-CO-003, E-CO-005 | A-CO-001, A-CO-005 |
| C-CO-002: preserve metadata across legacy structural authoring | E-CO-001, E-CO-003, E-CO-005 | A-CO-001, A-CO-005 |
| C-CO-003: expose complete metadata and schema generation | E-CO-001, E-CO-002, E-CO-004 | A-CO-002, A-CO-005 |
| C-CO-004: version every changed machine contract | E-CO-002, E-CO-003 | A-CO-002, A-CO-005 |
| C-CO-005: distinguish field and namespace changes | E-CO-001, E-CO-003 | A-CO-003, A-CO-005 |
| C-CO-006: expose one exact migration command and receipt | E-CO-002, E-CO-004 | A-CO-004, A-CO-005 |
| C-CO-007: exclude query and Assessment commands | E-CO-001, E-CO-002 | A-CO-006 |

Implementation action identities are:

- **A-CO-001:** implement metadata mutation and legacy-authoring preservation.
- **A-CO-002:** implement shared format-6 human and machine views.
- **A-CO-003:** implement detailed comparison and `linklog` changes.
- **A-CO-004:** implement the exact repository migration CLI and receipt.
- **A-CO-005:** add grammar, atomicity, schema, rendering, and migration fixtures.
- **A-CO-006:** keep query and Assessment command work outside this slice.

## Follow-ups

- Preserve the exact acceptance and review records with this ADR.
- Treat normative conversion, implementation, migration, and repository
  synchronization as separately authorized lifecycle actions.
