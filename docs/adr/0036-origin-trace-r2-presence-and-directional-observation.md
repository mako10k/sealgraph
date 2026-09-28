# ADR 0036: Origin Trace R2 の原文残存と方向別観測

- Status: Proposed（所有者の設計承認待ち）
- Date: 2026-09-17
- Decision Owner: Operator
- Requirement Authority: [Accepted Issue #17 R1](../process/issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) と [Accepted R2 差分](../process/issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)。有効要件は R1 の exact 本文に R2 §2～§3 を適用したもの。
- Revises: [Accepted ADR 0032](0032-content-origin-trace-and-directional-observation.md) の §5、§6 の比較結果、§7 の比較結果への集計、§8 の対応曖昧性、§9 の比較操作、およびこれらに依存する §11～§12。旧版の exact bytes と承認記録は履歴として保存する。
- Companion: 比較方法は ADR 0037、公開 CLI と schema は ADR 0038 で提案する。本 ADR 単独で実装・移行・公開を承認しない。

## 1. Context

ADR 0032 は全最適 byte alignment が合意する対応を使って `MATCHES/DIFFERS/UNRESOLVED` を出す設計だった。その後 Accepted R2 は、各 External run の原文 bytes が現在ファイルの**どこか**に連続して残るか、という判定へ R1 R7～R9 を改訂した。変更後の範囲推定は別の仕事になった。対応の唯一性や実編集履歴の証明は要件ではない。

この改訂で ADR 0032 の immutable OriginMap/SourceSnapshot（§3）、一 REF・一 Seal の作成と中間 Seal（§4）、exact Cause の方向と到達性（§7）、既存 STALE と source compare の独立（§10）は維持する。ADR 0033 の保存形式を変えない。

## 2. Decision（提案）

### 2.1 各 run の判定

External run ごとに、記録済み SourceSnapshot の `[source_start, source_start+length)` から exact byte 列 P を得る。現在の `source_key` binding が指すファイルの安定して読めた全文を F とする。Trace の構造検証と記録時の copy 一致は比較前に済ませる。Untraced run や Seal 全文を F に検索しない。

| 論理結果 | 判定根拠 | 示せること |
|---|---|---|
| 原文残存 | P が F の連続部分列として一件以上ある | 一件の一致開始位置と固定した観測版 |
| 原文の完全一致なし | F の全有効開始位置を確認し、P がない | 検索が完了したこと。削除・改変の種類は不明 |
| 判定未完了 | 読取り不能、不安定な観測、検索中断など | 理由、確認できた範囲と未確認部分 |

一件見つけた時点で原文残存の判定を完了できる。複数箇所のうち最初に発見した位置は**選択された一致**であり、歴史的に同一の箇所だとは認定しない。旧位置が変更され別位置に同じ P がある場合も原文残存とする。近傍不一致やファイルサイズ差だけで完全一致なしとしない。現在観測の identity と時刻を結果に付けるが、複数ファイルの同時 snapshot や後日再現性は保証しない。

### 2.2 変更後の範囲推定

完全一致なしと確認した run では、現在の対応範囲を利用者が調べるための**推定**を別に生成できる。推定は根拠、候補範囲、複数候補または候補なしを表す。実際の由来や唯一性、意味の保存を保証しない。推定方式や起動契機は ADR 0037/0038 の対象である。

推定の失敗・未実施・複数候補は、すでに完了した「完全一致なし」を取り消さない。推定を走らせるために原文残存確認を待たせない。候補を選んでも OriginMap、SourceSnapshot、Seal、Cause、Assessment、STALE を書き換えない。

ADR 0032/0034 の明示的な対応宣言は、固定した二版に対する利用者の調査入力として残せる。ただし宣言は P が F にあるかという機械的な判定を上書きしない。宣言で推定候補を示す、または補足する際は、自動検索結果と宣言結果を別欄に出す。宣言が競合しても完全一致判定は保ち、宣言側だけ未解決とする。

### 2.3 自身・上流・下流

ADR 0032 §7 の観測集合、Candidate と HEAD の分離、Cause の向き、direct/indirect、経路選択を維持する。`L(s)` の External run ごとの内容を §2.1 に置き換え、`Own(s)`、`Upstream(s)`、`Downstream(s)` はそれぞれ観測対象の local results を独立に集計する。差異とは「完全一致なし」、判定不能とは「判定未完了」に対応する。両者と未調査・Trace なし・External run なしは別軸で保持し、推定の成否を集計状態に混ぜない。

Cause Link は今までどおり Seal **全体**を指す。上流または下流の差異は、その exact Seal の local observation に由来する。新 HEAD へ target を差し替えず、逆向き到達性から上流の異常を再帰的に生成しない。外部ファイルの変更だけでは既存 STALE 値を変更しない。意味論、内容の矛盾、因果の妥当性を監査しない。

### 2.4 観測の完全性

REF/HEAD/Candidate、binding、各 source 観測版を固定し、必要な再確認で競合を検出したら確定結果として公開しない。構造不正は比較結果へ変換せずエラーとする。安定した F を読み切れない場合は判定未完了とし、存在しないとは報告しない。比較途中で止める場合、既に得た一致は残存確認済み、未発見の run は未完了とする。

グラフ到達性が未完了なら観測集合全体の不存在を主張しない。既に調べた exact Seal/run の結果と、未調査集合を分けて示す。比較の完全性とグラフの完全性を混同しない。

## 3. 主要な選択肢と結果

| 選択肢 | 利点 | 制約 |
|---|---|---|
 本提案：完全一致を軽い判定、変更後範囲は別推定 | 見つかった時点で終了でき、R2 の目的と一致 | 重複時に元の箇所は特定できない |
 旧全最適 alignment を判定に使用 | 対応根拠を厳密に比較できる | Accepted R2 の「どこかに完全一致」と違う結果になり、重い処理が判定の前提になる |
 旧位置付近だけ検索 | 平均で早い可能性 | 相殺編集・移動を見落とし、不存在を誤って報告する |

速度は処理分離を意味する。対象 workload、処理時間・メモリの数値閾値、実測は未決である。未測定の数値を実装開始条件にしない。原文全文の保存量と安定読取りの費用も未測定。

## 4. Claim / Evidence / Action

| Claim | Evidence | Action |
|---|---|---|
 C1: run の exact bytes の全域検索で残存を判定する | Accepted R2 R7/R8、AC4/5/12/13/15 | ADR 0037 の漏れない検索と独立した検証例に落とす |
 C2: 推定は判定と独立する | Accepted R2 R9、AC14 | ADR 0037/0038 で推定の起動・結果を分離する |
 C3: 方向・Cause・STALE を保つ | 維持された R1 R11～R19、ADR 0032 §7/§10 | ADR 0038 と実装計画で local 集計、旧 target、STALE 不変を確認する |

## 5. Review and follow-up

決定点は §2 の比較結果の置換と宣言を推定側に分離する扱いである。旧 ADR 0032 の全最適対応の結果名を、同じ意味のまま新判定に流用しない。公開名・wire 版・実行予算は ADR 0038 の候補で明示する。

Owner disposition: 未承認。本 ADR が採用されるまでは Accepted R2 が上位要件であり、旧 ADR の矛盾する比較条項を実装根拠にできない。
