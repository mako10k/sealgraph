# ADR 0030 Format-6 Cause Link metadata storageとmigration レビュー用日本語訳 — 2026-09-10

この文書は、`docs/adr/0030-format6-link-metadata-storage-and-migration.md`
全体に対する日本語のレビュー支援です。英語ADRが正本であり、この翻訳は
第2の権威あるartifactではありません。

## タイトルとStatus

ADR 0030: Format-6 Cause Link metadata storageとmigration

Status: 2026-09-10時点で`Proposed`。independent reviewとOperator acceptanceは
pending。

Decision owner: Operator。

Requirement authority: accepted ADR 0029と、そのexact accepted requirement。

このADRが所有するのは、persisted bytes、typed-schema version、resource limit、
historical-object readability、repository migration、ADR 0028 change-preimageへの
影響だけである。exact metadata mutation grammarとhuman/machine inspection schema
は、別途必要なCLI/output companion ADRが所有する。このADRがProposedである間、
implementationもnormative conversionも承認されない。

## Context

Format 5はCause Linkを`target_seal`、
`previous_revision_seal_of_target_seal`、`messages`だけでencodeする。typed reader
はunknown memberをrejectする。Accepted ADR 0029はidentity-bearingな`metadata`
collectionを一つ追加するため、既存format-5 objectへそのmemberを加えることは
compatible extensionではなく、accepted byte contractへの違反になる。

historical format-5 SealとそのIDは、validかつaddressableなままでなければならない。
すべてのhistorical Provenanceをre-encodeするとProvenanceIDとSealIDが変わり、
exact referenceをgraph全体で置換する必要が生じる。一方、format-4からformat-5
へのtransitionと異なり、ADR 0029はCauseまたはrevision graphの意味を変えない。
したがってformat-6 readerは、二つのgraph authorityを持たずに、限定されたexact
historical format-5 typed-object readerを維持できる。

ADR 0028は、将来のsuccessor contractにpersisted Assessment referenceを別途
要求している。そのfeatureはaccepted Cause Link metadata scopeには含まれない。
このADRはADR 0028のassessment-free change preimageにmetadataを反映しなければ
ならないが、Assessment adoption storageを新たに作ってはならない。

## Decision

### Repositoryとtyped-schemaのversion

successor repository formatは6とする。exact config bytesは次のとおり。

```text
repository_format = 6
object_format = sha256
ref_format = manifest-v1
```

各行の後にLFを一つ置き、それ以外のbytesは置かない。

Format 6は次を導入する。

- `sealgraph/seal/v6`
- `sealgraph/provenance/v2`
- `sealgraph/candidate/v6`

`sealgraph/material/v1`、Blob identity、loose-object path、REF manifest v1、
attachment、root/draftの意味、Cause direction、revision assertionは変更しない。

format-6 writerはSeal v6、Provenance v2、Candidate v6だけをemitする。format-6
repositoryは、後述のcompatibility ruleに従い、exact historical Seal v5、
Provenance v1 object、Candidate v5 fileも保持してreadできる。member shapeから
schema versionを推測しない。

### Exact successor record

Seal v6はcompact canonical JSONで、exact member orderは次のとおり。

```text
schema, material, provenance
```

logical shapeは次のとおり。

```json
{"schema":"sealgraph/seal/v6","material":"<material-id>","provenance":"<provenance-id>"}
```

すべてのSeal v6は、valid Material v1を一つ、valid Provenance v2を一つ参照する。

Provenance v2のexact member orderは次のとおり。

```text
schema, root, draft, cause_links
```

Candidate v6はCandidate v5のmember orderを維持する。

```text
schema, ref, expected_ref_head, content, attachments,
root, draft, cause_links
```

schema valueはexactly `sealgraph/candidate/v6`である。

Provenance v2とCandidate v6の両方で、Cause Linkのexact member orderは次のとおり。

```text
target_seal, previous_revision_seal_of_target_seal, messages, metadata
```

