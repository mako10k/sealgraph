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

## 2. llmthink

Role: design reasoning and semantic audit, not runtime provenance storage.

Existing llmthink conventions use human-readable `.think` documents and a structured reasoning/audit model. This repository keeps a seed document at:

```text
docs/decisions/sealgraph-design.think
```

Suggested optional workflow:

```sh
llmthink dsl audit docs/decisions/sealgraph-design.think --pretty
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
- never copy secret values into seal messages,
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
