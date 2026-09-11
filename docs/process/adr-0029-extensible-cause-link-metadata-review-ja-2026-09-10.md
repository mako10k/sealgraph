# ADR 0029 レビュー支援 — 2026-09-10（完全な日本語訳）

英語の[`ADR 0029`](../adr/0029-extensible-cause-link-metadata.md)が正本である。この文書は、その全review scopeの日本語訳であり、別のdecisionを追加しない。

## Titleとmetadata

ADR 0029：拡張可能なCause Link metadataとrevision-context boundary。

Status：2026-09-10 Proposed。独立reviewとOperator acceptance待ち。

Decision owner：Operator。

Requirement authority：2026-09-10にacceptされたexact revision-2 requirement。

Acceptされた場合、このADRが所有するのはsemantic/responsibility boundaryだけである。exact persisted bytes、repository/typed-schema version、numeric resource limit、migration、CLI grammar、machine-output schemaには、別途acceptされたcompanion ADRが必要である。このProposed ADRはimplementationまたはnormative conversionを許可しない。

## Context

Format 5のCause Linkはimmutable Provenance内にあり、1つのtarget SealID、observer-scoped previous-revision SealID set、free-form `messages` setを持つ。これはstructural revision evidenceを保持するが、1つ以上のexact previous Sealに対して、なぜexact targetをrevisionとしてassertするのかをversioned structured formで説明する場所を持たない。

Accepted requirementは`messages`を保持し、structured useをoptionalにし、vocabularyごとのcore fieldではなく1つのgeneric extensible areaを追加し、CauseLinkIDを作らずLinkをProvenance内に保つ。motivating profileはprogress、Cause inconsistencyのcorrection、reconciliationを含むが、それらをcore graph stateにはしない。

Accepted ADR 0023はstructural Cause/revision meaning、ADR 0025はformat-5 typed-Blob identity、ADR 0027はformat-5 authoring/inspection schema、ADR 0028はupstream-impact Assessment identity/evidence/adoption/disposition/admissionを所有する。Proposed ADR 0017とproposalにはgeneric metadataの有用なrationaleがあるが、format-4 assumptionとquery-language designを結合しておりnon-normativeである。

## Decision

### 1つのembedded generic metadata collection

Successor Cause Linkは現在の3fieldを保持し、optionalなlogical `metadata` collectionを1つだけ追加する。各entryは`namespace`、nullable `schema`、`canonical_value`を持つ。

```text
cause_link := {
  target_seal: SealID,
  previous_revision_seal_of_target_seal: [SealID],
  messages: [string],
  metadata: [metadata_entry]
}

metadata_entry := {
  namespace: string,
  schema: string | null,
  value: canonical_value
}
```

`messages`は現在のidentity-bearing、sorted、duplicate-free string setの意味を保持し、metadataなしでも利用できる。既存messageはstructured valueへparse、reclassify、migrateしない。

Cause LinkはProvenance内のrecordのままである。CauseLinkID、independent Link Blob、active-Link set、別Link publication lifecycleは作らない。MetadataはProvenanceの一部としてcommitされ、SealIDからtransitively committedされる。

### Namespaceとschema responsibility

`namespace`はSealgraph core外のownerが所有するnon-empty、case-sensitive UTF-8 identifierである。1つのLinkには1つのexact namespaceにつき最大1entryだけを持つ。storage companionがdeterministic namespace orderを選択する。Coreはnamespace ownershipを検証せず、alias/caseを書き換えず、truth、trust、approval、preference、authorityを割り当てない。

`schema`はnullまたはnamespace ownerが選ぶnon-empty、case-sensitive、immutable-version identifierである。Coreは保存・表示するがdereferenceせず、conformanceを意味しない。Schema registry、network retrieval、mandatory namespace-specific validationはこのADRの範囲外である。

このADRはpersisted `sealgraph` namespaceを予約せず、virtual query namespaceも採用しない。将来のcore-owned persisted namespaceには別のaccepted decisionが必要であり、Proposed ADR 0017から推論しない。

### Canonical value responsibility

