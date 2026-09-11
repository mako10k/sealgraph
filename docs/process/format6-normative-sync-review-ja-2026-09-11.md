# format 6 規範文書同期候補 オーナーレビュー資料

Status: Step 2 complete / owner route `REVIEW` / independent review pending

この文書はレビュー支援用の非規範資料である。英語の4ファイルが候補の
正本であり、この日本語訳は変更箇所全体を確認するためのものである。
変更されていない箇所はレビュー範囲外として省略する。

2026-09-11の最初のオーナーレビューで、format 6追加コマンドの具体的なhelp
navigationを明記するため`REVISE`が指示された。本候補はその指示を反映して
Step 1から再開したrevisionであり、以前のdigestやレビュー状態を継承しない。

そのrevisionはオーナーの`REVIEW`選択後にStep 3独立レビューを完了した。Step 4で
オーナーはF-01〜F-04を解消するため`REVISE`を選択した。本候補はselector世代、
`upstream-change/v2`、Candidate v5 direct publication、明示的acceptance criteriaを
反映して再度Step 1から開始する。前回Step 3のdigestやレビュー状態は継承しない。

## 候補スナップショット

- `docs/requirements.md`: `828bf756da52243d7c5ac5c8a87917820a471574ab16e62353543851a956c6e4`
- `docs/architecture.md`: `18352065ba6874691516c9fbe5d0cf51a618618078d0fe188f0b4fe0ee3cdab1`
- `docs/storage-format.md`: `c01fd58c8d655dfc248c20fb92d41078b5faf1bc7a799d7d5b441dea1a6d6f88`
- `docs/cli.md`: `95225e234c0bca875cc8b34c2a9d4c748888fa0e6df219c46249c81c840cd3ad`

## 権威と変更境界

- 根拠: Accepted ADR 0029、0030、0031。
- 既存権威の扱い: format 5契約は維持し、format 6後継契約を追加する。
- 対象: format 6のmetadata、世代付きstorage、明示的migration、CLI/output。
- 対象外: metadata query、Assessment、format 7、release、実リポジトリ移行、
  Git sidecarの新機能、既存format 4 migrationの変更。
- 互換性: 新規repositoryはformat 5のまま。format 6は明示的migrationのみ。
  歴史的なv5/v1 IDとbytesは変更しない。
- 不明点: この候補を規範文書として受理するかはオーナー判断待ち。

## 自己レビュー

- ADR 0029のgeneric・opaque・namespaced metadataと秘密平文禁止を維持した。
- ADR 0030のv5/v1履歴保持、v6/v2後継write、config-only migration、
  cross-generation pairing拒否を維持した。
- ADR 0031の単一namespace mutation、legacy Link authoringのmetadata保持、
  v3 schema matrix、human表示、migration receiptを維持した。
- F-01: `@SEAL_TOKEN`をcurrent repository formatが許可するcanonical Sealへ修正し、
  format 5はv5のみ、format 6はvalid v5/v6 exact pairingを受け入れることを明記した。
- F-02: Assessment-free `upstream-change/v2`、complete format-6 Cause Link、
  metadata-sensitive `change_id`、v1/Assessment非書換えを同期した。
- F-03: Candidate v5 publicationはfileを予備書換えせず、memory上のexact v6
  projectionをvalidateしてProvenance v2/Seal v6を作ることを同期した。
- F-04: 下記にexact acceptance criteriaを列挙した。
- 実装やテストにしか存在しない新しい意味、namespace、完了条件は加えていない。
- optional/futureのquery、Assessment、sidecarを受入条件へ昇格していない。

## 受入基準

- 4つの規範文書がAccepted ADR 0029〜0031の必須contractと矛盾せず、format 5の
  accepted contractを維持する。
- format 6のgeneric metadata、canonical limits、identity、graph independence、
  v5/v6 exact pairing、明示的config-only migrationが欠落なく同期される。
- `@SEAL_TOKEN`がcurrent repository formatで有効な全canonical Seal世代を解決し、
  format 5でSeal v6を誤って受け入れない。
