# Architecture

## 1. Design center

The main implementation goal is to share the smallest valid read semantics
between:

1. standalone `.sealgraph` object storage, and
2. Git-backed content/history used by `git sealgraph`.

The shared abstraction is not “Git repository”. It is immutable content reading
plus sealgraph-specific refs/domain logic. Native v2 does not pre-commit a
future Git backend to the native SHA-256 `ObjectID` representation: a sidecar
with SHA-1/SHA-256 or blob/tree/commit sources requires an explicit typed
identity decision before its product reader is introduced.

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

At a seal ownership boundary, code must retain both the expected REF and seal
ObjectID. A bare ObjectID identifies stored bytes; it does not by itself
authorize using a seal as another REF's HEAD, parent, or tag target.

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

Git compatibility in standalone is deliberately narrower than repository
compatibility. An explicitly configured Git SHA-256 low-level object API may
hash and read native loose blobs. `.sealgraph` is not a Git repository, must not
be attached as a SHA-1 alternate, and must not be subjected to Git maintenance
or porcelain lifecycles.

## 5. Git content reader

Git sidecar should use a mature Git SDK rather than hand-implementing packfiles, deltas, alternates, worktrees, or repository object-format details.

Initial candidate: `go-git` stable v5 series.

Adding the SDK is not required for standalone ODB conformance. Before a product
`GitObjectReader` is implemented, an ADR must decide supported Git object
formats, typed identities, materialized-versus-external content references, and
the meaning of blob/tree/commit content. Until then, `ObjectReader` describes a
narrow architectural seam rather than a claim that native and Git identities
are interchangeable.

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

1. acquire the repository-wide writer guard,
2. load one REF candidate version,
3. validate candidate and complete dependency admissibility,
4. canonicalize payload,
5. write immutable objects,
6. revalidate state required by the writer protocol,
7. atomically CAS-update exactly one REF HEAD,
8. clear only the unchanged candidate version,
9. release the writer guard.

The successful REF CAS is the publication linearization point. Ref update must
retain compare-and-swap semantics even though cooperative writers are
serialized, because external filesystem mutation does not honor the writer
guard. Object writes that precede a failed publication remain immutable and may
be reported as dangling.

## 8. Future extension points

Keep interfaces narrow enough to permit:

- alternate content stores,
- remote CAS readers,
- signatures/attestations,
- richer relation types,
- machine-readable command output.

Current seals remain owned by exactly one REF. Internal traversal may load an
object by ID, but every binding to a HEAD, parent, tag, or selector validates
the owner REF. If future requirements introduce a cross-REF alias or federated
lookup, model it as an explicit scoped resolver returning the resolved
`(REF, seal ID)` identity. Do not weaken current ownership validation or add a
speculative native-v2 field.

Do not implement these until requirements exist.
