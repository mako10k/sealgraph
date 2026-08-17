# Content ingest acceptance — 2026-08-17

Status: `CONTENT_INGEST` complete. SG-BL-001 and SG-BL-002 are complete in the
local source; no external Issue was changed.

## Achieved scope

- `add --content-file PATH|-` preserves exact regular-file or stdin bytes,
  including NUL, CRLF, non-UTF-8 bytes, and a missing final LF.
- `--content` and `--content-file` conflict before repository traversal.
- Missing paths, directories, final symlinks, FIFOs, and devices fail before
  creating or changing a candidate. `add` still never seals automatically.
- ADR 0014 adds the standalone read-only command:

  ```sh
  sealgraph manifest --source SOURCE --file PATH [--file PATH ...]
  ```

- The command emits canonical `sealgraph/path-manifest/v1` JSON plus LF with a
  fixed `path-digest-only` claim, explicit source, algorithm names, bytewise
  path-sorted entries, exact byte lengths/SHA-256 digests, and an aggregate
  SHA-256 over the canonical entries array.
- Portable relative path validation rejects absolute paths, empty/dot/dot-dot
  components, backslash, controls, duplicates, missing paths, directories,
  symlinks in any component, FIFOs, sockets, and devices before stdout.
- No `.sealgraph` repository, Git state, environment-derived identity, glob,
  recursive walk, object, candidate, REF, Link, or Seal is inferred or written
  by `manifest`.

The fixed two-file canonical fixture has SHA-256
`920797199f7bc62b012d56fbd31c96da4115fa3f3ac5ebb0181fdc66da9d0f14`.

## Focused standalone dogfood

The implementation binary was built with `-buildvcs=false -trimpath` and had
SHA-256 `88f7b1b4bea0c0f0fef7c9c5f75df29777c3bbd09940228bbfa9d441722ff98c`.
It operated on a Git-metadata-free archive of predecessor commit
`b140f1fa3d4aec7fd4ecb7ce43721342c5454e83`, with the source identity supplied
explicitly as `git:b140f1fa3d4aec7fd4ecb7ce43721342c5454e83`.

Two invocations listing `docs/requirements.md` and `docs/architecture.md` in
opposite option order emitted byte-identical 545-byte manifests:

```text
manifest sha256            b14d88cc7797a95bfb40226787873f18516de66b3dfc377a41726d8ac0d0a833
entries aggregate sha256  a0cdaca7bf402f7769abc6a63320a78b70e20d2a3052d82b2dd21d01eaa1abc0
content ObjectID           9ae3886269f48b19d9c44111bd3d96e2d847e8e681821b7ed09ae80a2f5deeea
```

The aggregate was independently recomputed from the emitted canonical entries
array. Supplying the manifest through `add manifest/probe --root
--content-file PATH` produced one byte-identical candidate with the ObjectID
above. `show manifest/probe` still failed with `REF not found`, proving that no
automatic seal or REF publication occurred. The tracked project `.sealgraph`
was not used or changed by this focused run.

## Repository validation

```text
gofmt -w .                       OK
go vet ./...                     OK
go test ./...                    OK
go test -race ./...              OK
npm ci                           OK; 0 vulnerabilities
npm run clone-check              OK; 46 files, 0 clones
make complexity-check            OK; no function above 20
make deadcode-check              OK; 0 reported unreachable functions
perttool document check          OK
perttool dag analyze             OK; PTDAG-302 display warning only
git diff --check                 OK
```

## Remaining boundary

`USABLE` is not reached by this slice. `OPERATOR_CONTRACT` remains the sole
next PERT frontier. Recurring dogfood, Git sidecar, release, push/publication,
and external tracker mutation remain separately gated.
