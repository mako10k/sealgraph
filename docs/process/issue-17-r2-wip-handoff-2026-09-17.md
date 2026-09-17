# Issue #17 R2 WIP 引継ぎ — 2026-09-17

- 保存対象ブランチ：`codex/issue-17-origin-trace-r2-wip`。2026-09-17 の `main` 基点は `c23aa34cea273d7aa2beb94b9c9d69a924560fe6`。
- 対象：Issue #17 の R1/R2 要件、承認・レビュー、ADR 0032～0038、実装計画 P1/P2、専用 PERT、SealGraph の対応する objects/REFs、案内文書。別 worktree の Candidate 比較作業は含めない。
- 現在地：R2 要件と ADR 0036～0038・実装計画 P2 は Accepted。R2 と４文書の計５件を Seal 済み。exact ID と readback は[Seal 実行記録](issue-17-r2-downstream-seal-2026-09-17.md)。`sealgraph fsck` は ok、`stale --scan` は対象なし。
- 利用者向け Issue #17 機能：未実装。runtime は format 5/6。format 7 移行、公開 CLI、配布、実データ移行は未実施。
- 旧 [Issue #17 PERT](issue-17-origin-trace.pert) は行指向・全最適比較と旧環境 gate を含み、Accepted R2 と不整合。次作業の選択や開始判断にそのまま使用しない。
- 再開点：このブランチとリモート SHA、作業ツリー、勤務状態を読み戻す。Accepted [P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) と現在の R2 設計に対して、PERT と ENVIRONMENT_REVIEW の適合を再確認する。実装の次段階を選ぶ前に、対象タスクの現時点の価値・前提・未決を確認する。commit/push の成功だけで実装許可や完成としない。
- 未決：対象 workload と数値性能条件、推定の実測費用、status 表示の必要性。status は詳細側試用後に戻る判断である。旧 Accepted ADR 0033 等の全てを今回の５ Seal の Cause graph が網羅するわけではない。

この引継ぎは保存時点の範囲と再開点を示す。commit SHA とリモート照合結果は Git 履歴と最終報告で確認する。
