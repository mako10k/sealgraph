# Issue18 観測区分R1 自己レビューと所有者レビュー入力

対象: `docs/process/issue-18-observation-requirements-r1-2026-11-04.md`
SHA-256: `96e4aba15517472049c5134b5720ef6c4689fafb01ca2d3477fe4fcc45aa1df4`
Stage: author candidate/self-review completed; first-owner-review待ち。現正本の新タスクへの採択は未実施。

## 出所と権威の処置

公開Issue: https://github.com/mako10k/sealgraph/issues/18 。取得したbody SHA-256 `ff535d4a3b8ba77844d09c7b327d7da4d39eb189434fbe031e8e218bd5af64e6`、Issue updatedAt `2026-09-18T12:11:11Z`。公開Issueは提案/問題の記録で、実装の許可や現行normative契約を変更する権威とは扱わない。

現在のsource commit b5e3712と、その後の成果受入記録を入力とする。原本はrequirements §3–4、revision_graph.go/statusRecord/inspect、Accepted ADR群。現在のSTALEはactive non-leafとhistorical/detachedを含む。candidate/draftの既存集約は現行契約の動作として保持し、新たな観測要件がAcceptedになるまではコードを変更しない。

## 自己レビュー

- INSIDE / 提案方針の所有者判断: 本候補は既存stale membership/admission/frontier/refs-onlyを保ち、追加の観測事実で理由とbaselineを分ける。Issue18が列挙する別案を自動採用していない。この方針を受益者の目的に照らして所有者が判断する。
- INSIDE / 観測の証拠: 現行staleFactsは!isActiveLeafにより複数理由を集約し、statusはCandidate presenceをUNSEALED、draftはCandidate OR HEADとして返す。R1 inspectはこのstatus recordを再利用するため、今回の残差がある。
- INSIDE / 意味と存在の区別: no observed successorと不存在、Cause到達とactive所属、Candidate存在と差分、版ごとのdraftを分けた。既存data modelを変更する要件にはしていない。
- OUTSIDE / 基本・詳細設計: CLI/列/短縮ラベル/追加スキーマ/API/アルゴリズム。具体値がまだないことを現フェーズの新ブロッカーへ持ち上げない。
- OUTSIDE / 未採択探索案: separate draft pointer、cascade、multi-pointer publication、lightweight status、model変更。現行の個別判断/Seal規則と混同しない。
- BOUNDARY_DISPUTE: 新しい表示方針自体は要件フェーズの所有者判断。短縮ラベルは設計。他の帰属争点は現時点で把握なし。

## 独立レビューに渡す問い案

1. 受益者がrevision、active所属、review理由、Candidate存在/差分、各版draft、workfileを誤認せず区別できるという目的に、候補の観測意味とfixtureが足りるか。
2. no observedと不存在、active集合外とsuccessor観測を混同していないか。
3. 現行membership/admission/stable output保護と矛盾していないか。変更必須なschema/interface選択を無断で確定していないか。
4. Issueのdraft/cascade探索案や任意の性能/短縮表示を新しい受入義務にしていないか。
5. この要件で固定する意味と、基本/詳細設計に渡す具体策の境界は妥当か。

## 最初の所有者ルート

未選択。推奨は、上記方針が意図と合うならREVIEW_THEN_DECIDE。REVISEなら新しいR2へ戻る。REVIEW_THEN_REVISEなら質問を追加し独立レビュー後にR2へ、REVIEWなら質問追加なしで独立レビュー後second-owner-reviewへ進む。候補本文は所有者の経路選択を受けるまで変更せず、レビューを自動開始しない。

レビュー後は矛盾/証拠不足未知/任意将来/範囲外とフェーズ内外を区別して報告する。独立レビューから自己承認へ遷移しない。本文変更は新revisionで最初に戻る。

## 数値見積もり（提案・低信頼度）

候補の所有者レビュー後の独立レビュー/改訂/要件受入まで、内部1〜3時間を暫定仮定とする。実装全体の見積もりではなく、所有者応答待ちは不明。既存PERTはUX R1完了で、この候補はまだ実行DAGに投影していない。正式採択時に下位タスク/資源/Pointと実績計測を計画化する。計画未算出の期間をゼロ扱いしない。