- Assessment-free `upstream-change/v2`がcomplete format-6 Cause Linkをcommitし、
  metadata changeが`change_id`へ反映され、既存v1/Assessment Blobを変更しない。
- format-6 repositoryのCandidate v5 readはside-effect freeであり、mutationだけが
  fileをv6へupgradeし、publicationは予備file rewriteなしのin-memory v6 projection
  からProvenance v2/Seal v6を作る。
- format 6追加コマンドの8つのhelp navigationがdocumented spellingどおり成功し、
  repository bootstrapやmutationを行わない。
- query、registry、validator、Assessment-reference、release、実migration、Git
  sidecar implementationはoptional/futureまたはout of scopeのままである。

## `docs/requirements.md` 変更箇所の完全な日本語訳

### Status

これはformat 5およびformat 6の規範契約である。Accepted ADR 0023、0025、
0026、0027がformat 5境界を定義し、Accepted ADR 0029、0030、0031が
format 6のCause Link metadata、storage/migration、CLI/output境界を追加する。
新規repositoryはformat 5で初期化され、format 6へは明示的な5→6 migration
によってのみ移行する。通常利用では、どちらもformat 4をfail closedする。

### Sealの親境界

format 5とformat 6にはintrinsic parent fieldがない。revision evidenceは、
immutable observerが作るCause Link内にのみ存在する。

### format 6 Cause Link metadata

format 6 Cause Linkは必須の`metadata`配列を追加する。各entryは、一意で
空ではないUTF-8 `namespace`、nullまたは空ではないUTF-8 `schema`、1つの
bounded canonical JSON `value`を持つ。entryはnamespaceのbyte順でsortする。
metadataはcoreに対してopaqueで、identity-bearingかつobserver-localである。
新しいedge、authority、validation成功、approval、trusted eventを意味しては
ならない。秘密平文は禁止のままである。format 5 Cause Linkはshared inspection
viewでのみempty metadataへprojectされ、歴史的bytesとIDは変更されない。

### Selector世代

Repository-wideな`SEAL_TOKEN`は4〜64文字のlower-case hexであり、native ODB全体で
一意に解決され、current repository formatが許可するcanonical Seal generationへ
decodeされなければならない。したがってformat 5はSeal v5だけを受け入れ、format 6は
required Provenanceとexactにpairするvalid Seal v5/Seal v6を受け入れる。

### Candidate authoring

`add`、`link`、`unlink`、およびformat 6の`link-metadata set/remove`は、
1つのdestination REFについて次のCandidate stateを編集する。attachment mutation
commandは引き続き存在しない。

format 6では、legacy `add`/`link` authoringは同じexact targetの既存metadataを
保持し、新規targetにはempty metadataを作る。`link-metadata set`だけが1つの
complete namespace entryを追加または置換し、`link-metadata remove`だけが
既存namespaceを1つ削除する。両操作は既存CandidateまたはREF baselineとexact
Cause targetを要求し、選択されていないfieldをすべて保持し、complete successor
graphをvalidateして、expected-old Candidate versionを1回置換する。migration、
seal、REF移動、metadata意味解釈、batch mutationは行わない。

format-6 repositoryでCandidate v5を読む処理はside-effect freeで、Link metadataを
memory上でemptyへprojectする。別途authorizeされたCandidate mutationが成功した
場合だけcomplete successorをCandidate v6として書く。PublicationはCandidate v5
bytesを直接publishせず、fileをhidden preliminary actionとして書き換えない。exactな
in-memory Candidate-v6 projectionを構築・validateし、Provenance v2/Seal v6を作り、
既存のexpected-old observationとone-REF CAS workflowへ進む。

### Machine output

format 5の成功machine outputは`show/v2`、`candidate-show/v2`、
`candidate-compare/v2`、`status/v3`、`stale/v2`、`graph/v2`、`impact/v2`、
`log/v2`、`linklog/v2`、`compare/v2`、`fsck/v2`を使用する。format 6では、
Link、assertion source、typed schema generation、comparisonを含む各schemaが
`/v3`へ進む。`status/v3`と`stale/v2`は変わらない。format 6 shared Link
recordは完全なmetadata、Seal/Provenance世代、target別・namespace別の詳細な
comparison recordをADR 0031どおり含む。変更された意味を古いschemaで出しては
ならない。

