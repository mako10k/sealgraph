# Issue #17 `ENVIRONMENT_REVIEW` — 初期測定条件と実測

- 測定日：2026-09-18、19:13～19:19 JST。
- 対象：Accepted [R3 実装計画 P3](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) §2・§3 P8 と [正本 PERT](issue-17-origin-trace.pert) の `ENVIRONMENT_REVIEW`。本書は合成入力の初期測定記録であり、性能合格基準、利用可能な最大サイズ、実利用環境の保証ではない。
- 入力と各試行の生値：[process-measurements.csv](measurements/issue-17-environment-review-2026-09-18/process-measurements.csv)、別実行のシステムコール I/O 観測：[syscall-io-sample.json](measurements/issue-17-environment-review-2026-09-18/syscall-io-sample.json)。両ファイルをこの測定の正本証拠として保存する。大きな合成 repository と出力文書は一時領域のみで、Git に含めない。

## 条件

| 項目 | 観測・方法 |
|---|---|
| 実装 | ローカル branch `codex/issue-17-origin-trace-r2-wip`、測定前 HEAD `a7476df`。`go build -o /tmp/sealgraph-env-review-bin ./cmd/sealgraph` でビルドした binary SHA-256 は `acac7c630f88de272065a699e974448f142d56077204dafc086464ec679ebd3d`。 |
| platform | WSL2 Linux `6.18.33.2-microsoft-standard-WSL2`、x86_64、Intel Core Ultra 7 155H、22 logical CPU、Go 1.27.1、実行時 RAM 23 GiB、/tmp は ext4。これは対象 platform の採用決定ではない。 |
| repository | /tmp に `sealgraph init` で新規 repository を作り、テストで使用する format 7 config に置き換えた。これは実 repository の 5/6→7 移行操作ではない。公開 CLI の `add`、`trace set`、`trace source bind`、`seal` で単一 root Seal・単一 External run を作成した。 |
| 一意入力 | 元ファイル S は `bytes(range(256))` の反復をちょうど 1 MiB または 16 MiB に切り、中央へ ASCII `MAGIC-ORIGIN-17!`（16 bytes）を置換。run はその16 bytes。現在 F は同一、先頭へ `x` 4096 bytes 挿入、または末尾1 byteだけ `?` に変更した3種。元版 S は同一のまま。 |
| 大量一致入力 | S=F=`a` の1 MiB、run=`a` 1 byte、旧開始位置524288。重複する一致位置は1,048,576箇所。 `trace occurrences --view current --limit 100` の初回と cursor 続き1回を確認。 |
| transport | 1 MiB/16 MiB の一意入力の Seal repository を native dump。exact dump fileを使い、毎回別の空 directory に `load --format native-blobs-v1 --file PATH --max-input-bytes (file size+1)`。1 MiB dump 1,400,268 bytes、16 MiB dump 22,371,788 bytes。 |
| 時間・メモリ | 各ケース4回連続実行し、初回をウォームアップとして除く後続3回の外側 `time.perf_counter_ns` 壁時計中央値と最小～最大、GNU `/usr/bin/time` の `%M` 最大 RSS 中央値を記載。CLI は毎回新しい process。OS cache 等は固定していない。 |
| I/O | GNU time `%I/%O` の filesystem block counter を raw CSV に収録。別実行で `strace -f` により `read,pread64,readv,write,pwrite64,writev` の成功返却 bytes を集計した。これは syscall subset の観測であり、mmap、cache、全 filesystem I/O を包含しない。出力文書 bytes と dump input bytes は実ファイル長。 |

## 結果

3回の測定中央値。括弧内は最小～最大の壁時計。RSS は process 最大値であり、ホスト全体の使用量ではない。

