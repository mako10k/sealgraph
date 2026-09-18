# Issue #17 ADR 0040 独立レビュー — 2026-09-18

- 対象：[ADR 0040 Origin Trace の一致位置一覧とページング公開契約](../adr/0040-origin-trace-occurrence-listing-and-paging.md) の 2026-09-18 Proposed 全文
- 対象 SHA-256：`6606b1341ac5b0124aecaf537af4152d3d9eba65cacaa9ab1e88b98af8ac53e6`
- レビュー方式：本文を書き換えない独立レビュー。Accepted R3 の AC16～AC21、ADR 0033・0036～0039、CLI と保存形式の境界を照合
- 独立レビュー判定：`PASS_WITH_FINDINGS`。P0～P2 の指摘なし、P3 の確認事項１件
- 所有者判断：未実施。レビュー判定は ADR の採用・Seal・実装承認ではない

## 独立レビューの確認事項と調整結果

| ID | 独立レビューの指摘・不確実性 | 調整結果 |
|---|---|---|
| P3-1 | ADR 0036～0038 の本文には履歴上の `Status: Proposed` が残る。レビュー時の入力だけでは、これらの exact snapshot が現在 Accepted であることを独立に確認できない。 | [R2 下流４文書の承認記録](issue-17-r2-downstream-acceptance-2026-09-17.md)が、所有者の承認範囲と各 exact SHA-256 を示す。現行ファイルの SHA-256 は ADR 0036 `ce0918a6fcfa73d17788ddb9f03b0736efb8c4775c949405748e036fb7de301f`、0037 `ccac49c6c1d93b611a72ab83df0bab20c9b70fa888e195a1772f85d4c47d32fd`、0038 `e50f007c2b534eebd8ded0ec57c649d215fb371444ad3186959e88dea3648be1` で記録と一致した。履歴本文を変更せず、採用状態を承認記録で読む。ADR 0040 の修正は不要。 |

独立レビューは、R3 の一件による原文残存判定と全位置一覧の分離、`snapshot/current/both` の版選択、重複一致、ページの連続 prefix、版が変わった際の cursor 失効、未完了と空集合の区別を確認した。`trace compare` v2、OriginMap、SourceSnapshot、Cause、STALE を本候補が暗黙に変更する指摘はなかった。

## 残る判断と検証限界

- 既定 100 件、専用 `trace occurrences`、版固定 cursor、新 schema は提案であり、所有者の採否待ち。
- レビューは文書間の整合を対象とした。CLI の動作、全文復元、ページ性能、実際の利用 workload は検証していない。
- Issue #17 の runtime、公開 CLI、形式 7、移行、P2/PERT の整合は未実施。これらの状態をレビュー合格から推定しない。
