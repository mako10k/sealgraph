# ADR 0040: Origin Trace の一致位置一覧とページング公開契約

- Status: Proposed（所有者による本書の exact snapshot の採否待ち）
- Date: 2026-09-18
- Decision Owner: Operator
- 上位要件：[Accepted Issue #17 R1 + R2 + R3-c1](../process/issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)。特に R3 §1～§4、AC16～AC21。R3 の[非 draft 要件 Seal](../process/issue-17-origin-trace-requirement-r3-seal-2026-09-18.md)は exact 本文を固定する。
- 先行設計：[Accepted ADR 0039](../process/issue-17-adr-0039-acceptance-2026-09-18.md) の必要時導出とページング方針。[ADR 0033](0033-origin-trace-storage-and-migration.md) の全文 SourceSnapshot Blob、[ADR 0036](0036-origin-trace-r2-presence-and-directional-observation.md) の残存・方向別観測、[ADR 0037](0037-origin-trace-r2-search-and-estimation.md) の一件探索と推定の分離、[ADR 0038](0038-origin-trace-r2-cli-and-observation-output.md) の `trace compare` v2 と `selected_match_start`。
- 追加・置換範囲：一致位置一覧専用の読取 CLI と出力 schema を追加する提案。ADR 0038 の `trace compare`、`sealgraph/trace-compare/v2`、`status` 保留、ADR 0033 の canonical 保存形式を改訂しない。旧 Accepted snapshot と Seal は保持する。
- 実行状態：Issue #17 runtime、公開 CLI、形式 7、実データ移行は未実装。本書は実装・公開・移行の承認または完了ではない。

## 1. Context

R3 では、External run の P が現在ファイル F のどこかに一件存在すれば原文残存を確定する。同じ P が複数箇所にある場合、比較処理が選んだ `selected_match_start` は一件であり、位置の全集合ではない。必要な位置は、固定した SourceSnapshot 全文 S または安定観測した F から導出する。短い P の反復で全位置を永続保存すると件数が大きくなるため、一覧・提案には既定の返却上限とページングが要る。

ADR 0039 はこの方向を Accepted とした一方、一覧の操作名、どの版を要求するか、継続ページの版固定、件数上限、公開 schema を後続判断に残した。ADR 0038 の比較出力は `selected_match_start` 一件と存在判定を扱い、全位置の表現ではない。これらを一つの出力へ暗黙に同一視すると、速い存在判定と一覧取得の費用・完全性の境界が曖昧になる。

## 2. Decision candidate

### 2.1 対象と操作

一つの選択 baseline の一つの External run に対し、次の**読取専用**操作を提案する。

```sh
sealgraph trace occurrences (--ref REF | --seal SELECTOR) \
  --run-index N --view snapshot|current|both \
  [--limit N] [--cursor TOKEN] [--format human|json]
```

`--ref` と `--seal` は ADR 0035/0038 と同じ排他的選択とする。`--ref` は Candidate があればそれを対象にし、なければ当該 REF の HEAD Seal を対象にする。`--seal` は解決した exact Seal を対象にする。結果は選んだ baseline の種別と exact identity を示す。REF に Candidate も HEAD もない場合、または選択 baseline に OriginMap がない場合は、空の一致集合を装わず理由付きで失敗する。

`--run-index` は OriginMap の `runs` 配列における 0 始まりの index で、Untraced run を含めて数える。指定先が存在しないか External run でなければ理由付きで失敗する。External run の SourceSnapshot 全文を S、run の記録開始位置を p、正の長さを L、`P=S[p:p+L]` とする。OriginMap と SourceSnapshot の構造・copy 一致は一覧の前に検証する。P は Seal 全体やファイル全体ではなく、その run の exact bytes である。

`--view snapshot` は保存済み S の `Occ(S,P)`、`--view current` は当該 `source_key` の local binding から安定して読めた F の `Occ(F,P)` を列挙する。`--view both` は一回の操作で両集合を別の `view` 値で返し、全体の順序を `snapshot` の開始位置昇順、続いて `current` の開始位置昇順とする。同じ数値の開始位置も両 view で別の entry とし、由来箇所の継続を主張しない。binding がない、読めない、観測が安定しない場合は `current` を空集合と判定せず、§2.3 の未完了とする。

各 view の一致は `0 <= q <= |B|-L` と `B[q:q+L] == P` を満たす全ての q であり、重なり合う一致も含む。`|B|<L` は確認済みの空集合。開始位置 q と length L は byte 単位とする。`snapshot` view は SourceSnapshot.content Blob の immutable bytes を使い、後日の working file 変更や削除に影響されない。位置の導出結果は OriginMap、Seal、Cause、Assessment、STALE に書き戻さない。

本操作は一致位置一覧だけを返す。ADR 0038 の `trace compare` は一件命中で存在を確定する処理のままとし、本操作の全件列挙を呼び出さない。`trace compare --estimate` の変更後範囲候補と、完全一致した位置 entry は別の概念である。ADR 0038 の `selected_match_start` は同 ADR 0037 の探索順で最初に見つけた代表値で、byte offset の最小値や歴史的な移動先を意味しない。

一致位置を材料にした別の提案操作を公開する場合も、Accepted R3 §3 により既定件数上限、固定した対象版、ページングが必要になる。本 ADR はその提案方式や wire record を創設せず、既存の diff による `--estimate` を一致位置の提案へ読み替えない。提案操作を実装する前に、その結果と本一覧との関係を別の設計判断で固定する。

### 2.2 上限・ページ順・継続

`--limit` 省略時の１応答の最大 entry 数を **100** とする案を採る。これは人間・JSON 出力を有限にする既定の表示件数であり、実測性能保証や最大ファイルサイズではない。`--limit` の明示値は正の整数とし、既存の整数表現範囲を超える値は拒否する。明示値に新しい製品固有の最大値を設けるかは、対象 workload と実測が得られるまで決めない。`both` でも２集合の合計 entry 数に一つの `limit` を適用する。

結果は定めた view/開始位置順の連続した prefix を返す。返却上限に達した場合、次の一致の有無まで確認し、存在すれば `has_more=true` と `next_cursor` を返す。存在しなければ `has_more=false`、`next_cursor=null`。上限に達していない場合も残りの有無を確定してから完了ページとする。最終ページに達して初めて、取得済みの全ページを合わせた一覧が対象版の全集合となる。単一ページの `READY` は全集合の完了を意味しない。

`--cursor` はこの操作だけが発行する opaque な継続識別子とする。具体的な byte encoding は公開契約に含めない。識別子は、baseline の exact SealID または Candidate digest と OriginMapID、run index と P、view、`limit`、snapshot BlobID、current を含む場合は binding record と安定して観測した F の exact BlobID/byte 長、最後に返した `(view,start)` に結び付く。次ページでは同じ選択と引数を再評価し、現在ファイルを再読取して同じ exact bytes か確認する。入力・binding・baseline・現在版が変わった場合、`PAGE_CONTEXT_CHANGED` として成功ページを返さず、最初から取得し直すよう案内する。改変・不正な cursor は `PAGE_TOKEN_INVALID` とする。cursor だけから source の存在や内容を信用しない。

この方式では、current の旧 bytes を canonical Blob や永続 cache に追加保存しない。各ページで F の安定読取りと byte identity の検証を繰り返す費用がある。実装は１ページの位置を得るため、全位置をメモリへ列挙する必要はない。多数ページの時間・I/O、`limit=100` の使いやすさは未測定であり、実測後に必要なら別判断で調整する。

### 2.3 結果と未完了の意味

新しい機械出力 schema を `sealgraph/trace-occurrences/v1` とする案を採る。成功 JSON は既存 CLI と同じ compact UTF-8 JSON と LF 一つ。正規 ID、整数、null、配列の規則は ADR 0035 の後継 Trace 出力に従う。以下を exact member order とする。

```text
trace_occurrences: schema, selection, run, view, observation, page
selection: kind, requested, baseline
baseline: kind, seal_id, candidate_digest, origin_map_id
run: run_index, source_snapshot_id, source_key, source_start, length, pattern_sha256
observation: snapshot_blob_id, current_blob_id, current_byte_length, binding_digest
page: limit, entries, state, has_more, next_cursor, reason
entry: view, start, length
```

`selection.kind` は `ref|seal`。`baseline.kind` は `candidate|seal` とし、該当しない `seal_id` または `candidate_digest` は null。`origin_map_id` と `source_snapshot_id` は exact typed IDs。`pattern_sha256` は P の raw SHA-256 であり、native BlobID と区別する。`snapshot_blob_id` は ADR 0033 の SourceSnapshot.content。`current_blob_id` は安定して読めた F の native content BlobID、`current_byte_length` はその byte 長で、current を読まない view または読取り不能時は null。`binding_digest` はその source_key の local binding record の exact bytes に対する SHA-256、snapshot-only では null。`entries` は zero or more の `{view,start,length}`。どの entry も歴史的同一性や意味の一致を表さない。

`page.state` は `READY|INCOMPLETE`。`READY` では全 entry は定めた順序の連続 prefix で、`has_more` は boolean、`next_cursor` は `has_more=true` のときだけ non-null、`reason` は null。`INCOMPLETE` では、得られた entry は検証済みの prefix だけであり、`has_more=null`、`next_cursor=null`、`reason` は `SOURCE_READ_FAILED|OBSERVATION_UNSTABLE|SEARCH_INTERRUPTED` の一つ。entry が空でも対象版で一致が無いとは主張しない。途中結果から継続するのではなく、新しい初回要求で再観測する。プロセス中断など成功 JSON document を組み立てられない場合は ADR 0038 と同様に成功結果を出さない。構造不正と cursor 不整合は成功 `INCOMPLETE` へ置き換えずコマンドエラーとする。

人間向け表示は baseline、run、各 view の byte 版 identity、返却した位置、続きの有無、未完了理由を示す。`has_more=true` の一覧を「全位置」と呼ばない。元ファイルが変更・削除されたときも `snapshot` view は保存済み全文 S から導出できる。ADR 0033 の全文復元能力は実装・transport 検証で別途確認し、この一覧操作の成功だけを AC20 の完全な証拠としない。

## 3. 先行決定との権限関係

| 入力 | この候補が受ける意味 | この候補が決める範囲 |
|---|---|---|
| Accepted R3 §1～§4、AC16～AC21 | 一件で残存判定、必要時に全位置を導出、重複を含む、S/F を区別、表示上限・ページング、S 全文の復元 | 一覧の公開操作、版選択、１応答上限、継続と出力 |
| ADR 0033 | S 全文の immutable Blob と typed closure | 新しい source 保存形式・全位置保存を追加しない |
| ADR 0036/0037 | presence の三分岐、一件発見の優先探索、推定の分離 | 一覧の順序は位置昇順とし、presence の選択位置と区別 |
| ADR 0038 | `trace compare` と v2 schema、自身・上流・下流の詳細、status 保留 | 専用 `trace occurrences` と新 v1 schema。旧 v2 を再解釈しない |
| ADR 0039 | 全位置の必要時導出、offset は一件・ヒント、ページング | 同 ADR が保留した公開契約を具体化する |

Cause は Seal 全体の exact SealID を指し、ファイル差異だけで既存 STALE を変えない。位置一覧の結果を方向別状態へ追加伝播しない。意味論を監査しない。ADR 0039 の Accepted Seal は R2 等を Cause とした承認時の immutable snapshot であり、新しい R3 要件 Seal への Cause を持つ本 ADR 候補で後継の権限接続を明示する。旧 Seal や承認記録を書き換えない。

## 4. Alternatives and consequences

| 選択肢 | 利点 | 不利益 |
|---|---|---|
| 専用 `trace occurrences` と新 schema（本候補） | 存在判定の早期終了を保ち、一覧だけの費用とページ完全性を見分けられる | コマンド・schema が一つ増える |
| `trace compare --all-matches` を拡張 | 操作数を増やさない | R2 の軽い比較、方向別 graph、推定と一覧の応答・費用が同じ操作に集中する |
| 位置を OriginMap に全件保存 | 読取時の列挙が単純 | 反復する短い P で永続データが大きくなり、Accepted ADR 0039 と合わない |
| stateless cursor で current を毎回再読取（本候補） | current 全文の新しい永続保存が要らず、版違いを検出できる | ページごとの読取り・検索の費用がある |
| current 全文をページ間で保存 | 同じ版から続けやすい | 保存場所・寿命・機密性・cleanup を新たに定義する必要がある |

既定 100 件は出力件数を有限にする設計提案であり、ユーザーの workload や表示環境に対する最適値の証拠はない。別値を選ぶ場合は、一覧利用場面と実測値を使って本候補の数値を改訂する。ページ cursor は利用者が後日同じ current 版を復元する保証ではない。現行 F が変われば続きは失敗し、初回から再取得する。`snapshot` は immutable Blob が保持される限り後日も同じ版を使える。

## 5. Claim / Evidence / Action

| Claim | Evidence | Action |
|---|---|---|
| C1: 位置一覧を分離すれば presence の一件早期終了を維持できる | Accepted R2 R7/R9、R3 §1、ADR 0037/0038 | 比較が一覧を起動しないことを設計・実装で確認 |
| C2: `snapshot/current/both` は必要な版を区別して導出できる | Accepted R3 §2/AC16/AC21、ADR 0039 §2.1 | 重複一致、両版の同 offset、複数 source の例で検証 |
| C3: 版に束縛した cursor は異なる F のページ混入を防げる | Accepted R3 §3/AC18/AC19、ADR 0039 §2.3 | F/binding/baseline の変更時に続きが失敗することを検証 |
| C4: 既存 SourceSnapshot Blob から元版位置を再導出できる | Accepted R3 AC20、ADR 0033 §2.2/§2.5 | 元ファイル変更・削除後の S 全文復元と位置導出を別々に検証 |
| C5: 選択一致は全位置・移動先の証明ではない | Accepted R3 §2、[R3 承認記録](../process/issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)の AC16 解釈、ADR 0037 §2 | compare と一覧の出力例で意味を混同しないことを確認 |

## 6. Implementation notes, review and follow-up

純粋な byte 部分列列挙はファイル/REF/CLI を直接読まない比較器に置き、repository は OriginMap/SourceSnapshot、Candidate または Seal、source binding と安定した現在全文の読取りを担当する。CLI は選択・cursor・JSON/human 表示を担当する。byte 列を各ページで最初から読み直す方式や、一ページの探索開始位置を使う方式は、上記の順序・版・結果意味を保てば実装選択とする。新しい canonical オブジェクト、全位置フィールド、永続 cache、Git 自動検出を追加しない。

確認例は `B="aaaa",P="aa"` の重複開始位置 0/1/2 を `limit=1` で３ページに分ける場合、`both` の snapshot/current 境界、`|B|<|P|`、末尾 q=`|B|-|P|`、F/binding/Candidate/REF HEAD のページ間変更、source 読取り失敗、検索中断、元ファイル変更後の snapshot 列挙を含む。各期待は Accepted R3 の AC と本 ADR の採用後の節に結び付け、実装結果を要件の根拠にしない。性能の workload・platform・閾値は未決であり、実測を数値保証へ自動変換しない。

レビュー対象は本書全文。独立レビューでは、① R3 の存在判定と一覧が混ざらないか、② 両 view とページの順序・版束縛が AC16～AC21 を満たすか、③ cursor と未完了の結果が「全位置」を誤って主張しないか、④ ADR 0033/0036～0039 の Accepted 境界や public schema の所有を侵さないかを照合する。Status は Proposed で、独立レビューの結果が所有者の採否を代行しない。

採用される場合、下流計画 P2 の後継版に一覧・ページングと AC16～AC21 の作業を割り当て、canonical Issue #17 PERT を CLI の owner control に従って整合させる。その後、必要な実装・検証・公開の判断へ進む。本候補の作成とレビューでは ADR 0038/0039 の accepted bytes、要件 R3 の Seal、計画 P2 を変更しない。
