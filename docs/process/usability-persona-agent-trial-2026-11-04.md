# UX R1 独立LLM役割操作試験（2026-11-04）

Result: **実施したsynthetic fixtureのS1〜6を完了**。作者とは別のエージェントが実バイナリを使い、対象ごとの原文/Candidate/根拠を読み、個別操作で経路を辿った。これはコードレビューとは別の操作記録である。ただし、この担当者は既にコードを読んだLLMで、初見の人間ではない。一般の初心者が迷わないことの実証や、所有者の製品成果受入ではない。

## 実行環境と証拠境界

- 試験repo: `/tmp/sealgraph-persona-trial-20261104-6WGNVx`。新規作成、Gitなし、初期Seal/Candidateなし。後述のsynthetic plaintextだけを入れた。試験後もこの場所に保存。
- 実行binary: `/home/katsumata-m/sealgraph/bin/sealgraph`
- binary SHA-256: `ad2ff5720c3f60a16ade7ff006444bda0dde2ff0698ccca90a0f16547c73de50`（試験前後で一致）。
- 主要入口: `docs/operations.md` SHA-256 `a30faaff5f620ac01587ff7eb66275af0237e30ab5a1a41929a0e76f0c6de9ff` と実バイナリ `help workflows`。詳細構文はそこから案内されたhelp add/review set/migrate repository/trace setとhistorical-review/v1の文書を参照。
- 操作中にコードを使って状態を書き換えたり、作者から個々のコマンドを教えてもらったりしていない。ただし事前のコードレビューによる知識はあり、それを初見と偽らない。
- main repoのコード/PLAN/Seal/Gitを操作していない。この報告だけをmain repoへ作成した。tmp内でもCause/Sealのスクリプト・ループ・自動再試行は実施していない。
- 下記の `SG` は記録の読みやすさのための**表記上の省略**で、全コマンドで上記binary絶対パスを実際に使った。シェル関数や自動発行スクリプトではない。

## 登録した特定版

| 記号 | REF/意味 | SealID |
| --- | --- | --- |
| P1 | premise、3 retries | `8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2` |
| D1 | design、P1に基づく3 retries | `5abb42d59c41239e9f90dffb390901c5440f7a3bb9684165fa89c54e53087df0` |
| P2 | premise、明示的に5 retriesへ変更 | `b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1` |
| D2 | design、P2へ採用変更、P2 previous P1をobserverとして主張 | `f8b0964b5d5d60af9324e8ff2b3e8b34493f46b650f6ce1ec8e271454421e8b4` |
| L1 | legacy、P1を歴史版としてretain | `aae22eec3eebeab40468df903b570bff3f27f675881a09307dc46e823d5f6a9e` |
| Q1 | quote、保存全文から24 bytes引用 | `410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076` |

計6 Sealを別々のコマンドで発行した。上表記号は以下の観測説明にだけ使い、実操作では実在するexact IDを選択した。

## S1 初回登録 — 実施、fixture内成功

最初の原文:

```text
premise.txt: maximum retries=3\n
design.txt: Use three retries for the synthetic policy.\n
```

原文をcatで読み、次を実行した。add後はCandidateのraw本文・JSON/比較を読み、上記のsynthetic scopeとexact Causeを意図したことを個別に判断してからsealした。

```text
SG init
SG add premise --root --clear-cause-links --content-file premise.txt --bind-source
SG candidate show premise --format human
SG candidate show premise --raw-content
SG seal premise
SG show premise --format json
SG show premise --raw-content
SG add design --non-root --content-file design.txt --bind-source --target @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --no-previous -m 'Synthetic design uses the three retries stated in this exact premise; initial premise has no asserted previous revision.'
SG candidate show design --format json
SG candidate compare design --format human
SG candidate show design --raw-content
SG seal design
SG show design --format json
SG show design --raw-content
SG inspect --format human
```

観測: initはformat5。P1はroot=true/Causeなし。D1はP1だけのexact Cause、root=false。raw本文は上記原文どおり。両REFはWORKFILE_MATCHES_HEAD、stale direct/transitiveなし。TraceはUNSUPPORTED_FORMATと明示され、format7へのexplicit migrationのhelpが案内された。rootを真実/承認とは解釈しなかった。

## S2 変更の種類を区別 — 実施、fixture内成功

premise.txtだけを `maximum retries=5\n` へ変更し、まだadd/sealしない状態で:

```text
SG inspect premise --format human
```

観測: WORKFILE_DIFFERS_FROM_HEAD、provenance stale direct=0、Trace UNSUPPORTED_FORMAT。ファイル差をstaleや原文検証済みと取り違えなかった。

その後P1原本、更新Candidate、publication baselineを読み、synthetic policyを5 retriesへ置き換えるという実際のfixture変更としてP2を発行:

