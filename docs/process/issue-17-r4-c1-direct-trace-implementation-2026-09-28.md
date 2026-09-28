# Issue #17 R4-c1 直接部分文字列 Trace 実装記録 — 2026-09-28

- 状態: draft 実装成果。ADR 0042 の採用判断は別。
- 対象 commit: `e5fde5eed897b0687e36183d7bcbfe13f21f2eb8`
- 上位要件: Accepted R4-c1 Seal `82798e71409cc92712837b02d9f994ee2b6a3b1dba79ecaf60880e2140a55625`
- 設計候補: ADR 0042 draft Seal `bfbe9ce2c2b4dabd1f5985b888b670bca584e1736408e78b56190a451d935ee7`

`trace set REF` に、既存 `--recipe PATH [--content-file PATH|-]` と排他的な `--source-file PATH --source-key KEY (--content STRING | --content-file PATH|-)` を追加した。直接入力は非空 UTF-8 の指定内容を元ファイル全文の bytes から探索し、最小 byte offset の単一 External run を作る。元ファイル全体には UTF-8 を要求しない。

Candidate 更新、元ファイル全文の SourceSnapshot、保存前告知と `trace-mutation/v2` receipt は既存 `TraceSet` を使用する。直接入力時の不一致・入力不備は Candidate を変更せず、Seal と local binding も自動作成しない。`docs/cli.md`、CLI help、path completion、CLI テストを同じ commit で更新した。保存 schema と repository API は変更していない。

実装対象は `internal/cli/trace.go`、`internal/cli/help.go`、`internal/cli/completion.go`、`docs/cli.md`、`internal/cli/trace_test.go`。実 repository は format 5 のままで、format 7 の操作は一時 repository のテストで検証する。