format 6のAssessment-free change identityは`sealgraph/upstream-change/v2`を使う。
ADR 0028のmember orderを維持し、`after_cause_links`はmetadataを含むcomplete canonical
format-6 Cause Linkを持つ。metadata arrayがすべてemptyでもformat-6 operationはv2
change IDだけをemit/compareし、metadataの追加・置換・削除は`change_id`を変える。
既存immutable v1 change/Assessment Blobは書き換えず、Assessment-reference persistenceは
別途gateされたままである。

### Human output

default human inspectionは任意content bytesを直接出してはならず、boundedで
曖昧でないescapeを使用する。format 6では、選択したLink metadataをnamespace、
schema、bounded canonical JSON全体として表示する。control characterはescape
されたままにし、format 5からprojectされたemptyを明示する。exact content
extractionは、metadataや追加newlineをstdoutへ混ぜない明示的bytes-only modeで
のみ提供できる。

### format 6 migration boundary

format 5→format 6の唯一のtransitionは次である。

```sh
sealgraph migrate repository --from 5 --to 6 [--format human|json]
```

transactionはexact format 5 stateをvalidateし、canonical object、REF manifest、
Candidate bytesをsnapshotし、`config`だけをatomicに置換し、format 6で再openして
完全なfsck/readbackを実行し、保持stateが不変であることを証明する。歴史的Seal、
Provenance、Candidate、Material、REF、tag、content Blobを書き換えてはならない。
commit後のdurability/readback/output uncertaintyは、migrationが既にcommit済みの
可能性を示し、自動retryを認めてはならない。

format 6は歴史的Seal v5/Provenance v1 pairとCandidate v5 fileをstrictに読み、
後継writeではSeal v6/Provenance v2とCandidate v6だけを書く。Seal v6は
Provenance v2とのみ、Seal v5はProvenance v1とのみpairできる。Materialはv1の
ままである。mixed-generation observationはこのexact pairingでのみvalidで、
metadataを歴史的identityへprojectしない。

## `docs/architecture.md` 変更箇所の完全な日本語訳

Statusは、format 5/6 typed Blob、parentless Candidate、Cause-scoped revision、
generic Link metadata、明示的5→6 migration、isolated format 4 extraction、
atomic universal migration load、inspection、REF manifest、scoped tag、move、
recovery coreを実装済みとする。Git viewは別工程のままである。

domain boundaryは`internal/domain`、`internal/domain/v5`、`internal/domain/v6`
からなり、generation付きAttachment、CauseLink、metadata、Material、Provenance、
Seal viewを扱う。format 4 payloadはisolated migration verifierからのみ到達でき、
通常repository APIはversioned format 5/6 typeだけを使用する。

canonical boundaryは`internal/canonical/v5`と`internal/canonical/v6`からなる。
v6 codecは全Cause Linkへbounded・namespace-sort済みcanonical JSON metadataを
追加し、Seal v6/Provenance v2 exact pairingを強制する。v5 codecは歴史的bytesを
exactに保ち、v6 memberを受け付けない。Material v1はsharedのままである。format 6の
Assessment-free change identityはgeneration-specificな`upstream-change/v2` canonical
recordを使い、Assessment referenceはこのsliceの対象外である。

repository coordinationでは、format 5のwhole-record Cause authoringとformat 6の
metadata-preserving legacy Cause authoringを区別する。さらにsingle-namespace Link
metadata set/remove、expected-old Candidate replacement、および保持stateをreadback
するexact config-only 5→6 migrationを追加する。

native object storeのimmutable loose object、SHA-256 envelope、full ID、loose REF、
canonical pack/packed-ref不使用はformat 5とformat 6の双方で維持する。