```text
SG add premise
SG candidate show premise --raw-content
SG candidate compare premise --format json
SG show @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --raw-content
SG seal premise
SG show premise --format json
SG compare @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 premise --text
SG inspect design --format human
```

比較は `- maximum retries=3` / `+ maximum retries=5` をescapeした行として表示し、全文と完全なCause/messages JSONの読取りコマンドを示した。D1はP1を指したままでSTALE_DIRECTになった。P2にはこの時点でintrinsic parentもprevious assertionも存在しない。P1が現HEADのrevision leafでなくなったことと、P2がP1をsupersedeするというobserverの主張は区別し、HEAD移動そのものをその主張へ読み替えなかった。

TraceのPRESENT/ABSENT_EXACT/UNDETERMINEDの区別も、後述S6/S5で実際に操作した。workfile/provenance/Traceは混同しなかった。

## S3 新版採用 — 実施、fixture内成功

stale --frontier、D1・P1・P2のraw本文、P1のimpactを読んだ。synthetic designを5 retriesへ変更する理由は、P2が明示的にretry値を3から5へ変更し、今回のdesignがその変更を採用するというfixtureの意図である。単にclean化するためのrevision assertionではない。

```text
SG stale --frontier --format json
SG show @5abb42d59c41239e9f90dffb390901c5440f7a3bb9684165fa89c54e53087df0 --raw-content
SG show @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --raw-content
SG show @b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1 --raw-content
SG impact @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --format human
```

design.txtを `Use five retries for the revised synthetic policy.\n` に変更し、Candidateなしの状態からaddした。元のP1 Causeが保持されるのを確認してから、新しいP2 Causeを別操作で追加。完全な2 Causeを読んだ後、不要となった旧P1 Causeを別操作で除去した。

```text
SG add design
SG candidate show design --raw-content
SG candidate show design --format json
SG link design --target @b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1 --previous @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 -m 'Read both synthetic policies: the explicit 3-to-5 retry change is the intended successor for this design; this observer asserts the previous relation and adopts five retries.'
SG candidate show design --format json
SG candidate show design --raw-content
SG unlink design --target @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2
SG candidate show design --format json
SG candidate compare design --format human
SG candidate show design --raw-content
SG seal design
SG show design --format json
SG show design --raw-content
SG inspect design --format human
```

観測: D2の唯一のCauseはP2、previous assertionはP1。これはD2 observerによる主張で、P2自身のparentではない。本文は5 retries。workfile matches、direct/transitive staleなしを読み戻した。

## S4 歴史版維持 — 実施、fixture内成功

別consumerの原本 `legacy.txt` は `Legacy synthetic consumer is intentionally frozen at three retries.\n`。このfixture自体が旧値3への固定を明示するため、P2の5へ変更するとその宣言に反する。現実の互換性を調査したという意味はない。

P1/P2とlegacy原文を読み、次を個別に実行:

```text
SG help review set
SG show @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --raw-content
SG show @b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1 --raw-content
SG add legacy --non-root --content-file legacy.txt --target @8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2 --no-previous -m 'This synthetic legacy consumer explicitly retains the original three-retry policy, not the newer five-retry policy.'
SG candidate show legacy --format json
SG candidate show legacy --raw-content
SG review prepare legacy --format json > legacy-review-facts.json
```

factsを実際に読み、target=P1、compared=P2、ACTIVE_NON_LEAF、draft=false、other headsはdesign=D2/premise=P2だけを確認した。生成ループではなく、1 targetについて次のrecordを手書きした（実ファイル `legacy-review.json`）。

```json
{"schema":"sealgraph/historical-review/v1","ref":"legacy","subject_seal":"6bb40b8cbbebce665f80599c355859092a42f539e84a9e9b0e4d44ea3e38db2d","other_heads":[{"ref":"design","seal":"f8b0964b5d5d60af9324e8ff2b3e8b34493f46b650f6ce1ec8e271454421e8b4"},{"ref":"premise","seal":"b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1"}],"entries":[{"target_seal":"8509fbf3bb6e4ed213acd1b34b78db31c7f872005263c1175bc127e9936750a2","compared_seals":["b70e9c6fd67a6eac81484d81eaed593dcb963f18ab780408d61cd7fe856a34c1"],"decision":"retain","difference":"Read both originals: maximum retries changes from 3 in the selected old premise to 5 in its observed successor.","impact":"Adopting 5 would change the intentionally frozen three-retry behavior declared in legacy.txt; this fixture retains the old value.","reason":"This synthetic legacy scope explicitly requires the original three-retry policy. The old exact premise is non-draft and its text supports that declared fixture scope. No real-system compatibility is claimed."}]}
```

