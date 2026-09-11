# Format 6 規範文書同期 独立レビュー日本語訳 — 2026-09-11

Status: completed report translation / non-normative review support

この文書は、オーナーが`REVIEW`として経路指定したexact snapshotに対するStep 3
独立要件レビューの完全な日本語訳である。英語の
`format6-normative-sync-independent-review-2026-09-11.md`が正本であり、
本資料は候補を変更せず、受理もしない。

## レビュー対象snapshot

- `docs/requirements.md`
  - SHA-256: `02bece52cfcc140d1c671530b9f29971886f05062aae52151bc80fa60683260e`
- `docs/architecture.md`
  - SHA-256: `47d652157b82076aa48984fe9de3e159b2e71663aea6af9030fa04dd6c96e896`
- `docs/storage-format.md`
  - SHA-256: `b53bad2c02729b5290d79db2a00b8255f665387111b44403af22816a38087dc0`
- `docs/cli.md`
  - SHA-256: `0e36fa4ec191a63194187bc28baaa7a481389319535b934e260198fd08cb5970`
- レビュー入力: `docs/process/format6-normative-sync-review-ja-2026-09-11.md`
  - SHA-256: `7530b1ce32e7a77e640d028a0dc9fc5807ee08970f410ffb2ba614b99e21ffe3`

レビュー前の5つのdigestは、すべてオーナーがレビューしたidentityと一致した。
レビュー後にも同じdigestを再計算し、変更されていないことを確認した。

## 権威と方法

権威あるsourceは、Accepted ADR 0029、Accepted ADR 0030、Accepted ADR 0031、
およびADR 0029が参照するaccepted revision-2 requirement authorityである。
Accepted format-5 ADR 0023、0025、0027は、維持されるformat-5契約とformat-6の
後継契約を区別するために使用した。

レビューでは候補を次の項目と比較した。

- generic、opaque、identity-bearing、observer-localなmetadata boundary
- exact format-6 typed record、limit、historical format-5 readability、
  mixed-generation graph behavior、Candidate projection、migration、
  `upstream-change/v2`のconsequence
- metadata mutation、legacy authoring preservation、human/machine output、
  comparison、`linklog`、migration receiptのcontract
- format-5 compatibilityとexternal namespace ownership
- 除外されたquery、registry、validator、Assessment-reference、release、
  repository migration execution、Git-sidecar implementation scope
- `docs/cli.md`へ追加された8つの具体的なhelp-navigation route

既存実装とテストは、非規範の比較evidenceとしてだけ使用した。文書化された8つの
help routeはすべて正常に実行され、登録済みparent/leaf command hierarchyと一致し、
repository statusを変更しなかった。

review dispositionへ依拠する前に、command-line `llmthink dsl audit`を
`format6-normative-sync-step3-2026-09-11`として実行した。依拠した監査結果は
`fatal=0`、`error=0`、`warning=0`であった。17件のhintはDSLの長い行の
readabilityだけに関するもので、実質的なreviewを制限しない。

## 分類済み所見

### F-01 — contradiction — P1/high

`docs/requirements.md` 111–114行と`docs/cli.md` 47–52行は、repository-wideな
`@SEAL_TOKEN`がformat-5 Sealとしてdecodeされることを要求している。format-6
repositoryでは、Accepted ADR 0030 193–202行によりCause targetとasserted previous
revisionはどちらの有効なSeal generationも指定でき、format-6 writerはSeal v6を
生成する。したがって候補の文言では、Acceptedなrepository-wide selectorを通じて
有効なformat-6 Sealへ到達できない。一方、`docs/storage-format.md`はtargetを
format-5に限定せずcanonical Sealとして正しく記述している。

これはformat-6 contractの矛盾であり、format-5文言の持ち越しであって、selector
behaviorの拡張要求ではない。最初に修正すべき箇所はStep 1の規範文書同期候補である。
既存実装は偶然、両方のsupported Seal generationをdecodeするが、それによって
規範上の矛盾が修復されるわけではない。

### F-02 — evidence gap or unresolved unknown — P2/medium

Accepted ADR 0030 283–302行は、Assessmentを伴わないchange identityを
`sealgraph/upstream-change/v2`へ進め、`after_cause_links`にcompleteなformat-6
Cause Linkを含め、metadata changeが`change_id`へ影響することを要求している。
4つの候補文書はいずれも、このaccepted successor schemaを明記も処置もしていない。

