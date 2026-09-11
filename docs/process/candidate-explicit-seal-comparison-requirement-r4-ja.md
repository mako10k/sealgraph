# Candidate明示Seal比較 要件R4 日本語対訳

Status: 非規範レビュー支援 — R4全範囲の日本語訳

Date: 2026-09-11

この文書は
`candidate-explicit-seal-comparison-requirement-r4.md`全体の日本語対訳であり、
権威ある要件本文ではない。

## 1. 目的と出典

可変なCandidateをレビューする運用者は、Candidateの公開基準だけでなく、明示的に
選んだ不変Sealと、Candidateの完全な予定Seal投影を比較できなければならない。

ownerが選んだ形式は次のとおりである。

```text
sealgraph candidate compare REF --against SELECTOR
```

R1とR2は独立レビューに失敗して改訂された。R3は指定digestで独立レビューに合格
したが未受理だった。そのStep 4判断前に、受理済みformat 6が通常の公開基準比較へ
`candidate-compare/v3`を割り当てたため、R3が明示比較へ同じIDを割り当てる前提は
古くなった。ownerは2026-09-11に適切な改訂を許可した。

R4はコマンド、比較方向、比較項目、整合観測、範囲を維持し、明示モードをv4へ移し、
format 6の権威を記録して要件ライフサイクルをStep 1から再開する。

## 2. 既存権威の扱い

変更するのはADR 0023、requirements、architecture、CLIにある「Candidate比較は公開
基準だけ」という制限のうち、新しい`--against`モードに関する部分だけである。

次を維持する。

- top-level compareは二つの明示的な不変Seal同士の比較
- 暗黙の前版推定をしない
- ADR 0023の完全な整合観測と再検証
- optionなしCandidate compareは公開基準比較
- format 5の通常JSONはADR 0027のv2
- format 6の通常JSONは受理済みADR 0031のv3
- source compareと全mutation境界
- format 6のLink metadata、世代型、比較の意味

受理後の新ADRは明示比較制限だけを置き換え、v4を割り当てる。releaseやmilestoneは
割り当てない。

## 3. 対象動作

`REF`は既存Candidate一つ、`SELECTOR`は現repository formatで許される不変Seal一つを
選ぶ。選択Sealがbefore、Candidateの完全な予定Sealがafterである。Material ID、
Provenance ID、content Blob ID、attachments、root、draft、完全なCause Linksを比較し、
format 6ではmetadataも含む。

対象を`EXPLICIT_SEAL`と表示し、入力selectorの綴りと完全なSeal viewを含める。現在の
REF headとexpected-head状態は公開競合情報として別に保ち、選択Sealで置換・再解釈しない。

全REF名から正確なmanifest bytesへのsorted map `O`を取得し、closureを検証し、結果を
bufferする。成功出力直前に`O`を再取得し、名前、存在・不在、HEAD、tag配列を含め完全一致
を要求する。

repository-wide `@SEAL_TOKEN` prefixだけは、全valid loose-object IDのsorted set `I_O`も
取得・再取得して完全一致を要求する。他selectorでは不要な`I_O` scanをしない。必要な
観測が変わった場合、権威ある比較結果を出力せず失敗する。

通常モードは変更しない。

```text
format 5: candidate compare REF -> candidate-compare/v2
format 6: candidate compare REF -> candidate-compare/v3
```

明示モードは両formatで`candidate-compare/v4`を出す。v4の項目はschema、ref、現在head、
expected-head状態、comparison_target、prospective、changesである。comparison_targetは
`EXPLICIT_SEAL`、入力selectorの正確な綴り、世代を明示する完全Seal viewを持つ。

v4は両formatで一つの固定された世代対応shapeを使う。format 5はSeal v5、Provenance
v1、Candidate v5と空metadata配列を示す。format 6は正確なv5/v6世代と完全metadataを
保つ。prospectiveとchangesは受理済みformat 6の世代対応・namespace単位比較shapeを
使う。human出力は明示対象を名付け、安全なbounded値と省略IDを維持する。

## 4. 対象外

Candidate同士、workfileと任意Seal、一般化operand、前版推定、text patch、raw content、
自動競合解決、mutation、Git検出、Git由来意味は追加しない。将来候補であり今回のblocker
ではない。

## 5. 仮定と未知

明示比較を最小の有用incrementと仮定する。format別に異なる明示schemaを作るより、一つの
世代対応v4が安全と仮定する。将来operandの需要と一般構文は未知だが受理blockerではない。

## 6. 受入条件

1. ADR・各文書・help・schemaが明示モードとformat別通常schemaを矛盾なく記述する。
2. `--against`が全許可selectorを解決し、不在・不正・曖昧時は結果を出さない。
3. 選択Sealが完全before、予定Candidateが完全afterで、公開競合情報を分離する。
4. ancestryや公開基準の意味を推定しない。
5. 明示モードだけが固定shapeのv4、通常format 5はv2、通常format 6はv3である。
6. 完全`O`と条件付き完全`I_O`再検証がADR 0023と一致する。
7. human/JSON testが履歴・current対象、Candidate divergence、両format、selector error、
   二種類の観測不一致を覆う。
8. 既存testと全repository検証が成功する。

## 7. 規範・非規範入力

正確なR4が受理された場合のみSection 1〜6が規範になる。ADR 0023、0027、0031と現行の
requirements、architecture、storage-format、CLIが先行権威である。実装・testは非規範の
実現可能性・現状証拠である。

## 8. 独立レビュー質問案

1. v2/v3権威を全て処理しschema IDを再利用していないか。
2. v4が両formatで固定shapeかつ世代・metadataを曖昧なく表すか。
3. 方向、項目、公開情報、`O`、条件付き`I_O`を維持しているか。
4. 通常モードを変えず明示比較だけを置換しているか。
5. 将来比較を受入blockerへ昇格していないか。

findingは矛盾、証拠gap/未知、optional/future、対象外に分類する。reviewは候補を編集・受理
しない。

## 9. Self-reviewとR3との差

出典、権威、範囲、仮定、未知、互換性、namespace所有、受入条件、review質問を明記した。
R3からの変更は、format 6権威の追加、通常modeのformat 5=v2・format 6=v3明記、明示modeの
競合v3から固定shape v4への移動だけである。その他のR3動作は全て維持する。