`canonical_value`はnull、boolean、bounded integer、UTF-8 string、ordered array、uniqueかつcanonically orderedなstring-keyed objectだけを扱う。Floating point、executable value、environment expansion、dynamic REF/HEAD selector、implicit type coercionは禁止する。Stringはimplicit Unicode normalizationを行わない。Array orderはidentity-bearing、object orderはcanonicalでありauthor-significantではない。

Storage companionはexact integer range/encoding、member order、escaping、depth、node count、string/key size、namespace-entry count、per-Link metadata bytes、必要なProvenance/object limitを選択する。Reader/writerの両方に適用し、environment、presentation destination、namespace meaningに依存させない。

Unknownだがstructurally validなnamespace/schemaはcanonically readableかつexact inspectableである。Coreがvalidateするのはstructure、bound、canonical bytes、identity commitmentだけで、domain meaningはvalidateしない。

### Structural graph independence

Cause/revision edgeは`target_seal`と`previous_revision_seal_of_target_seal`だけから導出する。Metadataはedgeを追加、削除、prefer、invalidateせず、validationを弱めない。Staleness、active leafness、history、impact、frontier membership、cycle detection、Cause-closure admissionはnamespace/schema/valueを解釈しない。

Metadataはidentity-bearingである。Metadata変更はProvenance identityを変え、したがってSealIDを変更し、assertion-source identity、comparison output、その他exact output bytesも変え得る。Semantic non-interpretationはbyte-identical identity/outputを意味しない。

### Revision-context profile

最初のdesignではrevision contextをgeneric profile exampleとして文書化し、core-owned persisted namespace/schemaにはしない。Namespace ownerは`revisions` arrayを持つversioned valueを定義でき、各entryは`previous_seal`、`kind`、`rationales`を持てる。

```text
revision_context := {
  revisions: [revision_explanation]
}

revision_explanation := {
  previous_seal: SealID,
  kind: string,
  rationales: [string]
}
```

Profileは各説明を1つのexact previous SealIDにbindする。`previous_seal`がLinkのstructural previous setに含まれることをprofileが要求でき、`progress`、`correction`、`reconciliation`などを定義できる。それらはnamespace-owner claimであり、coreはsupersession、preference、truth、approval、graph behaviorを導出しない。

Profileはtrusted actor/timeまたはseal-operation event dataを含まず、ADR 0028 upstream Assessmentではない。AssessmentID、change binding、adopted evidence、compatibility disposition、admission stateを持たず、満たせない。Sealgraphがnormative revision-context namespace/validatorをmaintainする場合は別のaccepted decisionが必要である。

### Candidateとinspection boundary

Metadata editingは1つのCandidate内の1つのexact target Linkだけに作用する。1namespaceのadd/replace/removeはexplicit operationである。Metadata-only operationは、同じauthorized operationが明示変更しない限り、target、previous set、`messages`、他namespaceを保持する。Rejected inputはprior Candidate bytesを変更しない。

Metadata mutationはSealを作成せず、REFを暗黙にmoveしない。Publicationは従来のone-Candidate、one-REF、expected-old CAS workflowを使う。

Human inspectionはnamespace/schemaを示し、arbitrary valueをbounded unambiguous escapingで表示する。Versioned machine outputはcomplete canonical valueを返す。Candidate comparisonと`linklog`は`messages` changeとmetadata namespace/value changeを区別する。Exact command spelling/output schemaはCLI companion ADRが所有する。

### Compatibilityとcompanion decision set

`messages`保持が保証するのはfield meaningとauthoring continuityだけである。Successor bytesをformat-5 readerが読めることも、identity-bearing metadata追加後のSealID保持も保証しない。

Normative conversionまたはimplementation前に、Operatorはcomplete companion decision setをacceptしなければならない。

1. Storage/migration ADR：repository/typed-Blob version、canonical bytes/limits、old-object readability、unchanged historical-ID addressability、migrationまたはmixed-schema policy、dump/load closure、ADR 0028 change identityへの影響をfixする。
2. CLI/output ADR：Candidate mutation grammar、exact-target selection、rejection atomicity、human rendering、versioned machine schema、comparison、`linklog` behaviorをfixする。

