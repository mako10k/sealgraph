# Issue #17 perttool 計画候補 P1 — 2026-09-17

実装計画 P1 を perttool の実行 DAG 候補にした。提示後に所有者が保存を承認し、[正式な計画ファイル](issue-17-origin-trace.pert)へ同一bytesで保存した（2026-09-17）。以下の候補提示時の内容と確認条件は履歴として保持する。
レビュー範囲は下記の全作業・依存関係・仮見積もり・実行状態、および末尾の候補全文。

## 対象と作用

- 新規保存先候補：`docs/process/issue-17-origin-trace.pert`。現時点では存在しない。
- 操作：新しい専用計画を作成。既存 `PLAN.pert` の終点・完了履歴を変更しない。
- 終点：採用する詳細契約の実装適合証拠が揃うこと。配布、実repository移行、製品の所有者受入とは別。
- 12 tasks、16 milestones、6 dependency gates、resource 1。
- 工数は規模から置いた未計測の相対ポイント三点仮説。比較器は全最適対応と曖昧性の実装不確実性が大きいため幅を広くした。担当1枠は計算用の仮定であり、実際の人員割当ではない。
- velocity と日付換算は設定していない。前提となる所有者判断の待ち時間は計算外。
- 詳細契約・対象環境の確定は blocked。他の作業は planned。開始・完了の実績を作っていない。
- status は詳細側の試用後に再評価する作業だけ。status表示の実装は追加しない。

## 作業と三点見積もり

楽観 / 最頻 / 悲観。値の採否と実作業後の見直しは別であり、日数や費用の約束ではない。

| ID | 作業 | 三点値 | 状態 |
|---|---|---|---|
| CONTRACT_REVIEW | S0 詳細契約の採用版を確定 | 1p / 2p / 4p | blocked |
| ENVIRONMENT_REVIEW | S0 対象環境・資源条件と見積もりを整える | 1p / 2p / 4p | blocked |
| FORMAT_TYPES | S1 型・codec・closure・歴史reader | 3p / 5p / 9p | planned |
| TRACE_AUTHOR | S2 Trace作成からSeal/表示まで | 3p / 5p / 9p | planned |
| RANGE_COMPARATOR | S3 全最適対応と宣言を実装 | 5p / 8p / 16p | planned |
| SELF_COMPARE | S4 binding・宣言と自身比較を接続 | 3p / 5p / 10p | planned |
| DIRECTION_COMPARE | S5 exact Causeの方向別集計・詳細表示 | 3p / 5p / 10p | planned |
| DETAIL_TRIAL | S6 詳細側の作成・変更・追跡を試用 | 1p / 2p / 4p | planned |
| STATUS_REVISIT | S6 試用結果からstatus保留を再評価 | 1p / 1p / 3p | planned |
| NATIVE_TRANSPORT | S7 native snapshotと移行異常系 | 3p / 5p / 10p | planned |
| NORMATIVE_SYNC | S8 規範文書・help・completionの整合 | 1p / 2p / 4p | planned |
| FULL_VERIFICATION | S8 全体検証・独立レビュー・証拠整理 | 2p / 3p / 6p | planned |

## perttool の検証・解析結果

- `document check`：errors 0、warnings 0。
- `dag analyze --schedule both --max-paths 8`：成功。
- precedence 下限：32.833p。critical path は S0 の二つの判断で分岐する2本。
- 共通の主経路：S0 → 型/codec → Trace作成/公開 → 自身比較 → 方向別集計 → 詳細試用 → status再評価 → 文書整合 → 全体検証。
- 作業担当1枠の resource schedule：49.333p。
- 両値とも blocked が計算開始時に解除されるという条件付き。`PTRES-303` を保持し、現在の実施可能な日程とは扱わない。
- `dag next`：推奨作業なし。先頭の `CONTRACT_REVIEW` / `ENVIRONMENT_REVIEW` が blocked。
- 独立した read-only 確認：P1 との依存関係・状態・status保留・transportの最終合流に具体的な対応エラーなし。

比較器は S0 後に型/codec と並行可能。native transport は Trace作成/公開後から別系統で進められる。最終統合は詳細試用側とtransport側の両方を待つ。
この並行可能性は DAG 上の関係であり、1担当という計算条件では直列化される。

## 正式保存に必要な確認

