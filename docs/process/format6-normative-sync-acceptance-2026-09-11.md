# Format 6 規範文書同期 受理記録 — 2026-09-11

Status: Accepted

Decision owner: Operator

Decision: 2026-09-11、OperatorはStep 4で`ACCEPT`を明示的に選択した。

## 受理したexact snapshot

- `docs/requirements.md`
  - SHA-256: `828bf756da52243d7c5ac5c8a87917820a471574ab16e62353543851a956c6e4`
- `docs/architecture.md`
  - SHA-256: `18352065ba6874691516c9fbe5d0cf51a618618078d0fe188f0b4fe0ee3cdab1`
- `docs/storage-format.md`
  - SHA-256: `c01fd58c8d655dfc248c20fb92d41078b5faf1bc7a799d7d5b441dea1a6d6f88`
- `docs/cli.md`
  - SHA-256: `95225e234c0bca875cc8b34c2a9d4c748888fa0e6df219c46249c81c840cd3ad`

受理は上記4ファイルのexact bytesだけに適用する。別のrevision、後続編集、
implementation label、release version、plan task、またはrepository stateへ自動的に
移転しない。

## 権威とレビュー証拠

- Requirement authority: Accepted ADR 0029、0030、0031、およびADR 0029が参照する
  accepted revision-2 requirement。
- Step 2 route: `REVIEW`、追加質問なし。
- Review input:
  `docs/process/format6-normative-sync-review-ja-2026-09-11.md`
  - SHA-256: `1d212d69cb37f0ae4770a78192264777cd5c5f61bda8e797124e006e3c187e33`
- Step 3 authoritative report:
  `docs/process/format6-normative-sync-independent-review-r2-2026-09-11.md`
  - SHA-256: `83af74ccf6278387a098d199ecf5efa28a0a1b9d076961500ecd986492e0408c`
- Step 3 result: `completed`。前回F-01〜F-04はresolved、7つのacceptance
  criteriaはすべてpass、material contradiction/evidence gapはなし。
- 日本語レビュー支援:
  `docs/process/format6-normative-sync-independent-review-r2-ja-2026-09-11.md`
  - SHA-256: `a0cbd7bccbdf07035a8fdc0a27516a534f26b4c7480fd9340f6a51946ce22eec`

## 受理範囲

- format 5 contractを維持したformat 6 Cause Link metadata successor
- generic・opaque・namespaced・identity-bearing metadataとgraph independence
- exact v5/v6 typed pairing、historical bytes/ID保持、Candidate v5 compatibility
- explicit config-only format-5-to-format-6 migration boundary
- `upstream-change/v2`とmetadata-sensitive change identity
- format 6 CLI/output schema、human representation、Candidate mutation
- current repository formatに応じたSeal selector generation
- format 6追加commandのrepository-independentなhelp navigation

## 維持される境界

この受理は、candidate edit、ADR change、implementation change、commit、push、PR、
merge、release、deployment、実repository migration、runtime mutation、Git-sidecar
implementation、metadata query/registry/validator、Assessment-reference storageを
authorizeしない。これらはそれぞれ別の権威と実行判断を必要とする。