Complete setはlegacy `messages` memberをexactに保持し、message textからnamespace、schema、revision kind、rationale、Assessment、その他structured meaningを推論しない。Metadataのないhistorical Sealをcorrupt、unapproved、semantically reclassifiedにしない。

General metadata query language、query AST、index、query mutation、schema registry、validator runtimeはこのdecision setに含めない。

### Proposed ADR 0017の処置

Acceptされた場合、ADR 0029はProposed ADR 0017に代わるcurrent generic Cause Link metadata semantic directionとなる。ADR 0017のformat-4 Link shape、singular message model、reserved virtual query namespace、SealGraphQL design、query language、namespace assignment、suggested numeric limits、implementation scopeは採用しない。

ADR 0029がacceptされるまでADR 0017はProposedのままとする。Acceptance follow-upでADR 0017をRejectedにし、ADR 0029へlinkする。Proposed replacementだけでprior statusを書き換えない。

## Alternatives

- Dedicated revision-kind/rationale field：later vocabularyごとにcore fieldが必要となり、progress/correction/supersession semanticsをcoreが持ち得るためreject。
- `messages`をstructured metadataで置換：compatibilityを壊し、structured authoringを強制し、既存textのlossy interpretationを招くためreject。
- Cause Linkをfirst-class Blob化：accepted requirementがembedded Linkを選び、Sealgraphにactive Link set/publication authority/replacement lifecycleがないためreject。
- ADR 0017とSealGraphQLを同時採用：query languageはaccepted use caseに不要で、semantic/storage/observation/cost/public CLI decisionを不必要に結合するためreject。
- Core revision-context namespaceを今決める：capabilityにはcore ownershipが不要で、validator/evolution policyなしのnamespace assignmentは不要なcompatibility promiseになるためdefer。
- Exact storage/CLI bytesを本ADRに含める：既存ADRがgraph semantics、typed bytes、migration、CLI schemaを分離しており、companion ADRの方がauthority/compatibility transitionを明示できるためreject。

## Consequences

- Callerは`messages`だけを使い続けるか、1つのgeneric mechanismでstructured Link-local meaningを追加できる。
- Revision explanationはexact previous Seal generationを示せるが、graph edge/core supersession stateにはならない。
- Unknown namespaceはportable/inspectableだが、coreにtrust/semantic validationされない。
- Metadata変更はimmutable Provenance/Seal identityを変える。
- Old readerはsuccessor Link shapeをsilentにacceptできない。
- Profile consumerはcore canonical readingと独立してnamespace-specific schemaを認識・validateする必要がある。
- Implementation前に少なくとも2つのcompanion ADRが必要となりreview workは増えるが、semantic/storage/migration/CLI authorityの混在を防ぐ。
- General metadata queryは別途require/acceptされるまで利用できない。

## Implementation notes

このADRがProposedの間、またはrequired companion ADRが未acceptの間、implementation actionは許可されない。Complete decision setが別途implementation instructionとともにacceptされた後、domain/CLI/storage分離、bounded canonical values、deterministic fixture、one-target Candidate mutation、rejection atomicity、separate messages/metadata output、ADR 0028 change-preimage対応、compatibility/migration fixture、normative document synchronizationを実施する。

## Review

2026-09-10 author self-reviewは、accepted requirementの10 criteria、structural field/opaque metadata/profile meaningの分離、ADR 0023/0025/0027/0028 precedence、ADR 0017のnon-adoption/disposition、messages/old readers/migration/SealID compatibility、core revision-context namespaceを割り当てない判断、implementation gateを確認した。

Independent ADR reviewとOperator acceptanceはpendingである。Review questionは次のとおり。

1. Semantic sliceはstorage/migrationおよびCLI/output companionを先取りせず、独立reviewに十分なdecisionを持つか。
2. External-profile choiceはhidden core schema promiseを作らず、accepted revision-context capabilityを満たすか。
3. Identity/output bytesが変化しつつ、全structural graph resultをmetadata meaningから独立させられるか。
4. ADR 0028 Assessment identity/admission boundaryを保持するか。
5. Proposed ADR 0017のquery/format-4 assumptionを継承せず処置できているか。

