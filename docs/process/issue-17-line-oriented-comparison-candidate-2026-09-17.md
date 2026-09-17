# Issue #17 行指向対応付け候補 — 2026-09-17

- 状態：Proposed（所有者の「行指向にしてくださいよ」という方向を反映した設計候補）
- 対象：Origin Trace の現在比較における対応付け部分
- 上位要件：Accepted Issue #17 R1（R7～R10、AC4～AC5）
- 関連契約：Accepted ADR 0033～0035、特に ADR 0034 §2～§3
- 実行権限：この候補文書の作成のみ。実装、公開 CLI/schema の確定、移行、commit/push は含まない

## 1. 位置付け

R1 は範囲 Trace、差異と判定不能の区別、exact Cause、既存 STALE、方向別観測を要求する。
対応付け方法、資源上限、公開 CLI と schema は R1 §9 の未決事項である。所有者は比較を行指向にする方向を明示したが、以下の具体化まで承認したものではない。
従って本書では、行指向という方向を前提にし、具体的な分割、投影、予算、出力の選択を Proposed とする。

本候補は ADR 0034 の自動 byte-token alignment を、line-token correspondence へ置き換える提案である。Cause Link は引き続き一つの exact Seal 全体を対象とし、Trace の範囲対応で Cause の粒度を変えない。Trace の結果は既存 STALE の入力にならず、自身・上流・下流の local observation を別々に計算してから Cause target の向きだけで集計する。

## 2. 入力と行 token

比較入力は、保存された SourceSnapshot の exact bytes、現在観測した exact bytes、元の一つの byte 範囲 `O=[a,b)` である。比較器は bytes を受け取り、ファイル、Git、REF、Cause graph を読まない。

各版を次の規則で左から右へ分割する。

1. `LF`（byte `0x0a`）を行終端とする。`LF` を含む token は、行本文とその `LF` を一つの token bytes とする。
2. 最後の byte が `LF` でない場合、残った末尾 bytes を終端なしの一つの token とする。空ファイルには token がない。
3. `CR`（`0x0d`）その他の byte はそのまま token bytes に含める。CRLF を LF に直したり、Unicode や UTF-8 を検証・正規化したりしない。
4. 各 token は `old_start, old_end` または `current_start, current_end` の byte offset を保持する。`end-start` は token の exact byte length である。
5. 分割は表示上の行番号を補助的に付けても、対応・比較・hash の入力を行番号や文字数へ置き換えない。

例：`alpha\nベータ\n末尾` は、LF を含む `[0,6)`、`[6,16)`、終端なしの `[16,22)` という三 token になる（例示の日本語 bytes は説明上の範囲であり、実際の offset は UTF-8 等の bytes を数える）。`alpha\r\nbeta\n` は `[0,7)` と `[7,12)` で、`\r` は第一 token の内容である。

## 3. line-token correspondence

line token を単位に、保持（同一 exact token bytes）、挿入、削除を許す順序付き alignment を求める。保持の cost は 0、挿入・削除の cost は 1 とする。置換は削除と挿入の組合せであり、意味上の置換や編集履歴を証明しない。

最小 cost を達成する全最適経路について、対象 token の surviving correspondence、内部の edit block、左右の境界対応、current 側 token 範囲列が一致する場合だけ対応を合意する。単一の diff、任意の tie-break、最初に得た経路だけで確定しない。実行順だけが異なり同じ token block 列になる経路は同一視する。

同じ本文の行が複数ある、行の削除と挿入が同じ cost になる、移動・複製・分割・結合の候補が競合する場合は、検索位置や代表経路を選ばず `UNRESOLVED` とする。全削除も検索失敗だけでは確定しない。明示対応宣言の identity 固定、矛盾時の unresolved、意味論を監査しない境界は ADR 0034 の契約を維持する。

## 4. 任意 byte 範囲の投影

OriginMap の run は任意の byte 範囲なので、行境界に揃っているとは仮定しない。line-token の合意結果を元の `O` へ次の保守的規則で投影する。

