# Issue18 現行状態との差分調査（2026-11-04）

対象: https://github.com/mako10k/sealgraph/issues/18 の2026-11-04取得本文とpush済みb5e3712。これは次候補の読取り照合で、Issue18の全提案を採択/実装する判断ではない。

## 結論

UX R1のinspectはworkfile/stale/Trace/historical claimを分離したが、Issue18の「Revision上の位置」「active集合への所属」「staleになった理由」「Candidate/HEADそれぞれのdraft」「Candidateの存在と差分の別」はまだ一つの入口で分離しきれていない。したがってIssue18全体が完了したとは言えない。

| Issue18の要求 | 現行根拠 | 観測した残差 |
| --- | --- | --- |
| Revisionの観測とactive所属を区別 | requirements §3、revision_graph.goのstate分類、graph JSON | graphには観測/assertionがあるが、status/inspectのstaleは理由別ではない |
| NONLEAFとhistorical/detachedの区別 | revision_graph.go staleFactsはisActiveLeaf=falseを共通のdirect対象へ集約 | direct target ID/経路はあるが、理由が同じ表示状態へ潰れる |
| Candidate存在と差分 | statusRecordは存在だけでUNSEALED、candidate compareは別API | CandidateがHEADと同一でも存在だけでUNSEALED。inspectでは比較への導線のみ |
| Candidate/HEAD draft別表示 | statusForREFはCandidate draft OR HEAD draft | 一つのdraft boolのため、どちら由来かがこの欄では不明 |
| local workfileのbaseline | SourceStatus/inspect workfile、R1ガイド | Candidate/HEAD/NONEの軸は既に分離。これを維持する |
| publication/stale/機械出力の契約 | requirements §3–4、cli §0/6、既存スキーマ | 名前だけの編集でmembershipやadmissionを変えてはならない |
| separate draft pointer/recursive draft generation | Issue18自身がexploratory/unacceptedと記載 | 今回の実装候補には含めない。保存形式/原子可視性/自動発行の別契約であり、現在の個別判断規則と混ぜない |

## 意味を確認した原本

requirements §3はactive集合をcurrent headsからRevision edgesだけで辿るとし、Cause到達だけではactiveにならない。STALE_DIRECTはactive non-leafに加えhistorical/detached targetも含む。このためHEAD移動後のdetachedと、観測されたsuccessorがある状態は同義でない。

repository/revision_graph.go:115〜134はCandidateの有無とdraftを読み、HEAD draftをORしてstaleを別途計算する。174〜190はdirect targetがactive leafでないことを同じ一覧へ集約する。cli/inspection_json.go:335〜385はCandidate存在をNO_CANDIDATE/UNSEALEDとし、単一draftを出力する。inspectは既存statusRecordを再利用するためこの集約を引き継ぐ。

これらは現行Accepted契約どおりで、コードの不具合と断定して既存出力を変更する根拠ではない。新たな要件判断で観測の見え方を改善する対象とする。

## 推奨する次の範囲（未採択）

Issue18の観測区分に限定した要件案を作る。既存stale集合・frontier・refs-only・通常seal admissionを維持し、理由とbaselineを独立に読める増分を提案する。draft pointer/再帰生成/軽量化/短縮ラベルは別の未採択案として分離する。現在の既存スキーマを黙って破壊せず、必要な公開出力の追加方法は後続設計で決める。

次の成果は「要件候補＋自己レビュー＋所有者レビュー入力」。現正本のUX R1完了条件は変更しない。候補作成は調査結果の提示であり、実装採択・Issue更新/クローズ・新しいSeal発行・定期通知を意味しない。
