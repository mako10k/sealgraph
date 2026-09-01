# CLI contract

Status: the checked-in standalone CLI creates and opens repository format 5.
Accepted ADRs 0023, 0025, 0026, and 0027 define the incompatible Cause-scoped
revision, typed-Blob, migration, and machine-output boundary.

## 0. Discovery, diagnostics, and output

Help is repository-independent:

```sh
sealgraph help
sealgraph help COMMAND
sealgraph COMMAND --help
sealgraph help candidate show
```

Help and completion MUST NOT bootstrap `.sealgraph`, open bound source files,
inspect Git, or update cache state. Usage errors exit 2. Repository,
integrity, concurrency, and I/O failures exit 1. Success exits 0.

Diagnostics distinguish `error`, `reason`, explicit next-action `hint`, and a
help route. Failure never silently repairs, relinks, reseals, selects a REF, or
retries an ambiguous publication.

Inspection defaults to aligned abbreviated human output on a terminal and the
command's versioned JSON document on a known pipe/file. `--format human|json`
overrides detection. Machine documents use full IDs. `--raw-content` and
`stale --refs-only` are separate exact protocols and do not mix metadata.

## 1. Selector grammar

| Form | Meaning |
| --- | --- |
| `REF` | exact current HEAD of REF |
| `@SEAL_TOKEN` | repository-wide unique 4-64 lower-hex native object prefix that decodes as a format-5 Seal |
| `REF@hex` | Seal in the current HEAD's observed revision closure |
| `REF@TAGNAME` | immutable exact tag target in that REF's manifest |

There is no `@latest`. Bare lower-hex may be a REF and is not an object
selector. Authoring persists only resolved full SealIDs, never selector
spelling or dynamic HEAD references.

All selectors used by one graph-dependent operation resolve against one
coherent REF-manifest observation. Prefix and graph inventories are validated
before output or Candidate replacement, then revalidated.

## 2. Initialization and migration

### `sealgraph init`

```sh
sealgraph init
```

An absent target creates exactly:

```text
repository_format = 5
object_format = sha256
ref_format = manifest-v1
```

It creates canonical `objects` and `refs/seals` plus empty runtime `index` and
`locks`. It never detects or inspects Git. A complete format-5 repository is
idempotent; missing safe runtime directories may be bootstrapped explicitly.

An exact format-4 config fails before mutation with
`FORMAT4_REQUIRES_MIGRATION` and this sequence:

```sh
# Run with the format-5 binary in the retained format-4 source repository.
sealgraph migrate extract --source-format 4 --format universal-blob-v1 > repository.dump.json

# Run with the format-5 CLI in a directory where .sealgraph is absent.
sealgraph load --format universal-blob-v1 < repository.dump.json
```

### `sealgraph migrate extract`

`migrate extract` is the only format-5 command that opens format 4. It is
read-only, has no source mutation operation, and is dispatched outside the
ordinary format-5 repository runtime. There is no general dual reader,
in-place migration, automatic source cleanup, or legacy-parent fallback.

Both `--source-format 4` and `--format universal-blob-v1` are required exactly
once. No positional source path, output path, repair, ignore, compatibility, or
Git option is accepted. The source is exactly the current directory's
`.sealgraph`.

The extractor validates the exact format-4 config, complete physical loose
object inventory, every REF manifest and scoped tag, the complete rooted
parent/Cause closure, referenced content and attachment Blobs, and absence of
Candidate state. It constructs output only from the first complete source
capture and requires an equal second capture immediately before delivery.
Successful stdout is exactly one canonical
`sealgraph/universal-blob-migration/v1` document plus LF. Semantic-change
warnings are emitted only after successful document delivery.

### `sealgraph load`

```sh
sealgraph load --format universal-blob-v1 < repository.dump.json
```

The format option is required exactly once and no positional argument is
accepted. Load verifies exact canonical document bytes, embedded format-4
payloads and IDs, extractor semantic classifications, and the complete
projected Cause/revision graph before target creation.

Load then stages a format-5 repository and
`sealgraph/universal-blob-load-receipt/v1`, validates it with fsck and the
repository digest, synchronizes nested state, atomically publishes only to an
absent target, synchronizes the parent directory, and reads back repository and
receipt. It never merges, replaces, repairs, or opens the source repository.

Successful stdout is the exact canonical receipt. Nonzero semantic-loss counts
are also warned on stderr:

```text
SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=N
SEMANTIC_CHANGE_COLLAPSED_REVISION_DROPPED count=N
SEMANTIC_CHANGE_MERGED_CAUSE_LINKS count=N
```

Pre-publication failure, published durability uncertainty, published readback
failure, and published receipt-undelivered failure are distinct. A
post-publication failure MUST NOT be answered by retrying load or deleting the
target automatically.

