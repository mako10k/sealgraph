# Issue #17 `NATIVE_TRANSPORT` 実行記録 — 2026-09-18

- 上位契約：Accepted Issue #17 R3-c1 AC20、[Accepted ADR 0033～0035](adr-0033-0035-origin-trace-acceptance-2026-09-17.md)、[Accepted P3](issue-17-origin-trace-implementation-plan-r3-acceptance-2026-09-18.md)。本記録は上位契約を変更しない。
- 対象：format 7 の全 retained Blob（opaque orphan を含む）、exact REF manifest/tag、Candidate を `sealgraph/native-snapshot/v1` で運ぶ `dump --format native-blobs-v1` と、named file から absent target へ公開する `load --format native-blobs-v1`。
- 着手判断：ADR 0033 A5 の実装前性能条件は、後続の Accepted R2/ADR 0036 と所有者が採用した P3 の「未測定の数値を着手条件にしない」実行方針で更新されたものとして扱った。snapshot の保存・transport 形式は変えず、実測と必要な閾値判断は `ENVIRONMENT_REVIEW` に残す。この判断は command-line LLMThink audit で fatal/error/warning 0 を確認した。
- 出力境界：dump は読取専用で、成功まで snapshot bytes を stdout に出さない。load は exact input と staged closure を検証し、named file の同じ bytes を公開直前に再読取してから no-replace publish する。公開後は repository を再読取し、件数を確認して `sealgraph/native-load/v1` を返す。失敗後の自動再試行・削除は行わない。
- 元版：元ファイルを削除してから dump/load しても、Seal 範囲外の byte を含む SourceSnapshot 全文と exact ID を復元する。
- 局所性：local source binding、correspondence、cache、logs、lock は snapshot に含めず、legacy `universal-blob-v1` load は維持する。
- 提供：ローカル実装とテストの記録であり、push、release、実データ移行、利用者環境への提供は含まない。
- 後続の計画照合：Accepted ADR 0035 §4.3 には format 5/6→7 の明示 migration command もあるが、現行 `NATIVE_TRANSPORT` の記述は native dump/load と S 全文復元を対象とし、その migration command の実装は含めていない。現行コードにも該当 command はない。Issue #17 全体適合の前に、正本 PERT のどの後続 task がこれを担うか確認する。

## 検証

- `gofmt -w .`、`go vet ./...`、`go test ./...`：成功。
- `npm ci`、`npm run clone-check`：成功。既存 v5/v6 間と native load の既存 mode verifier との類似を各 1 件報告したが、終了値は 0。
- `make complexity-check`、`make deadcode-check`：成功。
- codec/repository と CLI を別々に読取専用レビューし、修正必須の指摘なし。外部プロセスが writer guard を迂回して同時変更する場合の観測や公開直前の極小競合窓を、実測による保証とは扱わない。
