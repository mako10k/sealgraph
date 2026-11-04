# 改善計画の個別Seal実行記録（2026-11-04）

計画ソースcommit: `700afa8`。ユーザーはコミット後のSeal更新と実装進行を依頼した。新規2 REFを個別に判断し、別々のadd/Candidate確認/seal/readbackで作成した。スクリプト・ループによる発行やCause更新は行っていない。

| 対象 | REF | SealID |
| --- | --- | --- |
| 所有者指示の実行者記録 | `sealgraph/decision/usability-plan-20261104` | `cc2ae40ad5caff9bd968cec410e8b4b23378b757f0f030ff6c489c37ac9e396d` |
| 採用した計画の固定版 | `sealgraph/process/usability-delivery-plan` | `6718cbf73eee629090e9d53784355127fe4f7fc367a1adebff0cfe0e66613e0f` |

## 判断1: 指示の記録

原本 `usability-plan-acceptance-2026-11-04.md` を読み、ユーザーの具体的な計画適用指示と、その後のコミット/Seal/実装指示を確認した。新しいREFで旧版とCandidateはなかった。追加全体が差分。rootはこのチャットに基づく実行者記録をprovenance境界とする宣言であり、署名や独立した本人確認はない。Causeなしを明示したCandidateのraw本文・root=true・draft=false・EXPECTED_ABSENTを確認して個別発行した。発行後のraw本文は原本とbyte一致した。

## 判断2: 計画の固定版

原本 `PLAN.pert`、承認候補、依拠する指示記録の原文、発行後の正確なSealIDを確認した。正本と候補のSHA-256は `baf0564caa9498641a14d04cc04e55330cef36a6783818dfd7933c0e49d165a6`。新しいREFで旧版/Candidateなし。追加全体が差分。

根拠は指示記録の `cc2ae40ad5caff9bd968cec410e8b4b23378b757f0f030ff6c489c37ac9e396d` を採用し、previous assertionなしとした。これにより11タスク/7マイルストーンを含む計画特定版の採用根拠が追跡できる。要件/設計/成果の正しさや受入を認定しない。Candidateのraw本文、唯一のCauseのexact IDとmessage、EXPECTED_ABSENT、root=false/draft=falseを確認して個別発行した。発行後のraw本文は正本とbyte一致し、Causeも上記IDで読み戻した。

## 既存依存と結果の境界

既存spec/architecture/Issue17のSealは変更していない。今回の計画に黙って最新根拠として接続せず、変更不要の以前の特定版を維持する。原文の意味的な再承認は行っていない。発行後fsckはresult=ok、34 Seals/28 REFs、unreferenced Blobなし。stale一覧は空だが、これを内容の正しさ・ユーザー理解・成果受入の証拠にしない。

この計画Sealは進捗開始前の固定版。後続の実績更新後のPLAN.pertと同じ版だとは主張しない。次の工程はUX_REQUIREMENTSで、詳細要件の未決定事項を具体的な候補にする。
