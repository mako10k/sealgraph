# sealgraph

`sealgraph` is a local provenance-sealing CLI.

It records logical content as immutable seals, links each seal to the exact upstream seal generations it depended on, and derives the impact of upstream supersession through an N:M DAG.

The core question is not only “what changed?” but:

> **Which exact generation of each upstream basis was reviewed when this content was sealed, and what became stale after an upstream supersession?**

## Status

The checked-in standalone runtime now implements the format-4 native core:
REF-independent Seal and Link identity, separated candidate revision/CAS
state, exact selectors, and atomic logical-v1 load into an absent repository.
The prior format-3 dump remains available from commit `5b24d47` for explicit
source export. The normative requirements are in
[`docs/requirements.md`](docs/requirements.md); the frozen native byte contract
and migration boundary are in [`docs/storage-format.md`](docs/storage-format.md),
ADR 0011, and ADR 0012.

Active revision indexing, normal non-root admission, stale/status/graph/history,
`derive`, `add --parent`, tags, and REF move are the next separately sequenced
slices. The tracked project `.sealgraph` intentionally remains format 3 until
the tag contract allows lossless dogfood conversion; the format-4 runtime does
not open it directly.

## Two product surfaces

### Standalone

```sh
sealgraph init
sealgraph add REQ-001 --root --content 'Authentication is required.'
sealgraph seal REQ-001

sealgraph show REQ-001

# Explicit conversion into a different directory with no .sealgraph target:
sealgraph load --format logical-v1 < repository.dump.json
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

A format-4 Seal commits to:

- content blob identity,
- attachments,
- exact upstream target seal identities,
- optional exact parent revision identity,
- root/draft semantics,
- canonical format/version metadata.

REF paths, selector spelling, tags, publication expectation, actor, and time do
not enter Seal identity. Multiple REFs may point to the same Seal.

Actor, time, and seal-operation rationale are not implicit seal fields. When a
domain needs them, it seals the claim as separate content and links it to the
exact subject seal.

The design is Merkle-DAG-like: a downstream seal includes direct upstream seal identities; those upstream seals already commit to their own upstream dependencies.

## Core CLI

```text
sealgraph init
sealgraph add
sealgraph link
sealgraph unlink
sealgraph candidate
sealgraph seal
sealgraph show
sealgraph load --format logical-v1
```

The format-4 binary consumes, but never directly opens, a format-3 migration
source. Load requires an absent `.sealgraph`, stages and validates the complete
repository, publishes it with atomic no-replace semantics, and emits every
old-to-new SealID mapping. Tag-bearing input fails closed until `TAG_CONTRACT`.

Later phases retain the following planned surface:

```text
sealgraph attach
sealgraph detach
sealgraph fsck
sealgraph derive
sealgraph add --parent
sealgraph tag
sealgraph log
sealgraph linklog
sealgraph diff
sealgraph status
sealgraph stale [--frontier] [--refs-only] [--scan]
sealgraph impact
sealgraph graph
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
npm ci
npm run clone-check
```

The pinned jscpd check is development/CI tooling only; it is not a sealgraph
runtime dependency.

See:

- [`AGENTS.md`](AGENTS.md)
- [`docs/index.md`](docs/index.md)
- [`docs/requirements.md`](docs/requirements.md)
- [`docs/architecture.md`](docs/architecture.md)
- [`docs/storage-format.md`](docs/storage-format.md)
- [`docs/cli.md`](docs/cli.md)
- [`docs/integrations.md`](docs/integrations.md)
- [`docs/process/dogfooding-plan.md`](docs/process/dogfooding-plan.md)
- [`docs/process/release-checklist.md`](docs/process/release-checklist.md)
