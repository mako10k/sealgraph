# Link Metadata and SealGraphQL design proposal

Status: final design proposal for review; not an accepted persisted or CLI
contract. No implementation is authorized by this document.

Audit record: `llmthink dsl audit` reported zero fatal, error, or warning
findings for the companion decision model. Sections 11 through 13 record the
separate symmetry, consistency, and completeness reviews. Those reviews pass
the responsibility split and generic model while retaining the listed public
and persisted decisions as explicit implementation gates.

## 1. Decision summary

SealGraph should add generic, namespaced, immutable metadata to each exact
Cause Link. SealGraph core owns only structural validity, deterministic
canonicalization, resource bounds, and identity commitment. Namespace owners
define all domain meaning.

SealGraphQL (working name) should initially be a read-only query language over
Seals, REFs, revision edges, Cause Links, and Link Metadata. It may select and
project facts, but it must not mutate a Candidate, create a Seal, move a REF,
repair provenance, or turn a matching predicate into approval.

Schema declaration is optional and queryable. Schema conformance validation is
not part of canonical decoding or initial query execution.

## 2. Goals and non-goals

### Goals

- Carry application-defined structured facts on one exact Cause edge.
- Avoid global key collisions through explicit namespaces.
- Commit metadata into Candidate and Seal identity deterministically.
- Support exact, reproducible filtering and traversal over historical and
  current graph observations.
- Leave room for separately adopted schema validators without making them a
  core read dependency.

### Non-goals

- Defining approval, trust, authorship, time, confidence, or relation kinds.
- Changing Cause traversal, stale propagation, admissibility, or cycle rules
  based on metadata.
- Dynamic links to a REF HEAD.
- Remote schema fetching, schema registries, inference, full-text search, or
  similarity search.
- Query-driven relink, repair, batch seal, or multi-REF publication.
- Treating Link Metadata as secret storage.

## 3. Responsibility boundary

| Layer | Owns | Does not own |
| --- | --- | --- |
| Canonical/domain core | value types, bounds, ordering, duplicate rejection, hash commitment | namespace meaning, schema truth, trust |
| Namespace owner | vocabulary, interpretation, optional schema and version policy | Seal identity or graph invariants |
| Repository | Candidate editing, exact-target resolution, one-REF seal/CAS publication | implicit metadata repair or schema fetching |
| Query engine | deterministic selection, traversal, projection, coherent observations | mutation, approval, inference |
| Optional validator | explicitly selected schema conformance check | canonical readability or hidden admission policy |

## 4. Proposed persisted model

A Link remains one exact Cause edge:

```text
Link = target_seal + message + metadata[]
```

Illustrative, non-final format bytes:

```json
{
  "target_seal": "fedcba...",
  "message": "basis",
  "metadata": [
    {
      "namespace": "org.example.review",
      "schema": "https://example.org/schemas/review/v1",
      "value": {
        "role": "approver",
        "score_millis": 950,
        "status": "accepted"
      }
    }
  ]
}
```

The example conveys no built-in approval semantics. To core, `status` is only
a string beneath an opaque namespace.

### 4.1 Metadata entry

Each entry has exactly these required members in this order:

```text
namespace, schema, value
```

- `namespace` is a non-empty UTF-8 string identifying the vocabulary owner.
- `schema` is either JSON `null` or a non-empty UTF-8 identifier.
- `value` is one canonical metadata value.
- Unknown members are errors.
- There is at most one entry for a namespace on one Link.
- Entries are sorted by bytewise UTF-8 `namespace`.

Namespace strings are opaque and case-sensitive. The recommended convention is
a lowercase reverse-DNS name, optionally followed by slash-separated local
components, for example `org.example.review` or
`org.example.review/security`. Core does not verify DNS ownership or rewrite
case, Unicode, separators, or aliases.

