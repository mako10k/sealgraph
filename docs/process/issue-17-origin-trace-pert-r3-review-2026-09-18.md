# Issue #17 PERT R3 改訂プレビュー（所有者確認用）

- 日付：2026-09-18
- 状態：Proposed。canonical [Issue #17 PERT](issue-17-origin-trace.pert) の DAG は未変更。
- 対象：Accepted R1/R2/R3・ADR 0033～0040 に対する [後継計画 P3 候補](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) の作業投影。
- exact 変更要求：[30 件の perttool atomic batch](issue-17-origin-trace-pert-r3-batch-2026-09-18.json)。人が読む変更範囲は下記の全項目。JSON は CLI への機械入力であり、製品要件ではない。
- `perttool batch apply` のプレビュー：元 9,215 bytes → 候補 11,336 bytes、48 text edits。対象範囲 `dag`、必要な owner は `user`。元文書の確認時刻は 2026-09-18 11:55 JST。digest は CLI 出力に保持する。

## 現状と変更理由

現 canonical PERT は旧 P1 の投影で、行指向の `RANGE_COMPARATOR` と数値性能条件待ちの `ENVIRONMENT_REVIEW`→`READY` gate を持つ。構文検査は成功するが、`dag next` は着手候補を返さない。これは Accepted R2/R3 の byte 存在判定・必要時の位置導出と異なり、P2/P3 の順序を表していない。

旧 `CONTRACT_REVIEW done` は ADR 0033～0035 の過去の採用履歴として維持する。R3、ADR 0039/0040、P3 の採用をその done へ読み替えない。実績イベントのない状態も保持し、見積りを実績や納期へ換算しない。

## 変更候補の全範囲（日本語）

| 対象 | 変更する意味 |
|---|---|
| project、START、READY、ENVIRONMENT、COMPARATOR、TRIAL、TRANSPORT、DOCS、FINISH | 表題・説明を R3/P3 の作業投影へ直す。計画の採用前は実装候補を選ばないと明記する。マイルストーンの state は変更しない |
| `ENVIRONMENT_REVIEW` | `blocked` を `planned` とし、未定の数値閾値確定ではなく初期測定条件・時間/メモリ/I/O の記録に変更する。旧三点見積りは仮説として残す |
| `READY_ENV` | `ENVIRONMENT→READY` を `ENVIRONMENT→TRIAL` へ変更する。測定記録は試用には必要だが、format 7 の着手条件にはしない |
| `RANGE_COMPARATOR` | ID と既存三点見積りを保持し、内容を External run の byte 残存比較器へ変更する。p、p+Δ、閉範囲、残りの全有効開始位置を探索し、一件で残存、全域不在で `ABSENT_EXACT`、未完了を別にする。行指向・全最適対応・全件位置列挙を除く。見積りは実装前に再評価する |
| `FORMAT_TYPES`、`TRACE_AUTHOR` | SourceSnapshot 全文 Blob、OriginMap の有効な一件、typed closure、旧 reader、Seal 範囲外まで S 全文を復元する検証を明記する。既存依存と三点見積りは維持する |
| `SELF_COMPARE`、`DIRECTION_COMPARE` | 自身の presence と選択一件、方向別 local facts を分離し、位置一覧・推定の結果を混ぜない。既存依存と三点見積りを維持する |
| `DETAIL_TRIAL`、`STATUS_REVISIT` | 仮設文書で推定、位置ページ、版変更、元版復元を試す。status は試用後の再判断だけで、このタスクに実装を含めない |
| `NATIVE_TRANSPORT`、`NORMATIVE_SYNC`、`FULL_VERIFICATION` | native dump/load 後の元版全文、独立した occurrences v1 と compare v2、AC1～AC21、必須の code validation を出口に追加する。公開・実データ移行・release は含めない |
| 新 `PLAN_ACCEPTED`、`PLAN_REVIEW`、`READY_PLAN` | P3 exact 計画の採否を owner が記録する blocked task と、採用後に `READY` へ進む gate を追加する。旧 `READY_CONTRACT` は残し、過去の契約採用履歴を保持する |
| 新 `ESTIMATE`、`ESTIMATE_COMPARE`、`JOIN_ESTIMATE` | 自身の観測後、`ABSENT_EXACT` の変更後範囲推定を独立実装し、試用前に検証する |
| 新 `OCCURRENCES`、`OCCURRENCES_LIST`、`JOIN_OCCURRENCES` | authoring 後、`trace occurrences` の元版/現在版/両方、重複、既定 100 件、cursor の版束縛、v1 schema と未完了を独立実装し、試用前に検証する |

既存の完了タスクは削除・再開しない。新タスクの三点ポイントは仮設の相対規模見積りで、`PLAN_REVIEW=1/1/2p`、`ESTIMATE_COMPARE=3/5/10p`、`OCCURRENCES_LIST=3/5/10p` とする。測定後に別の preview と owner control で再評価する。

## CLI プレビューと判断境界

元ファイルは `document check` が成功（16 milestones、12 tasks、6 gates）。batch の候補は元ファイルへ書かずに `perttool` が生成した。候補を一時ファイルに投影して `document check` は成功（19 milestones、15 tasks、9 gates、診断 0）。`dag analyze --schedule both` は成功したが、`PLAN_REVIEW` が blocked の時点で解決した場合に限るという resource schedule 警告 `PTRES-303` を返した。`dag next` は `ENVIRONMENT_REVIEW` だけを現在の着手候補とし、`PLAN_REVIEW` は blocked、実装タスクは upcoming とした。この推薦は初期測定の計画状態であり、P3 の採用や実装の承認を示さない。

`perttool` は今回の DAG 変更について、actor `codex` による assertion-free preview に対し `required_owner_confirmations=["user"]`、`write_authorized=false` を返した。よって所有者が **この exact batch の DAG 改訂**を確認するまで canonical PERT へ書き込まない。P3 計画本文の採否も別の判断であり、batch の承認だけでは P3 を Accepted としない。

採用の選択肢は、① P3 とこの batch をそれぞれ採用して PERT に適用、② P3 または PERT 案を修正して再プレビュー、③ 現行 PERT を保持。①なら digest guard を付けた `perttool batch apply --write` を一回行い、`document check`・schedule・next・Git diff で読み戻す。候補と元ファイルが変わった場合は同じ owner assertion を使い回さず、新しいプレビューと判断へ戻す。
