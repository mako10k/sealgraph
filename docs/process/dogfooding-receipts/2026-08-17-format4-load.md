# Format-4 tracked dogfood load — 2026-08-17

Status: `FORMAT4_DOGFOOD_LOAD` passed. The tracked standalone repository is
canonical format 4 and the format-3 source remains recoverable from outer Git
history.

## Frozen inputs

```text
source commit              36bf115550c514ecddce21f1c062d344f649f8e6
format-3 exporter commit   5b24d47cb66e2ff2be000d4c9cb32cb59e7957fa
Go                         go1.26.5 linux/amd64
format-3 binary sha256     e7aade2f9685ae3e620ed1c833673c0f3e07b6cee8502263be7012834a815d23
format-4 binary sha256     c1a8582eca286e2cf23e821d0e89670a67ee0979c49a79d765ced532460e7372
source snapshot sha256     78014968a9a86f832e2ea8f5b684d8ced74caf40f28bffbcd50b35d0d6bc2b99
```

Both binaries were built with `-buildvcs=false -trimpath` from source archives
or the frozen worktree. The format-3 binary ran twice against the tracked
source. Both successful dumps were byte-identical, emitted no stderr, and left
the complete `.sealgraph` relative path, type, mode, and regular-file digest
snapshot unchanged.

```text
dump bytes                 45000
dump sha256                f63f44f7017884a11978be48eb323a33dc487dc41983c25d796cd4f32d2c1125
objects                    4
seals                      4
refs                       4
tags                       4
excluded objects           0
```

## Empty-target load receipt

The format-4 binary loaded the canonical dump only into an absent target. Its
1,867-byte canonical receipt has SHA-256
`4aa34b5c8b6b25678454a424ba4582c800e198698ef2d6d04d462f9af5155435`
and reports source digest `f63f44f7017884a11978be48eb323a33dc487dc41983c25d796cd4f32d2c1125`.

| Format-3 Seal | Format-4 Seal |
| --- | --- |
| `014d65f08ff03d616ae9f17a8f9a1e2a64f55269a0706d481b09f5d0505cdee4` | `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4` |
| `907ac4a24ff402b05e00311f14d70399d0a3162aba4aa4ce31b9737c3082d773` | `9661a245a560ee595a8d75dd0fda6700bf0aec875bffb6e68b0758723f03ae39` |
| `cc16307efc0e2537413a667996c1c872725e640478760af6c37198ab3c4b4d99` | `3de6bd9dee2e3f0537f818834706a04966073621b579dc82cafecb7bf3d93624` |
| `ed84f04e3f7ab7ca7b5868773b160f71515a38e8d2a44471f43cbf4f40a5c708` | `3107ab4ae270f90971d6d6cb00497b00cb2637fb188d2eb041609d68fa8317fd` |

The receipt contains four mappings, zero collapse groups, four rewritten REFs,
four rewritten tags, and `published_repository_format = 4`. Each loaded REF
was `CLEAN`; `stale --scan` emitted zero bytes. The canonical config is exactly:

```text
repository_format = 4
object_format = sha256
ref_format = manifest-v1
```

No legacy `refs/tags` entry, candidate, persisted stale value, or mixed-format
Seal was published. Each tag appeared in its REF manifest and retained its
rewritten target.

## Same-material sibling freeze

The fully loaded repository first passed isolated `status`, `stale --scan`,
`graph`, `log`, `show`, and tag-list inspection. The tested directory was then
installed at the project root only after the complete old `.sealgraph` was
moved intact to a separate temporary backup.

The storage REF was explicitly moved from `sealgraph/spec/storage-v3` to
`sealgraph/spec/storage-format`; its one `format-3` tag moved with the manifest
and still targets historical parent
`fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4`.
Two children were then sealed from that exact parent:

| REF | Head | Content | Result |
| --- | --- | --- | --- |
| `sealgraph/spec/storage-format` | `376084c544d9b85c68c62bf335dc41e7782847475d699b1c456c0f32465888e2` | `245ebf756abd859485ec0276bfa423aea4d9161527221c3be9f89d685aa6bb7b` | current format-4 storage document |
| `sealgraph/spec/storage-v3-preserved` | `854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f` | `8501010c40b75133a103380504d16d61906cacb4f99636d58eba7763bf4c174a` | exact old material preserved |

The preserved child has the same content, flags, attachments, and three exact
Cause Links/messages as its parent. Both children are distinct
`CURRENT_LEAF` revisions; the common parent alone is `STALE_REVISION`. No REF
points to the parent, no sibling is selected as preferred, all five current
REFs report `CLEAN`, and `stale --scan` still emits zero bytes.

## Repository validation

The final tracked state passed:

```text
gofmt -w .                       OK
go vet ./...                     OK
go test ./...                    OK
go test -race ./...              OK
npm ci                           OK; 0 vulnerabilities
npm run clone-check              OK; 0 clones
make complexity-check            OK; maximum 20
make deadcode-check              OK; 0 unreachable functions
perttool document check          OK
perttool dag analyze             OK; PTDAG-302 display warning only
git diff --check                 OK
```

Runtime cache, candidate/index, lock, and log paths are not canonical tracked
state. A checkout made only from the staged index therefore failed closed on
its missing runtime `index/` directory; explicit `sealgraph init` bootstrapped
the ignored directories, after which the same five clean heads, zero-byte
stale scan, graph, and historical tag were read back. The dump, load receipt,
temporary binaries/repositories, and intact format-3 backup were kept outside
the repository during validation and moved to desktop trash after the new
commit was independently read back.

## Boundary after acceptance

This receipt completes only explicit tracked format-4 conversion and sibling
behavior. `CONTENT_INGEST` and `OPERATOR_CONTRACT` are the next PERT frontier.
Recurring dogfood, Git sidecar, release, tag/publication, push, and external
tracker mutations remain separately gated.
