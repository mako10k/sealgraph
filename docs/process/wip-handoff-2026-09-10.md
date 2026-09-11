# WIP handoff — 2026-09-10

Status: end-of-day interruption handoff. The optional extensible Cause Link
metadata requirement and ADRs 0029 through 0031 are accepted. The format-6
implementation has not started. The checked-in runtime, normative product
documents, and tracked dogfood remain format 5.

Continuation branch: `codex/cause-derived-revision-links`.

## Completed today

- Accepted requirement revision 2 for optional, namespaced, extensible metadata
  embedded in each Cause Link. Legacy `messages` remain for compatibility and
  no message-to-metadata inference is permitted.
- Confirmed that Cause Links remain embedded Seal records rather than gaining a
  separate first-class object identity.
- Accepted [ADR 0029](../adr/0029-extensible-cause-link-metadata.md), which fixes
  the generic metadata model, namespace ownership, canonical value domain,
  limits, and the deferred query/validator boundary.
- Accepted [ADR 0030](../adr/0030-format6-link-metadata-storage-and-migration.md),
  which fixes Seal v6, Provenance v2, Candidate v6, historical format-5
  projection, v6-only writes, upstream-change/v2 identity, complete `fsck`
  accounting, and the config-only migration boundary.
- Accepted [ADR 0031](../adr/0031-format6-link-metadata-cli-and-output.md), which
  fixes one-target/one-namespace Candidate mutation, legacy authoring
  preservation, successor inspection/comparison schemas, typed `linklog`
  endpoints, and the exact public migration command and receipt.
- Preserved the independent reviews, focused ADR 0031 rereview, complete
  Japanese review support, and acceptance receipts under `docs/process/`.
- Reconciled [PLAN.pert](../../PLAN.pert) and the
  [implementation plan](implementation-plan.md) with the completed format-5
  baseline and the accepted format-6 decision set. No historical work event was
  fabricated during that reconciliation.

## Validation receipt

The current documentation and plan WIP passed:

```text
requirement revision 2 review:       accepted
ADR 0029 independent review:         pass
ADR 0030 independent review:         pass
ADR 0031 focused rereview:            pass
ADR 0029 through 0031 owner review:   accepted
LLMThink plan reconciliation audit:  fatal=0 error=0 warning=0
LLMThink interruption handoff audit: fatal=0 error=0 warning=0
perttool document check PLAN.pert:    OK
perttool dag analyze --schedule both: OK
perttool dag next --format json:      FORMAT6_METADATA_STORAGE_SLICE
git diff --check:                     OK
```

No Go or JavaScript implementation changed, so the full runtime validation
suite was not rerun for this documentation-only WIP.

## Exact next frontier

`PLAN.pert` recommends exactly `FORMAT6_METADATA_STORAGE_SLICE` (5p):

```text
Implement format-6 metadata storage and migration primitives.
```

The planned slice adds Seal v6, Provenance v2, Candidate v6, canonical bounded
metadata, exact historical v5 reading and Candidate projection, v6-only writes,
`upstream-change/v2`, complete `fsck` generation accounting, and an atomic
config-only migration primitive. It explicitly does not migrate this project
repository.

Before starting implementation in the next session:

1. run the repository-start and work-time preflights;
2. read the accepted requirement revision 2 and ADRs 0029 through 0031;
3. recheck the storage slice against the current code and normative documents
   under the planned-task freshness gate;
4. run the required CLI LLMThink audit for the implementation decision; and
5. only then mark `FORMAT6_METADATA_STORAGE_SLICE` active and implement it when
   the user has explicitly instructed that start.

## Explicitly not started or performed

- format-6 storage, codec, reader, writer, migration, `fsck`, CLI, or output
  implementation;
- project-repository format-5-to-format-6 migration;
- normative requirements, architecture, storage-format, or CLI synchronization;
- query language, metadata validator, Assessment command, or Git sidecar work;
- PR creation or merge, release, deployment, publication, or Issue mutation.

The next session should preserve the serial order in `PLAN.pert`: storage and
migration primitives first, CLI and inspection runtime second, then normative
synchronization and full repository validation.
