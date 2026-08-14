# WIP handoff — 2026-08-14

Status: end-of-day handoff. The format-4 design is accepted, but the checked-in
runtime and tracked dogfood remain format 3. No format-4 runtime work has
started.

## Completed today

- Accepted [ADR 0011](../adr/0011-ref-independent-seals-and-branching-revisions.md)
  and committed it as `e84b0abbc91e7f84c7b0e61a413b315b7531b6d6`.
- Synchronized `requirements.md`, `architecture.md`, `storage-format.md`,
  `cli.md`, `integrations.md`, the llmthink models, process documents, and
  `PLAN.pert` with the accepted format-4 direction.
- Fixed the immutable Seal shape to content/provenance properties only:
  `schema`, `parent_revision`, `content`, `attachments`, `links`, `root`, and
  `draft`. REF names, Link `target_ref`, stale, actor, time, and operation
  message are absent.
- Defined branching revision ancestry separately from exact Cause Links.
  Stale is derived from the current-REF-rooted active revision DAG. An active
  non-leaf or historical/detached Cause is not reported as clean.
- Fixed the first Git-sidecar seam as read-only views of the same native
  `.sealgraph` paths and bytes. Hooks remain explicit validation-only; Git
  content import and the SDK version are deferred.
- Kept tracked `.sealgraph` dogfood unchanged in format 3.

## Validation receipt

The accepted-design commit passed:

```text
llmthink sealgraph-design:        fatal=0 error=0 warning=0
llmthink seal-revision-dag:       fatal=0 error=0 warning=0
perttool document check PLAN.pert OK
perttool dag analyze PLAN.pert    OK
perttool dag next PLAN.pert       OK
gofmt -w .                       OK
go vet ./...                     OK
go test ./...                    OK
npm run clone-check              0 clones
git diff --check                 OK
```

`perttool dag analyze` emitted only `PTDAG-302`: critical-path enumeration was
truncated at the requested one representative path. The DAG itself validated.

The llmthink `pending` info is intentional. It corresponds to named approval
gates, not a hidden contradiction.

## Exact next frontier

`PLAN.pert` selects `FORMAT3_LOGICAL_DUMP`:

```text
Add deterministic read-only format-3 logical dump.
```

Before implementation, write out and obtain approval for the remaining dump
contract details:

1. public command and versioned output envelope;
2. exact representation and deterministic order of content, attachments,
   Seals, Links, REFs, and tags;
3. whether mutable/corrupt candidates are rejected, omitted, or represented;
4. format-3 REF-scoped tag collision handling;
5. explicit many-old-SealIDs-to-one-format-4-SealID mapping and duplicate
   reporting;
6. format-4 empty-repository load validation and publication boundary;
7. failure behavior that guarantees the dump is read-only and never repairs
   provenance.

Do not mark that follow-up contract accepted without operator approval.

## Resume procedure

1. Verify `main`, clean worktree, and equality with `origin/main`.
2. Read ADR 0011, the `storage-format.md` migration boundary, and this handoff.
3. Re-run `perttool document check PLAN.pert` and
   `perttool dag next PLAN.pert --format json`.
4. Draft and audit the logical-dump contract before changing Go runtime code.
5. After approval, implement only the read-only format-3 dump slice with
   deterministic fixtures and corruption/no-mutation tests.

## Explicitly not started

- format-4 reader/writer or mixed-format support;
- tracked dogfood conversion;
- `derive`, `add --parent`, `mv`, rename-safe tags, cache/`--scan`, or revised
  impact runtime;
- Git SDK, Git sidecar, hooks, or merge assistance;
- release/tag/publication work or external Issue changes.

The next session should preserve this order: format-3 logical dump contract and
implementation first, then format-4 native core, revision graph, and explicit
dogfood load.