recordを原文と照合してから:

```text
SG review set legacy --file legacy-review.json --format json
SG review show legacy --candidate --format human
SG candidate show legacy --raw-content
SG candidate show legacy --format json
SG review show legacy --candidate --format json
SG seal legacy --review-historical
SG show legacy --format json
SG show legacy --raw-content
SG inspect legacy --format human
```

観測: L1はnon-draftでP1を保持。historical claimはRECORDED/applicable=true、provenance direct stale=1が残る。成功を最新整合や意味的承認へ読み替えていない。後述のformat7移行とQ1追加後もclaimは残り、current subject/other-headsが変わってapplicable=false（REVIEW_SUBJECT_CHANGED）と返った。これを過去判断の削除と混同しなかった。

迷った/補助を用いた点: historical recordの構造はhelpから案内されたstorage-format §15で調べる必要があった。human review showは長いDifference/Impact/Reasonをellipsisで省略するので、完全理由は自作record原本と `--format json` で照合した。これは作者からのコマンド指示ではないが、コードを読んだ経験があり初心者負荷の証明には使わない。完全理由のJSON案内補足を統合担当へ提案した。

## S6 原文監査とS2のTrace区別 — 実施、fixture内成功

Trace未対応のformat5から、点検の案内に従ってconfigとmigration helpを読み、tmp repoだけを明示的にformat7へ移行した。

```text
cat .sealgraph/config
SG help migrate repository
SG migrate repository --from 5 --to 7 --format json
```

観測: MIGRATED、既存v5 Seal 5件保持、Candidateなし。新たな原文は全文39 bytes `header\nquoted original sentence\nfooter\n`、引用24 bytesは `quoted original sentence`。

原文/引用ファイルを実際に読んで、全文保存を受け入れ、別のexplicit root quoteを作成:

```text
SG help trace set
SG add quote --root --clear-cause-links --content-file quote.txt
SG candidate show quote --raw-content
SG trace set quote --source-file source.txt --source-key source-A --content-file quote.txt --format json
SG trace show --ref quote --format json
SG candidate show quote --format json
SG candidate show quote --raw-content
SG inspect quote --format human
SG seal quote
SG show quote --format json
SG trace source bind source-A --file source.txt --format json
SG inspect quote --format human
SG trace read --ref quote --baseline head --run-index 0 --view snapshot --format human
```

各Candidate/OriginMapを読み、保存原文のoffset7から24 bytesが一致することを確認して個別に発行した。Q1は上表。SourceSnapshotは `9bd579c8e88a431d1f69594f72240f4e4517d222004be648bfdb03eac5e565da`、saved blobは `c0f40dec4a32a0cdf0613ceb74a0e8ac21b4af727bf7d4cd9356d7fb0743c0f9`。

binding前のinspectはUNDETERMINED/unresolved=true、binding後はPRESENT/unresolved=false。source bindingと保存snapshotは別の操作だと判別できた。Q1にCauseはないので、そのこと自体を原文の意味的権威とは解釈しない。

元のsource.txtを `saved-source-original.txt` へ保存した後、現在ファイルだけを74 bytes `new preamble\nquoted original sentence\nquoted original sentence\nnew footer\n` にした。引用の同じbytesが2件あるので、単一位置を歴史的移動と断定できない。

```text
SG trace read --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view snapshot --format human
SG trace read --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view current --format human
SG trace read --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view snapshot --raw-source > recovered-original.bin
cmp saved-source-original.txt recovered-original.bin
SG trace occurrences --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view both --limit 1 --format json
SG trace occurrences --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view both --limit 10 --format json
SG review show @aae22eec3eebeab40468df903b570bff3f27f675881a09307dc46e823d5f6a9e --format json
```

観測: snapshotは保存全文39 bytes/offset7、currentは74 bytes/代表offset13、saved/current IDは別。limit1はsnapshot位置7だけ、has_more=trueなので全集合とはしなかった。別のlimit10のcomplete listingではsnapshot7/current13/current38の3件、has_more=false。cursorでの継続操作自体は未試行。raw復元は原本とbyte一致。

さらにcurrent sourceだけを `replacement text without the quoted sentence\n` にした後の `inspect quote --format human` はABSENT_EXACT/difference=trueで、provenance staleは0だった。これは引用bytesがないという結果であり、意味的な削除/矛盾とは断定しなかった。

## S5 未確認を保留 — 実施、fixture内成功

上のcurrent sourceを `source-current-missing.txt` へ退避し、明示bindingが指すsource.txtを欠落させた。fixture準備後に `.sealgraph` 全42ファイルのpath/byte hash一覧を読取り専用で保存した。

