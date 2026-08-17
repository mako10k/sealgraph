# ADR 0013: REF manifests, scoped tags, and atomic move

Status: accepted on 2026-08-17.

## Context

ADR 0005 maps one logical REF directly to
`.sealgraph/refs/seals/<REF>`. ADR 0006 separately maps a scoped tag to
`.sealgraph/refs/tags/<REF>/<ENCODED_TAGNAME>`. Those two loose-path layouts
cannot represent all path-form scopes without file/directory collisions. They
also require more than one canonical path to move a REF and all of its tags.

Format 4 deliberately removes REF ownership from immutable Seal and Link
bytes, but its tag storage and REF move were left gated by ADR 0011. The
tracked format-3 repository has not yet been loaded into format 4, so this is
the last useful point to choose one collision-free format without a dual
reader or automatic migration.

## Decision

### One canonical manifest per REF

Each logical REF is one compact canonical manifest at:

```text
.sealgraph/refs/seals/<REF>/.ref
```

`.ref` is a reserved terminal marker, not a REF component. The logical REF
still uses the accepted slash-separated path grammar byte-for-byte and without
case folding, escaping, cleaning, or Unicode normalization. The marker layout
allows `design` and `design/api` to coexist as distinct logical REFs.

The manifest schema is `sealgraph/ref/v1`. Its exact member order is:

```text
schema, head, tags
tag: name, target
```

Illustrative canonical bytes, with no trailing LF, are:

```json
{"schema":"sealgraph/ref/v1","head":"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef","tags":[{"name":"reviewed/1.0","target":"fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"}]}
```

`head` and every tag `target` are full native SealIDs. Tags are sorted by raw
TAGNAME using bytewise UTF-8 order. Duplicate names, invalid names, unknown
members, noncanonical ordering, malformed IDs, trailing bytes, symbolic links,
and non-regular manifest entries fail closed.

The repository config includes:

```text
ref_format = manifest-v1
```

This distinguishes the accepted format-4 REF representation from the interim
plain-head format used before this ADR. The runtime does not dual-read or
automatically migrate that interim representation.

### Immutable scoped tags

A tag remains an immutable external alias in one REF UI scope. Its binding is
stored in that REF's manifest but never enters Seal or Link bytes. Creating an
absent tag rewrites the manifest atomically while requiring the observed HEAD
to remain unchanged. Repeating the same name and target is idempotent;
retargeting, deleting, or force-moving a tag is rejected.

The accepted TAGNAME grammar and injective percent encoding remain useful for
display, interchange, and future adapters, but encoded names are no longer
canonical filesystem leaves. The manifest stores the raw TAGNAME.

CLI creation accepts a current or explicitly REF-scoped selector:

```text
sealgraph tag REF TAGNAME
sealgraph tag REF@SEAL_OR_TAG TAGNAME
```

An unscoped `@SEAL_TOKEN` does not identify the required tag scope and is
rejected. `REF@hex` retains the current-parent-ancestry scope assertion.
Existing tags may continue to name a Seal that later becomes historical or
detached.

### Narrow atomic REF move

The standalone move surface is exactly:

```text
sealgraph mv OLD_REF NEW_REF
```

Both names are explicit valid REFs and must differ. The source must exist and
the destination must be absent. Exact source or destination candidate state
blocks the operation; `mv` never rewrites, moves, discards, seals, or adopts a
candidate. The operator seals or discards source intent explicitly first.

All native mutations remain under the repository-wide writer guard. After
validating the complete source manifest and every referenced Seal, the
implementation prepares only destination directories and performs one atomic
same-filesystem no-replace rename of the `.ref` manifest. That rename is the
commit point, so HEAD and the complete tag namespace move together. Directory
durability is synchronized after the rename; a post-commit durability error
reports that the move may already be visible and requires explicit inspection.
There is no old-name alias, redirect, reflog, automatic retry, or repair.

### Runtime paths

Candidate and per-REF lock files use reserved terminal markers below their
existing runtime roots so prefix REFs can coexist:

```text
.sealgraph/index/<REF>/.candidate
.sealgraph/locks/refs/<REF>/.lock
```

They remain noncanonical and untracked. `mv` does not move them.

## Consequences

- REF/tag prefix collisions are removed without an opaque namespace registry.
- One small canonical file remains the merge/conflict unit for one logical
  REF, and it is directly inspectable in a future Git tree view.
- Adding a tag and advancing a HEAD contend on the same manifest CAS, so one
  cannot silently discard the other.
- A REF rename moves all scoped tags atomically without changing any SealID or
  Link.
- Prefix REFs can coexist, but no hierarchy, inheritance, or recursive move
  semantics are inferred from their spelling.
- The interim format-4 plain REF files and the ADR 0006 `refs/tags` tree are
  deliberately unsupported. Format-3 conversion remains explicit dump/load.

## Approved implementation slice

Approval covers the manifest codec/store, tag listing/creation/resolution,
single-REF `mv`, tag-bearing logical-v1 load, selector integration, focused
crash/no-clobber/collision tests, and synchronized documentation. It does not
cover tag deletion or force, recursive REF moves, candidate moves, automatic
migration, tracked dogfood conversion, Git sidecar behavior, release, or
publication.
