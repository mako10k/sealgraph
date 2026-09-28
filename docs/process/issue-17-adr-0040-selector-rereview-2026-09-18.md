# Issue #17 ADR 0040 baseline 選択の差分レビュー — 2026-09-18

- 対象：[ADR 0040](../adr/0040-origin-trace-occurrence-listing-and-paging.md) の `trace occurrences` 選択規則、出力 identity、cursor 束縛、確認例
- 最終候補 SHA-256：`7efab99f4cda853b0108baf8a0f99b3a88d23873f044ad1cbe0dddd242c35fd9`
- 経緯：[初回の全文独立レビュー](issue-17-adr-0040-independent-review-2026-09-18.md)後、コマンド全体との照合で `--ref` の暗黙 Candidate 優先が ADR 0035/0038 の Candidate・HEAD 区別と不整合と判明。所有者の修正指示により、対象 baseline の明示選択へ変更した。
- レビュー範囲：変更した選択契約と、Accepted ADR 0035/0038 の `trace show/compare` 選択意味との整合。変更していない位置列挙・ページ数・未完了意味は再レビュー対象外。
- 独立レビュー：変更後の中間候補 SHA-256 `322eaad1a1a3b6bafe6a718ed578472d1f7614df2e7e013563aba9550535eff0` を読取専用で確認。判定 `PASS_WITH_FINDINGS`、P0～P2 なし、P3 が１件。
- 所有者判断：未実施。ADR は Proposed のままで、レビューは採用・Seal・実装を意味しない。

## 差分の確認

`--ref REF` では `--baseline candidate|head` を必須とし、指定した側がなければ失敗する。`--seal SELECTOR` は exact Seal を選び、`--baseline` と併用しない。一覧は一つの baseline の一つの run に限定する。`selection.baseline` は選択後の Candidate digest または SealID を示す。cursor はその選択と exact identity に束縛し、選択していない側の変更だけでは失効しない。

| ID | 独立レビューの指摘 | 調整結果 |
|---|---|---|
| P3-1 | 中間候補の cursor 文に「`--ref` と `--baseline`、または `--seal`」とあり、文字面では組合せが少し曖昧。ただし操作の排他規則と schema から二択として読め、採用を妨げる問題ではない。 | §2.2 を排他的な選択組 `(REF, --baseline candidate|head)` または `(--seal SELECTOR)` と明記した。その他の選択・結果・cursor 意味は変えていない。最終候補のハッシュを再確認する。 |

独立レビューは `trace show/compare --ref` の Candidate/HEAD 表示を変更しないこと、暗黙 fallback がないこと、JSON identity と cursor の整合、Candidate/HEAD が異なる場合の確認例を照合した。CLI 実装・実動作・性能は対象外。旧全文レビューは旧候補の exact bytes に対する履歴であり、最終候補へそのまま適用したとは扱わない。
