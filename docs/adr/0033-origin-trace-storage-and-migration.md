# ADR 0033: Origin Trace の保存形式と移行境界

- 状態：Proposed
- 日付：2026-09-17
- 決定者：Operator
- 言語：日本語。レビュー対象は本ファイル全体（JSON schema 名、コマンド名、識別子は技術上の固有表記として原文のまま保持する）。
- 要件根拠：Accepted Issue #17 R1
- 上位設計：ADR 0032 D1。承認候補 SHA-256 `072314438483af19d2eb9b84b36d6a5b15d1ed358068f1849da473f34a3c38d0`
- 実行権限：なし。本提案は実装、移行、dump/load、commit、push、release、deployment を許可しない。
- 関連：ADR 0025、ADR 0028、ADR 0030、ADR 0031、`docs/storage-format.md`

## 1. 文脈と根拠

ADR 0032 §3.1〜3.3 は OriginMap、SourceSnapshot、範囲比較、方向別観測を定義し、§9〜12 の A1 は永続 schema、canonical bytes、typed closure、歴史形式の可読性、移行、ADR 0028 の change preimage を保存形式 companion に委ねている。Issue #17 R1 の承認記録は、これらの保存・移行・資源事項を未決として残している。

根拠の対応は次のとおりである。

- E1：ADR 0032 §3.1〜3.3、§4、§9〜12、および上記 D1 digest。
- E2：`docs/process/issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md` の「残る事項と承認範囲」。R1 の未決事項として保存場所・版識別、移行、資源上限を挙げる。ただし全文 source snapshot の保持方針は R1/ADR 0032 の設計として扱う。
- E3：ADR 0025 の「One universal immutable Blob」「Typed instances」「Format-5 canonical structured Blobs」。Blob は単一の immutable 層で、型はその上の検証義務である。
- E4：ADR 0030 の「Repository and typed-schema versions」「Historical typed-object compatibility」「Candidate compatibility」「Repository migration」。世代を schema 文字列で厳密に分け、旧 ID と bytes を保持する。
- E5：ADR 0028 の Assessment closure と upstream-change identity。Assessment の採用参照は Cause/revision edge と別の typed closure である。
- E6：`docs/storage-format.md` §13 の `sealgraph/upstream-change/v2`。member order は `schema, before_seal, after_material, after_root, after_draft, after_cause_links` である。
- E7：`docs/cli.md` §2 の `migrate repository`、`load`、`load-receipt`。現行仕様に汎用 `export` コマンドはない。

## 2. 提案する決定

### 2.1 後継世代

後継 repository 世代を format 7 と提案する。

```text
repository_format = 7
object_format = sha256
ref_format = manifest-v1
```

追加する typed schema は `sealgraph/origin-map/v1`、`sealgraph/source-snapshot/v1`、`sealgraph/provenance/v3`、`sealgraph/seal/v7`、`sealgraph/candidate/v7` とする。物理 store は ADR 0025 の universal immutable Blob のままとし、`sealgraph/material/v1`、Blob identity、attachments、Cause Link、previous-revision assertion、root/draft、REF manifest、Cause の向きは変更しない。schema 文字列で dispatch し、member 形状から世代を推測しない。

### 2.2 typed record と canonical bytes

全 record は ADR 0025 の compact canonical UTF-8 JSON、末尾 LF なし、厳密な member order、必須 member、unknown-member 拒否、decode/validate/re-encode、byte equality を使う。

`SourceSnapshot` の順序は `schema, source_key, content`。

```json
{"schema":"sealgraph/source-snapshot/v1","source_key":"<opaque-key>","content":"<blob-id>"}
```

`source_key` は空でない UTF-8 の不透明な値で、path、REF、Git から推測しない。`content` は source 全体の immutable Blob である。

`OriginMap` の順序は `schema, content, runs`。

```json
{"schema":"sealgraph/origin-map/v1","content":"<content-blob-id>","runs":[{"kind":"external","length":20,"snapshot":"<source-snapshot-id>","source_start":100},{"kind":"untraced","length":10}]}
```

external run の順序は `kind, length, snapshot, source_start`、untraced run の順序は `kind, length` とする。run は content 順で、ソートしない。length は正、source_start は非負。隣接 untraced は結合し、隣接 external は同一 snapshot かつ source 範囲が連続する場合だけ結合する。ゼロ長、隙間、重複、保存時の非 canonical adjacency は拒否する。

`Provenance v3` の順序は `schema, root, draft, cause_links, origin`。

```json
{"schema":"sealgraph/provenance/v3","root":false,"draft":false,"cause_links":[],"origin":"<origin-map-id>"}
```

`origin` は OriginMap ID または JSON `null`。`Seal v7` は `schema, material, provenance` の順序を保ち、Material v1 と Provenance v3 を参照する。`Candidate v7` の順序は `schema, ref, expected_ref_head, content, attachments, root, draft, cause_links, origin` とし、origin は OriginMap ID または `null` とする。

