# Issue #17 詳細設計一式 — レビュー記録

- 日付：2026-09-17
- 上位：[ADR 0032 D1 承認記録](adr-0032-origin-trace-acceptance-2026-09-17.md)
- 対象：以下の日本語文書全体。詳細設計の新しい外部契約は Proposed。
- Reviewer：独立 `gpt-5.6-luna`、read-only、`/root/design_review`。
- 範囲：Accepted R1/ADR 0032 との矛盾と、保存・比較・CLI の間で実装結果を変える契約不整合。
- 対象外：採用済みの全文保持/Candidate非overlayの再審議、未設定の性能値を勝手に合格基準にすること。

## 最終レビュー対象

| 文書 | SHA-256 |
|---|---|
| [ADR 0033 保存形式](../adr/0033-origin-trace-storage-and-migration.md) | `c71d26a77779ddb3ce744c738c6f981ab52f0291a39828d91163e3dddce327fc` |
| [ADR 0034 対応判定](../adr/0034-origin-trace-correspondence.md) | `070bfff4da77bf2db22b22849ef6cd9c0fc3064e470f4940c3290a8858554f66` |
| [ADR 0035 CLI/output](../adr/0035-origin-trace-cli-and-observation-output.md) | `fed487d5ca6a15558e558288faef5d38fa13f1af2b102d960df92e60adc891cf` |

初回は ADR 0033 だけ `505c422207affdb9cb3f8791b9fd2febde931b5ec093154062b7097a068461f4`。
ADR 0034/0035 は初回と最終で同じ bytes。再確認は ADR 0033 の変更した二箇所と、その前提に限定した。

## 初回所見と処置

初回 verdict は `PASS_WITH_FINDINGS`。所見を以下に原文で保持し、調整者の処置を分離する。

### F1 — Reviewer P1

> Candidate origin to published Provenance origin preservation is not explicitly contracted. ADR 0033 defines Candidate v7 with origin and Provenance v3 with origin (lines 60–68), while ADR 0035 defines trace set as a Candidate update and says ordinary add preserves origin when content is unchanged (lines 49–58). However, no companion clause requires publication/sealing to copy the Candidate’s exact origin into the resulting Provenance v3, reject a mismatch, and retain the same OriginMap identity. Without that invariant, a valid Candidate Trace can be silently lost or altered at publication, violating accepted R1/ADR0032’s immutable origin requirement. Add an explicit Candidate→Provenance equality/transfer rule and corresponding mismatch failure/readback contract.

**処置：RESOLVED。** 既存の一 Candidate 公開と Accepted な immutable Trace は黙った由来消失を既に許していないため、「消失を許可する仕様が存在した」とは解釈しない。
後継型の publication 対応を明示する指摘として採用し、ADR 0033 §2.4 に exact origin/null 転記、content 等値、公開前の検証、公開後の readback と曖昧な結果の reconciliation を記述した。
製品の不具合が再現されたという記録ではない。

### F2 — Reviewer P2

> ADR 0033 load has an impossible noncanonical validation requirement. Line 102 requires validating a “final coherent source observation,” but lines 98–104 explicitly exclude local source bindings, correspondence, timestamps, and working files from native snapshot transport. A load cannot validate current source observation without those inputs. Limit load validation to canonical typed closure, REF/tag manifests, Candidates, and exact bytes; leave current-source observation for a later explicitly configured comparison.

**処置：RESOLVED。** `source observation` が dump 元 repository の観測と current source file の観測のどちらを指すか不明確だった。
ADR 0033 §2.5 に、dump は repository capture を照合し、load は stable input bytes と staging 内 canonical 構造を検証することを明記した。
除外した local binding/current source file を load の検証条件にしない。

初回に矛盾なしと確認された範囲は、旧 bytes/ID の保持、全最適対応と置換/削除の区別、未調査の null、方向別集計、Candidate-only の NO_HEAD、native snapshot とローカル設定の分離である。

## 変更箇所の独立再確認

Reviewer verdict: **PASS**。

> The prior P1 is resolved by ADR 0033 §2.4, line 88: seal REF now explicitly copies Candidate v7 origin to Provenance v3 exactly, preserves null, forbids regeneration or silent loss, validates Material content equality and closure before REF CAS, and performs post-publication readback/reconciliation.
>
> The prior P2 is resolved by ADR 0033 §2.5, lines 102–104: dump now explicitly captures one coherent repository state, while load validates only staged canonical repository/input bytes and explicitly does not require excluded local bindings or current source-file observation.
>
> No remaining concrete concern was found within the focused changed clauses.

この PASS は変更箇所の確認であり、所有者による詳細契約の承認でも実装の動作証明でもない。

## 調整者の確認と残る判断

- ADR 0032 D1、R1、R1 承認記録の SHA-256 を照合し、承認済み bytes を変更していないことを確認した。
- 相対リンク、文書間の参照、index、空白差分を確認した。
- コードと canonical repository state を今回変更していない。製品テスト・実 migration・native dump/load は実行していない。
- CLI LLMThink による継続方針/所見処置の監査は fatal 0 / error 0 / warning 0。未決判断は Proposed として残した。監査は仕様の承認を代替しない。
- 下流作業は schema/version、公開 command/output、明示 budget、native transport の候補採否を待つ。対象 workload/platform と性能合格基準は未設定であり、テスト都合で決めない。
- 新しい native dump の提案は無参照も含めた全 Blob を出力する。既存の format4 移行用 dump/load の読み替えではない。保持範囲と explicit load 境界を ADR 0033/0035 の判断対象として示した。
- 実現した機能利用価値はまだない。今回の寄与は、承認済み論理設計から詳細保存・比較・操作契約へ進み、二つの曖昧な境界を実装前に明文化したことである。
- 執筆を Luna 2 タスクに分担し、別の Luna 1 タスクで独立レビューした。課金・総トークン費用と作業時間は未計測。

## レビュー後の所有者判断：status 表示の保留

2026-09-17、所有者は status への表示追加を一旦保留し、詳細側で試して必要性が見えたら戻るよう指示した。
ADR 0035 §4.2/§6 をこの判断に合わせ、§8 に発言と範囲を記録した。変更は status の扱いだけであり、コード・schema・比較アルゴリズムを変更していない。
更新後 ADR 0035 SHA-256：`3546777194a10d6ef1ce2c21b1b4a2913591bb961c7cbeb4d2d75941c623cc00`。上記の独立レビューは変更前の対象 bytes に対する結果として保持し、この更新後の bytes を独立レビュー済みとは扱わない。