- 変更されていない token が対応した場合、その token 内の元 subline offset を同じ token 内の current subline offset へ写す。行頭の移動だけで bytes が同じなら `MATCHES` に位置変化を付ける。
- 挿入・削除・置換の edit block の old 側が `O` に完全に含まれる場合、その block の current 側を対応範囲に含める。連結した対応先 bytes が元範囲と異なれば `DIFFERS` とする。変更行の一部だけを含む場合には、この規則で行全体を取り込まない。
- 純粋な行挿入の old 側位置が `a` または `b` なら対象外、`a` より大きく `b` より小さければ対象内とする。境界所属が最適経路によって異なる場合は `UNRESOLVED` とする。
- edit block が `O` の左または右の境界をまたぐ場合、どの byte が対象に帰属するかを推測せず、その範囲を `UNRESOLVED` とする。
- `O` が一つの token の途中から始まり途中で終わる場合、その token が unchanged なら subline offset を保持する。token 自体が変更・削除・別 token と競合する場合は、部分を `DIFFERS` と断定せず `UNRESOLVED` とする。
- 現在範囲は複数の disjoint byte range として保持し、包絡範囲へ潰さない。結果は `MATCHES`、`DIFFERS`、`UNRESOLVED` と `NOT_EXAMINED` の調査状態を別軸で持つ。

ASCII の例：`intro\nvalue:12\nend\n` の先頭に `title\n` を挿入すると、元の `[6,15)` は current `[12,21)` に移り、`MATCHES` となる。`value:12\n` を `value:13\n` に変えた場合、その行全体の旧範囲 `[6,15)` は `DIFFERS`、その行の一部だけの旧範囲 `[12,14)` は `UNRESOLVED` とする。

## 5. 予算と未完了

予算は「理論上の全ての行格子」ではなく、実装が実際に探索した work unit を数える。候補として、alignment の各訪問 cell、同値辺の全最適経路合意を調べる追加訪問、対象範囲 projection の各検査を別々に数え、比較呼出しの明示予算へ加算する。共有した memoized cell は一度だけ数えるが、未検査の同値辺を検査済みとは扱わない。

予算超過または合意判定を完了できない停止時は、近似結果や一つの経路を採用せず `NOT_EXAMINED` と理由を返す。行数の二乗格子全体を必ず走査することも、O(ND) の計算量や全最適解検査の証明も、この候補では主張しない。既承認の明示予算必須・既定値なしと、独立した graph traversal 予算は維持する。比較予算の正確な計数単位と引数名、時間・メモリとの関係は詳細化が必要である。製品性能の数値基準は ENVIRONMENT_REVIEW に残る。

## 6. 維持する境界と公開面の保留

- exact Cause target、previous-revision assertion、既存 STALE の導出と名前は変更しない。Trace 差異を STALE や Cause の意味へ変換しない。
- directional aggregation は、各 Seal の local observation を先に確定し、Cause target の到達方向ごとに独立集計する。上流集計を下流へ再利用したり、集計結果を別 Seal の local result として扱ったりしない。
- binary を無条件に拒否する規則、UTF-8 必須、文字列入力への限定は追加しない。任意 bytes を LF byte で分割するが、UTF-8 の妥当性や意味は検査しない。
- 公開 command、flag、終了コード、machine schema、wire member 名・順序、保存 record 名、format version は本書では確定しない。ADR 0035 の新 schema 選択と予算引数の詳細へ戻す。

## 7. 未解決の設計トレードオフ

行単位は前方挿入や通常の行編集で計算量と表示の理解を抑えやすい一方、一行に巨大な変更が集中すると subline projection が `UNRESOLVED` になりやすい。byte 単位の細粒度を残す代案は arbitrary range を細かく投影できるが、反復 bytes、全最適経路、予算の負担が増える。行 token の equality を exact bytes に限定する案は正規化による誤同一視を避けるが、改行形式を変えた入力は差異として観測される。この候補はこれらを明示したまま、所有者の行指向を Proposed 詳細へ落とす。

