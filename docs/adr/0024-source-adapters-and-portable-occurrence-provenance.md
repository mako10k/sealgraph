# ADR 0024: Source adapters and portable occurrence provenance

- Status: Accepted
- Date: 2026-08-28
- Decision Owner: Operator
- Related Claims: C-SA-001, C-SA-002, C-SA-003, C-SA-004, C-SA-005
- Related Evidence: E-SA-001, E-SA-002, E-SA-003, E-SA-004
- Related Reviews: Architecture and boundary review recorded below
- Supersedes: None; extends ADR 0019 and the Git-source deferral in ADR 0011
- Superseded by: None

## Context

ADR 0019 defines one machine-local binding from a logical REF to a regular
working file. It is input configuration only: it is excluded from Candidate,
Seal, canonical REF state, SealID, dump/load, and outer-Git tracking. An
explicit `add` copies exact bytes into a Candidate, and `seal` never rereads the
working file.

The future Git sidecar can read a commit tree, but the accepted design did not
decide how one REF may select a file occurrence at one exact Git commit. The
product also needs symmetric non-Git operation and portable provenance without
making Git a hidden runtime dependency.

Git does not attach a parent/history edge to a blob. A commit points to one
tree and to parent commits; a path in that tree resolves to a tree entry and
blob. File history is derived from commit ancestry plus path/diff policy and
may use rename heuristics. It is not intrinsic blob identity.

The operator confirmed on 2026-08-28 that non-Git operation remains the
standard workflow, Git is an optional source adapter, and durable source
occurrence evidence is separately sealed and related through an exact Cause
Link.

## Decision

### Source adapter boundary

Sealgraph core consumes exact bytes through explicit source adapters. The
source adapter chooses and validates bytes for `add`; it never publishes a Seal
directly.

The conceptual binding is a tagged union:

```text
SourceBinding :=
    WorktreePath {
        path
    }
  | GitTreeEntry {
        object_format
        commit_oid
        path
        blob_oid
        file_mode
    }
```

This establishes:

- C-SA-001: Standalone non-Git operation is the default and remains complete.
- C-SA-002: Every source binding is non-canonical, local, replaceable input
  configuration rather than Seal provenance.
- C-SA-003: All adapters converge at exact Candidate content identity; `seal`
  consumes only the validated Candidate.

The checked-in `sealgraph/source-binding/v1` path binding is the
`WorktreePath` variant. A future versioned binding representation may add the
tag explicitly. Reading an old v1 binding treats it as `WorktreePath` without
rewriting it.

### Git tree-entry binding

Git-specific source selection exists only under the explicit `git sealgraph`
surface. Standalone `sealgraph` never discovers `.git`, changes behavior inside
a worktree, or parses Git-specific binding options.

A Git binding records:

- the exact Git object format (`sha1` or `sha256`);
- a full commit OID resolved at bind/rebind time, never a branch, tag, or other
  movable revision spelling;
- the exact path in that commit tree;
- the resolved full blob OID;
- the exact regular-file mode.

The outer repository is the explicitly opened repository of the
`git sealgraph` invocation. A remote URL, branch name, credential-bearing URL,
hostname, or absolute checkout path is not canonical source identity.

The initial Git file adapter accepts ordinary blob entries only. Trees,
submodules/gitlinks, and symbolic-link entries fail explicitly. The adapter
reads the exact blob stored in the commit tree; it does not apply checkout
filters, LFS smudge, working-tree encoding, or line-ending conversion.

Binding verifies that `(object_format, commit_oid, path)` resolves to the
recorded `(blob_oid, file_mode)`. Missing or mismatched objects fail without a
Candidate update. Rebind and unbind retain ADR 0019's explicit expected-old
discipline; no branch movement silently retargets a binding.

An illustrative, non-authoritative future CLI shape is:

```sh
git sealgraph source bind REF --commit COMMIT --file PATH
git sealgraph add REF
sealgraph candidate compare REF
sealgraph seal REF
```

Exact CLI spelling, path-byte machine encoding, and the binding schema version
remain implementation-contract decisions. They may not weaken the identities
or mutation boundaries accepted here.

### Non-Git operation

The standard non-Git cycle remains:

```sh
sealgraph source bind REF --file PATH
sealgraph add REF
sealgraph candidate compare REF
sealgraph seal REF
```

The working path is a local input locator, not portable provenance. Rename or
machine relocation uses explicit source rebind. Missing input does not stage a
deletion, drop a REF, alter a Candidate, or create a revision fact.

Cross-machine canonical repository copies normally have no bindings. Each
machine recreates its local binding explicitly and compares exact source bytes
with Candidate or HEAD before updating anything.

### Portable source occurrence provenance

When an operator needs durable evidence of where bytes occurred, the evidence
is ordinary separately sealed content. A generated Seal may name that evidence
Seal through an exact Cause Link.

A Git occurrence claim should contain at least:

```text
system, object_format, repository_identity,
commit_oid, path, blob_oid, file_mode
```

A non-Git file occurrence claim should contain at least:

```text
system, caller_supplied_source_identity,
logical_path, exact_byte_digest, size
```

The existing explicit path-manifest command is a suitable non-Git
path/digest-only claim source. Repository identity, source identity, and path
meaning belong to the claim schema/author, not to hidden environment
inference. Absolute local paths, credentials, mutable remote URLs, hostnames,
and modification times are not portable identity by default.

