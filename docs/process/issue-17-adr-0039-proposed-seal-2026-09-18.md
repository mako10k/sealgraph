# Issue #17 ADR 0039 提案 Seal 記録 — 2026-09-18

- 対象: [ADR 0039: Origin Trace の一致位置を全文から導出する](../adr/0039-origin-trace-derived-occurrence-positions.md)
- 状態: Proposed。所有者は方針を示し、後継 ADR の作成と Seal 化を依頼した。本書の exact bytes の採用判断、後継要件の採用、実装は未了。
- REF: `sealgraph/decision/adr-0039-derived-occurrences`
- SealID: `79bf4bf5095252f790e544da74c897d3f01caac414f618eaff12e516445ad06c`
- 本文 SHA-256: `671aa30df05a4f2c907f90514afc2de9a574143e95da223a120d01e2c91e38c6`
- `root=false`、`draft=true`。

## 直接 Cause

| 参照元 | exact target SealID | 関係 |
|---|---|---|
| Accepted R2 要件 | `c6a994dd3b5d9eba151888f00ecc80921ba71e082fb101e6fb565d8507ad8110` | 既存の上位要件。新方針との整合は後継要件で確定する |
| Accepted ADR 0036 | `1ea59f63cc262c99bf0796185b9f8d65ba9dcc890fcccb8eaad949d91eb69ad1` | 後継対象の原文残存設計 |
| Accepted ADR 0037 | `8ddf95e9cc30c3a16715b0a1c95cdedd28e6686a00ab72457f468dc4d4ba4be5` | 後継対象の探索設計 |
| Accepted ADR 0038 | `8735c8d58e35821ecd0dc56bda148a55edfe643b9e6b17bac2c304989b7c02de` | 後継対象の表示・出力設計 |

全ての target は exact SealID として解決され、各 Cause は `--no-previous` で target に前世代がないと主張した。Seal 前の Candidate raw content とファイルの SHA-256 が一致し、Seal 後の raw content も同一だった。`sealgraph show --format json` で REF、SealID、draft、４ Cause を読み戻した。`sealgraph fsck --format json` は `result=ok`（14 Seals、14 REFs、unreferenced blob なし）、`sealgraph stale --scan --format json` は `statuses=[]`。

SealGraph は本文の同一性、記録した参照、保存整合性を示す。ADR の採用、旧決定の失効、要件適合、機能実装、性能、リモートへの同期は示さない。次の権限判断は後継要件と本 ADR の exact snapshot の採否である。
