# Issue #17 R4-c1 ADR 0042 独立レビュー — 2026-09-28

- 対象: [ADR 0042 日本語全文](../adr/0042-direct-substring-trace-authoring.md)、SHA-256 `270d6982869e58e6f99e32d8ca0d7d237fa5aba35e229c227b7edb8d39123afe`
- 対応する draft Seal: `d0d8af5c7b9f5e3aad661f604bed22eb78b8f733504f2f032250ae731ee0f38e`
- 上位: [Accepted R4-c1 要件](issue-17-direct-substring-r4-acceptance-2026-09-18.md)、Accepted ADR 0033/0035/0039/0040/0041 の各 exact 本文と承認記録。
- 方式: 本文を編集しない独立担当の読み取り専用レビュー。現行 CLI/repository は実現可能性の証拠に限り、要件の権限とは扱わない。

## 固定した段階境界

現在の ADR が決めるのは、公開 CLI 構文、文字列と元ファイルの入力、recipe 経路との排他、Candidate/REF の失敗境界、成功出力と既存契約の限定後継化である。後続の実装・検証では、関数の分割、Go API、fixture、コードと help/completion と `docs/cli.md` の更新を決める。例えば `--source-key` の必須性は現在の設計判断であり、最小 offset を返す内部関数名は後続実装の判断である。

## 指摘分類

| 分類 | 観測と根拠 | 現段階の結果 |
| --- | --- | --- |
| INSIDE | ADR 0042 §Decision candidate は Accepted R4-c1 の直接入力、最小 byte offset、単一 External run、元全文 Snapshot、不一致・競合時の Candidate/REF 不変、binding/Seal 非作成を具体化する。 | 確定的矛盾なし。 |
| INSIDE | recipe 排他と既存経路維持は Accepted ADR 0035 §2/§2.1 の Candidate 更新・安全な相対 path 契約と整合する。ADR 0033/0035 の後継範囲も限定されている。 | 確定的矛盾なし。 |
| INSIDE | 登録時 `source_start` と後続 `selected_match_start` を区別し、全位置一覧は ADR 0040 の責務とする。 | Accepted R4-c1 と ADR 0039/0040 に矛盾なし。 |
| INSIDE | 成功時の `sealgraph/trace-mutation/v2`、一件の `stored_sources`、ファイル入力名は Accepted ADR 0041 §2.1～2.2 と整合する。 | 確定的矛盾なし。 |
| OUTSIDE | 現行 `internal/cli/trace.go` は recipe 専用。`internal/repository/trace_author.go` の `TraceSet` は原子的 Candidate 更新、全文保存、競合検出を実装済みで、receipt も v2。直接入力の実装、help/completion、CLI 文書、テストは後続成果。 | 後続実装・検証へ送る。現 ADR の改訂 blocker にはしない。 |
| BOUNDARY_DISPUTE | なし。 | 現在と後続の責務を変更しない。 |

この exact ADR は所有者の採否判断に進める状態である。レビューは所有者の採用を代行しない。採用、修正、別案の指定が次の判断であり、設計採用後の非 draft Seal、実装、検証は別の成果として扱う。