OriginMap の `content` と、外側の Material が指す content Blob は同一 BlobID でなければならない。全 run が untraced の map と `origin: null` は区別する。OriginMap は Cause/revision edge ではない。

### 2.3 構造検証と整数表現

length、source_start、run 数、配列 index は、表現上は `0..18446744073709551615` の unsigned 64-bit JSON integer と提案する。canonical spelling は最短の十進表記で、符号、先頭 zero、負値、小数、指数、浮動小数を拒否する。加算・乗算の overflow は allocation や範囲比較の前に検出する。

これは構造表現の境界であり、性能保証ではない。content byte 長への run 長の完全一致、source Blob 内の範囲、該当 byte の一致、typed Blob の存在・schema・canonical bytes を検証する。不正構造を MATCHES、DIFFERS、UNRESOLVED、NO_TRACE に変換しない。

最大 file size、run 数、graph size、memory、CPU、storage budget は accepted authority に定義されていないため、この ADR では数値上限を追加しない。workload、platform、metric、threshold は実装前の未決事項である。

### 2.4 Candidate authoring guard

Origin authoring は successor repository 専用とする。local source binding と correspondence の非 canonical record は CLI companion が所有し、typed closure に含めない。

- `trace set REF --recipe PATH [--content-file PATH|-]` は既存 Candidate にだけ適用する。recipe、source の exact bytes、content、map を検証し、Candidate の content と origin を一つの atomic Candidate update として置き換える。root/draft、attachments、Cause Link は保持する。
- 新規 Candidate は既存 `add` で作成し、`trace set` は暗黙に作成しない。
- `trace clear REF` は origin を明示的に削除する。
- origin を持つ Candidate の content を通常編集で変更する場合、同じ明示操作で有効な trace を置くか明示的に clear しない限り拒否する。map の黙った削除・不正な保持は禁止する。
- authoring 中に新しい SourceSnapshot/OriginMap の immutable Blob を put してよい。ただし既存 Blob は overwrite せず、Seal/REF の公開は既存の別途 authorized publication path まで行わない。

`seal REF` は観測した Candidate v7 の `origin` を、その exact ID または null のまま Provenance v3.origin に写す。map を再生成・正規化し直して別 ID に変えず、null に落とさない。Material.content も Candidate.content と一致させ、公開前に両方の等値と closure を検証する。不一致なら REF CAS の前に拒否して元 Candidate/REF を保持する。公開後は REF が指す Seal の Material/Provenance を読み戻し、この二つの一致を確認する。CAS 後の readback 失敗は公開済みの可能性を保持して reconcile し、自動 rollback/reseal しない。歴史 Candidate の明示 projection は §2.7 の origin:null 規則に限定する。

### 2.5 typed closure と native snapshot

closure は `Seal v7 -> Material v1/Provenance v3 -> OriginMap v1 -> SourceSnapshot v1 -> source Blob` とする。Cause/revision traversal と typed object traversal は分ける。

format 7 の typed closure transport として、`sealgraph/native-snapshot/v1` の native snapshot document を提案する。root CLI が command spelling を所有し、この ADR は document shape と storage invariants を所有する。

```json
{"schema":"sealgraph/native-snapshot/v1","repository_format":7,"object_format":"sha256","ref_format":"manifest-v1","blobs":[{"id":"<blob-id>","data_base64":"<RFC4648-padded>"}],"refs":[{"ref":"<REF>","manifest_base64":"<canonical-ref-bytes>"}],"candidates":[{"ref":"<REF>","candidate_base64":"<canonical-candidate-bytes>"}]}
```

`blobs` は物理的に保持されている全ての valid Blob を含め、typed role を推測できない unreferenced Blob は opaque orphan として同じ一覧に保持する。`refs` は全ての exact REF manifest と tag を含み、`candidates` は全ての Candidate を含む。各配列は ID または REF の bytewise 順で一意に並べる。base64 は RFC 4648 padded form、whitespace なしで、decode 後の payload bytes をそのまま表す。snapshot は format-7 config の値と、全ての object/REF/Candidate の exact bytes を含む。

snapshot の dump は、OriginMap、SourceSnapshot、source Blob を含む typed closure、全ての root と Candidate prospective state、さらに physically retained な opaque orphan を含む。local/source binding、correspondence、logs、cache、locks など非 canonical state は含めない。orphan の意味や型を dump 側で推測しない。dump は repository writer guard と全 config/object/REF/Candidate inventory の最終照合により、一つの coherent repository capture から出力する。current source file を比較・再取得する意味ではない。

load は `.sealgraph` が absent の target にだけ許可し、既存の non-empty store へ merge しない。入力を検証・stage し、BlobID hash、canonical typed bytes、REF manifest の identity、Cause/previous/typed closure の全参照、Candidate prospective state、staging 内の config/object/REF/Candidate の整合と入力 document の安定した exact bytes を検証した後、exact bytes と IDs を保ったまま atomic publish する。load は除外した local binding や現在の source file の観測を要求しない。公開直前に destination が absent のままであることを検証する。既存の `sealgraph migrate extract` / `load` / `load-receipt` による legacy universal-blob transport は変更しない。root CLI の提案コマンドは `dump --format native-blobs-v1` と `load --format native-blobs-v1 --file PATH --max-input-bytes N` とし、receipt は CLI companion の success output とする。

