# ADR 0019: Local source binding and content-only add refresh

Status: Accepted

Date: 2026-08-21

## Context

The format-4 runtime can import exact bytes with `add --content-file`, but a
practical standalone workflow needs an explicit local association between one
logical REF and one working file. This repository favors that bounded utility
while larger semantic changes proceed separately in `refgraph-*`.

Git terminology creates hazards here. A sealgraph binding is machine-local
input configuration, not Git tracked membership. A candidate is similar to an
index only at the sealing boundary: `seal` publishes reviewed candidate bytes
and never rereads a working file. Git discovery, watching, automatic add, and
automatic seal remain absent.

The Git-expert review and llmthink analysis are recorded in
`docs/decisions/2026-08-21-local-source-git-ux.think`.

## Decision

### Local source state and namespace

One REF may have one non-canonical local source binding stored at
`.sealgraph/index/<REF>/.track`. It is excluded from the canonical REF
manifest, candidate, Seal, logical dump/load, canonical fsck, SealID, and outer
Git tracking. Its absence never changes canonical repository validity.

The public CLI avoids Git's `track` terminology:

```sh
sealgraph source bind REF --file PATH
sealgraph source show REF
sealgraph source list
sealgraph source rebind REF --from OLD_PATH --file NEW_PATH
sealgraph source unbind REF --from PATH
```

`bind` is expected-absent creation; repeating the same binding is idempotent.
A different binding requires `rebind`, which validates the new file and exact
observed old path before one atomic replacement. `unbind` also requires the
exact observed path. Missing or mismatched state is an error, not a no-op.

`show` and `list` read binding records without opening source files. `show`
requires one existing binding; an empty `list` succeeds. List order is REF
byte order. All source commands share safe path representation and the
`sealgraph/source/v1` JSON schema. Mutation receipts include operation, REF,
before/after path, and `candidate=UNCHANGED`.

A binding may exist before its REF or candidate. Such binding-only state is
visible to source inspection and status, but not canonical fsck or stale.
There is no initial source restore, implicit last operation, binding reflog,
automatic backup, or import. An unbind receipt supplies the exact path needed
for explicit rebinding. Cross-machine export/import requires a later ADR if
real usage demonstrates the need.

### Add source resolution

Explicit `--content` and `--content-file PATH|-` remain mutually exclusive.
When neither is supplied, `add REF` resolves content in this order:

1. the REF's local source binding;
2. the exact REF spelling as a path, only for an absent REF and absent
   candidate.

An existing REF or candidate without a binding never falls back to a
coincidentally named file. It fails and requires `--content-file PATH` or
`source bind`. The REF-as-path creation shorthand performs no cleaning,
search, glob expansion, recursive walk, extension inference, or Git access.

`add --bind-source` persists the named input used by `--content-file PATH` or
the initial REF-as-path shorthand. It is invalid with `--content` and stdin.
It reuses bind's expected-absent/same-binding rule and cannot retarget.

Plain file add remains a one-time import. Every successful add receipt states
REF, source mode, exact safely represented path, and
`SOURCE_BINDING=NONE|BOUND`; NONE tells the operator that a later contentless
refresh requires an explicit file or binding.

When a REF is already bound, an explicit different `--content-file` is
rejected rather than creating a candidate from one path while leaving the next
contentless refresh bound to another. The operator explicitly rebinds or
unbinds first.

### Content-only refresh

Contentless `add REF` is a content-only refresh. It preserves the existing
candidate's root, draft, Links, attachments, parent revision, and publication
expectation. When no candidate exists, those fields originate from current
HEAD through the normal edit baseline. Explicit semantic mutation options may
change only the fields they name. This prevents a routine file refresh from
silently resetting identity-bearing state.

The source file is a portable working-directory-relative regular file. No
symlink path component is followed. The implementation compares file identity,
size, modification time, mode, and exact-byte digest across the read. An
observable replacement or mutation is `CHANGED_DURING_READ` and produces no
plausible candidate result.

### Candidate and binding publication

Candidate and binding are two files and format 4 has no multi-file commit
root. The initial slice therefore does not claim all-or-nothing crash
atomicity. `add --bind-source` publishes the candidate first, then the new
binding. The only permitted interrupted state is updated candidate with no new
binding. A new binding must never identify a source whose candidate update did
not complete. Binding publication failure reports that candidate may already
be updated and requires explicit readback.

### Status

Status treats candidate/HEAD and workfile/baseline as separate axes. Human and
`sealgraph/status/v2` output do not use bare `CLEAN` or `TRACKED_CLEAN` for the
combined state. Working-file relations name their comparison target, for
example:

```text
CANDIDATE_TO_HEAD=UNSEALED
WORKFILE_TO_CANDIDATE=WORKFILE_MATCHES_CANDIDATE
WORKFILE_TO_HEAD=WORKFILE_DIFFERS_FROM_HEAD
```

The candidate is the baseline when present, otherwise current HEAD. A binding
with neither has baseline NONE. Missing and unsafe/unreadable inputs are
`SOURCE_MISSING` and `SOURCE_UNREADABLE`; neither stages deletion. Status uses
stable per-file observations but does not claim a simultaneous filesystem-wide
snapshot.

### Seal, move, and recovery boundaries

`seal REF` reads only the validated candidate. Source binding, status, file
change, hooks, and watchers never add or publish automatically.

An exact binding at either endpoint blocks `mv`. The diagnostic identifies
the operation as REF-only and states that no working path is moved. The
operator inspects, explicitly unbinds, moves the canonical REF, then binds the
new REF. Missing files are restored or unbound explicitly; there is no
`git add -u` analogue and neither action deletes candidate or HEAD content.

## Consequences

- The common bound workflow is `edit -> add REF -> inspect -> seal REF`.
- Canonical format 4 and Seal identity remain unchanged.
- Source configuration is intentionally local and reconstructable.
- Create, read-one, read-all, update, and delete operations are explicit and
  symmetric.
- Git-like convenience does not imply Git tracked membership, checkout,
  deletion staging, file movement, or worktree-wide snapshot semantics.