## Evidenceとtraceability

- E-LM-001：accepted requirement/acceptance recordがowner-approved outcome、compatibility distinction、open successor decision、authority boundaryを確立する。
- E-LM-002：ADR 0023が3-field Cause Link、observer-scoped previous assertion、structural edge union、staleness/history/admissionを定義する。
- E-LM-003：ADR 0025がexact Cause LinkをProvenanceID/SealIDへcommitし、metadata extensionに別accepted typed schemaを要求する。
- E-LM-004：ADR 0027がformat-5 Candidate authoring/inspection/comparison/`linklog` schemaとchanged meaning用successor schemaを所有する。
- E-LM-005：ADR 0028がupstream Assessment identity/evidence/adoption/disposition/admissionを所有し、AssessmentIDをmessage/attachmentへ隠すことをrejectする。
- E-LM-006：Proposed ADR 0017/proposalはone namespaced metadata set/canonical valueのnon-normative rationaleを提供するが、obsolete Link/query assumptionも持つ。
- E-LM-007：`architecture.md`がnew persisted fieldにstorage-format change、deterministic fixture、compatibility consideration、approved ADRを要求する。

Exact traceabilityは次のとおりである。

| Claim | Evidence | 必要なimplementation action |
| --- | --- | --- |
| C-LM-001：current fieldとembedded Link identityを保持 | E-LM-001, E-LM-002, E-LM-003 | A-LM-001, A-LM-003 |
| C-LM-002：optional generic metadata collectionを1つ追加 | E-LM-001, E-LM-006 | A-LM-001, A-LM-003 |
| C-LM-003：namespaceをopaque、canonical valueをboundedに保つ | E-LM-001, E-LM-003, E-LM-006, E-LM-007 | A-LM-001, A-LM-003, A-LM-005 |
| C-LM-004：graph semanticsをmetadata meaningから独立させる | E-LM-001, E-LM-002 | A-LM-003, A-LM-005 |
| C-LM-005：core graph stateなしでrevision contextをsupport | E-LM-001, E-LM-002 | A-LM-002, A-LM-003 |
| C-LM-006：upstream Assessmentを代替しない | E-LM-001, E-LM-005 | A-LM-001, A-LM-002, A-LM-005 |
| C-LM-007：explicit Candidate/publication boundaryを保持 | E-LM-001, E-LM-004 | A-LM-002, A-LM-003, A-LM-005 |
| C-LM-008：compatibility/migrationを明示 | E-LM-001, E-LM-003, E-LM-004, E-LM-007 | A-LM-001, A-LM-005 |
| C-LM-009：metadataをADR 0017 query scopeから分離 | E-LM-001, E-LM-006 | A-LM-004 |
| C-LM-010：complete accepted companion setを要求 | E-LM-001, E-LM-003, E-LM-004, E-LM-007 | A-LM-001からA-LM-005 |

Implementation action identityは次のとおりである。

- A-LM-001：別途authorizationの下で、accepted storage/migration companionをimplementする。
- A-LM-002：別途authorizationの下で、accepted CLI/output companionをimplementする。
- A-LM-003：domain、repository、inspection、graph behaviorをimplementする。
- A-LM-004：Proposed ADR 0017を処置し、query workを分離したままにする。
- A-LM-005：canonical、compatibility、migration、graph-independence、rejection-atomicity、output fixtureを追加し、normative documentを同期する。

いずれも本Proposed ADRだけでは実行を許可されない。

## Follow-ups

- Operator acceptance前にこのexact Proposed ADRを独立reviewする。
- Implementationせずstorage/migrationおよびCLI/output companion ADRをdraftする。
- ADR 0029 acceptance後、Proposed ADR 0017をRejectedとしてADR 0029へlinkする。
- Complete companion setのindependent review、acceptance、別途authorization前にnormative conversionまたはimplementationを始めない。
