# sealgraph

`sealgraph` is a local provenance-sealing CLI.

It records logical content as immutable seals, links each seal to the exact upstream seal generations it depended on, and derives the impact of upstream supersession through an N:M DAG.

The core question is not only “what changed?” but:

> **Which exact generation of each upstream basis was reviewed when this content was sealed, and what became stale after an upstream supersession?**

## Status

The Phase 1 native standalone vertical slice implements `init`, `add`, `link`,
`seal`, `show`, and direct `status`. The normative requirements are in
[`docs/requirements.md`](docs/requirements.md); the frozen native byte contract
is in [`docs/storage-format.md`](docs/storage-format.md) and ADR 0005.

## Two product surfaces

### Standalone

```sh
sealgraph init
sealgraph add REQ-001 --root --content 'Authentication is required.'
sealgraph seal REQ-001 -m 'Initial requirement'

sealgraph add DESIGN-001 \
  --content 'JWT authentication design' \
  --depend-on REQ-001
sealgraph seal DESIGN-001 -m 'Design reviewed against current requirement'
```

`sealgraph init` is standalone even when run inside a Git working tree. It does not detect or inspect `.git`.

### Git sidecar

Git integration is a separate plugin surface:

```sh
git sealgraph init
git sealgraph status
git sealgraph conflicts
git sealgraph resolve REF
```

The executable name is `git-sealgraph`, which Git exposes as `git sealgraph`.

Sidecar mode may read Git blobs, commits, history, and merge state while keeping sealgraph provenance semantics separate from Git commit semantics.

## Key semantics

```text
ROOT@R2
   |
   v
REQ@Q4
   |
   v
DESIGN@D7
```

If `ROOT` is superseded from `R2` to `R3`, existing seals do not change:

```text
ROOT@R3 HEAD

REQ@Q4 HEAD    -> ROOT@R2   direct stale
DESIGN@D7 HEAD -> REQ@Q4    transitive stale
```

Repair is intentionally explicit and sequential. Each affected REF must be relinked/reviewed and resealed. There is no recursive repair command.

A seal commits to:

- logical REF identity,
- content blob identity,
- attachments,
- exact upstream target seal identities,
- previous seal identity,
- root/draft semantics,
- seal message,
- canonical format/version metadata.

The design is Merkle-DAG-like: a downstream seal includes direct upstream seal identities; those upstream seals already commit to their own upstream dependencies.

## Core CLI

```text
sealgraph init
sealgraph add
sealgraph link
sealgraph seal
sealgraph show
sealgraph status
```

Later phases retain the following planned surface:

```text
sealgraph attach
sealgraph detach
sealgraph unlink
sealgraph diff
sealgraph log
sealgraph linklog
sealgraph impact
sealgraph graph
sealgraph stale
sealgraph fsck
```

Git-only helpers are intentionally separate:

```text
git sealgraph init
git sealgraph status
git sealgraph conflicts
git sealgraph resolve
```

## Development

Requires Go 1.26+.

```sh
go test ./...
go vet ./...
```

See:

- [`AGENTS.md`](AGENTS.md)
- [`docs/index.md`](docs/index.md)
- [`docs/requirements.md`](docs/requirements.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/storage-format.md`](docs/storage-format.md)
- [`docs/cli.md`](docs/cli.md)
- [`docs/integrations.md`](docs/integrations.md)
- [`docs/process/dogfooding-plan.md`](docs/process/dogfooding-plan.md)
