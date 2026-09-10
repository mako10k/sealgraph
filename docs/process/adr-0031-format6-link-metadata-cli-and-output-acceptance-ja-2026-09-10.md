# ADR 0031 Format-6 Link metadata CLIとoutput acceptance（日本語）— 2026-09-10

この文書は英語のacceptance receipt全体に対する日本語のreview supportです。
英語記録が正本であり、この翻訳は第2の権威あるdecision artifactではありません。

Status: ADR 0031のowner-acceptance receipt。このreceiptはexact CLI/output decisionを
記録しますが、normative conversion、implementation、migration、repository
synchronization、release、deploymentを承認しません。

## Acceptしたtechnical snapshot

- Decision owner: Operator
- Owner decision: ACCEPT
- Decision date: 2026-09-10
- ADR path: docs/adr/0031-format6-link-metadata-cli-and-output.md
- exact reviewed technical candidate SHA-256:
  82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47
- initial independent-review path:
  docs/process/adr-0031-format6-link-metadata-cli-and-output-independent-review-2026-09-10.md
- initial review verdict: P1 finding 1件を伴うFAIL
- focused-rereview path:
  docs/process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-2026-09-10.md
- focused-rereview SHA-256:
  6dac3b056fa32388521eb4309534409ec42432650e34409c4aa9983599b8625d
- focused-rereview verdict: PASS、P0からP3のfindingなし、以前のP1はRESOLVED

accept後に加えたStatus、acceptance link、完了済みreview wording、follow-up wordingにより、
ADR file digestは当然変わります。これらはlifecycle transitionを記録し、上記で識別した
accepted technical decision bytesを変更しません。

## Owner decision

2026-09-10、revised complete candidateと完全な日本語review supportが提示され、focused
independent rereviewがPASSを返した後、Operatorは明示的にACCEPTを選びました。

accepted decisionは次を定めます。

- 一つのCandidate内のexact target/namespace一つに対する専用のlink-metadata set/remove
  commandを導入する
- metadata-only editでmessagesと指定されていない全metadataを保存する
- legacy add/link syntaxがstructural previous-revision/message arrayだけを変更するとき、
  metadataを保存する
- complete format-6 shared machine record、comparison record、successor schema versionを定義する
- linklog/v3にexact newer/previous Seal/Provenance identityとgenerationを持たせ、
  format-5から投影されたempty metadataとformat-6に保存されたempty metadataを区別する
- complete escaped human inspectionとexplicit historical-projection labelを定義する
- exact format-5-to-format-6 migration command/receiptを選択し、ambiguous committed result後の
  automatic retryを禁止する
- query language、schema registry/validation、ADR 0028 Assessment commandをscope外に保つ

## 維持されるgateとunknown

acceptanceは次を承認または完了しません。

- normative requirements、architecture、storage-format、CLI conversion
- implementationとexact code fixture
- repository migrationの実行
- ADR 0028 Assessment-reference storage/command
- metadata query/validation facility
- commit、push、PR、merge、release、deployment、publication

## Authority boundary

このacceptanceが承認するのはADR lifecycle transitionとそのdurable receiptだけです。
normative conversion、implementation、migration、synchronization、後続project taskには、
それぞれ別のuser authorityが必要です。
