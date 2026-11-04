# UX要件R1 自己レビュー / first-owner-review入力準備

対象: `docs/process/usability-requirements-r1-2026-11-04.md`
SHA-256: `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d`
状態: 作者による自己レビューのみ。独立レビュー・所有者の要件受入は未実施。

## 境界

現フェーズの対象は受益者、観測可能な動作、対象外、通知範囲と受入意味。CLI綴り/内部API/アルゴリズムは基本・詳細設計へ渡す。元の3者レビューと採用計画を上位根拠とし、既存実装を要件の権限へ読み替えない。

## 根拠の特定版

計画Seal `6718cbf73eee629090e9d53784355127fe4f7fc367a1adebff0cfe0e66613e0f`、指示記録Seal `cc2ae40ad5caff9bd968cec410e8b4b23378b757f0f030ff6c489c37ac9e396d`。以下は読んだ現行原本のハッシュであり、過去のSealがこの本文と同一であるという主張ではない。

| 原本 | SHA-256 |
| --- | --- |
| `docs/requirements.md` | `ac9c0bb82b9920add3d0378c4a5b5d83c0a815a0748a24a76c3629b85ed4e73f` |
| `docs/cli.md` | `ce325c21989ffafa61ce95cfbc6e1a63ab4601fceaa9080c274b731ea93f3ce3` |
| `docs/storage-format.md` | `3e0e6f2b9b2a7a21d001efc6300733e2d0011de151573a4e2e219d6945985d5d` |
| `docs/architecture.md` | `ac1fcf848855ff1ef080ba4ea0949e0208225208e9727063e8a03a6e62de60b0` |
| `docs/adr/0019-local-file-tracking-and-implicit-add.md` | `50de17fa79a1349ba72390925e66f2e3fb67f6598a054c4121fe6f7ee372d13f` |
| `docs/adr/0022-terminal-first-human-output.md` | `f3d895843aa79a13ab026352b9c9001f13c414fe639d5cd37c1a6db09a7d7f21` |
| `docs/adr/0043-reviewed-historical-cause-publication.md` | `cf582560a6afb0660a1306af79e67d2dfa2238e90c4654b947c2e4f1f70a5237` |

## 現行権威の処置

Acceptedな1操作1REF/immutable/explicit root/exact Cause/standalone Git非検出/sourceとstaleの独立/Traceの既存意味/historical review/出力とJSON境界を維持する。新しい公開CLIは設計候補で別途レビューする。保存形式変更や既存スキーマ破壊をこの要件で暗黙に許可しない。既存version/namespaceの所有権移転はない。

## 結果

- INSIDE / evidence gap: R1 §2 UX-R02のN1は点検呼出し時の通知で、無操作時の通知を満たさない。所有者がN1を採用するか、N2へ改訂するかを決めるまで、契約として受け入れられたとは扱えない。
- INSIDE / evidence gap: 3者6場面の閾値と、初見実利用者の試験かLLM役割試験かの証拠種別はR1 §3で提案した。独立レビュー担当/実利用者の確保は未確認であり、成功を主張しない。
- INSIDE / correspondence: 6 WorkをUX-R01〜07/S1〜6へ対応させ、正確な根拠版の原文確認、更新/維持/保留、原文監査、LLMの入口を保持した。自動更新やclean化への代替は含めない。
- OUTSIDE / 基本・詳細設計: CLIの綴り、サイズ/計算量上限と失敗時処理、raw抽出・本文差分・原文文脈のAPI/JSON設計は未実施。設計フェーズの作業であり、この要件へ未決定のコマンドを既成事実として入れない。
- OUTSIDE / 実装・試験: go buildは成功したが、これは現行コードのビルド確認のみ。新機能実装/独立操作レビュー/成果受入の証拠ではない。
- BOUNDARY_DISPUTE: N1/N2は製品機能範囲で要件フェーズの所有者判断とする。今回これを設計者の都合で決めない。その他に把握したフェーズ帰属の争点はない。

## 次の所有者判断

R1を対象にREVISE、REVIEW_THEN_REVISE、REVIEW_THEN_DECIDE、REVIEWのいずれかを選ぶ。推奨はN1の範囲が意図と合う場合にREVIEW_THEN_DECIDE、無操作時の通知も必要ならREVISEでN2を含むR2へ改訂。どちらも今回の実装依頼を取り消す意味ではなく、依存する製品契約の相違を解消するための判断。
