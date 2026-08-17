# CLI contract

Status: the checked-in runtime implements the format-4 native core, load, and
active revision/Cause graph commands, plus REF-manifest tags and atomic move.

## 0. CLI discovery and diagnostics

The standalone binary is self-describing without opening `.sealgraph` or
inspecting Git:

```sh
sealgraph --help
sealgraph help
sealgraph help add
sealgraph add --help
sealgraph help candidate show
sealgraph candidate show --help
sealgraph help selectors
sealgraph help concepts
sealgraph help usecases
```

Root help is a compact command/topic index. Command help is the invocation
contract: purpose, Usage, positional operands, required/optional/repeatable
options, defaults, dependencies or conflicts, important invariants, examples,
and related next actions. `help concepts` explains the domain model;
`help selectors` is the exact selector grammar; and `help usecases` contains
copyable multi-command review workflows. These views are generated from one
runtime help registry rather than independent command-local prose.

Human diagnostics use a stable navigation shape where applicable:

```text
error: <failed operation>
reason: <violated contract or invariant>
usage: <accepted invocation shape>
hint: <one or more explicit inspection or correction steps>
help: sealgraph help <topic>
```

Usage errors exit 2 before repository traversal. Operational, integrity, and
domain-invariant failures retain their existing nonzero behavior and add only
human navigation. Hints never select a REF, relink, reseal, repair, or invent a
`--force` bypass. Typo suggestions are display-only and require the operator to
review and execute a command explicitly.

This slice does not introduce stable diagnostic codes or a JSON error schema.
`--format human|json` continues to select successful inspection output only;
errors remain on stderr and successful command-specific JSON schemas are
unchanged. A future machine diagnostic contract requires an independently
versioned schema and compatibility decision.

## 1. Common selector grammar

Commands that resolve an immutable Seal use:

| Selector | Meaning |
| --- | --- |
| `REF` | exact current HEAD of REF |
| `@SEAL_TOKEN` | repository-wide unique ODB prefix that decodes as a canonical Seal |
| `REF@TOKEN` | Seal selected in a REF UI scope |

Hexadecimal tokens contain 4 through 64 lower-case hex characters. `REF@hex`
requires the Seal to equal or be a `parent_revision` ancestor of current REF
HEAD. `REF@non-hex` resolves an immutable tag. Siblings and detached Seals use
`@SEAL_TOKEN`. Bare Seal tokens are not accepted because a lower-hex REF is
valid. Only full resolved SealIDs are persisted or emitted as identity
receipts.

## 2. Standalone mutation

### `sealgraph init`

Always initializes standalone `.sealgraph` and never inspects, detects,
mentions, or configures Git.

For an existing format-4 repository with valid canonical `config`, `objects`,
and `refs`, explicit `init` may recreate missing empty runtime `index` and
`locks` directories. It never creates, changes, deletes, migrates, or repairs a
canonical object or REF. Read commands do not bootstrap implicitly.

Successful output distinguishes all outcomes without exposing the checkout
path: `INITIALIZED standalone repository runtime=index,locks`,
`BOOTSTRAPPED_RUNTIME index,locks` (only labels actually created), or
`ALREADY_COMPLETE`.

### `sealgraph add`

Creates or updates one destination REF's candidate:

```sh
sealgraph add ROOT-001 --root --content 'External premise'

sealgraph add DESIGN-001 \
  --content 'Design' \
  --depend-on ROOT-001 \
  --depend-on @<exact-historical-seal-prefix>

sealgraph add revised/api \
  --parent design/api@<ancestor-prefix> \
  --content 'new material'
```

`--depend-on REF` resolves current HEAD immediately. `--depend-on
REF@TOKEN` and `--depend-on @SEAL_TOKEN` resolve an explicit Seal. Persisted
Links contain only exact full target SealIDs.

`--content` and `--content-file PATH_OR_DASH` are mutually exclusive. File and
stdin inputs preserve exact bytes. Filesystem input accepts one regular
non-symlink file; directories, devices, and symlinks fail before candidate
state changes. The source path is not persisted.

