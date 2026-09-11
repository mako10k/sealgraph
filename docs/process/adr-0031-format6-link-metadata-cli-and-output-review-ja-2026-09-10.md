# ADR 0031 Format-6 Cause Link metadata CLIとoutput レビュー用日本語訳 — 2026-09-10

この文書は`docs/adr/0031-format6-link-metadata-cli-and-output.md`全体の日本語
review supportです。英語ADRが正本であり、この翻訳は第2の権威あるartifactでは
ありません。

## タイトル、Status、scope

ADR 0031: Format-6 Cause Link metadata CLIとoutput

Status: `Accepted`。

2026-09-10、exact reviewed technical candidateに対するOperatorの明示的なacceptanceに
よりAccepted。

Acceptance record:
`docs/process/adr-0031-format6-link-metadata-cli-and-output-acceptance-2026-09-10.md`。

Execution authorityはseparate。acceptanceはnormative conversion、implementation、
migration、commit、push、release、deployment、その他のexternal writeを承認しない。

Decision owner: Operator。

Requirement authority: accepted ADR 0029とaccepted storage companion ADR 0030。

このADRは、exact metadata mutation command、format-5 authoring continuity、
format-6 human inspection、versioned machine schema、comparison、`linklog`、
format-5-to-format-6 migration command/receiptを所有する。metadata query language、
schema registry、validator runtime、ADR 0028 Assessment-reference commandは定義しない。
このacceptanceはimplementationもnormative conversionも承認せず、それぞれseparate
execution authorityを必要とする。

## Context

format-5 `link`は3-field Cause Link全体を置換する。format 6でomitted metadataを
empty replacementとすると、messages-only callerが、名前すら指定していないstructured
dataを削除できてしまう。一方、同じstructural groupへrepeatable metadata flagを
追加すると、positional groupingとshell quotingが曖昧になる。

format-5 inspectionは、`show`、Candidate inspection、comparison、graph、impact、
log、`linklog`の間でexact Cause Link/assertion-source recordを共有する。metadataは
そのnested recordの意味とexact output bytesを変える。同じschema identifierを再利用
すると、一つのmachine schemaは一つの意味を持つというaccepted ruleに違反する。

Format 6はexact historical format-5 Seal、Provenance、Candidateもreadする。inspection
ではlogical empty metadata projectionを示す必要があるが、historical Provenance v1や
Candidate v5 bytesに`metadata` memberが存在したと主張してはならない。

## Decision

### 1-namespace専用mutation command

public metadata mutation commandは次のとおり。

```text
sealgraph link-metadata set REF --target TARGET --namespace NAMESPACE
  (--schema SCHEMA | --no-schema)
  (--value-json JSON | --value-file PATH_OR_DASH)
  [--format human|json]

sealgraph link-metadata remove REF --target TARGET --namespace NAMESPACE
  [--format human|json]
```

`REF`はlogical Candidate REF一つであり、Seal selectorではない。Candidateがなければ、
current REF headからordinary mutable Candidate baselineを作る。REF missing、root baseline、
exact target Link absent、corrupt Candidate、invalid prospective graphの場合は、Candidateを
作成・変更せずにfailする。

`--target`はexactly onceで、accepted selector grammarを使う。operationのcoherent
repository observationの下でfull SealID一つへresolveし、Candidate内のstored
`target_seal`がそのIDと一致するCause Link一つを要求する。したがってtarget REFのheadが
進んだ後、bare REFはhistorical Linkにmatchしない。diagnosticはresolved full IDを示し、
Candidateをinspectしてexact selectorを使うよう案内する。

`--namespace`はexactly onceでADR 0030に従う。`set`は`--schema`または`--no-schema`
のexactly oneを要求する。前者はnon-null identifier、後者はJSON nullを与える。
namespace schemaをdereferenceまたはvalidateしない。

`set`はexactly oneのvalue sourceを要求する。

- `--value-json JSON`: option argumentとしてcomplete JSON value一つ
- `--value-file PATH_OR_DASH`: named regular non-symlink file、または`-`ならstdinから
  complete JSON value一つを読む

