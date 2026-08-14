# External specification review and mitigation analysis

Status: design analysis; not an accepted ADR.

This document records the external-specification review requested before the
next implementation slice. It separates operator-confirmed constraints from
recommended mitigations and deferred decisions. It does not silently accept an
ADR or authorize Git-sidecar implementation.

The structured reasoning source is
[`../decisions/2026-08-14-external-spec-review.think`](../decisions/2026-08-14-external-spec-review.think).

## 1. Confirmed constraints

### 1.1 Standalone Git compatibility

Standalone compatibility has two goals:

1. the `sealgraph` command lifecycle remains safe and internally consistent;
2. an explicitly configured Git SHA-256 low-level object API can identify and
   read native loose blobs without identity disagreement, silent translation,
   or destructive side effects.

This is an object/storage/forensics contract. It does not make `.sealgraph` a
Git repository and does not inherit Git commit, branch, merge, checkout,
reflog, garbage-collection, or maintenance semantics. In particular:

- `.sealgraph/objects` is compatible with the Git SHA-256 loose-blob envelope;
- conformance is exercised in an explicit SHA-256 Git context;
- a SHA-1 repository must not use the SHA-256 native object directory as an
  alternate;
- `.sealgraph/config` is a sealgraph config, not a promise that Git can open
  `.sealgraph` as a repository;
- low-level read refusal in an unsupported object-format context is preferable
  to guessed translation;
- Git GC, prune, repack, ref transactions, and porcelain are outside the
  standalone contract and must not be directed at `.sealgraph`.

The current bidirectional `hash-object`/`cat-file` conformance test is evidence
for this boundary. It is not evidence for Git-sidecar content semantics.

### 1.2 REF ownership of seals

A current native seal belongs to exactly one logical REF. Its owner REF is part
of canonical bytes and therefore its identity. A seal cannot be reused as the
HEAD, parent, or tag target of another REF. Parent chains stay within one REF.

Dependency links may cross REF boundaries because a link stores both the
target REF and the concrete target seal. Crossing the graph edge does not
change the target seal's owner.

Internal code should distinguish a bare object identity from a scoped seal
identity such as `(REF, seal ID)`. A future alias or resolver may cross a scope
boundary explicitly, similarly to a scoped tag lookup, but it must preserve the
resolved seal's owner and define stale/selector behavior at that time. Native
v2 does not add speculative persisted fields for that future.

## 2. Blocking consistency issue

### 2.1 Seal publication and candidate loss

The current REF CAS is necessary but not sufficient. A seal command can read
candidate C1, another command can publish candidate C2, and the first command
can then seal C1 and unconditionally delete C2. Dependency HEADs are also read
one at a time, so the current contract does not identify a coherent publication
snapshot.

The recommended pre-1.0 mitigation is a repository-wide writer coordination
protocol for every mutating sealgraph command:

1. acquire the repository writer guard;
2. load one candidate and the required closure;
3. validate ownership, DAG admissibility, draft policy, and closure HEADs;
4. write immutable objects;
5. revalidate the candidate/heads required by the chosen protocol;
6. publish exactly one target REF with expected-old CAS;
7. clear the candidate only if it is still the version that was sealed;
8. release the guard.

The successful target REF CAS/atomic rename is the seal publication
linearization point. Immutable objects written before a failed publication may
remain dangling and must be reported rather than deleted or repaired.

This guarantee applies to cooperative sealgraph writers. An outer Git process
or manual filesystem edit that ignores the guard is an external concurrent
mutation. Sealgraph should detect and reject it where possible, but must not
claim impossible exclusion of arbitrary writers.

Alternative: candidate revision CAS plus deterministic closure-lock ordering.
That permits more concurrency but creates a larger lock protocol and failure
surface. The analysis preferred repository-wide coordination during the
experimental phase. The operator accepted it on 2026-08-14; ADR 0007 records
the transaction contract.

## 3. Recommended semantic decisions

### 3.1 Draft closure — accepted

A normal non-draft seal must not contain any reachable draft seal
in its complete dependency closure. A draft candidate may depend on current or
historical draft/normal seals. This keeps `provisional` transitive and prevents
a normal `CLEAN` head from hiding draft-only provenance.

The operator accepted this policy on 2026-08-14. Draft remains distinct from
stale, and no downstream candidate is automatically marked draft, relinked, or
resealed.

