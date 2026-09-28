# Issue #17 R4-c1 — ADR 0042 設計判断入力（2026-09-28）

- レビュー対象: [ADR 0042 日本語全文](../adr/0042-direct-substring-trace-authoring.md)、SHA-256 `270d6982869e58e6f99e32d8ca0d7d237fa5aba35e229c227b7edb8d39123afe`
- 提案の由来: [draft Seal と直接 Cause の記録](issue-17-r4-adr-0042-proposed-seal-2026-09-28.md)。draft Seal は所有者採用を意味しない。
- 現在段階: Proposed CLI 設計。採否は未決。
- 上位: [Accepted R4-c1 要件](issue-17-direct-substring-r4-acceptance-2026-09-18.md)と[要件 Seal](issue-17-direct-substring-r4-seal-2026-09-28.md)、Accepted ADR 0033/0035/0039/0040/0041 の該当契約。

## 目的と変更

現行 `trace set REF --recipe PATH` は利用者に byte offset と recipe の作成を求める。提案は同じ `trace set` に、`--source-file PATH --source-key KEY --content-file PATH|-` の排他的な直接入力モードを加える。入力 bytes を Candidate 内容全体とし、元ファイル中の最小 byte offset を単一 External run に保存する。既存の recipe 経路、format 7、`trace-mutation/v2` receipt、元ファイル全文保持、明示 Seal、local binding 非作成を維持する。R4-c1 の公開 CLI について、以前に採用された直接入力の設計 baseline はない。

初回提示後の変更は、採用時に ADR 0035 §2/§2.1 と ADR 0033 §2.4 の recipe 必須形をどの範囲で後継化するか、一文で限定した点だけである。コマンド形、入力 bytes、保存・出力の提案は変えていない。初回提示の digest `c1baed8c` は現在のレビュー候補ではない。

## レビュー境界

この段階で決めるのは、公開コマンド・flag、入力方法、recipe との排他、失敗と成功出力の契約である。例えば `--source-key` を利用者に要求するかはここで決める。後続実装では、関数名、局所的なコード分割、fixture、探索処理の具体的な Go API を決める。例えば最小 byte offset を返す内部関数名は今回の判断対象ではない。実装を検証するテストと Seal はさらに後続の成果であり、今回の設計採用だけで完了しない。

## 選択と影響

1. **提案を採用**: 既存 mutation と receipt を共有できる。二つの入力モードの排他検証が必要になる。入力文字列は file または stdin から読み、argv に置かない。
2. **別コマンドへ修正**: 入力形は独立して見えるが、既存 `trace set` と同じ Candidate 更新・receipt の契約が別の公開操作に分かれる。
3. **インライン文字列を許す形へ修正**: 短い入力は容易だが、shell 履歴とプロセス引数に指定文字列が残り得る。

元ファイル全文は既存 Trace と同じく保持される。`source_key` はパスから推測できないため明示入力とした。保存時の最小 `source_start` と、後日の `trace compare` の `selected_match_start` は同一を保証しない。要件本文の bytes は変更しない。

所有者には ADR 0042 日本語全文の採用、修正、別案の指定を求める。承認された exact revision を固定した後に、上位 Cause を持つ設計 Seal、R4-c1 実装計画・PERT 接続、コード・検証へ進む。