Accepted ADRは引き続き権威を持つため、schemaがrejectされた証拠ではない。これは
必須の規範同期contractの欠落であり、後続readerがnested meaningを変更しながら
`upstream-change/v1`を維持する余地を生み得る。

### F-03 — evidence gap or unresolved unknown — P2/medium

Accepted ADR 0030 214–231行は、format-6におけるCandidate-v5の3つのbehaviorを
区別している。すなわち、side-effect-freeなread projection、authorized Candidate
mutationのときだけ行うupgrade、およびCandidate fileを事前に書き換えず、memory上で
Candidate-v6 projectionを構築・validateして行うpublicationである。候補は最初の
2点をhigh levelで記録しているが、direct-publication ruleを明記していない。

この省略はformat-5 compatibilityに関係する。実装者またはoperatorが、`seal`の
前提としてCandidate-v5 fileの書換え、または隠れた予備migrationが必要だと推測する
可能性があるためである。

### F-04 — evidence gap or unresolved unknown — P3/low

Step 2レビュー入力は、source provenance、scope、exclusion、compatibility、1つの
acceptance unknown、self-review claim、要求されたreview questionを示しているが、
requirement-review lifecycleが要求する候補のacceptance criteriaを明示的に列挙して
いない。権威あるADRとexact candidate snapshotからレビューを完了するための十分な
evidenceは得られたため、報告は`not-reviewable`ではない。

次のStep 1 revisionでは、self-review claimを代用するのではなく、4文書の同期を
acceptableとするexact criteriaをレビュー入力に記載すべきである。

### F-05 — optional or future candidate — P3/informational

optional/future candidateがreview済みacceptance boundaryへ昇格した箇所はない。
Metadata query/filter/AST/index work、schema registry/validator、core-owned
revision-context namespace、Assessment-reference commandはいずれも追加されていない。
既存のfuture Git-sidecar constraintも、この候補でimplementationを約束していない。

### F-06 — out of scope — P3/informational

Release、実repository migration、commit、push、deployment、Git-sidecar
implementation、ADR 0028 Assessment adoption/storageは本レビューの対象外である。
これらの領域のevidenceをrequirement authorityとして使用していない。

## 合格したレビュー領域

- Metadataは1つのgenericかつon-diskで必須のcollectionであり、その利用はoptionalの
  ままである。また、containing Cause LinkおよびProvenanceへ埋め込まれたままで、
  CauseLinkIDや独立publication lifecycleは追加されていない。
- Namespace/schema/value recordはbounded、canonical、identity-bearing、coreには
  opaqueであり、Cause/revision graph semanticsから独立したままである。
- Historical format-5 object bytes/IDは不変で、Seal-v5/Provenance-v1と
  Seal-v6/Provenance-v2のexact pairingが維持されている。
- Legacy format-6 `add`/`link`は指定されていないmetadataを保持し、明示的な
  whole-Link deletionは引き続きmetadataを削除する。
- 変更されるmachine recordは`/v3`へ進み、`status/v3`と`stale/v2`は不変である。
  historical projected empty metadataとstored v2 emptyは区別されている。
- config-only 5-to-6 migrationはexplicit、one-way、atomicで、commitされた可能性が
  ある不確実状態ではretry不可のままである。
- Sealgraph-owned metadata namespaceやexternal schema ownershipは割り当てられて
  いない。
- 候補内のすべての具体的なhelp-navigation spellingはcurrent command registryと
  一致し、repository-independentに成功する。

## Step 3 disposition

独立レビューは`completed`である。変更されていない候補は、選択済み`REVIEW` routeに
従ってStep 4へ進むが、F-01により、このexact snapshotをacceptするのは安全ではない。
レビュアーのrecommendationは`REVISE`であり、Step 1へ戻ってselector contradictionを
修正し、必須の同期gapを解消する。このrecommendationはreview inputにすぎず、
requirement textの編集でもowner decisionでもない。

本レビューはrequirementをacceptせず、candidate edit、ADR change、implementation
change、commit、push、PR、merge、release、deployment、repository migration、runtime
mutationのいずれもauthorizeしない。
