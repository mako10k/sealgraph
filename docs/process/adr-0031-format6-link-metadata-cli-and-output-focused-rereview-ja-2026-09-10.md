# ADR 0031 Format-6 Link metadata CLIとoutput focused再レビュー（日本語）— 2026-09-10

この文書は英語のfocused rereview記録全体に対する日本語のreview supportです。
英語記録が正本であり、この翻訳は第2の権威あるdecision artifactではありません。

## レビュー対象の識別

- 対象: `docs/adr/0031-format6-link-metadata-cli-and-output.md`
- 段階: `Proposed; 2026-09-10に改訂`
- 期待するSHA-256:
  `82051186454a9b098b4fa2ebb704cc56ccc0d5e27885cb34f3b082c8b822ec47`
- rereview前digest: 一致
- rereview後digest: 一致
- review中のcandidate変更: なし
- reviewer role: 1名の独立したread-only focused ADR reviewer
- 判定: `PASS`

このrereviewはADR 0031を変更またはacceptせず、normative conversion、implementation、
migration、commit、push、release、deployment、その他のexternal writeを承認しません。

## Findings

focused scope内にP0、P1、P2、P3 findingはありません。

新しいdefect-producing source-cause phaseは確立されませんでした。

## 以前のP1のdisposition

`RESOLVED`。

revisionは、両方のowning endpointについて、次のcompleteなentry-level identityとschema
generationを追加しました。

- `newer_seal_id`、`newer_seal_schema`
- `newer_provenance_id`、`newer_provenance_schema`
- `previous_seal_id`、`previous_seal_schema`
- `previous_provenance_id`、`previous_provenance_schema`

ADR 0031 232–240行はownership mappingを明示しています。`before`はprevious endpoint、
`after`はnewer endpointに属し、null Link sideでもendpoint generationが残ります。

これは、addition、removal、two-sided changeにおいて、Provenance v1から投影された
`metadata: []`とProvenance v2に保存されたempty metadataを区別するのに十分です。
changed Link/namespaceごとにgeneration fieldを繰り返さず、ADR 0030 191–208行を満たします。
以前のdefect-producing phaseはCLI machine-schema designであり、改訂entry schemaはその
omissionを直接修正しています。

## Focused reviewの質問

### 1. Endpoint classification

**結果: Pass。** すべての`before`/`after`に曖昧さのないowning endpointがあり、validな
Seal/Provenance generation pairingがあります。

### 2. Orientationとnull case

**結果: Pass。** `before = previous`と`after = newer`が明示されています。addition、
removal、v5-to-v6、v6-to-v5のorientationをすべて分類できます。

### 3. Member ordering

**結果: Pass。** `linklog/v3`はADR 0027のtop-level orderを維持し、entry member orderだけを
明示的に置換します。既存の`(minimum_newer_depth, newer SealID, previous SealID)`による
entry sortingは変わりません。

### 4. Human outputとfixture

**結果: Pass。** human entryは両endpoint generationを表示し、implementation noteは
mixed-generation projection-versus-storage fixtureを要求します。

### 5. 新しい矛盾

**結果: Pass。** 新しい矛盾はありません。endpoint fieldは既にsuccessorである
`linklog/v3` contract内に収まり、graph membership、filtering、entry ordering、Cause Link
comparisonの意味を変えません。

## 未解決の質問

focused rereview scope内にはありません。

## Coverage

focused rereviewは次を対象にしました。

- revised ADR 0031のStatus、comparison/`linklog` record、schema matrix、human inspection、
  alternative、implementation fixture、review、follow-up
- accepted ADR 0030 historical typed-object compatibility 191–208行
- accepted ADR 0027 `linklog` topology/ordering 455–532行
- current `internal/repository/history.go`のownership/orientation
- current `internal/cli/inspection_json.go`のformat-5 topology

revisionがassumptionへ具体的な矛盾を導入しなかったため、以前Passしたunchanged areaは
再度開きませんでした。

## Reviewer結論

exact revised candidateの判定は`PASS`で、P0からP3のfindingはありません。以前のP1は
resolvedです。ADR 0031は、Operatorによる別の`ACCEPT`または`REVISE` decisionまで
`Proposed`のままです。
