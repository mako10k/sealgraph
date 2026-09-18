# Issue #17 R4-c1 直接部分文字列 Trace 要件 — 承認記録

- 状態: Accepted requirement revision（2026-09-18、所有者による二回目レビュー判断）
- 対象: [R4-c1 要件候補](issue-17-direct-substring-requirement-r4-c1-2026-09-18.md)全文
- 対象 SHA-256: `c42f678fc965027c6c8536998499ce063d9ba82eaa7e6806044cce443b4bdb42`
- 基底: Accepted Issue #17 R1、R2-c2、R3-c1。候補本文の `Proposed` は承認前の exact snapshot の記録として保持する。
- Step 2: 所有者は [初回レビュー入力](issue-17-direct-substring-r4-first-review-input-2026-09-18.md)を提示された後、追加質問なしの `REVIEW` を選択した。
- Step 3: [独立レビュー](issue-17-direct-substring-r4-independent-review-2026-09-18.md)は `completed`。確定的矛盾なし。登録時の最小 byte offset と比較時の選択一致位置の区別について明確化案を一件記録した。

## 所有者の判断

候補は初回提示時から変更せず、独立レビューの論点とともに再提示した。所有者は `ACCEPT` と回答した。これにより、上記 exact 候補全文を Accepted Issue #17 R4-c1 要件改訂とする。

登録時に複数一致すれば最小 byte offset を `source_start` に記録する。この規則は R4-c1 の新しい登録操作に適用される。候補本文が維持するとした既存 `trace compare` の探索順と `selected_match_start` の意味は変えない。独立レビューの明確化案は採用済み候補の bytes を変更せず、後継 CLI 設計で明示する。

## 効力と次の境界

R4-c1 は、既存 Candidate の内容を指定文字列に設定する単一元ファイル・単一 External run の直接入力、見つからない場合のエラー、先頭一致位置の保存、元ファイル全文保持を新たに要求する。root/Cause、Seal の明示公開、local binding、既存 recipe、比較、位置一覧、STALE の既存契約を維持する。

公開コマンド・flag、文字列入力方法、出力 schema、help/completion、既存 `trace set` との関係は後継設計の判断である。今回の採用は要件の権限だけを確定し、ADR、実装、Seal、実 repository 移行、push、merge、release を実施または承認しない。利用者が現在使える直接指定機能はまだない。