### `sealgraph manifest`

```sh
sealgraph manifest --source SOURCE \
  --file docs/requirements.md \
  --file docs/architecture.md
```

This read-only command emits one canonical `sealgraph/path-manifest/v1` JSON
document plus LF. `SOURCE` is required exact caller input and is never inferred
from Git or the environment. Each `--file PATH` uses the same explicit
working-directory-relative slash path as both the read source and the semantic
manifest path. Entries are sorted bytewise, exact file bytes use SHA-256, and
the aggregate is SHA-256 over the exact canonical JSON `entries` array.

Paths are valid UTF-8 portable relative paths with no empty, `.`, `..`,
backslash, control, or DEL component. Duplicate, missing, directory, symlink
in any component, FIFO, socket, and device inputs reject the complete output.
There is no glob expansion, recursive walk, normalization, Git discovery, path
mapping, object write, candidate mutation, Link, or seal.

The fixed `claim` value is `path-digest-only`: named files are not stored as
attachments merely because their paths and digests appear. The resulting bytes
may be reviewed and then supplied explicitly to `add --content-file -`.

Each ordinary `add` updates content and explicitly sets root/draft from that
invocation. Existing dependencies are retained unless one or more
`--depend-on` arguments replace the set. Changing root never edits Links
implicitly.

`add NEW_REF --parent SOURCE` is only for an absent destination REF and absent
candidate. It resolves an exact revision parent, inherits no material, records
`expected_ref_head = null`, and fails publication if the destination appears.
An existing REF update records observed current HEAD as both
`parent_revision` and `expected_ref_head`; alternate-parent override is rejected
by the current format-4 CLI.

### `sealgraph derive`

Creates a same-material child candidate without publishing it:

```sh
sealgraph derive preserved/api --from @<source-seal-prefix>
```

Destination REF and candidate must be absent. `derive` copies exactly content,
attachments/metadata, direct Cause Links/messages, root, and draft. It sets the
source Seal as `parent_revision` and copies no source parent, REF names, tags,
stale/cache, event, or candidate metadata. `expected_ref_head` is null.

Source resolution and complete parent validation finish before one candidate
file is written. Missing, corrupt, ambiguous, scope-mismatched, and destination
conflicts leave no partial candidate.

### `sealgraph link`

Changes dependencies without replacing content:

```sh
sealgraph link DESIGN-001 --depend-on REQ-001
sealgraph link DESIGN-001 --depend-on @<seal-prefix> \
  -m 'Requirement basis used by this design'
```

Format 4 has one domain-independent Cause edge and no link kind. `-m MESSAGE`
is optional exact-edge rationale and participates in candidate/Seal identity.
It is not actor, time, approval, or Seal-operation metadata. Duplicate exact
target SealIDs are errors.

### `sealgraph unlink`

Removes exactly one resolved target from one candidate:

```sh
sealgraph unlink DESIGN --upstream REQ
sealgraph unlink DESIGN --upstream @<old-target-seal-prefix>
```

A bare REF resolves its current HEAD and therefore does not match an older
stored target after the REF advances. Candidate inspection prints the exact
selector required. Missing or ambiguous target edges are errors. Unlink never
changes content, attachments, root/draft, another Link, REF HEAD, or a Seal.

### Planned `sealgraph attach` / `sealgraph detach`

These mutation commands are not exposed by the current standalone beta. Their
planned contract changes named immutable attachment material in one candidate;
attachment names are unique, and attachments remain semantically distinct from
Cause Links. Attachment-bearing repositories are still readable, inspectable,
and loadable.

### `sealgraph candidate`

```sh
sealgraph candidate show REF [--raw-content]
sealgraph candidate diff REF
sealgraph candidate discard REF
```

`candidate show` validates material, exact Cause targets, `parent_revision`,
and publication expectation. It displays `PARENT_REVISION`,
`EXPECTED_REF_HEAD`, current destination HEAD, and their relations separately.

