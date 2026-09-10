# Cause Link拡張可能metadata要件 revision 2 acceptance — 2026-09-10（日本語訳）

この文書は、英語のacceptance record全体に対するreview支援用の完全な日本語訳である。英語版が正本であり、この訳は別の要件や判断を追加しない。

- status：accepted requirement
- decision owner：Operator
- owner decision：`ACCEPT`
- decision date：2026-09-10
- accepted requirement revision：2
- accepted candidate path：`docs/process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`
- accepted candidate SHA-256：`83acad3c80f8dadd0f49271829e80efa6bbeb3f5d8adb23c30f928f1f7876ac2`
- first-owner route：`REVIEW`
- review-input path：`docs/process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`
- review-input SHA-256：`afa7ce33dcbeaad76aa93cb8c91cdffb6e83ebf2d66b5fd945008147e43e5087`
- independent-review path：`docs/process/cause-link-extensible-metadata-requirement-r2-independent-review-2026-09-10.md`
- independent-review SHA-256：`de76431a8e1a0dccf6cca977b84dd771f61964b63d86500c77b6dd7b93cdf03a`
- independent-review result：`completed`。acceptanceを妨げるcontradictionまたはevidence gapなし

## Decision

Operatorは、上記で識別されたexact revision-2 candidate bytesだけを、将来のCause Link extensible-metadata designを統治するrequirementとしてacceptする。別のacceptance recordを用いることで、review済みcandidate bytesを変更せずに保存する。candidateのhistorical headerをreview後に書き換えない。

Accepted requirementは次を定める。

- `target_seal`、`previous_revision_seal_of_target_seal`、`messages`を、messages-only useの継続を含む既存の意味のまま保持する。
- structured Link-local meaningのため、optionalかつgenericでnamespacedな`namespace` / `schema` / `value` metadata mechanismを1つ追加する。
- deterministic、bounded、identity-bearingで、ProvenanceIDとSealIDを通じてcommittedされるmetadataを要求する。
- progress、correction、reconciliationなどのrevision-context explanationを、それらをcore graph stateにせず表現可能にする。
- Cause LinkをProvenanceにembeddedのまま保ち、CauseLinkIDを作らない。
- structural Cause/Revision calculationがmetadata meaningを解釈しないよう要求する一方、identity-bearing metadataの変更時にidentityとexact output bytesが変わることを認める。
- revision contextをseal-operation metadataおよびADR 0028 upstream Assessmentから分離する。
- persisted implementationの前に、明示的なsuccessor storage、CLI、compatibility、migration decisionを要求する。

## Review lineage

1. Revision 1はself-reviewとfirst owner reviewを完了した。
2. Operatorは`REVIEW`を選択した。独立review finding F-01はsemantic non-interpretationとoutput-byte invarianceの間の曖昧さを検出した。
3. Step 4でOperatorは`REVISE`を選択した。
4. Revision 2はStep 1へ戻り、candidate identity/review routingと受入基準5だけを変更した。Finding F-02からF-05は要件として追加されなかった。
5. Operatorはrevision 2について`REVIEW`を選択した。独立reviewはacceptanceを妨げるcontradictionまたはevidence gapなしで完了し、review前後にcandidateとreview-inputのdigestを確認した。
6. Step 4でOperatorはexact revision-2 candidateについて`ACCEPT`を選択した。

## 保持されるnon-blocking decision

Acceptanceは次を選択しない。

- final namespace ownerまたはrevision-context schema identifier
- exact canonical valueおよびresource limit
- successor repository、Seal、Provenance、Candidate、dump/load、output schema version
- mixed historical-schema readingまたはmandatory repository conversion
- exact CLI syntaxまたはinitial query scope
- Sealgraphがrevision-context profileをmaintainするか、external exampleだけをdocumentするか

これらは、accepted requirementがdesignまたはimplementationをgateすると定める範囲で、mandatory successor decisionのままである。Optionalおよびout-of-scopeのreview findingは、このrecordによってacceptance criterionにはならない。

## Authority boundary

このacceptanceは、上記で識別されたexact candidate bytesだけにrequirement authorityを確立する。それ自体では、ADRの作成またはaccept、normative product documentのrevision、persisted formatまたはexternal namespaceの選択、implementationまたはmigrationを許可しない。また、commit、push、PR、merge、release、deployment、その他のexternal writeも許可しない。各downstream artifactとeffectは、それぞれ独自のreviewおよびauthorization boundaryを保持する。
