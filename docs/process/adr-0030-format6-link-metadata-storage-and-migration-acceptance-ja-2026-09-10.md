# ADR 0030 Format-6 Link metadata storageとmigration acceptance（日本語）— 2026-09-10

この文書は英語のacceptance receipt全体に対する日本語のreview supportです。
英語記録が正本であり、この翻訳は第2の権威あるdecision artifactではありません。

Status: ADR 0030のowner-acceptance receipt。このreceiptはexact storage/migration
decisionを記録しますが、normative conversion、implementation、migration、
repository synchronization、release、deploymentを承認しません。

## Acceptしたtechnical snapshot

- Decision owner: Operator
- Owner decision: `ACCEPT`
- Decision date: 2026-09-10
- ADR path: `docs/adr/0030-format6-link-metadata-storage-and-migration.md`
- exact reviewed technical candidate SHA-256:
  `b879af866be94106c8a7675a1477e9ccfabfdc823058c4b3bb7157aa757f198e`
- independent-review path:
  `docs/process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-2026-09-10.md`
- independent-review SHA-256:
  `720cfae1198455366c49e897568f931a4da05b153afa54c0f70c8a1dfdd03ed8`
- independent-review verdict: `PASS`、P0からP3のfindingなし

accept後に加えたStatus、acceptance link、完了済みreview wording、follow-up wordingに
より、ADR file digestは当然変わります。これらはlifecycle transitionを記録し、
上記で識別したaccepted technical decision bytesを変更しません。

## Owner decision

2026-09-10、complete candidateと完全な日本語review supportが提示され、independent
reviewが`PASS`を返した後、Operatorは明示的に`ACCEPT`を選び、続行を指示しました。

accepted decisionは次を定めます。

- repository format 6、Seal v6、Provenance v2、Candidate v6を選択する
- Material v1、Blob identity、REF manifest、graph meaningを維持する
- exact historical Seal v5、Provenance v1、Candidate v5、message、ID、REF、tagを
  保存する
- format 6内でexact read-only v5 compatibilityを許し、v6 successor objectだけを
  writeする
- canonical metadata bytesとresource limitを確定する
- explicit config-only format-5-to-format-6 migrationを選択する
- metadata-bearing assessment-free change identityとして
  `sealgraph/upstream-change/v2`を選択する
- Assessment-reference storageとCLI/output contractを別々にgateする

## 維持される未決事項

acceptanceはまだ次を選択しません。

- exact migration CLI spelling
- metadata mutation grammarとinput source
- human outputとversioned machine inspection schema
- comparisonと`linklog` successor schema
- Assessment-reference persistence
- implementationまたはnormative-conversion sequencing

## Authority boundary

このacceptanceはADR lifecycle transitionと、指示されたCLI/output companion drafting
への続行を承認します。まだ書かれていないcompanionのaccept、normative product
document変更、implementation、migration、commit、push、PR、merge、release、
deployment、その他のexternal writeは承認しません。
