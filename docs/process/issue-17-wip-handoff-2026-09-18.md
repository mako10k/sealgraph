# Issue #17 WIP 引継ぎ — 2026-09-18

## 保存対象と現在地

- 継続ブランチ: `codex/issue-17-origin-trace-r2-wip`。作業 checkout は `/home/katsumata-m/sealgraph`。別 worktree の `codex/candidate-explicit-compare` は対象外。
- 基点: `main` と `origin/main` は確認時 `c23aa34cea273d7aa2beb94b9c9d69a924560fe6`。push 前の `origin/codex/issue-17-origin-trace-r2-wip` は `e5f167b7e74bb4a19e37744b4ad85405495dca4a`。本書に新しい HEAD は固定せず、再開時にリモートと照合する。
- Accepted Issue #17 R1/R2/R3 に対する format 7 実装、明示 migration、native transport、詳細 CLI、規範文書の内部検証は [FULL_VERIFICATION](issue-17-full-verification-2026-09-18.md) に記録。専用 [PERT](issue-17-origin-trace.pert) の P3 内部タスクと milestone は完了扱い。これは実 repository 移行、release、公開・最終受入の証拠ではない。
- `make install PREFIX="$HOME/.local"` によりローカルへ `sealgraph` 本体と Bash 補完を導入。インストール先とビルド元の byte 一致、ヘルプと補完を確認。通常ビルドの表示は `0.1.0-dev`。次のリリース番号は未採用で、公開済み最新タグは `v0.1.0-beta.6`。
- [R4-c1 直接部分文字列 Trace 要件](issue-17-direct-substring-r4-acceptance-2026-09-18.md)は所有者が Accepted。候補 exact bytes はレビュー後も不変。指定文字列を Candidate 内容全体とし、元ファイルの最小 byte offset の一致を登録する。見つからなければエラー。既存の `trace compare` の一致選択は変更しない。R4-c1 の CLI 設計・実装・検証は未着手。

## 再開位置

1. `worktimectl agent` と `$repository-start` で共有勤務状態、checkout、作業ツリー、`origin/main`、継続ブランチのリモート先端を新たに確認する。push の結果は本書ではなくリモート ref で確認する。
2. R4-c1 の後継 CLI 設計候補を作る前に、Accepted [要件候補本文](issue-17-direct-substring-requirement-r4-c1-2026-09-18.md)、[独立レビュー](issue-17-direct-substring-r4-independent-review-2026-09-18.md)、ADR 0035/0039/0040/0041 と現在の help/実装を照合する。登録時 `source_start` の最小 offset と、比較時 `selected_match_start` の探索順を明確に分ける。公開コマンド・入力方法・出力 schema・help/completion を設計判断にする。新しい PERT 担当 task はまだない。
3. R4-c1 の実装は後継設計と作業範囲が承認されてから開始する。既存 R1～R3 の内部検証を R4-c1 の実装証拠に流用しない。
4. Issue #17 全体の merge とリリース番号・公開は別判断。現ブランチは WIP で、PR・merge・新タグ・GitHub Release・実 repository 移行は行っていない。`beta.7` は候補名にすぎず、ローカルの `0.1.0-dev` を公開版と呼ばない。

## 価値・未解決・同期境界

- R1～R3 の利用価値: ローカルの開発 checkout とインストール先で format 7 の機能を試せる。一般利用者への配布・アクセスは未確認。
- R4-c1 の利用価値: 要件のみ確定。直接指定コマンドは現時点で存在せず、利用者向け実現価値は 0。後継設計→実装・検証→提供判断が残る。
- 不確実性: 実利用 workload・性能閾値、公開版の番号と範囲、実 repository 移行対象、PR/merge 時期は owner の別判断を要する。既存の合成測定値を性能保証にしない。
- 本 handoff と WIP commit/push は継続位置の同期に限る。別 worktree、`main`、タグ、release、GitHub Issue、実データは変更しない。
