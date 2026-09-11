# Cause Link拡張可能metadata要件候補 — 日本語レビュー支援（2026-09-10）

この文書は、次の英語文書の完全な日本語訳であり、レビュー支援用です。
規範的な候補または独立した要件ではありません。

1. `cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
2. `cause-link-extensible-metadata-requirement-review-input-2026-09-10.md`

## 第1部：要件候補 Revision 1

- 状態：Step 1要件候補。自己レビュー完了。第1回オーナーレビュー待ち
- 候補revision：1
- 決定オーナー：Operator
- 権限：候補のみ。規範ではなく、実装権限でもない
- 提案されたレビュー入力：
  `cause-link-extensible-metadata-requirement-review-input-2026-09-10.md`

### 目的

既存のCause Link契約を維持しながら、自由記述の`messages`では不十分な場合に、
observerが構造化されversion管理された意味を追加できるようにする。動機となるprofileは、
progress、Cause不整合のcorrection、reconciliationなど、targetが1個以上の正確な過去Sealの
revisionであると主張する理由を説明する。汎用拡張機構は、vocabularyごとに新しいcore fieldを
追加することなく、将来のLink-local vocabularyにも利用できなければならない。

### 元となる指示

- **D-CLM-001:** 互換性のため`messages`を維持する。
- **D-CLM-002:** 構造化revision contextは任意とし、callerは`messages`だけを引き続き使用できる。
- **D-CLM-003:** 将来の構造化Link-local dataのため、拡張可能領域を1つ追加する。
- **D-CLM-004:** revision理由専用fieldと第2の汎用拡張経路を両方追加せず、1つの拡張機構により
  revision-context構造を一般化する。
- **D-CLM-005:** Cause Linkはimmutable Provenance Blob内のrecordのままとする。独立してaddress可能な
  Blobにはせず、CauseLinkIDも追加しない。

これらの指示が現在のオーナー起点の要件ソースである。ただし、それだけではこの候補の正確な
文言を受理せず、ADR、format変更、実装、migration、commit、push、release、deploymentを
許可しない。

### 既存権限の扱い

- Accepted ADR 0023は、observer-scoped revision assertion、structural edge union、staleness、
  history、admissionについて引き続き権威を持つ。MetadataはCause edgeまたはRevision edgeを
  作成・削除しない。
- Accepted ADR 0025は、Universal Blob layerとSealからMaterialおよびProvenanceへのjoinについて
  引き続き権威を持つ。Cause LinkとそのmetadataはProvenance contentのままであり、SealIDから
  推移的にcommitされる。metadataを保存する前に、後継のpersisted formatが必要である。
- Accepted ADR 0027はformat-5 authoringとinspectionについて引き続き権威を持つ。後続のAccepted
  CLI/schemaがmetadata mutationとoutputを定義しなければならない。
- Accepted ADR 0028はupstream-impact Assessmentについて引き続き権威を持つ。汎用Link metadataで、
  AssessmentID、evidenceを伴うdisposition、adoption reference、seal-admission resultを模倣してはならない。
- Proposed ADR 0017と`../proposals/link-metadata-and-sealgraphql.md`は非規範の設計入力である。
  この候補が再利用するのは、分離可能なnamespaced `namespace` / `schema` / `value`概念だけである。
  旧Link shape、旧revision model、query language、namespace割当、resource limit、implementation scopeは
  受理しない。後続ADRはADR 0017をrevise、replace、withdraw、または別の方法で明示的に処置しなければならない。

### 対象範囲

- 各exact Cause Link上の、任意のlogical metadata set 1つ。
- namespace付きでversion識別可能なstructured value。
- 決定的でboundedかつidentity-bearingなcanonical表現。
- 汎用metadataで表現されるversioned revision-context profile。
- Candidate authoringおよびhuman/machine inspection要件。
- 既存`messages`およびSealのcompatibilityとmigration要件。

### 対象外

- Cause Linkを独立address可能なBlobまたは独立active graph entityにすること。
- Cause direction、revision-edge derivation、stale propagation、frontier membership、通常の
  Cause-closure admissionを変更すること。
- ADR 0028 upstream Assessmentの置換、またはそのadmission gateの充足。
- progress、correction、reconciliation、trust、approval、truth、supersessionをcore graph stateとして扱うこと。
- query language、schema registry、network schema retrieval、signature、trusted actor/time、secret storage。
- 正確な後継format number、CLI spelling、migration command、release scope、implementation sequence。

### 必須logical model

後継契約は既存Cause Linkの3 fieldを維持し、汎用metadata collectionを1つだけ追加しなければならない。

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

logical metadata collectionは空でもよい。後継storage decisionは空集合に対する正確なcanonical表現を
1つ選択し、omissionに環境依存の意味を与えてはならない。

#### 既存field

`target_seal`、`previous_revision_seal_of_target_seal`、`messages`はformat 5の意味を維持する。
特に次を満たす。

- `messages`はidentity-bearingで、sorted、duplicate-freeなstring setのままとする。
- 空の`messages` arrayは引き続きvalidとする。
- 既存message textをstructured metadataとして再分類またはparseしない。
- revision edgeはexact previous SealID arrayだけから導出し、metadataからは導出しない。

#### 汎用metadata

各metadata entryは、exactかつcase-sensitiveな1つのnamespaceにより所有される。1つのCause Linkは、
1つのnamespaceについて最大1 entryを含められる。`schema`はnull、またはnamespace ownerが選択した
immutableかつversion-specificなidentifierである。Coreはidentifierをdereferenceせず、valueが
conformすると推論しない。

`canonical_value`は、null、boolean、integer、string、array、string-keyed objectに十分なboundedかつ
deterministicなsubsetをsupportしなければならない。storage decisionはordering、encoding、nesting、
node count、string size、entry count、total byte limitを固定しなければならない。floating-point identity、
executable value、environment expansion、dynamic selectorは禁止する。

Metadataはimmutable Provenance contentである。namespace、schema、valueを変更するとProvenanceIDが変わり、
したがってSealIDも変わる。Metadataは追加graph edgeを作らず、3つのstructural Cause Link fieldの
validationを弱められない。

未知だが構造的にvalidなnamespaceはreadableかつinspectableなままとする。Coreはそのexact canonical valueを
保持し、domain meaning、truth、approval、authorityを割り当ててはならない。

### Revision-context profile

拡張機構は、次のlogical informationを持つversioned profileを表現できなければならない。これは例示的な
shapeであり、Accepted namespace、schema identifier、canonical byte contractではない。

```json
{
  "namespace": "example.revision-context",
  "schema": "example.revision-context/v1",
  "value": {
    "revisions": [
      {
        "previous_seal": "<full-seal-id>",
        "kind": "correction",
        "rationales": [
          "過去revisionは不整合なCauseを使用していた。"
        ]
      }
    ]
  }
}
```

profileは各説明を、Cause Linkのprevious set全体ではなく1つのexact previous Sealに対応付ける。
conforming profileは、各`previous_seal`がstructural previous arrayに含まれることを要求でき、
`progress`、`correction`、`reconciliation`、または拡張可能な同等値を定義できる。それらの値は
namespace ownerのclaimのままであり、core graph semantics上でSealをinvalidate、supersede、preferしない。

このprofileはobserverのrevision assertionを説明する。seal-operation metadataではなく、trusted actorやtimeを
記録しない。また、ADR 0028のupstream Assessmentでもない。AssessmentID、change binding、evidence adoption、
compatibility disposition、admission effectを持たない。これらの強いclaimのいずれかが必要な場合は、別途Acceptedの
mechanismを引き続き必要とする。

### Authoringおよびinspection要件

- Metadata mutationは、1つのCandidate内の1つのexact target Linkに作用する。
- 1 namespaceのadd、replace、removeは明示的なoperationとする。
- metadata-only editは、同じauthorized Candidate operationで明示的に変更しない限り、target、previous array、
  `messages`、その他すべてのnamespaceを維持する。
- rejectされたmetadata inputはCandidateを変更しない。
- human inspectionはnamespaceとschemaを識別し、arbitrary valueをescapeとoutput bound付きで表示する。
- versioned machine outputはtruncationなしにcomplete canonical valueを返す。
- Cause Link comparisonと`linklog`は、`messages`変更とmetadata namespace/value変更を区別する。
- Metadata mutationは暗黙にSealをpublishしたりREFをmoveしたりしない。

### Compatibilityおよびmigration要件

`messages`を維持することでsemanticおよびauthoring continuityは得られるが、新しいcanonical bytesを旧format-5
readerが読めるようにはならず、identity-bearing metadata追加後のSealIDも維持しない。

したがって、後継decisionは次を個別に示さなければならない。

1. どの旧typed Blob schemaを引き続き読めるか。
2. repositoryにexplicit one-way format migrationが必要か。
3. 未変更の既存SealとそのIDをどのようにaddressableなままにするか。
4. 編集された旧Cause Linkをどのようにnew-format ProvenanceおよびSealにするか。
5. dump/loadおよびmigrationがobject closureとexact metadata bytesをどのように保持するか。

Migrationは既存`messages` memberをすべて正確に保持しなければならない。legacy message textからnamespace、schema、
revision kind、rationale、upstream Assessment、その他structured interpretationを作ってはならない。metadataがないことで、
既存historical Sealをcorrupt、unapproved、またはsemantically reclassifiedにしてはならない。

### Acceptance criteria

この要件は、後続のreviewed designが次を示した場合にのみ満たされる。

1. Cause LinkはCauseLinkIDなしのProvenance-embedded recordのままである。
2. format-5の3 fieldの意味を維持し、metadataなしで`messages`だけを引き続き使用できる。
3. 専用の第2拡張経路なしに、1つのgeneric namespaced metadata mechanismでrevision-context例をsupportできる。
4. Metadataがidentity-bearing、deterministic、boundedで、安全にinspectできる。
5. Structural Cause/Revision behaviorがmetadata meaningからbyte-for-byteで独立し、未知namespaceで弱められない。
6. Revision-context dataをADR 0028 Assessment evidenceと区別でき、そのadmission stateを満たせない。
7. Candidate mutationがexplicitかつnamespace単位でisolatedされ、publishしない。
8. humanおよびversioned machine inspectionが`messages`とmetadataを別々に表示する。
9. compatibility claimがfield continuity、old-object readability、old-reader support、migration、SealID preservationを区別する。
10. migrationがinferred structured meaningなしでlegacy messagesを保持する。

### 前提および未解決decision

- 最終namespace ownerとimmutable revision-context schema identifierは未割当である。
- exact canonical value limitは未選択である。
- 後継repository、Seal、Provenance、Candidate、dump/load、output schema versionは未選択である。
- mixed historical-schema readingとmandatory repository conversionのどちらにするか、明示的compatibility decisionが必要である。
- exact CLI syntaxとgeneral metadata queryを初回deliveryに含めるかは未決であり、この要件候補はどちらも暗示しない。
- Sealgraphがmaintained revision-context profileを提供するか、external exampleだけを文書化するかはowner decisionである。
  generic mechanismはどちらもsupportしなければならない。

### Step 1自己レビュー

- **Source provenance:** 上記D-CLM-001からD-CLM-005はowner-originatedである。既存ADRとproposalは比較・制約sourceである。
- **Prior authority:** Accepted ADR 0023、0025、0027、0028を維持する。ただしpersisted implementationには別途Acceptedの
  successor storage/CLI decisionが必要である。Proposed ADR 0017を暗黙に受理しない。
- **Compatibility:** 候補は`messages` continuityをold-readerおよびSealID compatibilityから明示的に分離する。
- **Namespace:** illustrative exampleによって実在namespaceまたはschema identifierを割り当てない。
- **Optional/future work:** query language、validator、built-in profile ownershipは非blockingの後続decisionである。
- **Unknowns:** 上記unresolved decisionはmaterial review questionであり、implementation assumptionで埋めていない。
- **Later effects:** ADR作成・改訂、normative conversion、implementation、migration、commit、push、release、deploymentは、
  この候補では許可されない。

## 第2部：独立レビュー入力

- 状態：ownerにより独立レビューが要求済み
- 候補revision：1
- 候補path：
  `docs/process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
