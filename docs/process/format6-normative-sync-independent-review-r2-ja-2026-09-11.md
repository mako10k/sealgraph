# Format 6 規範文書同期 独立レビューr2日本語訳 — 2026-09-11

Status: completed report translation / non-normative review support

これは、オーナーが追加質問なしの`REVIEW`として経路指定した改訂snapshotに対する、
新しいStep 3独立レビューの完全な日本語訳である。英語の
`format6-normative-sync-independent-review-r2-2026-09-11.md`が正本であり、
以下で特定するexact candidateとreview-input bytesだけをレビュー対象とする。

## レビュー対象snapshot

- `docs/requirements.md`
  - SHA-256: `828bf756da52243d7c5ac5c8a87917820a471574ab16e62353543851a956c6e4`
- `docs/architecture.md`
  - SHA-256: `18352065ba6874691516c9fbe5d0cf51a618618078d0fe188f0b4fe0ee3cdab1`
- `docs/storage-format.md`
  - SHA-256: `c01fd58c8d655dfc248c20fb92d41078b5faf1bc7a799d7d5b441dea1a6d6f88`
- `docs/cli.md`
  - SHA-256: `95225e234c0bca875cc8b34c2a9d4c748888fa0e6df219c46249c81c840cd3ad`
- レビュー入力: `docs/process/format6-normative-sync-review-ja-2026-09-11.md`
  - SHA-256: `1d212d69cb37f0ae4770a78192264777cd5c5f61bda8e797124e006e3c187e33`

レビュー前の5つのdigestは、すべてオーナーがレビューしたidentityと一致した。
レビュー後に再計算し、変更されていないことを確認した。

## 権威と方法

権威あるsourceは、Accepted ADR 0029、Accepted ADR 0030、Accepted ADR 0031、
およびADR 0029が参照するaccepted revision-2 requirement authorityである。
Accepted format-5 ADR 0023、0025、0027は、維持されるformat-5契約とformat-6の
後継契約を区別するために使用した。

独立レビューでは次を確認した。

- レビュー入力に記載されたすべてのacceptance criterion
- 前回所見F-01〜F-04について、accepted sourceおよび改訂candidate bytesに対する処置
- generic metadata semantics、canonical bounds/identity、graph independence、exact
  typed-schema pairing、historical format-5 readability、Candidate-v5
  projection/publication、config-only migration、human/machine inspection、
  `upstream-change/v2`
- format-5 compatibilityとold-reader failure boundary
- optional/futureおよびout-of-scopeの封じ込め
- metadata namespace/schema ownership
- 8つの具体的なparent/leaf help-navigation route

既存実装とテストは非規範の比較evidenceとしてだけ使用した。一時binaryをrepository
外から呼び出し、文書化された8つのhelp routeをすべて確認した。すべて成功し、
`.sealgraph` directoryは作成されなかった。`internal/cli`、`internal/repository`、
`internal/canonical/v6`のfocused testは成功した。

dispositionへ依拠する前に、command-line `llmthink dsl audit`を
`format6-normative-sync-step3-r2-2026-09-11`として実行した。依拠した結果は
`fatal=0`、`error=0`、`warning=0`、`hint=0`であった。

## 前回所見への処置

### F-01 — resolved

selector contractは、`@SEAL_TOKEN`がcurrent repository formatで許可されるcanonical
Seal generationへdecodeされることを要求するようになった。`docs/requirements.md`は
exact boundaryも明記する。format 5はSeal v5だけを受け入れ、format 6はrequired
Provenanceとpairするvalid Seal v5/Seal v6 objectを受け入れる。これはAccepted ADR
0030のmixed-generation contractと一致し、format-6 Sealを除外しなくなった。

### F-02 — resolved

候補はAssessment-free `sealgraph/upstream-change/v2` schemaをrequirements、
architecture、storage-format、CLI文書に記録した。accepted member order、completeで
metadata-bearingな`after_cause_links`、metadata-sensitiveな`change_id`、immutable
v1/Assessment Blob、別途gateされたAssessment-reference boundaryを維持している。

### F-03 — resolved

