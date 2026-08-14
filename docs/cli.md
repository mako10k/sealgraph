# CLI contract

Status: initial command contract.

## 1. Standalone

### `sealgraph init`

Always initializes standalone `.sealgraph`.

It never inspects Git.

For an existing format-3 repository with valid canonical `config`, `objects`,
`refs/seals`, and `refs/tags`, an explicit `init` invocation also recreates missing empty
`index` or `locks` runtime directories. This supports an outer checkout that
tracks canonical state but ignores runtime-only directories. It never creates,
changes, deletes, or repairs an object or REF, and it rejects unsafe or invalid
existing paths. Read-only inspection commands do not perform this bootstrap
implicitly.

### `sealgraph add`

Creates/updates one REF's working candidate.

Examples:

```sh
sealgraph add ROOT-001 --root --content 'External premise'

sealgraph add REVIEW-001 --draft --content 'Provisional review' \
  --depend-on REQ-001@<seal-id-prefix-or-tag>

sealgraph add DESIGN-001 \
  --content 'Design' \
  --depend-on ROOT-001 \
  --depend-on REQ-001@<seal-id>
```

`--depend-on REF` resolves REF HEAD immediately and records the concrete seal identity in the candidate.

`--depend-on REF@TOKEN` resolves TOKEN as either a 4-to-64-character lower-case
hexadecimal object prefix or a REF-scoped tag. Resolution must be unique and
must identify a canonical seal owned by REF. Only the full seal ID is stored.

Content may also be read without shell/argv round-tripping:

```sh
sealgraph add ADR-0006 --root \
  --content-file docs/adr/0006-experimental-native-v2-ids-and-tags.md
printf '%s' 'exact bytes' | sealgraph add ROOT --root --content-file -
```

`--content` and `--content-file` are mutually exclusive. File and stdin input
preserve bytes exactly. Filesystem input accepts only a regular non-symlink
file; directories, devices, and symlinks are rejected before candidate state is
changed. The source path is not persisted.

Each `add` invocation sets the candidate's `root` and `draft` flags from that
invocation. Re-adding without `--draft` is the explicit Phase 1 way to move a
reviewed candidate out of draft; dependencies are retained unless one or more
`--depend-on` arguments replace the candidate dependency set.

`root` is an attribute of the next seal generation, not a permanent REF type.
Changing it is an explicit candidate change and never mutates an older seal.
Changing to root does not silently remove retained dependencies; they must be
removed explicitly before the root candidate can be sealed. Changing to
non-root likewise requires at least one explicit dependency before sealing.

### `sealgraph link`

Changes dependencies without forcing content replacement.

```sh
sealgraph link DESIGN-001 --depend-on REQ-001
sealgraph link DESIGN-001 --depend-on REQ-001@<seal-id-prefix-or-tag> \
  -m 'Requirement basis used by this design'
```

Native v3 has only one domain-independent dependency edge and no link kind.
`link -m MESSAGE` records optional rationale on each dependency added or
replaced by that invocation. It is part of candidate/seal identity and describes
the dependency relation, not the seal operation or an actor/time claim.

### `sealgraph tag`

Creates or lists immutable tags scoped by one REF:

```sh
sealgraph tag DESIGN-001
sealgraph tag DESIGN-001 reviewed
sealgraph tag DESIGN-001@<seal-id-prefix-or-tag> baseline/2026-08
```

Creating the same tag for the same seal is idempotent. Retargeting, deleting,
or force-moving a tag is not supported. A tag can replace a seal ID in an
explicit selector, but canonical refs, links, candidates, and output always use
the resolved full 64-character lower-case ID.

### `sealgraph unlink`

Removes exactly one candidate dependency explicitly:

```sh
sealgraph unlink DESIGN --upstream REQ
sealgraph unlink DESIGN --upstream REQ@reviewed-v1
```