- 候補SHA-256：
  `a419872a4971db1d8fcaebe797fde085d9a9ef107b15f93f72eb600138e5cc96`
- 第1回review後のowner route：`REVIEW`
- route選択：2026-09-10、現在のSealgraph要件レビュースレッド

この入力は上記candidateのexact bytesだけに束縛される。candidate変更には、新revision、digest、self-review、first owner reviewが
必要である。この文書はcandidateをacceptせず、ADR、format change、implementation、migration、commit、push、release、deploymentを
許可しない。

### Source provenanceおよびprior authority

要件sourceはcandidateのD-CLM-001からD-CLM-005として記録されたOperator directionである。すなわち、`messages`維持、
structured revision contextのoptional化、generic extension area 1つ、concrete use caseのgeneralization、CauseLinkIDなしで
ProvenanceにCause Linkを埋め込むこと、である。

Accepted ADR 0023、0025、0027、0028に対してcandidateをreviewする。Proposed ADR 0017と詳細proposalは非規範の比較資料としてのみ
扱う。特にADR 0017の旧Link shape、revision assumption、query language、namespace reservation、limit、scopeのacceptanceを
推論しない。

### Review scope

対象内：

- 既存3 Cause Link fieldおよびそのsemanticsのpreservation。
- namespaced `namespace` / `schema` / `value` metadata mechanism 1つ。
- deterministic identity commitment、bound、安全なinspection。
- generic metadataのoptional useとしてのrevision-context profile。
- structural Cause/Revision behaviorおよびADR 0028 Assessmentとのseparation。
- Candidate mutation、output、compatibility、migration要件。
- 10個のacceptance criteriaのcompletenessとtestability。