Seal publicationは、format-6 repositoryでCandidate v5をloadした場合、そのexactな
Candidate-v6 projectionをmemory上で構築し、Candidate fileを書き換えずにvalidateと
canonical publicationを行う。

## `docs/storage-format.md` 変更箇所の完全な日本語訳

Statusは、新規native format 5を初期化し、明示的config-only migration後は
strictなformat 5 historyを保持しながらnative format 6を書く。format 4には
明示的`universal-blob-v1` extract/load boundaryだけを提供し、通常readerは
format 1〜4を解釈しない。

current runtimeはgraph cacheをpersistしないため、`stale --scan`はsemanticに
同一である。

format 6 configはformat 5から先頭値だけが変わる。

```text
repository_format = 6
object_format = sha256
ref_format = manifest-v1
```

唯一のtransitionはADR 0030の明示的atomic config-only commandであり、object、
REF manifest、tag、Candidate、identityを書き換えない。format 6 repositoryは
exact historical Seal v5/Provenance v1とCandidate v5を受け入れるが、後継writeは
Seal v6/Provenance v2とCandidate v6のみである。cross-generation pairingはcorrupt。

Provenance v2のmember orderは`schema, root, draft, cause_links`のままで、Cause
Link orderは`target_seal, previous_revision_seal_of_target_seal, messages,
metadata`である。

metadataはunique namespaceのbyte順でsortされた必須arrayである。entry orderは
`namespace, schema, value`。schemaはnullまたはemptyでないUTF-8、valueは1つの
bounded canonical JSON valueである。closed value modelはnull、boolean、shortest
signed 64-bit integer、bounded UTF-8 string、array、object。object keyはuniqueで
byte順にsortし、host map decode前にduplicateをrejectする。floating point、
negative zero、exponent、invalid UTF-8、depth/node/byte超過、unknown memberは
fail closedする。exact boundとescape ruleはADR 0030を規範とする。

Candidate v6はCandidate v5 member orderを維持し、各Cause Link内へmetadataを
追加する。Candidate fileはmutable canonical JSON plus LFのままである。historical
Candidate v5 readerはmemory上だけempty metadataへprojectし、fileは実際に成功した
Candidate mutationによってのみv6へupgradeされる。

PublicationはCandidate v5 bytesを直接publishせず、Candidate v5 fileをpreliminary
stepとして書き換えない。memory上でexact Candidate-v6 projectionを構築・validateし、
Provenance v2/Seal v6を作成し、expected-old/coherent-observationを再validateしてから
one-REF CAS workflowを使う。

Seal v6のmember orderは`schema, material, provenance`のまま。schemaは
`sealgraph/seal/v6`で、provenanceは`sealgraph/provenance/v2`でなければならない。
Materialは`sealgraph/material/v1`のまま。complete metadataはProvenance identityと
Seal identityに影響するが、structural/revision edgeは作らない。

format 6のAssessment-free change identityは`sealgraph/upstream-change/v2`を使い、
member orderは`schema, before_seal, after_material, after_root, after_draft,
after_cause_links`である。`after_cause_links`はmetadataを含むcomplete canonical
format-6 Cause Linkを持つ。metadataがすべてemptyでもformat-6 operationはv2 change
IDだけをemit/compareするため、metadata mutationは`change_id`を変える。既存immutable
v1 upstream-change/Assessment Blobは書き換えず、migrationはAssessmentをfabricate/adopt
しない。Assessment-reference storageはADR 0028で別途gateされたままである。

## `docs/cli.md` 変更箇所の完全な日本語訳

Statusは、standalone CLIがformat 5 repositoryを作成し、format 5/6をopenする。
ADR 0023、0025、0026、0027がformat 5 boundary、ADR 0029、0030、0031がformat 6
Link metadata、storage/migration、outputを定義する。

`@SEAL_TOKEN`はrepository-wideで一意な4〜64文字lower-hex native object prefixで、
current repository formatが許可するcanonical Sealへdecodeされる。

format 6追加コマンド群は、既存のrepository-independentなhelp規則に従い、次の
parent/leaf navigationから到達できる。