### 3.2 Candidate lifecycle

Before recurring dogfood, the public one-REF lifecycle should support:

- inspecting the complete candidate before sealing;
- comparing the candidate with its base/current HEAD;
- `unlink` for one explicit upstream REF;
- explicitly discarding a corrupt or stale-base candidate;
- refusing automatic relink, reseal, or candidate repair.

The exact CLI spelling remains open. A dedicated candidate surface or explicit
`--candidate` selectors should be compared without importing Git reset/index
semantics.

### 3.3 Safe content presentation

Default human `show` should print content identity, byte size, and a bounded
escaped preview. Messages and other arbitrary strings should also use an
unambiguous escaped representation. Raw content should be available only
through an explicit bytes-only mode that does not mix payload with metadata.

Versioned machine output must define a binary encoding and structured fields.
Preview limits, raw option naming, and JSON encoding remain open.

### 3.4 Root scope — accepted clarification

Root is a seal-generation property, not an unrecorded immutable
REF type. It is already identity-bearing canonical state, so a boundary change
can be reviewed, diffed, and retained in immutable history without another
registry. Documentation should say `root seal generation` where `root REF`
would imply a lifetime invariant.

The operator accepted this clarification on 2026-08-14. A transition creates a
new immutable seal and does not mutate earlier generations. It also does not
implicitly add or remove links; the candidate must independently satisfy root
or non-root dependency admissibility before sealing.

### 3.5 Integrity is not authority

Native v2 seals are unauthenticated assertions by a repository writer. Their
hashes prove the integrity and identity of bytes and linked objects; they do
not prove actor identity, approval authority, trusted time, or truth. The
identity-bearing `created_at` field is not an authenticated timestamp.

This clarification does not introduce signatures. Signatures and attestations
remain separate future requirements.

## 4. Additional consistency actions

### 4.1 Git sidecar gate

Standalone ODB conformance does not require go-git in the runtime. Before a
product Git reader is introduced, a separate ADR must decide:

- supported Git SHA-1/SHA-256 object formats;
- typed external identity representation;
- whether Git data is materialized as a native blob or referenced externally;
- the meaning of blob, tree, and commit as content sources;
- prefix resolution and ownership across native and Git object populations.

This is a Phase 5 gate, not a current standalone defect.

### 4.2 ADR dogfood causality

The native-v2 round is useful bootstrap evidence, but its requirements and
architecture inputs already contained the ADR 0006 conclusion. It should be
described as retrospective bootstrap, not proof of the decision-formation
order.

Future decision dogfood should use this sequence:

1. seal pre-decision premises/evidence;
2. seal the resulting ADR against those exact generations;
3. update and supersede normative documents against the ADR seal;
4. inspect the downstream stale/repair path explicitly.

### 4.3 Remaining contract backlog

The following findings remain suitable for later contract work and do not
override the blocker above:

- define whether tag operations require an active current REF and how orphaned
  historical tag scopes are inspected;
- resolve the known REF-scoped tag loose-path collision before tags become
  routine;
- define filesystem case-sensitivity and byte-length support boundaries;
- publish a command-by-selector matrix;
- separate result state, usage error, integrity failure, and operational
  failure exit codes;
- make per-edge dependency messages atomic without adding domain-specific link
  kinds.

## 5. Implementation order and status

1. [x] Implement accepted writer coordination and no-loss candidate clearing
   with focused multi-process tests.
2. [x] Implement complete-closure draft rejection tests.
3. Implement candidate inspection/diff/unlink/discard and binary-safe `show`.
4. repeat decision-document dogfood using the causal order above.
5. resolve remaining operator/tag contracts.
6. decide the Git-sidecar identity ADR before adding its runtime SDK seam.

## 6. llmthink audit result

The structured document was audited with:

```sh
llmthink dsl audit \
  docs/decisions/2026-08-14-external-spec-review.think --pretty
```

The audit reported zero fatal, error, or warning findings. It intentionally
reported contradiction-candidate hints between the three accepted decisions
and their explicitly rejected alternatives:

- repository-wide writer coordination versus fine-grained candidate/closure
  locking;
- transitive draft prohibition versus a local-only draft label;
- root as a generation property versus a REF-lifetime property.

Those tensions are represented with explicit comparisons and are not
suppressed. Publication, draft, and root alternatives are retained as rejected
decisions with rationale after their accepted resolutions.
