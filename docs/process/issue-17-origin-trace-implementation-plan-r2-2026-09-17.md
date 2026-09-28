# Issue #17 Origin Trace — R2 実装計画 P2（候補）

- 日付：2026-09-17
- 状態：Proposed。実装・公開・移行の着手記録ではない。
- 対象：Accepted [R1](issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) + [R2](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md) に対する、作成・保存・現在の原文残存確認・変更後範囲推定・方向別詳細観測。
- 設計入力：Accepted ADR 0032～0035 の R2 と矛盾しない部分、改訂候補 [0036](../adr/0036-origin-trace-r2-presence-and-directional-observation.md) → [0037](../adr/0037-origin-trace-r2-search-and-estimation.md) → [0038](../adr/0038-origin-trace-r2-cli-and-observation-output.md)。0036～0038 は未承認であり、計画はそれらを Accepted とみなさない。
- 前版：[P1](issue-17-origin-trace-implementation-plan-2026-09-17.md) は行指向・全最適比較を含む歴史候補。R2 の実装順序には本 P2 を使う。
- 関連 PERT：[旧投影](issue-17-origin-trace.pert) は line-oriented comparator と旧 ENVIRONMENT_REVIEW gate を含むため、R2 着手の推薦・予定確定には使わない。PERT の改訂は別作業。
- レビュー対象：本書全文（日本語）。

## 1. 目標・現在地・利用価値

利用者は、複数の大きな原文を分割せず、抜粋と独自表現から小さな atomic Seal を作る。Trace は Seal content の byte 範囲を immutable な原文全文と対応付ける。後から各 External run の原文が現在ファイルのどこかに完全一致で残るかを素早く調べ、完全一致がなければ変更後の範囲候補を別途推定する。詳細側で自身・上流・下流を追える。Cause Link は Seal 全体、STALE は従来定義のまま、意味論は監査しない。

2026-09-17 の読取時点では runtime は format 5/6、Issue #17 の機能コードは未実装。Accepted R2 は設計への入力であり、ADR 0036～0038 と本計画は候補。利用者が現時点で使える新機能はゼロ。P2 の寄与は、旧 P1 の行指向・全最適比較を実装根拠から外し、R2 の機能までの残る工程を示すこと。利用可能になる日は不明。費用・納期・対象最大サイズも未測定。

## 2. 開始条件と保留

| 項目 | 扱い |
|---|---|
 要件 | Accepted R1 の R7～R9/AC4/AC5 を R2 で置換し、AC12～15 を追加。他の R1 は保持 |
 設計 | ADR 0036～0038 の比較・推定・公開契約が所有者に採用されてから、その契約に依存する実装へ進む。旧 ADR の衝突節は根拠にしない |
 保存 | ADR 0033 の SourceSnapshot 全文、OriginMap byte runs、format 7、historical reader、transport を維持 |
 推定詳細 | 採用された ADR 0037/0038 に従い、実装で決定的な diff 方式・tie-break を選んで方式版を記録。数値 budget は初版の必須条件にしない |
 環境・性能 | 対象 workload/製品の数値性能閾値は未決。R2 の速度方針に従い、未計測の数値を原文確認の着手条件にしない。初期の代表入力と測定環境を記録し、実測後に必要なら所有者へ性能契約を提案する |
 status | 詳細側で試した後に必要性を判断。現時点で status 表示を追加しない |
 実データ移行・公開 | 仮設 repository の検証と区別。実行先・公開操作・リリースは本計画で実行しない |

## 3. 作業順序