inputはinsignificant JSON whitespaceや、semanticに同じnon-canonical string escapingを
使ってよい。lossy host-map decode前にduplicate object keyをrejectし、closed value model
とADR 0030の全limitをvalidateし、canonicalizeする。末尾のnon-whitespace JSON tokenは
認めない。dynamic selector、environment expansion、interpolation、executable valueは
存在しない。REFのように見えるJSON stringもstringのまま。

`set`はnamespace absentならaddし、presentならcomplete schema/value entryをreplaceする。
`remove`はexact namespaceの存在を要求し、そのentryだけを削除する。同じentryへの`set`
はCandidate bytesを変えずidempotentにsuccessする。absent namespaceのremoveはfalse
changeを報告せずfailする。

両operationはtarget、previous-revision IDs、`messages`、他namespace、content、
attachments、root/draft、publication expectation、他Linkを保存する。complete successor
Candidateをvalidate/bufferしてから、expected-old Candidate-file replacementを一度だけ行う。
invalid inputまたはconcurrent Candidate changeではoriginal bytesを変更しない。Seal creation、
REF move、immutable Blob write、migration、metadata meaning interpretationは行わない。

### 既存Link authoringのcontinuity

format 6でも既存`add ... --target ...`と`link REF --target ...` grammarを維持し、selected
Linkのprevious-revision arrayと`messages` arrayを置換する。existing exact-target Linkでは
complete metadata collectionを保存し、新しいtarget Linkではexact empty metadataを作る。

これはformat-5 whole-record wordingの意図的なsuccessor refinementである。legacy invocation
は、名前を指定していないextension areaを削除できない。metadata変更は`link-metadata set`
または`link-metadata remove`だけで行う。

`unlink REF --target TARGET`はmetadata全体を含むexact-target Link全体を引き続き削除する。
`add REF --root --clear-cause-links`も全Linkと内包metadataを削除する。これらはwhole-Link
deletionを明示している。clear-all-metadata、multi-target mutation、wildcard namespace、
implicit migrationは追加しない。

### Mutation receipt

`link-metadata set/remove`はADR 0022のdestination selectionに従う。terminal stdoutはhuman、
non-terminal stdoutはJSONがdefaultで、`--format`がoverrideする。

JSON receipt schemaは`sealgraph/link-metadata-mutation/v1`、exact member orderは次。

```text
schema, action, ref, target_seal, namespace, before, after,
candidate_schema, prospective_provenance_id, prospective_seal_id
```

`action`は`SET`または`REMOVED`。`before/after`はnullable complete metadata-entry record。
`SET`はnon-null `after`、`REMOVED`はnon-null `before`とnull `after`を要求する。idempotent
same-entry `SET`では両方がequal non-null。Candidate schemaはexactly
`sealgraph/candidate/v6`、prospective IDはresulting valid Candidateから計算したfull lower-hex。

human receiptはaction、REF、full target ID、namespace、schemaまたは`none`、abbreviated
prospective IDを示す。arbitrary valueは表示せず、次のexplicit inspection commandを示す。
failure時はreceiptをemitしない。

### Format-6 shared machine record

format-6 machine documentはADR 0027のcompact UTF-8 JSON+LF、exact member order、full ID、
array-present rule、no-partial-output failureを使う。shared recordは次に置換する。

```text
metadata_entry:
  namespace, schema, value

cause_link:
  target_seal, previous_revision_seal_of_target_seal, messages, metadata

assertion_source:
  observer_seal, observer_seal_schema,
  observer_provenance, observer_provenance_schema,
  target_seal, previous_revision_seal_of_target_seal, messages, metadata

seal_view:
  seal_id, seal_schema, material_id, provenance_id, provenance_schema,
  content_blob_id, content_bytes, attachments, root, draft, cause_links

candidate_view:
  ref, candidate_schema, expected_ref_head,
  prospective_seal_id, prospective_material_id, prospective_provenance_id,
  content_blob_id, content_bytes, attachments, root, draft, cause_links
```

schema fieldはexact v5/v6、provenance v1/v2、candidate v5/v6を示す。すべてのmachine
`cause_link`は`metadata`を含む。historical Provenance v1/Candidate v5では`[]`で、enclosing
schema fieldがADR 0030 historical projectionであることを示す。v2/v6ではcomplete stored
canonical array。unknownだがstructurally validなnamespace/valueをtruncateまたはomitしない。

### Comparison record

