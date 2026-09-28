# Issue #17 P7 `OCCURRENCES_LIST` 実行記録 — 2026-09-18

- 根拠：Accepted Issue #17 R3-c1、[ADR 0040 の承認記録](issue-17-adr-0040-acceptance-2026-09-18.md)。本記録は要件や ADR の改訂ではない。
- 範囲：一つの External run について、保存済み SourceSnapshot、安定して読んだ current file、または両方の完全一致開始位置を読取専用で列挙する `trace occurrences`。
- 実装：重複する byte 一致を昇順に列挙する有限ページャー、明示的な Candidate／HEAD／Seal 選択、既定 100 件、版に束縛した cursor、`sealgraph/trace-occurrences/v1` JSON と人間向け出力を追加した。`trace compare` の一件探索は一覧処理を呼ばず、canonical object と REF には書き戻さない。
- 検証：重複一致、`both` のページ境界、最終ページ、既定上限、元ファイル削除後の snapshot、current 読取失敗、cursor の改変と対象版変更、選択 baseline 変更、run 指定、処理取消しをテストした。完了時の全体検証結果は下記に追記する。
- 残る制約：同期的なページ内走査に回復可能な検索中断を発生させる条件は定義されておらず、`INCOMPLETE / SEARCH_INTERRUPTED` を実行時に生成する経路はない。処理取消し時には成功ページを出さない。対象 workload での時間・I/O と既定 100 件の使いやすさは未計測であり、性能保証はしない。
- 適用範囲：このローカル実装と検証の記録である。push、release、利用者環境への提供は含まない。

## 完了時の全体検証

- `gofmt -w .`、`go vet ./...`、`go test ./...`：成功。
- `npm ci`、`npm run clone-check`：成功。既存の v5/v6 間の重複を 1 件報告したが、終了値は 0。
- `make complexity-check`、`make deadcode-check`：成功。