| 段階 | 前提 | 完了時の具体的成果と検証 |
|---|---|---|
 P0 契約固定 | R2 Accepted、ADR 0036～0038 の採否 | 差分の採用版と clause 対応を固定。旧 PERT の不適合を明記 |
 P1 型と保持 | ADR 0033 | format 7 の typed SourceSnapshot/OriginMap/Provenance、canonical bytes/ID、旧 v5/v6 読取り、閉包の検証 |
 P2 authoring | P1 | 一 REF の Candidate content+map 同時更新、原文全文固定、Seal 作成・show、仮設 repository の明示移行。複数原文・Untraced・中間 Seal を試す |
 P3 原文残存比較器 | ADR 0036/0037 採用。P1 と並行可 | ファイル・REF 非依存の External run byte search。旧位置・サイズ差の優先と全有効開始位置の網羅、最初の一致で終了。存在/不存在/未完了を返す |
 P4 自身の詳細観測 | P2+P3、ADR 0038 採用 | local source binding、安定読取り、Candidate と HEAD の区別、JSON/human の run 結果。完全一致確認に diff/宣言/推定を起動しない |
 P5 方向別観測 | P4 | exact Cause graph の local facts を自身・上流・下流に独立集計。direct/indirect、観測範囲と不完全性、STALE 不変 |
 P6 変更後範囲推定 | P4、ADR 0037/0038 採用。P5 と順序独立 | `ABSENT_EXACT` の run のみ、決定的な一つの diff と必要な宣言から候補と根拠を提示。複数/候補なし/未完了を表示し、不存在判定を維持 |
 P7 詳細側試用 | P5+P6 | 仮設文書で作成→原文変更→自身/上流/下流の発見→推定を実操作し、status 再検討材料と実測を得る |
 P8 transport・全体適合 | P2 後に transport 可。最終は P7 後 | format 7 native dump/load、旧形式、CLI/help/completion/規範文書、AC1～15 と独立レビューの証拠を揃える |

P3 の自動比較は、原文 P が現在 F の部分列かを判定する。P6 の diff 推定を P3/P4 の前提に戻さない。P5 の方向別集計も推定の成否を差異/判定不能に混ぜない。P8 の適合証拠は公開、実 repository 移行、所有者の最終受入を自動的に成立させない。

## 4. 主な接続箇所と検証

既存コードへの接続候補は旧 P1 §3「現行コードへの接続箇所」を参照する。実装着手時に現在の branch/worktree とコードを再確認し、古い行指向 S3 を流用しない。比較器は immutable bytes を受ける独立 module、repository は安定読取り・binding・Candidate/graph、CLI は選択と表示を担当する。Git 自動検出や semantic auditor を入れない。

| 要件・場面 | 主な段階と独立した期待 |
|---|---|
 AC1～3 | P1/P2：大原文から小 Seal、複数 source と独自表現、中間 Seal の whole-Seal Cause |
 AC4/5/12/13 | P3/P4：位置移動、重複、相殺編集、末尾境界、全文不在、読取り/探索未完了 |
 AC14 | P6：推定失敗でも確認済み不在は残る |
 AC15 | P2～P6：一 Seal 内の複数 External run の結果と出所、Untraced の非検索 |
 AC6～10 | P5/P7：自身・上流・下流の独立結果、exact target、STALE 不変 |
 AC11 | P1～P8：構造不正を検出し、意味上の同義/矛盾/無意味な Cause を監査しない |

P3 では短い決定的な byte 列を網羅し、優先探索の有無判定を単純な全域部分列検索と比較する。個々の結果に、Accepted R2 のどの AC が期待を定めるかを明記する。実装が返した値を期待値の権限にしない。P4/P5 は読み取り失敗、観測競合、graph 予算切れを区別する。P6 は候補が一つ、複数、なし、未完了、宣言競合を扱い、推定と原文確認の独立を確認する。

初期測定は小さな仮設 byte 列と既存文書の複製を対象に、入力長、run 数、source 数、検索の命中位置、非一致、経過時間、peak memory を記録する。測定結果を未承認の製品性能保証や最大サイズに変換しない。旧 [ENVIRONMENT_REVIEW](issue-17-environment-review-2026-09-17.md) の byte 格子値・30秒/1 GiB 案は R2 の実行条件ではない。

コード変更を完了するときは repository AGENTS の `gofmt -w .`、`go vet ./...`、`go test ./...`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check` を実行する。Git integration を変更した場合は focused integration test も行う。今回の文書候補だけでは code suite を完了証拠としない。

## 5. PERT・時間・次の判断

旧専用 PERT の `CONTRACT_REVIEW done` は旧 ADR 0033～0035 の採用履歴として残るが、R2 改訂 ADR の承認済みを意味しない。`RANGE_COMPARATOR` の行指向/全最適記述と `ENVIRONMENT_REVIEW` の数値条件 gate は現行 R2 と整合しない。PERT は本 P2 と採用された設計に合わせて別途改訂・解析し、`dag next` をそのまま実装指示に使わない。旧ポイントを日数や実績へ変換しない。

次の所有者判断は ADR 0036～0038 の採否と、その中で明示した推定方式・公開 schema の採否。採用後、P0 で対象の実装環境と初期測定条件を記録し、P1/P3 から小さな slice で実装する。工数、納期、実測性能、配布時期は未知。