The occurrence claim is not a second Seal schema and does not make a Git OID a
SealID. Sealgraph core validates the occurrence Seal and exact Cause identity;
the claim's namespace owner defines any Git- or filesystem-specific meaning.

This establishes C-SA-004: portable source location evidence is separately
sealed content, not source-binding state.

### History and Cause-scoped Revision boundary

Git commit parents, file-history queries, rename detection, working-file
timestamps, and filesystem watcher events MUST NOT automatically create a
Seal, Cause Link, or Revision Link.

If ADR 0023 is later accepted, its
`previous_revision_seal_of_target_seal` facts remain explicit Seal-domain
claims. A Git source adapter may display read-only Git history as review
evidence, but it must not populate those facts without an explicit reviewed
Candidate mutation.

This establishes C-SA-005: external source history is evidence and navigation,
not authoritative Seal revision semantics.

## Alternatives

### Make Git the primary repository mode

Rejected because it would violate standalone-default behavior and make
non-Git operation incomplete or environment-dependent.

### Store binding state inside every Seal

Rejected because local paths, repository locations, and source availability
are machine-specific and would make equal bytes acquire different SealIDs.

### Use a Git commit or blob OID as native Seal identity

Rejected because Git identity is typed by repository object format and object
kind, while Seal identity commits to Sealgraph canonical material and Cause
provenance.

### Infer Seal Revision from Git file history

Rejected because file history is derived and may depend on path and rename
heuristics. Git commit ancestry and Seal Revision express different claims.

### Persist a zero-copy external Git reference in the first adapter

Rejected for the first slice. It would make ordinary Seal reads depend on an
external repository and requires a separate availability, trust, migration,
and canonical-schema decision.

## Consequences

Good:

- Non-Git and Git inputs share one explicit Candidate/seal review boundary.
- Existing standalone path binding remains valid and sufficient.
- Git bindings are immutable with respect to branch movement.
- Portable provenance can record either Git or non-Git occurrences without
  adding Git fields to core Seal identity.
- Cause-scoped Revision stays independent from external history heuristics.

Bad / Risk:

- A local binding alone cannot reproduce source provenance on another machine.
- Durable occurrence evidence requires an additional reviewed Seal and Cause
  Link.
- Git path-byte representation and linked-worktree behavior need exact tests
  before the Git binding CLI is implemented.
- A missing Git object or partial clone blocks exact materialization; the
  adapter must not fetch implicitly.

Neutral:

- Binding export/import, automatic watch, automatic add, automatic seal, and
  automatic history conversion remain absent.
- The Git sidecar still needs an explicitly selected and proven Git SDK.
- This ADR accepts the responsibility split, not a release or implementation
  schedule.

## Implementation Notes

Implementation Action A-SA-001 is to introduce a versioned source-binding
adapter interface without changing Candidate or Seal schemas. Legacy v1 path
bindings decode as `WorktreePath` and are not rewritten on read.

Implementation Action A-SA-002 is to implement `GitTreeEntry` only in
`git-sealgraph`, resolve movable input to exact identities before one binding
write, and add regular-blob, linked-worktree, object-format, partial-clone,
filter, path-byte, mismatch, and no-mutation fixtures.

Implementation Action A-SA-003 is to materialize exact adapter bytes through
the existing native ObjectWriter/Candidate boundary. `seal` remains adapter-
independent and Candidate-only.

Implementation Action A-SA-004 is to document portable occurrence claim
schemas as application content contracts. Core must not infer or require those
claims for ordinary source binding.

No implementation action authorizes Git discovery in standalone code,
automatic network fetch, source-watch behavior, automatic Candidate mutation,
or automatic Seal/Link publication.

## Review

- Specialist perspective: the authoring review found the separation of Git
  physical identity, local binding state, native content identity, and portable
  provenance claim coherent. No independent specialist review is claimed.
- Non-specialist perspective: both Git and non-Git follow bind, add, inspect,
  and seal; only the binding resolver differs.
- Root-chain review: A-SA-001 through A-SA-004 trace to this Accepted ADR,
  C-SA-001 through C-SA-005, and E-SA-001 through E-SA-004.

## Evidence

- E-SA-001: ADR 0002 and the standalone requirements prohibit implicit Git
  discovery and define `git sealgraph` as a separate surface.
- E-SA-002: ADR 0019 and `internal/repository/source.go` implement one
  non-canonical REF/path binding and Candidate-only publication.
- E-SA-003: ADR 0011 and `docs/integrations.md` separate typed Git physical
  identity and commit-tree views from native Seal identity.
- E-SA-004: Explicit operator confirmation on 2026-08-28 selected the
  non-Git-default, adapter-only Git, separately sealed occurrence policy.

## Follow-ups

- Specify the exact source-binding v2 bytes and CLI in an implementation slice.
- Select and prove the Git SDK capability matrix before Git binding runtime
  work.
- Add example Git and non-Git occurrence claim schemas without making either a
  core read dependency.
- Keep ADR 0023's Revision aggregation decisions independent from adapter
  history.

## Acceptance Record

The operator explicitly accepted this responsibility split and provenance
boundary on 2026-08-28. Implementation remains separately sequenced.
