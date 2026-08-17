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
move, tag-preserving logical load, exact file/stdin content ingestion, and a
deterministic explicit-path digest manifest builder.
The beta candidate also includes read-only full-inventory `fsck` with
versioned JSON output and explicit historical/detached inventory reporting.
The prior format-3 dump remains available from commit `5b24d47` for explicit
source export. The normative requirements are in
[`docs/requirements.md`](docs/requirements.md); the frozen native byte contract
and migration boundary are in [`docs/storage-format.md`](docs/storage-format.md),
ADR 0011, ADR 0012, and ADR 0013.

The tracked project `.sealgraph` is now format 4. It was converted explicitly
through the commit-`5b24d47` read-only logical dump and the format-4 empty-target
loader; it was not opened or rewritten in place by the new runtime. The
conversion and same-material sibling receipt is recorded in
[`docs/process/dogfooding-receipts/2026-08-17-format4-load.md`](docs/process/dogfooding-receipts/2026-08-17-format4-load.md).

## Standalone beta surface

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

`v0.1.0-beta.2` is standalone-only. It has no file synchronization, watcher,
working-tree reconciliation, Git discovery, or Git sidecar. `manifest` reads
only explicitly named files and emits a deterministic path/digest claim; it
does not track or synchronize those files. The `git-sealgraph` source entry
point is an unreleased placeholder for a separate future adoption decision.

Attachment-bearing repositories can be read, inspected, and converted, but
this beta does not expose `attach` or `detach` mutation commands.

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

Discover the complete invocation contract from the binary itself:

```sh
sealgraph --help
sealgraph help add
sealgraph add --help
sealgraph help candidate show
sealgraph help selectors
sealgraph help concepts
sealgraph help usecases
```

Command help includes argument cardinality, option repeatability and conflicts,
important provenance invariants, examples, and related explicit next actions.
Usage and domain errors point back to the same help topics. Navigation detects,
explains, and suggests inspection or explicit review; it never repairs,
relinks, reseals, or chooses a REF automatically. Help is repository-independent
and does not inspect Git.

```text
sealgraph init
sealgraph manifest --source SOURCE --file PATH [--file PATH ...]
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

`manifest` is a read-only deterministic path/size/SHA-256 claim builder. It
uses only explicit relative files and an explicit source identity, performs no
Git discovery, and does not store the named files or create a candidate/seal.

Later phases retain the following planned surface:

```text
sealgraph attach
sealgraph detach
```

Potential Git-only helpers are a separate, unadopted product discussion:

```text
git sealgraph init
git sealgraph status
git sealgraph conflicts
git sealgraph resolve
```

## Development

Requires Go 1.26+.

Build and install the standalone development binary:

```sh
make build
install -m 0755 bin/sealgraph "$HOME/.local/bin/sealgraph"
```

Uninstall by removing only the installed binary:

```sh
rm "$HOME/.local/bin/sealgraph"
```

No command installs hooks, services, configuration, or a Git plugin. Release
artifacts are Linux amd64 tar archives containing `sealgraph`, `LICENSE`, and
`README.md` only.

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
