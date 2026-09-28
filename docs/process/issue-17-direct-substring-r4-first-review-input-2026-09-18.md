# Issue #17 R4-c1 初回レビュー入力

- 対象: [R4-c1 要件候補](issue-17-direct-substring-requirement-r4-c1-2026-09-18.md)全文。exact SHA-256: `c42f678fc965027c6c8536998499ce063d9ba82eaa7e6806044cce443b4bdb42`。
- 出所: 2026-09-18 の所有者発言と二つの明示回答。Accepted R1/R2/R3 と ADR 0033/0035/0039 の既存境界を維持する提案。
- 範囲: 既存 Candidate に対する一つの元ファイル・一つの External run の直接入力。既存 recipe、複数 run、Seal の明示公開、Cause/STALE は対象外。
- 確認する点: 指定文字列を Candidate content 全体にすること、複数一致の最小 byte offset、見つからない場合の Candidate 不変、元ファイル全文保持が所有者意図に合うか。
- 未決: exact CLI 文法、文字列入力方法、出力 schema、実装・公開時期。
- 独立レビューへの問い: accepted 要件や ADR と矛盾しないか、単一 run の範囲が過不足ないか、受入条件が失敗時の状態を十分区別するか。
- 初回所有者レビュー経路: 所有者は候補全文と本入力の提示後に `REVIEW` と回答した。追加質問なしで exact 候補の独立レビューへ進み、その結果を受けて二回目の所有者レビューで採否を判断する。独立レビューは候補を変更・承認しない。
