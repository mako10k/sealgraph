# Cause Link拡張可能metadata要件 独立レビュー — 2026-09-10（日本語訳）

この文書は、英語の独立レビュー記録全体に対するレビュー支援用の完全な日本語訳である。英語版が正本であり、この訳は別の要件や判断を追加しない。

- 状態：`completed`
- レビュー結果：受入基準5の明確化までacceptanceを保留
- ownerが選択したroute：`REVIEW`
- candidate revision：1
- candidate path：
  `docs/process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
- review前後のcandidate SHA-256：
  `a419872a4971db1d8fcaebe797fde085d9a9ef107b15f93f72eb600138e5cc96`
- review-input path：
  `docs/process/cause-link-extensible-metadata-requirement-review-input-2026-09-10.md`
- review前後のreview-input SHA-256：
  `6bb9e8243cb60b1e3a8412912baff1b5736cc7879abb26bc7df76ec480bcb178`
- 変更境界：独立reviewerはfileを変更せず、candidateはreview全体を通じてbyte-identicalだった

このreviewはcandidateをacceptせず、ADR、normative conversion、format change、implementation、migration、commit、push、release、deploymentを許可しない。

## 結果

独立reviewは完了し、acceptanceを妨げるevidence gapを1件検出した。generic modelのその他の部分は、調査したaccepted authorityと整合する。blocking wordingに実質的に異なる2つの読み方があるため、このcandidate revisionでは`ACCEPT`は次のrouteとして利用できない。ownerは`REVISE`を選択できる。または、review questionを追加もしくは変更する場合に限り`REREVIEW`を選択できる。

## 所見

### F-01 — `evidence gap/unknown` — acceptanceを妨げる

受入基準5は、structural Cause/Revision behaviorが「metadata meaningからbyte-for-byte independent」であることを要求する（candidate 221-222行）。この表現には実質的に異なる2つの読み方がある。

1. graph-semanticの読み方：edge membership、staleness、admission、history、および関連するstructural resultsがnamespace meaningを解釈しない。これは、metadataがedgeを作らずstructural validationを弱められないというcandidateのrule（candidate 130-137行）およびADR 0023のexact edge derivation（ADR 0023 101-125行）と整合する。
2. output-byteの読み方：metadata bytesが変わってもstructural inspectionまたはresult bytesが同一のままである。この読み方は成立しない。metadataはProvenanceIDとSealIDを変え（candidate 130-131行）、assertion-source outputはobserver SealとProvenance identityを含み（ADR 0023 61-70行、ADR 0027 151-166行）、candidate自身がcomparisonと`linklog`にmetadata変更の露出を要求するためである（candidate 176-188行）。

狭いgraph-semanticの読み方は内部的に整合するため、これはcontradictionではない。しかし、現状の基準は明確でもtestableでもない。revisionでは、identityまたはoutput bytesが不変であると暗示せず、意図するinvariantを明記する必要がある。

### F-02 — `evidence gap/unknown` — successor compatibility gateのみ

candidateは、変更されていない既存SealとIDをどのようにaddressableのままにするかをsuccessor decisionで説明するよう正しく要求しつつ、mixed historical-schema readingとmandatory conversionのどちらにするかを未決としている（candidate 191-209行、232-245行）。現在のformat-5 authorityはgeneral dual readerではなくisolated migration boundaryを用い（requirements 448-473行）、public meaningの変更にはsuccessor schemaが必要である（ADR 0027 41-70行）。後続decisionでは、「addressable」がnative typed decoding、old-to-new mapping、またはretained migration evidenceのどれを意味するかを定義する必要がある。F-01の修正後、これはgeneric requirementを妨げない。

### F-03 — `evidence gap/unknown` — successor namespace/storage gateのみ

candidateは`namespace`と`schema`についてexact non-empty、validity、reservation ruleをまだ定めていない（candidate 117-128行、232-245行）。non-normative proposalは可能なpolicyを1つ記録している（proposal 91-120行）が、candidateはそのpolicyを採用しないことを明記している（candidate 50-56行）。これは意図的に未解決のsuccessor choiceであり、contradictionでもgeneric requirementへの独立したblockerでもない。

### F-04 — `optional/future` — acceptanceを妨げない

revision-context profileのproject ownership、最終namespaceとschema、validator support、exact CLI/query spelling、およびquery languageは後続choiceである（candidate 232-245行）。proposalもgeneric metadataと後続validator/query decisionを分離している（proposal 180-200行）。

### F-05 — `out of scope` — acceptanceを妨げない

first-class Cause Link identity、query language、schema registryまたはnetwork retrieval、signature、trusted actor/time、secret storageは明示的に除外されている（candidate 67-79行）。調査したaccepted sourceのいずれも、このrequirementにそれらを要求していない。ADR 0028はSealgraphにおけるindependently active Linksをrejectしている（ADR 0028 721-725行）。

## 合格した重要範囲

- candidateは既存3fieldを保持し、`messages`をsorted、duplicate-free、identity-bearing setのままにしており、現行requirementと整合する（requirements 80-97行）。
- Cause LinkはCauseLinkIDなしでProvenanceにembeddedのままである。metadataは新たなgraph edgeを作らず、ProvenanceIDとSealIDを通じてtransitively committedされ、ADR 0025のidentity boundaryと整合する（ADR 0025 188-214行）。
- candidateはunknown format-5 fieldを追加するのではなくsuccessor formatとCLI decisionを要求している。現行format-5 typed Blobはunknown memberをrejectする（storage format 103-134行）。
- revision-context valueはnamespace-owner claimのままであり、coreのpreference、truth、trust、approval、supersessionを作れず、normative revision boundaryと整合する（requirements 62-70行）。
- generic metadataはADR 0028のAssessment identity、adoption、admissionを代替できない。後続designが継承する帰結として、exact Cause Link metadataの変更はADR 0028の`change_id`を変更し、古いchangeに対するassessmentをinapplicableにする（ADR 0028 174-197行）。
- compatibility textはmessage valueの保持をold-readerおよびSealID compatibilityから分離し、legacy messageからmetadataを推論することを禁止している（candidate 191-209行）。
- ADR 0017はproposedのままであり、non-normative comparisonとしてのみ利用された（ADR 0017 1-42行）。

## Review coverageとverification

独立reviewerはexact candidateとreview input、Accepted ADR 0023、0025、0027、0028、Proposed ADR 0017とそのdetailed proposal、および`requirements.md`、`architecture.md`、`storage-format.md`、`cli.md`の関連部分を調査した。primary agentは、この結果を記録する前に、blocking findingと報告されたsuccessor gateについて引用されたprimary sourceを独立に確認した。束縛された2つのdigestはreview前後に確認された。

## 次のowner decision

- `REVISE`：F-01の曖昧さを除くようcandidateを改訂し、新しいdigestを作成し、self-reviewを繰り返してfirst owner reviewへ戻る。
- `REREVIEW`：このcandidate revisionを変更せず、review questionを追加または変更して、新しい独立reviewを実行する。
- `ACCEPT`：F-01がacceptance-blockingである間、このrevisionでは利用できない。
