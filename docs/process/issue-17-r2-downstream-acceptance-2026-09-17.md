# Issue #17 R2 下流４文書の承認記録 — 2026-09-17

- 状態：Accepted（所有者による設計・計画の承認）
- 決定者：ユーザー（Operator）
- 承認範囲：直前に提示した ADR 0036、0037、0038、実装計画 P2 の各全文
- 上位要件：[Accepted Issue #17 R1 + R2](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)

| 承認対象 | 提示時と承認直後の SHA-256 |
|---|---|
| [ADR 0036 原文残存と方向別観測](../adr/0036-origin-trace-r2-presence-and-directional-observation.md) | `ce0918a6fcfa73d17788ddb9f03b0736efb8c4775c949405748e036fb7de301f` |
| [ADR 0037 完全一致探索と変更後範囲推定](../adr/0037-origin-trace-r2-search-and-estimation.md) | `ccac49c6c1d93b611a72ab83df0bab20c9b70fa888e195a1772f85d4c47d32fd` |
| [ADR 0038 CLI と観測出力](../adr/0038-origin-trace-r2-cli-and-observation-output.md) | `e50f007c2b534eebd8ded0ec57c649d215fb371444ad3186959e88dea3648be1` |
| [実装計画 P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) | `3737ec11f741c344ad917cad8daff252a2addac15b11885a452999d1adedf21d` |

## 所有者の決定

４文書の目的、変更点、設計判断、未測定の性能条件、旧 PERT の不整合を提示した後、所有者は次のように回答した。

> OK、４文章承認、Sealをお願いします。

この発言により、上表の exact snapshots を Accepted とする。候補本文の `Proposed` や `Owner disposition: 未承認` は提示時の履歴としてそのまま保存し、現在の承認状態は本記録で示す。Accepted R2 に矛盾する旧 ADR 0032/0034/0035 の比較関連条項は、採用された 0036/0037/0038 の対応箇所で置き換える。旧ファイルの bytes や過去の採用記録は変更しない。ADR 0033 の保存方式、whole-Seal Cause、既存 STALE、意味論非監査、status 表示の保留は維持する。

この承認は、４文書の設計・計画の採用であり、Issue #17 の機能実装、公開 CLI の存在、対象 workload の数値性能保証、旧 PERT の改訂、実 repository の形式移行、commit/push、リリースを完了または実行したことを意味しない。

## Seal の前提と実行状況

SealGraph には Accepted R1 の要件 Seal `0f6adb86be04ac509f6affd117dc49013b6186e4da45e53a375a3f198990a80b` がある。承認時点では Accepted R2 の exact 要件 Seal はない。４文書の Cause を R2 の正確な上位権限へ向けるには、R2 要件の Seal が別途必要である。所有者の発言は対象を「４文章」と指定しているため、この記録だけで５件目の R2 Seal を作成済み、または承認済みとは扱わない。旧 R1 Seal へのリンクや root 指定で R2 上位権限を代用しない。

本記録の作成時点では４文書の Seal は未作成。必要な上位 R2 Seal の扱いを明示した後、ADR 0036→0037→0038→P2 の順に、各 exact 文書を一 REF・一 Seal で公開し、各結果を読み戻す。成功した結果 ID は別の実行記録で示し、この承認記録へ推測で書き込まない。