## 8. 根拠と後続

- Claim C1（提案）：行 token は LF を含む exact byte 区間として扱う。所有者の行指向指示と R1 の範囲保持を満たすための具体化案であり、LF の選択自体を旧 ADR から導かれる要件とは扱わない。
- Claim C2：全最適経路の token-level 合意と保守的 projection が自動結果の根拠である。Evidence：ADR 0034 §2.2～§2.3。
- Claim C3：Cause、STALE、方向別 local aggregation は比較方式から独立する。Evidence：R1 R11～R17、ADR 0032 §6～§8、ADR 0034 §2.5。
- Action A1：ADR 0034 の対応方式をこの行 token 候補へ更新するか、採否を owner disposition に記録する。
- Action A2：採用後に ADR 0035 の CLI/output、予算引数、machine schema と決定的例をこの投影契約へ照合する。

## 9. 決定的な小例

次の例は line-token の対応と arbitrary byte range 投影を確認するための候補である。`MATCHES` や `DIFFERS` は意味上の同一性ではなく、exact bytes と合意した対応の結果である。

| old | current | 対象 | Proposed 結果 |
|---|---|---|---|
| `a\nb\n` | `title\na\nb\n` | 旧 `[0,4)` | `MATCHES`、current `[6,10)`、位置変化 |
| `a\nb\n` | `a\nc\n` | 旧 `[2,4)` | `DIFFERS`、変更 token は対象内 |
| `same\n` | `same\n` | 旧 `[1,3)` | `MATCHES`、subline offset を保持 |
| `x\nq\n` | `x\nq\nq\n` | 旧 `[2,4)` | 反復 token の最適対応が競合すれば `UNRESOLVED` |
| `left\nright\n` | `left\nR\n` | 旧 `[3,8)` | edit block が範囲境界をまたぐなら `UNRESOLVED` |

最初の例では先頭 `title\n` は対象外なので、対象 token の bytes がそのまま移動する。二番目は `b\n` と `c\n` が異なるため `DIFFERS` である。四番目は文字列検索で一つの `q\n` を選ばない。五番目は `right\n` の一部だけを対象にしても、行 token 全体の変更 block が対象境界を横断するため保守的に未確定とする。

## 10. 導入判断の境界

この候補で owner direction として扱うのは「比較の主単位を line token にする」ことである。LF の扱い、終端なし最終 token、subline offset、partial boundary の `UNRESOLVED`、実探索量の予算は技術的詳細の Proposed である。これらは実装着手時に採否を記録する。

特に、変更された行の一部が OriginMap の範囲に入る場合に、行全体の差異をそのまま範囲全体へ拡張する案は採らない。完全に対象内の edit block は `DIFFERS`、境界をまたぐ block は `UNRESOLVED` とし、行単位の粗さで arbitrary byte 範囲の境界を上書きしない。

反対に、未変更 token の内部 offset は捨てない。行単位へ集約した結果を byte offset へ戻せない設計では、R1 の範囲表示と整合しないためである。この二つの規則の組合せが、本候補で残す主要なトレードオフである。

ADR 0034 の byte 例 `abcXdef` → `QQabcXdef` は、byte-token なら対象全体の前方挿入を `MATCHES` とできる。しかし改行のない一行では `abcXdef` と `QQabcXdef` が別の line token になるため、この候補の line-token 方式は、行全体を対象にした場合に自動で同じ結果を保存しない。変更 token の対象投影が完全に内部なら `DIFFERS`、境界や対応が合意しない場合は `UNRESOLVED` となる。前方挿入を `MATCHES` として保つには、挿入が独立した `LF` 終端 token の前にある、または後続段階で line 内 subtoken 規則を所有者が採用する必要がある。この差は隠さず、AC4 の決定例を line 方式で再確認する際の未解決トレードオフとして記録する。

本書は設計候補であり、機能利用価値、実装適合、性能、移行完了を示さない。
