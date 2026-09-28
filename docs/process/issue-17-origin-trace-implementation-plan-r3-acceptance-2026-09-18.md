# Issue #17 実装計画 P3 と PERT DAG 改訂の承認記録 — 2026-09-18

- 状態：Accepted plan。Issue #17 の機能実装、Seal、公開、実データ移行、release の完了または着手記録ではない。
- 決定者：ユーザー（Operator）。
- 対象計画：[P3 日本語全文](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md)、SHA-256 `cff3db3420a8e9c5abd07455e14f4b7148b74dcef22d50dce0521633a44c82fd`。
- 対象 PERT 改訂：[レビュー資料](issue-17-origin-trace-pert-r3-review-2026-09-18.md)と[exact batch](issue-17-origin-trace-pert-r3-batch-2026-09-18.json)、batch SHA-256 `9eecdeb2143520f1cbb0e2a641a073a6e2d83bde2c26b3877693e5c1ca803aa5`。
- 上位権限：Accepted R1/R2/R3、ADR 0033～0040 の該当節。旧 P2 は Accepted の immutable な履歴として保持する。

## 所有者の決定

P3 の目的、旧 P2 からの変更、費用・時期の未知、PERT の全変更範囲と `perttool` の DAG owner control を提示した。所有者への質問は、① P3 の全文を Accepted とすること、② 提示した exact batch の DAG 改訂を確認して canonical PERT に適用することを別々に名指しした。所有者は「承認します。」と回答し、両方を承認した。

候補 P3 本文の `Proposed` は承認前の exact snapshot の履歴として保持し、現在の採用状態は本記録で示す。計画と PERT の採用は、個々の実装タスクの開始、数値性能契約、実データ移行、配布、最終受入を代行しない。

## PERT の適用と読戻し

承認後、canonical [Issue #17 PERT](issue-17-origin-trace.pert) の元 digest `sha256:b0d0f761148ab2269b857b17100930e5491eaa81b21936100003b048a57b1339` に対し、`perttool batch apply` を `--actor codex --accepted-by-owner user --write --expect-digest` で一回適用した。CLI は `affected_scopes=["dag"]`、owner `user`、`write_authorized=true`、`written=true` と報告した。読戻しの digest は `sha256:20a727a68168d1e7fc3018dd59bbec37adc271192bf771d866fd46ce35ad5ebe` で、承認前プレビューの候補 bytes と一致した。

`PLAN_REVIEW` は batch 中で「P3 採用待ち」の blocked task として作成された。上記所有者決定を本記録へ記した後、`perttool task finish` によって 2026-09-18 12:06:04 JST の記録完了イベント `PLAN_REVIEW_ACCEPTED_20260918` を追加した。作業時間・effort の実績値は観測していないため記録しない。`PLAN_ACCEPTED` と `READY` は依存関係の到達事実に合わせ `reached` とした。旧 `CONTRACT_REVIEW done` は保持された。

最終読戻し digest は `sha256:da6731ed229b00869b27b12989d56639c7216e08a8b9e9ee964ae425e954283f`。`document check` は成功し、19 milestones、15 tasks、9 gates、errors/warnings 0。`dag analyze --schedule both` は診断なしで成功し、precedence 30.667 point、resource 58.167 point を返した。`dag next` は `FORMAT_TYPES` を次候補とした。ポイントは未計測の相対規模仮説で、日数・実績・納期ではない。次候補の着手前には現行 code と要件・設計の fit を再確認する。

## 残る境界

Issue #17 runtime は依然として未実装であり、利用者に提供された新機能はゼロ。P3 は Accepted 契約に従う実装順序を与えるが、コード・適合検証・公開・移行は残る。初期測定で workload と platform の観測値を得ても、所有者未決の数値性能閾値を自動的に確定しない。`trace occurrences` の位置一覧、`trace compare` の原文残存と範囲推定、Cause/STALE/status の境界を維持する。
