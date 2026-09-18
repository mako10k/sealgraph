# Issue #17 実装計画 P3 追補 — format 5/6→7 明示移行

- 日付：2026-09-18。
- 根拠：所有者の「それも計画に含めてください」という追加指示。直前に示した対象は、[Accepted ADR 0035](../adr/0035-origin-trace-cli-and-observation-output.md) §4.3 の format 5/6→7 移行コマンドを、[Accepted P3](issue-17-origin-trace-implementation-plan-r3-2026-09-18.md) と[正本 PERT](issue-17-origin-trace.pert) の残工程へ割り当てること。
- 位置付け：P3 の承認・Seal 済み exact 本文は変更しない。この追補は P9 の実装漏れを埋める作業計画であり、要件・ADR の改訂、移行実行、公開、実データの変換または最終受入ではない。

## 追加する P9 成果

`MIGRATE_FORMAT7` を native dump/load とは独立した実装タスクとする。format 7 の旧形式 reader が利用可能になった後、次の**既に採用された契約**を実装・検証する。

1. `sealgraph migrate repository --from 5 --to 7 [--format human|json]` と `--from 6 --to 7` を明示操作として受け付ける。`--from` は実際の形式と一致必須。format 7 からの再移行と暗黙移行は拒否する。
2. [Accepted ADR 0033](../adr/0033-origin-trace-storage-and-migration.md) §2.7 の config-only transaction を守る。旧 object/REF/tag/Candidate の bytes と ID を保持し、source capture、staged `fsck`、atomic config publication、公開後の再読取と inventory 照合を行う。履歴 Seal、origin、永続 receipt を生成しない。
3. ADR 0035 §4.3 の `sealgraph/repository-migrate/v2` JSON と human 出力、件数の公開後読戻し、出力未配達時の `MIGRATION_COMMITTED_OUTPUT_UNDELIVERED` を実装する。不確実な公開後状態を自動再実行・rollback しない。既存 5→6 操作と v1 receipt を維持する。
4. 仮設 format 5/6 repository で旧 Seal/Candidate の保持、形式不一致・既移行拒否、公開前失敗と公開後の不確実性を検証する。実 repository に対する移行は本タスクの対象外とする。

## PERT と出口

既存の `FORMAT_TYPES` 到達点 `TYPES` から `MIGRATE_FORMAT7` を開始し、新設 `MIGRATION` に接続する。`MIGRATION` から `INTEGRATION` へ gate を置き、既存 `NATIVE_TRANSPORT` と `STATUS_REVISIT` の成果と合流させる。これにより `NORMATIVE_SYNC` は migration の公開コマンド/help/completion/規範文書も照合でき、`FULL_VERIFICATION` は両方の移行経路を含めた全体適合を確認できる。既完了タスクを未完了へ戻さない。

作業量の三点値は **3/5/10 point、期待値 5.5 point** の暫定仮説とする。既存の 5→6 config-only 経路を参考にしたが、5/6 の二経路、format 7 inventory、receipt と失敗境界の費用は未計測である。着手前に現在のコードへの適合と残量を見直す。Point は経過時間・工数ではなく、時刻見通しには別途観測 throughput と作業可能時間を使う。

この追加で Issue #17 の利用者向け機能が直ちに公開されるわけではない。内部全体適合までには `MIGRATE_FORMAT7`、`NORMATIVE_SYNC`、`FULL_VERIFICATION` が残る。実データ移行・配布・所有者の最終受入は別の権限と予定を要する。
