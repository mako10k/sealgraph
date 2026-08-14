# Integrations

Integrations are optional developer/product adapters. They must not become hidden prerequisites of the standalone core.

## 1. Git sidecar

Git sidecar is the only product-level integration in the initial architecture.

Executable:

```text
git-sealgraph
```

User invocation:

```sh
git sealgraph ...
```

It may read:

- Git blobs,
- trees,
- commits,
- object database,
- first-parent/history data where needed,
- index stages for three-way conflict analysis,
- merge state.

Use a mature Git SDK for physical Git reading. Do not hand-roll packfile/delta/alternate/worktree support.

Initial dependency candidate when implementation reaches this phase:

```text
github.com/go-git/go-git/v5
```

Pin an exact stable version when added.

Standalone Git compatibility does not require this dependency. Native
conformance is limited to an explicit Git SHA-256 low-level loose-object test;
it does not open `.sealgraph` as a Git repository or attach its object directory
to an outer SHA-1 repository.

The first SDK adoption point is the Git-sidecar `GitObjectReader`, where pack,
alternate, repository object-format, and prefix lookup support justify a mature
library. Native `.sealgraph/objects` keeps its small explicit loose-blob
implementation and proves compatibility with bidirectional Git CLI conformance
tests. Current go-git SHA-256 support requires its documented SHA-256 build
mode; that build contract must be fixed and tested when the sidecar dependency
is introduced.

Before that product seam is implemented, an ADR must decide supported Git
object formats, typed external identities, whether content is materialized or
referenced, and which blob/tree/commit semantics are first-class. The current
native `ObjectID` must not be generalized speculatively only to make room for an
unselected sidecar design.

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
