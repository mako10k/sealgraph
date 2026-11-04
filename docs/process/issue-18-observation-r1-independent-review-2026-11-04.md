# Issue18 観測区分R1 独立要件レビュー

2026-11-04。Review status: **completed**。候補はreviewable。これは要件のACCEPT、設計・実装の許可、成果受入ではない。

## 固定した入力とレビュー境界

- 候補: `docs/process/issue-18-observation-requirements-r1-2026-11-04.md`、SHA-256 `96e4aba15517472049c5134b5720ef6c4689fafb01ca2d3477fe4fcc45aa1df4`。
- 所有者レビュー入力: `docs/process/issue-18-observation-r1-review-input-2026-11-04.md`、SHA-256 `1eeae68ce1c0ec6456aef98266fc79a804831ff9401e2fdedc62fe320c29d102`。
- 差分調査: `docs/process/issue-18-status-gap-2026-11-04.md`、SHA-256 `fbf37148ec7ded5ddb379fb6f04d2884f631eb2d9a770669332e72811040da77`。

独立したレビュアーが、候補本文を変更せず確認した。原本としてrequirements §3–4とCandidate inspection契約、CLIのstatus/stale/graphと機械出力契約、Accepted ADR0023/0027/0043/0044、`internal/repository/revision_graph.go`、`internal/cli/inspection_json.go`を照合した。Issue本文の取得記録は入力文書の出所情報として扱い、今回リモートの再取得はしていない。Issueにある探索案を採択済みの権威とは扱わない。

現フェーズのINSIDEは、区別する観測事実、その証拠と限界、既存契約を維持する選択、範囲・受入シナリオの整合性。OUTSIDEは具体的CLI綴り、列配置、短縮ラベル、内部API、アルゴリズム、追加スキーマの具体形と実装検証。後続設計が未作成であること自体は要件の不備にしない。

## 結論

**候補本文の必須修正となる確定矛盾は見つからなかった。** 同じR1をsecond-owner-reviewへ返せる。既存STALE集合・frontier・refs-only・通常seal admissionを維持しながら、観測理由と各baselineの事実を別に読むという選択は、現行規範と両立する。

これは所有者の目的にその選択が合うことの代行判断ではない。残る要件判断は「STALE membership変更や軽量status置換を採らず、追加観測で誤認を減らす範囲を採択するか」である。

## 現フェーズ内の確認

| 分類 | 確認結果と根拠 |
| --- | --- |
| INSIDE / 確定矛盾 | なし。requirements §3–4では観測集合とRevisionだけによるactive集合は異なり、STALE_DIRECTはactive non-leafとactive集合外をともに含む。R1条項1–3は集合を変えず、その根拠を区別するため整合する。 |
| INSIDE / 観測の限界 | 条項1–2は「未観測」と不存在を分け、Cause到達をactive所属に変換しない。ADR0023のobserver-scoped assertionと整合し、historical/detachedを証拠以上に確定しない。branching siblingを唯一のlatestへ変えないシナリオも適切である。 |
| INSIDE / baselineの分離 | 現行status実装はCandidateの存在をUNSEALED、draftをCandidate OR HEADとして集約する。条項4–6は既存の集約を誤りと断定せず、追加観測で存在・比較・publication expectation・各版draft・workfileを区別する。欠落をfalseに変換しないため、HEADなしのケースも説明可能である。 |
| INSIDE / 契約保護 | 条項7は既存公開契約を保護し、必要な新契約の版と利用方法を設計に要求している。ADR0027の既存スキーマ、ADR0044の加算的read-only方針と両立する。具体的な新出力経路の選択はまだ確定していない。 |
| INSIDE / 判断と観測 | 条項6・8はADR0043/0044のstale、記録された歴史版判断、現在の適用可能性、承認の区別を維持する。観測追加から自動Cause更新やSeal発行を導かない。 |
| INSIDE / 証拠不足・未知 | 初見実利用者が表示を誤認しないという効果は未検証。候補自身が未実施と明記しており、要件の整合性レビューを実利用者試験の成功へ読み替えない。性能の一般保証もない。既存membership保持が所有者の意図を満たすかは所有者判断として残る。 |

受入シナリオ案は、successorを観測する場合としない場合、branching、Candidateの同一/差分/HEADなし、版別draft、workfileと歴史版review、既存契約の維持を横断している。観測区分という今回の範囲を判断する材料として足りる。後続の試験fixtureや表示の検証結果まで存在するとは主張しない。

## 後続設計への注意と範囲外

以下は本文の必須修正ではなく、既に保護するとした契約を設計で具体化するときの注意である。

| 分類 | 引継ぎ事項 |
| --- | --- |
| OUTSIDE / 基本・詳細設計 | ADR0027ではCandidate比較のpublication baselineは`expected_ref_head`に固定され、現在のHEADとの関係は別の観測である。条項4の明示したHEAD版が現在HEADであっても、この既存比較を置き換えず、どのexact版との比較か、期待HEADと現在HEADの関係を区別する。HEAD_ADVANCED等も取り落とさない。 |
| OUTSIDE / 基本・詳細設計 | requirements §4と現行`staleFacts`ではdirect staleがある場合、従来のSTALE_TRANSITIVE分類は付かない。条項3に基づき追加の深い理由・経路を観測できても、従来のラベル所属、frontier、refs-onlyの結果をその診断情報で変更しない。 |
| OUTSIDE / 基本・詳細設計 | 既存機械出力の契約保護と新観測の提供を両立する出力経路・版を選び、従来経路をどう利用できるかを示す。公開JSONへ未版管理のfieldを追加してよいという採択ではない。 |
| OUTSIDE / 任意・将来提案 | 短縮ラベルや画面配置の具体案、速度改善は未確定。一般的な性能保証、特定の試験人数・時間目標を今回の新しい要件に追加しない。 |
| OUTSIDE / 明示対象外 | separate draft pointer、cascade、multi-pointer publication、draft-inclusive prospective graph、promotion、軽量status置換、Revision/hyperedge変更、Issue17追加fragment semanticsは未採択のまま。追加するには別の所有者判断が必要である。 |
| BOUNDARY_DISPUTE | 未解決のフェーズ帰属争点は検出しなかった。membership維持の採否は現在の要件判断、具体的出力設計は後続フェーズに属する。 |

## 所有者へ返す経路

ユーザーのREVIEW_THEN_DECIDE指示に従い、**変更していない同じR1と本報告をsecond-owner-reviewへ返す**。所有者は、この限定範囲をACCEPTする、変更点を指定してREVISEする、具体的な未解決論点を指定してREREVIEWする、の判断を行う。今回のレビュー完了から自動的にACCEPTや追加レビューへ遷移しない。

候補を変更する場合は新revisionとして自己レビュー・最初の所有者ルートへ戻す。今回、要件本文・コード・PLAN・Sealは変更せず、commit/pushも行っていない。
