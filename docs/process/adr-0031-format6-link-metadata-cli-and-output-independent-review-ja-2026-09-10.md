# ADR 0031 Format-6 Link metadata CLIとoutput 独立レビュー（日本語）— 2026-09-10

この文書は英語の独立レビュー記録全体に対する日本語のreview supportです。
英語記録が正本であり、この翻訳は第2の権威あるdecision artifactではありません。

## レビュー対象の識別

- 対象: `docs/adr/0031-format6-link-metadata-cli-and-output.md`
- 段階: `Proposed`
- 期待するSHA-256:
  `f3082d68aad0638e3af5c28f5199a4644490af8b3d12a846281d1091185eff9f`
- review前digest: 一致
- review後digest: 一致
- reviewer role: 1名の独立したread-only repository-wide ADR reviewer
- 判定: `FAIL`

このreviewはcandidateを変更せず、ADR 0031をacceptせず、normative conversion、
implementation、migration、commit、push、release、deployment、その他のexternal
writeを承認しません。

## Findings

### P1 — `linklog/v3`は投影されたformat-5 metadataを識別できない

**acceptance blocking: はい。**

#### Claim

提案された`linklog/v3` recordは、`metadata: []`へ投影されたhistorical format-5
Cause Linkと、実際に空のmetadata arrayを保存するformat-6 Cause Linkを区別できません。

#### Evidence

- Accepted ADR 0030 204–208行は、successor inspection schemaを通じて公開される
  historical v5 Linkを、Provenance v1に存在したと主張するbytesではなくhistorical
  projectionとして明示的に識別するよう要求しています。
- ADR 0031 177–181行は同じ要件を繰り返し、enclosing schema fieldがprojectionを
  識別すると述べています。
- ADR 0031 193–199行の`cause_link_change`には、before/afterを所有するSealまたは
  Provenanceのschema generationがありません。217–219行では`linklog`が同じrecordを
  使用します。
- ADR 0031 245–247行は、substituteされるshared recordを除き、ADR 0027の既存の
  top-levelおよびentry shapeを維持します。
- Accepted ADR 0027 465–470行の`linklog_entry`にはnewer/previous Seal IDがありますが、
  newer/previous SealまたはProvenanceのschema fieldはありません。
- current implementationもownership関係を確認できます。`history.go` 180–197行は
  newer/previous endpoint Sealが所有するCause Linkを比較し、`inspection_json.go`
  540–575行はchange recordの周囲にそれらのIDだけを出力します。

#### Reasoning

`linklog/v3`のtop-level schemaはoutput document versionを識別するものであり、比較する
各endpointのstorage generationを識別しません。supporting assertionはassertion
observerを識別し、target revision observationはnewer Sealをtargetとするassertionを
記述します。どちらも、比較される各Cause Linkを所有するProvenance generationを
確実には識別しません。

したがって、consumerが`before.metadata: []`または`after.metadata: []`を見ても、
stored format-6 dataなのかlogical format-5 projectionなのかを判断できません。
これはADR 0030のexplicit projection contractと、enclosing schema fieldがその区別を
提供するというADR 0031自身の記述に違反します。

#### 因果分類と修正境界

- **root cause:** CLI machine-schema designが、projection provenanceの根拠となる
  endpoint generation contextを持たせずに、typed Seal/Candidate viewの外側で
  `cause_link_change`を再利用したことです。
- **contributing cause:** 空のmetadata arrayの意味は所有するProvenance generationに
  依存するにもかかわらず、shared change recordをself-containedとして扱ったことです。
- **escape or detection cause:** author self-reviewはschema-version bumpとhistorical
  projectionを別々に確認しましたが、exact `linklog_entry` topologyを通してprojection
  provenanceを追跡しませんでした。test不足はdefectを作った原因ではありません。
- **corrective action:** 各`linklog/v3` before/after sideを、そのexact endpoint Sealまたは
  Provenance generationへ曖昧さなく関連付けるようcandidateを改訂し、変更後のexact
  member orderを規定します。
- **recurrence-prevention action:** 一方の`[]`がprojected、他方がstoredとなる
  v5-to-v6およびv6-to-v5の`linklog` fixtureと、すべてのprojected recordに対する
  topology checkを要求します。

command-line LLMThinkのcausal auditはfatal、error、warningがすべて0でした。
不確実性は低いです。

#### `REVISE`後のcoordinator disposition

`Current correction required`を採用しました。改訂candidateは、completeなnewer/previous
Seal/Provenance identityとschema generationを`linklog/v3` entryごとに一度追加し、
`before`はprevious、`after`はnewerが所有すると定義し、mixed-generation projection
fixtureを要求します。このdispositionはoriginal findingまたはverdictを変更しません。
revised candidateのSHA-256は
`82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47`で、focused
rereviewが必要です。

## 追加findingがなかったreview領域

### Legacy authoring continuity

**結果: Pass。** 既存format-6 `add`/`link` syntaxでmetadataを保存することは、整合した
successor refinementです。structural previous/message replacementを維持しつつ、
omissionが新しいextension areaを削除することを防ぎます。explicit `unlink`とroot
clearingは、明示されたwhole-Link scopeを引き続き削除します。

### Metadata mutation

**結果: Pass。** `link-metadata set`と`remove`は、exact one-target、one-namespaceの
Candidate mutation、expected-old replacement前のcomplete validation、rejection
atomicityを提供します。publishもmigrateもしません。

### Schema matrixとcomparison

**結果: P1の`linklog` defectを除きPass。** schema matrixは、それ以外の変更された
shared recordを網羅しています。`status/v3`、`stale/v2`、narrow protocolの維持は、
それらの意味が変わらないことと整合します。Candidate comparisonとimmutable
comparisonはenclosing Candidate/Seal schema fieldを持つため、`linklog`と同じ曖昧さは
ありません。

### Human output

**結果: implementation limitation付きでPass。** terminal-safety violationは確立
されませんでした。candidateはcanonical JSON escapingを要求し、control characterを
literalに出力せず、ADR 0030によりcomplete metadata arrayを65,536 canonical bytesへ
制限し、ADR 0022のwidth-safe behaviorを維持します。implementationは最大value sizeで
width-safe renderingを実証する必要がありますが、このverification needは第2のdesign
findingではありません。

### Migrationと除外scope

**結果: Pass。** migration syntax、config-only transaction、receipt、automatic retryを
行わないbehaviorはADR 0030と整合します。query language、registry/validation、ADR 0028
Assessment commandは正しく除外されています。downstream `impact/v3`はADR 0028の別の
`upstream-impact/v1` contractと区別されたままです。

## 未解決の質問

修正designでは、endpoint generationを`linklog_entry`ごとに一度持たせるか、各
`cause_link_change` sideへ個別に持たせるか、または別のself-contained typed wrapperを
使うかを選ぶ必要があります。これはADR revision decisionであり、reviewerによる編集
ではありません。

## Coverage

reviewは次を対象にしました。

- ADR 0031のdefinition、alternative、consequence、review claim、
  Claim/Evidence/Action traceability
- current requirements、architecture、storage-format、CLI document
- accepted ADR 0022、0023、0027、0028、0029、0030
- accepted revision-2 requirementとそのacceptance record
- current-fit evidenceとしてのcurrent Candidate lifecycle、expected-old persistence、
  history comparison、machine-output、human-output implementation

## Reviewer結論

exact candidateの判定は`FAIL`で、acceptance-blockingなP1 findingが1件あり、P0、P2、
P3 findingはありません。ADR 0031は`Proposed`のままです。適切な次のlifecycle actionは
`REVISE`であり、改訂したexact candidateを改めてreviewします。
