# Integrations

Integrations are optional developer/product adapters. They must not become hidden prerequisites of the standalone core.

## 1. Git sidecar

Git sidecar is a separate explicit product surface:

```text
executable: git-sealgraph
invocation: git sealgraph ...
```

ADR 0011 fixes one native format for both entry points. The sidecar does not
introduce Git-backed SealIDs, another Seal schema, or Git commit/branch meaning
inside the revision/Cause DAG.

The first integration seam is a read-only exact path/byte view of canonical
`.sealgraph` files from:

- outer worktree diagnostics;
- the prospective staged result tree;
- one immutable commit tree;
- merge stage 1/2/3 conflict entries associated with validated
  BASE/OURS/THEIRS complete trees.

These views feed the same native config/object/REF decoders and domain graph as
standalone. Only the real worktree `.sealgraph` is mutable, through the same
native writer and one-REF CAS publication protocol.

Use a mature Git SDK for physical repository, index, tree, pack, alternate, and
linked-worktree reading. Do not hand-roll packfiles/deltas or silently fall back
to Git CLI. Pin an exact stable version only after one released binary proves
its supported SHA-1/SHA-256 capability matrix. Current SDK-version selection is
a deferred gate, not a native-domain API choice.

Standalone Git compatibility does not require the SDK. It remains an explicit
Git SHA-256 low-level loose-object conformance test and does not open
`.sealgraph` as a Git repository, attach it to an outer ODB, or invoke Git
maintenance.

Prospective staged-tree validation checks canonical layout, native object hash,
REF/tag reachability, complete parent/Cause closure, and forbidden runtime
paths. It captures/revalidates the complete index observation. Unsupported Git
or native format, transformed canonical bytes, concurrent index change, or
missing partial-clone object fails without native mutation, automatic fetch,
dual reader, or repair.

Hook integration is explicit, opt-in, and validation-only. It never
self-installs, overwrites another hook, stages, seals, advances a REF, relinks,
repairs, commits, or pushes. Exact hook setup/dispatch CLI is separately gated.

Git source blob/tree/commit/tag material import is deferred. Later exact blob
materialization can use the native ObjectWriter without changing format 4;
zero-copy external references or type-specific projections require a separate
persisted contract and ADR.

## 2. llmthink

Role: design reasoning and semantic audit, not runtime provenance storage.

Existing llmthink conventions use human-readable `.think` documents and a structured reasoning/audit model. This repository keeps a seed document at:

```text
docs/decisions/sealgraph-design.think
```

The 2026-08-14 external-spec review and mitigation analysis is modeled at:

```text
docs/decisions/2026-08-14-external-spec-review.think
```

Suggested optional workflow:

```sh
llmthink dsl audit docs/decisions/sealgraph-design.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-external-spec-review.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-candidate-lifecycle.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-seal-event-metadata.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-reseal-required.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-seal-revision-dag.think --pretty
llmthink dsl audit docs/decisions/2026-08-17-format3-logical-dump.think --pretty
```

When a major architecture choice changes:

1. update the relevant ADR,
2. update the `.think` decision document when it models that decision,
3. optionally audit with llmthink,
4. keep sealgraph tests as the executable source of truth.

Do not make llmthink a build dependency.

## 3. secdat

Role: secret-safe execution and developer/release credential supply.

Sealgraph core SHOULD work without any secret.

If future integration/release tests need credentials, prefer an explicit secdat execution boundary rather than plaintext `.env` files or committed test credentials.

Example shape only:

```sh
secdat exec ... -- go test ./...
```

Exact supply/route/demand policy belongs to deployment-specific configuration, not this repository scaffold.

Rules:

- never put secret plaintext in `.sealgraph`,
- never copy secret values into seal content or link messages,
- never include decrypted secrets in fixtures/snapshots/logs,
- do not add a hard runtime dependency on secdat,
- if command JSON/output from secdat is consumed, use its secret-safe/preflight surfaces rather than scraping plaintext.

## 4. perttool

Role: development planning and execution order, not sealgraph provenance semantics.

A seed plan is stored as:

```text
PLAN.pert
```

Suggested optional checks:

```sh
perttool document check PLAN.pert
perttool dag analyze PLAN.pert
perttool dag next PLAN.pert --format json
```

The plan may track implementation milestones such as:

- canonical format,
- native repository,
- stale graph,
- CLI vertical slice,
- Git sidecar,
- merge conflict assistant.

Do not import PERT task/gate semantics into sealgraph unless separately specified.

## 5. Boundary summary

```text
llmthink -> why/decision/audit during development
perttool -> what next / project sequencing
secdat   -> secret-safe execution boundary
sealgraph -> product provenance sealing semantics
```

These tools should complement each other without creating circular runtime dependencies.

## 6. Clone detection

jscpd is pinned through `package-lock.json` and runs only as development/CI
tooling:

```sh
npm ci
npm run clone-check
```

The scan covers Go implementation and tests under `internal/` and `cmd/` using
the repository `.jscpd.json`. It is not imported by, invoked by, or required at
runtime by either sealgraph executable.

## 7. Complexity and dead-code analysis

Cyclomatic complexity and whole-program reachability are separate from
`go vet`. The Makefile pins their development-only versions and CI runs both:

```sh
make complexity-check  # gocyclo v0.6.0; fail above 20 per function
make deadcode-check    # x/tools deadcode v0.49.0; include test executables
```

The commands use versioned `go run` tool modules and do not add runtime or
library dependencies to `go.mod`. A deadcode report requires judgment, but no
reported function is accepted silently: remove it, exercise the intended
entrypoint, or document and test a platform-specific build configuration.