### `sealgraph load-receipt`

```sh
sealgraph load-receipt --source-document-sha256 HEX
```

This read-only recovery command is the only retryable path after
`LOAD_PUBLISHED_RECEIPT_UNDELIVERED`. It requires the one matching regular
receipt file, validates its exact canonical bytes, validates the complete
format-5 repository, and requires the current repository digest to equal the
receipt before emitting those exact bytes and their final LF. Missing,
mismatched, noncanonical, or stale receipts fail without partial stdout. It
does not create, repair, migrate, or republish repository state.

## 3. Candidate authoring

Format 5 uses one-target whole-record Cause operations:

```text
CAUSE_GROUP := --target TARGET
               (--previous PREVIOUS [--previous PREVIOUS ...] | --no-previous)
               [-m MESSAGE ...]
ROOT_MODE   := --root --clear-cause-links | --non-root
```

`--target` occurs exactly once when present. `--previous` and `-m` are
repeatable. `--previous` and `--no-previous` are mutually exclusive. There is
no positional association, multi-target invocation, implicit parent assertion,
`--depend-on` alias, `derive`, or `add --parent`.

### `sealgraph add`

```sh
sealgraph add REF \
  [--content CONTENT | --content-file PATH|-] [--bind-source] [--draft] \
  [--root --clear-cause-links | --non-root] \
  [--target TARGET (--previous PREVIOUS ... | --no-previous) [-m MESSAGE ...]]
```

`--content` and `--content-file` conflict. A named content file is one exact
regular non-symlink file; `-` is exact stdin. `--bind-source` applies only to a
named valid path and publishes the local binding after Candidate publication.

A new root requires `--root --clear-cause-links` and forbids a target group. A
new non-root requires `--non-root` and one complete group. An existing edit
preserves omitted root mode and omitted Cause group. `--root
--clear-cause-links` atomically sets root and clears all Links. `--non-root`
must leave at least one Link.

Without explicit content, add reads the exact local source binding, or uses
REF-as-path only for an entirely new REF/Candidate. It preserves omitted
semantic state. Add writes content and Candidate only; it never seals.

Examples:

```sh
sealgraph add premise --root --clear-cause-links --content 'External premise'

sealgraph add design/api --content-file design.md --non-root \
  --target requirements/api --no-previous

sealgraph add design/api --target requirements/api \
  --previous @0123 -m 'reviewed revision relationship'
```

### `sealgraph link` and `unlink`

```sh
sealgraph link REF --target TARGET \
  (--previous PREVIOUS ... | --no-previous) [-m MESSAGE ...]

sealgraph unlink REF --target TARGET
```

`link` creates or replaces the complete record for the resolved target and
preserves all other Candidate fields and Cause Links. It never unions an old
message or previous array. `unlink` removes exactly that target record. A bare
target REF resolves its current HEAD and therefore does not match a stored
historical target after the REF advances.

Retargeting is explicit and normally takes two reviewed mutations: add/replace
the new complete target record, then unlink the old exact target. Neither step
may leave an invalid root-with-Cause or non-root-without-Cause Candidate.

### `sealgraph candidate`

```sh
sealgraph candidate show REF [--raw-content] [--format human|json]
sealgraph candidate compare REF [--format human|json]
sealgraph candidate discard REF
```

Show reports the parentless Candidate, prospective Material/Provenance/Seal
IDs, current REF head, and `EXPECTED_ABSENT`, `EXPECTED_CURRENT`,
`HEAD_ADVANCED`, `HEAD_MISSING`, or `UNEXPECTED_HEAD` state.

Compare uses only `candidate.expected_ref_head` as the immutable publication
baseline. It does not infer a revision predecessor. Discard removes exactly
one Candidate, including a corrupt Candidate through the explicit bounded
discard path; it removes no REF, Blob, source binding, or descendant state.

### `sealgraph seal REF`

Seal publishes at most one new Seal for exactly one REF. Under the repository
writer guard it reloads the exact Candidate version, validates content,
attachments, typed prospective IDs, combined graph, current observation, CAS
expectation, and normal Cause closure. A normal non-draft Candidate requires
every reachable Cause to be a non-draft active revision leaf. Draft preserves
intentional historical/provisional provenance visibly.

Publication writes immutable Material, Provenance, and Seal Blobs, revalidates
the observation, CAS-updates one REF, and removes only the unchanged Candidate
version. There is no batch, force, automatic relink, or automatic stale repair.

## 4. Local source and explicit manifests

```sh
sealgraph source bind REF --file PATH
sealgraph source rebind REF --from OLD_PATH --file NEW_PATH
sealgraph source unbind REF --from PATH
sealgraph source show REF
sealgraph source list
sealgraph source compare REF

sealgraph manifest --source SOURCE --file PATH [--file PATH ...]
```