```text
mv source.txt source-current-missing.txt
SG inspect quote --format human
SG inspect quote --format json > pending-inspect.json
SG trace read --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view current --format json
SG trace read --seal @410b93a47211e2492107307875e9f05027da51cf226f823bc16998075cfb2076 --run-index 0 --view snapshot --raw-source > pending-recovered-original.bin
cmp saved-source-original.txt pending-recovered-original.bin
```

観測: inspectはUNDETERMINED/SOURCE_READ_FAILED、unresolved=true。trace read currentはINCOMPLETE/SOURCE_READ_FAILED、selected_start/source lengthはnull。snapshotの全文は依然復元できた。現在の原文確認ができないため、Cause更新・relink・追加Seal・勝手なbinding変更をせず、未確認として保留した。

読取り前後の `.sealgraph` 全42ファイルのpath/hash一覧が完全一致した。`pending-before.json` と `pending-after.json` のSHA-256はともに `a8697e1981c56d0fdaf025aa420c02fe672d39884fa91470e875b7c8d65692c4`。canonicalだけでなく当時存在したlocal/runtimeファイルも含めた実観測である。

## 最終観測と証拠ファイル

最後に `inspect --format json`、`fsck --format json`、`stale --format json` を再実行し、tmpの `final-inspect.json` / `final-fsck.json` / `final-stale.json` に保存。

- fsck: result=ok、6 Seals、4 REFs、27 Blobs、unreferenced Blobなし、OriginMap/SourceSnapshot各1。
- stale: legacyだけSTALE_DIRECTでP1を指す。これを消す追加操作はしていない。
- source-A bindingは存在するがsource.txtは欠落したまま。Q1の現在原文は未確認、保存原文は復元可能。
- `saved-source-original.txt` / `recovered-original.bin` / `pending-recovered-original.bin` のSHA-256はすべて `60e9a60c887a94a3881ad41f1c6b9eb861917418cea4681016dabf0eb9eb2f86`。
- 手書き `legacy-review.json` のSHA-256は `27b5733814228486f3b37b22e57ae90f13f8b3e036ba6734d9c9276dd7f64e5b`。

## 使い勝手の観測と限界

入口→help→exact originals→個別判断→個別mutation→readbackの経路を、試した範囲では辿れた。担当作者から実行コマンドの逐次支援は受けていない。historical schemaの参照とhuman理由の省略に対してJSONを選ぶ負荷はあった。inspectのTrace complete=true/unresolved=trueは「走査対象runは扱ったが結論は未確認」という別軸であり、completeだけをall-clearと読まない必要があった。後者は今回のLLMには区別できたが、初見の人間に伝わるかは未検証。

全S1〜6をこの小さなsynthetic fixtureで操作したという結果までを主張する。以下はこの試験では未実施:

- 初見の実利用者試験、複数人の行動/時間/誤解率、一般の長大リポジトリでの運用。
- format6の独立操作、2MiB/バイナリ/計算量上限/故障注入/並行競合の独立操作。自動テストの証拠が別にあっても、この操作試験の実績には算入しない。
- cursorによるページ継続、全てのerror branch、全てのhistorical/detached/transitive graph。
- 既存の本番利用者データ、常駐/定期通知、製品の意味的正しさの一般証明。

新しい受入基準やN2/GUIの義務は追加していない。所有者の製品受入、release/install、main側のSeal判断を行った記録ではない。

## 試験後followupの限定的な独立確認

作者が試験後に追加した2差分だけを、mainのコードを変更せず確認した。旧実装レビュー報告のexact snapshotは書き換えていない。

- `internal/cli/trace.go` 現SHA-256 `a64b711008948d2d78fbddbe71bc7e437d721bd49571fbf4b80f293ddbcefbf3`。新しいR: module commentの1文（空行込み）をメモリ上で除くと、旧独立実装レビューのSHA-256 `0fff32dda462f7f4a63d388e7c2803f84de25cd25fe59420736f3f8f10d95410` と一致した。よってこの差分は責務comment追加だけで、実行コード本文は不変。
- `docs/operations.md` 現SHA-256 `105af2049604b8b70ef4f24bc2ca29eba114e2246a98842f9a2af7fed1ece20a`。human理由の省略と完全な `review show --format json` を案内する追加1文をメモリ上で除くと、試験入口で固定した旧SHA-256 `a30faaff5f620ac01587ff7eb66275af0237e30ab5a1a41929a0e76f0c6de9ff` と一致した。試験で実際に用いたJSON読取り手段を明示する補足で、通知範囲や受入意味を変えていない。

この確認にmain repoのGit操作、コード/PLAN/Seal変更は用いていない。統合担当が実施した全チェック再実行を、この役割試験担当が独立に実行したという主張はしない。
