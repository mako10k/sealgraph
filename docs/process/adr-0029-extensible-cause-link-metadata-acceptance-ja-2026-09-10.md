# ADR 0029 拡張可能な Cause Link metadata acceptance（日本語）— 2026-09-10

この文書は英語のacceptance receipt全体に対する日本語のレビュー支援です。
英語記録が正本であり、この翻訳は第2の権威ある決定文書ではありません。

Status: ADR 0029のowner-acceptance receipt。このreceiptはexact semantic
decisionを記録しますが、companion decision、normative conversion、実装、
migration、repository synchronization、release、deploymentを承認しません。

## Acceptしたtechnical snapshot

- Decision owner: Operator
- Owner decision: `ACCEPT`
- Decision date: 2026-09-10
- ADR path: `docs/adr/0029-extensible-cause-link-metadata.md`
- exact reviewed technical candidate SHA-256:
  `3a4886671a44c7b34f8a09ce14da08b1f9445fd9cd97140b4cdf66e36ec8fd8c`
- independent-review path:
  `docs/process/adr-0029-extensible-cause-link-metadata-independent-review-2026-09-10.md`
- independent-review SHA-256:
  `cb92147d1f33c64d0105e0ef97ba91f4f8e490338575a7e275aa374936d6d5b4`
- independent-review verdict: `PASS`、P0からP3のfindingなし

accept後に追加されたStatus、acceptance link、完了済みreviewの記述、follow-up
の記述により、ADR file digestは当然変わります。これらはlifecycle transitionを
記録するものであり、上記で識別したaccepted technical decision bytesを変更
しません。

## Owner decision

2026-09-10、complete candidateと完全な日本語review supportが提示され、
independent reviewが`PASS`を返した後、Operatorは明示的に`ACCEPT`を選び、
続行を指示しました。

Operatorは、上記で識別したexact reviewed technical candidateを、拡張可能な
Cause Link metadataのsemanticおよびresponsibility boundaryとしてacceptします。
特に、このaccepted decisionは次を定めます。

- `target_seal`、`previous_revision_seal_of_target_seal`、`messages`の既存の
  意味を保存し、messagesだけを使うことを許す
- optionalなgeneric `namespace` / `schema` / `value` metadata collectionを
  一つ追加する
- Cause LinkをCauseLinkIDなしでProvenance内に埋め込んだままにする
- structural Causeおよびrevision behaviorをmetadataの意味から独立させつつ、
  metadataをProvenanceIDとSealIDへcommitする
- revision contextをcore-owned persisted namespaceではなく、外部generic
  profile exampleとして扱う
- ADR 0028 Assessmentとの独立した境界を保存する
- ADR 0017のformat-4、virtual namespace、SealGraphQL、query、numeric limit、
  implementation decisionを継承しない
- normative conversionまたはimplementationの前に、storage/migrationと
  CLI/outputのcompanion ADRを別々にacceptすることを要求する

## Lifecycleへの効果

- ADR 0029を`Proposed`から`Accepted`へ移す
- Proposed ADR 0017を`Rejected`へ移し、accepted ADR 0029へリンクする
- storage/migration companionを、最初の限定された次のdesign stepとする
- CLI/output companionは別途pendingのままとする

## 維持される未決事項

acceptanceは、まだ次を選択しません。

- exact repository、Seal、Provenance、Candidate、dump/load、output version
- canonical metadata bytes、integer range、resource limit
- old-object readability、mixed-schema operation、explicit migration policy
- exact Candidate mutation grammar、exact-target selection
- humanまたはversioned machine output schema
- comparisonまたは`linklog` successor behavior
- exact ADR 0028 change-preimage integration

general query language、query AST、index、schema registry、validator runtimeは、
accepted companion setの範囲外のままです。

## Authority boundary

このacceptanceは、decision lifecycle transitionと、指示されたcompanion ADR
draftingへの続行を承認します。まだ書かれていないcompanionのaccept、normative
product documentの変更、implementationやmigration、commit、push、PR、merge、
release、deployment、その他のexternal writeは承認しません。