The bare form removes the unique edge keyed by upstream REF. The qualified form
resolves the token and removes the edge only when the candidate stores that
exact seal generation. Missing edges and generation mismatches are errors. One
invocation accepts one `--upstream`. The command changes no content, root/draft
state, other edge, REF head, or seal.

Removing the last dependency from a non-root candidate is permitted as mutable
intermediate state, but the candidate cannot be sealed until the operator adds
a dependency or explicitly makes the next generation root.

### `sealgraph candidate`

Inspects or explicitly discards one mutable candidate:

```sh
sealgraph candidate show DESIGN
sealgraph candidate diff DESIGN
sealgraph candidate discard DESIGN
```

`candidate show` validates the candidate, content object, direct dependency
ownership, base ownership, and current REF head ownership. It displays the
recorded base, current HEAD, and derived `BASE_STATE`: `INITIAL`, `CURRENT`,
`HEAD_ADVANCED`, `HEAD_MISSING`, or `UNEXPECTED_HEAD`.

`candidate diff` compares content identity, attachments, direct links, root,
and draft with the recorded base. Current HEAD relation is displayed separately.
An initial candidate is compared with absent state. Parent publication is not
invented.

`candidate discard` removes only the exact candidate file. The explicit command
and REF are the confirmation; there is no prompt, `--yes`, or `--force`. It does
not move a REF, delete immutable objects, recurse through candidate paths, or
repair state. Missing and unsafe targets are errors. Corrupt candidate files
may be discarded explicitly.

### `sealgraph attach` / `detach`

Changes candidate attachments.

### `sealgraph seal REF`

Creates one immutable seal for one REF.

Seal accepts no event-message, actor, or timestamp option. Those are not core
seal fields. When a domain needs such evidence, it seals the claim as ordinary
content under another REF and links it to the exact subject generation.

There is no batch seal.

Normal non-draft seal validation includes DAG validity and dependency freshness rules defined by requirements.

In native v3, a normal seal requires a HEAD-consistent complete dependency
closure containing no draft seal. A draft may preserve an explicit historical
dependency and may depend on draft or non-draft generations. There is no generic
`--force` or ignore-validation option. Root seals have no dependencies; all
non-root seals, including drafts, require at least one.

Standalone mutation commands are serialized by one repository-wide writer
guard. A successful target REF CAS publishes the seal. If the candidate differs
from the exact version used for sealing, it is retained and the command reports
that the seal was published but candidate cleanup did not occur.

### Inspection

```sh
sealgraph show REF
sealgraph show REF@<seal-id-prefix-or-tag>
sealgraph show REF@<seal-id-prefix-or-tag> --raw-content

sealgraph candidate show REF [--raw-content]
sealgraph candidate diff REF

sealgraph diff REF
sealgraph diff REF@<old-prefix-or-tag> REF@<new-prefix-or-tag>

sealgraph status
sealgraph status REF

sealgraph log REF
sealgraph linklog REF
sealgraph impact REF
sealgraph graph
sealgraph stale [--frontier] [--refs-only]
sealgraph fsck
```

Phase 2 implements the first graph inspection slice:

- `status [REF]` derives both direct and transitive stale state for current
  heads. A current head may report both when its direct target has advanced and
  that exact historical target seal also contains older stale provenance.
- `stale` prints only current heads with `STALE_DIRECT` or
  `STALE_TRANSITIVE`; it prints `CLEAN` when none exist. It reads no candidate
  state, so `UNSEALED` and candidate-derived `DRAFT` remain `status` concerns.
- `stale --frontier` selects only a stale head whose current seal has no direct
  upstream REF that is itself currently stale. This is an upstream-first
  freshness-review frontier, not readiness, approval, reservation, or a batch
  reseal plan. Draft/historical provenance uses the same factual selection and
  is not described as mandatory work.
- `stale --refs-only`, with or without `--frontier`, emits exactly one valid
  logical REF plus LF per selected head, deduplicated in bytewise lexical order.
  It emits no heading, IDs, labels, quoting, or `CLEAN`; an empty result is zero
  bytes. Empty and non-empty successful results both exit 0.
