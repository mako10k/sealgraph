# Issue #17 — `FULL_VERIFICATION` 実行記録

- 日付: 2026-09-18（JST）
- 範囲: Accepted R1/R2/R3 の AC1〜AC21、Accepted ADR 0033・0036〜0041、Accepted [P3 計画](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) P9、[正本 PERT](issue-17-origin-trace.pert) `FULL_VERIFICATION`。P3 本文の `Proposed` は承認前の exact snapshot で、現在の採用状態は[承認記録](issue-17-origin-trace-implementation-plan-r3-acceptance-2026-09-18.md)による。
- 検証対象: ローカル branch `codex/issue-17-origin-trace-r2-wip` の format 7 実装、format 5/6 の保持、仮設 repository による migration/transport、公開 CLI、規範文書。実 repository の移行、push、release、所有者の最終受入は対象外。

## AC 対応と証拠

有効な AC1〜AC11 は [R1](issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) に [R2](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md) の AC4/AC5 置換を適用し、AC12〜AC15 を加えたもの。さらに [R3](issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md) の補足と AC16〜AC21 を適用する。下表の「確認」は要件や ADR の採用ではなく、対象 revision の実装・テスト・仮設操作についての証拠である。

| AC | 確認した結果 | 主な証拠 |
|---|---|---|
| 1 | 元ファイルを分割せず、小さな Seal の External run と全文 S を別に保持 | [format7 tests](../../internal/repository/format7_test.go) `TestFormat7ReadsFullSourceClosure`、[初期測定](issue-17-environment-review-2026-09-18.md) の 16 MiB S と 16-byte run |
| 2 | 二つの source と Untraced の三 run を区別し、Untraced に外部 snapshot を割り当てない | [Trace authoring tests](../../internal/cli/trace_test.go) `TestTraceCLIAuthorAndRecoverFullSources`、[詳細側試用](issue-17-detail-trial-2026-09-18.md) |
| 3 | 中間 Seal は Trace を保持し、後続 Seal の Cause は中間 Seal の exact 全体を指す | [Trace authoring tests](../../internal/cli/trace_test.go) `TestTraceCLIIntermediateSealAndAtomicContentUpdate` |
| 4 | 前方挿入で P が移っても `PRESENT` と代表一致位置を返す | [comparison tests](../../internal/tracecompare/compare_test.go) `TestComparePriorityAndBoundaries`、[詳細側試用](issue-17-detail-trial-2026-09-18.md) |
| 5 | 全域不在 `ABSENT_EXACT` と読取失敗・中断 `UNDETERMINED` を分離し、削除を断定しない | [comparison tests](../../internal/tracecompare/compare_test.go) `TestCompareExhaustiveSmallExistenceOracle` / cancellation、[own tests](../../internal/repository/trace_compare_own_test.go) `TestTraceCompareOwnReadFailuresRemainUndetermined` |
| 6 | 自身・上流・下流、直接・間接を独立に集計 | [direction tests](../../internal/repository/trace_direction_graph_test.go) `TestTraceDirectionGraphSeparatesDirectAndIndirectDiamondPaths`、[詳細側試用](issue-17-detail-trial-2026-09-18.md) の root/middle/leaf |
| 7 | run と Seal ごとの差異・未判定・出所・未調査を混ぜない | [local result tests](../../internal/cli/trace_compare_local_test.go) `TestBuildTraceCompareLocalPreservesPresenceAndCompleteness`、[batch tests](../../internal/repository/trace_compare_batch_test.go) |
| 8 | Cause の exact target 世代を保持し、Candidate/HEAD/履歴 Seal の基準を識別 | [prepare tests](../../internal/cli/trace_compare_prepare_test.go) `TestPrepareTraceCompareExactHistoricalSeal` と Candidate/HEAD tests、[direction tests](../../internal/repository/trace_direction_graph_test.go) |
| 9 | graph budget の打切りを scope/incomplete として示し、範囲外を一致扱いしない | [prepare tests](../../internal/cli/trace_compare_prepare_test.go) `TestPrepareTraceCompareBudgetKeepsLocalFacts` |
| 10 | ファイルだけを変えると Trace 比較は変わるが、既存 STALE/status は変わらない | [詳細側試用](issue-17-detail-trial-2026-09-18.md) の4 REFについて変更前後の `status/v3` JSON byte 一致 |
| 11 | 意味上矛盾する content と Cause を受け入れ、構造的な不正は拒否 | [format7 tests](../../internal/repository/format7_test.go) `TestFormat7DoesNotAuditContentOrCauseMeaning`、同 `TestFormat7RejectsCopiedByteMismatch` / 範囲外拒否 |
| 12 | 元箇所が変わって別の場所に P があっても `PRESENT` とし、唯一の歴史的箇所と呼ばない | [comparison tests](../../internal/tracecompare/compare_test.go) `shifted point before earlier copy` / `remaining after shifted`、[CLI output](../../internal/cli/trace_compare_local_test.go) |
| 13 | ファイル長の差がゼロでも旧位置以外にある P を発見 | [comparison tests](../../internal/tracecompare/compare_test.go) `same length with offset shift` と全小入力 oracle |
| 14 | 推定が未完了でも完了済みの `ABSENT_EXACT` は維持 | [estimate tests](../../internal/repository/trace_compare_estimate_test.go)、[local result tests](../../internal/cli/trace_compare_local_test.go) `TestBuildTraceCompareLocalKeepsAbsentWhenEstimateIncomplete` |
| 15 | 複数 External run の結果と source を個別に示し、Untraced は検索しない | [Trace authoring tests](../../internal/cli/trace_test.go)、[詳細側試用](issue-17-detail-trial-2026-09-18.md) の root 3 run |
| 16 | p は S で成立する一件。presence は F の一件で確定し、一覧は必要時に別導出 | [format7 validation](../../internal/repository/format7_test.go) copied-byte tests、[comparison tests](../../internal/tracecompare/compare_test.go)、[occurrences tests](../../internal/cli/trace_occurrences_test.go) |
| 17 | `aaaa` 中の `aa` は重複を含む 0/1/2 を導出し、全位置を保存しない | [page tests](../../internal/traceoccurrence/page_test.go) `TestPageOverlappingLimitOne`、[CLI pages](../../internal/cli/trace_occurrences_test.go) `TestTraceOccurrencesOverlappingPagesAndBothViews` |
| 18 | 多数一致を既定100件の有限ページに分け、`has_more` と cursor を返す | [CLI pages](../../internal/cli/trace_occurrences_test.go) `TestTraceOccurrencesDefaultLimitAndCurrentBindingChange`、[初期測定](issue-17-environment-review-2026-09-18.md) の1,048,576一致 |
| 19 | F、binding、選択 baseline が変わると cursor を失効させ、異版を混ぜない | [CLI pages](../../internal/cli/trace_occurrences_test.go) `TestTraceOccurrencesCursorDetectsCurrentChangeAndInvalidToken` / revalidation tests |
| 20 | 元ファイル削除後も範囲外を含む S 全文を exact 復元し、native dump/load 後も同一 bytes と closure を保持 | [native round trip tests](../../internal/repository/native_snapshot_test.go) `TestNativeSnapshotRestoresFullSourceAndOpaqueOrphan`、[詳細側試用](issue-17-detail-trial-2026-09-18.md) の A/B 全文・再 dump 一致 |
| 21 | S/F の位置と版を分け、同じ数値 offset から歴史上の同一箇所を断定しない | [CLI pages](../../internal/cli/trace_occurrences_test.go) `TestTraceOccurrencesOverlappingPagesAndBothViews`、[詳細側試用](issue-17-detail-trial-2026-09-18.md) の snapshot/current 表示 |

