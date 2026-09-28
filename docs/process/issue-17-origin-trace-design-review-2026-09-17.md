# Issue #17 範囲 Trace 設計 D1 — 独立レビュー記録

- 日付：2026-09-17
- 対象：[ADR 0032 D1](../adr/0032-content-origin-trace-and-directional-observation.md) 全体（日本語本文）
- 対象 SHA-256：`072314438483af19d2eb9b84b36d6a5b15d1ed358068f1849da473f34a3c38d0`
- 状態：Proposed。独立レビューは所有者による設計承認ではない。
- Reviewer：独立 `gpt-5.6-luna`、read-only、タスク `/root/design_review`。
- Scope：承認済み R1、関連 ADR 0024/0025/0028/0030/0031 との整合。保存 closure、対応推定の限界、Candidate/exact graph、方向別集計と STALE の独立性。
- 判定：`PASS_WITH_FINDINGS`。P0/P1 および現段階の修正を要する欠陥は確認されなかった。
- レビュー後の ADR 本文変更：なし。レビュー時と最終確認時は同じ対象版。

## 所見の扱い

Reviewer は以下を P2 の follow-up questions として報告した。重大度のラベルだけを根拠に、新しい欠陥・要件・承認ゲートに変換しない。

| 所見 | 調整者の照合・処置 | 所有する段階 |
|---|---|---|
| 1. 後継形式の構成 | ADR §10/§12 A1 は現 format 6 の閉じた schema を変更せず後継契約を要求している。形式の厳密な合成・歴史 reader matrix はまだ未決。現段階の矛盾ではない | 保存形式 companion A1 |
| 2. Candidate のグラフ表示 | R15 は overlay の採否を未決としており、ADR §7.1/§11 は非 overlay 案と制約を明示している。機械出力でも HEAD なしを明示する | CLI/output companion A3、設計所有者の案の採否 |
| 3. 推定アラインメント | ADR §5 は実編集履歴の証明ではないことと根拠種別を明示。方式の採否は §11 の判断点として保持。レビューによって採用済みにしない | 所有者判断、厳密な詳細は A2 |
| 4. 方向別集計 | R14/R17 と照合し、各 Seal 自身の結果だけを Cause 方向で集計する設計を確認。修正要求ではなく確認結果として記録 | §7 の設計、後続 A5 の適合検証 |

上記の未決事項は実装開始前に必要な詳細契約であり、この文書だけで実装可能な wire/CLI 契約一式が完成したとは主張しない。
レビューのために保存形式の番号、性能閾値、意味論の監査、Assessment 義務を追加しなかった。

## Reviewer 原文

> Verdict: **PASS_WITH_FINDINGS**  
> Verified candidate SHA-256: `072314438483af19d2eb9b84b36d6a5b15d1ed358068f1849da473f34a3c38d0`
>
> No P0 or P1 defects found. The ADR preserves accepted R1 authority and the accepted Cause, STALE, immutable Blob, binding, and format compatibility boundaries.
>
> P2 review findings / follow-up questions:
>
> 1. **Successor format composition remains a required companion decision, not a current defect.**  
>    ADR 0032 §3.1 says `Provenance = 既存項目 + origin` and §10 says successor Provenance/Seal/Candidate schemas are required (lines 63, 253–263). ADR 0030’s accepted Provenance v2 is already an exact closed schema with unknown members rejected (ADR 0030 lines 62–75, 93–116). The companion must explicitly define whether origin is a further successor after v2 or a combined successor, and the historical reader/migration matrix. The ADR correctly leaves this undecided.
>
> 2. **Candidate graph behavior is an explicit option, not an authority error.**  
>    The design chooses Candidate-only self comparison and excludes Candidate Cause overlay from directional graph aggregation (0032 lines 174–182, 272). R15 explicitly leaves Candidate inclusion undecided, so this is aligned. The CLI/output companion must make the limitation and “HEADなし／グラフ観測対象なし” result machine-visible; it must not be presented as complete Candidate upstream/downstream analysis.
>
> 3. **Inferred alignment is appropriately labeled as inference, with ambiguity preserved.**  
>    The minimum edit model and all-optimal-solutions criterion (0032 lines 121–138) do not overclaim edit-history or semantic truth. `inferred_alignment`, `declared_correspondence`, and `identical_snapshot` are separated. Adoption of this algorithm as the required policy remains an owner decision under §11 and must stay that way.
>
> 4. **Directional aggregation is structurally independent and does not launder STALE.**  
>    Cause-target edges alone drive aggregation; previous-revision assertions contribute only to the observed closure (0032 lines 169–172, 184–203). Per-Seal observations are aggregated directly, avoiding recursive aggregate reuse (lines 186–196). Trace outcomes remain separate from existing STALE derivation (lines 162–163, 212–213, 249–263). This matches R14/R17 and ADR 0028’s directional Cause semantics.
>
> No requirement contradiction, false certainty, or structural-closure defect was established. The pending review-record link is appropriately treated as expected process state, not an ADR issue.

## 調整者の確認と証拠の限界

- 承認済み R1 と承認記録のファイルを SHA-256 と exact Seal の raw content の両方で照合し、変更なしを確認した。
- ADR の要件対応表に R1～R19 と AC1～AC11 を対応付けた。これは設計の追跡可能性であり、実装の合格結果ではない。
- 文書の相対リンクと空白差分を確認。製品コードを変更していないため、製品テストによる機能検証は実施していない。
- CLI LLMThink で設計方針と所見処置を監査。最終結果は fatal 0 / error 0 / warning 0 / info 1 / hint 2。
  未決判断がある点を Proposed として保持した。二つの decision が同じ未決根拠を共有するという hint は、提案作成とレビュー後の提示という段階差であり、相反する採用決定をしていないことを照合した。
  audit は思考の構造検査であり、仕様適合・承認・外部事実の証明ではない。
- 成果はローカルの論理設計案とレビュー記録。機能はまだ利用できない。必要な残工程は ADR §12 に記載。
- 独立した制約抽出に Luna 1 タスク、設計レビューに Luna 1 タスクを使用。実際の課金・総トークン費用は未計測。
