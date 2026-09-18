# Issue #17 — `NORMATIVE_SYNC` 実行記録

- 実行日: 2026-09-18（JST）
- 根拠: Accepted Issue #17 R1/R2/R3 要件、Accepted ADR 0033・0036〜0041、[P3 計画](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) P9、[PERT](issue-17-origin-trace.pert) `NORMATIVE_SYNC`
- 範囲: format 7 の規範文書、公開 CLI の記述、help/completion と実装の整合。実 repository の移行、配布、最終受入は含まない。

## 権威と変更

R1/R2/R3 の Accepted snapshot が AC1〜AC21 の根拠である。ADR 0033 は保存形式、0036〜0038 は比較と観測の後継判断、0039/0040 は導出位置とページング、0041 は Trace 変更の receipt を定める。ADR 0032/0034/0035 のうち後継判断に置換されていない条項は引き続き適用される。この関係を [requirements](../requirements.md)、[architecture](../architecture.md)、[storage-format](../storage-format.md)、[CLI](../cli.md) に反映した。

文書は、whole-Seal Cause と既存 STALE、全文 SourceSnapshot、OriginMap の run、完全一致の一件探索と任意の範囲推定、独立した重複込み全開始位置のページング、own/upstream/downstream の詳細観測、format 5/6→7 の明示移行、native transport を区別して記述した。`status/v3` への Trace 追加は現行保留のままである。[README](../../README.md) の runtime、migration、公開 command の記述も format 7 に合わせた。

CLI help の trace command 登録は既に現行契約と一致していた。completion は `trace source show/rebind/unbind` の候補を REF でなく source key から取得するよう修正し、改行を含む key は行単位 completion から除外した。`source` と `trace source` の `--from` はファイル補完、`migrate repository --from` は形式 5/6 の補完とした。[文書索引](../index.md)も ADR 0039/0040 の古い未決・未実装記述を更新し、ADR 0041 を追加した。

整合確認中、format 7 の inspection JSON が `fsck/v4` 以外は旧 schema のままだと判明した。`show`、Candidate inspection、`compare`、`graph`、`impact`、`log`、`linklog` を format 7 で v4 に dispatch し、nullable `origin_map_id` と `origin` の比較欄を追加した。format 5/6 の v2/v3、`status/v3`、STALE の契約は維持した。また新規 repository の推奨 `.gitignore` に `/local/` を加え、local Trace binding/correspondence が Git sidecar に入らないようにした。既存 `.gitignore` の再初期化時上書きは行わない。

## 検証

- 対象文書の相対リンクに欠落はなかった。
- completion（`--from` のコマンド別候補を含む）、inspection v4、`.gitignore` 方針の focused test が成功した。
- `gofmt -w .`、`go vet ./...`、`go test ./...`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check`、`make completion-check` が成功した。clone-check は重複率 0.09% で通過した。`--from` 修正後に全て再実行した。
- 独立した文書レビューで ADR の後継範囲と `/local/` 記述の不整合を指摘され、修正後の再レビューで解消を確認した。別の独立レビューで `--from` 補完と文書索引の漏れを指摘され、上記の修正と回帰テストを加えた。

この記録は文書と公開操作の同期の証拠であり、AC1〜AC21 の全体適合や利用者への配布を確定しない。次の `FULL_VERIFICATION` が全体対応表と独立レビューを閉じる。
