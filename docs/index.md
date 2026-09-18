# Documentation index

The `docs/` tree is the system of record for design knowledge. `AGENTS.md` is only its map.

## Normative

- [`requirements.md`](requirements.md) — product requirements and invariants.
- [`storage-format.md`](storage-format.md) — canonical repository and hash model.
- [`cli.md`](cli.md) — CLI contract and command semantics.

## Architecture

- [`architecture.md`](architecture.md) — package/backend boundaries.
- [`integrations.md`](integrations.md) — Git sidecar, llmthink, secdat, perttool.

## Decisions

- [`adr/0001-provenance-seal-model.md`](adr/0001-provenance-seal-model.md)
- [`adr/0002-standalone-default.md`](adr/0002-standalone-default.md)
- [`adr/0003-git-compatible-low-level-storage.md`](adr/0003-git-compatible-low-level-storage.md)
- [`adr/0004-git-plugin-sidecar.md`](adr/0004-git-plugin-sidecar.md)
- [`adr/0005-native-v1-canonical-storage.md`](adr/0005-native-v1-canonical-storage.md)
- [`adr/0006-experimental-native-v2-ids-and-tags.md`](adr/0006-experimental-native-v2-ids-and-tags.md)
- [`adr/0007-linearized-publication-and-draft-closure.md`](adr/0007-linearized-publication-and-draft-closure.md)
- [`adr/0008-candidate-lifecycle-and-safe-output.md`](adr/0008-candidate-lifecycle-and-safe-output.md)
- [`adr/0009-separate-seal-event-metadata.md`](adr/0009-separate-seal-event-metadata.md)
- [`adr/0010-stale-review-frontier.md`](adr/0010-stale-review-frontier.md)
- [`adr/0011-ref-independent-seals-and-branching-revisions.md`](adr/0011-ref-independent-seals-and-branching-revisions.md)
- [`adr/0012-format3-logical-dump-and-load-boundary.md`](adr/0012-format3-logical-dump-and-load-boundary.md)
- [`adr/0013-ref-manifest-tags-and-atomic-move.md`](adr/0013-ref-manifest-tags-and-atomic-move.md)
- [`adr/0014-explicit-path-manifest.md`](adr/0014-explicit-path-manifest.md)
- [`adr/0015-operator-inspection-contract.md`](adr/0015-operator-inspection-contract.md)
- [`adr/0016-standalone-beta-integrity-and-release.md`](adr/0016-standalone-beta-integrity-and-release.md)
- [`adr/0017-generic-link-metadata-and-query-boundary.md`](adr/0017-generic-link-metadata-and-query-boundary.md)
- [`adr/0018-local-ref-recovery-journal.md`](adr/0018-local-ref-recovery-journal.md)
- [`adr/0019-local-file-tracking-and-implicit-add.md`](adr/0019-local-file-tracking-and-implicit-add.md)
- [`adr/0020-sealgraph-native-operation-vocabulary.md`](adr/0020-sealgraph-native-operation-vocabulary.md)
- [`adr/0021-do-not-expand-attachments.md`](adr/0021-do-not-expand-attachments.md)
- [`adr/0022-terminal-first-human-output.md`](adr/0022-terminal-first-human-output.md)
- [`adr/0023-cause-scoped-revision-links.md`](adr/0023-cause-scoped-revision-links.md)
- [`adr/0024-source-adapters-and-portable-occurrence-provenance.md`](adr/0024-source-adapters-and-portable-occurrence-provenance.md)
- [`adr/0025-universal-blob-seal-material-and-provenance.md`](adr/0025-universal-blob-seal-material-and-provenance.md)
- [`adr/0026-format4-to-universal-blob-migration.md`](adr/0026-format4-to-universal-blob-migration.md)
- [`adr/0027-format5-cli-authoring-and-inspection-schemas.md`](adr/0027-format5-cli-authoring-and-inspection-schemas.md)
- [`adr/0028-upstream-impact-assessment.md`](adr/0028-upstream-impact-assessment.md)
- [`adr/0029-extensible-cause-link-metadata.md`](adr/0029-extensible-cause-link-metadata.md)
- [`adr/0030-format6-link-metadata-storage-and-migration.md`](adr/0030-format6-link-metadata-storage-and-migration.md)
- [`adr/0031-format6-link-metadata-cli-and-output.md`](adr/0031-format6-link-metadata-cli-and-output.md)
- [`adr/0032-content-origin-trace-and-directional-observation.md`](adr/0032-content-origin-trace-and-directional-observation.md) — 履歴上 Accepted: Issue #17 R1 の範囲 Trace と方向別観測（[承認記録](process/adr-0032-origin-trace-acceptance-2026-09-17.md)）。比較関連条項は ADR 0036 で改訂。
- [`adr/0033-origin-trace-storage-and-migration.md`](adr/0033-origin-trace-storage-and-migration.md) — Accepted: 保存形式と移行（[承認記録](process/adr-0033-0035-origin-trace-acceptance-2026-09-17.md)）。
- [`adr/0034-origin-trace-correspondence.md`](adr/0034-origin-trace-correspondence.md) — 履歴上 Accepted: R1 の対応判定（[承認記録](process/adr-0033-0035-origin-trace-acceptance-2026-09-17.md)）。比較関連条項は ADR 0037 で改訂。
- [`adr/0035-origin-trace-cli-and-observation-output.md`](adr/0035-origin-trace-cli-and-observation-output.md) — 履歴上 Accepted: R1 の CLI と観測出力（[承認記録](process/adr-0033-0035-origin-trace-acceptance-2026-09-17.md)）。比較関連条項は ADR 0038 で改訂。
- [`adr/0036-origin-trace-r2-presence-and-directional-observation.md`](adr/0036-origin-trace-r2-presence-and-directional-observation.md) — Accepted: 原文残存・方向別観測。ADR 0032 の比較関連部分を改訂（[４文書承認記録](process/issue-17-r2-downstream-acceptance-2026-09-17.md)）。
- [`adr/0037-origin-trace-r2-search-and-estimation.md`](adr/0037-origin-trace-r2-search-and-estimation.md) — Accepted: 完全一致探索と独立した変更後範囲推定。ADR 0034 の比較関連部分を改訂（[４文書承認記録](process/issue-17-r2-downstream-acceptance-2026-09-17.md)）。
- [`adr/0038-origin-trace-r2-cli-and-observation-output.md`](adr/0038-origin-trace-r2-cli-and-observation-output.md) — Accepted: 軽い比較と任意の推定を分ける CLI/schema。ADR 0035 の比較関連部分を改訂（[４文書承認記録](process/issue-17-r2-downstream-acceptance-2026-09-17.md)）。
- [`adr/0039-origin-trace-derived-occurrence-positions.md`](adr/0039-origin-trace-derived-occurrence-positions.md) — Accepted: 全文からの一致位置導出と件数上限付き提示の後継設計判断（[承認・非 draft Seal 記録](process/issue-17-adr-0039-acceptance-2026-09-18.md)）。旧 [draft Seal](process/issue-17-adr-0039-proposed-seal-2026-09-18.md) は履歴。後継要件・公開契約との整合は未確定。

