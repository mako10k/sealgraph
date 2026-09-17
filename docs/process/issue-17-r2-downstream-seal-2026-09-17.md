# Issue #17 R2 要件と下流４文書の Seal 実行記録 — 2026-09-17

- 状態：５件のローカル Seal 作成・読み戻し完了
- 権限：[R2 要件の採用記録](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)、[下流４文書の承認記録](issue-17-r2-downstream-acceptance-2026-09-17.md)、追加 R2 Seal を明示的に承認した所有者回答
- 作業範囲：既存 R1 要件 Seal の下流に R2 要件１件、続いて承認済み ADR 0036・0037・0038・P2 を各１件 Seal。既存の承認対象本文は変更しない。

## 追加 Seal の所有者判断

R2 要件の exact Seal がないまま４文書を Sealing すると、R2 の上位権限へ Cause を向けられない。追加する１件の対象本文、REF、既存 R1 Seal への Cause、追加 REF と immutable Seal を作る効果を説明して確認した。所有者は次の選択肢を選んだ。

> はい、R2 要件 Seal を追加して４件を Seal する

この回答は R2 Seal の追加と指定４文書の Seal に限る。commit/push、実装、移行、公開の指示ではない。

## Exact readback

各件について `add` の Candidate content と Cause を確認し、一 REF ずつ `seal` した。直後の `show --format json` で SealID・REF・Cause を、`show --raw-content` の SHA-256 で exact 本文を照合した。

| 対象 | REF | SealID | content SHA-256 | Cause の exact SealID |
|---|---|---|---|---|
| [Accepted R2-c2](issue-17-origin-trace-requirement-r2-candidate-2026-09-17.md) | `sealgraph/requirement/issue-17-origin-trace-r2` | `c6a994dd3b5d9eba151888f00ecc80921ba71e082fb101e6fb565d8507ad8110` | `a1dd1f6afdbb11305d67bc1b6498440e666ff50a2c60c56f6b22f7dfdd0ed4b2` | R1 要件 `0f6adb86be04ac509f6affd117dc49013b6186e4da45e53a375a3f198990a80b` |
| [ADR 0036](../adr/0036-origin-trace-r2-presence-and-directional-observation.md) | `sealgraph/decision/adr-0036-origin-trace-r2-presence` | `1ea59f63cc262c99bf0796185b9f8d65ba9dcc890fcccb8eaad949d91eb69ad1` | `ce0918a6fcfa73d17788ddb9f03b0736efb8c4775c949405748e036fb7de301f` | R2 `c6a994dd3b5d9eba151888f00ecc80921ba71e082fb101e6fb565d8507ad8110` |
| [ADR 0037](../adr/0037-origin-trace-r2-search-and-estimation.md) | `sealgraph/decision/adr-0037-origin-trace-r2-search` | `8ddf95e9cc30c3a16715b0a1c95cdedd28e6686a00ab72457f468dc4d4ba4be5` | `ccac49c6c1d93b611a72ab83df0bab20c9b70fa888e195a1772f85d4c47d32fd` | ADR 0036 `1ea59f63cc262c99bf0796185b9f8d65ba9dcc890fcccb8eaad949d91eb69ad1` |
| [ADR 0038](../adr/0038-origin-trace-r2-cli-and-observation-output.md) | `sealgraph/decision/adr-0038-origin-trace-r2-cli` | `8735c8d58e35821ecd0dc56bda148a55edfe643b9e6b17bac2c304989b7c02de` | `e50f007c2b534eebd8ded0ec57c649d215fb371444ad3186959e88dea3648be1` | ADR 0037 `8ddf95e9cc30c3a16715b0a1c95cdedd28e6686a00ab72457f468dc4d4ba4be5` |
| [実装計画 P2](issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) | `sealgraph/process/issue-17-origin-trace-r2-plan-p2` | `5ab59c6a6574c122ed7a9239de1fc1c702579138fde90d7ce2823c9a31705fae` | `3737ec11f741c344ad917cad8daff252a2addac15b11885a452999d1adedf21d` | ADR 0038 `8735c8d58e35821ecd0dc56bda148a55edfe643b9e6b17bac2c304989b7c02de` |

対象は全件 `root=false`、`draft=false`。`--no-previous` は各 Cause target に前世代がないという初回リンクの明示的な主張として使った。動的 HEAD の選択表記は保存されず、上表の full SealID が記録された。

## 整合確認と限界

- `sealgraph fsck --format json`: `result=ok`、13 Seals、13 REFs、unreferenced blobs なし。
- `sealgraph stale --scan --format json`: `statuses=[]`。
- `sealgraph graph --format json`: 上表の５ REF がそれぞれ１件の `ACTIVE_LEAF` を指し、Cause は記載どおり。

これらは immutable identity、保存構造、記録された Cause と stale 観測の確認である。設計の真実性、意味的な要件適合、機能実装、性能保証、所有者の追加承認を証明しない。ADR 0033 等の過去の Accepted 文書は今回 Seal しておらず、その全ての条項をこの５件の Cause graph が網羅したとは主張しない。

実装、旧 PERT の R2 改訂、実 repository の形式移行、commit/push、配布は実施していない。保存先は現在のローカル作業ツリーであり、GitHub への同期は未確認・未実施。