すべてのmemberはrequired。empty metadataはexactly `"metadata":[]`とencodeする。
omissionとJSON `null`はinvalid。すべてのsuccessor typed boundaryでunknown memberは
invalid。

metadata entryのexact member orderは次のとおり。

```text
namespace, schema, value
```

`namespace`とnon-null `schema`はJSON string。`schema`は代わりにJSON `null`でも
よい。metadata entryはdecoded namespace UTF-8 bytesでsortし、exact namespace
bytesについてduplicate-freeとする。namespace comparisonはcase-sensitiveで、
Unicode normalizationをしない。

metadata以外のstring encoding、array、ID、Link uniqueness、typed Blob validation
はADR 0025のcanonical ruleを維持する。Cause Linkはexact `target_seal`について
uniqueで、次のtupleでsortする。

```text
(target_seal, previous_revision_seal_of_target_seal, messages, metadata)
```

valid Link間ではunique targetにより後続tuple memberは識別に使われないが、
complete canonical comparatorを定義するために含める。

### Canonical metadata value

metadataの`value`は、次のclosed setに含まれる再帰的canonical JSON value一つ。

- `null`
- `true`または`false`
- `-9223372036854775808`から`9223372036854775807`までのsigned 64-bit integer
- UTF-8 string
- canonical metadata valueのordered array
- UTF-8 string keyからcanonical metadata valueへのobject

integerは最短のbase-10 JSON spellingを使う。leading plus、leading zero、
negative zero、decimal point、fraction、exponent、whitespaceはinvalid。
floating pointとsigned 64-bit範囲外のintegerはinvalid。

stringはADR 0025のcanonical JSON scalar escapingを使い、Unicode normalization
をしない。array orderはidentity-bearing。object keyはJSON decode後にuniqueで、
decoded UTF-8 bytesのbytewise ascending orderで現れる。object orderはcanonical
であり、author-significantではない。異なるescape spellingが同じstringへdecode
される場合も含め、duplicate keyはinvalid。

decode、semantic validation、canonical re-encode、exact byte equalityを要求する。
generic host JSON serializerやmap iteration orderはcanonical encoderではない。

### Exact metadata resource limit

次のlimitはreaderとwriterの両方に対するcanonical validity ruleである。

- Cause Link一つにつきmetadata entryは最大64個
- namespace lengthはdecoded UTF-8 bytesで1から255
- non-null schema lengthはdecoded UTF-8 bytesで1から1,024
- value nestingは最大16 level。entryのroot `value`をlevel 1と数える
- Cause Link一つにつきvalue nodeは合計最大4,096。scalar、array、objectを
  それぞれ1 nodeと数える
- object keyまたはstring value一つにつきdecoded UTF-8 bytesで最大4,096
- complete metadata array valueはcanonical bytesで最大65,536。`[`と`]`を含み、
  member name、colon、周囲のCause Link bytesは含まない

すべてのlimitをunbounded recursionやallocationなしで確認する。limit超過はinvalid。
readerはtruncateせず、writerはpartial persistしない。

このADRは、containing Provenance、Candidate、Blob全体に新しいtotal size limitを
追加しない。既存`messages`とLink countにはそのようなsuccessor limitがなく、
ここで追加するとvalid format-5 recordのexact preservationを遡及的に妨げうる。
新featureはLink単位でboundedである。将来whole-object limitを追加する場合は、
別のcompatibility decisionが必要。

### Historical typed-object compatibility

repositoryを明示的にformat 6へmigrateした後、ordinary readはexactly次をacceptする。

- accepted format-5 ruleに従うSeal v5、Material v1、Provenance v1の組
- このADRに従うSeal v6、Material v1、Provenance v2の組

cross-pairingはinvalid。Seal v5はProvenance v2を参照できず、Seal v6は
Provenance v1を参照できない。Cause targetとprevious-revision IDは、どちらの
valid Seal generationを参照してもよい。graph derivationは両generationを
ADR 0023に従って扱い、metadataを解釈しない。