format-6 Candidate/immutable comparisonはADR 0027のexplicit baselineとno-inferred-predecessor
ruleを維持する。`changes.cause_links`は次の`cause_links_change`になる。

```text
cause_links_change:
  changed, before, after, records

cause_link_change:
  target_seal, before, after,
  previous_revision_seal_of_target_seal, messages, metadata

metadata_change:
  namespace, before, after
```

`before/after`はcomplete cause-link array、`changed`はexact equalityとのiff ruleに従う。
`records`はcomplete Linkが異なるtargetごとに一つで、full target SealID order。

`cause_link_change.before/after`はnullable complete Cause Linkで、少なくとも一方non-null。
previous-revisionと`messages`はordinary `change` recordで、Link add/remove時はnullable array。
`metadata`は変更namespaceだけを含むnamespace-ordered array。

`metadata_change.before/after`はnullable complete metadata entryで、少なくとも一方non-null。
equal entryはomit。schema-only、value-only、add/removeをmessage/structural previous changeから
区別するが、domain meaningは付与しない。

`linklog`も同じ`cause_link_change`を使う。`--upstream`はexact `target_seal`だけでfilterし、
metadataはstructural history edge、supporting assertion、depth、orderingを変えない。

各`linklog/v3` entryのexact member orderは次。

```text
minimum_newer_depth,
newer_seal_id, newer_seal_schema,
newer_provenance_id, newer_provenance_schema,
previous_seal_id, previous_seal_schema,
previous_provenance_id, previous_provenance_schema,
supporting_assertions, target_revision_observation, changes
```

newer/previous endpoint identityはcomplete。Seal/Provenance schema valueは`seal_view`で
定義したexact generation identifierを使い、各Seal/Provenance pairはADR 0030のvalid
typed pairingのいずれかでなければならない。各`cause_link_change`で、`before`はprevious
endpointが所有し、`after`はnewer endpointが所有する。null sideにはCause Linkがないが、
そのendpoint generationは残る。これにより、endpoint Provenance schemaはProvenance v1から
投影された`metadata: []`とProvenance v2に保存されたempty arrayを区別でき、changed
namespace/Linkごとにgeneration fieldを繰り返さない。

### Format-6 schema matrix

changed recordを含むcommandだけsuccessor schemaへ移す。

| Command | Format 5 | Format 6 |
| --- | --- | --- |
| `show` | `sealgraph/show/v2` | `sealgraph/show/v3` |
| `candidate show` | `sealgraph/candidate-show/v2` | `sealgraph/candidate-show/v3` |
| `candidate compare` | `sealgraph/candidate-compare/v2` | `sealgraph/candidate-compare/v3` |
| `graph` | `sealgraph/graph/v2` | `sealgraph/graph/v3` |
| `impact` | `sealgraph/impact/v2` | `sealgraph/impact/v3` |
| `log` | `sealgraph/log/v2` | `sealgraph/log/v3` |
| `linklog` | `sealgraph/linklog/v2` | `sealgraph/linklog/v3` |
| `compare` | `sealgraph/compare/v2` | `sealgraph/compare/v3` |
| `fsck` | `sealgraph/fsck/v2` | `sealgraph/fsck/v3` |

`status`は`status/v3`、`stale`は`stale/v2`を維持する。Link、assertion-source、typed schema、
comparison valueを含まず意味が変わらないためである。source、manifest、tag、REF、recovery、
raw-content、narrow publication receiptも同様に既存schema/protocolを維持する。

successor top-level member orderはshared record置換後もADR 0027を維持する。
`candidate-compare/v3`と`compare/v3`はnew `cause_links_change`を使う。`linklog/v3`は
top-level orderを維持し、new `cause_link_change` arrayを使い、そのentry orderを上記の
exact typed endpoint orderへ置換する。

`fsck/v3` exact member orderは次。

```text
schema, result, blobs,
seals, seals_v5, seals_v6,
materials,
provenances, provenances_v1, provenances_v2,
candidates_v5, candidates_v6,
refs, tags, active_seals,
historical_or_detached_seal_ids, unreferenced_blob_ids
```

countはnon-negative integer。aggregate seals/provenancesは各generation countの和。
Candidate countはcomplete valid Candidate namespaceをcoverする。ID arrayとphysical/coherent
validationはADR 0023/0025/0030を維持する。

