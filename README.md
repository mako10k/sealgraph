# sealgraph

`sealgraph` is a local provenance-sealing CLI.

It records logical content as immutable seals, links each seal to the exact upstream seal generations it depended on, and derives the impact of upstream supersession through an N:M DAG.

The core question is not only “what changed?” but:

> **Which exact generation of each upstream basis was reviewed when this content was sealed, and what became stale after an upstream supersession?**

## Status

The native standalone slices implement repository creation and sealing,
direct/transitive graph inspection, immutable seal history, link history, and
semantic generation diff, plus the deterministic read-only format-3 logical
dump required before the format-4 transition. The normative requirements are in
[`docs/requirements.md`](docs/requirements.md); the frozen native byte contract
and migration boundary are in [`docs/storage-format.md`](docs/storage-format.md),
ADR 0011, and ADR 0012. The checked-in runtime and tracked dogfood remain
format 3.

## Two product surfaces

### Standalone

```sh
sealgraph init
sealgraph add REQ-001 --root --content 'Authentication is required.'
sealgraph seal REQ-001

sealgraph add DESIGN-001 \
  --content 'JWT authentication design' \
  --depend-on REQ-001
sealgraph seal DESIGN-001

sealgraph tag DESIGN-001 reviewed
sealgraph show DESIGN-001@reviewed
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
- canonical format/version metadata.

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
sealgraph tag
sealgraph seal
sealgraph show
sealgraph log
sealgraph linklog
sealgraph diff
sealgraph status
sealgraph stale [--frontier] [--refs-only]
sealgraph impact
sealgraph graph
sealgraph dump --format logical-v1
```

The logical dump is a versioned migration artifact. It rejects any working
candidate and never changes `.sealgraph/`; format-4 load remains separately
gated and is not implemented by this slice.

`stale --refs-only` emits the complete stale REF set as a stable one-REF-per-line
stream. Adding `--frontier` selects only the upstream-most stale REFs to review
before their stale downstreams. It is a read-only observation, not a batch
reseal, reservation, or automatic reseal plan.

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