候補はside-effect-freeなCandidate-v5 read、authorized mutationだけによるCandidate
file upgrade、exact in-memory Candidate-v6 projectionを通じたdirect sealを区別する。
hidden preliminary Candidate-file rewriteを明示的に禁止し、既存のcoherent-observationと
one-REF CAS workflowの前にProvenance v2/Seal v6を作成する。

### F-04 — resolved

レビュー入力は7つのacceptance criteriaを明示的に列挙する。source-authority
consistency、format-5 compatibility、metadata/graph semantics、selector generation、
`upstream-change/v2`、Candidate-v5 publication、help navigation、excluded scopeを
対象とする。

## 分類済み所見

### Contradiction — none — P0 through P3

レビュー対象requirementまたはAccepted ADR 0029、0030、0031とのmaterialな矛盾は
見つからなかった。

### Evidence gap or unresolved unknown — none — P0 through P3

exact revised snapshotには、materialなmandatory-contractまたはreview-evidence gapは
残っていない。Acceptance自体は明示的なStep 4 owner decisionのままであり、この
lifecycle stateはcandidate defectではない。

### Optional or future candidate — none promoted — P0 through P3

Metadata query/filter/AST/index work、schema registry/validator、core-owned
revision-context namespace、Assessment-reference commandはacceptance criteriaへ昇格して
いない。既存のfuture Git-sidecar constraintも、この候補でimplementationを約束して
いない。

### Out of scope — contained — P0 through P3

Release、実repository migration、commit、push、deployment、Git-sidecar
implementation、format 7、既存format-4 migrationの変更、ADR 0028 Assessment
adoption/storageは本レビューの対象外のままである。これらの領域のevidenceを
requirement authorityとして使用していない。

## 受入基準の結果と合格領域

1. 4つの規範文書はAccepted ADR 0029〜0031と整合し、accepted format-5 contractを
   維持している。
2. Generic metadata、canonical limit、identity commitment、graph independence、
   v5/v6 exact pairing、明示的config-only migrationが同期されている。
3. `@SEAL_TOKEN`はcurrent repository formatで許可されるすべてのcanonical Seal
   generationを解決し、format 5でSeal v6を許可しない。
4. Assessment-free `upstream-change/v2`はcomplete format-6 Cause Linkをcommitし、
   metadata changeを`change_id`へ反映し、既存v1/Assessment Blobを維持する。
5. format 6でのCandidate-v5 readはside-effect freeで、mutationだけがfileをupgrade
   する。publicationはpreliminary file rewriteなしのin-memory v6 projectionを使う。
6. 文書化されたformat-6 help-navigation 8経路はrepository bootstrapやmutationなしで
   成功する。
7. Query、registry、validator、Assessment-reference、release、実migration、
   Git-sidecar implementationはoptional/futureまたはout of scopeのままである。

追加の合格項目：

- Metadataはcontaining Cause Link/Provenanceへ埋め込まれたままで、CauseLinkIDや
  independent Link publication lifecycleは追加されていない。
- Legacy `messages`は独立して使用可能なままで、structured metadataへparse/migrate
  されない。
- Unknown namespaceはinspect可能なままだが、coreからtruth、trust、approval、
  validation、graph meaningを与えられない。
- Historical format-5 object bytes/IDは不変で、projected empty metadataとstored v2
  emptinessは区別可能である。
- format-6 legacy `add`/`link`は指定されていないmetadataを保持し、明示的whole-Link
  deletionは引き続きそれを削除する。
- 変更されたmachine recordはsuccessor schemaを使い、変更のない`status/v3`と
  `stale/v2`はaccepted meaningを維持する。
- Sealgraph-owned metadata namespaceまたはexternal schema ownershipは割り当てられて
  いない。

## Step 3 disposition

独立レビューは`completed`である。material findingはなく、変更されていない改訂
snapshotは、選択された`REVIEW` routeに従ってStep 4へ進める。そこで`REVISE`、
`REREVIEW`、`ACCEPT`を選択できるのはOperatorだけである。

本レビューはrequirementをacceptせず、candidate edit、ADR change、implementation
change、commit、push、PR、merge、release、deployment、repository migration、runtime
mutationのいずれもauthorizeしない。