Source binding is non-canonical local input configuration. Bind is
create-only/idempotent for the same path; rebind and unbind require exact
observed old paths. Inspection does not open source files. Compare opens only
the selected safe file and compares it with Candidate content, otherwise HEAD
content. None of these commands seals or reads Git.

Manifest reads only explicitly named portable relative regular files, sorts
them, and emits deterministic `sealgraph/path-manifest/v1` content. It does not
open `.sealgraph`, infer Git identity, recurse, expand globs, add, or seal.

## 5. REF and tag mutation

```sh
sealgraph tag REF
sealgraph tag REF TAGNAME
sealgraph tag REF@SEAL_OR_TAG TAGNAME
sealgraph mv OLD_REF NEW_REF
sealgraph ref drop REF
```

Tags are immutable REF-scoped aliases stored in the REF manifest. Same binding
is idempotent; retarget, delete, force, and unscoped creation are absent.

`mv` moves one complete manifest, including tags, to an absent destination by
no-replace rename. Candidates or source bindings at either name block it. It
does not move a workfile or Candidate, retain an alias, or rewrite a Seal.

`ref drop` removes one manifest only after Candidate and binding absence and
creates a local recovery record. It never deletes immutable Blobs or workfiles.

## 6. Immutable and graph inspection

```sh
sealgraph show SELECTOR [--raw-content] [--format human|json]
sealgraph compare FROM_SELECTOR TO_SELECTOR [--format human|json]
sealgraph status [REF] [--format human|json]
sealgraph stale [--frontier] [--refs-only] [--scan] [--format human|json]
sealgraph graph [--format human|json]
sealgraph impact [--asserted-by OBSERVER_SELECTOR ...] \
  [--all-paths] [--max-paths N] SELECTOR [--format human|json]
sealgraph log [--all-paths] [--max-paths N] REF [--format human|json]
sealgraph linklog [--upstream TARGET_SELECTOR] REF [--format human|json]
sealgraph fsck [--format human|json]
```

`compare` requires exactly two explicit immutable selections and contains no
inferred revision field. Status separates Candidate/HEAD, source/baseline,
draft, self-stale, direct Cause stale, and transitive Cause stale.

The observed graph starts at every current head and reaches the fixed point
through Cause targets and their observer-scoped previous assertions. Revision
activity follows revision edges only. Graph output retains every assertion
source. `stale --frontier` is the upstream-first exact-Cause review frontier;
it is navigation, not a batch plan. `--scan` is semantically identical to a
cache bypass; the current format-5 runtime persists no graph cache.

Impact revision proof uses all observed assertions by default. Repeatable
`--asserted-by` restricts proof edges and scoped observations to exactly those
resolved observer Seals. Default output contains one first-match Cause path per
impacted head. `--all-paths` uses a positive `--max-paths` bound (default 100)
without changing membership or validation.

Log reports each structurally reachable revision once by minimum depth.
`--all-paths` reports bounded complete leaf-terminated paths. Linklog compares
complete Cause records across every structural revision edge; `--upstream`
filters only the final changed-target records and never changes edge evidence.

Fsck validates physical loose objects, typed roles, content and attachment
closure, exact manifests and tags, combined graph, active inventory, and final
observation without repair.

Format-5 JSON schemas are:

| Command | Schema |
| --- | --- |
| show | `sealgraph/show/v2` |
| candidate show | `sealgraph/candidate-show/v2` |
| candidate compare | `sealgraph/candidate-compare/v2` |
| status | `sealgraph/status/v3` |
| stale | `sealgraph/stale/v2` |
| graph | `sealgraph/graph/v2` |
| impact | `sealgraph/impact/v2` |
| log | `sealgraph/log/v2` |
| linklog | `sealgraph/linklog/v2` |
| compare | `sealgraph/compare/v2` |
| fsck | `sealgraph/fsck/v2` |

Exact member order and shared record shapes are normative in ADR 0027.

## 7. Local operational recovery

```sh
sealgraph recover show [OPERATION_ID] [--format human|json]
sealgraph recover OPERATION_ID [--format human|json]
```

Recovery requires one exact 32-lower-hex local operation ID. It restores only
the exact prior REF-manifest state when current state still equals the recorded
after-state. It never changes a typed Blob, interprets semantic intent, selects
the latest record, or performs reset/reflog/undo semantics.

## 8. Git sidecar

`git sealgraph ...` is a separate unreleased surface implemented by the
`git-sealgraph` executable. Standalone `sealgraph` never detects `.git`.
Any future sidecar uses the same format-5 `.sealgraph` bytes, offers read-only
Git views where approved, and never turns Git commit/merge success into a Seal,
automatic relink, or approval.
