# ADR 0012: Format-3 logical dump and atomic format-4 load boundary

Status: accepted on 2026-08-17.

Implementation status: complete. Commit `5b24d47` is the final format-3 dump
binary; the checked-in format-4 runtime implements the separately gated load,
including tag-preserving REF manifests. “Future” and “later” below describe the
sequencing at decision time, not missing work in the current runtime.

## Context

ADR 0011 removes owner REF names and Link `target_ref` from format-4 Seal
identity. The format-4 runtime deliberately does not read a format-3
repository, so the existing graph needs an explicit logical conversion
boundary before the runtime schema changes.

Format-3 repositories may also contain mutable candidates, dangling loose
objects, hierarchical REF-scoped tag paths, and distinct owner-salted SealIDs
that become one format-4 SealID. Guessing any of those cases would silently
change provenance or operator intent.

The complete accepted contract and examples are recorded in
[`../process/format3-logical-dump-proposal-2026-08-17.md`](../process/format3-logical-dump-proposal-2026-08-17.md).

## Decision

### Versioned read-only dump

The format-3 binary provides exactly:

```sh
sealgraph dump --format logical-v1
```

It emits one compact canonical `sealgraph/logical-dump/v1` JSON document plus
LF to stdout. The command has no output path, repair, ignore, compatibility,
or Git behavior. All validation and final observation checks complete before
stdout emission. It never changes `.sealgraph/`.

The exact top-level order is:

```text
schema, source_repository, objects, seals, refs, tags, excluded_objects
```

The document has no time, actor, hostname, source path, tool version, or nonce.
Equal validated logical observations therefore produce equal bytes.

### Logical export boundary

Current REF heads and immutable tag targets are graph roots. The dump includes
every canonical format-3 Seal reachable through parent or Cause edges and the
exact content/attachment blob bytes referenced by those Seals.

The exporter validates every physical loose object. A valid object that is
neither an exported Seal nor referenced material is not copied; its full ID is
reported in sorted `excluded_objects`. This avoids guessing a type for an
untyped arbitrary blob while making the omission explicit.

Any candidate entry rejects the dump, whether valid or corrupt. Candidate
bytes are never serialized. The operator explicitly seals or discards that
mutable intent first.

### Ordering and integrity

Material objects are sorted by full ID. Seals use a deterministic
dependency-first topological order over the union of parent and Cause edges,
with old full SealID as the ready-set tie break. REFs are sorted by name, tags
by `(ref, name)`, and excluded objects by ID.

Every object envelope/hash, canonical Seal, owner relation, REF, tag, material
reference, and graph is validated. The complete object/REF/tag/candidate
observation is revalidated after output buffering. A change or corruption
fails without plausible stdout and never repairs state.

### Tags and sequencing

The dump inventories the complete format-3 tag tree, rejects ambiguous or
corrupt physical attribution, and records logical `(REF, TAGNAME, target)`
triples. It does not select a format-4 storage path.

A future format-4 load rejects non-empty tags until the separately accepted
rename-safe tag namespace exists. It never drops tags or writes a private
deferred manifest. Consequently `TAG_CONTRACT` precedes tag-bearing
`FORMAT4_DOGFOOD_LOAD` in the implementation plan.

### Future load and mapping receipt

The future format-4 surface is:

```sh
sealgraph load --format logical-v1 < repository.dump.json
```

It parses the migration document rather than opening a format-3 repository.
The target `.sealgraph` must be absent. Load builds and validates a complete
format-4 repository in sibling staging and publishes it with one atomic
no-replace directory operation. It never merges with or replaces an existing
repository.

The load receipt includes every exported old-to-new SealID pair and every
many-old-to-one-new collapse group. The mapping is output evidence, not hidden
mutable repository state. Excluded objects have no Seal mapping.

Load is not implemented by the format-3 dump slice and remains separately
gated with the format-4 runtime.

## Consequences

- The current format-3 graph receives a deterministic migration artifact
  without a dual repository reader or in-place rewrite.
- Unsealed candidate intent cannot disappear during migration.
- Dangling objects remain in the source and are visible by identity without
  polluting the new repository automatically.
- Tag-bearing dogfood conversion waits for its canonical format-4 namespace.
- A later load can prove non-injective Seal identity conversion explicitly.
- The dump may require memory or external stdout storage proportional to the
  reachable material because it validates and buffers before emission.

## Approved implementation slice

Approval covers the logical-v1 model/encoder, strict format-3 inventory,
repository dump orchestration, CLI, deterministic/corruption/no-mutation
tests, and synchronized documentation. It does not cover format-4 Seal or
candidate code, load, tracked dogfood conversion, tag storage, REF move,
Git sidecar, release, or external tracker mutation.
