# ADR 0035: 範囲 Trace の CLI と方向別観測出力

- Status: Proposed（C1、2026-09-17）
- Decision Owner: Operator
- 上位設計：[ADR 0032 D1](0032-content-origin-trace-and-directional-observation.md)、[承認記録](../process/adr-0032-origin-trace-acceptance-2026-09-17.md)
- 上位要件：Issue #17 R1 Seal `0f6adb86be04ac509f6affd117dc49013b6186e4da45e53a375a3f198990a80b`
- 対象：公開コマンド、ローカル比較入力、表示と機械出力。日本語の本ファイル全体がレビュー範囲。
- 関連：[ADR 0033](0033-origin-trace-storage-and-migration.md)、[ADR 0034](0034-origin-trace-correspondence.md)
- 本文の新しい名前・出力契約は提案であり、現行 CLI に存在するという説明ではない。

## 1. Context と根拠

C-CLI-1：履歴 Trace と現在ファイルの観測を別操作にする（R3/R5/R7、ADR 0032 §3/5）。
C-CLI-2：Candidate 自身と immutable graph を同一基準として表示しない（R15、ADR 0032 §7.1）。
C-CLI-3：差異・判定不能・未調査・対象なしを機械出力で区別する（R10～R14）。
C-CLI-4：既存の source compare、STALE、Assessment の意味を変えない（R17、ADR 0028/0031）。

現行 CLI は [ADR 0027](0027-format5-cli-authoring-and-inspection-schemas.md) と
[ADR 0031](0031-format6-link-metadata-cli-and-output.md) の閉じた schema を使用する。
新情報を古い schema 名の下に追加しない。公開は従来どおり `seal REF` による一回一 Seal。

## 2. Decision — コマンド

以下の書き込み操作は ADR 0033 の後継形式だけで受け付ける。既存形式で暗黙移行しない。
`REF` は現行 REF 文法、`SELECTOR` は現行の immutable Seal 選択文法を使う。

```sh
sealgraph trace set REF --recipe PATH [--content-file PATH|-] [--format human|json]
sealgraph trace clear REF [--format human|json]
sealgraph trace show (--ref REF | --seal SELECTOR) [--format human|json]
sealgraph trace compare (--ref REF | --seal SELECTOR) \
  --max-alignment-cells N --max-graph-visits N [--format human|json]

sealgraph trace source bind KEY --file PATH
sealgraph trace source rebind KEY --from OLD_PATH --file PATH
sealgraph trace source unbind KEY --from PATH
sealgraph trace source show KEY
sealgraph trace source list

sealgraph trace correspondence put --file PATH
sealgraph trace correspondence show ID
sealgraph trace correspondence list
sealgraph trace correspondence remove ID
```

全コマンドで `--format human|json` を受け付ける。省略時の端末判定は ADR 0022 と同じ。
`--ref` と `--seal` は排他的かつどちらか必須。REF を immutable selector として暗黙解釈しない。

### 2.1 Candidate の更新

`trace set` は既存 Candidate 一件だけを更新する。存在しなければ、既存 `add` で root/Cause を明示した Candidate を先に作るよう案内する。
content-file 省略時は既存 content を使用し、指定時はその exact bytes を使用する。recipe と content は一つの Candidate 更新として検証する。
Cause、root/draft、attachments、expected_ref_head は保存したまま。読み取り時の Candidate exact bytes に対する CAS が失敗すれば競合として終了する。
`trace clear` は origin を none にする。他の Candidate 項目は変更しない。既に none なら変更なしの結果を返す。

Trace がある Candidate に対する従来 `add` は、content BlobID が同じ場合に origin を保存する。
content が違う場合は拒否し、`trace set --content-file` による同時更新、または明示的な `trace clear` を案内する。
Cause/metadata/attachments だけの変更で Trace を捨てない。Candidate discard は従来どおり Candidate のみを除去する。

recipe は UTF-8 JSON、schema は `sealgraph/trace-recipe/v1`、次の exact member 集合を持つ（入力のメンバー順と空白は自由）。

```json
{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"manual-A","file":"docs/large.md"}],"runs":[{"kind":"external","length":20,"source":"a","source_start":100},{"kind":"untraced","length":10}]}
```