### Human inspection

human `show`、`candidate show`、comparison、graph、impact、log、`linklog`はADR 0022の
terminal-first layoutとsafe-width behaviorを維持する。Seal、Provenance、Candidateには
format/schema generationを表示する。

各Cause Link内でtarget、previous revisions、messages、metadata entriesを分離する。
metadata entryはnamespace、schemaまたは`none`、complete valueをADR 0030 escapingのcompact
canonical JSONで表示する。control characterをliteral emitしない。empty metadataは`none`。
historical v5 projectionは`projected empty from v5`とlabelし、stored v1 contentとしない。

comparison/`linklog`はtarget add/remove、previous change、message change、per-namespace
metadata changeを別labelで示す。progress、correction、reconciliation、supersession、approval、
Assessment meaningを推論しない。valueはADR 0030でboundedであり、そのcanonical limit内で
silent truncationしない。既存safe escapingとstdout/stderr分離を維持する。

各human `linklog` entryは、newer/previous endpointのSeal/Provenance generationを
abbreviated identityと並べて表示する。projected empty metadata valueは所有するv1
endpointに対してlabelし、stored empty valueは所有するv2 endpointに対してlabelする。

### Repository migration commandとreceipt

ADR 0030 transactionはexactly次で起動する。

```text
sealgraph migrate repository --from 5 --to 6 [--format human|json]
```

`--from 5`と`--to 6`を各exactly once要求する。他のpair、current format推論、batch path、
target directory、downgrade、automatic continuation、force optionはない。current directory下の
`.sealgraph`だけを操作し、Gitをdetectせず、ADR 0030のconfig-only transactionだけを行う。

success JSON schemaは`sealgraph/repository-migrate/v1`、exact member orderは次。

```text
schema, from_format, to_format, result,
retained_seals_v5, retained_provenances_v1, retained_candidates_v5
```

formatはJSON integer 5/6、`result`はexactly `MIGRATED`。countはcomplete read-back format-6
inventory。human successも同じtransition/countを示す。

config transition commit後にsuccess outputをdeliverできなければ、可能な限りstderrへ
`MIGRATION_COMMITTED_OUTPUT_UNDELIVERED`を報告し、rerun禁止を案内する。bounded read-only
verificationは`sealgraph fsck --format json`で、successは`fsck/v3`。already-format-6 repository
ではconfig rewriteやsecond migration claimなしでfailする。

### QueryとAssessment boundary

metadata filter、search、query AST、index、namespace registry、schema fetch、validator command
を導入しない。callerはselected Seal/Candidateをinspectし、domain-specific validationはcore外。

ADR 0028 Assessment authoring/adoption/removal/review-state/impact-output successorは別途gated。
このADRはADR 0030でaccepted済みのassessment-free `upstream-change/v2` consequenceだけを
exposeし、Linkやmetadata entryへAssessment referenceを追加しない。

## Alternatives

- **`link`へmetadata flagを追加:** structural replacementとnamespace操作のpreservation ruleが
  異なり、omission/groupingが曖昧になるためreject。
- **既存`link`がomitted metadataをclear:** legacy messages-only invocationが指定していない
  structured stateを破壊できるためreject。
- **generic metadata query language:** accepted contractにはselected-object inspectionで十分で、
  syntax/index/cost/observation scopeは別decisionのためreject。
- **format-5 JSON schema再利用:** changed nested recordとgeneration labelにより同じidentifierへ
  二つの意味が生じるためreject。
- **human outputはmetadata digestだけ:** accepted requirementのinspectable valueを満たさないため
  reject。canonical escapingとboundでcomplete表示可能。
- **historical projection labelを隠す:** projected emptyはProvenance v1/Candidate v5 stored member
  ではないためreject。
- **endpoint generationを各`cause_link_change`で繰り返す:** 一つの`linklog` entry内の
  全Link changeは同じprevious/newer owning Provenanceを共有する。entry-level typed endpoint
  identityでcompleteであり、changed Link/namespaceごとの同じschema contextの重複を避けるため
  reject。

## Consequences

