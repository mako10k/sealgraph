# ADR 0041: Trace mutation receipt に入力ファイル名を含める

- Status: Proposed（2026-09-18。出力方針 B は所有者が選択済み。本書の exact schema は採否待ち）
- Decision Owner: Operator
- 対象: format 7 の `trace set` / `trace clear` 成功出力。保存形式、Trace の意味、`trace show/compare` は対象外。
- 上位契約: [Accepted Issue #17 R1/R2/R3](../process/issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)、[Accepted ADR 0033～0035 の記録](../process/adr-0033-0035-origin-trace-acceptance-2026-09-17.md)。
- 置換範囲: [ADR 0035](0035-origin-trace-cli-and-observation-output.md) §4.1 の `trace-mutation/v1` 成功 receipt、§4.2 の「狭い既存 mutation receipt は従来どおり」、[ADR 0038](0038-origin-trace-r2-cli-and-observation-output.md) の mutation receipt 維持条項、および [ADR 0027](0027-format5-cli-authoring-and-inspection-schemas.md) の狭い mutation receipt 維持条項が format 7 の `trace set/clear` に及ぶ範囲。ADR 0035 §2.1 の「操作結果」におけるファイル名の扱いも具体化する。これらの明示的な後継化は本 exact 候補の所有者承認を条件とし、旧 exact snapshots は保持する。
- レビュー範囲: 本書全文（日本語）。

## 1. Context

ADR 0035 §2.1 は、元ファイル全文を保持するファイル名と byte 数を、検証後・Blob 保存前の stderr と、成功した操作結果の両方へ含める。しかし同 ADR §4/§4.1 は未知 member を許さない固定 JSON `sealgraph/trace-mutation/v1` を定め、その `stored_sources` にファイル名がない。このため stdout の receipt だけを保持する呼出側は、どの入力ファイルを全文保持したかを読み取れない。ADR 0035 §4.2、ADR 0038、ADR 0027 の「狭い mutation receipt は従来どおり」という条項も、現行契約の維持を要求している。

現在の実装は stderr に入力ファイル名と byte 数を出し、JSON receipt には `byte_length` と Snapshot/Blob ID を出す。human stdout は source 件数だけを出す。所有者はこの二つの出力を合わせて「操作結果」とする案 A を退け、成功結果にもファイル名を含める案 B を選択した（2026-09-18「Bでしょ。 Aは不自然すぎる。」）。この選択はファイル名を出す方向を確定するが、member 名・順序・null 規則・schema 版の exact 契約までは指定していない。

ADR 0033 の SourceSnapshot は不透明な `source_key` と元ファイル全文 BlobID だけを持つ。`source_key` から path を推測しない。recipe の `file` は今回のコマンドが読んだ入力名であり、SourceSnapshot、OriginMap、Seal、後日の local binding における永続的な path 主張ではない。recipe が既存 SnapshotID を再利用するときは、今回読み取るファイルがない。

## 2. Decision candidate

### 2.1 成功結果の表示

`trace set` は、新規ファイル入力ごとに、検証後・最初の Blob 保存前の stderr へ相対入力ファイル名と exact byte 数を表示する。成功後の human stdout にも、同じファイル名と byte 数を source ごとに表示する。JSON stdout では §2.2 の `input_file` と `byte_length` を返す。stderr は成功 receipt の代わりにはしない。保存対象表示のための追加確認待ちは設けない。

既存 SnapshotID を再利用した source は今回のファイル入力がないため、JSON の `input_file` を `null` とし、human stdout では SnapshotID と「既存 Snapshot 再利用」を表示する。`trace clear` には source 入力がなく、`stored_sources` は空配列とする。

### 2.2 JSON receipt

`trace set` と `trace clear` の成功 JSON schema を `sealgraph/trace-mutation/v2` とする。compact UTF-8 JSON と末尾 LF 一つ、未知 member 拒否、次の exact member order を適用する。

```text
mutation_receipt: schema, operation, ref, before_digest, after_digest, changed, stored_sources
stored_source: source_key, snapshot_id, content_blob_id, byte_length, input_file
```

`input_file` は file 入力なら recipe に記した作業ディレクトリ相対の path をそのまま返す。既存 SnapshotID 入力なら `null`。`file` 入力の source は、同じ full bytes が既に object store に存在していても、その操作の入力名を返す。`input_file` は今回の操作に限る情報であり、canonical object、後日の binding、歴史上唯一のファイル名を表さない。raw source bytes は receipt に含めない。

`stored_sources` は recipe の source 一件につき一件を返し、同じ SnapshotID を使う複数 source も省略しない。SnapshotID 昇順を主キーとし、同一 ID では `input_file` の `null` を先に、その後は非 null path の UTF-8 byte 昇順とする。全 member が同じ重複 entry の順序は結果 bytes に影響しない。その他の値、digest、`changed`、空配列、操作単位は ADR 0035 §4.1 のまま。

§4 の固定 schema 規則に従い、`v1` 名で追加 member を出さない。本 exact 候補が所有者に Accepted された場合に限り、上記の receipt 維持条項を format 7 の `trace set/clear` について置換し、同操作は `v2` だけを出す。`v1` への出力切替や negotiation を設けない。旧 repository で Trace 操作を可能にしない。`trace show/compare` と既存コマンドの schema は変更しない。

## 3. Alternatives and consequences

| 案 | 効果 | 採否理由 |
|---|---|---|
| A: stderr と stdout の組を操作結果とする | `v1` は維持できる | stdout receipt だけでは入力名が分からず、所有者が退けた |
| B1: `v1` の `stored_sources` に member を追加 | schema 名は増えない | ADR 0035 の exact/unknown-member 規則と ADR 0027 の一 schema 一意味の規則に反する |
| **B2: `v2` に `input_file` を追加** | stdout だけで今回の入力名と byte 数を読める | 本提案。schema 更新、human 出力とテストの変更が必要 |

入力ファイル名が JSON receipt に残るため、receipt を保管する側にもその相対 path が残る。既に stderr では表示するが、JSON の長期保存・共有先は呼出側が管理する。Snapshot 再利用時の `null` を「元ファイルが存在しない」または「元版全文を復元できない」と解釈しない。元版全文は SourceSnapshot.content Blob から復元する。

## 4. Claim / Evidence / Action

| Claim | Evidence | Action |
|---|---|---|
| C-41-1: 成功結果にも元ファイル名が必要 | ADR 0035 §2.1 と所有者の B 選択 | `trace set` の human/JSON 成功出力に入力名と byte 数を出す |
| C-41-2: 旧 `v1` の member 集合は変更しない | ADR 0035 §4/§4.1、ADR 0027 C-IO-001 | `trace-mutation/v2` を定義し、format 7 の set/clear に適用する |
| C-41-3: 入力 path は immutable 由来の一部ではない | ADR 0033 SourceSnapshot と ADR 0035 recipe の二形式 | `input_file` を操作時の情報に限定し、Snapshot 再利用時は null とする |

## 5. Implementation, review and follow-up

本 ADR が Accepted になった後、CLI の set/clear receipt と human 表示、help、固定 JSON テストを更新する。保存 object、Candidate/Seal、SourceSnapshot/OriginMap、`trace show/compare`、元ファイル全文の復元処理は変更しない。`trace set` の pre-store stderr を維持し、成功 JSON 一文書だけを stdout に出す。検証後に `TRACE_AUTHOR` の P2 完了を再判定する。

所有者が B の方向を選択済みであることと、本 exact `v2` 候補を Accepted とすることは別の判断である。本 ADR の採用判断には、ADR 0035 §4.1/§4.2、ADR 0038、ADR 0027 の receipt 維持範囲を format 7 の `trace set/clear` に限って後継化する判断を含む。独立レビュー、所有者の採否、実装、ローカル検証、公開・移行はそれぞれ別の状態として記録する。現時点で保護すべき外部 `v1` 消費者の有無と配布状態は未確認であり、公開・互換性の主張はしない。