sources の各項目は `name,source_key,file` または `name,snapshot` のどちらか一方。後者は既存 SourceSnapshotID を指定する。
name は非空・重複禁止。未使用の sources、存在しない name、未知/重複メンバー、非整数や不正範囲を拒否する。
外部 run は `kind,length,source,source_start`、untraced run は `kind,length` のみ。
recipe の整数規則は ADR 0033 と同じ。runs の順序が content 順を表す。全被覆・コピー一致・統合規則も同 ADR に従う。
SourceSnapshot を作る全体 bytes の取得は同じ file ごとに一回の安定読み取りとし、その bytes から ID と比較対象を得る。
`source_key` と current file の binding は recipe から自動登録しない。

入力パスは現行 workfile と同じ、作業ディレクトリ基準の portable relative regular non-symlink path。
`.sealgraph` 自体への入力、祖先 symlink、`..`、絶対パスを拒否する。recipe は named file のみ、content-file だけ `-` を認める。
元全文を保存するファイル名と byte 数を、検証後・Blob 保存前に stderr に表示し、操作結果にも含める。保存対象を表示するために追加の確認待ちは入れない。呼出側は全文保持を理解した上でこの明示操作を選ぶ。
検証失敗で Candidate は変えない。保存途中の失敗で残る未参照 Blob を自動削除・再公開しない。

### 2.2 現在の source binding

`KEY` は SourceSnapshot.source_key と完全一致する非空 UTF-8。正規化や REF・パスからの推測はしない。
bind は新規作成、同じ path なら変更なし、異なる path なら拒否する。rebind/unbind は observed old path が一致する場合だけ一件更新する。
show/list はファイル内容を開かない。比較時にだけ選択された current file を読む。
同じ key の異なる snapshot も、現在の一つの binding に対してそれぞれ比較する。

ローカル保存先は `.sealgraph/local/trace-sources/<sha256(UTF8(KEY))>.json` とする。
record は `schema,source_key,path`、schema は `sealgraph/trace-source-binding/v1`。
ファイル名と source_key のハッシュ一致を検証する。値と安全なパスの検証、期待旧 bytes に対する CAS を行い、部分ファイルを正式ファイルとして読まない。
ローカル設定は Candidate/Seal/dump に含めない。別環境への load 後は未設定であり、元のローカル絶対位置を復元しない。

### 2.3 明示的な対応宣言

put は、ADR 0034 の元 snapshot・元範囲・現在 BlobID・現在範囲群/削除・理由を持つ宣言一件を検証してローカルに保存する。
ID は宣言の正規化 JSON bytes の SHA-256（Blob envelope なし）。canonical BlobID とは別の比較入力 ID である。
同じ exact 宣言の再登録は変更なし。競合する宣言を暗黙に置換しない。
宣言の canonical record は `schema,source_snapshot,source_start,length,current_blob,current_ranges,deleted,reason,declared_at` の順。schema は `sealgraph/trace-correspondence/v1`。
source_snapshot は typed ID、current_blob は観測した全文の BlobID、source_start/length は元の半開範囲を表す（length は正）。
current_ranges は宣言順の `{start,length}` 配列で、各 length は正、同じ current 上で重複を禁止する。隣接は許容して分割を保持し、順序変更・統合をしない。
`deleted=true` のときだけ current_ranges は空、false のときは一件以上。reason は非空 UTF-8。declared_at は宣言者が明示した `YYYY-MM-DDTHH:MM:SSZ` の UTC 時刻で、現在時刻から暗黙補完しない。
入力 JSON は順序・空白を許すが未知/重複メンバーを拒否し、ADR 0033 の正規 string/integer 規則で保存・ID計算する。put は対応する current bytes が利用可能な binding を読み、current_blob が一致する場合だけ範囲を検証して保存する。現在版が違う場合は入力不整合として拒否し、過去の宣言を現在版へ付け替えない。
保存先は `.sealgraph/local/trace-correspondences/<ID>.json`。保存 bytes は ID 計算に使う compact 正規 JSON（LFなし）そのものとし、読み取り時に exact bytes と filename ID を照合する。remove は exact ID 一件のみ。
一覧は ID 順。削除は過去の観測結果を書き換えず、今後の比較入力から外す。