```sh
sealgraph help migrate
sealgraph help migrate repository
sealgraph migrate repository --help
sealgraph help link-metadata
sealgraph help link-metadata set
sealgraph link-metadata set --help
sealgraph help link-metadata remove
sealgraph link-metadata remove --help
```

`sealgraph migrate repository --from 5 --to 6 [--format human|json]`は、両format
optionをexactに1回ずつ要求する。format 5 repositoryをvalidate/captureし、config
のみatomic置換し、format 6でreopen/fsckしてobject、REF manifest、Candidate bytes
が不変であることを検証する。format推測、history rewrite、downgrade、batch path、
commit可能性があるmigrationのretryは行わない。成功は
`sealgraph/repository-migrate/v1`を出し、commit後output failureは
`MIGRATION_COMMITTED_OUTPUT_UNDELIVERED`と`fsck`案内を出す。

format 6の`add`/`link`はexisting exact targetのmetadataを保持し、新規targetには
empty metadataを作る。`unlink`はLink全体を削除する。

`link-metadata set`はREF、exact TARGET、NAMESPACE、schema/no-schemaの一方、
value-json/value-fileの一方を要求する。`link-metadata remove`はREF、exact TARGET、
NAMESPACEを要求する。format 6専用であり、setはcomplete namespace entryを追加・
置換し、removeはexisting namespaceだけを削除する。named fileはregular
non-symlink、`-`はexact stdin。canonicalizationはduplicate object keyとmodel外valueを
rejectする。選択外のCandidate/Link fieldを保持し、expected-old replacement前に
validateし、成功時に`sealgraph/link-metadata-mutation/v1`を出す。metadata解釈、
migration、seal、batch targetは行わない。

format-6 repositoryでCandidate v5をsealする場合、最初にexact Candidate-v6 projectionを
memory上で構築・validateし、Provenance v2/Seal v6を作る。Candidate fileをhidden
preliminary actionとして書き換えない。

format 6 repositoryでは`show`、`candidate show`、`candidate compare`、`graph`、
`impact`、`log`、`linklog`、`compare`、`fsck`がADR 0031の`/v3` schemaを使う。
`status/v3`と`stale/v2`は不変。shared recordはexact generationとcomplete metadataを
公開し、comparisonはtarget別Cause changeとnamespace別metadata changeを公開する。
human outputはgenerationをlabelし、bounded canonical metadata value全体を表示し、
historical projected emptyとstored v2 emptyを区別する。

format 6のAssessment-free change identityは`sealgraph/upstream-change/v2`である。
`after_cause_links`はmetadataを含むcomplete format-6 Cause Linkを持ち、metadata arrayが
emptyでもformat-6 operationはv2 change IDだけをemit/compareする。したがってmetadata
mutationは`change_id`を変える。既存v1 change/Assessment Blobは書き換えず、ここでは
Assessment-reference commandを追加しない。

future sidecarはformat 5固定ではなく、同じsupported native `.sealgraph` bytesを使う。

## オーナールート

前回snapshotでは、2026-09-11にオーナーが`REVIEW`を選択し、Step 3完了後の
Step 4で`REVISE`を選択した。新しいdigestのStep 1候補について、オーナーは
2026-09-11に追加質問なしの`REVIEW`を選択した。候補4文書を変更せずStep 3の
独立レビューを実行し、その後Step 4へ進む。

- `REVISE`: Step 1へ戻して候補を修正する。
- `REVIEW_THEN_REVISE`: 独立レビュー後、必ずStep 1へ戻す。
- `REVIEW_THEN_DECIDE`: 独立レビュー後、Step 4で判断する。
- `REVIEW`: 追加質問なしで独立レビューし、Step 4へ進む。

提案する独立レビュー入力は、上記4 digestのsnapshotについて、ADR 0029–0031との
contradiction、F-01〜F-04の解消、上記acceptance criteria、format 5 compatibility
regression、optional/future scopeの混入、外部schema namespaceの誤割当がないかを
確認することである。
