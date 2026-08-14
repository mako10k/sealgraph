# ADR 0002: Standalone initialization is unconditional

Status: accepted

## Decision

`sealgraph init` always creates standalone mode and does not inspect Git.

Running inside a Git working tree does not change its behavior.

## Rationale

Repository semantics must be explicit and predictable. Git sidecar is a different product feature, not an environment-dependent default.

## Consequences

- standalone code contains no Git discovery path,
- tests can assert no `.git` access during initialization,
- Git integration is invoked through `git sealgraph`.
