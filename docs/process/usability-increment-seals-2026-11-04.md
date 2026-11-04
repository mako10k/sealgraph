# UX R1 個別Seal判断・実行記録

Source implementation commit: `1706835`。各操作は原本/現Candidate/依拠するexact Seal/差分/影響を読み、担当者が個別に実行する。生成スクリプトやループで判断・Cause・発行を済ませない。

## 判断1: R1受入記録

原本はusability-requirements-r1-acceptance-2026-11-04.md。ユーザーのR1 ACCEPTと通知範囲N1、設計実装依頼を転記した実行者記録であり、署名や独立した本人確認ではない。依拠する計画は採用時の特定版 `6718cbf73eee629090e9d53784355127fe4f7fc367a1adebff0cfe0e66613e0f` を維持する。進捗が変わったから計画Causeを再発行/最新化する判断はしていない。原計画本文とrecordのR1後続判断境界を読み、後続の実装は当該ユーザー指示の範囲内で進めた。

新規REF `sealgraph/decision/usability-r1-acceptance` に旧版/Candidateなし。追加全体が差分。targetのpreviousを新たに主張する根拠はなくno-previous。対象は計画採用の後続であって既存の仕様/Issue17根拠の置換ではない。

判断1結果: `f64d7a35b482e618fc7a72237a543856f1c6efc543b11101bd7a6f8cc47a7c62`。Candidateのroot=false/draft=false/EXPECTED_ABSENT/唯一のexact Causeとraw本文一致を確認して個別発行した。発行後のshowと原本文byte一致を確認済み。

## 判断2: Accepted R1の原文

原本usability-requirements-r1-2026-11-04.mdはSHA-256 `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d` のまま。Status Proposedという本文を改変せず、別の受入記録がこのexact版のAcceptedを示す。通知N1、原文/差分/影響、更新/維持/保留、既存契約維持、3者試験と未実施実利用者の区別を改めて確認した。

依拠するのは読み戻した受入記録 `f64d7a35b482e618fc7a72237a543856f1c6efc543b11101bd7a6f8cc47a7c62`。旧版なし、新規REF sealgraph/requirement/usability-r1、追加全体が差分。previousを主張する旧要件はなくno-previous。既存normative仕様Sealを更新したという主張はせず、この増分の原文と採用根拠の特定版を固定する。

判断2結果: `469cb29b45b53fbc7a1b69f6fe8efe50460c52efac0e57cc134af5902bbf99f1`。Candidateと唯一のCause、raw原文一致を確認して個別発行、show/raw読み戻しも一致。

## 判断3: 設計判断

ADR0044とbasic/detailed designを原本で読み、R1にある操作導線を新しいread-only CLIへ配分し、既存保存形式/JSON/個別判断境界を維持することを確認した。具体的CLI/表示上限は所有者の設計実装依頼の範囲内で担当者が選択したもので、別途の所有者によるADR逐語承認を主張しない。独立コードレビューはこの境界との照合を行い、未解消必須指摘なし。

新規REF sealgraph/decision/adr-0044-read-only-navigation、旧版なし。追加全体が差分。依拠するのはR1の `469cb29b45b53fbc7a1b69f6fe8efe50460c52efac0e57cc134af5902bbf99f1`、対象のpreviousなし。この設計は点検出力/原文照合の追加であり、自動修復や定期通知へ拡張しない。副作用はread-onlyの新しい公開入口だけで、成果受入/実利用者実証を意味しない。

判断3結果: `71483ce60d4772ca08c74d7ef8689006d64c3ca21b30ad6b31a45678de8ca522`。原本とCandidate raw本文、唯一のR1 exact Causeを確認して個別発行。showとbyte読み戻し一致。

## 判断4: 実装と案内の特定版

原本は明示25ファイルのpath/size/SHA-256 manifest（sourceは担当者が明示したgit:1706835）。ソースコード、CLI文書、要件/設計の対応、独立レビューと結合検証記録を確認した。manifestは参照ファイルの本文を保存する機能ではない。ソース本文はlocal Git commit1706835と列挙パスから照合する必要があり、push/releaseは未実施。選んだ一覧以外を全リポジトリ内容として認定しない。

依拠する設計は読み戻したADR0044の `71483ce60d4772ca08c74d7ef8689006d64c3ca21b30ad6b31a45678de8ca522`。新規REF sealgraph/implementation/usability-r1、旧版なし・追加全体が差分、targetのprevious主張なし。4つのread-only表示/2つの案内を実装し、full checks/独立コードレビューを通した範囲を固定する。実利用者の理解・成果受入・定期通知を実装済みとは主張しない。

判断4結果: `df53550fd4082921e32dccfe5244d9eedfe30dc2d8ee04a40425d4b97862a7ae`。25ファイルの一覧・bytes/hash・source・aggregateと、Candidateの唯一のCauseを確認し個別発行。showとraw byte一致。ファイル本文がmanifestへ格納されたとは主張しない。

## 判断5: 検証記録

原本usability-integration-verification-2026-11-04.mdを読み、実行したgofmt/vet/test/npm ci/clone/complexity/deadcode/completionの観測結果と、実利用者/最終成果受入を未実施とする制限を確認した。独立コードレビュー原本のsnapshot hashも現実装と一致する。依拠する実装は `df53550fd4082921e32dccfe5244d9eedfe30dc2d8ee04a40425d4b97862a7ae` の固定path-digest claimで、検証対象commit1706835/25ファイルと対応する。

新規REF sealgraph/verification/usability-r1、旧版なし、追加全体が差分。Cause target previousなし。検証範囲を当該commitの追加CLIと案内へ限定し、実利用者/任意入力/全競合/公開環境へ一般化しない。

判断5結果: `26acfce3416f9ef1f583ec50e9b7e1bb2a21875787c89c9cf635c1a5dc3ff20b`。Candidateの原本文/唯一の実装exact Causeを確認し個別発行。show/raw読み戻し一致。

既存のCauseやSealを置き換えず、採用計画の固定版を保った。今回もstaleを消すためのCause付替えや再発行はしていない。発行後fsckとstale観測を行い、構造検査と意味的正しさを区別する。どのSealも実利用者試験や最終成果受入を認定しない。
