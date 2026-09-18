# Issue #17 R4-c1 直接部分文字列 Trace 要件 — 独立レビュー

- 日付: 2026-09-18（JST）
- 状態: completed。要件採否は未実施。
- 初回所有者経路: `REVIEW`。追加のレビュー質問なし。
- 固定対象: [R4-c1 要件候補](issue-17-direct-substring-requirement-r4-c1-2026-09-18.md)全文、SHA-256 `c42f678fc965027c6c8536998499ce063d9ba82eaa7e6806044cce443b4bdb42`。
- 照合先: Accepted Issue #17 R1/R2/R3 と承認記録、ADR 0033/0035/0039/0040/0041、現行 `requirements.md` / `cli.md`。実装とテストは要件の権限根拠にしない。
- 方法: 別エージェントによる読み取り専用レビュー。候補本文は変更していない。

## 結果と分類

1. **確定的な矛盾: なし。** 既存 Candidate の内容を指定文字列にし、単一 External run と元ファイル全文の SourceSnapshot を持ち、別途 Seal を公開する構成は上位契約と整合する。`trace occurrences` の全位置導出、Cause、STALE、binding の境界も維持する。
2. **証拠不足・未解消の意味境界: 1件（中）。** 候補の「最小 byte offset」は新しい登録操作で保存する `source_start` の選択規則である。一方、Accepted R3 の比較時 `selected_match_start` は探索順で最初に見つかった一件で、最小 offset を保証しない。独立レビュアーは、両者の区別を候補本文に明示しないと混同の余地があると指摘した。候補は既に「比較・位置一覧の意味は維持する」と記すため、これは確定的な矛盾ではない。採用前の明確化を提案するが、追加文の要否は所有者判断とする。
3. **既存契約と整合: 失敗時の状態。** 不一致・空文字列・読取失敗・Candidate 不在・競合で Candidate と REF HEAD を変えない受入条件は、既存の atomic Candidate 更新と整合する。exact error code・stdout/stderr は下流の CLI 設計事項である。
4. **任意の将来候補・範囲外:** CLI/flag、文字列入力方法、成功出力 schema、help/completion、既存 `trace set` との統合方法は候補が未決としている。独立レビューはこれらを新しい受入条件にしない。

## 二回目の所有者レビューへ

候補は初回提示時の exact bytes のまま。`REVISE` は候補へ登録時と比較時の位置選択の区別を追加して Step 1 に戻る。`REREVIEW` は候補を変えずレビュー質問を追加・変更して Step 3 に戻る。`ACCEPT` は現在の候補全文をそのまま採用する。いずれも ADR、実装、移行、push、merge、release は別の権限境界である。
