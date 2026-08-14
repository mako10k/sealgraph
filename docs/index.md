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

## Agent-auditable design

- [`decisions/sealgraph-design.think`](decisions/sealgraph-design.think)
- [`decisions/2026-08-14-reseal-required.think`](decisions/2026-08-14-reseal-required.think)

## Planning

- [`process/implementation-plan.md`](process/implementation-plan.md)
- [`process/backlog.md`](process/backlog.md)
- [`process/dogfooding-plan.md`](process/dogfooding-plan.md)
- [`process/reseal-required-proposal-2026-08-14.md`](process/reseal-required-proposal-2026-08-14.md)
- [`process/dogfooding-receipts/2026-08-14-r0.md`](process/dogfooding-receipts/2026-08-14-r0.md)
- [`process/dogfooding-receipts/2026-08-14-r1.md`](process/dogfooding-receipts/2026-08-14-r1.md)
- [`../PLAN.pert`](../PLAN.pert)
