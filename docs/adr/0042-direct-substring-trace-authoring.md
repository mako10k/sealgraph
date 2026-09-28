# ADR 0042: 部分文字列の直接入力による Trace 登録

- Status: Proposed（2026-09-28、所有者判断待ち）
- Decision Owner: Operator
- 上位要件: [Accepted Issue #17 R4-c1](../process/issue-17-direct-substring-r4-acceptance-2026-09-18.md)。要件 Seal は `sealgraph/requirement/issue-17-origin-trace-r4` の `82798e71409cc92712837b02d9f994ee2b6a3b1dba79ecaf60880e2140a55625`。
- 維持する決定: [Accepted ADR 0033/0035 の記録](../process/adr-0033-0035-origin-trace-acceptance-2026-09-17.md)に対応する SourceSnapshot・OriginMap と既存 recipe 経路、[Accepted ADR 0041 の記録](../process/issue-17-adr-0041-acceptance-2026-09-18.md)に対応する format-7 mutation receipt、[ADR 0039](0039-origin-trace-derived-occurrence-positions.md)・[ADR 0040](0040-origin-trace-occurrence-listing-and-paging.md) の比較と位置一覧。
- 対象: 新しい入力経路の公開 CLI、入力検証、成功出力、既存 `trace set` との関係。保存 schema、比較、Seal 公開、local binding は変更しない。
- 採用時の後継範囲: ADR 0035 §2 の `trace set REF --recipe PATH` 構文、同 §2.1 と ADR 0033 §2.4 の recipe 経路の記述に、以下の直接入力モードを追加する範囲だけを後継化する。既存 recipe モード自体、Candidate 更新境界、その他のコマンドと保存契約は維持する。所有者が本 ADR を採用するまでは、これらの Accepted 条項を変更しない。

## Context

R4-c1 は、利用者が元ファイルの byte 位置と JSON recipe を書かずに、指定した非空 UTF-8 文字列を既存 Candidate の内容全体として登録できることを要求する。元ファイル内の最小 byte offset を単一 External run の `source_start` に保存し、元ファイル全文を SourceSnapshot に保持する。現在の `trace set` は recipe を必須とし、位置指定を利用者に求める。

`source_key` は ADR 0033 の不透明な識別子であり、パスや REF から推測できない。入力文字列をコマンド引数に置くと shell 履歴やプロセス引数に残り得るため、既存の `--content-file PATH|-` を直接入力にも使用する。

## Decision candidate

既存 `trace set` に、recipe と排他的な直接入力モードを追加する。

```sh
sealgraph trace set REF --source-file PATH --source-key KEY \
  --content-file PATH|- [--format human|json]
```

`--source-file`、`--source-key`、`--content-file` はこのモードで各一回必須とする。`--recipe` と `--source-file` は排他的で、recipe モードに `--source-key` を混ぜることも拒否する。recipe モードの既存引数と動作は維持する。`KEY` は空でない UTF-8 の不透明値を利用者が指定し、パス、REF、ファイル内容から生成しない。`PATH` は既存 recipe の file と同じ作業ディレクトリ相対の安全な通常ファイルで、`.sealgraph/`、絶対パス、`..`、symlink を拒否する。`--content-file` は既存の安全な相対ファイルまたは標準入力 `-` を受ける。入力文字列を argv に直接置く flag は追加しない。

両入力を安定して読み、内容 bytes が空、または有効な UTF-8 でなければ失敗する。元ファイル全文に対して内容 bytes の連続一致を byte 0 から探索し、最初の一致を `source_start` とする。これは重複・重なりを含む一致のうち最小 byte offset である。一件もなければ失敗する。登録後の `trace compare` の `selected_match_start` は従来の探索順に従い、登録時の `source_start` と同じ位置を保証しない。全位置の列挙は `trace occurrences` の責務とする。

成功時は、読み取った内容 bytes を Candidate content 全体とし、一つの External run がその全体を覆う。元ファイル全文を SourceSnapshot の immutable Blob として保存する。既存 Candidate 一件の content と OriginMap を `trace set` の一回の原子的更新として公開し、root、draft、Cause、attachments、expected REF HEAD を維持する。Candidate がなければ既存 `add` を案内して失敗する。local binding、Seal、REF HEAD は暗黙に作成・変更しない。読み取り、入力検証、競合で失敗した場合は Candidate と REF HEAD を変えない。保存途中に残る未参照 Blob の扱いは既存 `trace set` と同じ。

元ファイル全文の名前と byte 数は、既存の保存前 stderr 告知と成功時 human/JSON receipt に出す。成功 JSON は ADR 0041 の `sealgraph/trace-mutation/v2` をそのまま使い、`operation="set"`、`stored_sources` は一件、`input_file` は `--source-file` の相対パスとする。新 member や schema 版は追加しない。人向け成功出力、失敗時の stdout/stderr と exit code は既存 `trace set` に従う。

実装は既存の `TraceSet` に、内容、元ファイル全文、単一 External run を渡す。新しい永続フィールド、repository format、別の mutation API は作らない。

## Alternatives and consequences

| 案 | 利点 | 負担・採らない理由 |
| --- | --- | --- |
| 提案: `trace set` の排他的な直接入力モード | Candidate 更新、receipt、失敗境界を既存経路と共有できる | 既存コマンドに二つの入力モードができるため、排他検証と help が必要 |
| 別の `trace set-substring` コマンド | 入力形は独立して見える | 同じ Candidate 更新と receipt の公開契約が二か所に分かれる |
| インラインの `--text STRING` | 短い入力では操作が少ない | 入力 bytes が argv や shell 履歴に残りやすい |
| recipe の自動生成だけを提供 | 現行 CLI を変更しない | R4-c1 の直接入力を公開操作として満たさない |

## Claim / Evidence / Action

| Claim | Evidence | 後続 Action |
| --- | --- | --- |
| C42-1: 位置と recipe を利用者に要求しない | Accepted R4-c1 目的・AC1〜AC2 | CLI で最小 byte offset を計算し単一 run を構成する |
| C42-2: 既存の由来・失敗境界を維持する | Accepted R4-c1 AC3〜AC4、ADR 0033/0035 | 既存 `TraceSet` と receipt を使用し、失敗時の Candidate/REF 不変を検証する |
| C42-3: 登録位置と比較位置を混同しない | R4-c1 承認記録、R4 独立レビュー、ADR 0039/0040 | CLI 文書と help に二つの意味を明記する |

## Review and follow-up

本 ADR の判断範囲は公開コマンド形、入力方法、検証・結果の境界までとする。実装内部の関数名、テスト fixture、処理の局所的な分割は後続実装で決める。採用後に `docs/cli.md`、help/completion、CLI と repository のテストを更新し、単一一致、重複・重なり、不一致、空・不正 UTF-8、読取失敗、Candidate 不在、競合、recipe 同等性、全文復元、binding/Seal 非作成を検証する。検証後に実装成果を上位の exact Seal に結び、別の非 draft Seal として記録する。

この Proposed 文書の作成は所有者採用、実装開始、実 repository 移行、push、merge、release を意味しない。
