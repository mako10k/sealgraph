# Native v3 material-identity dogfood receipt

Status: focused tracked regeneration passed on 2026-08-14.

## Frozen implementation input

- outer Git base commit:
  `1faf75d2237981355168e2c2a1e0449bac54c9c7`
- source state: reviewable uncommitted ADR 0008/0009 implementation worktree
- Go: `go version go1.26.5 linux/amd64`
- built `sealgraph` SHA-256:
  `d8c62207401fc778e87e54a842c88d63b0c350366388a57bb44308cd721e741e`
- product mode: standalone only
- repository config: `repository_format = 3`, `object_format = sha256`

This is pre-commit implementation evidence. It is not a release, signature,
trusted timestamp, actor authentication, or remote-publication claim.

## Exact document inputs

| REF | file | file SHA-256 | content ObjectID |
| --- | --- | --- | --- |
| `sealgraph/decision/adr-0009` | `docs/adr/0009-separate-seal-event-metadata.md` | `bae38ca8f0676d05096283e371970cf6f435f7029110f7555fd500874ff9d997` | `51724fd768cbf7713c6894f08c11f74b6714045aab8bf30ee6f4b1981448b1ac` |
| `sealgraph/spec/requirements` | `docs/requirements.md` | `1b10bbae310780459d8ac86095cec16980b3cb32184686bd8419f5ffd653a2a3` | `e453287c3f47837bfe973ff4b5bd55fdda4c94b1103ec97e9f778fd34016cf6c` |
| `sealgraph/design/architecture` | `docs/architecture.md` | `e183478440567e8f2e924e5c08e6865921c5f575d42c28a36ac928b2760865ca` | `4b63c1829d06fcb32566c7f4b15fa909ef2af4e38aad156a28d95f58b374baf3` |
| `sealgraph/spec/storage-v3` | `docs/storage-format.md` | `9ebdfdc8b032102e57ea25d3521718a57bb9768bcb0ef836a7a16f9de46f84fc` | `8501010c40b75133a103380504d16d61906cacb4f99636d58eba7763bf4c174a` |

File SHA-256 is a receipt digest. The content ObjectID is the Git-compatible
SHA-256 blob-envelope identity used by sealgraph.

## Resulting heads and tags

| REF | seal HEAD | tag |
| --- | --- | --- |
| `sealgraph/decision/adr-0009` | `907ac4a24ff402b05e00311f14d70399d0a3162aba4aa4ce31b9737c3082d773` | `accepted` |
| `sealgraph/spec/requirements` | `ed84f04e3f7ab7ca7b5868773b160f71515a38e8d2a44471f43cbf4f40a5c708` | `normative` |
| `sealgraph/design/architecture` | `cc16307efc0e2537413a667996c1c872725e640478760af6c37198ab3c4b4d99` | `reviewed-v3` |
| `sealgraph/spec/storage-v3` | `014d65f08ff03d616ae9f17a8f9a1e2a64f55269a0706d481b09f5d0505cdee4` | `format-3` |

The ADR is an explicit root provenance boundary. Requirements depend directly
on that ADR generation. Architecture depends on the ADR and requirements.
Storage v3 depends on the ADR, architecture, and requirements. All persisted
links contain full concrete seal IDs; edge messages describe only those
dependency relations.

## Material-identity readback

All four HEADs reported `CLEAN`. `show`, `log`, and `graph` displayed content,
parent, root/draft, and concrete dependency state without seal-operation event
fields.

An explicitly initialized temporary SHA-256 Git object context read each native
seal loose object with `git cat-file`. Every canonical payload had exactly these
top-level keys:

```json
["attachments","content","draft","links","parent","ref","root","schema"]
```

There was no top-level `message`, `created_at`, or `actor`. Edge-specific
`links[].message` remained canonical dependency relation state.

The new binary rejected the preserved format-2 repository as unsupported. It
did not rewrite, migrate, or repair that repository. The old tracked state
remains recoverable from outer Git history.

## Replacement boundary

The old `.sealgraph/` directory was moved intact before format-3 initialization,
not rewritten in place. The tracked diff deliberately removes old format-2
objects/REFs/tags and adds a fresh format-3 object graph. Runtime-only index and
lock paths remain ignored.

## Associated automated validation

The final validation run covers:

```text
gofmt -w .
go vet ./...
go test ./...
go test -race ./...
npm run clone-check
llmthink audits
perttool document/DAG checks
```