比較は、観測した宣言集合から exact snapshot/current BlobID/元範囲の一致するものを選ぶ。
該当する宣言の current_ranges/deleted が異なれば `UNRESOLVED` と候補 ID を示し、自動選択しない。同じ対応で reason/declared_at だけ違う宣言は、同じ対応を支持する ID 集合として保持する。
適用する宣言を一つにしたい場合は不要な宣言を明示的に remove する。自動推定より一意な宣言を優先し、根拠を `declared_correspondence` と表示する。
current BlobID が違う宣言は未適用として区別する。該当版の宣言がなければ自動比較に進む。
宣言自体も観測 bytes と合わせて構造検証し、バイト一致は比較時に確認する。意味上の正しさは検証しない。

## 3. 観測単位・リソース・終了状態

REF の場合は、Candidate 自身（あれば）と HEAD の exact Seal に対する graph observation を別項目に返す。
Candidate がなく HEAD があれば HEAD 自身を基準にする。両方なければ target-not-found。
Candidate だけなら graph は null、理由は `NO_HEAD`。上流・下流を空配列で一致扱いしない。
Seal 指定では current REF 解決後の exact ID を固定し、Candidate は調べない。

G の vertex 集合・Cause edges・前世代 assertion closure・経路選択は ADR 0032 §7 をそのまま使う。
最初に REF/Candidate/binding/宣言の観測を固定し、出力直前の再確認が不一致なら JSON 本体を出さず observation conflict で終了する。
外部ファイルは一件ずつ安定読み取りした bytes と時刻を共有し、全 source の同時 snapshot とは呼ばない。

二つの budget は正整数かつ明示必須。実行条件を利用者が指定するための制限であり、性能合格基準ではない。
`max-alignment-cells` の計数・停止は ADR 0034 に従う。`max-graph-visits` は closure 構築と双方向到達性計算で vertex を処理する回数の合計。再訪も数え、上限を超える処理は開始しない。
処理順は Candidate 自身、graph center 自身、残りの exact SealID 順とし、各 Seal 内では run 順。source pair の計算を同一実行内で再利用し、一度支払った alignment-cell を二重に数えない。
graph が完成しなければ、その scope の `complete=false` を返す。判明した事実だけを下限として返し、不在を証明したと表示しない。
source が未読のまま budget 終了した場合は NOT_EXAMINED、読んで I/O 失敗なら UNRESOLVED。

| 終了状態 | exit | stdout |
|---|---:|---|
| 観測完了（差異・判定不能を含む）、または限定された観測結果 | 0 | 完全な observation document。completeness は必ず確認する |
| 入力不正、対象不存在、履歴 Trace 構造不正、保存/整合性競合 | 1 | 成功 JSON は出さない。stderr に理由と必要な明示操作 |

差異の存在をコマンド失敗と同一視しない。budget 中断時にも JSON を途中で切断しない。
未完了を許容しない呼出側は `complete` と NOT_EXAMINED を検査する。
Ctrl-C など document を組み立てられない中断は成功出力にせず、プロセスの通常の中断終了を用いる。

## 4. JSON 契約

機械出力は compact UTF-8 JSON + 一つの LF。未知メンバーを含まない固定 schema とする。
以下は exact member order。ID は full lowercase hex、nullable 項目は明示 null、配列は空でも存在する。
文字列の escaping と数値の正規形は ADR 0027/0033、文字列集合は UTF-8 byte 順、Seal 集合は full ID 順、run は開始 byte 順。

```text
trace_show: schema, selection, baselines
baseline: kind, ref, seal_id, candidate_digest, content_blob_id, origin_map_id
trace_inspection: baseline, origin_map, source_snapshots
trace_compare: schema, selection, observation, candidate_own, graph, graph_reason, limits
selection: kind, requested, resolved_seal_id
observation: ref_heads, candidate_digest, binding_digest, declaration_digest, sources
source_observation: source_key, path, observed_at, blob_id, byte_length, error
candidate_own: baseline, local
graph: scope, center, locals, own, upstream, downstream
scope: kind, ref_heads, extra_seals, observed_seals, complete, stop_reason
local: baseline, trace_presence, external_run_count, compared_run_count,
       unexamined_run_count, has_difference, has_unresolved, complete, ranges
summary: seal_count, external_run_count, compared_run_count, unexamined_run_count,
         no_trace_seal_count, no_external_seal_count, has_difference,
         has_unresolved, complete, members
member: seal_id, direct, indirect, direct_path, indirect_path
limits: max_alignment_cells, used_alignment_cells, max_graph_visits, used_graph_visits
```