`candidate diff` compares content, attachments, exact Links/messages,
root/draft, and `parent_revision` with the immutable parent when present.
Current HEAD versus `expected_ref_head` is separate publication state.

`candidate discard` removes only the exact validated candidate file under the
writer guard. It is itself the explicit confirmation and has no prompt,
`--yes`, or `--force`. It does not move a REF, delete an object, recurse, repair,
or report a missing/unsafe target as success. Corrupt candidates remain
explicitly discardable.

### `sealgraph seal REF`

Creates exactly one immutable Seal and attempts to advance exactly one REF.
There is no batch form.

Seal has no message, actor, or timestamp option. Such claims are ordinary
separately sealed content with explicit Cause Links when required.

A root has no Cause Links. Every non-root, including draft, has at least one.
A normal non-draft publication requires every direct and reachable Cause target
to be a non-draft active revision leaf in one coherent current-head
observation. Draft may preserve active, historical, detached, draft, or
non-draft exact Causes. Parent admissibility is separate; parent never replaces
the Cause requirement. There is no generic validation bypass.

All native mutations hold one repository-wide writer guard. Successful
expected-old REF CAS is publication. Candidate cleanup removes only the exact
version sealed; a newer candidate is retained and reported. Dangling immutable
objects after failed CAS are reported, not deleted.

### `sealgraph tag`

```sh
sealgraph tag REF
sealgraph tag REF TAGNAME
sealgraph tag REF@SEAL_OR_TAG TAGNAME
```

The one-argument form lists the REF's tags in bytewise TAGNAME order as:

```text
TAG REF "TAGNAME" FULL_SEAL_ID
```

An empty namespace emits zero bytes and succeeds. Names use quoted escaped
presentation; the manifest retains the exact raw UTF-8 TAGNAME.

The two-argument forms create one immutable binding. Bare REF tags its current
HEAD. A hexadecimal scoped token must select the current HEAD or one of its
`parent_revision` ancestors; an existing tag may select its exact historical
target. An unscoped `@SEAL_TOKEN` is rejected because it provides no REF UI
scope. Repeating the same binding is idempotent. Retarget, delete, force, and
automatic tag creation do not exist. Success is:

```text
TAGGED REF "TAGNAME" FULL_SEAL_ID
```

### `sealgraph mv`

```sh
sealgraph mv OLD_REF NEW_REF
```

Moves exactly one REF manifest, including its HEAD and complete tag namespace,
to an absent destination with one atomic no-replace rename. Both names must be
valid and different. Exact candidate state at either name blocks the command;
the operator seals or discards it explicitly. `mv` never recursively moves a
prefix REF, rewrites a candidate, creates an old-name alias, modifies a Seal or
Link, or infers hierarchy from slash spelling. Success is:

```text
MOVED OLD_REF NEW_REF FULL_HEAD_ID tags=N
```

## 3. Immutable inspection

```sh
sealgraph show SELECTOR [--raw-content]
sealgraph diff SELECTOR [SELECTOR]
sealgraph log REF
sealgraph linklog REF
sealgraph graph
sealgraph fsck
```

`show` displays exact SealID, parent revision, content, attachments, Links and
messages, root, and draft. REF names are display annotations resolved from the
current observation, never immutable owner fields.

`log REF` resolves current HEAD then follows exact `parent_revision` IDs newest
first. Parent cycles or unreadable/noncanonical Seals fail. It does not compare
embedded names or act as a Git reflog.

`diff REF` compares current HEAD with its parent and fails for an initial Seal.
Two explicit selectors compare any two canonical Seals. A mode claiming one
revision line must validate parent ancestry rather than REF ownership.

`linklog REF` compares exact target SealID sets between adjacent revisions. It
reports add, remove, ancestry-based repoint, and Link-message change. Ambiguous
N:M matching stays explicit add/remove.