The exact namespace `sealgraph` and every namespace beginning `sealgraph/` are
reserved for core-defined virtual values. They are rejected in persisted
metadata input. This reservation is structural, not a claim that other
namespaces are owned or trusted.

A schema identifier is also opaque and case-sensitive. Namespace owners should
use immutable, version-specific identifiers. Core does not dereference it and
does not imply that the value was validated. `null` and a declared schema are
different identity-bearing states.

### 4.2 Canonical value model

The value model contains only:

- `null`;
- boolean;
- signed 64-bit integer, encoded as the shortest decimal JSON integer with no
  leading zero and with negative zero forbidden;
- UTF-8 string;
- array of canonical values, preserving order;
- object from UTF-8 string keys to canonical values, with keys sorted by
  bytewise UTF-8 order and duplicate keys rejected.

Floating point and arbitrary JSON numbers are absent. Domains needing decimal,
large integer, date, duration, binary, or unit-bearing values must define a
string or structured representation in their namespace. This avoids language-
dependent rounding, NaN/infinity, exponent, and negative-zero identities.

Strings receive no Unicode normalization. Producers that need normalized text
must specify it in their namespace schema. Canonical JSON escaping follows the
same rules as Seal payload strings.

### 4.3 Required resource limits

The first persisted contract must choose exact limits and test every boundary.
The recommended limits are:

- at most 64 metadata namespaces per Link;
- at most 16 nested array/object levels, counting the root value as level 1;
- at most 4,096 total value nodes per Link;
- at most 4,096 UTF-8 bytes per namespace, schema identifier, object key, or
  string scalar;
- at most 65,536 canonical metadata bytes per Link;
- at most 1,048,576 canonical Seal payload bytes.

All limits are byte/value limits, not display limits. Exceeding one is an
error before Candidate persistence or Seal creation. Readers enforce the same
limits. Implementations must measure bounds iteratively or otherwise avoid
unbounded recursion and allocation.

The exact numbers are part of the future format decision and may be adjusted
before that ADR is accepted; they cannot become environment-dependent.

### 4.4 Identity and graph invariants

- Metadata is part of the Link, Candidate, canonical Seal bytes, and SealID.
- Changing namespace, schema, value, array order, or any scalar creates a
  material Link difference and therefore a different SealID.
- Object key ordering and metadata entry ordering do not express meaning; the
  canonical encoder sorts them.
- Array order is meaning-bearing.
- At most one Cause Link may target one exact SealID, unchanged from format 4.
- Metadata does not create another edge and cannot contain a dynamic selector.
- Every Link participates in Cause cycles, closure admission, stale, impact,
  frontier, and traversal exactly as it does without metadata.
- Metadata must not contain secret plaintext; schema identifiers must not
  embed credentials.

## 5. Schema policy

Initial core behavior has three distinct levels:

1. **Structural validation, mandatory:** canonical type, ordering, uniqueness,
   bounds, UTF-8, and exact decode/re-encode equality.
2. **Schema declaration, optional:** store and query an opaque immutable-
   intended identifier.
3. **Schema conformance validation, deferred:** an explicit optional operation
   may validate a Candidate before sealing, but its availability or result must
   not change whether canonical historical Seals can be read.

A future validator requires its own threat and determinism contract. It must
define schema bytes/identity, resolver policy, offline behavior, recursion and
resource limits, supported vocabulary, and whether validation evidence is
itself sealed. Network retrieval during canonical decode, `fsck`, graph query,
or ordinary seal admission is forbidden.

Schema declarations are descriptive claims, not proof. Authentication,
authority, signatures, and trusted timestamps remain separate sealed content
or future explicitly designed features.

## 6. Candidate and CLI implications

The future mutation surface should extend the existing explicit `link`
Candidate operation, not add query mutation. Exact syntax remains deferred,
but it must satisfy these properties:

- target selection resolves before one Candidate write;
- adding, replacing, or removing one namespace is explicit;
- repeated namespace input is rejected rather than last-write-wins;
- metadata can be supplied from an exact regular file or stdin without shell
  quoting loss;
