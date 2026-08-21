# ADR 0020: Sealgraph-native operation vocabulary and Git-misuse navigation

Status: Accepted

Date: 2026-08-21

## Context

Sealgraph users commonly approach the CLI with Git vocabulary. That analogy is
useful for discovering an operation, but dangerous when an identical command
name suggests identical state or mutation semantics. Git's worktree, index,
commit, branch, and tracked-file deletion model does not match sealgraph's
workfile, local source binding, mutable candidate, immutable Seal, and movable
logical REF model.

ADR 0019 deliberately avoided calling a local source binding Git tracking.
The same discipline is needed across comparison, deletion, restoration, and
shell discovery. A user should be able to try a familiar Git-shaped command,
receive an error that explains the violated analogy and the available
sealgraph state transitions, then retry with the canonical sealgraph name.
Silently accepting a Git name as an alias would bypass that learning boundary.

The current beta exposes `diff` and `candidate diff`. Their output compares
semantic identity-bearing state such as content object identity, attachments,
Cause Links, root, draft, and parent revision. It is not a Git patch or a
worktree/index comparison. Pre-1.0 compatibility policy permits correcting
these names before they become stable.

REF removal also needs a distinct contract. Removing an active REF manifest
does not delete a workfile, immutable Seal, or content-addressed object, and it
must not be introduced before the accepted ADR 0018 recovery journal is
implemented for that transition.

## Decision

### Canonical comparison vocabulary

The canonical comparison commands are:

```text
sealgraph compare SELECTOR [SELECTOR]
sealgraph candidate compare REF
sealgraph source compare REF
```

`compare` compares immutable Seal material and provenance. With one selector,
it compares the selected current generation with its validated parent revision;
with two selectors, it compares the two exact resolved Seals.

`candidate compare` compares one mutable candidate with its recorded immutable
parent revision and reports its current REF publication expectation separately.

`source compare` compares one bound regular workfile with the candidate content
when a candidate exists, otherwise with current HEAD content. It identifies the
baseline as `CANDIDATE`, `HEAD`, or `NONE` and preserves ADR 0019's safe-read,
missing-source, unreadable-source, and changed-during-read boundaries.

Default comparison output remains bounded and binary-safe. A future textual
patch mode requires a separate output and disclosure decision; it is not
implied by the word `compare`.

The pre-1.0 CLI replaces `diff` with `compare` and `candidate diff` with
`candidate compare`. The old forms are not aliases. Invoking them produces the
Git-misuse/navigation diagnostic described below. `diff --draft` is never
introduced: `draft` is an identity-bearing provenance property, not a synonym
for mutable candidate state.

### REF removal vocabulary and ordering

The future canonical removal command is:

```text
sealgraph ref drop REF
```

`ref drop` removes exactly one current REF manifest from the active namespace.
It does not delete or modify a workfile, source binding, candidate, immutable
Seal, content object, Cause Link, or downstream Seal. Candidate or source
binding state at the REF blocks the operation and must be handled explicitly.
The complete tag namespace in the REF manifest leaves the active namespace
with that manifest.

There is no `rm`, `remove`, recursive, prefix, batch, cached, staged, workfile,
or force form. The name `drop` describes removal of the logical REF handle
without claiming deletion of the immutable graph or filesystem material.

`ref drop` MUST NOT be implemented until ADR 0018 recovery is implemented and
extended to journal its complete before/after manifest transition. Recovery is
selected by explicit operation ID and remains subject to exact after-state
comparison. Recovered-away or dropped immutable objects remain valid inventory.

The following operations remain intentionally distinct:

```text
sealgraph candidate discard REF
sealgraph source unbind REF --from PATH
sealgraph ref drop REF
sealgraph recover OPERATION_ID
```

No command combines these state changes or deletes the bound working file.

### Git-shaped misuse navigation

Recognized Git-shaped invocations fail before repository mutation. They are
diagnostic routes, not hidden aliases. A diagnostic follows this order:

1. identify the attempted Git operation or option;
2. explain which Git state assumption does not exist in sealgraph;
3. name the sealgraph state layers that the user may intend to change;
4. show exact canonical commands for those separate intentions;
5. point to command or concept help and require an explicit retry.

At minimum, navigation recognizes:

- `rm` and `remove`;
- `diff`, `diff --cached`, `diff --staged`, and `diff --draft`;
- `add .`, `add -A`, and `add -u`;
- `commit`;
- `checkout`, `switch`, and `branch`;
- `reset`, `restore`, and `clean`;
- file-movement expectations around `mv`.

Diagnostics MUST NOT infer the intended REF, path, baseline, operation ID, or
destructive action. They MUST NOT mutate, repair, relink, seal, drop, recover,
or open a source file merely to improve the suggestion.

Examples of required distinctions include:

```text
'rm' does not delete or stage a workfile in sealgraph
candidate discard removes only unsealed candidate state
source unbind removes only machine-local input configuration
ref drop removes only the active logical REF manifest
```

```text
'--cached' assumes Git index semantics
candidate compare compares candidate with its recorded parent revision
source compare compares a bound workfile with its candidate-or-HEAD baseline
```

```text
'draft' is an identity-bearing provenance property
candidate is the name of mutable pre-publication state
```

Unknown spellings that are not recognized Git-shaped forms continue through
the ordinary unknown-command or usage diagnostic. Display-only typo
suggestions remain non-mutating.

### Help and Bash completion

Root help, command help, concepts, use cases, errors, and completion share one
canonical command vocabulary. Git-shaped diagnostic names do not appear as
normal commands, aliases, or completion candidates.

Bash completion uses a thin script backed by a hidden read-only protocol:

```text
sealgraph __completion --bash WORD...
completions/sealgraph.bash
```

The binary, rather than the shell script, owns command, subcommand, option, and
repository-aware candidate selection. Completion may suggest canonical
commands, enum values, safe file paths, REFs, binding-bearing REFs, and
candidate-bearing REFs when appropriate for the parsed command position.

Completion MUST:

- remain read-only and never bootstrap a repository;
- never inspect Git or update a cache;
- never open a bound source file merely to complete a REF;
- fail silently from the shell's perspective when completion cannot be
  computed safely;
- preserve exact token bytes without evaluating shell text;
- avoid exposing corrupt or unsafe state as a plausible valid candidate;
- derive command and option candidates from the same registry used by help;
- have protocol and Bash-wrapper regression tests;
- be installed to the standard bash-completion directory by the documented
  install target when that integration is requested.

A custom binary override such as `SEALGRAPH_COMPLETION_BIN` may support
in-tree and user-prefix installations without changing completion semantics.

### Delivery order

Implementation proceeds in this order:

1. canonical vocabulary plus Git-shaped diagnostic navigation;
2. `compare`, `candidate compare`, and `source compare`;
3. dynamic Bash completion and its install/test surface;
4. ADR 0018 recovery journal and `recover` CLI;
5. recoverable `ref drop`.

The first three steps do not authorize `ref drop`. Recovery implementation
does not by itself authorize object deletion, workfile deletion, recursive
mutation, or arbitrary rollback.

## Consequences

- Git familiarity remains a discovery aid without becoming an accidental
  semantic contract.
- Users learn the sealgraph state model by retrying with canonical names.
- Scripts using beta `diff` forms require an intentional pre-1.0 update.
- Comparison commands become symmetric across immutable Seal, mutable
  candidate, and local source/workfile state.
- Candidate discard, source unbind, REF drop, and recovery remain separate and
  auditable operations.
- Bash completion improves discovery without advertising unsupported Git
  aliases or introducing repository mutation.
- REF removal remains blocked until an exact recoverable transition exists.