schema 値は show=`sealgraph/trace-show/v1`、compare=`sealgraph/trace-compare/v1`。
`trace_show.baselines` は trace_inspection 配列。REF では Candidate、HEAD の順に存在するものを返し、Seal なら一件。
`baseline.kind` は `candidate|seal`、candidate_digest は Candidate の exact file bytes の SHA-256。seal_id と candidate_digest は該当側だけ non-null。ref は REF 指定時のみ non-null。
`origin_map_id` と origin_map は Trace なしの場合 null。source_snapshots は参照する snapshot の ID 順の `{id,record}` 配列で、record は ADR 0033 の型をそのまま出す。raw source bytes は show に埋め込まない。
`selection.kind` は `ref|seal`、requested は入力文字列、resolved_seal_id は REF の HEAD または exact selection。HEAD がなければ null。
`ref_heads` は REF 順の `{ref,seal_id}` 配列。binding_digest は source_key 順の binding record 配列、declaration_digest は ID 順の `{id,record}` 配列を compact 正規 JSON（LFなし）へ符号化した bytes の SHA-256。対象集合は実行開始時のリポジトリ内の全 Trace 設定。candidate_digest は対象 REF の Candidate bytes、なければ null。ID のみで設定の再現を保証しない。
source_observation は source_key 順。同じ path が複数 key に対応する場合も key ごとに返す。blob_id/byte_length は読めない場合 null、error は成功なら null、失敗なら理由文字列。
`observed_at` は読み取り完了時刻の RFC3339 UTC 表現。原子性の証拠にはしない。
`candidate_own` は存在時のみ object、それ以外 null。`graph` は HEAD/exact Seal がある場合の object、それ以外 null。graph_reason は graph が存在する場合 null、Candidate だけの場合 `NO_HEAD`。
`scope.kind` は `observed-head-closure` または `observed-head-closure-plus-selected-seal`。
`scope.extra_seals` は明示選択で追加した SealID 集合、observed_seals は実際に得た vertex 集合。stop_reason は完了時 null、予算停止時 `GRAPH_BUDGET`。
`locals` は観測した exact SealID 順の local 配列。own は center の local、upstream/downstream は summary。
`local.trace_presence` は `NO_TRACE|NO_EXTERNAL_RANGES|EXTERNAL_RANGES`。ranges は ADR 0034 の範囲結果列を使い、run_index 順。
`complete` は対象集合の探索と予定した比較が完了した意味。UNRESOLVED や NO_TRACE がなく全て一致したという意味ではない。
summary.members はその方向で到達した全 Seal を含み、locals の結果へ結び付ける。direct_path/indirect_path は存在時 exact SealID 配列、非存在時 null。探索不完全なら null は「未発見」であり不存在ではない。
seal_count と各件数は observed 範囲の数。未確定な総数を推計しない。has_difference/has_unresolved が false でも complete=false なら全体不存在を意味しない。

### 4.0 範囲結果の exact records

```text
range_result: run_index, source_snapshot_id, source_key, old_range, current_blob_id,
              current_ranges, surviving_spans, edit_blocks, position_changed,
              result, reason, evidence_kind, examined, declaration_ids
range: start, length
surviving_span: old_start, current_start, length
edit_block: old_range, current_range, kind
```

range は半開範囲を start/length で表す。通常 length は正、edit_block の片側だけはゼロを許す。kind は `insertion|deletion|replacement`。
surviving_spans は ADR 0034 の surviving_pairs の連続した対応を最大長へ統合した表現。old_start/current_start 順に並べる。
current_ranges は自動対応なら current 順、宣言なら宣言順。自動結果の隣接範囲は統合し、宣言の隣接範囲は統合しない。
result は `MATCHES|DIFFERS|UNRESOLVED`、NOT_EXAMINED のときだけ null。examined は `EXAMINED|NOT_EXAMINED`。
evidence_kind は `identical_snapshot|inferred_alignment|declared_correspondence`、入力失敗/未調査/競合の場合は null。確定した対応がない場合、current_ranges/surviving_spans/edit_blocks は空、position_changed は null。
確定時の position_changed は、元範囲の開始と対応先先頭が違う、複数範囲である、または surviving span の offset 差がゼロでない場合 true。全削除なら null。内容差異と独立の位置情報である。
reason は ADR 0034 §2.5 の closed enum をそのまま wire 値にする。履歴/宣言の構造不正 INPUT_INVALID は成功 range_result にせずコマンドエラー。
現在 binding がない・読めない場合は repository が `UNRESOLVED/SOURCE_READ_FAILED` を生成し、source_observation.error に具体的な理由を置く。current_blob_id は null。未調査は `result=null/reason=BUDGET_EXCEEDED`。
自動判定理由の優先順は、boundary-straddling がある場合 BOUNDARY_STRADDLES、その他の像の競合は ALIGNMENT_AMBIGUOUS、全削除を同文探索で抑制した場合 REPEATED_TARGET、確定全削除 FULL_DELETION、その他の変更 INTERNAL_EDIT、一致 ALIGNMENT_AGREES。
MOVE_OR_DUPLICATION_AMBIGUOUS は最適像の競合が移動/複製候補であると追加検査で識別できた場合の ALIGNMENT_AMBIGUOUS の詳細理由であり、結果を変えない。
declaration_ids は採用した同一対応の全宣言 ID、または競合した全宣言 ID。非宣言方式なら空。
範囲結果は External run ごとに一件。Untraced run は範囲結果を作らず inspection の map で確認できる。