## 横断契約と検証結果

- [format7 tests](../../internal/repository/format7_test.go) は SourceSnapshot/OriginMap typed closure、source byte 不一致・範囲外拒否、旧 format 5/6 Seal・Candidate の厳格読取と世代の組合せを確認した。[migration tests](../../internal/repository/migrate_format7_test.go) は 5→7 / 6→7 の config のみの明示移行、保持 bytes/inventory、再実行と破損の拒否を確認した。
- [native round trip tests](../../internal/repository/native_snapshot_test.go) は全 Blob と未参照 Blob、typed closure、local binding 非輸送、欠損 closure の公開前拒否を確認した。[CLI tests](../../internal/cli/inspection_json_v4_test.go) は format 7 inspection v4 と旧形式 v2/v3、[completion tests](../../internal/cli/trace_source_test.go) は source key と `--from` の補完を確認した。
- 独立した読み取り専用レビューは AC1〜AC21 に確定的な欠陥・矛盾を見つけなかった。レビューは上記の native transport、旧形式読取、移行、help/completion を重点確認した。専用の source-export command は確認していないが、AC20 は immutable bytes からの exact 復元を要求し、repository API と native transport の readback で確認した。公開操作の追加要件には読み替えない。
- AC11/AC13 の回帰テスト追加後に `gofmt -w .`、`go vet ./...`、`go test -count=1 ./...`、`go test -race -count=1 ./internal/repository ./internal/cli`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check`、`make completion-check` を再実行し、全て成功した。clone-check は既知の2組・重複率0.09%で成功。

## 限界と残る境界

全体適合の証拠はローカル実装、合成入力、仮設 repository、独立レビューに限定される。実利用の原文種類・最大サイズ・対象 platform・許容時間/メモリは未定で、[初期測定](issue-17-environment-review-2026-09-18.md)の数値を性能保証にしない。`--estimate` は合成の大きい不在入力で長時間化したが、presence 判定とは分離されており、数値合格閾値は Accepted 要件にない。今回の結果は公開、実 repository 移行、所有者の最終受入を証明しない。
