# Dogfooding receipt: R2 recurring workflow

Date: 2026-08-17

Result: PASS. Canonical Seal metadata was committed and reproduced from a
fresh detached checkout before the PERT milestone was closed.

## Identity boundary

- `SOURCE_COMMIT`:
  `41440e97d586f32db2aa415a59c2ee7e2f5cc7c4`
- `SEAL_METADATA_COMMIT`:
  `955e02148b9a11510df0a3eb87fe1ee52a56cdb7`; it contains this receipt,
  object `2b0b458f...`, and the new REF manifest and identifies
  `SOURCE_COMMIT`; it is not claimed as sealed source.
- current outer checkout during sealing: `SOURCE_COMMIT` plus only uncommitted
  canonical Seal metadata and this receipt.

The source commit adds the recurring runbook and starts the PERT task. The Seal
metadata is deliberately a later commit because a commit cannot truthfully
contain a manifest that claims its own not-yet-known identity.

## Frozen source and manifest

The source was exported with `git archive SOURCE_COMMIT` into a fresh temporary
directory. The binary was built from that export, not from the later working
tree:

```text
go version                       go1.26.5 linux/amd64
build flags                      -buildvcs=false -trimpath
binary sha256                    6f0f2eba08bd2f52f66a058a22d707710b6dbc631d6aa33dcaafa68daa1b1b60
dogfooding-plan.md sha256        363199ee2d4a893481b739ef6ae305bf71d4873d6cafa46146b8c61236ae0103
manifest bytes                   434
manifest sha256                  713d1f30164b3fa4ca6e06b238fb2b4faef418c7fd7f25537a5b095f880538b3
manifest entries aggregate      99a692f263f527bbeaf28e07f1c847335a4420e92344978ec55b67156e0b097b
manifest content ObjectID        7900d20843f201ea48089b4cea81c5c1c67d52e5299e049ffb318f856095d255
```

The exact manifest source is
`git:41440e97d586f32db2aa415a59c2ee7e2f5cc7c4`; its sole explicit path is
`docs/process/dogfooding-plan.md`. Standalone manifest generation received
both values as operands and did not discover Git state.

## Explicit candidate and Seal

The new logical REF is `sealgraph/process/dogfood-workflow`. The reviewed
candidate contained the manifest and two concrete Cause links, added in
separate commands so each message remained explicit:

```text
3107ab4ae270f90971d6d6cb00497b00cb2637fb188d2eb041609d68fa8317fd
  message="normative standalone invariants"
3de6bd9dee2e3f0537f818834706a04966073621b579dc82cafecb7bf3d93624
  message="operator workflow boundary"
```

One `seal sealgraph/process/dogfood-workflow` invocation published exactly:

```text
2b0b458f4e784320d0a4679977fdddeda775865a8313f5e6a090cb6a4abeeca7
```

Candidate diff showed initial content addition and two Link additions in
canonical target-ID order. After publication, `show`, `status`, `graph`,
`impact`, `log`, and an explicit `diff` all emitted their version-1 JSON
schemas. All six current REFs were `CLEAN`; impact from the architecture Seal
included the new workflow REF as one structural downstream path.

Inspection artifact SHA-256 values (temporary, not committed as canonical
state) were:

```text
show     144cd9ba88600323e9f9d121a1f35d056a9c7e50b47972dc0ac215bea051c224
status   9b44efad881862b7ad91bfa93b6adccafe021a3332f067c4923fe7d2d1092dfc
graph    33ea80a61609aaea7cef4291077aaea051aa486b3d7382385393e1c76d7025cd
impact   ca88ddd6efd1a7a37f9195712a29cc758cbf122acbf68f45ce1fad2c95d8729c
log      67813b504c4dc5a6f4b848e21eda5324efb9c4593e130ee28ab432a36c4988a4
diff     9f7a6939f5ecd851a4969910d93305e465a2cae418d31c210898ba93e3ed14f2
```

## Commit boundary audit

The metadata commit stages only this receipt, two new immutable loose objects,
and one REF manifest. `.sealgraph/index`, `cache`, `locks`, logs, candidates,
the temporary export, generated JSON evidence, and the binary are excluded.
No hook, batch seal, recursive repair, automatic relink, Git sidecar, push, or
release action was used.

## Fresh-checkout acceptance

A detached worktree of exact `SEAL_METADATA_COMMIT`
`955e02148b9a11510df0a3eb87fe1ee52a56cdb7` initially lacked ignored runtime
directories. Explicit init reported exactly:

```text
BOOTSTRAPPED_RUNTIME index,locks
```

The binary built from the separate `SOURCE_COMMIT` export regenerated the
manifest byte-for-byte (`cmp` success, SHA-256 `713d1f...538b3`). Read-only
`show --format json` and `status --format json` in the fresh checkout reproduced
the original SHA-256 values `144cd9...c224` and `9b44ef...dfc`. All six heads
were clean, including exact workflow Seal `2b0b458f...eca7`.

The current outer checkout while recording this acceptance is the later
follow-up commit candidate containing only the verified receipt, backlog,
checklist, and PERT closure. It is neither `SOURCE_COMMIT` nor
`SEAL_METADATA_COMMIT` and is not claimed by the Seal. `DOGFOOD_R2` is accepted;
Git sidecar and release remain separate gates.