### 4.1 ローカル操作の出力

mutation receipt は `schema,operation,ref,before_digest,after_digest,changed,stored_sources` の順。
schema は `sealgraph/trace-mutation/v1`、operation は `set|clear`。digest は Candidate exact bytes、stored_sources は snapshot ID 順の `{source_key,snapshot_id,content_blob_id,byte_length}` 配列。
binding 操作は `schema,operation,source_key,before,after,changed`。schema=`sealgraph/trace-source-mutation/v1`、before/after は nullable binding record。
binding show/list は `schema,bindings`、schema=`sealgraph/trace-source-list/v1`、bindings は source_key 順。
correspondence put/remove は `schema,operation,id,changed`、schema=`sealgraph/trace-correspondence-mutation/v1`。
correspondence show/list は `schema,records`、schema=`sealgraph/trace-correspondence-list/v1`、records は ID 順の `{id,record}` 配列。
成功した mutation の receipt を再実行の根拠にせず、曖昧な終了時は show で照合する。

### 4.2 既存 inspection への反映

後継形式での show/candidate-show/candidate-compare/graph/impact/log/linklog/compare/fsck は ADR 0031 の v3 から v4 へ上げる。
同 ADR の member order を保ち、seal_view と candidate_view の末尾へ nullable `origin_map_id` を追加する。
許可する Seal/Provenance/Candidate schema は ADR 0033 の歴史 reader matrix と同じ組合せに限定する。旧 Seal の origin_map_id は null であり、保存 bytes を変更しない。
compare と candidate compare の changes の末尾へ `origin` を追加し、既存 `change` 形式の before/after を nullable OriginMapID、changed を ID 不一致とする。
Cause Link の before/after、metadata、assertion source は変更しない。linklog の対象は Cause の履歴のままで、Trace だけの変更を Cause 変更として出さない。
fsck/v4 は ADR 0027/0031 の top-level record 末尾へ `origin_maps,source_snapshots`（非負整数）の typed role 件数をこの順に追加する。raw source Blob は通常の Blob として数える。
status への Trace 状態の表示追加は保留とする。まず詳細側（trace show/trace compare）で試し、必要性が見えた場合に status への表示を再検討する。それまでは status/v3 を維持する。stale/v2、source compare/v1 の既存意味も維持し、Trace の方向別観測は専用 compare が提供する。
raw-content 出力、狭い既存 mutation receipt は従来どおり。旧 repository で新 schema を出さず、後継 repository で旧 schema の出力を要求する negotiation も設けない。

## 4.3 移行と snapshot transport

```sh
sealgraph migrate repository --from 5 --to 7 [--format human|json]
sealgraph migrate repository --from 6 --to 7 [--format human|json]
sealgraph dump --format native-blobs-v1
sealgraph load --format native-blobs-v1 --file PATH --max-input-bytes N
```

migration は ADR 0033 の config-only transaction。from は実際の repository format と一致必須、既に 7 なら拒否する。
成功 JSON は `sealgraph/repository-migrate/v2`、順序は `schema,from_format,to_format,result,retained_seals_v5,retained_seals_v6,retained_candidates_v5,retained_candidates_v6`。to=7、result=`MIGRATED`。各件数は公開後の再読み取りで確定する。
公開後に結果配達だけが失敗した場合は既存同様 `MIGRATION_COMMITTED_OUTPUT_UNDELIVERED` とし、再実行せず `fsck --format json` で format7 の `sealgraph/fsck/v4` と保持 inventory を確認する。
永続 receipt object を追加せず、履歴 Seal を生成しない。既存 5→6 の command と v1 receipt は維持する。

