# Cause Link拡張可能metadata要件 revision 2 独立レビュー — 2026-09-10（日本語訳）

この文書は、英語のrevision 2独立review記録全体に対するreview支援用の完全な日本語訳である。英語版が正本であり、この訳は別の要件や判断を追加しない。

- status：`completed`
- review outcome：acceptanceを妨げるcontradictionまたはevidence gapなし
- ownerが選択したfirst-review route：`REVIEW`
- candidate revision：2
- candidate path：`docs/process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`
- review前後のcandidate SHA-256：`83acad3c80f8dadd0f49271829e80efa6bbeb3f5d8adb23c30f928f1f7876ac2`
- review-input path：`docs/process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`
- review前後のreview-input SHA-256：`afa7ce33dcbeaad76aa93cb8c91cdffb6e83ebf2d66b5fd945008147e43e5087`
- mutation boundary：独立reviewerはfileを変更せず、両review inputはreview全体を通してbyte-identicalだった

このreviewはcandidateをacceptせず、ADR、normative conversion、format change、implementation、migration、commit、push、release、deploymentを許可しない。

## 結果

独立reviewは、contradictionおよびacceptanceを妨げるevidence gapなしで完了した。Revision 2はrevision 1 finding F-01を解消している。受入基準5は、graph-semantic non-interpretationを、identity-bearing metadataによるidentityおよびoutput-byte changeから分離した。exact candidateはownerのStep 4 decisionへ進める。

## 分類済み所見

### R2-F01 — `evidence gap/unknown` — non-blocking

最終namespace ownerとschema identifier、numeric canonical/resource limit、successor persisted/output schema version、historical readingとconversionのpolicy、exact CLI、maintained-profile ownershipは未決である（candidate 105-107行、123-132行、195-213行、240-253行）。

これらは、それぞれのsuccessor designまたはimplementation boundaryをgateするが、generic requirementのacceptanceは妨げない。candidateはdeterministic bound、explicit compatibility treatment、別途acceptedされたpersisted/CLI designを要求している。これはrepositoryのextension discipline（architecture 345-350行）およびaccepted schema authority（ADR 0025 343-366行、ADR 0027 594-624行）と整合する。

### R2-F02 — `optional/future` — non-blocking

general metadata query language、schema validator、Sealgraph-maintained revision-context vocabularyはoptionalな後続作業である（candidate 71-83行、249-253行）。candidateはcanonical readingまたはgeneric metadata mechanismにそれらを要求していない。Proposed ADR 0017とそのproposalはnon-normative comparison inputのままである（ADR 0017 31-42行、proposal 180-200行）。

### R2-F03 — `out of scope` — non-blocking

first-class Cause Link identity/lifecycle、trusted actor/time、signature、secret storage、ADR 0028 Assessmentの代替は正しくout of scopeである（candidate 71-83行、167-178行）。Accepted requirementsはrevision assertionがpreference、truth、trust、approvalを意味することを禁止し（requirements 62-70行）、ADR 0028はAssessment identity/admissionを別途所有するとともにSealgraphのindependently active Linksをrejectしている（ADR 0028 721-743行）。

### `contradiction`

検出なし。

## Review question coverage

1. `messages`は既存のfree-form、identity-bearing contractを保持し、single metadata collectionが唯一の新しいstructured extension pathである（candidate 85-119行）。
2. metadataはCauseLinkIDを作らず、ProvenanceIDとSealIDを通じてcommittedされるProvenance contentのままである（candidate 33-48行、ADR 0025 188-214行）。
3. criterion 5は、structural calculationがnamespace、schema、valueを解釈しないと述べる一方、ID、assertion-source identity、comparison、exact output bytesが変化し得ることを明記している（candidate 225-230行）。ADR 0023はexact targetとprevious SealIDだけからstructural relationを導出し（ADR 0023 101-134行）、ADR 0025はProvenance identity変更がSealIDを変更すると定める（ADR 0025 188-206行）。
4. revision contextにはtrusted actor/time、AssessmentID、change binding、evidence adoption、disposition、admission effectがない（candidate 167-178行）。
5. compatibilityはfield continuityをold-reader support、migration、object readability、SealID preservationから分離する（candidate 195-213行）。
6. Proposed ADR 0017はseparable generic modelだけに用いられ、後続decisionで明示的に処置する必要がある（candidate 54-60行）。
7. 未解決のnamespace、format、CLI、migration、limit choiceは、それぞれ後続designが満たすconstraintを伴うsuccessor gateとして記載されている。いずれもこのrequirementを内部的にincoherentまたはuntestableにしない。

## Review coverageとverification

独立reviewerはexact candidateとreview input、Accepted ADR 0023、0025、0027、0028、Proposed ADR 0017とそのdetailed proposal、および`requirements.md`、`architecture.md`、`storage-format.md`、`cli.md`の関連normative sectionを調査した。primary agentは、この結果を記録する前に、重要なprimary-source claimを独立に確認した。固定された両digestはreview前後に確認された。

## 次のowner decision

- `REVISE`：requirement textを変更する場合はStep 1へ戻る。
- `REREVIEW`：candidate digestを変更せず、review questionを追加または変更してStep 3を繰り返す。
- `ACCEPT`：上記で識別されたexact revision-2 candidate bytesだけをacceptする。