- `impact REF` starts from a current logical REF and reports every distinct
  path from a current downstream head to a seal link naming that REF. Paths are
  shown downstream-to-upstream and preserve the concrete historical seal IDs
  actually stored in the closure. Historical `REF@SEAL` impact sources are not
  part of this slice.
- `graph` prints current logical REF heads and their concrete direct links. A
  direct target is marked `HEAD` or `HISTORICAL head=<current-id>`.

Graph inspection validates reachable seal ownership and rejects an immutable
seal-ID cycle as an integrity error. It does not repair, relink, or reseal
anything. These commands are read-only and do not persist stale or impact
results.

Every `stale` mode captures the complete current REF/head set, derives against
that captured set, buffers the result, and revalidates all REF heads before
writing stdout. A changed or unreadable observation fails nonzero with empty
stdout and an instruction to rerun. A successful result is still not a
reservation: rerun after each explicit link or seal operation, and `seal`
revalidates dependencies before publication.

History inspection uses sealgraph generations, not Git history:

- `log REF` starts at the current REF head and walks the immutable `parent`
  chain newest first. Every generation must be a canonical seal owned by the
  requested REF, and a repeated parent seal ID is an integrity error.
- `linklog REF [--upstream REF]` walks the same validated history and compares
  each generation with its parent. It reports dependency additions, removals,
  repoints, and dependency-message changes. The optional filter limits events by the exact upstream logical
  REF. This is not a Git reflog and does not claim to record every mutable REF
  file movement.
- `diff REF` compares the current head with its parent. It fails explicitly for
  an initial seal with no parent.
- `diff REF@<old-seal> REF@<new-seal>` compares two exact canonical generations
  owned by the same logical REF. Cross-REF comparison and implicit current
  selectors in the two-argument form are rejected.

Semantic diff reports seal IDs and changes in content identity, attachments,
direct links, root/draft state, and parent. Dependency
changes distinguish add, remove, repoint, and dependency-message change. Attachment changes are keyed by
their unique canonical name. Content bytes are not emitted or text-diffed in
this slice: content is compared by its complete store/type/ObjectID identity,
which keeps human output bounded and safe for binary or control-byte content.
Messages and attachment metadata are quoted in human output.

Default `show` and `candidate show` print content identity, exact byte size, and
at most the first 256 input bytes as a bytewise ASCII-escaped preview. Printable
ASCII is literal except quote/backslash; LF/CR/TAB use `\\n`/`\\r`/`\\t`, and
all remaining bytes use lower-case `\\xhh`. Messages and other arbitrary string
metadata use the same safe quoted-byte representation.

With `--raw-content`, stdout contains the exact content bytes and nothing else:
no metadata, preview, prefix, or added newline. The complete object is validated
before output. Versioned machine-readable output remains a separate
cross-command contract.

`log`, `linklog`, and `diff` validate all required immutable objects before
printing a result. They never create runtime directories, append a log, move a
REF, repair history, or persist their derived output.

The Phase 2 human-readable text is for operator inspection. It is not yet a
versioned machine-readable output contract.

## 2. Status vocabulary

At minimum, output should be able to distinguish:

- CLEAN
- UNSEALED / candidate changed
- DRAFT
- STALE_DIRECT
- STALE_TRANSITIVE

These are orthogonal where appropriate. For example a REF can have an unsealed candidate and a stale current seal.

`STALE_DIRECT` and `STALE_TRANSITIVE` are also orthogonal. A downstream head
may have both during an explicit upstream-to-downstream repair sequence.

## 3. Historical dependencies

Historical dependency selection is a first-class feature, not corruption.

```sh
sealgraph link DESIGN --depend-on REQ@<older-seal-prefix-or-tag>
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

Structured machine-readable output contracts should be versioned before
external consumers depend on them. ADR 0010's deliberately minimal
`stale --refs-only` line stream is the narrow stable exception; future JSON is a
separate format.
