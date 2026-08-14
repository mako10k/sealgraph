# ADR 0008: Explicit candidate lifecycle and binary-safe human output

Status: accepted on 2026-08-14.

## Context

Candidate state is mutable, unsealed, and non-authoritative, while the existing
`REF@TOKEN` selector identifies immutable seal generations. Mixing candidate
selection into that grammar would blur the boundary and can collide with
REF-scoped tags.

Operators also need to inspect, compare, unlink, and explicitly discard a
candidate before sealing. The existing `show` command writes arbitrary content
bytes directly between metadata lines, which is unsafe for binary data,
terminal control bytes, and metadata-like content.

## Decision

### Candidate surface

Candidate operations use a dedicated mutable-state namespace:

```text
sealgraph candidate show REF [--raw-content]
sealgraph candidate diff REF
sealgraph candidate discard REF
```

Candidate diff compares with the candidate's recorded base. Current HEAD and a
derived base relation are displayed separately. No operation automatically
rebases, relinks, repairs, seals, or changes root/draft state.

`candidate discard REF` is itself the explicit destructive confirmation. It
removes only one exact regular candidate file under the repository writer
guard. It has no prompt, `--yes`, `--force`, recursive deletion, object
deletion, or REF movement. Missing and unsafe targets are errors. It remains
available for corrupt candidate recovery.

### Unlink

One dependency edge is removed with:

```text
sealgraph unlink REF --upstream UPSTREAM[@TOKEN]
```

A bare upstream names the unique edge keyed by its target REF. A qualified
upstream resolves TOKEN and removes the edge only when the candidate currently
stores that exact generation. Missing edges and generation mismatches are
errors. Unlink never changes content, root, draft, or another edge.

### Human and raw output

Default sealed and candidate show output contains content identity, byte size,
and a maximum 256-input-byte bytewise ASCII-escaped preview. Arbitrary string
metadata uses the same safe quoted-byte representation.

`--raw-content` changes stdout to exact content bytes only, with no metadata or
added newline. The full object is validated before output. Versioned machine
output remains a separate cross-command contract owned by SG-BL-006.

## Consequences

- Mutable candidate state does not enter immutable selector or tag grammar.
- Candidate intent remains anchored to its recorded base even after HEAD moves.
- Corrupt or stale-base candidates have an explicit, non-recursive recovery
  operation.
- Human show output is bounded and cannot inject raw terminal controls.
- Exact bytes remain available without mixing data and metadata.
- No persisted field or canonical seal encoding changes.
