# Issue #17 Accepted 実装計画 P3 の Seal 実行記録 — 2026-09-18

- 対象：[Accepted P3 計画](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md)の exact 全文。
- 採用権限：[P3 承認記録](issue-17-origin-trace-implementation-plan-r3-acceptance-2026-09-18.md)と、直前の推奨を実施する所有者指示。
- 対象本文 SHA-256：`cff3db3420a8e9c5abd07455e14f4b7148b74dcef22d50dce0521633a44c82fd`。
- REF：`sealgraph/process/issue-17-origin-trace-r3-plan-p3`。
- 非 draft SealID：`65548e23fb136a6085c33a46a8be47ed9332b42f7fa6dfeaef592fade00be9d8`。

## 実行前の確認

対象本文の SHA-256 は承認記録と一致した。Accepted R3 要件と ADR 0040 の現在の REF HEAD は以下の exact 非 draft SealID を指し、それぞれの `show --raw-content` は承認済み本文 SHA-256 と一致した。P3 REF と Candidate は存在しなかった。`fsck` は `result=ok`、`stale --scan` は空だった。

`add` の最初の呼出しは、同コマンドが受けない `--format` を指定したため入力エラーで終了した。Candidate 不在を再確認してから、R3 Cause を付けて `add` し、ADR 0040 Cause を `link` で追加した。

| 直接 Cause | exact SealID | previous assertion |
|---|---|---|
| [Accepted R3-c1 要件](issue-17-origin-trace-requirement-r3-seal-2026-09-18.md) | `de2fe3533eeb2553606f533f76c8d74e469356a746c6dde765694b94c67d1316` | なし（`--no-previous`） |
| [Accepted ADR 0040](issue-17-adr-0040-seal-2026-09-18.md) | `aef8b43404bff1c0e4531513c8358043c72993b5f2e7258aa2e9d68b171f0fc8` | なし（`--no-previous`） |

公開前の `candidate show` は `expected_ref_head=null`、`EXPECTED_ABSENT`、`root=false`、`draft=false`、上記２件の Cause と prospective SealID `65548e23fb136a6085c33a46a8be47ed9332b42f7fa6dfeaef592fade00be9d8` を示した。Candidate の raw content SHA-256 も対象本文と一致した。そこでこの REF に対し `seal` を一回実行した。

## 読戻しと限界

直後の `show --format json` は同じ SealID・REF・`root=false`・`draft=false`・２件の直接 Cause を示し、`show --raw-content` の SHA-256 は対象本文と一致した。Candidate は残っていない。`fsck --format json` は `result=ok`（18 Seals、17 REFs、unreferenced Blob なし）、`stale --scan --format json` は `statuses=[]`。

P2 は歴史的な前版計画であり、P3 の上位権限として Cause にしない。R3 は R2/R1 要件へ、ADR 0040 は ADR 0039 と R3 へ、ADR 0039 は ADR 0036～0038 等へ接続する。引用されている ADR 0033 など、Seal のない歴史文書まで Cause graph が網羅するとは主張しない。この実行が証明するのは exact bytes、宣言された Cause の向き、保存整合性と派生 stale 観測である。実装・意味的適合・公開・リモート同期の証明ではない。
