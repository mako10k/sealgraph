# Issue #17 R4-c1 直接部分文字列 Trace 検証記録 — 2026-09-28

- 状態: Accepted ADR 0042 に対する内部検証成果。対象実装 commit は `e5fde5eed897b0687e36183d7bcbfe13f21f2eb8`。
- 実装 Seal: `0fcfea54d006b2ce98b4756d5035b87dee5247d579618fb70c082ebf7e37be93`
- 上位要件 Seal: `82798e71409cc92712837b02d9f994ee2b6a3b1dba79ecaf60880e2140a55625`

format-7 の一時 repository で、インライン `--content`、ファイル `--content-file PATH`、標準入力 `--content-file -` を実行した。先頭 byte offset、重複・重なりのある一致、非 UTF-8 bytes を含む元ファイル、単一 External run、元ファイル削除後の全文復元、成功 receipt と明示 Seal 前の REF HEAD 不変を検証した。直接入力と同じ入力の recipe 経路が同一 Candidate を作ることも確認した。

不一致、空・不正 UTF-8 の指定内容、両 content flag の同時指定、recipe との混在、元ファイル読取失敗、Candidate 不在は失敗した。既存 Candidate は失敗前後で同一だった。Candidate 競合の検出と原子的公開は既存 `TraceSet` の repository テストと共通経路で検証した。

最終コード差分に対して `gofmt -w .`、`go vet ./...`、`go test ./...`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check` が通過した。独立レビューでは一旦元ファイル全体の UTF-8 要求が指摘されたが、Accepted R4-c1 の対象は指定内容の UTF-8 bytes であることを再照合し、指摘は撤回された。修正後 ADR 0042 の CLI 境界にも確定的な矛盾は見つからなかった。

実 repository は format 5 のままで、直接入力操作はここでは実行していない。ADR 0042 の採用判断と設計 Seal を確認した。本記録は実 repository 移行や利用者向けリリースの受入を主張しない。
