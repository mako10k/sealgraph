# Issue #17 R4-c1 ADR 0042 提案 Seal 記録 — 2026-09-28

- 対象: [ADR 0042 日本語全文](../adr/0042-direct-substring-trace-authoring.md)
- 状態: Proposed。所有者の採否は未決。
- 本文 SHA-256: `270d6982869e58e6f99e32d8ca0d7d237fa5aba35e229c227b7edb8d39123afe`
- REF: `sealgraph/decision/adr-0042-direct-substring-trace`
- draft SealID: `d0d8af5c7b9f5e3aad661f604bed22eb78b8f733504f2f032250ae731ee0f38e`
- `root=false`、`draft=true`。

## 直接 Cause

| 上位 authority | exact target SealID | 継承する範囲 |
| --- | --- | --- |
| Accepted R4-c1 要件 | `82798e71409cc92712837b02d9f994ee2b6a3b1dba79ecaf60880e2140a55625` | 直接部分文字列入力、最小登録位置、失敗境界 |
| Accepted ADR 0033 | `3132b225851f7ecc4c429d0c7970696048adad960d6ab4d745b9061ccbc9d97a` | SourceSnapshot、OriginMap、Candidate 更新 |
| Accepted ADR 0035 | `e1c2be778f6b97e71501451f0183e978de5426260d1f15c07d81d8c7286bc78e` | 既存 recipe 経路と CLI 境界 |
| Accepted ADR 0039 | `7b3175f2d6d3de53d7f6ed489c0c4da18ac016fe397f39a5354f472a2616e204` | 比較時の選択位置の意味 |
| Accepted ADR 0040 | `aef8b43404bff1c0e4531513c8358043c72993b5f2e7258aa2e9d68b171f0fc8` | 一致位置一覧の責務 |
| Accepted ADR 0041 | `6e15eca78687887e7314914141a1041182cd02bad5b01e904e2678449fad4953` | `trace-mutation/v2` 成功 receipt |

ADR 0033/0035/0041 の従来未登録だった Accepted 本文も、各承認記録の SHA-256 に一致することを確認してから非 draft Seal として登録した。ADR 0032 は R1 要件、ADR 0033/0034 は ADR 0032、ADR 0035 は ADR 0033/0034、ADR 0041 は ADR 0035・0038・R3 要件をそれぞれ上位 Cause とする。

公開前の Candidate は `expected_ref_head=null` で、raw content の SHA-256 と6件の exact Cause を確認した。公開後の raw content も上記 SHA と一致し、`show --format json` は `draft=true` と6件の Cause を返した。`sealgraph fsck --format json` は `result=ok`（25 Seals、24 REFs、unreferenced Blob なし）、`sealgraph stale --scan --format json` は `statuses=[]`。

この draft Seal は提案本文の immutable identity と記録した由来を示す。所有者がこの exact 候補を採用した場合にのみ、別の非 draft Seal として同じ REF に後継世代を作る。採用、実装、検証、実 repository 移行、push、merge、release はこの記録では完了しない。
