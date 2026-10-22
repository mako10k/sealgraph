# Issue #22: 個別レビュー済みの旧特定版依拠 — 実装・検証記録

2026-10-22 JST / ローカル実装・指定検証完了。未コミット。

対象: https://github.com/mako10k/sealgraph/issues/22

作業先: `/home/katsumata-m/sealgraph`

ブランチ: `codex/issue-22-historical-review`

基点: `21195bc273c97c751a0f91085c5b6c7482e7ac9a`。開始時の作業ツリーはclean。
リモート最新SHAとの新規比較は実施していない。

## 実現した動作

通常の `seal REF` は、従来のnon-draft active-leaf制約を保持する。
個別レビュー記録をCandidateに保存し、明示的に `seal REF --review-historical`
を選んだ場合には、確定した旧特定版への依拠をnon-draftで発行できる。
旧版への依拠はSTALE_DIRECT/STALE_TRANSITIVEとして残り、最新版への整合に
読み替えない。draftは暫定状態のまま区別する。

追加したCLIは次のとおり。これは操作形の説明であり、実リポジトリへの
発行を実施した記録や、その実行の承認ではない。

```text
review prepare REF
review set REF --file PATH|-
review clear REF
review show SELECTOR
review show REF --candidate
seal REF --review-historical
```

prepareは事実の列挙のみで、判断や理由を生成しない。setは既存Candidate
一つの専用レビュー添付だけを更新し、CauseやHEADを変更しない。入力には
全direct/transitive Causeの正確なSeal ID、比較版、維持／未確認の判断、
差分、影響、維持理由を含める。未確認は保存できるが、新経路での発行は止める。

記録を、専用添付を除いたprospective SealIDと、対象REF以外の全HEAD集合に
結び付けた。本文、他の添付、OriginMap、Cause、世代、観測HEADが変われば
適用不能となる。対象自身のHEADは既存のexpected-old/CASと完全観測再検証で
保護する。別REFへの記録流用、draft依存、網羅不足も拒否する。

showは、保存した判断と現在の適用可否、選択した完全SealIDを分けて表示する。
`applicable` は記録と版の対応を意味し、発行成功や意味上の承認を意味しない。
通常発行の拒否時のhintにも個別レビュー経路と未確認時の保留を案内した。
helpとAGENTS.mdには個別判断をスクリプトで代替しない原則を記載した。

既存のMaterial添付／Blob構造を使い、既存Seal・Provenance・Candidateの
typed schemaとformat 5/6/7は変更していない。新しい保存内容は通常の証拠Blob
`sealgraph/historical-review/v1`。任意の添付操作、汎用force、自動relink、
自動再Seal、一括発行は追加していない。

## 判断の根拠と変更範囲

- [ADR 0043](../adr/0043-reviewed-historical-cause-publication.md): 設計、既存契約の置換範囲、互換性と限界。
- [要件](../requirements.md) §4.1、[CLI](../cli.md) §10、[保存形式](../storage-format.md) §15、architecture、index、AGENTS.mdを整合。
- `internal/historicalreview`: 1 MiB上限、重複・未知・欠落・大文字別名キー、null、曖昧なID等を拒否する記録codec。
- `internal/repository/historical_review*.go`: 原本・観測に結び付いた事実生成、記録操作、適用性表示、発行ゲート。
- CLI dispatch/help/completionと既存の発行・観測経路に限定して接続。

[権限・自己レビュー記録](issue-22-authority-and-review-2026-10-22.md)に、
Issueとの対応、旧契約の扱い、現在フェーズの境界を記録した。
独立レビューや、所有者が新規文書の各byteを読了したという主張はしていない。

実装・仕様の対象版は[作業ツリーmanifest](issue-22-working-tree-manifest-2026-10-22.json)
の23ファイルで固定。manifest自体のSHA-256は
`70c585a92d07d4df8067cf0e08420948cc85f3b7c303b6afdc8bcd0073619888`。
この引継ぎ文書、manifest自身、検証ログは自己参照を避けるため対象外。

## 検証

[保存した検証出力](issue-22-validation-transcript-2026-10-22.txt)を参照。

| 検査 | 観測した結果 |
| --- | --- |
| `gofmt -w .`、変更Goファイルの最終gofmt | 実施。無関係なtracked変更なし |
| `go vet ./...` | 成功 |
| `go test ./...` | 全パッケージ成功。repository 97.225秒、CLI 73.421秒。この後の変更に影響する範囲は下記で再検証 |
| final focused selection | 成功、経過19.08秒。codec、直接／到達／detached依拠、format 5/6/7、未確認、draft、変更版、CAS／HEAD競合、読戻し、不変bytes、fsck、help／completion／診断を含む |
| その後の読戻しassertion分離 | 該当direct/transitive発行テスト成功、repository 1.486秒。製品コードは変更なし |
| `npm ci` | 成功 |
| `npm run clone-check` | 成功。観測された重複率0.08%、既存2件 |
| `make complexity-check` | 最終成功。途中でテスト関数の複雑度21を検出し、読戻し検証を補助関数へ分離して上限20以内に修正 |
| `make deadcode-check` | 成功 |
| `make completion-check` | 開発binary buildとBash regression成功。グローバルinstallなし |
| `git diff --check`、未追跡追加ファイルの末尾空白確認 | 成功 |

全量テストは最初の実装統合時に一度実行した。その後のcodec・表示・診断・
テスト整理について、影響するパッケージ／ケースだけを再実行した。
現在のmanifestに対する証拠は、変更のない範囲の全量結果と変更範囲の最終結果の
組合せであり、最終版で二度目の全量実行を行ったという主張ではない。
全ての発行試験は一時リポジトリで実施した。

## 未実施・限界

記録は実行者の主張であり、理解・正しさ・ユーザー承認の証明にはならない。
無関係なREFのHEAD変更でも適用不能とする保守的な設計である。
asserted lineageがないdetached対象では比較leaf集合は空になるが、他の全HEADの
特定版と実行者の差分・影響・理由は記録に残る。

knの3件の発行、registry／Cause／Seal変更、以前の自動発行履歴の再承認、
リリース、インストール、push、Issueのcloseは行っていない。

PERTの読取検査は成功したが、Issue22をfinishへ追加する更新はPTGOV-101の
所有者確認ゲートで書込み前に拒否されたため未反映。既存PLAN.pertと履歴は
保持した。[観測・診断](issue-22-planning-observation-2026-10-22.json)を保存した。
過去5件のvelocityは、この実装のactive effortへ換算していない。
内部実装・指定検証の残作業は0。計画の所有者確認と、依頼対象外の採用・公開・
実運用への適用は別の判断として残る。
