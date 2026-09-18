# Issue #17 ADR 0040 承認記録 — 2026-09-18

- 状態：Accepted（所有者による後継公開契約の採用）
- 対象：[ADR 0040: Origin Trace の一致位置一覧とページング公開契約](../adr/0040-origin-trace-occurrence-listing-and-paging.md) の修正後全文
- 対象本文 SHA-256：`7efab99f4cda853b0108baf8a0f99b3a88d23873f044ad1cbe0dddd242c35fd9`
- 基底：Accepted [Issue #17 R3-c1](issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md) と [ADR 0039](issue-17-adr-0039-acceptance-2026-09-18.md)
- レビュー：[初回の独立レビュー](issue-17-adr-0040-independent-review-2026-09-18.md)、[baseline 選択修正後の差分レビュー](issue-17-adr-0040-selector-rereview-2026-09-18.md)。いずれも所有者の採否を代行しない

## 所有者の判断

ADR 0040 のコマンド全体との対称性・一貫性・網羅性を再点検し、`--ref` の暗黙 Candidate 優先を修正した。修正後の日本語全文、差分レビュー、Proposed 状態を提示し、次の判断を更新後全文の採否と明示した。所有者は次のように回答した。

> Acceptします。

この回答により、上記 SHA-256 の ADR 0040 全文を Accepted とする。本文中の `Status: Proposed` や採否待ちの記述は承認前の exact snapshot の履歴として保持し、現在の採用状態は本記録で示す。初回レビューの旧 digest を修正後本文のレビュー証拠として読み替えず、選択規則の差分レビューを併せて参照する。

## 採用範囲と残る作業

採用した契約は、一つの External run について元版 S・現在版 F・両方から必要時に一致位置を導出する専用の読取 CLI、既定 100 件の返却上限、版に束縛したページング、`sealgraph/trace-occurrences/v1` の出力である。`--ref REF` では `--baseline candidate|head` を必須とし、`--seal SELECTOR` では exact Seal を選ぶ。指定 baseline がない場合の暗黙 fallback はしない。存在判定の一件探索、`trace compare` v2、Cause、STALE、status 表示の保留、canonical 保存形式は本承認で変更しない。

この承認は ADR 0040 の設計採用であり、ADR の Seal、P2/PERT の改訂、CLI/runtime 実装、元ファイル全文復元やページ性能の実測、形式 7 の移行、commit/push、release の承認または完了を意味しない。位置に基づく別の提案操作と、元ファイル全文を一件ずつ取り出す専用 CLI の要否は、本 ADR が確定しない後続判断である。
