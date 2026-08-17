# ADR 0014: Explicit-path digest manifests

Status: accepted for the `CONTENT_INGEST` slice on 2026-08-17.

## Context

`add --content-file PATH|-` already preserves exact bytes, but recurring
dogfood still needs a deterministic way to state which named files were
reviewed. Inferring a repository root, Git commit, glob expansion, recursive
directory walk, or source identity would cross the standalone boundary and
would make the selected input set less explicit.

## Decision

The standalone read-only command is:

```sh
sealgraph manifest --source SOURCE --file PATH [--file PATH ...]
```

`SOURCE` is one required non-empty valid UTF-8 caller-supplied identity. The
command never derives it from Git, the filesystem, an environment variable, or
another Sealgraph REF.

Each repeated `PATH` has one meaning: its exact slash-separated spelling is
the semantic manifest path, and the same relative spelling selects the file
below the current working directory. Paths are valid UTF-8 without control or
DEL bytes, are relative, contain no empty, `.` or `..` component, and contain
no backslash. Absolute paths, traversal, duplicates, missing paths,
directories, symbolic links in any component, devices, sockets, and FIFOs are
rejected before stdout. No glob, recursive walk, normalization, or path
mapping exists.

The command reads every regular file exactly, sorts entries by raw UTF-8 path
bytes, and emits compact canonical JSON plus one LF:

```json
{"schema":"sealgraph/path-manifest/v1","claim":"path-digest-only","source":"SOURCE","digest_algorithm":"sha256","aggregate_algorithm":"sha256-canonical-entries-v1","entries":[{"path":"docs/requirements.md","bytes":123,"sha256":"<64-lower-hex>"}],"aggregate_sha256":"<64-lower-hex>"}
```

Member order is fixed as shown. Entry member order is `path, bytes, sha256`.
`bytes` is the exact non-negative byte length in minimal decimal JSON form.
`sha256` hashes exact file bytes. `aggregate_sha256` hashes the exact canonical
JSON bytes of the complete `entries` array, from `[` through `]`. The schema,
claim, source, algorithm names, entries, aggregate, and trailing LF all affect
the native blob identity when the output is later supplied to `add`.

`claim = path-digest-only` means the output claims only the explicit semantic
paths, sizes, and observed digests. It does not claim that Sealgraph stored the
named files as content blobs or attachments. `manifest` writes no object,
candidate, REF, tag, cache, lock, or repository state and performs no seal.

## Consequences

- Input option order cannot change output bytes.
- A changed file byte, semantic path, or explicit source changes the manifest
  blob identity.
- Reproducibility does not depend on Git availability or checkout discovery.
- Shell glob expansion and recursive selection, if desired, remain explicit
  caller preprocessing and are not part of the stable CLI contract.
- A future semantic-path/source-path mapping or richer manifest field requires
  a separate versioned contract; this command does not guess one.
