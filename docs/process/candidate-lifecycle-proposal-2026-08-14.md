# Candidate lifecycle and safe-output proposal

Status: accepted by the operator on 2026-08-14; recorded by ADR 0008.

This proposal closes the two remaining items in the external-spec consistency
gate without importing Git index/reset semantics or adding persisted fields.
The structured analysis is in
[`../decisions/2026-08-14-candidate-lifecycle.think`](../decisions/2026-08-14-candidate-lifecycle.think).

## 1. Proposed CLI surface

```text
sealgraph candidate show REF [--raw-content]
sealgraph candidate diff REF
sealgraph candidate discard REF

sealgraph unlink REF --upstream UPSTREAM[@SEAL_OR_TAG]

sealgraph show REF[@SEAL_OR_TAG] [--raw-content]
```

`candidate` is a dedicated mutable-state namespace. It does not extend the
immutable `REF@TOKEN` selector grammar and does not introduce an index, stage,
checkout, reset, or worktree concept.

The alternatives `show --candidate`, `diff --candidate`, and
`REF@candidate` are not recommended. They mix mutable state into commands and
selectors whose current operands are immutable generations, make show/diff
parsing asymmetric, and risk collision with REF-scoped tags.

## 2. Candidate inspection

`candidate show REF` reads exactly one candidate and validates its candidate
schema, content object, direct dependency seal ownership, base seal ownership,
and current REF head ownership before printing. It does not mutate the
repository or claim that the candidate is admissible for a normal seal.

Human output contains at least:

```text
REF <ref>
CANDIDATE
BASE <full-seal-id-or->
CURRENT_HEAD <full-seal-id-or->
BASE_STATE INITIAL|CURRENT|HEAD_ADVANCED|HEAD_MISSING|UNEXPECTED_HEAD
CONTENT native/blob@<full-object-id> bytes=<decimal-size>
CONTENT_PREVIEW "<escaped-bytes>" truncated=<true-or-false>
ROOT <true-or-false>
DRAFT <true-or-false>
ATTACHMENTS <count>
DEPENDENCIES <count>
  depend-on <ref>@<full-seal-id> message="<escaped-bytes>"
```

`BASE_STATE` is derived and not persisted:

- `INITIAL`: candidate base and current HEAD are both absent;
- `CURRENT`: candidate base equals current HEAD;
- `HEAD_ADVANCED`: both exist and differ;
- `HEAD_MISSING`: candidate has a base but current HEAD is absent;
- `UNEXPECTED_HEAD`: candidate has no base but a current HEAD exists.

The command reports state rather than automatically rebasing, relinking,
discarding, or repairing anything. A corrupt candidate fails inspection and
names `candidate discard REF` as the explicit recovery operation.

## 3. Candidate diff

`candidate diff REF` compares the candidate with its recorded `base`, not with
whatever HEAD happens to be current at inspection time. The base is the state
from which the candidate was derived, so this comparison represents operator
intent even when the REF advanced later.

The output also includes `CURRENT_HEAD` and `BASE_STATE`. A base mismatch is a
visible state result, not an instruction to update the candidate. Sealing
continues to reject the mismatch.

The semantic comparison covers fields that exist in candidate state:

- content identity;
- attachments;
- direct links, including target repoints and message changes;
- root;
- draft.

The next parent is not invented. ADR 0009 subsequently removed seal-level event
message and `created_at` from the canonical schema. For an initial candidate
with no base, the diff reports the
candidate fields as additions from an absent state.

## 4. Unlink

`unlink` removes exactly one dependency edge from exactly one candidate:

```sh
sealgraph unlink DESIGN --upstream REQ
sealgraph unlink DESIGN --upstream REQ@reviewed-v1
```

The edge key is the upstream logical REF because native v3 allows only one
dependency per target REF.

- A bare `--upstream REQ` removes the candidate's current edge for `REQ`,
  regardless of its concrete generation.
- `--upstream REQ@TOKEN` first resolves TOKEN to a full seal ID and removes the
  edge only if the candidate currently targets that exact generation.
- An absent edge or generation mismatch is an error, not idempotent success.
- One invocation accepts exactly one `--upstream`.

Removing the last dependency from a non-root candidate is allowed as an
intermediate mutable state, but the candidate remains unsealable until the
operator explicitly adds another dependency or changes the next generation to
root. `unlink` never changes root/draft/content and never reseals.

`--depend-on` is not recommended for removal: bare `--depend-on REF` already
means “resolve current HEAD” on add/link, which is the wrong default when the
candidate intentionally contains a historical generation.

## 5. Discard

`candidate discard REF` removes exactly the candidate file for one validated
REF under the repository-wide writer guard. It does not move the REF, delete
immutable objects, recursively remove hierarchical candidates, or repair
anything.

The initial proposal treats the explicit three-word command plus exact REF as
the confirmation. It has no interactive prompt, `--yes`, `--force`, or public
candidate fingerprint. Missing candidates, directories, symlinks, and prefix
conflicts are errors rather than successful no-ops.

This keeps deterministic automation and avoids introducing a seal-like
identity for mutable state. If real concurrent operator workflows later need
inspect-then-delete CAS across separate commands, an optional exact candidate
version token can be designed from evidence rather than added speculatively.

Discard must also work when the candidate JSON is corrupt, because explicit
discard is the recovery path. The implementation resolves the exact safe path
from a validated REF, requires one regular non-symlink file, and performs no
recursive deletion.

## 6. Binary-safe human output

Default `show` and `candidate show` never write arbitrary content bytes
directly. They print content identity, exact decimal byte size, and at most the
first 256 input bytes as a bytewise ASCII-escaped preview.

Preview encoding is deterministic:

- printable ASCII bytes other than `"` and `\\` are literal;
- `"` and `\\` become `\\"` and `\\\\`;
- LF, CR, and TAB become `\\n`, `\\r`, and `\\t`;
- every other byte becomes lower-case `\\xhh`;
- the encoded value is enclosed in ASCII double quotes;
- `truncated=true` iff content contains more than 256 bytes.

Seal messages, link messages, attachment names, and media types use the same
safe quoted-byte representation in human output. This prevents control bytes,
Unicode display controls, or metadata-like lines from changing terminal
presentation while retaining an unambiguous representation of the original
UTF-8 bytes.

`--raw-content` is an explicit bytes-only mode. On success stdout contains the
exact content bytes and nothing else: no metadata, preview, prefix, or trailing
newline. Diagnostics remain on stderr. The complete object is read and
validated before stdout is written.

An alternative top-level `content` command is not recommended for this slice:
it would need a second selector rule to distinguish current seal content from
candidate content. Keeping raw extraction on the corresponding show surface is
more symmetric.

## 7. Machine output boundary

This proposal does not add `--format json`. SG-BL-006 remains responsible for
one versioned schema shared by show/status/stale/graph/impact/log/linklog/diff.
If machine output later includes content, binary bytes require an explicit
encoding and must never reuse the human preview as authoritative data.

## 8. Implementation slice after acceptance

1. Add repository-level candidate inspection/diff/unlink/discard operations.
2. Keep all mutations behind the ADR 0007 writer guard.
3. Add a shared CLI byte-quoting/preview presenter; do not put presentation in
   domain or repository packages.
4. Change sealed `show` default output and add bytes-only raw extraction.
5. Test corrupt candidate discard, guarded historical unlink, base/head
   mismatch, control/binary bytes, preview bounds, exact raw output, read-only
   inspection, and no recursive deletion.
6. Update normative CLI/requirements documentation and mark the external-spec
   consistency gate complete only after validation and dogfood.

No persisted field or canonical seal byte changes are proposed.
