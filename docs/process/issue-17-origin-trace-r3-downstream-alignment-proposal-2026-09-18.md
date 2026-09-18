# Issue #17 R3-c1 下流整合案 — ADR 0036～0039 と計画 P2

- 状態：Proposed。採用済み要件・設計・計画の改訂ではない。
- 日付：2026-09-18
- 上位要件：[R3-c1 要件差分](issue-17-origin-trace-requirement-r3-candidate-2026-09-18.md)は[承認記録](issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)により Accepted。基底は Accepted R1 + R2。本書の下流整合案は引き続き Proposed。
- 対象：Accepted [ADR 0036](../adr/0036-origin-trace-r2-presence-and-directional-observation.md)、[0037](../adr/0037-origin-trace-r2-search-and-estimation.md)、[0038](../adr/0038-origin-trace-r2-cli-and-observation-output.md)、[0039](../adr/0039-origin-trace-derived-occurrence-positions.md)、[計画 P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) の後継文書で必要な整合範囲。
- 承認済みの各 exact snapshot・Seal は履歴として保持する。本案は下流の採用判断を代行せず、実装・公開契約の根拠にしない。

## 1. 判断の出発点

所有者の新方針は、現在ファイルに P が一件でもあるかという残存判定を維持し、記録 offset は有効な初期 binding と探索ヒントとする。同じ P の全開始位置は、必要な操作の意味に応じて元版 S・現在版 F・両方から導出する。多い位置の表示・提案は既定上限とページングを持つ。元ファイル全文 S の復元は Seal 範囲外まで保証する。R3-c1 はこの限定差分を要件候補として記した。

ADR 0039 は所有者が Accepted とした後継設計判断だが、本文自体が R2 と公開契約の整合を後続条件としている。ADR 0033 の SourceSnapshot 全文 Blob と typed closure は元版復元の既存設計である。2026-09-18 時点で Issue #17 runtime は未実装であり、設計の存在を機能の提供・復元試験の完了とみなさない。

## 2. ADR ごとの後継整合

| 承認済み文書 | 維持する exact 意味 | 後継文書で明示すべき点 | 権限と順序 |
|---|---|---|---|
| ADR 0033 | `SourceSnapshot.content` は元ファイル全文 Blob。OriginMap の p・L と copy 一致を検証し、Seal typed closure が source Blob を含む | 新しい復元用 Blob は要らない。実装・dump/load で、元ファイル変更後にも S 全文の exact 復元を確認する | 保存決定の変更は現時点で不要。必要な変更が見つかった場合は別の設計判断へ戻す |
| ADR 0036 | P が F に一件でもあれば残存、全域に無ければ不存在、未完了を別状態。自身・上流・下流と Cause/STALE は独立 | §2.1 の選択一致は代表一件と明記。位置一覧は対象版の `Occ(B,P)` を別途必要時に導出し、存在判定に全件列挙を要求しない。p と選択位置の数値差を移動の証明としない | R3 採用後、ADR 0039 と整合した後継設計で決める。旧 ADR の Seal は書き換えない |
| ADR 0037 | p、p+Δ、閉区間、残りの全域という一件発見向け優先探索。`ABSENT_EXACT` の変更後範囲推定は別処理 | 位置一覧の導出は一件命中で終了する検索と別の処理契約。対象 B/P、重複一致、全件のページ列挙、途中終了・版変更を扱う。推定候補と完全一致位置一覧は異なる結果 | 一覧のアルゴリズムや効率化は要件の意味を変えずに後継設計で選ぶ。速度数値は未決 |
| ADR 0038 | `trace compare` の `presence` と `estimate` は別軸。`selected_match_start` は探索で選んだ位置、`status` への Trace 追加は保留 | 一覧・提案の起動操作、S/F の選択、ページング、既定件数上限、固定版の識別、終了・中断の出力を決める。`selected_match_start` や `position_changed` を全位置や歴史的移動として再解釈しない。既存 v2 schema の意味を暗黙に拡張しない | 公開 CLI/schema・互換性は所有者による後継設計の採否が必要。R3 候補だけでは命名しない |
| ADR 0039 | 全位置を必要時に S/F から導出、p はヒント、存在判定は一件で完了、位置提示は件数上限・ページング | Accepted R3 が確定したら、その exact 上位要件との clause 対応を記録する。本文に残る Proposed 表記は承認前 snapshot の履歴として扱う | 採用記録の条件を満たすまでは、未決の公開契約を ADR 0039 から補完しない |

