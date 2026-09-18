# Issue #17 `STATUS_REVISIT` — 詳細側試用後の status 保留再評価

- 日付：2026-09-18。
- 上位判断：[Accepted ADR 0035](../adr/0035-origin-trace-cli-and-observation-output.md) §4.2・§8 と [ADR 0038](../adr/0038-origin-trace-r2-cli-and-observation-output.md) §4 は、Trace 状態の `status` 表示を一旦保留し、詳細側の試用後に必要性が見えれば戻す。STALE、`source compare`、Cause の意味は変更しない。
- この記録の範囲：[Accepted P3](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) P8 と [正本 PERT](issue-17-origin-trace.pert) の `STATUS_REVISIT`。新しい表示・flag・schema・性能閾値を採用または実装する決定ではない。

## 観測

| 根拠 | 確認できたこと |
|---|---|
| [詳細側の仮設試用](issue-17-detail-trial-2026-09-18.md) | 4 REF の `status/v3` JSON は現在の A/B ファイル変更の前後で `cmp` が byte 一致を確認した（両方の SHA-256=`e6c5f7165b7e822bd3509070f5e1f9069e2be5b0bcccf7dc3d1e411164748ceb`）。全 REF の STALE は false のまま。一方、`trace compare` は root 自身の2 run、root 下流、middle 自身/上流、leaf 上流の完全一致なしを示した。root/middle/leaf の観測には個別の REF 選択を使った。 |
| 同試用の表示 | `status --format human` は4 REFを285 bytesで一覧。変更後の root/leaf `trace compare --format human` は各2,592 bytesで、SealID・run・直接/間接経路と理由を表示。詳細に行けば結果を追えるが、既存の status 一覧では Trace 差異のある REFを見分けられない。 |
| 同試用の境界 | 無関係な page REF の source file を削除しても root compare の Observation にその読取り失敗行が現れた。root の own/upstream/downstream 完全性は維持された。一覧で Trace を集計する案では、どの REF・sourceを読み、失敗をどこへ表示するかを決める必要がある。 |
| [初期測定](issue-17-environment-review-2026-09-18.md) | 小入力での status JSON は約4～5 ms、compare JSON は約8～9 ms。別の16 MiB合成原文で compare は中央値約279 ms、最大 RSS 約125 MiB。status 全 REF へ組み込んだ場合の時間・メモリ・I/Oではない。実利用 REF数・ファイル規模・利用頻度・許容値は未確認。 |

## 解釈と選択肢

観測から確定するのは「一覧に Trace 差異が出ない」「詳細コマンドでは追える」「複数 REF を一度に見る場合に操作数が増える」まで。利用者がその一覧を必要としているか、ファイル読取りを `status` に含める費用を許容するかは、この合成試用だけでは決まらない。位置の数値差を原意・履歴上の同じ箇所と呼ばず、`status` と STALE を混同しないことは全案に共通する。

| 経路 | 得られること | 費用・残る論点 |
|---|---|---|
| 既存保留を継続（今回の推奨） | `status/v3` を変えず、必要な REF の `trace compare` で現在の全詳細を確認できる。未定の読取り費用を日常の一覧へ持ち込まない | Trace 変化が一覧に出ず、複数 REF の追跡では選択と出力の読解が必要。必要性の判断は将来へ残る |
| 明示的に呼ぶ一覧を後続設計として提案 | 利用者が選んだときだけ複数 REF の要約を得られる可能性がある | 新たな公開操作・出力契約が必要。対象 REF、own/upstream/downstream の粒度、未読取/不完全性、source読取り、予算、測定条件を所有者が決める必要がある |
| 既定の `status` に Trace 要約を追加する後続設計 | 既存一覧から変更候補へ進みやすい可能性がある | `status/v3` と既存 human 出力の変更、ファイル読取り・失敗・時間/メモリ/I/Oの増加があり得る。実 workload と許容値、要約が詳細の代わりと誤認されない表示境界が未決 |

## 今回の処置と戻る条件

**既存の保留を継続する。** これは所有者が採用した現行境界を維持する処置であり、「status 表示は不要」とする恒久的な製品判断ではない。仮設試用により一覧の欠落は観測できたが、その欠落を埋める価値と費用の釣合いは未判定。現時点で新しい公開範囲を決める根拠はない。`status/v3` と `trace compare/v2` の実装・仕様は変更しない。

戻る最小の契機は、所有者が「複数 REF の Trace 差異を一覧から見つけたい」など具体的な利用場面を示すこと。その時は、対象 REF数・原文規模、一覧で欲しい own/upstream/downstream の単位、読取り失敗と未調査の表現、既定表示か明示操作かを確認し、必要な時間/メモリ/I/O条件を定義・測定する。新しい公開意味を採る場合は、Accepted ADR 0035/0038 の所有者管理の後継判断へ戻す。今回の `STATUS_REVISIT` 完了を新しい status 仕様の承認に読み替えない。