## Agent-auditable design

- Issue #17 は [R2 差分要件](process/issue-17-origin-trace-requirement-r2-candidate-2026-09-17.md)を Accepted（[承認記録](process/issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)）。原文の完全一致確認と変更後範囲推定を分離する。[独立レビュー](process/issue-17-r2-c2-independent-review-2026-09-17.md)は採用判断の入力。[行指向の旧指示](process/issue-17-line-oriented-direction-2026-09-17.md)と[当時の詳細候補](process/issue-17-line-oriented-comparison-candidate-2026-09-17.md)は撤回後の履歴であり、今後の実装根拠にしない。旧 ADR 0032/0034/0035 の比較関連部分と[旧実装計画](process/issue-17-origin-trace-implementation-plan-2026-09-17.md)は R2 と不整合。ADR 0036～0038 と[新計画 P2](process/issue-17-origin-trace-implementation-plan-r2-2026-09-17.md)は[４文書承認記録](process/issue-17-r2-downstream-acceptance-2026-09-17.md)の exact snapshots で Accepted。[R2 と４文書の Seal 実行記録](process/issue-17-r2-downstream-seal-2026-09-17.md)に ID と読み戻し結果を示す。
- 2026-09-18 の所有者方針に基づく[ADR 0039](adr/0039-origin-trace-derived-occurrence-positions.md)は、位置を全文から必要時に導出する後継判断として Accepted。Accepted R2 の後継要件改訂と ADR 0036～0038・計画 P2 との実装上の整合は別途必要である。
- [Issue #17 R3-c1 要件差分](process/issue-17-origin-trace-requirement-r3-candidate-2026-09-18.md)は[承認記録](process/issue-17-origin-trace-requirement-r3-acceptance-2026-09-18.md)により Accepted、[非 draft Seal 記録](process/issue-17-origin-trace-requirement-r3-seal-2026-09-18.md)あり。[初回レビュー入力](process/issue-17-origin-trace-requirement-r3-review-input-2026-09-18.md)、[独立レビュー](process/issue-17-origin-trace-requirement-r3-independent-review-2026-09-18.md)、[ADR 0036～0039・P2 の整合案](process/issue-17-origin-trace-r3-downstream-alignment-proposal-2026-09-18.md)を参照。後継公開契約・実装は未決。

- [`decisions/sealgraph-design.think`](decisions/sealgraph-design.think)
- [`decisions/2026-08-14-reseal-required.think`](decisions/2026-08-14-reseal-required.think)
- [`decisions/2026-08-14-seal-revision-dag.think`](decisions/2026-08-14-seal-revision-dag.think)
- [`decisions/2026-08-17-format3-logical-dump.think`](decisions/2026-08-17-format3-logical-dump.think)
- [`decisions/2026-08-17-link-metadata-sealgraphql.think`](decisions/2026-08-17-link-metadata-sealgraphql.think)
- [`decisions/2026-08-17-local-ref-recovery.think`](decisions/2026-08-17-local-ref-recovery.think)
- [`decisions/2026-08-21-local-source-git-ux.think`](decisions/2026-08-21-local-source-git-ux.think)

## Proposals

- [`proposals/link-metadata-and-sealgraphql.md`](proposals/link-metadata-and-sealgraphql.md)

## Planning

- [`process/issue-17-origin-trace.pert`](process/issue-17-origin-trace.pert) — Issue #17 専用PERT旧投影（保存承認済み、比較関連部分は R2 と不整合。次作業の選択に使わない）。

- [`process/issue-17-origin-trace-implementation-plan-2026-09-17.md`](process/issue-17-origin-trace-implementation-plan-2026-09-17.md) — Issue #17 の旧 P1 候補（行指向・全最適比較を含む履歴）。
- [`process/issue-17-origin-trace-implementation-plan-r2-2026-09-17.md`](process/issue-17-origin-trace-implementation-plan-r2-2026-09-17.md) — Issue #17 の R2 実装計画 P2、Accepted（[４文書承認記録](process/issue-17-r2-downstream-acceptance-2026-09-17.md)）。
- [`process/issue-17-r2-wip-handoff-2026-09-17.md`](process/issue-17-r2-wip-handoff-2026-09-17.md) — 2026-09-17 の保存範囲と次回再開点。

- [`process/implementation-plan.md`](process/implementation-plan.md)
- [`process/backlog.md`](process/backlog.md)
- [`process/dogfooding-plan.md`](process/dogfooding-plan.md)
- [`process/release-checklist.md`](process/release-checklist.md)
- [`process/reseal-required-proposal-2026-08-14.md`](process/reseal-required-proposal-2026-08-14.md)
- [`process/seal-revision-dag-proposal-2026-08-14.md`](process/seal-revision-dag-proposal-2026-08-14.md)
- [`process/wip-handoff-2026-08-14.md`](process/wip-handoff-2026-08-14.md)
- [`process/wip-handoff-2026-09-10.md`](process/wip-handoff-2026-09-10.md)
- [`process/format3-logical-dump-proposal-2026-08-17.md`](process/format3-logical-dump-proposal-2026-08-17.md)
- [`process/format3-logical-dump-acceptance-2026-08-17.md`](process/format3-logical-dump-acceptance-2026-08-17.md)
- [`process/format4-native-core-acceptance-2026-08-17.md`](process/format4-native-core-acceptance-2026-08-17.md)
- [`process/format4-revision-graph-acceptance-2026-08-17.md`](process/format4-revision-graph-acceptance-2026-08-17.md)
- [`process/format4-tag-contract-acceptance-2026-08-17.md`](process/format4-tag-contract-acceptance-2026-08-17.md)
- [`process/content-ingest-acceptance-2026-08-17.md`](process/content-ingest-acceptance-2026-08-17.md)
- [`process/operator-contract-acceptance-2026-08-17.md`](process/operator-contract-acceptance-2026-08-17.md)
- [`process/local-ref-recovery-contract-acceptance-2026-08-17.md`](process/local-ref-recovery-contract-acceptance-2026-08-17.md)
- [`process/cause-scoped-revision-decision-review-2026-08-28.md`](process/cause-scoped-revision-decision-review-2026-08-28.md)
- [`process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-candidate-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-candidate-r2-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-review-input-r2-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-r2-review-ja-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-r2-review-ja-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-r2-independent-review-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-r2-independent-review-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-r2-independent-review-ja-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-r2-independent-review-ja-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-r2-acceptance-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-r2-acceptance-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-r2-acceptance-ja-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-r2-acceptance-ja-2026-09-10.md)
- [`process/adr-0029-extensible-cause-link-metadata-review-ja-2026-09-10.md`](process/adr-0029-extensible-cause-link-metadata-review-ja-2026-09-10.md)
- [`process/adr-0029-extensible-cause-link-metadata-independent-review-2026-09-10.md`](process/adr-0029-extensible-cause-link-metadata-independent-review-2026-09-10.md)
- [`process/adr-0029-extensible-cause-link-metadata-independent-review-ja-2026-09-10.md`](process/adr-0029-extensible-cause-link-metadata-independent-review-ja-2026-09-10.md)
- [`process/adr-0029-extensible-cause-link-metadata-acceptance-2026-09-10.md`](process/adr-0029-extensible-cause-link-metadata-acceptance-2026-09-10.md)
- [`process/adr-0029-extensible-cause-link-metadata-acceptance-ja-2026-09-10.md`](process/adr-0029-extensible-cause-link-metadata-acceptance-ja-2026-09-10.md)
- [`process/adr-0030-format6-link-metadata-storage-and-migration-review-ja-2026-09-10.md`](process/adr-0030-format6-link-metadata-storage-and-migration-review-ja-2026-09-10.md)
- [`process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-2026-09-10.md`](process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-2026-09-10.md)
- [`process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-ja-2026-09-10.md`](process/adr-0030-format6-link-metadata-storage-and-migration-independent-review-ja-2026-09-10.md)
- [`process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-2026-09-10.md`](process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-2026-09-10.md)
- [`process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-ja-2026-09-10.md`](process/adr-0030-format6-link-metadata-storage-and-migration-acceptance-ja-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-review-ja-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-review-ja-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-independent-review-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-independent-review-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-independent-review-ja-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-independent-review-ja-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-ja-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-focused-rereview-ja-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-acceptance-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-acceptance-2026-09-10.md)
- [`process/adr-0031-format6-link-metadata-cli-and-output-acceptance-ja-2026-09-10.md`](process/adr-0031-format6-link-metadata-cli-and-output-acceptance-ja-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-independent-review-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-independent-review-2026-09-10.md)
- [`process/cause-link-extensible-metadata-requirement-independent-review-ja-2026-09-10.md`](process/cause-link-extensible-metadata-requirement-independent-review-ja-2026-09-10.md)
- [`process/upstream-impact-assessment-decision-2026-08-31.md`](process/upstream-impact-assessment-decision-2026-08-31.md)
- [`process/upstream-impact-assessment-acceptance-2026-08-31.md`](process/upstream-impact-assessment-acceptance-2026-08-31.md)
- [`process/format5-dogfood-migration-proposal-2026-09-01.md`](process/format5-dogfood-migration-proposal-2026-09-01.md)
- [`process/format5-dogfood-migration-review-2026-09-01.md`](process/format5-dogfood-migration-review-2026-09-01.md)
- [`process/dogfooding-receipts/2026-09-01-format5-load.md`](process/dogfooding-receipts/2026-09-01-format5-load.md)
- [`process/standalone-beta-acceptance-2026-08-17.md`](process/standalone-beta-acceptance-2026-08-17.md)
- [`process/release-v0.1.0-beta.2-receipt.md`](process/release-v0.1.0-beta.2-receipt.md)
- [`process/release-v0.1.0-beta.3-checklist.md`](process/release-v0.1.0-beta.3-checklist.md)
- [`process/release-v0.1.0-beta.3-receipt.md`](process/release-v0.1.0-beta.3-receipt.md)
- [`process/release-v0.1.0-beta.4-checklist.md`](process/release-v0.1.0-beta.4-checklist.md)
- [`process/release-v0.1.0-beta.4-receipt.md`](process/release-v0.1.0-beta.4-receipt.md)
- [`process/release-v0.1.0-beta.5-checklist.md`](process/release-v0.1.0-beta.5-checklist.md)
- [`process/release-v0.1.0-beta.5-receipt.md`](process/release-v0.1.0-beta.5-receipt.md)
- [`process/release-v0.1.0-beta.6-checklist.md`](process/release-v0.1.0-beta.6-checklist.md)
- [`process/release-v0.1.0-beta.6-receipt.md`](process/release-v0.1.0-beta.6-receipt.md)
- [`process/dogfooding-receipts/2026-08-17-format4-load.md`](process/dogfooding-receipts/2026-08-17-format4-load.md)
- [`process/dogfooding-receipts/2026-08-17-r2-recurring.md`](process/dogfooding-receipts/2026-08-17-r2-recurring.md)
- [`process/dogfooding-receipts/2026-08-14-r0.md`](process/dogfooding-receipts/2026-08-14-r0.md)
- [`process/dogfooding-receipts/2026-08-14-r1.md`](process/dogfooding-receipts/2026-08-14-r1.md)
- [`../PLAN.pert`](../PLAN.pert)
