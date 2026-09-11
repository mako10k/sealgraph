# Format-5 tracked dogfood migration proposal — 2026-09-01

Status: `PROPOSED`; execution is not authorized. This record freezes the exact
format-4 source observation, migration document, warning set, format-5 preview,
destination paths, and command sequence required by
[ADR 0026](../adr/0026-format4-to-universal-blob-migration.md). It does not
authorize repository replacement, commit, push, release, or deployment.

## Decision requested from the Operator

Approve or reject this exact migration candidate as one unit:

1. source commit and tracked `.sealgraph` identity below;
2. canonical dump SHA-256
   `9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048`;
3. the two `UNOBSERVED_PARENT_DROPPED` records and the independent one
   many-old-to-one-new Seal collapse;
4. the complete Seal, REF, and tag mappings below;
5. the exact destination, retained-source, and audit paths;
6. the exact command sequence and its fail-stop boundary;
7. the configured/documented external-reference inventory and rewrite
   classification below; and
8. one exclusive maintenance window in which no Sealgraph or direct filesystem
   writer changes the source from final preflight through final readback.

Independent review of this document and the exact dump bytes is required before
the decision. Approval of the candidate will authorize only the migration
sequence in [Execution candidate](#execution-candidate), including the loader's
sibling repository publication and final `.sealgraph` replacement. Git push,
PR, merge, tag, release, and external artifact publication remain separately
gated.

## Claim / evidence / action trace

| ID | Claim | Evidence | Authorized action after exact approval |
| --- | --- | --- | --- |
| C-DM-001 | The candidate dump is bound to the current admitted format-4 dogfood value. | E-DM-001, E-DM-002, E-DM-003 | A-DM-001 regenerate and retain the exact approved dump. |
| C-DM-002 | Every identity rewrite and semantic change is explicit; no Candidate or excluded object is silently converted. | E-DM-003, E-DM-004 | A-DM-002 load only the approved bytes and require the approved warning/receipt. |
| C-DM-003 | The projected format-5 repository is complete and valid before the tracked source is touched. | E-DM-005, E-DM-006 | A-DM-003 validate the sibling target before replacement. |
| C-DM-004 | Every configured or documented SealID persistence surface is classified as active rewrite, historical retention, or no reference. | E-DM-007 | A-DM-004 rewrite only the active tracked `.sealgraph`; preserve every historical reference. |
| C-DM-005 | A complete format-4 source remains directly selectable as `<workdir>/.sealgraph` without being renamed, edited, or marked by migration. | E-DM-008 and the copy-verify/no-delete command sequence | A-DM-005 establish the exact retained source, relocate only the tracked-checkout copy, and install the validated format-5 directory. |

## Frozen source and tool evidence

```text
source repository              /home/katsumata-m/sealgraph
source commit                  80086d232c1f1b6d49a28ca142915518ea409872
source tree                    0e7e5624b9d9397ba85da8a91b6d08458296fe1d
.sealgraph Git subtree         a89957ce7db747df7546fa9bb4d8a3b5aa80f4bc
tracked-index listing sha256   fd11a81a2cdd96ff40fce85d4360006e5374c1acc93a1eb682b995e8b524e5fc
physical snapshot sha256       f3faf0a0eecc9b8490b12c5457a8933da193f95f6caeca61bf1c07e0e8cde314
physical snapshot entries      71
physical snapshot evidence     /tmp/sealgraph-dogfood-format5-proposal.7E9DIU/source-physical-snapshot.txt
source repository format       4
Go                             go1.26.7 linux/amd64
reproducible binary sha256      80779c759f41247f9f5e134ab22273f2e29f190f3753a5bfc8e55f43927f2c5c
reviewed binary evidence       /tmp/sealgraph-dogfood-format5-proposal.7E9DIU/sealgraph.fixed-a
build command                  go build -buildvcs=false -trimpath -ldflags=-buildid= -o PATH ./cmd/sealgraph
```

E-DM-001: `HEAD`, its tree, the `.sealgraph` subtree, and the tracked-index
listing were captured independently. The worktree was clean before this
proposal was added. The physical snapshot records relative path, entry kind,
mode, size, and SHA-256 for every regular file, including explicitly excluded
local state. That physical snapshot describes the review-time source; excluded
local state is not canonical migration identity. Execution is instead gated by
the tracked subtree/index plus exact regenerated dump bytes.

E-DM-002: two builds using the command above but different output paths were
byte-identical. The empty Go Build ID is deliberate: an earlier self-review
found that the default path-dependent Build ID made the whole-binary digest
unsuitable as a reproducible execution gate.

## Exact dump and warnings

The reviewed candidate bytes are currently retained at the task-scoped path
`/tmp/sealgraph-dogfood-format5-proposal.7E9DIU/dogfood.fixed.json`.
Owner approval binds to the digest and contents, not to the temporary pathname.
Execution must regenerate the same bytes and retain them under the audit path
below before touching the source.

```text
schema                         sealgraph/universal-blob-migration/v1
dump bytes                     69331
dump sha256                    9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048
objects                        6
format-4 Seals                 7
REFs                           6
tags                           4
excluded objects               0
materialized parent records    0
unobserved parent records      2
collapsed revision records     0
merged Cause-Link records      0
extract stderr bytes           50
extract stderr sha256          dce9595d24b2e64602836725847f23b4ec006d85d363f803bfe1dfc1bf187d7e
```

Exact stderr is:

```text
SEMANTIC_CHANGE_UNOBSERVED_PARENT_DROPPED count=2
```

E-DM-003: the isolated extractor ran twice against the unchanged source and
produced byte-identical dump and stderr outputs. Source config remained format
4 and `git status --short` remained empty.

The constant excluded-state declaration is exactly:

```json
["source_bindings","cache","event_logs","recovery_journal","locks","temporary_files"]
```

There is no Candidate and no excluded loose object. Source bindings, cache,
empty lock directories, and other declared local categories stay only in the
retained format-4 directory. They are not provenance and are not recreated by
load.

## Semantic changes and identity collapse

Both old parent edges are intentionally unobserved in format 5 because no
format-4 Cause Link owns them:

| Old child | Old parent | New child | New parent |
| --- | --- | --- | --- |
| `376084c544d9b85c68c62bf335dc41e7782847475d699b1c456c0f32465888e2` | `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4` | `612f3bdde49332a3e21b2ae8866a1cc8581c5fca862297204cff117cff9f3054` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |
| `854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f` | `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |

The former global edges acquire no format-5 graph meaning. The second record
does not create a self-revision edge. Independently, exact canonical projection
proves that these two old Seals become one new Seal:

```text
854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f \
  -> 271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef
fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4 \
  -> 271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef
```

E-DM-004: the canonical dump contains the complete old-graph conversion input
and two unobserved semantic records. The load receipt contains the complete
old/new projection detail and the derived collapse group. There is no
materialized, collapsed-revision, or merged-Cause-Link record.

## Complete Seal mapping

| Format-4 Seal | Format-5 Seal |
| --- | --- |
| `2b0b458f4e784320d0a4679977fdddeda775865a8313f5e6a090cb6a4abeeca7` | `e3aa0d40877c8e098905bcd163bafeac540d2b0e767f2e69e54d94e69e0e59b9` |
| `3107ab4ae270f90971d6d6cb00497b00cb2637fb188d2eb041609d68fa8317fd` | `ebf307daa43bb69f13d8220d1ec552f17a0221db59290d7ab15b16001a97d28d` |
| `376084c544d9b85c68c62bf335dc41e7782847475d699b1c456c0f32465888e2` | `612f3bdde49332a3e21b2ae8866a1cc8581c5fca862297204cff117cff9f3054` |
| `3de6bd9dee2e3f0537f818834706a04966073621b579dc82cafecb7bf3d93624` | `93c7cdb95e452da1c91d95240fe431f5c911c30d654312f05c71d9de3af56f46` |
| `854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |
| `9661a245a560ee595a8d75dd0fda6700bf0aec875bffb6e68b0758723f03ae39` | `eb26e64a37ca4ba097f8d4b802111803414eac5d427834fad60a6f316d4c62ef` |
| `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |

## Complete REF and tag mapping

| REF | Format-4 head | Format-5 head |
| --- | --- | --- |
| `sealgraph/decision/adr-0009` | `9661a245a560ee595a8d75dd0fda6700bf0aec875bffb6e68b0758723f03ae39` | `eb26e64a37ca4ba097f8d4b802111803414eac5d427834fad60a6f316d4c62ef` |
| `sealgraph/design/architecture` | `3de6bd9dee2e3f0537f818834706a04966073621b579dc82cafecb7bf3d93624` | `93c7cdb95e452da1c91d95240fe431f5c911c30d654312f05c71d9de3af56f46` |
| `sealgraph/process/dogfood-workflow` | `2b0b458f4e784320d0a4679977fdddeda775865a8313f5e6a090cb6a4abeeca7` | `e3aa0d40877c8e098905bcd163bafeac540d2b0e767f2e69e54d94e69e0e59b9` |
| `sealgraph/spec/requirements` | `3107ab4ae270f90971d6d6cb00497b00cb2637fb188d2eb041609d68fa8317fd` | `ebf307daa43bb69f13d8220d1ec552f17a0221db59290d7ab15b16001a97d28d` |
| `sealgraph/spec/storage-format` | `376084c544d9b85c68c62bf335dc41e7782847475d699b1c456c0f32465888e2` | `612f3bdde49332a3e21b2ae8866a1cc8581c5fca862297204cff117cff9f3054` |
| `sealgraph/spec/storage-v3-preserved` | `854cf6e6b0b8c8eebf5704463f17085ee4a22df7667ed0aaf692918e5c2c684f` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |

| REF / tag | Format-4 target | Format-5 target |
| --- | --- | --- |
| `sealgraph/decision/adr-0009` / `accepted` | `9661a245a560ee595a8d75dd0fda6700bf0aec875bffb6e68b0758723f03ae39` | `eb26e64a37ca4ba097f8d4b802111803414eac5d427834fad60a6f316d4c62ef` |
| `sealgraph/design/architecture` / `reviewed-v3` | `3de6bd9dee2e3f0537f818834706a04966073621b579dc82cafecb7bf3d93624` | `93c7cdb95e452da1c91d95240fe431f5c911c30d654312f05c71d9de3af56f46` |
| `sealgraph/spec/requirements` / `normative` | `3107ab4ae270f90971d6d6cb00497b00cb2637fb188d2eb041609d68fa8317fd` | `ebf307daa43bb69f13d8220d1ec552f17a0221db59290d7ab15b16001a97d28d` |
| `sealgraph/spec/storage-format` / `format-3` | `fe0fb5d4dff403d82d819aa93c5be6dc0b0eb876667e08e4b9065f6f5a54dcd4` | `271c95a44b8908c613433c90bc23ee6bd44b70fa0fc5c403ff3fd6d4fd4db8ef` |

## Isolated format-5 preview

The exact approved candidate was loaded into an absent task-scoped sibling
target without touching the tracked source.

```text
published format               5
load receipt bytes             7720
load receipt sha256            739d68ed690d8f28f0eab1282835f20818826e8fe41f265eb1cf45491241fd19
source document sha256         9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048
repository digest              2232438b9cb37a3f068a7d81e44bcc00f03f46412802b77629a9478ef6f105b0
format-5 physical Seals        6
typed object role entries      17
unchanged content Blobs        6
collapse groups                1
REFs                           6
tags                           4
fsck result                    ok
fsck sha256                    358646536024f3ea6295cfcdcd1f92163b404662e32420edd9837d3e9160faa5
status records                 6; no Candidate, draft, or stale head
graph nodes                    6
```

E-DM-005: loads performed with the initial exact build and the reproducible
build produced byte-identical receipts and warning outputs. The loaded config
is exactly format 5.

E-DM-006: `fsck --format json` returned `sealgraph/fsck/v2`, `result: ok`, 23
Blobs, 6 Seals, 6 Materials, 5 Provenances, 6 REFs, 4 tags, no detached Seal,
and no unreferenced Blob. The two previews produced byte-identical fsck output.

## SealID consumer inventory

E-DM-007 inventories every persistence or integration surface named by
`docs/integrations.md`, the configured Git remote, and the repository's GitHub
release/tracker surfaces. Each search used the exact seven old SealIDs.

| System / owner | Observed persistence | Classification and action |
| --- | --- | --- |
| Current tracked `.sealgraph` / repository Operator | Six active REF manifests contain the old heads/tag targets. | **Active rewrite.** The exact receipt mapping in this proposal rewrites all six REFs and four tags. |
| Current tracked process documents / repository Operator | Only `2026-08-17-format4-load.md` and `2026-08-17-r2-recurring.md` contain the old IDs outside `.sealgraph`. | **Historical retention.** Do not rewrite evidence of the format-4 event. |
| `origin/codex/cause-derived-revision-links` at `2a2b08b72475e36efea6873608c60695e8ca81f7` / repository Operator | Contains 20 exact textual old-ID occurrences. | **Deferred synchronization.** A later separately authorized push carries this migration receipt mapping; no remote write is part of migration execution. |
| `origin/main` at `2143304854fa90227c64f17ef6ec71780d6e55e4` / repository Operator | Contains 20 exact textual old-ID occurrences. | **Deferred synchronization.** A later separately authorized PR merge carries the active mapping; historical commits remain unchanged. |
| `origin/plan/normalize-recovery-completion` at `b1166444172e0f257cdd09bd14216ca67b505c15`, `origin/release/v0.1.0-beta.5` at `a18c2f9b58e8bf81fefbf3bc0009c55f9c2d1b72`, and `origin/release/v0.1.0-beta.6` at `2143304854fa90227c64f17ef6ec71780d6e55e4` / repository Operator | Each contains 20 exact textual old-ID occurrences and is at or behind the observed `origin/main` history. | **Historical retention.** Migration execution does not update or delete these remote refs. |
| Git tags `v0.1.0-beta.1` through `v0.1.0-beta.6` and their GitHub release/source artifacts / repository Operator | Each immutable tag contains 20 exact textual old-ID occurrences: 10 in REF-manifest head/tag fields and 10 in historical process documents. GitHub exposes all six prereleases; beta.6 assets are checksums plus one Linux binary archive. | **Historical retention.** Tags, past releases, and source archives are not rewritten. The first format-5 release is separately gated. |
| GitHub Issues, PRs, issue/PR/review/commit comments, and review bodies for `mako10k/sealgraph` / repository Operator | Exact API/search checks returned zero old-ID hits. | **No reference; no rewrite.** |
| GitHub Wiki for `mako10k/sealgraph` / repository Operator | Repository metadata enables Wiki, but the wiki repository has no `HEAD` and is not created. | **No reference; no rewrite.** |
| Git sidecar / repository Operator | It uses the same real worktree `.sealgraph` and defines no second Seal store. | **Covered by active rewrite.** No independent mapping target exists. |
| llmthink `/home/katsumata-m/llmthink` | Exact recursive search excluding `.git` returned zero hits; integration consumes repository-local `.think` documents. | **No reference; no rewrite.** |
| secdat `/home/katsumata-m/secdat` | Exact recursive search excluding `.git` returned zero hits; integration is an execution/secret boundary. | **No reference; no rewrite.** |
| perttool `/home/katsumata-m/perttool` | Exact recursive search excluding `.git` returned zero hits; integration consumes `PLAN.pert`, which also has zero hits. | **No reference; no rewrite.** |
| RefGraph `/home/katsumata-m/refgraph` | Exact recursive search excluding `.git` returned zero hits and it is not a configured Sealgraph persistence integration. | **No reference; no rewrite.** |
| GitHub Actions workflow definitions | Exact current-tree search returned zero hits. | **No reference; no rewrite.** Past run products are release/build evidence, not active SealID selectors. |

No configured or documented persistence surface remains unclassified. This is
a bounded inventory, not a universal claim that an unidentified system cannot
exist. If the Operator identifies another SealID consumer before execution, the
candidate must stop and add its receipt-based classification before a fresh
review; the current approval path cannot waive that gate.

## Exact paths and retained state

```text
reviewed binary      /tmp/sealgraph-dogfood-format5-proposal.7E9DIU/sealgraph.fixed-a
audit artifacts      /home/katsumata-m/sealgraph-format5-audit-80086d2
sibling load target  /home/katsumata-m/sealgraph-format5-migration-80086d2
retained workdir     /home/katsumata-m/sealgraph-format4-retained-80086d2
retained source      /home/katsumata-m/sealgraph-format4-retained-80086d2/.sealgraph
cutover-copy workdir /home/katsumata-m/sealgraph-format4-cutover-80086d2
cutover checkout     /home/katsumata-m/sealgraph-format4-cutover-80086d2/.sealgraph
final target         /home/katsumata-m/sealgraph/.sealgraph
```

E-DM-008: the audit, sibling-target, retained-workdir, and cutover-copy paths
were absent at proposal freeze; the reviewed binary path already existed with
the frozen digest. Execution first copies the complete current `.sealgraph`
into the retained workdir, reproduces its physical snapshot and approved dump,
and then treats that unmodified copy as the formal migration source. The later
rename relocates only the tracked checkout copy so the final target can be
installed. No step deletes the formal source, checkout copy, artifact, or
receipt.

## Execution candidate

The following is the complete proposed action. It is intentionally not run by
preparation or review. Before execution, the primary agent must re-read the
exact approved proposal digest and run the preflight checks without weakening
an expectation. The Operator must also confirm the exclusive maintenance window
in Decision item 8; a process scan can supplement but cannot replace that human
coordination boundary.

```sh
set -eu
umask 077
export GIT_OPTIONAL_LOCKS=0

repo_path=/home/katsumata-m/sealgraph
audit_path=/home/katsumata-m/sealgraph-format5-audit-80086d2
stage_path=/home/katsumata-m/sealgraph-format5-migration-80086d2
retained_repo_path=/home/katsumata-m/sealgraph-format4-retained-80086d2
cutover_repo_path=/home/katsumata-m/sealgraph-format4-cutover-80086d2
reviewed_binary=/tmp/sealgraph-dogfood-format5-proposal.7E9DIU/sealgraph.fixed-a
reviewed_snapshot=/tmp/sealgraph-dogfood-format5-proposal.7E9DIU/source-physical-snapshot.txt
expected_source_commit=80086d232c1f1b6d49a28ca142915518ea409872
expected_source_subtree=a89957ce7db747df7546fa9bb4d8a3b5aa80f4bc
expected_index_listing=fd11a81a2cdd96ff40fce85d4360006e5374c1acc93a1eb682b995e8b524e5fc
expected_dump=9550786c7e0eba75d77667a4634605fcebfeff03ea86205f11ba8d725fca3048
expected_binary=80779c759f41247f9f5e134ab22273f2e29f190f3753a5bfc8e55f43927f2c5c

cd "$repo_path"
test "$(git rev-parse HEAD)" = "$expected_source_commit"
test "$(git rev-parse HEAD:.sealgraph)" = "$expected_source_subtree"
test "$(git ls-files -s .sealgraph | sha256sum | cut -d ' ' -f1)" = \
  "$expected_index_listing"
git diff --quiet -- .sealgraph
git diff --cached --quiet -- .sealgraph
test -z "$(git status --porcelain --untracked-files=all .sealgraph)"
test "$(sha256sum "$reviewed_binary" | cut -d ' ' -f1)" = \
  "$expected_binary"
test "$(sha256sum "$reviewed_snapshot" | cut -d ' ' -f1)" = \
  f3faf0a0eecc9b8490b12c5457a8933da193f95f6caeca61bf1c07e0e8cde314
test "$(mv --version | sed -n '1p')" = 'mv (GNU coreutils) 9.4'
test "$(cp --version | sed -n '1p')" = 'cp (GNU coreutils) 9.4'
test ! -e "$audit_path"
test ! -e "$stage_path"
test ! -e "$retained_repo_path"
test ! -e "$cutover_repo_path"

mkdir "$audit_path"
mkdir "$stage_path"
mkdir "$retained_repo_path"
mkdir "$cutover_repo_path"
expected_device=$(stat -c '%d' "$repo_path")
test "$(stat -c '%d' "$audit_path")" = "$expected_device"
test "$(stat -c '%d' "$stage_path")" = "$expected_device"
test "$(stat -c '%d' "$retained_repo_path")" = "$expected_device"
test "$(stat -c '%d' "$cutover_repo_path")" = "$expected_device"
install -m 0700 -- "$reviewed_binary" "$audit_path/sealgraph"
install -m 0600 -- "$reviewed_snapshot" \
  "$audit_path/source-physical-snapshot.txt"
test "$(sha256sum "$audit_path/sealgraph" | cut -d ' ' -f1)" = \
  "$expected_binary"
test "$(sha256sum "$audit_path/source-physical-snapshot.txt" | \
  cut -d ' ' -f1)" = \
  f3faf0a0eecc9b8490b12c5457a8933da193f95f6caeca61bf1c07e0e8cde314

snapshot_repository() {
  snapshot_root=$1
  snapshot_output=$2
  find "$snapshot_root" -mindepth 1 -print0 | sort -z | \
    while IFS= read -r -d '' entry; do
      rel=${entry#"$snapshot_root"/}
      if test -d "$entry"; then
        printf 'D\t%s\t%s\n' "$(stat -c '%a' "$entry")" "$rel"
      elif test -f "$entry"; then
        printf 'F\t%s\t%s\t%s\t%s\n' \
          "$(stat -c '%a' "$entry")" "$(stat -c '%s' "$entry")" \
          "$(sha256sum "$entry" | cut -d ' ' -f1)" "$rel"
      elif test -L "$entry"; then
        printf 'L\t%s\t%s\n' "$(readlink "$entry")" "$rel"
      else
        printf 'S\t%s\n' "$rel"
      fi
    done > "$snapshot_output"
}

snapshot_repository "$repo_path/.sealgraph" \
  "$audit_path/precopy-source-physical-snapshot.txt"
cmp "$audit_path/source-physical-snapshot.txt" \
  "$audit_path/precopy-source-physical-snapshot.txt"
cp -a --reflink=never -T -n -- "$repo_path/.sealgraph" \
  "$retained_repo_path/.sealgraph"
test -d "$retained_repo_path/.sealgraph"
snapshot_repository "$retained_repo_path/.sealgraph" \
  "$audit_path/retained-source-physical-snapshot.txt"
cmp "$audit_path/source-physical-snapshot.txt" \
  "$audit_path/retained-source-physical-snapshot.txt"

cd "$retained_repo_path"
"$audit_path/sealgraph" migrate extract \
  --source-format 4 --format universal-blob-v1 \
  > "$audit_path/repository.universal-blob-v1.json" \
  2> "$audit_path/extract.stderr"
"$audit_path/sealgraph" migrate extract \
  --source-format 4 --format universal-blob-v1 \
  > "$audit_path/repository.repeat.universal-blob-v1.json" \
  2> "$audit_path/extract.repeat.stderr"
cmp "$audit_path/repository.universal-blob-v1.json" \
  "$audit_path/repository.repeat.universal-blob-v1.json"
cmp "$audit_path/extract.stderr" "$audit_path/extract.repeat.stderr"
test "$(sha256sum "$audit_path/repository.universal-blob-v1.json" | \
  cut -d ' ' -f1)" = "$expected_dump"
test "$(sha256sum "$audit_path/extract.stderr" | cut -d ' ' -f1)" = \
  dce9595d24b2e64602836725847f23b4ec006d85d363f803bfe1dfc1bf187d7e

cd "$stage_path"
"$audit_path/sealgraph" load --format universal-blob-v1 \
  < "$audit_path/repository.universal-blob-v1.json" \
  > "$audit_path/load.receipt.json" \
  2> "$audit_path/load.stderr"
test "$(sha256sum "$audit_path/load.receipt.json" | cut -d ' ' -f1)" = \
  739d68ed690d8f28f0eab1282835f20818826e8fe41f265eb1cf45491241fd19
cmp "$audit_path/extract.stderr" "$audit_path/load.stderr"
"$audit_path/sealgraph" fsck --format json > "$audit_path/staged.fsck.json"
test "$(sha256sum "$audit_path/staged.fsck.json" | cut -d ' ' -f1)" = \
  358646536024f3ea6295cfcdcd1f92163b404662e32420edd9837d3e9160faa5

cd "$repo_path"
test "$(git rev-parse HEAD)" = "$expected_source_commit"
test "$(git rev-parse HEAD:.sealgraph)" = "$expected_source_subtree"
test "$(git ls-files -s .sealgraph | sha256sum | cut -d ' ' -f1)" = \
  "$expected_index_listing"
git diff --quiet -- .sealgraph
git diff --cached --quiet -- .sealgraph
test -z "$(git status --porcelain --untracked-files=all .sealgraph)"
test "$(sed -n 's/^repository_format = //p' .sealgraph/config)" = 4
snapshot_repository "$repo_path/.sealgraph" \
  "$audit_path/precutover-checkout-physical-snapshot.txt"
cmp "$audit_path/source-physical-snapshot.txt" \
  "$audit_path/precutover-checkout-physical-snapshot.txt"
test "$(stat -c '%d' "$repo_path/.sealgraph")" = "$expected_device"
mv --no-copy -T -n -- "$repo_path/.sealgraph" \
  "$cutover_repo_path/.sealgraph"
test ! -e "$repo_path/.sealgraph"
test -d "$cutover_repo_path/.sealgraph"
test -d "$retained_repo_path/.sealgraph"

cd "$retained_repo_path"
"$audit_path/sealgraph" migrate extract \
  --source-format 4 --format universal-blob-v1 \
  > "$audit_path/cutover-source.universal-blob-v1.json" \
  2> "$audit_path/cutover-source.stderr"
cmp "$audit_path/repository.universal-blob-v1.json" \
  "$audit_path/cutover-source.universal-blob-v1.json"
cmp "$audit_path/extract.stderr" "$audit_path/cutover-source.stderr"

test "$(stat -c '%d' "$stage_path/.sealgraph")" = "$expected_device"
mv --no-copy -T -n -- "$stage_path/.sealgraph" "$repo_path/.sealgraph"
test ! -e "$stage_path/.sealgraph"
test -d "$repo_path/.sealgraph"

cd "$repo_path"
"$audit_path/sealgraph" fsck --format json > "$audit_path/final.fsck.json"
cmp "$audit_path/staged.fsck.json" "$audit_path/final.fsck.json"
"$audit_path/sealgraph" load-receipt \
  --source-document-sha256 "$expected_dump" \
  > "$audit_path/final.load-receipt.json"
cmp "$audit_path/load.receipt.json" "$audit_path/final.load-receipt.json"
test "$(sed -n 's/^repository_format = //p' .sealgraph/config)" = 5
test "$(sed -n 's/^repository_format = //p' \
  "$retained_repo_path/.sealgraph/config")" = 4
test "$(sed -n 's/^repository_format = //p' \
  "$cutover_repo_path/.sealgraph/config")" = 4
git status --short
```

The `cp -a --reflink=never -T -n` operation is the source-retention write. It
does not rename, edit, or mark the current format-4 source; `-T -n` prevents
nesting or replacement if the destination appears after preflight. The complete
copied path, types, modes, sizes, bytes, dump, and warning set must match before
it becomes the formal migration source. The two `mv --no-copy -T -n`
operations then relocate only the tracked-checkout copy and install the loaded
target.
`--no-copy` and the same-device gates prohibit copy/delete fallback; `-T`
prevents accidental nesting; and `-n` plus immediate postconditions prevents
overwriting a path that appears after preflight. The new repository passes
complete load/fsck validation before either move. After the first move, the
unchanged formal source must again reproduce the approved dump and stderr
before final install.

If the first move succeeds but re-extraction or the second move fails, `set -e`
stops with the full formal source at `retained_repo_path/.sealgraph`, the
tracked-checkout copy at `cutover_repo_path/.sealgraph`, and the validated
format-5 repository at `stage_path/.sealgraph`. The repository-root
`.sealgraph` is either absent or an unknown path created by an actor that
violated the maintenance window. The agent must inspect those exact paths
read-only, report the observed state, and obtain separate recovery authority;
it must not silently restore, retry, delete, or choose a different destination.

This working-tree replacement makes no atomic-exchange or crash-durability
claim across the two moves. Interruption leaves the explicit source, staged,
retained, and final paths as inspection evidence; it does not authorize an
inferred recovery action.

The redirected audit copies are hash-verified but are not claimed crash-durable
by this shell sequence. The loader created fsync-synchronized repository
contents and receipt bytes inside its sibling target; those bytes are retained,
but the later cutover makes no final-namespace durability claim. Outer Git HEAD
still identifies the tracked format-4 canonical source. Before release, the
dump, receipt, migration record, and resulting Git revision must be synchronized
and read back under their separate gates.

The exclusive maintenance window excludes known cooperative writers. A direct
filesystem actor that ignores that coordination remains a residual risk. The
post-move extractor narrows this risk by binding the final install to a fresh
double capture from the retained source; it does not claim a system-wide lock
that the format-4 runtime does not implement.

After a successful move, a later format-5 validation failure also stops without
automatic downgrade. ADR 0026 rollback means selecting the retained format-4
source with the appropriate runtime, not rewriting the format-5 directory in
place.

## Completion and release boundary

After separately authorized successful execution:

1. record the actual outputs and final Git diff in
   `docs/process/dogfooding-receipts/2026-09-01-format5-load.md`;
2. rerun full repository validation, including artifact smoke of the exact
   format-4 extractor and format-5 loader path;
3. obtain separate commit authorization and read back that commit;
4. complete normative/PLAN/release-checklist synchronization and independent
   release review; and
5. obtain separate push, PR, merge, tag, and publication authorization at
   their respective gates.

This proposal closes none of those later gates.

## Review record

After self-review, freeze the SHA-256 of this exact proposal and review it
together with the exact dump digest above. Record independent findings in
`format5-dogfood-migration-review-2026-09-01.md` so a passing review does not
change the reviewed proposal bytes. Any later proposal edit requires a new
digest and fresh independent review.