Default human output never writes arbitrary content/metadata bytes directly.
Content preview is at most 256 input bytes with bytewise ASCII escaping.
Printable ASCII is literal except quote/backslash; LF/CR/TAB use
`\n`/`\r`/`\t`, and other bytes use lower-case `\xhh`.

`--raw-content` makes stdout exact content bytes only, with no metadata or
added LF. The complete object validates before output.

Read commands do not create runtime directories, append logs, repair, relink,
reseal, move a REF, or persist their derived output.

## 4. Status and stale

```sh
sealgraph status [REF]
sealgraph stale [--frontier] [--refs-only] [--scan]
```

Status can report orthogonal facts:

- `CLEAN`
- `UNSEALED`
- `DRAFT`
- `STALE_SELF`
- `STALE_DIRECT`
- `STALE_TRANSITIVE`

`STALE_SELF` means current HEAD is an active non-leaf revision.
`STALE_DIRECT` means an exact direct Cause target is not an active current leaf,
whether it is an active non-leaf or historical/detached.
`STALE_TRANSITIVE` means a deeper Link-only Cause target is not an active
current leaf while all direct targets are active current leaves. Parent edges
determine leafness but are not traversed as Cause edges.

`stale` selects current REFs having at least one stale fact and never reads
candidate state. `--frontier` keeps a stale REF only when no other stale current
head Seal appears strictly earlier in its exact Link-only Cause closure.
Unselected descendant tips, parent edges, and candidates do not affect
frontier membership.

`--refs-only` emits exactly one valid REF plus LF per selected current path in
bytewise order, no header/ID/label/quoting/`CLEAN`, and zero bytes for an empty
set. Empty and non-empty success exit 0.

`--scan` bypasses cache reads, performs canonical current-head-rooted scan, and
refreshes disposable cache when possible without changing stdout. Cache write
failure warns; canonical corruption fails. Cache is never repair truth.

Every multi-REF mode captures all current REF heads, derives and buffers, then
revalidates the complete set. Change or unreadability fails nonzero with empty
stdout. Success is an observation, not a reservation.

## 5. Impact

```sh
sealgraph impact [--all-paths] [--max-paths N] SELECTOR
```

Every selector resolves to exact source Seal `h`. Impact reports distinct
current downstream Seals whose Cause paths first reach `h` or one of its
`parent_revision` ancestors. `h` itself is excluded as a downstream result.
Multiple current REF aliases at one downstream Seal share one computation and
are displayed sorted.

Default output includes one deterministic shortest Link-edge path per impacted
Seal. Distance is Link count; equal lengths choose the bytewise lexical full
SealID sequence.

`--all-paths` emits distinct simple paths ordered by edge count then SealID
sequence. `--max-paths N` is valid only with `--all-paths`, requires a positive
integer, defaults to 100, and applies per downstream Seal. Another path emits an
explicit truncation marker and still exits 0.

Path limits never reduce impact membership, hide an impacted Seal, skip full
reachable graph validation, or weaken current-head snapshot revalidation.

## 6. Historical Cause selection

Historical, detached, and sibling exact Cause selection is first-class, not
corruption:

```sh
sealgraph link DESIGN --depend-on @<older-seal-prefix>
```

A draft candidate may preserve it. Normal publication enforces active-leaf
Cause closure. No `--force` suppresses provenance validation.

An older result can be preserved by deriving a same-material sibling child,
explicitly relinking one downstream candidate to it, and sealing that one REF.
The old downstream Seal remains immutable and stale; no pin flag, automatic
target choice, or recursive repair is introduced.

## 7. Format-3 logical dump and format-4 load

The final format-3 binary at commit `5b24d47` exposes one migration export:

```sh
sealgraph dump --format logical-v1
```

`--format logical-v1` is required exactly once. The command accepts no
positional operand, output path, repair, ignore, or compatibility option. On
success stdout is exactly one compact canonical
`sealgraph/logical-dump/v1` JSON document followed by LF, and stderr is empty.

