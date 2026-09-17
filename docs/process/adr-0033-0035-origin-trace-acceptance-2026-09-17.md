# ADR 0033～0035 承認記録 — 2026-09-17

- 状態：Accepted（所有者による詳細契約の採用）
- 対象：以下の3文書の全文。候補時の本文・状態表記を含む exact bytes を保存し、現在の採用状態は本記録で示す。

| 対象 | 承認対象 SHA-256 |
|---|---|
| [ADR 0033 保存形式・移行](../adr/0033-origin-trace-storage-and-migration.md) | `c71d26a77779ddb3ce744c738c6f981ab52f0291a39828d91163e3dddce327fc` |
| [ADR 0034 対応判定](../adr/0034-origin-trace-correspondence.md) | `070bfff4da77bf2db22b22849ef6cd9c0fc3064e470f4940c3290a8858554f66` |
| [ADR 0035 CLI・観測出力](../adr/0035-origin-trace-cli-and-observation-output.md) | `3546777194a10d6ef1ce2c21b1b4a2913591bb961c7cbeb4d2d75941c623cc00` |

## 所有者判断

[CONTRACT_REVIEW 判断資料](issue-17-origin-trace-contract-review-2026-09-17.md)を提示し、次の範囲を明示して採否を求めた。

> 比較予算の明示必須と、全 Blob の native dump/load を含め、3文書を採用してよいですか？

所有者の回答：

> はい。

この回答を、上記の全文と明示した二つの選択肢を含む承認として記録する。根拠は所有者の回答であり、レビュー・監査の PASS ではない。

## 確定した詳細契約

- format 7 と後継 typed schema、正規 bytes、構造検証、Candidate→Seal の exact origin 保持、旧形式 5/6 の bytes/ID を保持する明示移行。
- 全最適 byte 対応の対象範囲における合意、内部編集・削除・境界競合の区別、固定した二版への明示対応宣言。
- 専用 Trace 操作と方向別の詳細観測、公開 machine schema、比較の alignment cells / graph visits 予算の明示必須。
- 無参照分を含む全 valid Blob、REF/tag、Candidate の native dump/load。原文全文 snapshot を含み、local binding・対応宣言等は含めない。load は `.sealgraph` が存在しない宛先だけを対象とする。
- status 表示は詳細側で試して必要性が見えたら再検討する。Cause Link の whole-Seal 対象、既存 STALE、意味論を監査しない方針を維持する。

## 残る事項と読み戻し

対象 workload、platform、資源条件・性能基準は ENVIRONMENT_REVIEW に残る。今回の採用はそれらの値、実装完了、実データ移行、dump/load 実行、commit/push・配布の決定を含まない。

対象ファイルの SHA-256 は承認記録作成前に照合し、提示済みの値と一致した。上位は Accepted Issue #17 R1 と [ADR 0032 D1 承認記録](adr-0032-origin-trace-acceptance-2026-09-17.md)。独立レビューの範囲・指摘処置・status 保留反映後のレビュー限界は[既存レビュー記録](issue-17-origin-trace-companion-review-2026-09-17.md)を保持する。

機能利用価値はまだ未実現。今回、実装前提のうち詳細契約の採否が確定した。実績工数・費用は未計測。
