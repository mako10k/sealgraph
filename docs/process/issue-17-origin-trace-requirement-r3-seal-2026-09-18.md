# Issue #17 R3-c1 Accepted 要件の Seal 実行記録 — 2026-09-18

- 対象：[Accepted R3-c1 要件本文](issue-17-origin-trace-requirement-r3-candidate-2026-09-18.md)全文
- 採用権限：[R3-c1 承認記録](issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)と、直前に推奨した R3 要件の Seal 化を実施する所有者指示
- 対象本文 SHA-256：`1179004a32f3e105823bcdd714e736def52ccb5716382fa5a5f40263b649d3c3`
- REF：`sealgraph/requirement/issue-17-origin-trace-r3`
- 非 draft SealID：`de2fe3533eeb2553606f533f76c8d74e469356a746c6dde765694b94c67d1316`
- 直接 Cause：[Accepted R2-c2 要件 Seal](issue-17-r2-downstream-seal-2026-09-17.md) `c6a994dd3b5d9eba151888f00ecc80921ba71e082fb101e6fb565d8507ad8110` のみ。`--no-previous` は当該 target に構造上の前世代がないとの assertion として記録した。

## 実行と読み戻し

対象の R3-c1 ファイル SHA-256 と承認記録の対象 SHA-256 を照合した。既存の R2 REF は上記の非 draft Seal を指し、`stale --scan` に状態はなかった。R3 REF と Candidate は存在しなかった。

R3 REF に対象ファイルの exact bytes を明示入力し、`root=false`、`draft=false`、R2 要件 Seal への一件の Cause を持つ Candidate を作った。Candidate の `expected_ref_head` は absent、prospective SealID は上記の ID、raw content の SHA-256 は対象ファイルと一致した。その後、R3 REF について一度 `seal` を実行した。

Seal 直後の `show --format json` では、REF、SealID、`root=false`、`draft=false`、R2 への direct Cause 一件、空の previous-revision assertion を読み戻した。`show --raw-content` の SHA-256 は対象ファイルと同じ。`sealgraph fsck --format json` は `result=ok`（16 Seals、15 REFs、unreferenced blob なし）、`sealgraph stale --scan --format json` は `statuses=[]` だった。

R3 は R2 要件を上位 Cause とする。ADR 0039 は R3 要件より下流の設計判断であり、R3 の Cause に含めない。ADR 0039 の既存 Seal は R2 等を Cause とした歴史的な採用状態のまま保持する。新しい R3 との exact な下流接続は、後継設計・Seal を扱う別の作業で判断する。

この記録が示すのは immutable 本文 identity、保存済み Cause、REF の読み戻し、保存構造と派生 stale の状態である。要件への意味上の適合、後継 ADR の採用、Issue #17 runtime の実装・公開、remote 同期を証明しない。
