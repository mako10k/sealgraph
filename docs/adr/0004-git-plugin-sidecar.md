# ADR 0004: Git sidecar is exposed as a Git plugin

Status: accepted

## Decision

Git sidecar is implemented by a `git-sealgraph` executable and invoked as:

```sh
git sealgraph ...
```

Do not expose the integration as `sealgraph git ...`.

## Rationale

`sealgraph git init` visually resembles a wrapper around `git init`.

`git sealgraph init` correctly communicates that sealgraph is an extension attached to an existing Git repository.

## Consequences

- `cmd/git-sealgraph` is a distinct entry point,
- core libraries remain shared,
- Git-specific merge/index/history code stays outside standalone initialization.