対象外：

- successor formatまたはCLIの実装・acceptance。
- Cause Linkのfirst-class Blob化。
- query language、schema registry、validator runtime、signature、trusted time/actor、secret storageのdesign。
- final namespace name、version number、resource limit、command spelling、migration command、release sequenceの選択。
- useful adjacent featureをacceptance blockerとして提案すること。

### Review対象のacceptance criteria

reviewerは、candidateが矛盾なく次のすべてを要求しているか判断しなければならない。

1. CauseLinkIDなしのProvenance-embedded Cause Link。
2. unchanged format-5 field meaningと継続的な`messages`-only use。
3. revision-context例を表現できるgeneric extension path 1つ。
4. identity-bearing、deterministic、boundedでsafe-renderedなmetadata。
5. namespace meaningから独立したstructural graph behavior。
6. ADR 0028 Assessment identityまたはadmission effectの代替禁止。
7. implicit publishのないexplicitかつnamespace-isolatedなCandidate mutation。
8. `messages`とmetadataに対するhuman/machine visibilityの分離。
9. field、reader、migration、SealIDに関するprecise compatibility claim。
10. inferred structured meaningなしのexact legacy-message preservation。

### 既知のunknown

- final namespace ownerおよびrevision-context schema identifier。
- exact canonical valueおよびresource limit。
- successor repositoryおよびtyped-Blob schema version。
- mixed historical-schema readingまたはmandatory conversion。
- exact CLIおよびinitial query scope。
- revision-context profileをproject-maintainedにするかexternal exampleだけにするか。