historical readerはschema-exactかつread-only。unknown v5 memberをacceptせず、
stored metadataを合成せず、objectを書き換えず、新しいv5 objectをemitしない。
successor inspection schemaから示す場合、historical v5 Linkのlogical metadataは
`[]`とし、Provenance v1 bytesに存在すると主張するのではなくhistorical projection
であることを明示する。

format-5 runtimeはrepository format 6をrejectし、Seal v6、Provenance v2、
Candidate v6をreadしない。`messages`の維持はfield meaningのcompatibilityであり、
old-reader compatibilityではない。

### Candidate compatibility

format-6 repository readerはexact Candidate v5とCandidate v6 fileをacceptする。
Candidate v5のreadはside-effect freeで、各Linkのlogical metadataを`[]`へproject
する。別途authorizedされた最初のCandidate mutationは、結果のCandidate全体を
v6としてwriteする。v5 fieldとmessageをすべてexactに保存し、その同じauthorized
operationがselected Linkへmetadataを与えない限り、empty metadataを追加する。

Candidate v5のopen、inspection、comparison、validationではupgradeしない。
rejected mutation inputはoriginal Candidate bytesを変更しない。新しいformat-6
Candidateは常にv6。

Candidate v5を直接publishすることは禁止する。publicationは最初にexact in-memory
v6 projectionを構築、validateし、Provenance v2とSeal v6を作成し、既存のexpected-old
およびobservation revalidationを行い、その後だけone-REF CAS workflowへ進む。
hidden preliminary actionとしてold Candidateをin-place rewriteしない。

### Repository migration

repository format 5から6へのmigrationは、明示的かつ別途authorizedされた一つの
repository transactionとする。exact CLI spellingはCLI/output companionが所有する。
open、inspection、Candidate mutation、publication時にautomatic migrationしない。

migration operationは次を行う。

1. accepted repository-wide process writer guardを取得する
2. exact format-5 configとcomplete successful format-5 fsckを検証する
3. complete canonical REF-manifest map、Candidate-file byte map、physical
   loose-object identity setをcaptureする
4. すべてのCandidateがexact Candidate v5であり、すべてのreachable typed objectが
   format 5でvalidであることを確認する
5. exact format-6 configをbufferする
6. 三つのmapをrecaptureし、byte-for-byte equalityを要求する
7. config fileだけをexact format-6 bytesでatomically replaceし、fileとcontaining
   directoryをsynchronizeする
8. repositoryをformat 6としてreopenし、retained v5 object、Candidate projection、
   REF、complete graphをvalidateし、そのreadback後だけsuccessをemitする

transactionはimmutable Blobのrewrite/copy、REF move、tag change、Candidate rewrite、
metadata fabrication、Seal creationを行わない。canonical mutationは一つのatomic
config replacementだけなので、interruption後はexact format-5 configまたはexact
format-6 configのいずれかになる。durabilityまたはreadback resultがambiguousなら
uncertainとして報告し、automatic second migrationを起動しない。

Format 6はdowngrade operationを持たない。format-6 repositoryにはformat 5が
解釈できないv6 objectやCandidate fileが存在しうる。

### Dump、load、closure

既存`universal-blob-v1` documentは、accepted format-4-to-format-5 migration
boundary専用のままとする。format 6のsourceまたは生成に拡張しない。

最初のformat-5-to-format-6 transitionは上記config-only migrationを使い、
二つ目のdump/load protocolを導入しない。format-4 repositoryをrecoverする場合、
operatorはまずaccepted format-4-to-format-5 extract/loadをabsent targetへ行い、
format-5 resultをverifyし、その後format-5-to-format-6 migrationを別途authorizeする。

repository copy、outer-Git synchronization、将来のlogical dump/load surfaceは、
retained v5/v6 immutable BlobのbytesとIDをすべてexactに保存し、current REF manifest
とCandidate bytesも保存しなければならない。closure operationはhistorical v5
objectをv6としてre-encodeしたり、messageからmetadataを推論したりしない。

### ADR 0028 change identity

Format-6 assessment-free change identityはsuccessor schema
`sealgraph/upstream-change/v2`を使う。ADR 0028のexact member orderを維持する。

