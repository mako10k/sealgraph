# Cause Link拡張可能metadata要件 revision 2 レビュー支援 — 2026-09-10

## レビュー範囲

英語のrevision 2 candidateとproposed review inputが正本である。この文書は、すでに提示済みのrevision 1から変更された全範囲の日本語訳である。変更されていないPurpose、source directions、logical model、revision-context profile、compatibility要件、受入基準1-4および6-10、未解決decisionは省略した。省略部分はrevision 1から変更されていない。

## Candidateの変更全文

### 識別とroute

- title：Cause Link拡張可能metadata要件candidate revision 2 — 2026-09-10
- status：Step 1 requirement candidate、self-review完了、first owner review待ち
- candidate revision：2
- decision owner：Operator
- authority：candidateのみ。normativeでもimplementation authorityでもない
- proposed review input：`cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`
- previous candidate：`cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`
- revision basis：ownerがrevision 1の完了済み独立review後に`REVISE`を選択した。revision 2はfinding F-01だけを解消する。

### 受入基準5

Structural Cause/Revisionのedge membership、staleness、history、admissionは、metadataのnamespace、schema、valueを解釈せずに導出され、unknown namespaceによって弱められてはならない。metadataはidentity-bearingであるため、その変更はProvenanceID、SealID、assertion-source identity、comparison output、およびその他のexact output bytesを変更し得る。semantic non-interpretationはbyte-identical outputを要求しない。

### Step 1 self-reviewへの追加

Revision 2は、candidate identity/review routingと受入基準5の明確化を除きrevision 1を保持する。graph-semantic non-interpretationをidentityおよびoutput-byte changeから分離し、optional finding F-02からF-05を新要件として採用せずに、独立review finding F-01を解消する。

## Proposed review inputの変更全文

### 識別とroute

- title：Cause Link拡張可能metadata要件revision 2 review input — 2026-09-10
- status：proposed independent-review input。revision 2のfirst owner review待ち
- candidate revision：2
- candidate path：`docs/process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`
- candidate SHA-256：`83acad3c80f8dadd0f49271829e80efa6bbeb3f5d8adb23c30f928f1f7876ac2`
- first review後のowner route：`REVIEW`
- route選択：2026-09-10、現在のSealgraph要件レビュースレッド

### Revision provenance

Revision 2は、revision 1の独立review後にownerが選んだ`REVISE` routeに従う。変更するのはcandidate identity/review routingと受入基準5だけである。structural graph semanticsはmetadata meaningを解釈しないが、identity-bearing metadataはIDとexact output bytesを変更し得る。Finding F-02からF-05はsuccessor gate、optional/future item、またはout-of-scope observationのままであり、要件として追加しない。

### Review対象の受入基準5

Structural graph semanticsはnamespace、schema、valueを解釈しない。ただし、metadata変更がIDまたはexact output bytesを保持するとは主張しない。

### Review question 3

受入基準5は、semantic non-interpretationとidentity-bearing metadataによるID/output-byte changeを区別しつつ、すべてのstructural Cause/Revision calculationをnamespace meaningから独立させているか。

## Owner route記録

このrevision 2はStep 1へ戻った新snapshotであり、revision 1の`REVIEW` routeやreview statusを継承しない。ownerはrevision 2のfirst owner reviewとして`REVIEW`を選択した。追加owner questionなしで独立reviewを実行し、その後Step 4へ進む。この選択はcandidateをacceptしない。