これらはevidence gapまたはlater owner decisionである。未決のままにするとstated requirementがinternally incoherent、accepted authorityと
incompatible、またはuntestableになる場合のみ、reviewerはcontradictionを報告する。optional design preferenceはoptional findingのままにする。

### Review question

1. `messages`を維持しながらmetadata collectionを1つ追加することで、競合する2つのextension mechanismを作らずowner directionを満たすか。
2. Cause Linkをindependently addressedにせず、metadataはProvenance BlobおよびSealIDを通じて正しく配置・commitされているか。
3. 未知または矛盾したnamespaceを含め、すべてのstructural Cause/Revision resultをmetadata contentから独立させられるか。
4. revision-context profileはseal-operation metadataおよびADR 0028 upstream Assessmentから明確に分離されているか。
5. compatibilityとmigration statementは、old readerまたはSealIDが自動的にcompatibleだというclaimを防ぐため正確かつ十分か。
6. candidateはProposed ADR 0017の独立してvalidな部分だけを再利用し、proposalのlater dispositionを明示しているか。
7. acceptance前に固定すべきunresolved namespace、schema、format、CLI、limit decisionに依存するacceptance criteriaがあるか。

### 必須reviewer result

各material resultを次のいずれか1つに分類する。

- candidateまたはauthoritative sourceとのcontradiction。
- evidence gapまたはunresolved unknown。
- optionalまたはfuture candidate。
- out of scope。

`completed`または`not-reviewable`を返し、exact primary-source locationを引用し、不確実性を保持し、candidateを編集しない。
独立review開始前に、第1回owner reviewでOperatorがreview後のrouteを指定しなければならない。