```text
schema, before_seal, after_material, after_root, after_draft, after_cause_links
```

`after_cause_links`にはrequired metadata arrayを含むcomplete canonical format-6
Cause Link valueを入れる。他のfield meaningはADR 0028を維持する。format-6 operation
は、すべてのmetadata arrayがemptyでもv2 change IDだけをemit、compareする。
したがってmetadataのaddition、replacement、removalは`change_id`を変える。

既存immutable v1 upstream-changeまたはAssessment Blobはrewriteしない。migrationは
Assessmentをfabricateせず、一つもadoptしない。別のADR 0028 storage successorが、
Assessment referenceのpersist方法とhistorical v1 assessmentのrepresentationまたは
rejectionを決定する。このmetadata companionはそのreference pathを作らない。

## Alternatives

### すべてのformat-5 objectをformat 6としてrewriteする

schema bytesによりProvenanceIDとSealIDが変わり、graph-wide reference rewriteが
必要となり、unchanged historical-ID addressabilityを損なうためreject。

### version変更なしでformat 5へmetadataを追加する

format-5 readerはunknown memberをrejectし、exact bytesがtyped identityを決める。
同じschemaに二つの意味を作るためreject。

### migration後もすべてのhistorical objectをrejectする

REF headとexact Cause referenceを利用不能にするか、禁止したidentity rewriteを
必要とするためreject。限定されたv5 readerは一つのgraph meaningを維持し、format 4
のincompatible parent semanticsとは本質的に異なる。

### repository migrationで全Candidate fileを書き換える

one-file format transitionをmulti-file transactionにし、operatorがrepository
transitionだけを要求してもmutable WIPを変えるためreject。side-effect-free v5 read
とauthorized edit時のwriteがCandidate boundaryを保存する。

### Format 6へAssessment referenceを追加する

ADR 0028の別途gated storage featureを取り込むため、このADRではreject。
ここではassessment-free change preimageだけを更新する。

### 大きくなったCause Linkにupstream-change/v1を再利用する

同じschema identifierの下でexact nested value meaningが変わるためreject。
Version 2がidentity transitionを明示する。

## Consequences

- historical format-5 SealID、ProvenanceID、message、REF head、tag、exact Cause
  referenceは変更されずaddressableなまま
- 新しいmetadata-bearingまたはmetadata-empty publicationはformat-6 identityを使う。
  semantically unchangedなv5-to-v6 resealでもSealIDは維持されない
- format-6 readerは限定されたexact v5 typed-object/Candidate readerを持つが、
  writer formatはsuccessor一つだけ
- immutable object、REF、Candidateを書き換えないためrepository migrationは小さくatomic
- legacy messageやProvenanceへ遡及的total size limitを課さず、metadataをdeterministicに
  boundする
- format-5 binaryはformat-6 repositoryに対してfail closed
- ADR 0028 change IDはv2へ移り、metadata変更時に変わる
- Assessment adoption storage、CLI grammar、public inspection schema、query、registry、
  validator runtimeは、別途decide、authorizeされるまでunavailable

## Implementation notes

このADRがProposedである間、またはCLI/output companionがacceptされる前は、
implementation actionを承認しない。

complete decision setがacceptされ、別途implementation authorityが与えられた後は、
次を行う。

- v5 readerをv6 writerから分離する
- metadata depth、node、byte limitをiterativeにvalidateする
- canonical valueとresource limitの全境界にexact fixtureを追加する
- v5 object/Candidate readにwrite pathがないことを証明する
- migrationがconfigだけを変更すること、transitionとcomplete graphをreadbackすることを
  証明する
- v5/v6 mixed graphのclosure、cycle、staleness、history、admissionをtestする
- Candidate v5 projectionとrejection atomicityをtestする
- empty/add/replace/remove metadataについてupstream-change/v2 IDをtestする
- normative documentは別途authorizedされたimplementation change setだけで同期する

## Review

author self-reviewは次を確認した。

