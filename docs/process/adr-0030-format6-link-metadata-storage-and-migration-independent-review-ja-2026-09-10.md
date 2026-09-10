# ADR 0030 Format-6 Link metadata storageとmigration 独立レビュー（日本語）— 2026-09-10

この文書は英語の独立レビュー記録全体に対する日本語のreview supportです。
英語記録が正本であり、この翻訳は第2の権威あるdecision artifactではありません。

## レビュー対象の識別

- 対象: `docs/adr/0030-format6-link-metadata-storage-and-migration.md`
- 段階: `Proposed`
- 期待するSHA-256:
  `b879af866be94106c8a7675a1477e9ccfabfdc823058c4b3bb7157aa757f198e`
- review前digest: 一致
- review後digest: 一致
- reviewer role: 1名の独立したread-only repository-wide ADR reviewer
- 判定: `PASS`

このreviewはADR 0030のaccept、Status変更、companion decision、normative
conversion、implementation、migration、commit、push、release、deployment、
その他のexternal writeを承認しません。

## Findings

P0、P1、P2、P3 findingはいずれもありません。

design defectが確立されていないため、source-cause phaseは該当しません。

## レビュー質問

### 1. Format 6内でのhistorical v5 reading

**結果: Pass。** ADR 0026は、format-4の非互換性をglobal `parent_revision`から
Cause-scoped assertionへの置換、すなわち異なる二つのgraph authorityを抱える
こととして識別しています（ADR 0026 26–49行）。Accepted ADR 0029はこれと異なり、
既存のtargetとprevious-revision fieldだけからgraphを導出し、metadataをstructurally
opaqueに保ちます（ADR 0029 125–136行）。

ADR 0030はexact v5/v1とv6/v2 typed pairingだけをacceptし、v5 readerを
schema-exactかつread-onlyにし、両方へ一つのADR 0023 graph meaningを適用します
（ADR 0030 182–203行）。二つのgenerationはbytesが異なりますが、競合するCause
またはrevision semanticsを持ちません。これは限定されたcompatibility readerであり、
format-4 mixed graph authorityへの回帰ではありません。

### 2. Typed-schema transition

**結果: Pass。** ADR 0030はSeal v6、Provenance v2、Candidate v6を導入し、
v6-only writeを要求し、member shapeによるschema推測を禁止します（53–67行）。
successor recordを確定し、すべてのsuccessor Linkで`metadata`をrequiredにします
（68–118行）。Seal/Provenance cross-pairingを禁止し（184–193行）、変更された
assessment-free preimageへ`upstream-change/v2`を割り当てます（274–293行）。

Accepted ADR 0025はexact typed-schema validation、canonical re-encoding、exact
byte equality、unknown-member rejectionを要求します（ADR 0025 82–89行、93–128行）。
意味が変わるpersisted typeにはすべて新しいidentifierが与えられます。Material v1、
Blob identity、REF manifest v1の意味は変わらないため、不必要なversion変更はしません。

### 3. Config-only migration

**結果: Pass。** Accepted ADR 0025はexact Candidate bytesをmutable optimistic-version
stateとして確立しています（ADR 0025 216–244行）。ADR 0030はCandidate v5をmutation
なしでreadし、別途authorizedされたCandidate mutationまたはpublication projection
のときだけ変換します（205–222行）。

migrationはwriter guard、exact format-5 config、complete fsck、REF・Candidate・
object inventoryのdouble capture、configだけのatomic replacement、file/directory
synchronization、complete format-6 reopen/readbackを要求します（224–252行）。Blob、
REF、tag、Candidate、metadata、Sealのmutationを明示的に禁止し、ambiguous durability
またはreadbackをautomatic retryなしのuncertainとして扱います。

したがってinterruption後は、multi-file semantic rewriteの途中ではなく、exact
format-5 configまたはexact format-6 configになります。両側とも、それぞれの
transition側で適用可能なuntouched v5 objectとCandidate bytesをreadできます。

### 4. Canonical valueとresource dimension

**結果: Pass。** Accepted ADR 0029はinteger representation、encoding、ordering、
depth、node、string/key、entry、byte boundをこのcompanionへ委ねています
（ADR 0029 99–123行）。ADR 0030はentry order、namespace order/uniqueness、signed
64-bit integer range/spelling、scalar encoding、array significance、object-key
ordering/uniqueness、exact byte revalidationを確定します（99–155行）。

また、entry、namespace、schema、nesting、node、key/string、complete metadata-array
byte limitを確定し、truncationまたはpartial persistenceなしのbounded validationを
要求します（157–180行）。valid historical messageまたはLink countを遡及的に
invalidにしうる新しいcontaining-object limitは設けないと明示的に決定しています。
したがって、新たに独立して変動するresource dimensionはboundedであるか、既存の
compatibility dimensionとして維持することが明記されています。

### 5. ADR 0028 change identity

**結果: Pass。** Accepted ADR 0028はassessment-free change preimageを定義し、
すべてのCause Linkをidentity-bearingとして含め、self-reference回避のためAssessment
referenceを除外します（ADR 0028 174–197行）。historical Sealを遡及的にinvalidに
したりAssessmentをfabricateしたりすることも禁止します（352–358行）。

ADR 0030はpreimage member orderを維持し、complete format-6 Cause Linkを使い、
変更された意味を`upstream-change/v2`へ移します（274–287行）。既存immutable v1
change/Assessment Blobを保存し、Assessment-reference representationを別途gatedな
ADR 0028 storage successorへ委ねます（289–293行）。metadataは`change_id`を
変えますが、AssessmentID、adoption path、disposition、fabricated migration stateは
このsliceへ入りません。

## Uncertaintyとlimitation

repository process writer guardは、supported protocol外のdirect filesystemまたは
outer-VCS mutationをcontinuousにmonitorしません。ADR 0023 158–171行はそのような
mutationをunsupportedとし、207–221行はcontinuous monitoringを約束せずobservation
claimを限定しています。supported writerはguardでserializeされ、既存Blobはimmutable
のままです。このlimitationはverdictを変更しません。

## 未解決事項

次は別途gatedであり、ADR 0030のdefectではありません。

- exact migration CLI spelling
- public human outputとversioned machine output schema
- persisted Assessment reference、closure、historical Assessment handling
- implementation fixture
- normative-document conversion

## Coverage

reviewは次を対象にしました。

- ADR 0030のinternal definition、alternative、consequence、migration step、
  review question、Claim/Evidence/Action traceability
- accepted ADR 0023、0025、0026、0027、0028、0029
- accepted revision-2 requirementとacceptance record
- current requirements、architecture、storage-format、CLI document
- current-fit確認に必要なformat-5 config、fsck、Candidate、writer-guard implementation

expected CLI/output workと、別途gatedなAssessment-reference storageは、指示どおり
defect scopeから除外しました。

## Reviewer結論

exact candidateの判定は`PASS`で、P0からP3のfindingはありません。ADR 0030は、
Operatorによる別の`ACCEPT`または`REVISE` decisionまで`Proposed`のままです。
