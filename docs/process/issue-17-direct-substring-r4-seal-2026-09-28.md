# Issue #17 R4-c1 Accepted 要件の Seal 記録 — 2026-09-28

- 対象: [R4-c1 要件本文](issue-17-direct-substring-requirement-r4-c1-2026-09-18.md)全文
- 採用権限: [所有者の Accepted 記録](issue-17-direct-substring-r4-acceptance-2026-09-18.md)
- 本文 SHA-256: `c42f678fc965027c6c8536998499ce063d9ba82eaa7e6806044cce443b4bdb42`
- REF: `sealgraph/requirement/issue-17-origin-trace-r4`
- 非 draft SealID: `82798e71409cc92712837b02d9f994ee2b6a3b1dba79ecaf60880e2140a55625`
- 直接 Cause: [Accepted R3-c1 要件 Seal](issue-17-origin-trace-requirement-r3-seal-2026-09-18.md) `de2fe3533eeb2553606f533f76c8d74e469356a746c6dde765694b94c67d1316` の一件

## 実行と読戻し

承認記録が対象とする exact 要件ファイルの SHA-256 を照合し、上記値に一致した。実行前の standalone repository は format 5、`fsck=ok`、`stale --scan` は空。R3 REF は上記の非 draft Seal を指し、R4 REF と Candidate は存在しなかった。

R4 REF に要件ファイルの exact bytes を `--content-file` で指定し、`root=false`、`draft=false`、R3 Seal を対象とする Cause 一件を持つ Candidate を作成した。R3 target に構造上の前世代がないことを `--no-previous` で記録した。公開前の Candidate は `expected_ref_head=null`、`EXPECTED_ABSENT`、prospective SealID が上記 ID であり、raw content の SHA-256 が承認記録と一致した。その後、この REF を一度 Seal した。

公開後の `show --format json` は上記 SealID、`root=false`、`draft=false`、R3 への直接 Cause 一件、空の previous-revision assertion を返した。`show --raw-content` の SHA-256 は承認済み本文と一致。`sealgraph fsck --format json` は `result=ok`（19 Seals、18 REFs、unreferenced Blob なし）、`sealgraph stale --scan --format json` は `statuses=[]`。

R4-c1 は Accepted R3-c1 の後継要件であり、Cause は R4 から R3 の exact Seal へ向けた。後継 CLI 設計・実装・検証は R4 要件の Cause には入れない。公開 CLI の具体形と、実装・検証成果の下流 Seal は別の段階で扱う。