All object, Seal, REF, tag, candidate, graph, and final-observation checks
complete before stdout emission. A candidate of any validity blocks the dump.
The command does not acquire the mutation guard, create runtime files, repair
state, or inspect Git. Output-sink failure is nonzero; consumers accept a dump
only with exit zero and complete canonical parsing.

The checked-in format-4 runtime consumes that document with:

```sh
sealgraph load --format logical-v1 < repository.dump.json
```

The target `.sealgraph` must be absent. Load stages and validates a complete
format-4 repository, rewrites all
parent/Link/REF/tag targets through a complete old-to-new mapping, and uses one
atomic no-replace directory publication. It never merges with or replaces an
existing repository. Non-empty tags are rewritten to full format-4 SealIDs and
published inside their REF manifests; none are dropped or privately deferred.

A platform without atomic no-replace directory publication rejects load before
target publication; it does not weaken the boundary to check-then-rename.

Successful stdout is the canonical compact
`sealgraph/logical-load-receipt/v1` document plus LF. It includes the exact
source dump SHA-256, every old-to-new mapping, every many-to-one collapse,
rewritten REF and tag records, and published format `4`. A prior
`.sealgraph-load-*` staging path is reported for explicit inspection and is
never adopted or deleted automatically.

## 8. Git plugin

Git integration is invoked explicitly as:

```sh
git sealgraph <command>
```

and implemented by `git-sealgraph`. It uses the same native `.sealgraph`
format. Initial capability contracts, with final names separately gated, are:

- prospective staged-tree validation;
- immutable commit-tree validation/inspection;
- merge conflict evidence from stages 1/2/3 plus validated
  BASE/OURS/THEIRS complete trees;
- opt-in validation-only hook dispatch.

A hook command validates what Git will commit, including unchanged base paths,
not a different worktree. It never self-installs, overwrites an existing hook,
stages, seals, advances a REF, relinks, repairs, commits, pushes, or treats
success as approval.

Merge assistance may show exact BASE/OURS/THEIRS target SealIDs, revision
ancestry, and Cause differences. Different bytes at one immutable native object
path are corruption. Divergent REF/tag targets remain explicit. Selecting a
resolution does not create a Seal or imply review.

Git blob/tree/commit/tag material import is not in the initial sidecar.

## 9. Exit and machine output

- `0`: successful command, including empty factual result or documented path
  truncation;
- `1`: reserved domain/status condition where explicitly documented;
- `2`: CLI usage error before repository traversal;
- other nonzero: operational, integrity, snapshot, or unsupported-format
  failure.

Structured machine-readable outputs must be versioned before external use.
`stale --refs-only` is the intentionally narrow stable line exception.

`fsck [--format human|json]` performs a complete read-only inventory of loose
objects, REF manifests/tags, canonical Seals, material references, and both
parent and Cause DAGs. Success JSON uses `sealgraph/fsck/v1`. Historical or
detached Seals and unreferenced valid blobs are reported separately and do not
fail the command. Corruption, missing references, unsafe paths, or cycles fail
nonzero; `fsck` never repairs, removes, repacks, caches, or changes modes.

The standalone beta has no file synchronization/watch/import surface.
`manifest` is an explicit path/digest claim builder only. Attachment fields are
read, preserved by load, and inspected, but beta does not expose `attach` or
`detach` mutation commands.

`show`, `status`, `stale`, `graph`, `impact`, `log`, `linklog`, and `diff`
accept `--format human|json` in any argument position. Human is the default;
JSON uses a command-specific `sealgraph/<command>/v1` schema. Raw content and
the REF-only line protocol cannot be combined with JSON. JSON contains full
ObjectID strings and arrays of ObjectIDs for paths, not presentation strings.

Human output uses `SEALED_STATE`, `STRUCTURAL_IMPACT`, and
`REVISION_CAUSE_GRAPH` headings. `CLEAN` describes candidate/stale state only;
it does not compare working files. A REF is a movable logical identity, impact
is structural rather than stale-only, root is a provenance boundary rather
than trust, and Seal/link history is not Git commit/reflog history. Standalone
commands do not discover or inspect Git.