- rejected input produces no Candidate change;
- `unlink` still removes the complete exact-target Link;
- `derive` copies Link Metadata exactly;
- candidate show/diff and linklog distinguish message and namespace changes;
- default human output escapes and bounds arbitrary metadata;
- exact machine output uses a versioned JSON schema.

An explicit `link metadata remove`-like operation may later remove one
namespace without removing the Cause Link. It must not be named or behave like
an update of an immutable Seal; it edits only one Candidate.

## 7. SealGraphQL read model

“SealGraphQL” is a working product name, not a commitment to implement the
GraphQL specification. The first design should use a versioned query AST as
the normative contract. A textual syntax may compile to that AST later.

### 7.1 Entity model

- `Seal`: immutable SealID and canonical attributes.
- `Ref`: one name and its coherently observed current HEAD.
- `RevisionEdge`: child to exact `parent_revision`.
- `CauseLink`: dependent/source Seal to exact target Seal, message, metadata.
- `MetadataEntry`: namespace, optional schema identifier, and value.
- `DerivedFact`: active leaf, detached/historical, stale class, impact, or path,
  computed from an explicitly defined observation.

Direction names are fixed:

```text
outgoing Cause: dependent/source -> upstream/target
incoming Cause: upstream/target <- dependent/source
revision parent: child -> parent
revision child: parent <- child
```

Traversal never mixes revision and Cause edges unless the query explicitly
selects both in a path expression.

### 7.2 Metadata addressing and predicates

A metadata location is `(namespace, JSON Pointer)`. RFC 6901 pointer escaping
is used after selecting one namespace; the empty pointer selects the complete
namespace value. Namespace selection is exact and case-sensitive.

An entry predicate may also require `schema_eq` with one exact identifier or
`schema_is_null`. Schema predicates test the entry declaration itself and do
not validate the value. A missing namespace matches neither predicate.

#### Reserved virtual `sealgraph` namespace

SealGraphQL exposes intrinsic Link fields through a virtual namespace so the
same path and predicate machinery can query both core facts and user metadata:

```json
{
  "source_seal": "<full dependent SealID>",
  "target_seal": "<full upstream SealID>",
  "message": "basis"
}
```

The query locations are therefore `("sealgraph", "/source_seal")`,
`("sealgraph", "/target_seal")`, and `("sealgraph", "/message")`.
`source_seal` is derived from the Seal containing the Link; `target_seal` and
`message` come from the canonical Link. The object is never persisted as a
metadata entry and does not add bytes to Seal identity. Its values cannot
disagree with the underlying Link fields.

The virtual namespace is always present for a valid Link and has the fixed
core schema `sealgraph/virtual-link/v1` in query output. It is read-only,
cannot be supplied to Candidate mutation, and is not returned as a member of
the persisted `metadata` array. Observation-dependent facts such as REF name,
current-head status, stale class, or traversal direction are not placed in this
namespace; they remain explicit query entity/derived-fact fields.

The initial predicate set should be:

- `exists` / `missing`;
- type test: `null`, `boolean`, `integer`, `string`, `array`, `object`;
- strict typed `eq` / `ne`;
- integer `lt`, `le`, `gt`, `ge`;
- string `starts_with`;
- object `contains_key` where the addressed value is an object;
- boolean composition `all`, `any`, and `not`.

Missing is not JSON `null`. Every comparison, including `ne`, returns false for
a missing location. Type mismatch is false for a well-formed comparison, not
an error; an operator applied to a query literal of the wrong declared type is
a query error. `not` negates a boolean predicate result, so callers needing to
exclude missing values must combine `exists` explicitly. No implicit
string/number, case, Unicode, date, or schema coercion occurs.

Regular expressions, locale collation, floating comparisons, full-text search,
and user-defined functions are deferred because they complicate portability,
resource bounds, and deterministic behavior.

