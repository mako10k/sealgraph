# sealgraph

`sealgraph` is a local provenance-sealing CLI.

It records logical content as immutable seals, links each seal to the exact upstream seal generations it depended on, and derives the impact of upstream supersession through an N:M DAG.

The core question is not only “what changed?” but:

> **Which exact generation of each upstream basis was reviewed when this content was sealed, and what became stale after an upstream supersession?**

## Status

The checked-in standalone runtime now implements the format-4 native core and
active revision graph: REF-independent Seal and Link identity, separated
candidate revision/CAS state, exact selectors, atomic logical-v1 load,
branching parents, active-leaf admission, stale/frontier, history, and bounded
impact, plus collision-free REF manifests, immutable scoped tags, atomic REF
move, and tag-preserving logical load.
The prior format-3 dump remains available from commit `5b24d47` for explicit
source export. The normative requirements are in
[`docs/requirements.md`](docs/requirements.md); the frozen native byte contract
and migration boundary are in [`docs/storage-format.md`](docs/storage-format.md),
ADR 0011, ADR 0012, and ADR 0013.

The tracked project `.sealgraph` intentionally remains format 3 until the next
explicit dogfood-conversion slice; the format-4 runtime does not open it
directly or migrate it in place.

## Two product surfaces

### Standalone

```sh
sealgraph init
sealgraph add REQ-001 --root --content 'Authentication is required.'
sealgraph seal REQ-001

sealgraph show REQ-001
sealgraph tag REQ-001 reviewed/1.0
sealgraph mv REQ-001 requirements/REQ-001

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

The first sidecar may present exact staged/commit `.sealgraph` trees and merge
stages to the same native validators while keeping Sealgraph provenance
semantics separate from Git commit semantics.

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

If `ROOT` publishes an active child revision from `R2` to `R3`, existing seals
do not change:

```text
ROOT@R3 HEAD

REQ@Q4 HEAD    -> ROOT@R2   direct stale
DESIGN@D7 HEAD -> REQ@Q4    transitive stale
```

Review is intentionally explicit and sequential. Each affected REF must be
relinked/reviewed and publish one new revision. There is no recursive repair
command.

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
sealgraph add --parent
sealgraph derive
sealgraph link
sealgraph unlink
sealgraph tag
sealgraph mv
sealgraph candidate
sealgraph seal
sealgraph show
sealgraph log
sealgraph linklog
sealgraph diff
sealgraph status
sealgraph stale [--frontier] [--refs-only] [--scan]
sealgraph impact [--all-paths] [--max-paths N]
sealgraph graph
sealgraph load --format logical-v1
```

The format-4 binary consumes, but never directly opens, a format-3 migration
source. Load requires an absent `.sealgraph`, stages and validates the complete
repository, publishes it with atomic no-replace semantics, and emits every
old-to-new SealID mapping while preserving rewritten tags inside REF
manifests.

Later phases retain the following planned surface:

```text
sealgraph attach
sealgraph detach
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
npm ci
npm run clone-check
make complexity-check
make deadcode-check
```

The pinned jscpd, gocyclo, and deadcode checks are development/CI tooling only;
they are not sealgraph runtime dependencies. Functions with cyclomatic
complexity above 20 and RTA-unreachable functions (including test entrypoints)
fail the build.

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