| 操作と入力 | 結果確認 | 壁時計 ms | 最大 RSS MiB | 出力 bytes |
|---|---|---:|---:|---:|
| compare 一致 1 MiB | `PRESENT`、旧位置524288 | 27.7 (27.2～27.9) | 16.0 | 3,482 |
| compare 先頭へ4 KiB挿入 1 MiB | `PRESENT`、選択位置528384、`position_changed=true` | 30.0 (28.2～30.7) | 17.2 | 3,480 |
| compare 不在 1 MiB | `ABSENT_EXACT` | 29.0 (28.1～31.0) | 17.1 | 3,490 |
| compare 一致 16 MiB | `PRESENT` | 279.3 (279.3～304.3) | 124.7 | 3,487 |
| compare 不在 16 MiB | `ABSENT_EXACT` | 266.7 (249.4～280.1) | 125.6 | 3,493 |
| occurrences 大量一致 1 MiB、初回100件 | 位置0～99、`has_more=true` | 21.1 (21.0～21.6) | 13.6 | 6,003 |
| native dump 1 MiB | canonical snapshot | 31.7 (30.4～32.1) | 17.2 | 1,400,268 |
| native dump 16 MiB | canonical snapshot | 410.7 (350.2～447.3) | 166.2 | 22,371,788 |
| native load 1 MiB | `LOADED`、Blob 7、REF 1 | 309.4 (291.6～330.3) | 24.1 | 197 |
| native load 16 MiB | `LOADED`、Blob 7、REF 1 | 2,070.5 (1,929.3～2,486.0) | 243.9 | 197 |

`occurrences` 続きページは別の単発実行で位置100～199の100件、`has_more=true`、GNU time 0.03秒、最大 RSS 13,392 KiB。native load した両サイズの repository で `fsck` は `result=ok`。再 dump の exact bytes は元 dump と `cmp` で一致した。

`--estimate` 付きの不在比較を、同じ構成のより小さい合成入力で別途試した。4 KiB は0.06秒、16 KiB は1.00秒で `ABSENT_EXACT` と `CANDIDATES` を返した。64 KiB は `timeout 10s` に達して終了コード124、1 MiB は76.89秒時点でも完了せず、測定者が signal 15 で停止した。後二者に成功 JSON はない。ここでの上限と停止は測定手順であり、製品の数値 budget や `INCOMPLETE` 応答ではない。推定時間の伸びはこの合成入力に対する観測に限り、実 workload や複雑度の一般則を証明しない。

別実行の syscall I/O 観測では、16 MiB compare の成功 read 系返却 bytes は約67.4 MB、16 MiB native load は約45.8 MB、16 MiB dump の stdout 文書は22,371,788 bytesだった。GNU time のウォーム後3回では全ケース `%I=0`。これは page cache に影響され、物理媒体から読まなかったことや総 I/O がゼロだったことを示さない。各ケースの `%O` と syscall subset の内訳は生値を参照。

## 解釈・残る判断

- この測定で、P8 詳細側試用に使用できる合成入力・platform・コマンド・初期時間/メモリ/I/O 記録が揃った。実際の受益者が使うファイル種類・サイズ、同時実行数、許容時間/メモリ、対象 platform は未決で、測定値を合格基準へ昇格しない。
- `--estimate` は少なくとも今回の64 KiB合成不在ケースで10秒以内に完了しなかった。詳細側試用では、推定を実行する入力サイズと停止方法を明示し、観測結果を `PRESENT/ABSENT_EXACT` の意味変更に使わない。数値予算や公開契約を追加するなら所有者と設計へ戻す。
- Accepted ADR 0035 §4.3 の明示的 5/6→7 migration command は現行 PERT の `NATIVE_TRANSPORT` 対象外で未実装。今回の測定はその代行ではない。Issue #17 全体適合前に、正本計画の担当と見積もりを照合する。

再現の最小経路：同じ revision で binary を buildし、上表の byte 構成で新規仮設 repository を作る。`add root --root --clear-cause-links --content MAGIC-ORIGIN-17!`、`trace set root --recipe recipe.json`、`trace source bind source-A --file source.bin`、`seal root` の順に実行する。recipe は source `{name:s,source_key:source-A,file:source.bin}` と external run `{length:16,source:s,source_start:|S|/2}`。compare は `trace compare --ref root --max-graph-visits 100 --format json`、一覧は `trace occurrences --ref root --baseline head --run-index 0 --view current --limit 100 --format json`。各 command の stdout をファイルへ保存し、外側の process 時間と GNU timeを採る。大量一致入力は content/run を `a`/1 byteへ変更する。
