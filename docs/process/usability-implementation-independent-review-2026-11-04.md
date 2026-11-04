# UX R1 実装の独立レビュー（2026-11-04）

Result: **completed**。下記スナップショットの追加CLI実装を読取り専用で独立レビューした。レビュー中に指摘した5件のコード/導線問題は担当作者が修正し、修正を原本で再確認した。現スナップショットに未解消の必須コード修正指摘はない。これは全入力・全競合順序の完全性の証明や、実利用者試験・所有者の製品受入・release/installの完了ではない。

## 1. 対象の特定版と境界

Git HEAD: `5ea13bd3594fd8ce316501c6a96138b289e03f49`。追加/変更中の実装であり、HEADそのものをレビュー対象の完成版と扱わない。以下のSHA-256がレビューした作業ツリーのexact file snapshot。後から当該ファイルが変われば、その差分について追加確認が必要。

| 対象 | SHA-256 |
| --- | --- |
| `docs/process/usability-requirements-r1-2026-11-04.md` | `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d` |
| `docs/process/usability-requirements-r1-acceptance-2026-11-04.md` | `0188db2ebd37eb634f303e55f2512a631628326e7f0f76303045838885f9eb64` |
| `docs/adr/0044-read-only-operator-navigation.md` | `d791a2ee7abea960d5c87889880af4bf6580649ae842b2736bad5cd14bbac047` |
| `docs/process/usability-design-2026-11-04.md` | `41267d927cfaabc2da91c2add0a0f050d5dc420363f150f8ff67e4958ea2494b` |
| `internal/cli/inspect.go` | `88b70b98dd28d578e85cd0a650c83c646ca5415728230c9b6d7834ff18927ee5` |
| `internal/cli/inspect_review.go` | `db33fda6349f9293ea76ce4aa0ad44946834b134977af5eb3a6cb53fa75ca6c6` |
| `internal/cli/inspect_human.go` | `2659936d0cd77083828eff77a0207707599904d534e50c38fdf1916baa417c18` |
| `internal/cli/inspect_test.go` | `fdf15e12f362aaa2435b7be9e56508f502435b268bc422ae00184219dc3eaf94` |
| `internal/cli/compare_text.go` | `7f069ccdbf8230b9b276170e6dc906f245fcf6368a7076d7c20650bba677ae3e` |
| `internal/cli/compare_text_test.go` | `18f28da3329702b2c6f92bd4c34abb72989e9855363d0dce68f9f73be3aa7d85` |
| `internal/cli/trace_read.go` | `6a47d84bf6afaa46664da6a97401b084a859b5ba2d6de665ce73067e7d59445b` |
| `internal/cli/trace_read_test.go` | `7967a4dd68173c4b2c8d5ba840b9533ff8375b7720cdfcaa99243be9f5309232` |
| `internal/cli/operator_output.go` | `389a5e3ce24046cd096f28848f4e94c0811953bad6813bc464d1974382e19d81` |
| `internal/cli/workflows_help.go` | `52599553803a136088bfdf4795a70aa79e912c617e81a6e72455d8098d52c3bb` |
| `internal/cli/run.go` | `66e9fcdb95a18c8b1315a18c8d1055fdb45b38e1bdf3bbc5cc66435737c1ff81` |
| `internal/cli/trace.go` | `0fff32dda462f7f4a63d388e7c2803f84de25cd25fe59420736f3f8f10d95410` |
| `internal/cli/help.go` | `45a1d06ffb4f7f262678087b8fdb6b484aa719996736380e49101722f2316472` |
| `internal/cli/completion.go` | `f3461da2783cf61e6cba4945c526f97928883475c074a87bfbb73d784bb1867b` |

Accepted R1は `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d`。所有者はN1（点検時の通知）を明示ACCEPTし設計実装進行を依頼した受入記録を確認した。ADR0044と基本/詳細設計はその範囲内の追加読取りCLIの配分であり、保存形式・既存JSON・個別判断/Sealの境界を保持する。

このレビューの現フェーズ内対象は `inspect`、`compare --text`、`trace read` の実装、dispatcher/help/completion/workflowsヘルプ、exact baseline/原文/不完全観測の表現、出力/計算量境界とエラー伝播。運用文書全体の6場面検証、リポジトリ全検査、実利用者操作試験、所有者による製品受入は別の結合/操作レビュー工程に残る。基本設計を選び直したり既存human renderer全般を作り直したりするレビューにはしていない。

## 2. 確認した実装境界

- `inspect`: 全headsを捕捉し、Statusの既存グラフ検証後に表示件数を制限する。omitted/completeness、source/baseline、stale、Trace、historical claim/applicabilityを別軸に保つ。Candidate digest/workfile identity/Trace結果/review結果とheadsを出力前に再照合する。多ファイル同時snapshotや将来の予約とは主張しない。
- `compare --text`: 2つのimmutable選択を固定して内容を読み、元のselectorを再解決して選択が変わっていないことを確認する。既存compare JSONは既存経路のまま。入力preview/差分行budgetは各64KiB、LCSは1048576 cellsを上限とし、超過時は明示的なfallback/OMITTEDとなる。全文のBlob読取りと検証まで64KiBへ限定した機能ではない。
- `trace read`: explicit baseline/External run/source viewを既存occurrence入力/再検証APIで扱う。snapshotとcurrentを分け、未束縛/読取失敗はINCOMPLETE。バイナリを含むhuman excerptはescape、JSON bytesはbase64。whole sourceのexact rawは明示要求だけで出力し、取得不可はエラー。代表位置やoffset差を編集移動の証拠にしない。
- 新コマンドはcanonical writer、Cause/review mutation、Seal発行、Git discovery、background pollingを呼ばない。出力エラーの確認は今回の新しいoperator出力に限定している。