### 7.3 Query roots and observation

Initial roots are:

- exact explicitly selected SealIDs;
- all valid canonical Seals in the local object inventory;
- one coherent complete current REF/head observation.

Historical inventory queries must distinguish valid unreachable objects from
current graph facts. Queries involving REF, active/detached status, stale,
frontier, or impact use one complete REF/head observation and revalidate it
before successful output. Candidate state is excluded unless a future root
explicitly models Candidate inspection.

Canonical corruption, missing objects, cycles, or changed observations fail
without partial stdout. Query limits may truncate presentation only when the
result explicitly marks truncation; they must not make a partial result appear
complete.

### 7.4 Determinism and output

- The query AST has a versioned schema, for example
  `sealgraph/query/v1`; output has a separate versioned schema.
- Results use full IDs and deterministic bytewise ordering.
- Object projections preserve canonical key ordering.
- Query plan, wall clock, filesystem enumeration order, cache presence, and
  parallel execution do not affect logical results.
- Default human output uses bounded escaped representations.
- Machine JSON never emits arbitrary metadata as mixed terminal text.
- Resource-limit failure is explicit; silent row dropping is forbidden.

An implementation may use indexes or disposable caches, but they bind to the
repository format and exact object/REF observation needed by the query. Missing
or invalid indexes fall back to canonical scanning; indexes are never
authoritative and never enter Seal identity.

### 7.5 Illustrative query AST

This is explanatory, not yet a public schema:

```json
{
  "schema": "sealgraph/query/v1",
  "from": {"current_refs": true},
  "traverse": {"edge": "cause", "direction": "outgoing", "depth": {"min": 1, "max": 8}},
  "where": {
    "metadata": {
      "namespace": "org.example.review",
      "pointer": "/status",
      "eq": "accepted"
    }
  },
  "select": ["source_seal", "target_seal", "metadata"],
  "order_by": ["source_seal", "target_seal"]
}
```

The result says only that bytes matching the predicate exist on an exact Cause
Link. It does not say that the review is authentic, authoritative, current, or
accepted by SealGraph.

The same predicate form can select a core target without a separate field
language:

```json
{
  "metadata": {
    "namespace": "sealgraph",
    "pointer": "/target_seal",
    "eq": "fedcba..."
  }
}
```

## 8. Mutation boundary

The initial SealGraphQL contract is query-only. A metadata “update” requires:

1. selecting exactly one destination REF/Candidate;
2. resolving one exact target SealID;
3. editing the Candidate Link Metadata;
4. validating the complete candidate and Cause closure;
5. explicitly sealing once;
6. publishing that one REF through expected-old CAS.

Those are repository commands, not query side effects. A future mutation
language would merely be another explicit client of this protocol and would
need invocation fail-fast behavior, concurrency/CAS semantics, single-write
bounds, dry-run representation, and separate approval. It must not introduce
batch seal, automatic relink, recursive repair, or partial multi-REF success.

## 9. Compatibility and migration

Adding `metadata` changes canonical Link, Seal, Candidate, diff/linklog output,
fixtures, and SealIDs. It therefore requires:

- a new repository format and schema identifiers;
- exact canonical member/order and boundary fixtures;
- deterministic old-to-new SealID mapping in dependency-first order;
- an explicit read-only export and absent-target atomic load, or another
  separately approved migration contract;
- no dual interpretation of format-4 bytes and no ignored unknown member;
- updates to `fsck`, history, graph fixtures, dogfood state, CLI docs, and
  release compatibility notes.

The format transition must preserve target Seal identity mapping and Link
metadata exactly. It must not infer namespaces from existing messages or turn a
message into structured metadata automatically.

## 10. Security and operational review

- Metadata is untrusted input and may contain terminal controls, misleading
  Unicode, oversized nesting, or adversarial key sets; safe output and hard
  bounds are mandatory.
