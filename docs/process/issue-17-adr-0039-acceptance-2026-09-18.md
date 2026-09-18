# Issue #17 ADR 0039 承認記録 — 2026-09-18

- 状態: Accepted（所有者による後継設計判断の採用）
- 対象: [ADR 0039: Origin Trace の一致位置を全文から導出する](../adr/0039-origin-trace-derived-occurrence-positions.md) の全文
- 対象本文 SHA-256: `671aa30df05a4f2c907f90514afc2de9a574143e95da223a120d01e2c91e38c6`
- 提示済み draft Seal: `79bf4bf5095252f790e544da74c897d3f01caac414f618eaff12e516445ad06c`。承認直前に同 Seal の raw content と対象ファイルの SHA-256 一致を確認した。

所有者は ADR 0039 の提示と draft Seal の報告を受けた後、次のように指示した。

> ADｒをＡｃｃｅｐｔします。Ｓｅａｌもお願いします。

この発言により、上記 exact bytes の ADR 0039 を Accepted とする。本文中の `Proposed`、承認待ち、および Proposed/draft Seal に関する記述は提示時点の状態を表す履歴として保持する。現在の採用判断は本記録で示し、採用後の非 draft Seal を同じ REF に別の immutable Seal として作る。旧 draft Seal は削除・書換えしない。

採用範囲は、変更有無の byte 部分列存在判定を維持し、記録 offset を初期ヒントとし、必要な時点に操作の意味に応じて元版・現在版のどちらかまたは両方から一致位置を導出し、多数の表示・提案には既定上限とページングを使う後継設計判断である。元ファイル全文の immutable Blob 保持は ADR 0033 の既存設計を使う。

本 ADR 自身が明記する条件は維持する。Accepted R2 要件の exact 文書はまだ改訂されておらず、ADR 0036～0038 の置換範囲と公開 CLI/schema の後継契約は、後継要件との整合を確定するまで実装権限として扱わない。既定上限の数値、ページ識別子、各操作の版選択、性能保証は未決である。本承認と Seal は実装、移行、commit、push、release を許可または完了しない。

## 採用後の Seal と読み戻し

- REF: `sealgraph/decision/adr-0039-derived-occurrences`
- 非 draft SealID: `7b3175f2d6d3de53d7f6ed489c0c4da18ac016fe397f39a5354f472a2616e204`
- `root=false`、`draft=false`。直接 Cause は Accepted R2 要件と ADR 0036～0038 の４件で、[提案 Seal 記録](issue-17-adr-0039-proposed-seal-2026-09-18.md)に示す exact SealID と一致する。
- Seal 前の Candidate は `expected_ref_head` が旧 draft SealID と一致し、本文・Cause は不変、draft だけが `true` から `false` に変わった。Seal 後の `show --raw-content` と対象ファイルの SHA-256 は一致した。
- `sealgraph fsck --format json` は `result=ok`（15 Seals、14 REFs、unreferenced blob なし）。旧 draft Seal は `historical_or_detached_seal_ids` に１件として残る。`sealgraph stale --scan --format json` は `statuses=[]`。

この検査は immutable identity、記録済み Cause、保存整合性を示す。要件改訂、意味上の適合、実装完了、リモート同期を証明しない。
