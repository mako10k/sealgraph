# ADR 0038: Origin Trace R2 の比較 CLI と観測出力

- Status: Proposed（所有者の設計承認待ち）
- Date: 2026-09-17
- Decision Owner: Operator
- Requirement Authority: [Accepted Issue #17 R1 + R2](../process/issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)。
- Governing design proposals: [ADR 0036](0036-origin-trace-r2-presence-and-directional-observation.md)、[ADR 0037](0037-origin-trace-r2-search-and-estimation.md)。
- Revises: [Accepted ADR 0035](0035-origin-trace-cli-and-observation-output.md) の `trace compare` 文法、§2.3 の宣言適用、§3 の比較予算、§4 の `trace_compare`/範囲結果/集計/limits、§5 の比較表示、および依存する例。旧版の exact bytes と承認記録を履歴として保存する。
- Unchanged: `trace set/clear/show/source`、対応宣言の put/show/list/remove と local 保存形式、mutation receipt、移行・native transport、旧 inspection の後継版、status 表示保留。現在の実装に新 CLI があるという主張ではない。

## 1. Context

旧 ADR 0035 は `trace compare` の実行に `--max-alignment-cells` を必須とし、全最適対応の `MATCHES/DIFFERS/UNRESOLVED` と推定範囲を一つの `range_result` にまとめた。Accepted R2 の高速な完全一致確認は diff・変更後範囲推定を前提にしない。完全一致なしと推定失敗は同時に表現する必要がある。既存の v1 schema 名に異なる意味を載せない。

## 2. Decision（提案）— 操作

```sh
sealgraph trace compare (--ref REF | --seal SELECTOR) \
  --max-graph-visits N [--estimate] [--format human|json]
```

`--estimate` を省略すると、各 External run の原文残存確認と、自身・上流・下流の詳細観測だけを行う。diff、宣言選択、変更後範囲推定を起動しない。`--estimate` を指定した場合だけ、完全一致なしと判定した run の変更後範囲候補を追加計算する。推定の資源限界や処理失敗は推定未完了として表示し、完了済みの原文確認を維持する。原文確認が処理途中で終わった場合はその run を未完了と表示する。プロセスの Ctrl-C など出力 document を組み立てられない中断は、旧 ADR 0035 §3 のとおり成功 JSON を出さない。

`--max-alignment-cells` は R2 の `trace compare` では受け付けない。旧 v1 契約の引数を原文探索の予算として読み替えない。v2 に検索・推定専用の数値 budget flag は設けない。安定読取り後の検索は一件の命中または全域の確認まで行う。読取り失敗は `SOURCE_READ_FAILED`、不安定な観測は `OBSERVATION_UNSTABLE`、安定読取り後に回復可能な資源エラーで検索が終わった場合は `SEARCH_INTERRUPTED` と区別する。いずれも原文不存在を主張しない。プロセス Ctrl-C 等で結果文書を作れない場合は成功 JSON を出さない。未計測の数値を既定の中断条件にしない。`--max-graph-visits` は旧 §3 の閉じた graph 観測範囲を対象に引き続き正整数・明示必須とする。graph の予算切れは graph の完全性を下げるが、すでに完了した local run の判定を変えない。推定の回復可能な資源エラーは `estimate.state=INCOMPLETE` と理由へ写す。具体的な数値性能閾値は未決であり、原文確認の着手条件にしない。

既存の `--ref` と `--seal` の排他性、Candidate 自身と HEAD graph の区別、exact SealID による選択、成功/失敗 exit、完全な JSON document の出力、観測中の変更検出は旧 ADR 0035 §2/§3 を維持する。構造不正はエラー。安定 F を読めない場合は判定未完了であり、完全一致なしとは表示しない。

対応宣言は `--estimate` 時だけ固定した source/current identity に照合し、推定候補として提示する。相反宣言は候補を複数残して宣言側の競合を示す。宣言を `PRESENT` や `ABSENT_EXACT` の上書きに使わない。宣言の `deleted=true` は宣言者の判断として表示し、自動削除判定へ昇格しない。

## 3. Decision（提案）— machine output v2

比較の新 schema は `sealgraph/trace-compare/v2` とし、旧 v1 の比較 schema をこの操作の出力として再利用しない。`trace show`、mutation、binding、correspondence、migration/transport の schema は旧 ADR 0035 のまま。v2 の exact member order は、旧 ADR 0035 §4 の `trace_compare` 以下を基底として、次の置換を適用する。列挙しない record/member、UTF-8・integer・null・配列の規則、REF/Seal/Candidate/graph scope と経路の規則は旧 §4 を維持する。

```text
trace_compare: schema, selection, observation, candidate_own, graph, graph_reason, limits
observation: ref_heads, candidate_digest, binding_digest, declaration_digest, sources
local: baseline, trace_presence, external_run_count, compared_run_count,
       unexamined_run_count, has_difference, has_unresolved, complete, ranges
summary: seal_count, external_run_count, compared_run_count, unexamined_run_count,
         no_trace_seal_count, no_external_seal_count, has_difference,
         has_unresolved, complete, members
limits: max_graph_visits, used_graph_visits
range_result: run_index, source_snapshot_id, source_key, old_range, current_blob_id,
              presence, presence_reason, examined, selected_match_start,
              position_changed, estimate
estimate: state, method, candidates, reason
estimate_candidate: current_ranges, evidence_kind, method, reason, declaration_ids
range: start, length
```

`observation.declaration_digest` は `--estimate` を指定しなかった場合 null。`--estimate` 時は旧 §4 の ID 順宣言集合の digest を使う。推定入力を固定した証拠であり、宣言がない場合も空集合の digest を入れる。`sources` は旧 §4 の `source_observation` を使い、安定読取りを終えた exact current BlobID/byte_length、時刻、エラーを示す。

`presence` は `PRESENT|ABSENT_EXACT|UNDETERMINED` のいずれか、処理を始めていない run だけ null。`examined` は `EXAMINED|NOT_EXAMINED`。安定 F が読めずに評価を試みた run は `UNDETERMINED/EXAMINED`、着手前または検索を終えず中断した run は `UNDETERMINED/NOT_EXAMINED` とする。`presence_reason` は `EXACT_MATCH|NO_EXACT_MATCH|SOURCE_READ_FAILED|OBSERVATION_UNSTABLE|SEARCH_INTERRUPTED|NOT_EXAMINED` の一つ。`PRESENT` では `selected_match_start` が非負整数、`position_changed` は旧開始位置との数値不一致。それ以外では両者 null。`ABSENT_EXACT` に current 範囲や編集種別を自動で付けない。

`estimate.state` は `NOT_REQUESTED|NOT_APPLICABLE|CANDIDATES|NO_CANDIDATE|INCOMPLETE`。`--estimate` なしなら `NOT_REQUESTED`、原文が存在するか原文判定が未完了なら `NOT_APPLICABLE`。原文不存在時のみ候補、候補なし、未完了を使う。`estimate.method` は自動推定に用いた方式名・版で、未実行なら null。候補なし・中断でも、方式を開始したならその版を示す。`candidates` は候補なしの場合空配列、存在する場合は調査方式の結果順。各候補の `current_ranges` は半開の `{start,length}` 配列で、`evidence_kind` は `single_diff|declared_correspondence`、`method` は候補を作った方式名と版、`reason` は根拠説明。`declaration_ids` は採用した宣言 ID 集合、非宣言なら空。宣言された削除は空 `current_ranges` と `declared_correspondence` で表す。推定が中断した場合は `INCOMPLETE` と理由を示し、完了済みの presence はそのまま保持する。具体的な diff 方式版は ADR 0037 §3 に従い実装時に固定する。

`local.compared_run_count` と `summary.compared_run_count` は `PRESENT` と `ABSENT_EXACT` の件数。`has_difference` は `ABSENT_EXACT` の存在、`has_unresolved` は `UNDETERMINED` の存在を表し、両方 true になり得る。`unexamined_run_count` は `NOT_EXAMINED` の件数。Trace なし・External run なしは旧 §4 の独立した `trace_presence`/件数を維持し、一致へ換算しない。各 `complete` は次の範囲だけを示す。推定側の `INCOMPLETE` はどの presence/graph の `complete` も下げず、推定側に表示する。上流・下流は exact Seal の local facts を集計し、推定の件数で差異・判定不能を変えない。

| field | true の条件 | false になる例 |
|---|---|---|
| `local.complete` | 対象 Seal の全 External run について、命中・全域不在・読取り失敗等の評価結果を得た | 未着手 run または検索途中の run がある |
| `graph.scope.complete` | 予定した exact graph closure と方向別到達性を全て調べた | `--max-graph-visits` に到達した |
| `summary.complete` | graph scope が完全で、当該方向の全 member の `local.complete` が true | graph 未完了、またはその方向の member に未完了 run がある |

読取り不能の run は `UNDETERMINED/EXAMINED` だが、失敗理由まで評価し終えたのでそのことだけでは `local.complete` を下げない。検索中断は `UNDETERMINED/NOT_EXAMINED` とし、その local と含む summary を不完全にする。`candidate_own` の local は Candidate 自身の run に関する完全性であり、HEAD graph の完全性とは独立。`graph.own` は center の local と同じ。graph の予算停止で、既に完了した local の `complete` を false に戻さない。scope 外の Seal が存在しないとは報告しない。

旧 `current_ranges/surviving_spans/edit_blocks`、`MATCHES/DIFFERS/UNRESOLVED`、`inferred_alignment`、`max_alignment_cells/used_alignment_cells` は新 range_result/limits に入れない。後継 v2 に旧 field を空欄として残して意味を偽装しない。

## 4. 人間向け表示

最初に選択した Candidate または exact Seal、観測版、graph scope と完全性を示す。自身・上流・下流ごとに「原文残存」「完全一致なし」「判定未完了」「未調査」「Trace なし」「External run なし」を区別し、該当する SealID/run/理由/経路へ辿れるようにする。原文残存には選択された一致開始位置を示し、「その位置が元の箇所とは限らない」と説明する。完全一致なしには削除等を断定せず、`--estimate` 時だけ推定候補と根拠、複数候補・候補なし・推定未完了を別表示する。

`status` への Trace 表示追加は、所有者の既存判断どおり保留する。詳細側の試用結果に必要性があれば戻る。既存 `stale`、source compare、Cause の表示意味は変えない。

## 5. Alternatives, evidence and follow-up

| 選択肢 | 利点 | 不利益 |
|---|---|---|
 `trace compare` の `--estimate`（本提案） | 旧操作のまま軽い確認と任意の詳細推定を分けられる | 一つの出力 schema に推定状態も持つ |
 `trace estimate` を別コマンドにする | 実行費用を操作名で明確に分離できる | 対象・観測版・graph 文法が重複し得る |
 旧 v1 と全最適 budget を維持 | 旧設計と同じ形 | R2 の判定意味と異なり、利用者が軽い確認を行いにくい |

| Claim | Evidence | Action |
|---|---|---|
 C1: 推定なしの詳細比較を選べる | Accepted R2 速度方針、R7～R9 | CLI focused test と実測で確認 |
 C2: 不存在と推定失敗を同時に示せる | Accepted R2 AC14 | JSON/human で両方が保持される例を確認 |
 C3: 方向別・STALE・status を維持 | 維持された R1 R10～R19、旧 ADR 0032/0035 の境界、所有者の保留判断 | graph/detail trial で確認 |

推定方式の実装名・tie-break は ADR 0037 §3 に従い実装時に固定し、方式版として出力する。数値性能条件と必要な追加の公開資源制限は実測後に判断する。公開 CLI/schema としての本提案は所有者の採否が必要。Owner disposition: 未承認。
