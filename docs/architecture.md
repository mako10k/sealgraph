# Architecture

## 1. Design center

The main implementation goal is to share read semantics between:

1. standalone `.sealgraph` object storage, and
2. Git-backed content/history used by `git sealgraph`.

The shared abstraction is not “Git repository”. It is an immutable object reader plus sealgraph-specific refs/domain logic.

## 2. Process surfaces

```text
                 +----------------------+
                 |    lib sealgraph     |
                 | domain / graph / CLI |
                 +----------+-----------+
                            |
             +--------------+--------------+
             |                             |
      cmd/sealgraph                 cmd/git-sealgraph
       standalone                      Git plugin
             |                             |
   NativeObjectReader              GitObjectReader
      .sealgraph                       .git
             \                             /
              +------ ObjectReader -------+
                            |
                     shared decoding
```

`git-sealgraph` is intentionally a separate executable. Git discovers executables named `git-<name>` on PATH and exposes them as `git <name>`.

## 3. Package boundaries

### `internal/domain`

Pure semantic value types:

- ObjectID
- ContentRef
- Link
- Attachment
- SealPayload
- REF-related identities/state

No filesystem, Git, CLI, or environment access.

### `internal/store`

Storage interfaces:

- ObjectReader
- ObjectWriter
- RefStore
- TagStore

Native and Git-backed implementations belong below this boundary.

### `internal/graph`

Derived graph behavior:

- direct stale
- transitive stale
- reverse impact
- cycle detection
- provenance traversal

Never persist derived stale state here.

### `internal/history`

Read-only inspection derived from immutable seal payloads:

- parent-chain traversal,
- link add/remove/repoint events between adjacent generations,
- semantic differences between two generations of one logical REF.

History traversal validates canonical seal loading, REF ownership, and parent
cycles before returning results. It does not read Git history or maintain a
reflog, and it does not persist derived events or differences.

### `internal/repository`

Coordinates:

- candidate state,
- seal creation,
- ref updates,
- object store selection,
- validation.

It MUST receive repository mode explicitly. It MUST NOT infer sidecar mode by probing for `.git`.

### `internal/cli`

Parsing/presentation only. Domain decisions remain in repository/graph packages.

## 4. Native object store

Standalone uses `.sealgraph/objects`.

Native v2 characteristics:

- immutable loose objects,
- Git-compatible object envelope/layout where practical,
- SHA-256 native object identity,
- full-hex canonical IDs with user-input unique-prefix resolution,
- immutable REF-scoped lightweight tags,
- no canonical packfiles,
- no packed refs.

The exact byte contract belongs in `storage-format.md` and must be locked by fixture tests before production use.

## 5. Git content reader

Git sidecar should use a mature Git SDK rather than hand-implementing packfiles, deltas, alternates, worktrees, or repository object-format details.

Initial candidate: `go-git` stable v5 series.

Important boundary:

```text
Git SDK = physical Git reading
Sealgraph = provenance semantics
```

Do not delegate sealgraph REF/seal semantics to Git branches/commits merely to increase high-level Git CLI compatibility.

## 6. Repository modes

Modes are explicit configuration selected by separate entry points.

### standalone

- command: `sealgraph init`
- canonical content: `.sealgraph`
- seals: `.sealgraph`
- refs: `.sealgraph`
- Git access: none

### git-sidecar

- command: `git sealgraph init`
- Git content/history source: `.git`
- sealgraph seal metadata/refs: `.sealgraph`
- merge inspection: Git-aware
- implicit Git-to-seal approval: forbidden

The modes are mutually explicit; standalone does not discover or recommend sidecar.

## 7. Transaction boundary

Creating a seal conceptually requires:

1. load one REF candidate,
2. validate candidate and dependency admissibility,
3. canonicalize payload,
4. write immutable objects,
5. atomically update exactly one REF HEAD,
6. clear/update that REF's candidate state.

The ref update must support compare-and-swap semantics to detect concurrent changes.

## 8. Future extension points

Keep interfaces narrow enough to permit:

- alternate content stores,
- remote CAS readers,
- signatures/attestations,
- richer relation types,
- machine-readable command output.

Do not implement these until requirements exist.