ADR 0036～0038 の承認済み内容で R3 と両立する箇所は保存する。後継の修正候補を旧ファイルへ直接追記して、既存 Seal の content と承認対象を変えない。後継 ADR の数・番号と、0039 を新要件 Seal にどう接続するかは、R3 採用後に exact 上位 Cause を確認して決める。

## 3. 計画 P2 と PERT の整合案

| P2 段階 | 維持する成果 | R3 採用後に追加・変更する出口条件の候補 |
|---|---|---|
| P0 契約固定 | Accepted 要件と設計の clause 対応を固定 | R3 の採否、ADR 0039 の採用条件、後継 ADR 0036～0038 の公開境界を先に解決する |
| P1 型と保持 / P2 authoring | SourceSnapshot 全文、OriginMap、typed closure、仮設 repository での Seal | `source_start` は有効な一件として検証し、S 全文を Seal 範囲外まで保存・復元できる証拠を追加する。全位置の永続フィールドは追加しない |
| P3 原文残存比較器 | 一件命中で残存、全域確認で不存在、未完了を返す | 全位置導出を必須前提にしないことを明記。代表の選択位置を全集合と呼ばない |
| P4 自身の詳細観測 | Candidate/HEAD・F 観測の識別、run ごとの結果 | 必要な位置一覧をどの操作で得るか、元版・現在版・両方の選択とページ間の版固定を、後継公開設計が決めた後に反映する |
| P5 方向別観測 | exact Cause による自身・上流・下流の集計 | 位置一覧・ページングの未完了を presence や既存 STALE の意味へ混ぜない。各 run の出所を保つ |
| P6 変更後範囲推定 | `ABSENT_EXACT` 後にだけ推定を実行し、根拠・候補・失敗を別表示 | `Occ(B,P)` の完全一致候補と、diff 等で推測した変更後範囲を区別する。位置に基づく提案を出す場合は件数上限・ページングを適用する |
| P7 詳細側試用 / P8 全体適合 | 仮設文書での操作、transport、AC1～15 の適合証拠 | 複数・重複・大量の一致、ページ間の F 変更、元ファイル変更後の S 全文復元を AC16～21 に対応付けて確認する。status 追加は試用後の別判断 |

旧 [専用 PERT](issue-17-origin-trace.pert) は行指向の比較器と古い ENVIRONMENT_REVIEW gate を含む。2026-09-18 の `document check` は構文上 `ok`、`dag next` の推薦は空であり、現行 R2/R3 の実装順序・完成予測の根拠にしない。R3 と後継 ADR/計画が採用された後、PERT の同じ canonical ファイルの対象タスク・依存・gate を owner control と CLI mutation workflow で改訂し、再度 check、schedule、next を読む。過去の完了履歴や実績を捏造・書換えしない。

## 4. 境界と次の判断

本案は Accepted R3-c1 に対する下流の提案であり、採用済み ADR の権限ではない。R3-c1 は exact snapshot の独立レビューを経て所有者が採用した。次に公開操作/schema、既定件数上限、各操作が S/F のどちらを使うかを後継設計で決め、その設計の採用と P2 後継計画・PERT の整合を経て実装へ進む。

現在利用者が使える Issue #17 の新機能はない。本案は Accepted 仕様との衝突範囲と残る判断を可視化したものであり、実装・検証・提供・日程の確約を示さない。
