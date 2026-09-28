# Issue #17 R4-c1 ADR 0042 採用記録 — 2026-09-28

- 状態: Accepted decision（所有者回答による）。
- 対象: [ADR 0042 日本語全文](../adr/0042-direct-substring-trace-authoring.md)、SHA-256 `40144c7c70fbd637f74b9a9d51cc403fb9a92ef612054552e8be5104ede669ef`。
- 採用前の draft Seal: `bfbe9ce2c2b4dabd1f5985b888b670bca584e1736408e78b56190a451d935ee7`。
- Accepted 設計 Seal: `b18fcd8b381ffc5581024d4aa3d851dee7b51cb209647bf6fc9260c564da9070`（非 draft、上位 Cause 6 件）。
- 上位: Accepted R4-c1 要件と Accepted ADR 0033/0035/0039/0040/0041 の該当契約。

所有者には対象全文と旧候補からの変更を提示した。変更はインライン `--content STRING` を追加して既存 `--content-file PATH|-` を保つこと、および UTF-8 必須条件が指定内容のみに掛かることの明確化である。所有者はこの exact 日本語全文を R4-c1 の公開 CLI 設計として「Accepted として進める」と回答した。

対象 ADR 本文の `Status: Proposed` は採用前の exact snapshot を示す履歴として保持する。この採用記録が対象の Accepted 状態を確定する。設計 Seal は同じ本文 bytes と既存の6件の上位 Cause を持つ非 draft 世代として作成し、後続の実装・検証はその exact Seal に従う。
