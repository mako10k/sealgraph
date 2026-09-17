# ADR 0032: コンテンツの範囲 Trace と方向別の差異観測

- Status: Proposed（設計候補 D1）
- Date: 2026-09-17
- Decision Owner: Operator
- 言語：日本語を本文とする。レビュー対象は本ファイル全体。
- Requirement Authority: 承認済み Issue #17 R1。本文中の R1～R19、AC1～AC11 はその節番号を指す。
- Related Claims: C1～C8、Evidence: E1～E4、Actions: A1～A5
- Supersedes: 現時点ではなし。採用時に必要となる既存契約の変更は §10。
- Execution Authority: 設計作成のみ。実装、既存データの移行、commit/push は別。

## 1. Context — 目的と上位根拠

大きな原文を分割することなく、抜粋と独自表現を組み合わせた小さな Seal を作る。
その由来を固定し、原文改訂後は各 Seal 自身の比較結果を Cause グラフ上の上流・下流から確認する。

| Evidence | 上位入力・観測 | 本設計での使用 |
|---|---|---|
| E1 | [Issue #17 R1](../process/issue-17-origin-trace-requirement-candidate-2026-09-17.md)、[承認記録](../process/issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) | 全機能の要件。候補時点の本文表記は変更せず、別の承認記録で Accepted を示す |
| E2 | [現行要件](../requirements.md) §2～3、§14.1、[ADR 0024](0024-source-adapters-and-portable-occurrence-provenance.md)、[ADR 0025](0025-universal-blob-seal-material-and-provenance.md) | whole-Seal Cause、immutable Blob、local binding と永続記録の分離 |
| E3 | [ADR 0028](0028-upstream-impact-assessment.md) | Assessment と今回の方向別観測の境界 |
| E4 | [ADR 0030](0030-format6-link-metadata-storage-and-migration.md)、[ADR 0031](0031-format6-link-metadata-cli-and-output.md)、main `c23aa34` | 現行形式の unknown-member 拒否、形式移行・CLI との接続 |

E1 の exact requirement Seal:
`0f6adb86be04ac509f6affd117dc49013b6186e4da45e53a375a3f198990a80b`。
その Cause は acceptance Seal
`def735fadec4d1ffb980646e848dc094e6c2c3f0b314625362bea04adbf02417`。
要件ファイル SHA-256:
`aa4b31ba8b5612288a32795792f6483102125ffe78a21de0353a0cc9788bc2a7`。
本設計を後で Seal 化する場合の authority Cause は E1 の requirement Seal とする。

Cause は常に exact Seal 全体を指す。Trace は Cause の粒度を変えない。
ファイル差異は STALE の入力にしない。意味論、内容の正しさ、atomic 性を自動判定しない。

## 2. Decision — 採用を提案する構成

以下は設計上の選択を含む提案であり、承認済み R1 が既に保存形式やアルゴリズムまで決定したという意味ではない。

```text
Seal ──Material──> 小さな content Blob
  └──Provenance──> Cause Links ──> exact Seal 全体
           └────> OriginMap ──> SourceSnapshot ──> 元ファイル全体の Blob

local source binding: source_key ──> 現在読むファイル
比較: SourceSnapshot / OriginMap + 現在のファイル -> LocalObservation
集計: LocalObservation + exact Cause graph -> 自身 / 上流側 / 下流側
```

元ファイルを Blob として保存する。元ファイル全体を収める大きな Seal は不要である。
複数の小さな Seal は同じ元 Blob を共有できる。ただし元ファイルの各版全体を保持するため、Seal の内容が小さくなることと総保存容量の削減は別である。

## 3. Immutable な由来のモデル

### 3.1 論理レコード

次は型と関係の設計であり、公開 JSON の schema 名・メンバー順・コマンド構文ではない。

```text
SourceSnapshot { source_key, content: BlobID }
OriginMap      { content: BlobID, runs: Run[] }
Run            = External { length, snapshot: SourceSnapshotID, source_start }
               | Untraced { length }
Provenance     = 既存項目 + origin: OriginMapID | none
```

- `SourceSnapshotID` と `OriginMapID` は canonical typed Blob の ID。物理 Blob 層には新しい種類を追加しない。
- `source_key` は利用者が明示する不透明な出所識別子。非空 UTF-8、完全一致、暗黙の正規化なし。パス、REF、Git commit を暗黙採用しない。同じ元文書の版は同じ key を使う。違う文書を同じ key で宣言したかという意味上の真偽は監査しない。
- 現在のローカル絶対パスは SourceSnapshot に入れない。場所の変更で過去の由来を変更しない。
- Material は現行 content と attachments のまま。Trace の追加・変更は ProvenanceID と SealID に反映する。content が同じでも由来が違う Seal を区別できる。
- OriginMap の content と、そこへ到達した Seal の Material.content は一致必須。同一 Provenance を違う content の Material と組み合わせた Seal は拒否する。
- `origin = none` は Trace 記録なし。全 Untraced の map は「範囲を記録したが外部対応なし」であり、none と区別する。
- root は Cause の境界を意味する。root に OriginMap があっても構わない。Trace は Cause でも revision assertion でもない。

### 3.2 バイト範囲と正規化

位置は content の生バイト上の 0 始まり半開区間とする。文字数、行数、改行変換、文字コード正規化を使わない。
run の開始位置は、それ以前の length の和から得る。次を構造検証する。

1. length は正整数、source_start は非負整数。演算はオーバーフローを検出する。数値の wire 表現・上限は §12 の形式仕様で固定する。
2. runs は content を隙間・重複なく全被覆する。空 content の map は runs が空。ゼロ長 run は拒否する。
3. External の元範囲が SourceSnapshot.content 内に収まり、content の該当範囲とバイト一致する。
4. 隣接 Untraced は一つに統合する。隣接 External は同じ snapshot で元範囲も連続する場合だけ統合する。run 順序は content 順であり、ソートしない。
5. authoring は正規化した値を作る。保存済み typed Blob の読み取りでは非正規形を黙って修復せず拒否する。

例えば 60 バイトなら、`External(20,A@v1,100), Untraced(10), External(30,B@v1,5)`。
この map を持つ content は、A の `[100,120)`、独自の 10 バイト、B の `[5,35)` である。

### 3.3 保持・転送・監査

Seal から Provenance、OriginMap、SourceSnapshot、元 Blob までを immutable object closure に含める。
fsck・logical dump/load・到達性検査はこの typed closure を検証・保持する。Cause 探索と object closure 探索は別の関係として実装する。
元 Blob が欠ける・範囲が不正・コピーが一致しない場合は履歴データの構造エラーであり、現在の `DIFFERS` に変換しない。

元ファイル全体が保存・転送対象になる。抜粋以外も持ち出されるので、authoring 前の表示で入力元、全体の byte 数、保存される版を明示する。
これは元ファイルの秘密情報を保存してよいという許可ではない。既存の secret 禁止規則を保つ。
部分だけ保持する代案との相違は §11。任意の保存期間や自動削除を追加しない。

## 4. 作成とローカル binding

既存の REF→入力ファイル binding は維持する。それに加えて、Trace 比較用の `source_key → 現在のファイル` のローカル対応を設ける。
両者は異なる入力選択であり、片方から他方を自動推測しない。ローカル対応は noncanonical、dump/load の永続的な由来に含めない。

作成操作の論理入力は、対象 REF、期待する HEAD、content、OriginMap の run 指定、各 source_key に対応する元ファイルの観測である。
元ファイルは一度読み込んだ exact bytes から snapshot とコピー検証を行い、同じ候補内で再読み取りした別の版を混ぜない。
content と map は一つの Candidate 更新として検証して保存する。公開は既存どおり一回につき一 REF・一 Seal。

既存 Candidate の content を変える場合は、map を同時に更新するか、Trace を明示的に外す。無効な map の引き継ぎや黙った除去はしない。
元ファイルの変更では Candidate、Seal、Cause、revision assertion を自動生成しない。

言い換えの場合は、抜粋の中間 Seal E に map を置き、言い換えの Seal D の Cause を E 全体に向ける。
D 自身を全 Untraced または Trace なしで記録できる。D→E の意味、引用の十分性、文章の正しさは検査しない。

## 5. 改訂後の対応を調べる方法

### 5.1 比較入力と根拠

比較入力は元 snapshot の全文、現在読めた全文、元範囲、および存在すれば明示的な対応宣言である。
出力には元 snapshot ID・元範囲・現在の観測 BlobID・現在の範囲群・方式/方式版・理由を持たせる。
現在の観測 BlobID を計算しても、それだけでは Blob store に保存しない。

二つの版だけから実際の編集履歴は証明できない。以下の `MATCHES/DIFFERS` は、表示した対応方式に基づくバイト比較結果であり、実際の編集操作や意味の同一性の証明ではない。
対応の根拠は `identical_snapshot`、`inferred_alignment`、`declared_correspondence` を区別する。

### 5.2 自動比較の提案

1. 同じ source_key の元全文と現在全文が同じ BlobID なら、同位置で比較する。
2. 異なる場合は、全文のバイト列に対する最小挿入・削除アラインメントを用いる。保持は同一バイトのみ、挿入・削除の各コストは 1 とする。任意の一つの diff の tie-break を「一意な対応」と扱わない。
3. 全最適解が、対象範囲の残存バイトの対応と、その内部の編集領域について同じ対応を与える場合だけ推定を確定する。挿入・削除の実行順だけが異なる同じ編集領域は同一とする。全解の列挙は要求せず、最適経路 DAG 上の曖昧性として求める。
4. 内部の挿入・置換は対応後範囲に含める。範囲の直前・直後の挿入は範囲外とする。どちらに属するかが最適解で変わる場合は判定不能とする。
5. 対応したバイト列が元範囲と同一なら `MATCHES`。位置だけ変わっていれば、その位置変化も示す。対応が一意な変更なら `DIFFERS`。
6. 全削除の推定は、元範囲の両側の境界対応が一意に定まり、その間に対応する現在範囲がない場合に限る。ファイルの先頭・末尾を境界として使える。検索失敗だけでは削除としない。
7. 削除・挿入で表現される移動、複製、分割・結合で対応候補が競合する場合は `UNRESOLVED`。別位置に同一の対象バイト列がある場合も、それだけで移動成立にせず、自動削除確定を抑制して候補を示す。
8. 元文書全体や周辺の編集によって対応が曖昧になった場合、その範囲を `UNRESOLVED` とする。他の確定できた範囲の結果は失わない。

この方式は全文との整合と解の一意性を根拠にし、対象文字列の検索一致だけでは判定しない。
ただし、最小編集というモデル自体は推定である。この推定を確定結果に使う方針は今回の設計上の選択であり、所有者の採否判断を要する（§11）。
バイト単位なのでバイナリにも定義できるが、大きなファイルの時間・メモリ基準は未設定である。
予算内に曖昧性の検査を完了できなければ、近似解を確定結果にせず `NOT_EXAMINED` と予算終了理由を返す。

### 5.3 明示的な対応宣言

自動判定できない移動等は、元 snapshot ID、元範囲、現在の観測 BlobID、現在の範囲群または削除、宣言者が指定した理由を入力できる設計とする。
これは対応についての利用者の宣言であり、意味の監査や自動的な事実認定ではない。
ID と範囲の整合を検証し、指定された現在範囲群を宣言順に連結してバイト比較する。分割・結合を表示上保持し、元 Trace は変更しない。
同じ対象に競合する宣言があれば選択が必要であり、後勝ちにしない。現在の BlobID が変われば再利用しない。
宣言はローカルの比較入力として保存可能とし、canonical provenance や Assessment にはしない。

## 6. 自身の観測結果

| レベル | 記録する内容 |
|---|---|
| 比較基準 | Candidate の exact bytes の digest または exact SealID、content/map ID |
| source 観測 | source_key、binding の観測、読めた BlobID、読み取り時刻、読み取り失敗理由 |
| 範囲結果 | `MATCHES` / `DIFFERS` / `UNRESOLVED`、元・現在の範囲、位置変化、対応根拠 |
| 調査状況 | `NO_TRACE`、`NO_EXTERNAL_RANGES`、`NOT_EXAMINED` を結果とは別に記録 |
| 集計 | 範囲数、比較済み数、未調査数、差異あり、判定不能あり、調査範囲の完全性 |

現在の binding がない、ファイルが読めない、対応が曖昧なら `UNRESOLVED`。単に比較処理を実行していない範囲は `NOT_EXAMINED`。
Untraced run は比較対象外。外部 run がゼロなら一致件数ゼロと `NO_EXTERNAL_RANGES` を示し、全体一致と呼ばない。
差異と判定不能は独立の真偽値とする。一つの代表状態で優先順位を付けて片方を消さない。

既存の source compare の「REF に binding された入力ファイル全体と Candidate/HEAD content の比較」も維持する。
その結果と本節の Trace 元ファイル比較は別項目で表示する。`WORKFILE_DIFFERS_FROM_*` を範囲差異に読み替えず、STALE にも変換しない。

## 7. 上流側・下流側の集計

### 7.1 観測範囲と Candidate

既定のグラフ範囲は、観測時の全 REF HEAD から、現行の observed graph と同じく Cause target と previous-revision assertion の参照 closure で収集した exact Seals とする。
その集合内で、**Cause target の辺だけ**を方向別集計に使う。previous-revision assertion は依存経路ではない。
object store 内の全 orphan/history を自動探索しない。観測 REF heads と範囲を出力し、範囲外の依存者が存在しないとは主張しない。
明示的に指定された historical Seal が既定集合外なら、その Seal と必要な参照 closure を追加した集合を出力する。下流の探索範囲はその集合内に限る。

REF 指定の自身は Candidate があれば Candidate を、なければ HEAD を使う。
Candidate があるときは表示を次の二つに分け、ラベルと基準を必須にする。

- Candidate 自身の Trace 比較。
- HEAD の exact Seal を中心とする自身・上流側・下流側の比較。

Candidate の Cause はグラフへ overlay しない。HEAD がまだない REF は Candidate 自身だけを表示し、上流・下流は「HEAD なし／グラフ観測対象なし」とする。
これにより Candidate の upstream を調べたい需要は満たさない。代案の overlay との比較は §11。
exact Seal 指定では、その世代自身と同じ世代を中心としたグラフ結果を返す。Cause target を同じ REF の新 HEAD や Candidate に置き換えない。

### 7.2 集計関数

`G=(V,E)` の辺 `x→y` は x の Cause が y を指すことを意味する。
`L(s)` は Seal s **自身だけ**の §6 の観測結果。

```text
Own(s)        = L(s)
Upstream(s)   = aggregate L(t) for t reachable from s by >=1 Cause edges
Downstream(s) = aggregate L(t) for t that reach s by >=1 Cause edges
```

集計単位は exact SealID とその run で重複排除する。`Upstream(t)` や `Downstream(t)` を集計入力にしない。
差異、判定不能、未調査、Trace なし、比較対象なしをそれぞれ数え、存在しない対象集合と確認済みの一致を区別する。
構造不正の Seal があればエラーを示し、その分を欠いた集計を完全な結果として出さない。

各該当 Seal について長さ 1 の経路の存在を `direct`、長さ 2 以上の経路の存在を `indirect` として独立に保持する。
両方存在する場合は両方 true。直接経路があっても間接を隠さない。
表示する経路は各区分で最短を一つ、同長なら SealID 列の辞書順最小とする。下流経路も Cause の矢印の向きで出す。
経路は全経路列挙ではないことを明記する。要約件数や到達性をこの表示本数で切り詰めない。
到達性計算自体を中断した場合は、その区分を不完全とし、未確認集合を「差異なし」にしない。

### 7.3 例

`C→B→A` で B だけ差異なら、A の下流直接、B 自身、C の上流直接に現れる。
B の上流結果を A に戻して別の差異を生成しない。
A が差異・C が判定不能なら、B の上流差異と下流判定不能が同時に立つ。
`C→A` も存在すれば、C から A の差異は直接・間接の両方で示す。

古い `B1→A1` と新しい HEAD A2 がある場合、B1 の上流比較は A1 の Trace を使う。
A1 の source と現在のファイルを比較し、A2 の Trace を借用しない。A1/A2 に関する STALE 判定は既存の別計算である。

## 8. 観測の整合性と失敗

REF・Candidate・binding は repository の既存の整合した読み取り単位で固定する。
グラフ観測中に変更を検出した場合は結果を公開せず、競合として再実行を案内する。自動的な REF 修正は行わない。
外部ファイルは repository lock で固定できないので、各ファイルを読み取り、同一実行内ではその exact bytes を共有する。
ファイル読み取り中の変更が検出できれば該当 source を判定不能とする。

複数ファイルの観測は同時刻の snapshot を保証しない。観測ごとの時刻・BlobID を返し、途中で読んだ版を一つの論理的な同時点と呼ばない。
検出不能な外部書き込みまで防いだとは主張しない。後から再現するには現在の観測 bytes の別途保存が必要であり、compare が自動保存する仕様にはしない。

過去の Trace の構造不正、現在の I/O 失敗、対応の曖昧性、処理未完了を別理由にする。
どの場合も意味論を評価しない。構造的に有効な「赤は青と同義」という content や無意味な Cause を拒否しない。

## 9. CLI・内部構成への接続

公開する操作の責務を次のように分ける。コマンド名・flag・JSON schema の確定は §12 の companion で行う。

| 操作 | 入出力・副作用 |
|---|---|
| Trace authoring | 明示した source と runs から Candidate を作成・更新。読み込んだ source の全 bytes を保持 |
| source binding 設定 | source_key と local path の明示的な対応を変更。Seals は変更しない |
| Trace inspection | 固定された由来、run、snapshot を表示。現在との比較を実施したとは扱わない |
| Trace comparison | 選択した自身と exact graph の方向別観測を返す。Canonical write なし |
| 対応宣言 | exact な二つの版に対する比較入力をローカルに記録。Cause/Assessment/Seal の生成なし |

既存 `status`/`show`/`source compare` の JSON schema へ未定義項目を黙って追加しない。
機械出力は新 schema、既存 schema が選択された場合の挙動を companion で明記する。
人間向け出力は自身・上流・下流、基準世代、差異・判定不能・未調査、範囲を先に示し、経路と byte 詳細を展開可能にする。

Domain は run/typed参照/範囲検証、canonical は正規バイト表現、store は従来の Blob I/O に専念する。
repository が入力観測・binding・Candidate・closure を調停し、graph が exact Cause の到達性と経路を処理する。
比較器は immutable bytes を受け取り、ファイル・Git・REF を読まない。CLI は入力解決と表示を担当する。
Git の自動検出、意味解析、Assessment の生成を比較器へ持ち込まない。

## 10. 既存契約・履歴との関係

| 既存契約 | 提案する変更・維持 |
|---|---|
| ADR 0025 Material/Provenance | Material と Blob identity は維持。後継 Provenance に origin を追加し、Seal 結合時の content 一致検証を追加 |
| ADR 0024 occurrence | local binding は非 canonical のまま。今回の byte Trace は専用 typed record とする。一般の occurrence 証拠を自動変換しない |
| ADR 0030/0031 format 6 | 現形式へ未知フィールドを挿入しない。後継 repository/Seal/Provenance/Candidate と新 typed schemas・出力契約が必要 |
| ADR 0028 upstream change | Assessment の対象・必須条件は拡張しない。origin を含む後継 Provenance を扱う際の change preimage と型検証の整合を companion で定める |
| STALE / Cause / revision | 入力・優先規則・グラフ意味を変更しない。Trace とその object closure を依存辺にしない |

旧形式の exact bytes と SealID は保持し、旧 Seal には `NO_TRACE` を返す。
新機能対応 reader は後継形式と必要な歴史形式を明示 dispatch し、旧 typed object を再符号化して置き換えない。
旧 reader が後継形式を読めるという保証は設けず、unsupported format で拒否させる。schema の形から推測しない。
移行は明示操作。過去の content に現在のファイルの由来を後付けしない。古い Candidate を書き換える契機、rollback、dump/load の版と検証は companion で固定する。
この設計だけで format 番号、実施時期、migration の実行を決めない。

## 11. Alternatives・判断点・結果

| 判断 | 推奨案 | 代案と得失 |
|---|---|---|
| 由来の配置 | Provenance に typed OriginMap 参照 | Material に置けば内容と近いが同内容の由来差で MaterialID も変わる。非 canonical sidecar だけなら Seal が由来を固定できない。Cause metadata では root 等を自然に扱えず、Cause と Trace の責務が混ざる |
| 元版の保持 | 元ファイル全文を Blob 保持 | 抜粋のみなら容量・持ち出し範囲を減らせるが、全文の版や周辺を用いた比較には別の証拠・取得契約が必要。全文案は大容量・機密範囲の負担がある |
| 現在との対応 | 全文最小編集の一意な対応を、根拠を付けた推定として使用 | 明示宣言だけなら自動推定の誤対応を避けやすいが、変更の都度人手が必要。単一 diff や substring だけの判定は R9 に不足。推奨案も実編集履歴の証明ではなく、計算負担と判定不能が残る |
| Candidate | 自身だけを別枠比較、グラフは exact HEAD/Seal | Candidate overlay は公開前の影響を調べやすいが、未公開の仮想辺と実グラフの区別が増える。推奨案は未公開 Candidate の方向別比較を提供しない |
| 下流範囲 | 観測した HEAD の参照 closure | 全 object store 探索は保存された orphan も拾うが、範囲と量が広がる。推奨案では観測集合外の依存者は未知 |

利用者は原文を刻まずに小さな Seal を扱える。由来の構造検証と方向別の調査が可能になる。
一方、全文保持・比較計算・明示宣言の管理という費用が生じる。数値の資源見積もり、対象規模、性能保証は未測定。
現時点の機能利用価値は未実現。今回得られるものは、承認済み要件を実装へ渡すためのレビュー可能な構造設計である。
機能提供は設計判断、companion 契約、実装・検証・提供後。日付・費用額は未定。

## 12. Implementation Notes・Follow-ups

本 ADR は論理モデルと比較・集計の設計範囲を扱う。下記の未決事項を現行契約で埋めたことにはしない。

| Action | 上位根拠・Claim | 必要な後続成果 |
|---|---|---|
| A1 | E1 R1～R5/R18、C1/C2 | 保存形式 companion：schema 名・版・正規 bytes・整数/サイズ制約、Candidate、closure、旧形式と明示移行、ADR 0028 preimage |
| A2 | E1 R7～R10、C3 | 比較 companion：最適経路の同値性、境界・移動競合の厳密な判定、宣言の形式/保存/競合、資源予算と中断表示 |
| A3 | E1 R11～R17、C4～C6 | CLI/output companion：具体的操作、JSON versions、Candidate/HEAD の別表示、scope・完全性・失敗コード |
| A4 | E1 §9、C7 | 対象ファイル種別/サイズ/run数/グラフ規模とプラットフォームを確定し、時間・メモリ・保存量の測定条件と合格基準を定める |
| A5 | E1 AC1～AC11、C8 | 下表の受け入れシナリオを companion の確定契約へ対応付け、実装・検証する。今回コード変更は行わない |

| Claim | 設計で保持する義務 | 根拠 | 設計箇所・検証シナリオ |
|---|---|---|---|
| C1 | 大きなファイルと小さな content の範囲対応 | R1～R4 | §3、AC1/AC2。混合 run、空、境界、誤コピー、全被覆 |
| C2 | 過去の由来の固定と中間 Seal | R5/R6/R18 | §3～4、AC3。元版変更後の履歴検証、whole-Seal Cause |
| C3 | 対応の根拠、不確実性、位置変化 | R7～R10 | §5～6、AC4/AC5。前方挿入、確定変更/削除、重複、移動、未読/未調査 |
| C4 | 独立した方向、併存、経路 | R11～R14 | §7、AC6/AC7。鎖・diamond、直接と間接の併存、集計往復なし |
| C5 | exact 世代・観測範囲 | R15/R16 | §7～8、AC8/AC9。Candidate と旧 target、集合外依存者、観測中変更 |
| C6 | STALE とファイル比較の独立 | R17 | §6/10、AC10。同じグラフでファイルだけ変え、既存 stale 結果は維持 |
| C7 | 既存形式・責務・未決事項の明示 | E2/E3/E4、R3/§9 | §9～12。旧 ID 保存、非 canonical binding、現行形式の unknown-member 拒否 |
| C8 | 構造監査に限定 | R18/R19 | §3/8、AC11。意味が矛盾していても構造有効なら許容、不正範囲/typed参照は拒否 |

## 13. Review

独立レビューの対象は、E1 との対応、保存 closure、対応判定の限界、Candidate/exact 世代、方向別集計と STALE の独立性である。
レビューは仕様の承認・意味上の正しさ・製品完成の証明ではない。
レビュー記録は [設計レビュー記録](../process/issue-17-origin-trace-design-review-2026-09-17.md) に、対象 digest と所見・処置を記録する。
Owner disposition: 未承認。§11 の選択と §12 の未決範囲を含む本設計候補が判断対象。
