# Issue #17 ADR 0041 承認記録 — 2026-09-18

- 状態: Accepted（所有者による出力契約の採用）
- 対象: [ADR 0041: Trace mutation receipt に入力ファイル名を含める](../adr/0041-trace-mutation-receipt-input-file.md) の全文
- 対象本文 SHA-256: `2b299c8d38481dab146603cd2f05809f093b261096d2c94fcbe077ba5d32372d`
- 基底: Accepted Issue #17 R3、ADR 0035/0038、および owner の出力方針 B

## 所有者の判断

所有者は、stderr と stdout を合わせて成功結果とする案 A を退け、成功結果自体に入力名を含める B を選択した。その後、ADR 0041 の日本語全文、`v2` の exact member、Snapshot 再利用時の `null`、旧契約の限定的な置換、代案、リスクを提示した。所有者の回答は次のとおり。

> 承認します。

この回答により、上記 SHA-256 の ADR 0041 全文を Accepted とする。本文中の `Status: Proposed` や採否待ちの記述は提示時の履歴として保持し、現在の採用状態は本記録で示す。独立レビューで指摘された既存 receipt 維持条項との衝突は、ADR 0041 に ADR 0035 §4.1/§4.2、ADR 0038、ADR 0027 の format 7 `trace set/clear` に限る後継化と採用条件を明記し、再レビューで解消を確認した。レビューは採否を代行しない。

## 採用範囲と残る作業

format 7 の `trace set/clear` 成功 JSON は `sealgraph/trace-mutation/v2` を使い、`stored_sources` の `input_file` に今回の file 入力名または SnapshotID 再利用時の `null` を返す。human の成功出力にも file 入力名と byte 数、または SnapshotID 再利用を示す。pre-store stderr は維持する。旧 `v1` の member 集合や immutable object、Trace の意味、`trace show/compare` は変更しない。

この採用は設計判断であり、CLI 実装、検証、Seal、push、release の完了を意味しない。外部 `v1` 消費者と配布状態は未確認であり、互換性・公開の主張はしない。
