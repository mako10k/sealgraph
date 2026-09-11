# ADR 0029 拡張可能な Cause Link metadata 独立レビュー（日本語）— 2026-09-10

この文書は、英語の独立レビュー記録全体に対する日本語のレビュー支援です。
英語記録が正本であり、この翻訳は第2の権威ある決定文書ではありません。

## レビュー対象の識別

- 対象: `docs/adr/0029-extensible-cause-link-metadata.md`
- 段階: `Proposed`
- 期待する SHA-256:
  `3a4886671a44c7b34f8a09ce14da08b1f9445fd9cd97140b4cdf66e36ec8fd8c`
- レビュー前のdigest: 一致
- レビュー後のdigest: 一致
- reviewerの役割: 1名の独立したread-only ADR reviewer
- 判定: `PASS`

このレビューは、ADR 0029のaccept、Status変更、companion decisionの承認、
実装、normative文書への反映、migration、commit、push、release、deployment、
その他の外部writeを承認しません。

## Findings

P0、P1、P2、P3 findingはいずれもありません。

defect findingが確立されていないため、source-cause phaseは該当しません。

## レビュー質問

### 1. Semanticと責務のslice

**結果: Pass。** ADR 0029は、logical record、namespaceとschemaの所有者、
canonical valueの種類、graph-independence rule、Candidate mutationの性質、
inspectionの義務を確定しています。一方で、正確なbytes、数値limit、
migration、CLI grammar、machine schemaは、accepted companion ADRに明示的に
委ねています（ADR 0029 10–14行、43–113行、158–198行）。

これは、accepted requirementのlogical modelと、後続判断として残された
事項に一致します（requirement revision 2 85–141行、195–213行、240–253行）。
このsemantic sliceは単独でレビュー可能であり、companion decisionを先取り
していません。

### 2. 外部revision-context profile

**結果: Pass。** accepted requirementは、Sealgraphがprofileを保守するか、
外部exampleとするかをowner decisionとして明示的に残しつつ、generic mechanism
がどちらも支えられることを要求しています（requirement revision 2
240–253行）。

ADR 0029は、許された選択肢の一つを選んでいます。すなわち、外部の
versioned profile exampleであり、予約済みのpersisted core namespaceを作らず、
coreによるschema conformanceの約束もgraph semanticsも作りません（ADR 0029
72–87行、128–156行）。これにより、隠れたcore ownershipを作ることなく、
accepted capabilityを満たしています。

### 3. Structural graphとidentityの境界

**結果: Pass。** accepted structural modelは、exact targetとprevious-revision
assertionからCause relationとrevision relationを導出します（ADR 0023
43–70行、99–142行）。exact Provenance contentはSealIDによって推移的に
commitされます（ADR 0025 91–200行）。

ADR 0029はそれらのstructural inputを維持しながら、metadataの変更によって
ProvenanceID、SealID、assertion-source identity、comparison、exact output bytes
が変わることを正しく記述しています（ADR 0029 115–126行）。semanticを
解釈しないことと、bytesが不変であることを混同していません。

### 4. ADR 0028 Assessmentとの境界

**結果: Pass。** ADR 0028は、独立したAssessment Blob、AssessmentID adoption、
exact change binding、evidence、disposition、admission stateを要求します。
また、これらをmessages、attachments、generic Cause Link metadataで模倣する
ことを明示的に退けています（ADR 0028 55–104行、165–207行、259–368行）。

ADR 0029はこれらAssessmentの性質をすべて除外し、Cause Link bytesの変更が
ADR 0028のchange identityへ与える影響をstorage companionで扱うよう要求
しています（ADR 0029 152–156行、181–190行）。

### 5. Proposed ADR 0017の扱い

**結果: Pass。** ADR 0017は現在もProposedであり、generic metadataをformat-4
Linkの前提、予約済みvirtual namespace、SealGraphQLと結合しています
（ADR 0017 3–42行、リンクされたproposal 60–178行、224–240行）。

ADR 0029は、分離可能なgeneric metadataのrationaleだけを採用し、query
languageとformat-4の前提を明示的に除外し、ADR 0017のStatus変更をADR 0029の
accept後まで保留します（ADR 0029 197–210行）。一度もacceptされていない
proposalを、replacementのaccept後に`Rejected`とする扱いは整合しています。

## 未解決事項

以下は必須のcompanion decisionであり、このsemantic ADRのdefectでは
ありません。

- 正確なrepository versionとtyped-Blob version
- canonical encoding、integer range、resource bounds
- old objectのreadabilityと、mixed-schemaまたは明示的migrationのpolicy
- dump/loadとhistorical IDの扱い
- 正確なCandidate mutation grammarとtarget selection
- human outputとversioned machine outputのschema
- comparisonと`linklog`のsuccessor behavior
- ADR 0028 change-preimage bytesとの正確な統合

general query language、query AST、index、registry、validator runtimeはoptional
のままであり、accepted contractの範囲外です。

## Coverage

レビュー対象は次のとおりです。

- exact accepted revision-2 requirementとそのacceptance record
- ADR 0029のdefinition、alternative、consequence、gate、
  Claim/Evidence/Action traceability
- accepted ADR 0023、0025、0027、0028
- Proposed ADR 0017と、そこからリンクされた詳細proposal
- 現在のnormative requirements、architecture、storage-format、CLIの境界

現在のnormative format-5文書は、想定どおり、引き続き3-field Cause Linkを
規定しています。ADR 0029はこの非互換性を認識し、accepted successor
decision setが揃うまでnormative conversionとimplementationを禁止しています。

## Reviewer結論

exact candidateの判定は`PASS`で、P0–P3 findingはありません。candidateは、
Operatorによる別個の`ACCEPT`または`REVISE`判断まで`Proposed`のままです。
