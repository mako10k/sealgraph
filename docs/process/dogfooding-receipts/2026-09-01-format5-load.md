# Format-5 tracked dogfood load — 2026-09-01

Status: `FORMAT5_DOGFOOD_LOAD` passed locally. The tracked standalone
repository now uses format 5. The complete format-4 source, migration document,
load receipts, and cutover evidence remain retained outside the repository.
This record does not authorize commit, push, PR, merge, tag, release, or
publication.

## Authority and frozen inputs

The Operator approved the exact migration candidate recorded in
[`format5-dogfood-migration-proposal-2026-09-01.md`](../format5-dogfood-migration-proposal-2026-09-01.md)
after the three-scope review in
[`format5-dogfood-migration-review-2026-09-01.md`](../format5-dogfood-migration-review-2026-09-01.md).
Execution remained on the existing task branch and made no remote write.

```text
source commit                 80086d232c1f1b6d49a28ca142915518ea409872
proposal sha256               1b02d9f39c3102b8510ae76fa19655a70e93264a4f2f73fb9b2611030afe920b
review sha256                 76a770fe3804ce792d6efefd7a6a63d7007c8fb7d3a10e7df073da6d17e0695c
Go                            go1.26.7 linux/amd64
reviewed binary version       sealgraph 0.1.0-dev
reviewed binary sha256        80779c759f41247f9f5e134ab22273f2e29f190f3753a5bfc8e55f43927f2c5c
source physical snapshot      f3faf0a0eecc9b8490b12c5457a8933da193f95f6caeca61bf1c07e0e8cde314
source Git subtree            a89957ce7db747df7546fa9bb4d8a3b5aa80f4bc
tracked-index listing sha256  fd11a81a2cdd96ff40fce85d4360006e5374c1acc93a1eb682b995e8b524e5fc
```

The approved exclusive maintenance window covered final preflight through
final readback. The complete execution block stopped with status 0. GNU
`cp -n` emitted its standard portability warning; the proposal pinned GNU
coreutils 9.4, and the copied path, types, modes, sizes, bytes, dump, and warning
set all matched before cutover.

## Migration artifact and semantic result

The isolated format-4 extractor ran three times during cutover: twice before
load and once after relocating the tracked-checkout copy. Every document and
stderr pair was byte-identical. The formal retained source remained unchanged.

```text
document schema               sealgraph/universal-blob-migration/v1
document bytes                69331
document sha256               9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048
extract warning sha256        dce9595d24b2e64602836725847f23b4ec006d85d363f803bfe1dfc1bf187d7e
format-4 objects              6
format-4 Seals                7
REFs                          6
tags                          4
excluded objects              0
unobserved parent records     2
materialized parent records   0
collapsed revision records    0
merged Cause-Link records     0
```

Exact warning output was:

```text
SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=2
```

The receipt contains seven old/new Seal mappings and one collapse group:
format-4 Seals `854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f`
and `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4`
both map to format-5 Seal
`271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef`.
The proposal records the complete Seal, REF, tag, and semantic mappings.

## Published repository and readback

The validated sibling repository was installed at the tracked root only after
the retained source reproduced the approved artifact again. Staged and final
fsck bytes match, as do the original and recovered load-receipt bytes.

```text
published format              5
load receipt bytes            7720
load receipt sha256           739d68ed690d8f28f0eab1282835f20818826e8fe41f265eb1cf45491241fd19
repository digest             2232438b9cb37a3f068a7d81e44bcc00f03f46412802b77629a9478ef6f105b0
typed object role entries     17
unchanged content Blobs       6
physical Blob files           23
Seals                         6
Materials                     6
Provenances                   5
REFs                          6
tags                          4
collapse groups               1
fsck sha256                   358646536024f3ea6295cfcdcd1f92163b404662e32420edd9837d3e9160faa5
fsck result                   ok
```

All six REF heads are non-draft, `NO_CANDIDATE`, non-stale, and
`SEALED_STATE_CLEAN`. The graph has six active-leaf nodes. There are no detached
Seals or unreferenced Blobs.

Retained evidence is directly selectable at:

```text
audit artifacts       /home/katsumata-m/sealgraph-format5-audit-80086d2
formal format-4       /home/katsumata-m/sealgraph-format4-retained-80086d2/.sealgraph
cutover format-4 copy /home/katsumata-m/sealgraph-format4-cutover-80086d2/.sealgraph
format-5 target       /home/katsumata-m/sealgraph/.sealgraph
```

The formal and cutover sources both still declare format 4. The final target
declares format 5. No retained directory or audit artifact was deleted.

## Exact-binary artifact smoke

The reviewed binary was exercised again from
`/tmp/sealgraph-format5-artifact-smoke.48auXL` after cutover:

1. an ordinary `fsck` against the retained format-4 source failed closed with
   `FORMAT4_REQUIRES_MIGRATION` and named both explicit commands;
2. the migration-only extractor reproduced the approved document and warning;
3. an absent target loaded the document and reproduced the approved receipt;
4. `load-receipt` reproduced the same receipt;
5. fsck reproduced the approved result;
6. status reported six clean heads, stale scan returned an empty `statuses`
   array, and graph returned six nodes; and
7. the retained format-4 physical snapshot was identical before and after.

```text
format-4 refusal sha256       479d2b8c824251776befa7e6c46578a66e17c1dd770d0042a23edfa66d3714d9
source before/after sha256    f3faf0a0eecc9b8490b12c5457a8933da193f95f6caeca61bf1c07e0e8cde314
status sha256                 624ec2115987f79c9906dd3bcfdab52c415bd7d8c1872c88e35e7023f79840c8
stale sha256                  93759764dd53c9ec5b4f2dc542d842f9c8058700a3f7b412589cb21ea260b69a
graph sha256                  62777fa3fcbfb10c8fec65865d8ec4f9d3c56015764560a189b2f03f0e27abe3
```

Self-review initially asserted that a clean stale scan must emit zero bytes,
reusing the older format-4 receipt's output contract. Format 5 instead returns
canonical `sealgraph/stale/v2` JSON even when `statuses` is empty. The first
smoke stopped at that assertion without changing either source. The same
retained evidence was inspected, the JSON contract was checked explicitly, and
the unfinished graph/source-readback steps then passed in the same smoke
directory. This was a validation-expectation defect, not a product failure or a
second migration attempt.

## Git working-tree result

Relative to source commit `80086d2`, the canonical migration changes are:

- config format `4 -> 5`;
- seven tracked format-4 Seal object files removed;
- 17 new typed format-5 object files added while six content Blob files remain
  byte-identical, for 23 physical Blob files total; and
- six REF manifests rewritten according to the approved receipt, retaining
  four scoped tags at their mapped targets.

The proposal, independent-review record, this receipt, and their documentation
index entries are the only process-document additions or changes in this
working tree. No source code changed during migration or validation.

## Repository validation

Final invocations passed:

```text
gofmt -w .                              OK; no Go diff
go vet ./...                            OK
go test ./...                           OK
go test -race ./...                     OK
npm ci                                  OK; 0 vulnerabilities
npm run clone-check                     OK; 0 clones
make completion-check                   OK
make complexity-check                   OK; no function over 20
make deadcode-check                     OK; no unreachable function
perttool document check PLAN.pert       OK
perttool dag analyze PLAN.pert --schedule both
                                         OK
perttool dag next PLAN.pert --format json
                                         OK; planning evidence only
git diff --check                        OK
live fsck/load-receipt hash readback     OK
exact-binary migration artifact smoke   OK after expectation correction above
```

The PERT document is structurally and schedulably valid, but remains a prior
planning state: `CAUSE_REVISION_CONTRACT` is still active and the format,
runtime, and normative-sync slices remain upcoming. This receipt does not
rewrite that causal history or claim those PLAN tasks complete. Normative/PLAN/
release-checklist synchronization and independent release review remain the
next release gate.

## Boundary after execution

The successful local migration and its validation do not prove remote
synchronization, merge, tag, release-artifact contents, or publication. Commit,
push, PR, merge, release, and removal of retained evidence each require their
own authority and readback.