- messages-only callerはsyntaxを維持し、omissionでmetadataを消せない
- mutationはexplicit one-target/one-namespace/Candidate-only
- nested JSONはvalue-fileでshell quotingを避けられ、inlineも使える
- changed recordを含むmachine consumerはsuccessor schema対応が必要。unchanged schemaは維持
- comparison/`linklog`がstructural/message/metadata changeを区別
- historical projectionをstored bytesとしてfabricateせず、mixed-generation `linklog` entryは
  両方のowning endpoint generationを持つ
- complete human valueは大きくなりうるがADR 0030 maximum内
- migrationはexact command/versioned receiptを持つがambiguity時retryしない
- query、schema validation、Assessment commandはunavailableのまま

## Implementation notes

acceptance自体はimplementation actionを承認しない。complete companion setのacceptと別途authority後、
duplicate-key detection、complete Candidate buffering、shared view/diff、exact schema fixture、
v5 projection/v6 emptyの別test、legacy preservation/explicit deletion、idempotence/invalid input/
concurrency/output failure、comparison全class、`linklog --upstream` independence、migration success/
mixed v5-to-v6/v6-to-v5 `linklog`で一方がprojected empty、他方がstored emptyとなるfixture、
migration success/already migrated/pre-commit failure/output-undelivered/fsck readbackを実装・検証する。help、completion、
requirements、architecture、storage、CLI、integrations、planはauthorized implementation change set
だけで更新する。

## Review

author self-reviewは次を確認した。

1. old `link`/`add`はomissionでmetadataを消さない
2. set/removeはexactly one Candidate Link/namespaceでpublication/migration side effectなし
3. changed nested meaningを含むdocumentはversion upし、truly unchanged documentは維持
4. comparison/`linklog`がprevious/message/namespace changeを区別
5. historical projectionをhuman/machineで明示し、`linklog`にtyped previous/newer endpoint
   contextを含める
6. migration output ambiguityはautomatic second mutationを起動しない
7. query/Assessment scopeを除外

initial author self-review LLMThink auditはfatal/error/warningなし。その後independent reviewが
P1の`linklog` endpoint-generation omissionを1件識別した。revision reasoningとcausal auditも
fatal/error/warningなしで、revisionはexact typed endpoint contextとmixed-generation fixtureを
追加した。focused independent rereviewはP0からP3のfindingなしで`PASS`を返し、以前のP1を
resolvedとした。その後Operatorはexact reviewed technical candidateを明示的にacceptした。

## Evidenceとtraceability

- **E-CO-001:** ADR 0029がone-namespace explicit Candidate mutation、separate inspection、detailed
  comparison/`linklog`を要求
- **E-CO-002:** ADR 0030がformat-6 bytes、v5 projection、resource bound、config-only migration、
  upstream-change/v2を確定
- **E-CO-003:** ADR 0027がformat-5 grammar、schema versioning、shared record、comparison/historyを確定
- **E-CO-004:** ADR 0022がterminal-first selection、safe rendering、stdout/stderr、narrow receiptを確定
- **E-CO-005:** current implementationがformat-5 `link` whole-record replacementとnarrow mutation
  receiptを確認

| Claim | Evidence | Required action |
| --- | --- | --- |
| C-CO-001: dedicated one-namespace command | E-CO-001,003,005 | A-CO-001,005 |
| C-CO-002: legacy structural authoringでmetadata保存 | E-CO-001,003,005 | A-CO-001,005 |
| C-CO-003: complete metadata/schema generation表示 | E-CO-001,002,004 | A-CO-002,005 |
| C-CO-004: changed machine contractをversion up | E-CO-002,003 | A-CO-002,005 |
| C-CO-005: field/namespace changeを区別 | E-CO-001,003 | A-CO-003,005 |
| C-CO-006: exact migration command/receipt | E-CO-002,004 | A-CO-004,005 |
| C-CO-007: query/Assessment command除外 | E-CO-001,002 | A-CO-006 |

actionは、metadata mutation/legacy preservation、shared human/machine view、detailed comparison/
`linklog`、migration CLI/receipt、grammar/atomicity/schema/rendering/migration fixtureを実装し、query/
Assessment commandをscope外に保つこと。

## Follow-ups

- exact acceptance recordとreview recordをこのADRとともに保存する
- normative conversion、implementation、migration、repository synchronizationを、それぞれ
  separate authorityを要するlifecycle actionとして扱う