perttool の assertion-free batch preview は、今回の新しい終点（goal）と依存DAG（dag）を governed change と判定した。
`required_owner_confirmations=[user]`、`owner_confirmation_required=true`、`write_authorized=false`。
これは製品の承認ではなく、この候補を計画として保存するための perttool の確認条件である。

確認対象は上記の専用計画全体。代案はこの仮見積もりを修正してから保存すること。
保存を承認しても、詳細ADRの採用・実装開始・実データ移行・配布を承認したとは扱わない。
候補 bytes は 9144 bytes。新規ファイルなので既存ファイルの変更時刻・削除行はない。
候補 SHA-256：`5e7035ccbb1fc746e43cec4a4d0e99a65e71f733ac157d395c67cc1132e671bb`。

## 候補全文

```text
# Existing .pert plans should normally be maintained through perttool commands; direct DSL editing bypasses goal/DAG owner-confirmation checks.
project SEALGRAPH_ISSUE17:
  version 5
  title "Issue #17 範囲Traceと方向別観測"
  description "P1のIssue17専用投影。全見積りはscopeからの未計測三点仮説。velocity/納期なし。resourceは同時作業1の仮定。未承認の仕様と未設定の対象環境をblockedとして保持。statusは詳細側試用後に再検討。実装・移行・配布の実行承認ではない。"
  duration_unit point
  finish FINISH

milestone START:
  title "実装計画P1が保存されている"
  state reached

resource WRITER:
  title "作業担当1枠（計算用の仮定）"
  capacity 1

milestone CONTRACTS:
  title "詳細契約の採用版が確定"
  state planned

milestone ENVIRONMENT:
  title "対象環境と検証条件が確定"
  state planned

milestone READY:
  title "S0 実装前提が揃った"
  state planned

milestone TYPES:
  title "S1 後継型とclosure検証"
  state planned

milestone AUTHOR:
  title "S2 Trace作成とSeal公開"
  state planned

milestone COMPARATOR:
  title "S3 固定二版の範囲比較"
  state planned

milestone SELF_READY:
  title "自身比較の前提"
  state planned

milestone SELF:
  title "S4 自身のTrace比較"
  state planned

milestone DIRECTIONS:
  title "S5 方向別の詳細観測"
  state planned

milestone TRIAL:
  title "S6 詳細側の試用結果"
  state planned

milestone STATUS_DECISION:
  title "status保留の継続または再検討方針を記録"
  state planned

milestone TRANSPORT:
  title "S7 転送と移行の検証"
  state planned

milestone INTEGRATION:
  title "全体適合確認の前提"
  state planned

milestone DOCS:
  title "S8 文書と公開操作が整合"
  state planned

milestone FINISH:
  title "S8 全体の適合証拠が揃った（配布・実データ移行は別）"
  state planned

task CONTRACT_REVIEW START -> CONTRACTS:
  title "S0 詳細契約の採用版を確定"
  description "ADR0033-35の差分と未決事項を提示し、所有者の採否を記録。工数は資料整理分のみで判断待ち時間を含まない。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 1p
    most_likely 2p
    pessimistic 4p
  status blocked
  requires:
    WRITER 1
  blocked_reason "詳細契約の所有者判断が未記録"
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S0; ADR0033-35 Proposed"

task ENVIRONMENT_REVIEW START -> ENVIRONMENT:
  title "S0 対象環境・資源条件と見積もりを整える"
  description "対象データ、platform、budget、測定条件と合格基準を確定し、仮ポイントを見直す。外部回答待ち時間は工数外。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 1p
    most_likely 2p
    pessimistic 4p
  status blocked
  requires:
    WRITER 1
  blocked_reason "workload/platform/性能条件が未設定"
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S0; ADR0032 A4"

task FORMAT_TYPES READY -> TYPES:
  title "S1 型・codec・closure・歴史reader"
  description "canonical IDs、空/全被覆/overflow/コピー不一致、旧v5/v6不変を検証。物理Blobの意味は維持。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 3p
    most_likely 5p
    pessimistic 9p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S1; AC1/AC2/AC11; ADR0033"

task TRACE_AUTHOR TYPES -> AUTHOR:
  title "S2 Trace作成からSeal/表示まで"
  description "recipeとcontentのatomic更新、originのexact公開、CAS/readback、仮設5/6からの移行、Trace clear/guardとinspection。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 3p
    most_likely 5p
    pessimistic 9p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S2; AC1/AC2/AC3/AC11"

task RANGE_COMPARATOR READY -> COMPARATOR:
  title "S3 全最適対応と宣言を実装"
  description "純粋byte比較器、最適経路DAGの範囲像合意、置換/削除/反復/境界、宣言競合、budget終了を独立例で検証。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 5p
    most_likely 8p
    pessimistic 16p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S3; AC4/AC5; ADR0034"

task SELF_COMPARE SELF_READY -> SELF:
  title "S4 binding・宣言と自身比較を接続"
  description "source_key binding、固定二版の宣言、ReadStable、Candidate/HEADの別基準、NO_TRACE/対象なし/未調査を表示。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 3p
    most_likely 5p
    pessimistic 10p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S4; AC4/AC5/AC8"

task DIRECTION_COMPARE SELF -> DIRECTIONS:
  title "S5 exact Causeの方向別集計・詳細表示"
  description "local factsだけを集計。direct/indirect併存、経路、scope/budget、head競合、human/JSON、STALE不変。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 3p
    most_likely 5p
    pessimistic 10p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S5; AC6/AC7/AC8/AC9/AC10/AC11"

task DETAIL_TRIAL DIRECTIONS -> TRIAL:
  title "S6 詳細側の作成・変更・追跡を試用"
  description "仮設文書で自身/上流/下流を実際に表示。入力、版、予算、操作と結果を記録。公開完了とは扱わない。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 1p
    most_likely 2p
    pessimistic 4p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S6; AC1-AC11"

task STATUS_REVISIT TRIAL -> STATUS_DECISION:
  title "S6 試用結果からstatus保留を再評価"
  description "必要性が見えたら表示範囲と副作用を所有者へ提示。未判定なら保留を記録。status実装をこのtaskに含めない。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 1p
    most_likely 1p
    pessimistic 3p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S6; ADR0035 section8 owner decision"

task NATIVE_TRANSPORT AUTHOR -> TRANSPORT:
  title "S7 native snapshotと移行異常系"
  description "全Blob/opaque orphan/typed closure、exact旧ID、local除外、absent load、公開後出力失敗、legacy transport維持。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 3p
    most_likely 5p
    pessimistic 10p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S7; ADR0032 sections3.3/10; ADR0033/35"

task NORMATIVE_SYNC INTEGRATION -> DOCS:
  title "S8 規範文書・help・completionの整合"
  description "各sliceの文書更新を集約し採用契約との整合を確認。format4/5/6の説明と新schemaを混同しない。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 1p
    most_likely 2p
    pessimistic 4p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S8; docs/requirements architecture storage-format cli"

task FULL_VERIFICATION DOCS -> FINISH:
  title "S8 全体検証・独立レビュー・証拠整理"
  description "gofmt/vet/test/npm ci/clone/complexity/deadcodeとcompletion、必要なrace。AC対応と残課題を記録。実repo移行とreleaseなし。 三点値は未計測の相対規模見積もり。"
  estimate:
    optimistic 2p
    most_likely 3p
    pessimistic 6p
  status planned
  requires:
    WRITER 1
  source "docs/process/issue-17-origin-trace-implementation-plan-2026-09-17.md; S8; AGENTS validation; AC1-AC11"

gate READY_CONTRACT CONTRACTS -> READY:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"

gate READY_ENV ENVIRONMENT -> READY:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"

gate JOIN_AUTHOR AUTHOR -> SELF_READY:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"

gate JOIN_COMPARATOR COMPARATOR -> SELF_READY:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"

gate JOIN_TRIAL STATUS_DECISION -> INTEGRATION:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"

gate JOIN_TRANSFER TRANSPORT -> INTEGRATION:
  reason "P1の依存関係。全incoming完了が必要。実行承認を意味しない。"
```

## 保存結果（2026-09-17）

所有者の「はい。」は、この候補を指定パスへ保存する確認に対する承認。fresh preview と同一性を照合後、perttool batch apply の actor=codex / accepted-by-owner=user により一回の exclusive create を行った。
保存後SHA-256は提示候補と一致。document check は errors 0 / warnings 0。analyze は32.833p / 担当1枠49.333pを再確認し、blocked解除を仮定するPTRES-303を保持。dag next の推奨は空で、CONTRACT_REVIEW/ENVIRONMENT_REVIEWがblocked。
実装開始、詳細ADRの採用、環境条件の決定は行っていない。