重大な保存状態変更、未確認をcleanへ読み替える確定した経路、通常human出力への未escapeバイナリ流出、無限のdiff計算、既存JSON schema変更は、調べた追加コードでは検出しなかった。未検出は全ての将来入力/環境の保証ではない。

## 3. 指摘と修正の再確認

### IMP-01: inspectのHEAD Traceをseal baselineと表示していた

- **INSIDE / 対象設計との矛盾、P2 → 修正確認済み**。
- 元の経路は `buildTraceCompareLocal` の共通既定値を使い、REF HEADの結果もbaseline.kind=sealになった。ADR0044が求めるCandidate/HEADの明示とずれた。
- `inspectTraceLocal` がHEADをheadへ補正し、初回組立と再検証に同じhelperを使うことを確認。既存Trace JSON全体を変更していない。

### IMP-02: historical claimのhuman表示から判断本体へ辿りにくかった

- **INSIDE / R1 UX-R06の導線不足、P2 → 修正確認済み**。
- RECORDED/applicableだけではretain/unreviewed・difference/impact/reasonを読めず、具体的なreview表示への経路が欠けていた。
- human inspectがHEADには `review show @exact`、Candidateにはshell-quoted REFと `--candidate` を使った具体的な読取りコマンドを出すことを確認。claimとapplicability/staleは独立のまま。

### IMP-03: compareの「理由」読取り先が本文を返さないhuman表示だった

- **INSIDE / R1 UX-R04/05の導線不足、P2 → 修正確認済み**。
- format5とformat7の既存 `show --format human` はCause messagesの件数中心で、新しいcompareが案内する「recorded rationales」をそこから読むことができなかった。
- 新しいcompareの案内先を完全なCause/messagesを含む `show --format json` へ変更したことを確認。既存human renderer全般の変更は要求していない。

### IMP-04: 新しいhuman経路がstdout書込み失敗を成功扱いした

- **INSIDE / ADR0044のoperational error境界との矛盾、P2 → 修正確認済み**。
- inspect/trace read humanがfmtの書込みエラーを捨てて0を返し得た。
- 新 `writeOperatorOutput` がwrite errorとshort writeを非zeroへ変換する。両human経路をbufferへrenderしてこのhelperへ渡し、compare text/raw sourceにも適用した原本を確認。`TestOperatorHumanOutputDeliveryFailure` はerrorWriterで新しい両human経路を検証する。

### IMP-05: 集約中のhistorical claimの版の混在を再照合していなかった

- **INSIDE / exact baselineと再検証の証拠不足、P2 → 修正確認済み**。
- review組立は複数回のCandidate読取りを含むが、捕捉したprospective IDと内側のreview.SelectedSealの一致確認がなく、最終集約でreview集合を再照合していなかった。Candidateの一時変更などで別版のclaimが混ざる懸念があった。
- `inspectReviewFor` がselected IDを明示照合し、`revalidateInspectReviews` がreview集合を再取得/DeepEqualして相違をエラーにすることを確認。混在subjectを拒否する退行テストも追加されている。
- これは個別の検出/再照合の確認であり、多ファイル同時snapshotや全ABA順序への形式的証明ではない。設計が表明する観測限界を維持する。

## 4. 後続事項と範囲外

- **OUTSIDE / 任意の文言整合、対応確認済み**: ADRの「64KiB output」という曖昧さは、実際の入力preview/差分行budget各64KiBに合わせて修正された。ヘッダ・既存比較表・省略メッセージ・extract案内まで総計64KiBを保証する新しい要求は追加していない。
- **OUTSIDE / 結合・運用文書検証**: 3者6場面を入口だけで辿る試験、全チェック、初見の実利用者試験の証拠はこのcoreコードレビューでは作成していない。運用doc統合中にS4の初回 `review show --candidate` がrecord作成前に置かれるとREVIEW_REQUIREDになる点を担当作者へ伝えた。操作例の最終確認はその工程で行う。
- **OUTSIDE / 実利用者試験・所有者成果受入**: 未実施。作者以外のエージェントによるこのコードレビューを、初見の人の使い勝手実証や所有者の成果承認に置き換えない。
- **BOUNDARY_DISPUTE**: 今回のcoreレビュー対象と後続工程の帰属について未解決の争点は検出しなかった。N1/N2を再び要件問題として開き直したり、GUI・定期通知を新規ブロッカーへ加えたりしていない。

## 5. 検証と結論

レビュー担当が実行したfocused check:

```text
go test ./internal/cli -run 'Test(Inspect|TraceRead|CompareText|TextChanges|Workflow|Operator)' -count=1
ok github.com/mako10k/sealgraph/internal/cli 18.494s
```

上記実行後、担当作者が追加したmixed-subject退行テストを原本で確認し、次も実行した。

```text
go test ./internal/cli -run 'Test(InspectHistoricalClaimIsSeparateFromApplicability|OperatorHumanOutputDeliveryFailure)$' -count=1
ok github.com/mako10k/sealgraph/internal/cli 0.653s
```

最終snapshotの各hashを照合した。全repository検査の結果は統合担当の記録に委ねる。

本レビューはcompletedで、現snapshotに未解消の必須core修正はない。後続の結合検証・操作レビュー・所有者成果受入へ進むためのレビュー資料となる。タスク完了、Seal発行、commit/push、release/installをこの報告自体が許可または実行するものではない。本文・計画・コード・Sealをレビュー担当は変更していない。
