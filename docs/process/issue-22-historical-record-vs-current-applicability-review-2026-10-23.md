# Issue22: 発行時の判断記録と現在のapplicableの読取り照合

2026-10-23 JST。静的レビューのみ。対象はSealGraph main
`7519fbf048e6f2d238790d55f0a13ee814ebcb9e`。
参照したコード・ADR・CLI仕様の11ファイルがそのcommitのbytesと一致することを確認した。
kn側の前提は `08e54e2` の
[境界記録](https://github.com/mako10k/kn/blob/08e54e2/docs/reviews/kn-provenance-checker-historical-review-gap-2026-10-23.md)。
同commitのcheckerと記録は現在の実ファイルとも一致していた。

## レビュー境界

問いは、保存した特定版依拠の判断と、現在のHEAD観測に対する適用可否を、
現実装からどこまで別々に確認できるか。以下の反例はコードからの静的演繹であり、
実行結果ではない。テスト、build、実repoのSeal操作は実施していない。

INSIDE: CLI構文、出力schema、対象版の拘束、other_headsの比較、記録から
証明できること／できないことと、小さい反例。
OUTSIDE: CLI・kn checker・設計・受入基準の修正、過去の判断の再承認、
Seal/Cause/registry/remote/PERTの変更。
BOUNDARY_DISPUTE: 今回の読取り範囲ではなし。knがどの証拠を受入可能とするかは
本資料で決定しない。

## 結論

`record` は選択したSeal内に保存された判断の主張、`current` は今のHEAD観測で
再計算した事実、`applicable` はその両者を今照合した結果である。
別REFを次に発行しただけでも `applicable=false` になる。以前の判断記録を
書き換えたり、その判断が当時誤っていたと判定したりする動作ではない。
これはADR0043で明記された保守的な拘束と一致する。

ただし、現CLIに「発行当時のレビュー検査に合格した」という独立の結果項目はない。
**記録の存在・型の妥当性、対象への結び付き、当時のsnapshot上の整合、
当時実際にreview経路で発行した証拠、判断内容の正しさは別々である。**
`applicable=false` を過去の判断の無効と読み替えることも、記録があるだけで
発行当時の検査合格・理解・承認を認定することもできない。

## 正しいCLI構文とschema

7519fbfの構文は次の形。説明用であり、このレビューではコマンドを実行していない。

```text
sealgraph review show @<完全なSealID> --format json
sealgraph review show <REF> --candidate --format json
sealgraph show @<完全なSealID> --format json
```

`review show --seal ...` は未実装。`--seal` は別の `trace show` 等の構文である。
発行済みの特定版には `--candidate` を付けず、movable REFではなく完全なSealIDで
選択する。Candidate選択は、未存在の場合にHEADへfallbackしない。
根拠: [CLI parser](../../internal/cli/historical_review.go#L25)、
[CLI仕様](../../docs/cli.md#L710)。

出力は `sealgraph/historical-review-show/v1`。

| field | 実装での意味 |
| --- | --- |
| `selected_seal` | 選択した不変Seal、またはCandidateのprospective Sealの完全ID |
| `candidate` | 前者ならfalse、後者ならtrue |
| `record` | 選択対象の専用添付Blobをdecodeした保存済み主張 |
| `current` | 現在の全HEAD観測に選択対象を当てはめ、レビュー添付を除いたsubjectとCause閉包を再計算した結果 |
| `applicable` | 現在のbinding/coverage照合、全件retain、Causeのdraft不在が成立し、subject自身もnon-root/non-draftの場合true |
| `limitation` | 不成立時の最初の検査理由と、実行者の主張であり理解・正しさ・承認の証明ではない旨 |

`record` のschemaは `sealgraph/historical-review/v1`、fieldは
`schema,ref,subject_seal,other_heads,entries`。entryは
`target_seal,compared_seals,decision,difference,impact,reason`。
`current` は `sealgraph/historical-review-prepare/v1` の形で、
`schema,ref,subject_seal,other_heads,targets`。targetは
`target_seal,compared_seals,revision_state,draft`。
根拠: [inspection型と実処理](../../internal/repository/historical_review_inspection.go#L15)、
[record型・decode](../../internal/historicalreview/record.go#L18)、
[事実型](../../internal/repository/historical_review.go#L17)。

## 三段階以上に分けて確認する

### 1. 不変記録が何に結び付いているか

完全SealIDからMaterial/Provenance/添付を読む。専用添付の名称は
`sealgraph-historical-review.json`、media typeは
`application/vnd.sealgraph.historical-review+json`。
Blobのhashと型・内容／OriginMapを読む境界、記録の厳格なdecodeは存在する。
根拠: [LoadSeal](../../internal/repository/repository.go#L518)、
[添付読取り](../../internal/repository/historical_review_inspection.go#L25)、
[Blob hash検査](../../internal/store/native/object.go#L142)。

`record.subject_seal` は発行済み `selected_seal` そのものではない。
専用レビュー添付だけを除いて再計算したprospective SealIDである。
本文・他添付・Cause全体・root/draft・OriginMap・世代を拘束する。
同じ世代なら `record.subject_seal == current.subject_seal` を個別に確認でき、
other_headsの不一致とは切り分けられる。ただしこれは一項目の照合であり、
過去の完全なadmission合格を意味しない。
根拠: [添付除外とsubject導出](../../internal/repository/historical_review.go#L31)。

さらに、全direct/transitive Causeの正確な対象IDの網羅、各decision、差分・影響・
理由、draft不在を確認する。型decodeは文字列の内容が意味上正しいことを判断しない。
記録の添付Blob IDは通常の `show` のattachmentsから確認する。
`review show` のJSON再表示をそのままhashして元Blob IDとみなさない。

### 2. 発行時のsnapshot／実行が何で裏付けられるか

reviewed経路の実装は、記録を読んで当時観測したheads上でbindingとcoverageを
検査し、全件retain・draft不在を確認してから、通常の構造検査・観測再検証・
一対象のCASを通って発行する。
根拠: [admission](../../internal/repository/historical_review_inspection.go#L43)、
[経路選択とgraph検査](../../internal/repository/observation_v5.go#L266)、
[発行・CAS・読戻し](../../internal/repository/repository.go#L354)。

しかし、保存したSealやrecordには、`published_with_review`、
`historical_admission_passed`、発行時刻・実行者・expected-old HEADは保存されない。
CLI成功receiptも通常／reviewedで同じ形で、recoveryのoperation種別も `seal`。
したがって「記録が読めるので当時reviewed経路で発行した」は成立しない。
実際の経路・成功は、コマンドとexit status、完全SealIDへの読戻し、必要な
発行前の観測資料など別の実行証拠と対応付けて報告する必要がある。
根拠: [Seal型](../../internal/domain/v5/model.go#L18)、
[CLI分岐・共通receipt](../../internal/cli/run.go#L1176)、
[共通recovery記録](../../internal/repository/repository.go#L400)。

recordのother_headsと、記録されたsubjectを用い、当時の世代／同じ規則で
immutable閉包を再構成すれば、記録内snapshotに対するcomparison集合・網羅・
retain・draft不在の**内部整合**を再計算する材料はある。
対象自身の旧HEADはprospective subjectで置き換えるため、このレビュー事実の
再構成には使わない。一方、これは当時実際に全REFを完全観測したことや、
CASが成功したことを記録だけで証明する方法ではない。

現在の `review show` には「record.other_headsを観測源として検証する」モードも、
当時の合否を独立に返すfieldもない。この再構成を現CLIで検証済みとは報告しない。
当時snapshotを保存した読取専用環境と実行receiptがなければ、当時の完全な検査
成立は、その不足を未知として残す。新しいscratch構築や検査追加は本依頼では行わない。

### 3. 現在の観測へその判断を適用できるか

実装は現在の全REF/headから事実を生成し、対象自身のREFのみをother_headsから
除外する。他のREFは依存関係の有無にかかわらずすべて対象。
記録と現在のheadペア配列が一致しなければ
`REVIEW_OBSERVATION_CHANGED` を返す。
根拠: [現観測と除外](../../internal/repository/historical_review.go#L42)、
[完全比較](../../internal/repository/historical_review.go#L130)。

`InspectHistoricalReview` はそのerrorを `limitation` に入れ、recordを保ったまま
`applicable=false` を成功JSONとして返す。正常なfalseと、読取り・graph・観測競合で
command自体が失敗する場合を区別する。後者を「確認済みfalse」と扱わない。
またheads比較で先にreturnするため、そのerrorだけでは後段のcoverageやdraft等を
すべて検査済みとは言えない。
根拠: [表示判定と再検証](../../internal/repository/historical_review_inspection.go#L109)。

## 小さな反例1: 無関係な次発行で先のapplicableがfalse

以下は記号による静的例。IDは実在SealIDではない。実行していない。

- non-draftの旧 `P0` を保持し、現在の `P` は `P1`。最小化のためrevision assertionはなく、P0はdetached。
- 無関係なroot REF `Q` は `Q0`。
- non-root/non-draft `A` のCauseは正確なP0。Aの記録は全targetをretainとし、比較差分・影響・維持理由を記述。record.other_headsは `{P:P1,Q:Q0}`。compared_sealsはこの例では空集合。

| 段階 | 他HEAD | Aの不変記録 | A0をreview showしたapplicable |
| --- | --- | --- | --- |
| A0をreviewed経路で発行した直後 | P:P1, Q:Q0 | subject Aの特定内容、Cause P0、上記snapshot | 他条件が満たされていればtrue |
| 無関係なQをQ1へ明示発行した後 | P:P1, Q:Q1 | 同じA0内の同じBlob・同じretain判断 | false、REVIEW_OBSERVATION_CHANGED |

変わったのはQのHEADのみ。A0・P0・record・Aの判断対象は書き換わっていない。
新REF Bを次に作る場合にも、other_headsの集合へBが増えるため同様に不一致となる。
既存のactive非leafを維持する例でも、このheads完全比較の性質は同じ。

複数REFを順番に発行した後に、すべての発行済みrecordについて同時に
`applicable=true` を要求することは、一般にこの順序と両立しない。
false解消だけを目的に先のAを再Sealすると、今度は後のB等のsnapshotとずれ得る。
この循環を止めるための自動再Sealやsnapshotの無視は本資料では提案・実行しない。
ADR0043の[既知の帰結](../../docs/adr/0043-reviewed-historical-cause-publication.md#L36)
は、無関係なREF変更でも再確認を要し得るが歴史上の判断自体の無効化ではない、としている。

報告文の例（上の仮定例についてのみ）:

> A0はP0を維持する判断記録Rを不変添付として保持している。Rの対象内容・Causeとの
> 拘束は確認できた。別途保存した実行証拠により、当時のreviewed発行の合格を確認した。
> 現在はQ0→Q1によりother_headsが不一致でapplicable=false。今の観測への影響確認は
> このRでは裏付けられない。旧判断の誤りやA0の内容変更が判明したという意味ではない。

実行証拠がない場合、上の「当時の合格を確認した」は使わず、
「当時の検査合格はこの読取りだけでは未確認」と報告する。
recordのretainを勝手にunreviewedへ変更せず、保存された判断と現在の未確認を別項目にする。

## 小さな反例2: 添付の存在と全件retainだけでは過去の合格にならない

コード上可能な別の静的例。P0がactive leafの状態で、Aの内容Xに対する記録を
setした後、OriginMapのないAの内容をYへ明示変更する。addは他添付を保持する。
その後、reviewed指定のない通常sealはactive-leaf条件で発行でき、保存された
記録は依然retainでも、X向けsubjectとYのsubjectは異なる。
根拠: [addの状態保持と内容更新](../../internal/repository/repository.go#L159)、
[通常経路では記録gateを選ばない条件](../../internal/repository/observation_v5.go#L281)。
この例は新方式の回避を実施・推奨するものではなく、証拠の読み替えが成立しない反例である。

## 制限とknへの読み替え範囲

- `applicable=true` もfull publication preflight、原本の現在ファイルとの一致、意味上の正当性、理解・承認の証明ではない。
- format移行後のhistorical Sealに対しては、subject再計算が現在のrepository formatを使用する。世代の違いだけでもsubject不一致が起こり得るため、即座に保存記録の破損と断定しない。[世代選択](../../internal/repository/candidate_lifecycle.go#L117)。
- record.refは元の宛先を表す主張で、Sealのintrinsic ownershipではない。別REFへのaliasやrenameでrecordを書き換えない。
- STALEの事実は別表示のまま。staleゼロをreview成立へ、review添付をstale解消へ読み替えない。
- knの現在のcheckerは[165–167行](https://github.com/mako10k/kn/blob/08e54e2/scripts/requirements_materials.py#L165)でstaleを拒否し、[202–209行](https://github.com/mako10k/kn/blob/08e54e2/scripts/requirements_materials.py#L202)でregistry上の参照REFの現HEAD集合・理由と比較する。Issue22の不変記録に基づく歴史上の判断を判定する経路はない、というHub記録と一致する。
- kn側の新検査で当時の成立を認める証拠、今の未確認を残す表示、原本・registryとの照合条件は別の所有者判断／設計対象。このレビューから検査削除や新たな受入基準を確定しない。

INSIDEの照合結果: record/current/applicableの分離と無関係HEAD変更によるfalseは
仕様・コードが一致する。過去の検査合格を独立表示／再検証するCLI機能は存在せず、
その機能がある前提での運用は証拠不足となる。
OUTSIDE: その不足への機能追加、kn checkerの実装・受入変更は未実施。
BOUNDARY_DISPUTE: なし。実際のkn3件のレビュー成立・発行可否は判定していない。