欠落、型違い、noncanonical、hash mismatch、範囲不一致は、stdout 出力や公開の前に拒否する。local path、binding、correspondence、timestamp、working file は canonical dump/load の由来に含めない。

### 2.6 Historical read matrix

| repository | 読み取り | 書き込み | origin の表示 |
|---|---|---|---|
| format 5 | v5/v1/v5 の exact pair | format 5 rules | `NO_TRACE` |
| format 6 | v5/v1/v5 と v6/v2/v6 の exact pair | format 6 rules | `NO_TRACE`（metadata projection は ADR 0030） |
| format 7 | 上記に加え v7/v3/v7 | v7/v3/v7 のみ | 旧世代は `NO_TRACE`、v7 は null または typed origin |

Cross-generation pairing は不正とする。format 5/6 reader は format 7 を拒否する。旧 object の bytes、SealID、ProvenanceID は保持し、再符号化しない。

### 2.7 明示 migration と rollback 境界

format 5/6 から 7 への移行は、owner acceptance と implementation authority の後に実行する config-only migration とする。既存 object、REF manifest、tag、Candidate bytes を全て保持し、Seal を新規 author する対象選択は行わない。移行で origin を履歴に推測追加しない。

具体的な migration command 名と receipt schema は CLI companion で定める。ただし実行契約は、source capture、config-only staging、staged `fsck`、atomic config publication、repository digest と runtime format の readback を必須とする。既存 route では migration receipt は persistent canonical object ではないため、canonical migration receipt の readback は要求しない。format 5/6 の Candidate を format 7 runtime で publish する場合は、旧 Candidate の全 field を保持した `origin: null` の v7 projection を、通常の一回の Candidate/REF publication として作る。これは別の authoring ではなく、現行 Candidate の明示された successor projection である。

公開前の staging だけを破棄できる。公開後の自動 rollback、downgrade、delete、overwrite、retry は行わない。receipt 未配達や readback 不一致は、その状態を独立に readback して reconcile する現行 recovery contract に委ねる。この ADR は未定義の recovery command を追加しない。

### 2.8 ADR 0028 upstream-change preimage

現行 v2 の member order は E6 のとおりである。提案する v3 はその順序を継承し、末尾に `after_origin` を追加する。

```text
schema, before_seal, after_material, after_root, after_draft, after_cause_links, after_origin
```

`after_origin` は after state の完全な nullable OriginMap 参照である。origin の変更は、他の field が同じでも change identity を変える。これは Assessment-free identity の変更だけであり、AssessmentID、Cause/revision edge、Assessment adoption gate を追加しない。

## 3. Alternatives と結果

| 案 | 結果 |
|---|---|
| v5/v6 に origin member を追加 | unknown-member rejection と旧 identity を壊すため不採用。 |
| 非 canonical sidecar だけに保存 | Seal が由来を immutable に commit できないため不採用。 |
| Material attachment に source range を保存 | 既存 Material semantics と provenance を混同するため不採用。 |
| 全 source snapshot を Blob として保持 | ADR 0032 の採用済み設計を継承する提案。保存量と機密範囲の運用評価は別の実装前確認事項。 |

## 4. Claims / Evidence / Actions

| Claim | Evidence | Action |
|---|---|---|
| C1：由来は immutable typed provenance である | E1, E3 | A1：schema と closure をレビューする |
| C2：旧 ID/bytes は読める | E2, E4 | A2：v5/v6/v7 matrix と config-only migration をレビューする |
| C3：Cause、revision、Assessment の境界は不変 | E1, E5, E6 | A3：closure traversal と v3 preimage をレビューする |
| C4：Candidate content と origin は黙って分離しない | E1、Candidate guard | A4：trace set/clear の atomic 更新をレビューする |
| C5：性能・資源閾値は未決 | E2, E7 | A5：実装前に workload/platform/metric/threshold を定義する |

## 5. Consequences と risks

安定した由来 identity、完全な typed closure、旧世代の bytes/ID 保持、trace の黙った消失防止が得られる。費用は source 全体の保持、closure 増大、byte-range 検証、Candidate atomic update の実装複雑性である。性能、容量、機密データの具体的な合否基準はこの ADR では確定しない。

## 6. Review / follow-ups

- Owner review：format 7 の schema 名と member order。
- Owner review：unsigned-64-bit の表現境界と、運用上限を未決にする扱い。
- Follow-up：CLI companion で recipe grammar、local binding、correspondence、migration command/receipt、native snapshot の command routing と success output を定義する。
- Follow-up：実装前に workload、platform、metric、performance/resource threshold と検証条件を定義する。

本 ADR は Proposed のままであり、実装、実 migration、data rewrite を意味しない。
