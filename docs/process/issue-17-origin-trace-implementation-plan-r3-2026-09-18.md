# Issue #17 Origin Trace — R3 実装計画 P3（候補）

- 日付：2026-09-18
- 状態：Proposed。所有者の採用、実装着手、公開、実データ移行を意味しない。
- 上位契約：Accepted [R1](issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) + [R2](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md) + [R3-c1](issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)。Accepted ADR 0033～0040 の該当節。特に [ADR 0039](../adr/0039-origin-trace-derived-occurrence-positions.md) と [ADR 0040](../adr/0040-origin-trace-occurrence-listing-and-paging.md)。承認記録が旧本文の Proposed 表記より新しい採用状態を示す。
- 前版：[P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) は承認・Seal 済みの exact 履歴として維持する。本候補が採用された場合、R3 以後の作業順序は本書を使う。
- 関連 PERT：[Issue #17 専用 PERT](issue-17-origin-trace.pert) は作業順序の投影であり、要件・ADR・本書の採用を代行しない。
- レビュー対象：本書全文（日本語）。

## 1. 成果と境界

利用者は大きな原文ファイルを分割せずに小さな atomic Seal を作り、Seal content の External run を元ファイル全文の immutable SourceSnapshot と対応付けられる。後で run の exact bytes P が現在ファイル F のどこかに残るかを一件命中で確認し、完全一致が無い場合だけ変更後範囲を別処理で推定する。必要な場合は元版 S・現在版 F・両方の完全一致開始位置を、版を区別してページ取得できる。Seal 後に元ファイルが変化・削除されても S 全文を Seal 範囲外まで exact bytes で復元する。詳細側は自身・上流・下流を独立して観測する。

Cause Link は Seal 全体、STALE は従来定義のまま。byte 一致や位置の数値差は原意・履歴上の同じ箇所を証明しない。意味論は監査しない。`trace occurrences` は一致位置一覧であり、`trace compare` の presence、`--estimate` の推定、方向別状態、status 表示を変更しない。

2026-09-18 の確認時点で runtime は format 5/6、Issue #17 の機能は未実装。したがって利用者が現在使える新機能はゼロ。本書の寄与は Accepted R3・ADR 0040 に対する残作業と検証境界を明示すること。利用可能になる時期、工数、実 workload、性能閾値は不明。

## 2. 実装の前提と未決事項

| 項目 | 扱い |
|---|---|
| 要件と設計 | R1/R2/R3 と ADR 0033～0040 の Accepted exact snapshot を上位とし、各実装判断を該当 clause に戻して確認する。旧 P2 の R2 部分で両立する内容は維持する |
| 保存 | ADR 0033 の format 7、SourceSnapshot 全文 Blob、OriginMap byte runs、typed closure を使う。`source_start` は Seal 時点に成立した一件の binding・探索ヒント。全一致位置の永続フィールドや復元用の別 Blob を追加しない |
| 性能 | R2 の一件発見優先探索を維持する。数値 workload・platform・合格閾値は未決なので着手 gate にしない。代表入力で時間・メモリ・I/O を測り、必要なら別途所有者判断へ戻す |
| status と提案 | status 追加は詳細側の試用後に再検討する。位置に基づく別の提案操作は ADR 0040 の対象外であり、本計画では公開しない。既存 `--estimate` を一覧へ読み替えない |
| 実データと公開 | 仮設 repository での検証と実データ移行・配布・release は別。後者の対象と承認は本書で確定しない |

## 3. 段階・依存・出口

| 段階 | 前提 | 完了時に確認する成果 |
|---|---|---|
| P0 契約整合 | Accepted R3・ADR 0040 | R1/R2/R3、ADR 0033～0040 の clause 対応、旧 P2 との差分、status と提案操作の境界を固定する。本 P3 の採否を記録する |
| P1 型・保持 | P0、ADR 0033/0039 | format 7 の canonical SourceSnapshot/OriginMap/Provenance、SourceSnapshot 全文 Blob と typed closure、`source_start` の有効な一件・copy 一致、旧 format 5/6 reader を検証する |
| P2 authoring・復元 | P1 | 一 REF の Candidate content+map 同時更新、原文全文の immutable 保存、Seal 作成・表示、複数 source・Untraced・中間 Seal を仮設 repository で試す。元ファイルを変更・削除後、Seal 範囲外を含む S 全文を exact 復元する |
| P3 原文残存比較器 | P1、ADR 0036/0037。P2 と並行可 | ファイル・REF 非依存の byte 部分列検索を実装する。p、p+Δ、閉区間、残余の順で全有効開始位置を網羅し、一件で残存、全域不在で `ABSENT_EXACT`、未完了は別結果にする。全位置列挙を起動しない |
| P4 自身の詳細観測 | P2+P3、ADR 0038 | local source binding、安定読取り、Candidate/HEAD の別基準、各 run の presence と選択した代表一件、JSON/human 出力を接続する。diff・推定・位置一覧を presence の前提にしない |
| P5 方向別観測 | P4 | exact Cause graph の local facts を自身・上流・下流に独立集計。direct/indirect、観測範囲・不完全性、STALE 不変を確認する |
| P6 変更後範囲推定 | P4、ADR 0037/0038。P5 と順序独立 | `ABSENT_EXACT` の run のみ決定的 diff 等から変更後範囲候補・根拠・失敗を提示する。完全一致位置の一覧と混ぜず、不存在結果を保持する |
| P7 一致位置一覧 | P2、ADR 0039/0040。P3～P6 と独立に作成可 | 読取専用 `trace occurrences` の排他的 baseline 選択、0 始まり run index、`snapshot|current|both`、重複を含む位置昇順、既定 100 件、`limit`・cursor、版束縛、`sealgraph/trace-occurrences/v1` の JSON/human と未完了・失効を検証する。全件を保存せず、presence を呼び替えない |
| P8 詳細側試用 | P5+P6+P7 | 仮設文書で作成→原文変更→自身/上流/下流→推定と位置一覧を実操作する。大量一致、ページ継続と版変更、S 全文復元を含む。status 再検討材料、初期時間・メモリ・I/O の実測条件と結果を記録する |
| P9 transport・全体適合 | P2 後に transport を並行可。最終は P8 後 | format 7 native dump/load で S 全文と typed closure の復元を確認する。旧形式、CLI/help/completion、規範文書、AC1～AC21 の対応、独立レビューと repository 必須検証を揃える。公開・実 repository 移行・所有者の最終受入は別判断 |

P7 は P3 の一件命中探索を全件列挙へ変えない。P5 の方向別集計は P6 の推定成否や P7 のページ継続状態を binding 差異、STALE、上流・下流状態に混ぜない。P8 の実測は未承認の数値保証にならない。

## 4. 要件・受入場面との対応

| 上位受入 | 段階と独立した期待 |
|---|---|
| AC1～3 | P1/P2：小さな Seal、複数の元ファイル・独自表現、中間 Seal の whole-Seal Cause |
| AC4/5/12/13 | P3/P4：一件発見、重複・位置移動・相殺編集・末尾境界・全域不在・未完了。選択位置を唯一の由来と呼ばない |
| AC6～10 | P4/P5/P8：自身・上流・下流の独立観測、exact target、STALE 不変 |
| AC11 | P1～P9：構造不正を検出し、同義・矛盾・無意味な Cause の意味監査をしない |
| AC14/15 | P2～P6：不在後の推定失敗でも不在結果は維持し、複数 External run の出所と Untraced の非検索を確認 |
| AC16/17 | P1/P3/P7：p は有効な一件、最初の命中は存在判定だけ。`B="aaaa", P="aa"` の 0/1/2 を重複込みで導出し、全位置を保存しない |
| AC18/19 | P7/P8：有限の既定ページ、続きの有無、両 view の順序、F/binding/baseline 変更時の cursor 失効。ページを全集合と誤表示しない |
| AC20 | P1/P2/P7/P9：元ファイル変更・削除後に S 全文を Seal 範囲外まで復元し、元版位置を導出。native transport 後も exact bytes を検証 |
| AC21 | P7/P8：S/F の位置と版を区別し、同じ数値 offset から履歴上の対応を断定しない |

比較器は短い決定的 byte 入力の oracle と照合する。`B="aaaa",P="aa"`、`|B|<|P|`、最後の有効開始位置、相殺編集、検索途中終了を含める。cursor の検証では選択 baseline と非選択 baseline の変更を分け、現在 F の再読取り失敗・不安定・途中終了を構造不正と区別する。期待値は Accepted 要件と ADR の該当節に結び、実装結果を仕様の根拠にしない。

コード変更の完了時は repository AGENTS の `gofmt -w .`、`go vet ./...`、`go test ./...`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check` を実行する。Git integration 変更時は temporary repository による focused test も行う。文書候補の作成を code suite の完了証拠としない。

## 5. PERT・費用・次の判断

旧 PERT の `CONTRACT_REVIEW done` は ADR 0033～0035 の過去の採用履歴であり、R3 や本 P3 の承認履歴に読み替えない。行指向 `RANGE_COMPARATOR` を byte 存在比較器へ改め、性能数値未定を format 7 着手の gate にしない。一致位置一覧と推定を別タスクとし、trial の前に双方の検証を置く。既存の三点ポイントは未計測の仮説であり、日数・実績・納期に換算しない。新タスクの見積りも実測後に再評価する。

次の所有者判断は本 P3 全文の採否。採用後に実装の最初の slice を選び、現在のコードと計画の fit を再確認する。P9 の全体適合証拠は公開・移行・最終受入の代行ではない。所要時間、release、利用可能日はいずれも未知。