dump は format7 のみで、ADR 0033 の `sealgraph/native-snapshot/v1` を stdout に一文書出す。無参照分も含む全 Blob が対象であることを stderr に明示する。
load は named regular non-symlink file から読み、正整数 max-input-bytes を超える入力を拒否する。既存 universal-blob-v1 format4移行の mode は変更しない。
load は `.sealgraph` が存在しない destination だけに私有 staging から atomically publish する。既存 store に merge/overwrite しない。
成功出力は compact JSON+LF、schema=`sealgraph/native-load/v1`、順序 `schema,result,snapshot_sha256,repository_format,blobs,refs,candidates`。result=`LOADED`、format=7、snapshot_sha256 は exact input document bytes の SHA-256、各件数は公開後の readback。
公開前の失敗では destination を公開しない。公開後の出力失敗では `LOAD_COMMITTED_OUTPUT_UNDELIVERED` を stderr に示し、再実行せず fsck と inventory を照合する。自動削除・再試行しない。
source binding/correspondence は transport 対象外なので、load 後の current-source 比較は必要な binding を明示設定するまで未解決となる。

## 5. 人間向け表示と例

最初に比較対象（Candidate または exact Seal）、scope、完全性を示す。
次に自身・上流・下流ごとに差異/判定不能/未調査の件数、Trace なし/対象なしの件数を並べる。
該当する exact Seal、run、理由、根拠種別と経路を省略せず辿れるようにする。表示する経路は最短の直接/間接各一本であり、全経路列挙と呼ばない。
制御文字の escaping は ADR 0022 に従い、元 content を terminal 制御として解釈しない。

`C→B→A` で B に差異があれば、A の下流直接、B 自身、C の上流直接として同じ B の local を示す。
C に Candidate があっても、graph の center は C の HEAD。Candidate 自身の差異で A の下流差異を新規生成しない。
A の file が読めない場合は、A 自身および B/C の上流に UNRESOLVED が立ち、B の差異は消えない。
元ファイルのみの変更で STALE の値が変わったという表示は生成しない。

## 6. Alternatives と Consequences

専用 trace コマンドは元来の source binding と混同しにくい反面、利用者が覚える操作が増える。
status への統合は一覧性がある一方、元ファイル群の読み取りと既存出力への影響を確認する必要がある。読み取り費用は未測定であり、不採用を確定する根拠にはしない。詳細側を試して必要性が見えた場合に戻る判断とし、恒久的な除外にはしない。
明示 budget は実行条件を隠さないが、初回指定が必要。測定済み workload がない段階で既定値を性能保証のように示さないため、この案を提案する。
既存 Candidate を更新する方式は二操作になるが、Trace 用に別の root/Cause 作成文法を重複定義しない。
対応宣言の競合を remove で解消する案は単純だが、同時に複数の異なる解釈を切り替える用途には不便である。将来の選択 UI を今回の要件にしない。

## 7. Implementation Notes・Review・Follow-ups

A-CLI-1：ADR 0033/0034 の型・状態・budget 単位と相互照合し、仕様番号を確定する。
A-CLI-2：Accepted 後、CLI/reference と実装を同じ契約へ対応付ける。現行 docs/cli.md は本提案だけで変更しない。
A-CLI-3：AC1～AC11 の操作シナリオに、CAS 競合、旧形式拒否、未知 JSON member、原文の全量保存表示、Candidate-only の graph null を追加して適合検証する。

検証済みの速度・資源消費量はない。対象規模・プラットフォーム・合格閾値は ADR 0032 A4 の判断として残る。
レビューは companion 一式の整合に限定し、新たな意味論や自動修復の仕様を導入しない。
Owner disposition: 未承認。新コマンド・machine schemas・明示 budget が本候補の判断点。

## 8. status 表示の保留判断（2026-09-17）

所有者の発言：

> では、一旦、そこは保留します。ただし、詳細側で試して、必要そうになれば戻しましょう。

対象は status への Trace 状態の表示追加。詳細側で試した結果から必要性を判断し、必要ならこの設計判断へ戻る。試行結果はまだなく、必要性は未判定。
この部分判断は ADR 0033～0035 全体の承認や、性能基準の決定を意味しない。
