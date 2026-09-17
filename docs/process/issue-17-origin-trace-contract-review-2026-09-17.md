# Issue #17 — CONTRACT_REVIEW 所有者判断資料

- 日付：2026-09-17
- 状態：採否待ち。レビュー資料の作成は契約の承認ではない。
- 対象：ADR 0033～0035 の現行全文。3文書とも日本語であり、以下の要約だけでなくリンク先の全文が採用対象。
- 上位：Accepted Issue #17 R1 と ADR 0032 D1。
- 対応計画：[Issue #17 PERT](issue-17-origin-trace.pert) の CONTRACT_REVIEW。

## 目的と既承認部分

小さな Atomic Seal の内容と、大きな原文中の範囲との対応を記録し、現在の原文との差異を追えるようにするため、既承認の論理設計を保存・比較・操作の具体的契約にする。

Cause Link は Seal 全体を指し、STALE の定義は変えない。意味論は監査しない。自身・上流・下流の原文比較を分離し、Candidate 自身と HEAD graph も分離する。status 表示は詳細側で試して必要性が見えたら戻る、という所有者判断を維持する。

## 今回採否を求める具体化

| 対象 | 採用候補 | 利点・負担 | 代案と影響 |
|---|---|---|---|
| [ADR 0033 保存形式](../adr/0033-origin-trace-storage-and-migration.md) 全文 | format 7、OriginMap/SourceSnapshot、Seal/Candidate/Provenance の後継 schema、正規 bytes と構造検証。旧形式 5/6 の bytes/ID を保持する明示移行。Candidate の exact origin を Seal 公開時も保持 | 由来の immutable identity と公開時の保持条件が明確。後継 reader・移行対応が必要 | 別の schema/版境界なら本 ADR を修正して再確認する。旧 bytes を書き換える代案は現在案の互換境界と異なるため、別判断が必要 |
| [ADR 0034 対応判定](../adr/0034-origin-trace-correspondence.md) 全文 | 全文 byte 列の最小挿入・削除モデルで、全最適対応が対象範囲について合意した場合だけ自動確定。内部編集・削除・境界競合を区別。固定した二版への明示対応宣言を許す | 任意の diff 経路を由来と断定しない。反復や境界変更では判定不能が残り、利用者の宣言が必要になることがある。計算負担は未測定 | 合意を取るモデル自体を変更するなら、既承認 D1 への影響を整理して改訂する。単一路線の tie-break を確定扱いにすることは現行 D1 に適合しない |
| [ADR 0035 CLI・出力](../adr/0035-origin-trace-cli-and-observation-output.md) 全文 | 専用 trace 操作、原文 binding、対応宣言、自身・上流・下流の詳細出力、machine schema の版更新。比較時に alignment cells / graph visits の予算を明示必須とする | 部分観測・未調査・差異・判定不能を区別できる。一方、操作と必須引数が増え、初回指定の負担がある | 既定予算を導入するなら、利用条件と値の決め方を定めたうえで ADR 0035 を修正する。性能保証値とは区別する |

### 独立して明示する追加提案：native dump/load

ADR 0033 §2.5 と ADR 0035 §4.3 は、format 7 用の新しい `native-blobs-v1` dump/load を含む。既存の移行用 dump/load の呼び替えではない。

- dump は **参照中のものだけでなく、無参照分を含む保持中の全 valid Blob**、REF/tag、Candidate を出力する。現在の HEAD から到達しない過去・途中の内容も出力対象になる。
- 原文全文 snapshot も含むため、出力量は抜粋範囲の大きさだけでは決まらない。local binding、対応宣言、cache/log/lock は含めない。
- load は `.sealgraph` が存在しない宛先だけに検証済み staging を公開する。既存 store との merge/overwrite は行わない。
- 利点は exact bytes と保持 inventory を移せること。負担は出力量、および load 後の local binding 再設定。
- 到達可能な Blob だけを移す代案は出力量を減らせるが、無参照 Blob を保存するこの transport 契約とは異なる。native transport を保留する代案もあるが、採用範囲と依存する設計・計画を明示的に修正する必要がある。

**採否は、この全 Blob transport を含めるかを明示して行う。** 契約の採用は、現 repository の移行、dump/load 実行、外部への転送を許可するものではない。

## 証拠と限界

[独立レビュー記録](issue-17-origin-trace-companion-review-2026-09-17.md) では、Candidate→Seal の origin 保持と load の検証対象の二点を明文化し、変更箇所の再確認は PASS。ADR 0035 のその後の status 保留反映は所有者指示に基づく変更であり、更新後全文を独立レビュー済みとは扱わない。

今回の整理方針は command-line LLMThink で監査し、fatal/error/warning は各 0。これは論理整理の証拠であり、採用、実装適合性、性能の証明ではない。

対象 workload、platform、資源条件・性能基準は未決で、ENVIRONMENT_REVIEW に残る。比較の計算量と実用上の原文サイズの関係は実測していない。仕様採用後も、この判断を終えるまで PERT 上の実装開始条件は揃わない。

## 採否対象の版

レビュー用の名称は上記の文書名と日付とする。以下は exact bytes の識別記録。

| 文書 | SHA-256 |
|---|---|
| ADR 0033 | `c71d26a77779ddb3ce744c738c6f981ab52f0291a39828d91163e3dddce327fc` |
| ADR 0034 | `070bfff4da77bf2db22b22849ef6cd9c0fc3064e470f4940c3290a8858554f66` |
| ADR 0035 | `3546777194a10d6ef1ce2c21b1b4a2913591bb961c7cbeb4d2d75941c623cc00` |

## 推奨と所有者への判断依頼

現行3文書の採用を提案する。特に、**比較予算の明示必須**と、**無参照分を含む全 Blob native dump/load**を含む採用かどうかを判断いただく。変更希望があれば、その部分を Proposed のまま改訂し、影響する下流作業を保留する。

現時点の機能利用価値は未実現。今回の寄与は、既承認設計と追加の詳細契約を分け、所有者が具体的な負担と範囲を見て採否を決められる状態にしたこと。製品コード、既承認文書、PERT 状態は変更していない。費用・実装所要時間は未計測で、利用可能時期は残る実装・検証・提供条件の完了後となる。
