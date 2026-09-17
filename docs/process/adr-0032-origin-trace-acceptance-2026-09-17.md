# ADR 0032 D1 承認記録 — 2026-09-17

- 状態：Accepted（所有者による設計承認）
- 対象：[ADR 0032 D1](../adr/0032-content-origin-trace-and-directional-observation.md) 全体
- 承認対象 SHA-256：`072314438483af19d2eb9b84b36d6a5b15d1ed358068f1849da473f34a3c38d0`
- 所有者の発言：

> 承認します。
> 進めてください。

直前に提示した日本語の ADR 0032 D1 とその判断点に対する承認を記録する。
承認対象の本文を変更せず、候補時点の Status/Owner disposition を含めて保存する。
現在の Accepted 状態はこの記録で示す。

## 承認された設計範囲

- 原文全体を immutable Blob として保持し、小さな Seal の content の範囲を SourceSnapshot/OriginMap で対応付ける。
- OriginMap を後継 Provenance に含め、由来を immutable な Seal の識別に反映する。
- 全文の最小挿入・削除対応を、曖昧性を保持した根拠付き推定として扱う。明示的な対応宣言も比較入力にできる。
- Candidate 自身の比較と HEAD/exact Seal グラフの方向別観測を分け、Candidate の Cause を暗黙に overlay しない。
- 各 Seal 自身の比較結果から、自身・上流・下流を独立に集計する。
- Cause は whole-Seal、STALE は既存定義のまま。意味論を監査しない。
- ADR §10～12 の既存契約との接続および未決の詳細契約を、その明示された段階に残す。

独立レビューは [D1 レビュー記録](issue-17-origin-trace-design-review-2026-09-17.md) の `PASS_WITH_FINDINGS`。
承認の根拠は所有者の発言であり、レビュー結果ではない。

## 継続する作業と残る境界

設計作業を続け、ADR §12 A1～A3 の保存形式・比較方式・CLI/output の詳細候補を作成する。
後継形式番号・exact bytes・公開コマンド・資源予算など、D1 が未決としていた選択肢を今回の承認だけで決定済みとは扱わない。
対象規模・性能の合格基準（A4）は未設定。候補を提示して判断できる形にする。
実装の適合判定には詳細契約の確定が必要であり、この記録をコード完成・移行実施・リリースの証明にしない。
commit/push、実データの移行、外部公開は実施していない。

## 読み戻し

承認記録作成前に対象ファイルの SHA-256 を上記と照合した。
上位要件は Issue #17 R1 Seal
`0f6adb86be04ac509f6affd117dc49013b6186e4da45e53a375a3f198990a80b`。
本記録の作成は既存の要件・承認 Seal を変更しない。
