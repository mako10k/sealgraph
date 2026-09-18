# Issue #17 `DETAIL_TRIAL` — 詳細側の仮設文書試用

- 日付：2026-09-18。対象は Accepted [R3 実装計画 P3](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) §3 P8 と [正本 PERT](issue-17-origin-trace.pert) の `DETAIL_TRIAL`。status への新表示、性能基準、実データ移行、公開、最終受入の判断は本記録で行わない。
- 実行：branch `codex/issue-17-origin-trace-r2-wip`、試用前 HEAD `3739d23`。同 revision から build した binary の SHA-256 は `bda096ce37dbfc6ccc4ae4eb7eeec1a1bf7631ef70498fffc06c167231791ce8`。/tmp の専用仮設 repository でのみ操作した。`init` 後に、既存のテスト fixture と同じ format 7 config を置いた。これは5/6→7 migration の実行ではない。

## 仮設文書と手順

| REF | Seal content・Cause | Trace |
|---|---|---|
| `root` | `XYZ-UV`、root Seal `c3c59eaf3fe2` | run 0=`XYZ` を元 `a.txt` の byte 3、run 1=`-` を Untraced、run 2=`UV` を元 `b.txt` の byte 4。元ファイル全文はそれぞれ `abcXYZtailXYZ`（13 bytes）と `headUVtail`（10 bytes） |
| `middle` | `UV`、`root` の exact Seal を Cause、Seal `e2151a058f70` | `b.txt` の byte 4 から2 bytes |
| `leaf` | `leaf`、`middle` の exact Seal を Cause、Seal `5056cd580bc1` | Trace なし |
| `page` | `aa`、独立 root Seal `fe2cd5faa490` | 元 `page.txt=aaaa`、byte 0 から2 bytes |

`sealgraph add` → `trace set --recipe` → `trace source bind` → `seal` の公開 CLI で作成した。source binding は `key-A`→`a.txt`、`key-B`→`b.txt`、`key-page`→`page.txt`。root の6-byte contentを作る際に13-byte/10-byte の元ファイルを細切れ Seal にせず保持した。root の `trace show --ref root --format json` は2つの SourceSnapshot と、external run 0/2・untraced run 1 を示した。

各 REF を `trace compare --ref REF --max-graph-visits 100 --format json` で観測した。変更前は root と middle の External run が全て `PRESENT`、leaf は `NO_TRACE`、自身/上流/下流に差異はなかった。次に `a.txt` の先頭に `QQ` を追加すると root run 0 は `PRESENT` のまま、選択位置は旧位置3から5となった。この数字は同じ歴史上の箇所を証明しない。

その後、現在ファイルを `a.txt=abcXQZtailXQZ`、`b.txt=headUQtail` に変更した。

| 観測 REF | 自身 | 上流 | 下流 |
|---|---|---|---|
| root | run 0/2 とも `ABSENT_EXACT`、Untraced run 1 は検索対象外 | Seal 0、差異なし | 2 Seal、`middle` の差異を含む。差異あり |
| middle | B由来 run 0 が `ABSENT_EXACT` | root の差異あり | leaf は Trace なし、差異なし |
| leaf | `NO_TRACE`、差異なし | root と middle の差異あり。root は間接経路、middle は直接経路 | Seal 0、差異なし |

全 REF の graph scope は `complete=true`。root に `--estimate` を付けた別呼出しでは、2 run の `ABSENT_EXACT` を維持したまま、両方の estimate が `CANDIDATES`、方法 `single-diff-lcs-v1`、現在側の候補はそれぞれ `[3,6)` と `[4,6)` になった。候補は原意・歴史上の継続の証明ではない。短い入力のみを使い、先の[初期測定](issue-17-environment-review-2026-09-18.md)で長時間実行した入力サイズを推定へ流さなかった。

## 一致位置・版変更・復元

`trace occurrences --ref page --baseline head --run-index 0 --view both --limit 2 --format json` と返された cursor で3ページ取得した。

| ページ | entries | has_more |
|---|---|---|
| 1 | snapshot:0, snapshot:1 | true |
| 2 | snapshot:2, current:0 | true |
| 3 | current:1, current:2 | false |

初回 cursor を保持して現在の `page.txt` を `aaab` へ変更すると、続きは非ゼロ終了、成功 JSONなし、`PAGE_CONTEXT_CHANGED`。新規の snapshot 一覧は0/1/2の3件を維持し、current 一覧は0/1の2件になった。さらに元ファイルを削除すると snapshot は0/1/2のままで、current は `INCOMPLETE`・`SOURCE_READ_FAILED`。同じ数値位置でも snapshot と current の版は混同しない。

`a.txt`、`b.txt`、`page.txt` を全て削除した後に format 7 native dumpを作成した。`trace show` が参照する SourceSnapshot の全文 Blobをその dump から復元し、root の元 A/B bytes 全体がそれぞれ上記の13/10 bytesと完全一致することを確認した。SHA-256 は A=`696756241a220567430b485ea0c76388f86859743de767adff119ae41a1cd83e`、B=`72dbf7d22111051f11ea52aaa170bccac790903d769f38a80dc60514d4cc7144`。Seal run 外の `abc`、`tailXYZ`、`head`、`tail` も含む。別の空 directory へ native load した receipt は `LOADED`・format 7・Blob 25・REF 4。再 dump は元 dump と exact bytes 一致し、root の `trace show` JSON も一致した。

## status 再検討へ渡す観測

4 REF の `status --format json` は A/B の現在ファイル変更の前後で exact JSON が等しく、4 REF とも stale self/direct/transitive は false のままだった。これは Accepted の STALE 不変と整合する。`status --format human` は285 bytesで4 REFを一覧できるが、この試用では Trace の変化を示さない。各 REF の差異を追うため、`trace compare` を root/middle/leaf に個別実行した。変更後の human compare は root 2,592 bytes、leaf 2,592 bytesで、run、SealID、直接/間接経路を表示した。別 REF の `page` で source fileを削除すると、root compare の Observation にもその未読取 source の行が現れたが、root の own/upstream/downstream の完全性は維持された。

この操作負担と一覧欠落は status 表示を再検討する材料だが、利用者にとって表示を追加すべきか、どの範囲を表示するか、読取り費用を許容できるかは未判断。`STATUS_REVISIT` で扱う。試用の小入力における単発 CLI 外側時間は status JSON 4～5 ms、compare JSON 約8～9 ms、occurrences JSON 約3 msで、同一ホストの非統制な参考値のみ。性能合格値には使わない。

残る全体契約：Accepted ADR 0035 §4.3 の明示的 format 5/6→7 migration command は未実装かつ現行 PERT で担当未割当。この仮設 format 7 作成と native transport 試用は代行ではない。
