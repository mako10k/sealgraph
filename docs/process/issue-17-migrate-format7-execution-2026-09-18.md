# Issue #17 — MIGRATE_FORMAT7 実行記録

- 実行日: 2026-09-18（JST）
- 根拠: Accepted ADR 0033 §2.7、ADR 0035 §4.3、P3 migration 追補、PERT `MIGRATE_FORMAT7`
- 範囲: format 5/6 から 7 への明示的な repository migration の実装と仮設 repository での検証。実データ移行、配布、所有者の最終受入は含まない。

## 実装と確認

`sealgraph migrate repository --from 5 --to 7` と `--from 6 --to 7` を追加した。移行は source fsck と物理 bytes/inventory の capture、私有 staging 上の format 7 fsck と保持照合、公開直前の source 再確認、config の atomic replacement、公開後の再読取を行う。旧 object、REF、tag、Candidate は書き換えない。新規の履歴 Seal や origin は生成しない。source 形式不一致と既移行は拒否する。

成功時には `sealgraph/repository-migrate/v2` の順序付き JSON または human receipt を出す。公開後に出力だけが失敗した場合は `MIGRATION_COMMITTED_OUTPUT_UNDELIVERED` を返し、再移行ではなく format 7 `fsck --format json` で確認する。format 7 fsck は `sealgraph/fsck/v4` に `origin_maps` と `source_snapshots` を追加する。既存の 5→6 と v1 receipt は維持した。

仮設 format 5/6 repository のテストで、Seal/Candidate と未参照 Blob の保持、異なる source 指定と再移行の拒否、公開前の破損拒否、公開後の出力失敗時の readback を確認した。旧形式で未参照 Blob だった新形式データが移行後に型付き件数へ分類される場合も、bytes/ID を保持したまま通過する。

## 検証結果

2026-09-18 に `gofmt -w .`、`go vet ./...`、`go test ./...`、`npm ci`、`npm run clone-check`、`make complexity-check`、`make deadcode-check` が成功した。重複検出は初回に新規 fsck 定義とテスト準備の重複を検出したため共通化し、再実行で成功した。独立レビューで指摘された未参照 Blob の分類差による誤拒否も修正し、再レビューで解消を確認した。

この記録は実装・仮設データ検証の証拠であり、実 repository の移行成功や Issue #17 全体の完了を示さない。残る `NORMATIVE_SYNC` と `FULL_VERIFICATION` で公開文書・全体契約との照合を行う。
