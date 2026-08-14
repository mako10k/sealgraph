# CLI contract

Status: initial command contract.

## 1. Standalone

### `sealgraph init`

Always initializes standalone `.sealgraph`.

It never inspects Git.

### `sealgraph add`

Creates/updates one REF's working candidate.

Examples:

```sh
sealgraph add ROOT-001 --root --content 'External premise'

sealgraph add REVIEW-001 --draft --content 'Provisional review' \
  --depend-on REQ-001@<seal-id>

sealgraph add DESIGN-001 \
  --content 'Design' \
  --depend-on ROOT-001 \
  --depend-on REQ-001@<seal-id>
```

`--depend-on REF` resolves REF HEAD immediately and records the concrete seal identity in the candidate.

Each `add` invocation sets the candidate's `root` and `draft` flags from that
invocation. Re-adding without `--draft` is the explicit Phase 1 way to move a
reviewed candidate out of draft; dependencies are retained unless one or more
`--depend-on` arguments replace the candidate dependency set.

### `sealgraph link`

Changes dependencies without forcing content replacement.

```sh
sealgraph link DESIGN-001 --depend-on REQ-001
sealgraph link DESIGN-001 --depend-on REQ-001@<seal-id>
```

### `sealgraph unlink`

Removes a candidate dependency explicitly.

### `sealgraph attach` / `detach`

Changes candidate attachments.

### `sealgraph seal REF -m MESSAGE`

Creates one immutable seal for one REF.

Message is required in the initial contract.

There is no batch seal.

Normal non-draft seal validation includes DAG validity and dependency freshness rules defined by requirements.

In native v1, a normal seal requires a HEAD-consistent complete dependency
closure. A draft may preserve an explicit historical dependency. There is no
generic `--force` or ignore-validation option. Root seals have no dependencies;
all non-root seals, including drafts, require at least one.

### Inspection

```sh
sealgraph show REF
sealgraph show REF@<seal-id>

sealgraph diff REF
sealgraph diff REF@<old> REF@<new>

sealgraph status
sealgraph status REF

sealgraph log REF
sealgraph linklog REF
sealgraph impact REF
sealgraph graph
sealgraph stale
sealgraph fsck
```

## 2. Status vocabulary

At minimum, output should be able to distinguish:

- CLEAN
- UNSEALED / candidate changed
- DRAFT
- STALE_DIRECT
- STALE_TRANSITIVE

These are orthogonal where appropriate. For example a REF can have an unsealed candidate and a stale current seal.

## 3. Historical dependencies

Historical dependency selection is a first-class feature, not corruption.

```sh
sealgraph link DESIGN --depend-on REQ@<older-seal>
```

The output must make the resulting non-HEAD relationship obvious.

Draft is the primary intended mechanism for “reviewed only up to this generation for now”.

Do not add a generic `--force` that suppresses provenance checks.

## 4. Git plugin

Git integration is invoked as:

```sh
git sealgraph <command>
```

and implemented by `git-sealgraph`.

Initial sidecar surface:

```sh
git sealgraph init
git sealgraph status
git sealgraph conflicts
git sealgraph resolve REF
```

`git sealgraph resolve` is a conflict-resolution assistant, not a merge engine.

For a three-way REF conflict it should be able to show:

```text
BASE   <seal-id>
OURS   <seal-id>
THEIRS <seal-id>
```

and expand semantic differences among those seals.

Selecting ours/theirs changes the merge result REF. It does not create a new seal or claim that a review occurred.

## 5. Exit behavior

Initial convention to refine during implementation:

- 0: command completed successfully,
- 1: domain/status condition intended for scripting,
- 2: CLI usage error,
- other nonzero: operational/integrity failure.

Machine-readable output contracts should be versioned before external consumers depend on them.