- exact schema separationとold-reader failure
- historical IDとmessageの保存
- mixed graph semanticsを作らないmixed-generation graph behavior
- Candidate readとmutationの境界
- config-only migrationのatomicityとreadback
- canonical valueとlimitの完全性
- dump/loadを別用途へ流用しないこと
- Assessment-referenceへscope拡張しないADR 0028 change-preimage境界

independent reviewとOperator acceptanceはpending。reviewでは次に答える。

1. format 6内のexact v5 readingは、reject済みformat-4 mixed-semantics readerへの回帰
   ではなく、安全なcompatibility boundaryか
2. Seal v6、Provenance v2、Candidate v6、upstream-change/v2はすべてのschema-meaning
   collisionを防ぐか
3. config-only migrationはunsafeなhalf-migrated stateを作らず、すべてのimmutable ID、
   REF、message、Candidate byteを保存するか
4. canonical valueとすべての新しいresource dimensionはexactかつboundedか
5. ADR 0028 change identityを満たし、Assessment storageを先取りしていないか

## Evidence

- **E-SM-001:** accepted ADR 0029がsemantic boundary、required metadata model、
  compatibility distinction、companion set、implementation gateを定義する
- **E-SM-002:** accepted ADR 0025がformat-5 schema bytes、unknown-member rejection、
  canonical string、Provenance identity、SealID commitmentを定める
- **E-SM-003:** accepted ADR 0023がexact Cause targetとprevious-revision assertionだけから
  graphを導出する
- **E-SM-004:** accepted ADR 0026はformat-4 semantic breakがordinary mixed readerでなく
  isolated migrationを必要とした理由を示す。そのreject premiseはADR 0029の
  graph-preserving extensionには適用されない
- **E-SM-005:** accepted ADR 0028がassessment-free change identityを定め、
  Assessment-reference storageを別のsuccessor decisionとして残す
- **E-SM-006:** checked-in format-5 runtimeとnormative documentはrepository format 5、
  Seal v5、Provenance v1、Candidate v5、three-field Cause Linkを使う

| Claim | Evidence | 必要なimplementation action |
| --- | --- | --- |
| C-SM-001: explicit repository/typed-schema successorを使う | E-SM-001, E-SM-002, E-SM-006 | A-SM-001, A-SM-002 |
| C-SM-002: v5 objectとexact historical IDを保存する | E-SM-001, E-SM-002 | A-SM-001, A-SM-003 |
| C-SM-003: exact v5 historyをreadしつつv6だけをwriteする | E-SM-002, E-SM-003, E-SM-004 | A-SM-001, A-SM-003 |
| C-SM-004: deterministic bounded metadata bytesを定義する | E-SM-001, E-SM-002 | A-SM-001, A-SM-004 |
| C-SM-005: authorized mutationまでCandidate bytesを保存する | E-SM-001, E-SM-006 | A-SM-002, A-SM-004 |
| C-SM-006: coherent readback付きでconfigだけをmigrateする | E-SM-001, E-SM-004, E-SM-006 | A-SM-003, A-SM-004 |
| C-SM-007: assessment-free change identityをv2へ移す | E-SM-001, E-SM-005 | A-SM-001, A-SM-004 |
| C-SM-008: Assessment-reference storageを除外する | E-SM-001, E-SM-005 | A-SM-005 |

implementation action identityは次のとおり。

- **A-SM-001:** exact v6 domainとcanonical typed-object codecを実装する
- **A-SM-002:** exact Candidate v5 projectionとv6 persistenceを実装する
- **A-SM-003:** explicit config-only repository migrationとhistorical v5 readingを実装する
- **A-SM-004:** canonical、limit、identity、migration、graph、atomicity fixtureを追加する
- **A-SM-005:** Assessment-reference storageをこのimplementation sliceから除外し、
  accepted assessment-free change-preimage ruleだけを同期する

## Follow-ups

- exact Proposed ADRをOperator acceptance前にindependently reviewする
- このexact storage contractに対する別のCLI/output companionをdraftする
- 両companion ADRがindependently reviewed、acceptedされ、implementationが別途
  authorizeされるまでnormative conversionもimplementationも開始しない
