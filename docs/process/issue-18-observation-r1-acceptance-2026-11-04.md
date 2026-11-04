# Issue18 観測区分R1 要件受入記録

2026-11-04、ユーザーは「Issue #18の観測区分の要件案R1をACCEPTします。既存のstale集合・Seal条件・安定出力を維持し、観測理由とbaselineを分離する範囲で、設計・実装を進めてください。」と指示した。

対象原文: issue-18-observation-requirements-r1-2026-11-04.md。
SHA-256: `96e4aba15517472049c5134b5720ef6c4689fafb01ca2d3477fe4fcc45aa1df4`。
独立レビュー: issue-18-observation-r1-independent-review-2026-11-04.md、SHA-256 `55f54610088b45c248128802a0d365f66cb5be207e7bce8a56234af1e157f91f`、completed、必須改訂となる確定矛盾なし。経路はREVIEW_THEN_DECIDE → ACCEPT。

原文のProposed表記は改変せず、この記録で当該exact版をAcceptedとする。既存STALE membership/frontier/refs-only/normal admissionを保ち、Revision観測、active所属、理由、Candidate存在/比較、版別draft、workfile baselineを分離する増分を採用した。新たな公開出力は設計で具体化し、既存安定出力を無断変更しない。draft pointer/cascade/原子multi-pointer/軽量status置換/データモデル変更は含まない。

設計・実装進行は今回依頼された範囲内で行う。設計の具体策を担当者がこの契約内で選択することと、後日の製品成果受入を区別する。この記録はpush/release/install/Issueクローズや初見実利用者試験の成功を意味しない。