- Namespace and schema strings are identifiers, not URLs to fetch during core
  operations.
- Predicates must not execute code, resolve environment variables, access the
  network, or read paths outside the repository.
- Query cost needs explicit maximum depth, visited-Seal, visited-Link, result,
  and output-byte limits before implementation.
- Secret plaintext remains forbidden; queries and logs must not accidentally
  turn metadata into an unbounded disclosure channel.
- Indexes/caches are disposable derived state and must fail back to canonical
  facts rather than repair or override them.

## 11. Symmetry review

| Dimension | Symmetric rule |
| --- | --- |
| Add/remove | One namespace can be added or removed explicitly on one Candidate Link; neither is silent or Seal mutation. |
| Encode/decode | Every accepted value has one encoding; every decoded value must re-encode byte-identically. |
| Outgoing/incoming | Both directions describe the same exact Cause Link; source and target names never reverse meaning. |
| Current/historical | The same metadata predicate applies; current-only derived facts require a coherent REF observation. |
| Present/missing/null | Presence, absence, and explicit null remain three distinct query states. |
| Schema/no schema | Both are readable and queryable; neither changes Cause semantics or implies validation. |
| Persisted/virtual namespace | Both use the same predicate addressing; only user namespaces are persisted and identity-bearing. |
| Human/machine output | Both expose the same facts; human output is bounded/escaped and machine output is versioned. |
| Cache/scan | Both yield the same logical result; cache cannot add or suppress matches. |

No asymmetric automatic behavior was found necessary. Mutation remains
deliberately asymmetric with query because publication has stronger authority
and concurrency requirements.

## 12. Consistency review

- Exact `target_seal` and one-Link-per-target invariants are unchanged.
- Metadata remains material edge state, consistent with the existing Link
  message, while actor/time/trust claims remain external domain semantics.
- All Cause Links continue to participate equally in admissibility and stale
  propagation; metadata cannot create a hidden non-staling link kind.
- Canonical sorting, duplicate rejection, UTF-8 handling, safe output, coherent
  REF observations, and disposable-cache rules reuse existing principles.
- Read-only query cannot bypass the one-REF Candidate/CAS publication model.
- A format change is explicit rather than silently extending format 4.

The proposal intentionally does not reinterpret prior Link messages, REF
names, tags, parent edges, attachment metadata, or sealed content.

## 13. Completeness review

Covered decisions include data types, identity, ordering, duplicates, limits,
namespace ownership, schema declaration/validation, query direction, missing
versus null, comparison typing, current versus historical observation,
determinism, output safety, caching, mutation authority, migration, security,
and compatibility.

The reserved `sealgraph` namespace covers intrinsic source, target, and message
without duplicating canonical state. Its reserved prefix leaves room for later
versioned core virtual namespaces, but adding fields still requires a query
schema version decision.

The following remain explicit gates rather than accidental omissions:

1. Final numeric persisted limits.
2. Exact format number and migration release sequencing.
3. Exact Candidate mutation CLI and machine schemas.
4. Normative query AST, grammar, cost model, and output schema.
5. Optional schema language, immutable schema-byte identity, resolver, and
   validator evidence.
6. Query indexing strategy and cache format.
7. Whether “SealGraphQL” remains the public name.
8. Any query mutation surface.

None of these gates is required to accept the responsibility split and generic
metadata model; each is required before implementing its corresponding public
or persisted boundary.

## 14. Recommended sequence

1. Accept or revise ADR 0017 and the generic value/namespace/schema policy.
2. Freeze exact limits and canonical format in a format-transition ADR.
3. Specify Candidate Link Metadata mutation and inspection with boundary
   examples.
4. Implement the storage/domain vertical slice with canonical fixtures and
   migration tooling.
5. Freeze a read-only query AST and conformance examples.
6. Implement canonical-scan queries first; add disposable indexing only after
   measured need.
7. Consider optional validators and mutation syntax only as separate proposals.
