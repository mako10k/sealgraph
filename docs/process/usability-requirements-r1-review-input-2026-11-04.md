# UX要件R1 独立レビュー入力（所有者ルート指定待ち）

対象原本: `docs/process/usability-requirements-r1-2026-11-04.md`
SHA-256: `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d`
作者自己レビュー: `docs/process/usability-requirements-r1-self-review-2026-11-04.md`
自己レビューSHA-256: `5109990f05839087e2b9cc0899a84f728845f2bbe7316042488faebf35f9643b`

Stage: first-owner-review待ち。所有者のルート: 未選択。owner-added review questions: まだない。独立レビュー未開始。レビュー後の経路: 所有者がREVIEW/REVIEW_THEN_DECIDEならsecond-owner-review、REVIEW_THEN_REVISEならR2作成。REVISEならレビューを始めずR2へ戻る。

## 権威と対象範囲

根拠の特定版と権威の処置は自己レビューの原本表と処置節。受益者/通常操作/通知の範囲/原文照合/受入場面をレビューする。保存形式変更、既存公開スキーマ破壊、未承認の常駐通知、Git sidecar、auto Cause/SealはR1の採用範囲外。過去のAccepted sourceをR1が置き換えたと扱わない。

## レビュー質問

1. 3者の迷わない運用という上位目的に、UX-R01〜07とS1〜6の対応が足りているか。
2. N1が無操作時の通知を実現しないことが明瞭で、手動点検を黙って代替納品する構造になっていないか。
3. exact版/原文/Candidate/差分/影響/判断理由への導線と、更新/歴史版維持/保留の分岐を要件として十分定義したか。
4. format5/6/7、Traceなし/失敗/不完全、複数位置、ページ途中、バイナリ、省略時の意味を既存Accepted契約と混同していないか。
5. JSON/bytes-only/standaloneの境界、staleとreview claim適用可否を保持しているか。
6. 受入場面の合格意味は観測可能か。独立担当未確保と実利用者/LLM役割試験の区別が隠されていないか。
7. 後続のCLI/API/アルゴリズムの設計事項を現段階の新たな受入義務へ持ち上げていないか。

## 報告の契約

対象の本文を変更せず、exact digestを再確認してレビューする。各指摘は上位根拠/対象との矛盾、証拠不足/未知、任意/将来候補、範囲外に分類する。重大度を理由に任意提案をブロッカーへ変えない。フェーズ内外の帰属と、その根拠を示す。レビュー結果はcompletedまたはnot-reviewableで、要件を自己承認しない。所有者の指定ルートに戻す。

不明点: N1/N2の所有者判断、独立レビュー担当/実利用者の予定。これらを成立済みとは主張しない。R1本文変更は新しい版を必要とし、古いレビュー結果を引き継がない。
