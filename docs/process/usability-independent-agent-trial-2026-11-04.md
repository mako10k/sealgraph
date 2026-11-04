# 独立Agent操作試験（2026-11-04）

## 結果と観測範囲

独立した操作担当Agentが、捨てられる試験repoで六場面を実操作した。通常案内から初回登録、workfile点検、Revision/active観測、新版採用、旧版保持と未確認の保留、原文監査まで到達。実行を妨げる製品不具合はこの試行では観測しなかった。人間の初見試験ではなく、実利用者の成功率・理解・必要時間の証拠にはならない。既存の受入を変更する試験でもない。

操作担当はコード・既存テスト・過去試験結果を読まず、AGENTS.md、準備資料、docs/operations.md、README.md、CLI helpから開始した。親Agentから解法の援助なし。親からの版・書込範囲・文書編集中の連絡は進行管理情報として受領した。場面5ではoperations S4が参照するdocs/storage-format.md §15を追加参照した。helpと通常操作ガイドだけではreview JSONの完全な構造を書けず、仕様参照が必要だった点を別記する。ソース内部調査はなし。

## 環境、時間、再現資料

- 開始実測: 2026-11-04T12:48:35.521413315+09:00。操作後チェック時刻: 12:51:34.438004238+09:00（約179秒）。報告作成時間は別。各場面境界の個別時計は取らなかったため、場面別秒数は未測定。
- Linux 6.18.33.2-microsoft-standard-WSL2 x86_64、Go go1.27.3 linux/amd64、非TTY実行。
- HEAD: `2467d3a8c0ad34785afc9a6932ede75744701d89`。親報告では受入runtime b3c4fda/eb2798fと同じ。自分の確認では`git diff --name-only -- '*.go' go.mod go.sum`は空。PLAN.pertと準備資料は親が編集中でdirty。これらをruntime変更と混同しない。
- 試験場所: `/tmp/sealgraph-independent-trial-cSRXXA`。ビルドはrepoから`go build -o /tmp/sealgraph-independent-trial-cSRXXA/sealgraph ./cmd/sealgraph`。SHA256: `dd8f1f6e07ea29abf7ef2e7c21a0872509024472b09a0ba302deb6a94bb28c88`。
- 試験repoは初期format5、その後明示的にformat7へmigration。実repoの`.sealgraph`・コード・Git状態・remoteは変更していない。この報告以外のrepo書込なし。
- 判断の逐次メモ: 試験場所の`judgments.txt`。保存証拠: `final-observation.json`, `final-inspection.json`, `pending-review.json`, `historical-head-review.json`, `pending-candidate.json`, `retained-seal.json`, `quote-seal.json`, `source-failure.txt`, `occurrences.json`, `saved-original.txt`, `retain-review.json`。一時場所のため恒久保存は保証しない。操作時全出力はこの実行のツール記録にある。全stdoutを最初からファイル記録したわけではない。
- worktimectl agentは開始と操作後に`No work-time action is required.`、planned_end=20:00、required_actions=[]を確認。共有作業日を変更していない。

## 正確な版

以下のP1等は報告内の読みやすさのための略号であり、操作時はfull IDを手入力した。

| 略号 | SealID | 内容 |
| --- | --- | --- |
| P1 | 1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8 | `The room has one blue chair.\n` root |
| D1 | e7d90aba40ab0b1676ef9b34fea9f1f1d283b078f5f454fea8b15231fddf36dc | 窓際にblue chairを置く。Cause=P1 |
| P2 | d1d19f1f6f5db8a4f518901ba6cb723995021446a08a36b7c73a055858a98091 | `The room has one blue chair and one desk.\n` root |
| D2 | c0e2f7450b88735259015f1ab5d92823e2d16f8b02275117921108bd9002ef2b | 扉脇にblue chairを置く。Cause=P2、previous=P1 |
| R1 | 32dae0c39a9f7330e7c78f9cc5d3a2b166f7e70ffba77bf1cc32eee0df4fb3b2 | 窓際配置、Cause=P1、historical-review付き |
| Q1 | d54113ace734d550587711285d3e30821375c61f39e64b5c22d4156ee464fbb8 | `the blue chair`の14 bytes、root、External Trace |

## 場面ごとの実行記録

### 1 初回登録 — 実行・到達

実行順は`./sealgraph help workflows`、`help observe`、`init`。premise.txtとdesign.txtを実際に読み、`add premise --root --clear-cause-links --content-file premise.txt --bind-source`、`candidate show premise --format human`、`candidate show premise --raw-content`。29 bytes・Causeなし・境界を確認して個別に`seal premise`、`show premise --format json`/`--raw-content`でP1を読戻した。

次にexact P1とdesign原本を読んで、「chairの存在だけを配置の根拠とし、previousは未知」という個別判断を記録。`add design --non-root --content-file design.txt --bind-source --target @1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8 --no-previous -m 'The exact premise establishes the blue chair that this placement uses.'`を実行。Candidate show/compare/rawで確認し、別の操作で`seal design`。D1本文・Cause・IDを読戻した。

操作担当の説明: REF premise/designは移動する名前、P1/D1は不変の特定版。rootは試験のroom記述をここから先は追わない境界であり、真実性や承認を意味しない。詰まり・再試行・援助なし。

### 2 workfile点検 — 実行・到達

design.txtのみを`Place the blue chair beside the door.\n`へ変更。`inspect design --format human`が`NO_CANDIDATE`, `WORKFILE_DIFFERS_FROM_HEAD`, `baseline: HEAD`, `stale self=false direct=0 transitive=0`を出した。次に`add design`でrefresh後、`WORKFILE_MATCHES_CANDIDATE`, `baseline: CANDIDATE`, `UNSEALED`になった。原本差分とprovenance staleは別の観測であり、sealがファイルを再読するわけではないと説明できた。バックグラウンド通知は試しておらず、guide/helpのon-demand契約として理解した。詰まりなし。

### 3 Revisionとactive観測 — 実行・到達（一枝）

premise.txtにdeskを加え、`add premise`後、原本・Candidate・`candidate compare premise --format human`・旧原文を読んだ。chair維持とdesk追加の影響を判断し、単独の`seal premise`でP2をpublish、読戻した。

`observe @P1 --format human`（実行時full ID）ではP1=`active=false active_leaf=false Observed successors=[]`。D1がP1をCause参照していてもactiveにはならず、HEADがP2になっただけではrevision assertionにならない。後続が未観測であり不存在の証明ではない。

場面4のD2をpublish後の同じobserveではP1=`active=true active_leaf=false Observed successors=[P2]`となり、observer=D2、target=P2、previous=P1の正確な根拠が表示された。最後に`observe --inventory --format json`を保存した。outside-activeからhistorical/detachedを断定できない旨も出力に明示された。複数revision leaf、経路上限超過、detachedの個別fixtureは未実施。一枝の成功からそれらを検証済みとしない。

### 4 新版採用 — 実行・到達

`show design --raw-content`、`candidate show design --format human`/`--raw-content`、P1/P2原文、`compare @P1 @P2 --text`、`impact @P1 --format human`を読んだ。差分はchair維持とdesk追加。Candidateの扉配置はfixtureとして自分で選び、desk追加によってchair存在が否定されないと判断。

個別に`link design --target @d1d19f1f6f5db8a4f518901ba6cb723995021446a08a36b7c73a055858a98091 --previous @1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8 -m 'Desk addition preserves the blue chair; this authored room update succeeds P1 and supports door placement.'`。Candidateの両Causeを読戻し、P2にprevious P1があることを確認した上で、別の判断・操作として`unlink design --target @1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8`。

最終Candidate show JSON/raw、candidate compare、status JSONで本文・Cause・baseline・root=false・draft=falseを確認し、単独`seal design`。D2本文とCauseを読戻した。判断・Cause操作・Seal発行はそれぞれ一回ずつで、ループ・発行script・自動retryなし。`status`のstaleはHEAD D1を示し得る一方CandidateはP2なので、CandidateとHEADを分けて読む必要がある。詰まりなし。

### 5 旧版保持と保留 — 実行・到達、schema参照の負荷あり

窓配置のretained.txtとexact P1を読み直し、別REF retainedのCandidateを`add retained --non-root --content-file retained.txt --bind-source --target @P1 --no-previous -m ...`で準備。自分の判断は「deskの追加に依存しないchair存在だけの文書なのでP1を意図的に保持」。P1/P2原文・差分・impact、Candidate全文とCauseを再読した。P1はrootなのでtransitive Causeはなし。

`review prepare retained --format json`と`help review set`から、subject/other_heads/targetsは得られたが、作成するrecordの全JSON構造がhelpにはない。operations S4からstorage-format §15へ進み、schema、entriesと必須項目を読んで、対象一つのretain-review.jsonを自分で記述。これはコード/テスト調査ではないが、「通常案内＋helpだけ」で閉じなかった追加参照として数える。review prepareを自動的なretain判断へ変換していない。

`review set retained --file retain-review.json`、`review show retained --candidate --format human`、Candidate show/rawを読んで個別に`seal retained --review-historical`。R1を読戻し、`inspect retained`が`direct=1`, `RECORDED applicable=true`を表示。historical reviewはstaleを消さず、実行者のclaimにとどまる。

保留はretained.txtを`Place the blue chair near the window if the floor supports it.\n`へ変更、`add retained`後にCandidate原文を読んで判断した。floor/loadの情報はP1/P2にないので、新しい条件の支持は未確認。Cause変更・SealをせずCandidateを残した。inspectはCandidate review=`applicable=false REVIEW_SUBJECT_CHANGED`、HEAD R1 claim=`applicable=true`を別表示。後のformat7 migration/new REF追加後は他HEAD/subject条件も変化しているため、当初の適用性を現在へ持ち越さない。最終pending-review.jsonにapplicable=falseを保存。未確認をcleanにする操作なし。

### 6 原文監査 — 実行・到達

Traceはformat5では`UNSUPPORTED_FORMAT`だった。`.sealgraph/config`を読み、`help migrate repository`に従って試験repoだけで`migrate repository --from 5 --to 7`を明示実行。receiptはMIGRATED、retained_seals_v5=5、retained_candidates_v5=1。歴史SealにTraceを自動補完していない。

`source.txt`=`Source note: the blue chair is available.\n`（42 bytes）、`quote.txt`=`the blue chair`（14 bytes）を作成。`add quote --root --clear-cause-links --content-file quote.txt --bind-source`後、`help trace set`に従い`trace set quote --source-file source.txt --source-key room-note --content-file quote.txt`。source全文・Candidate全文・Traceを読んだ。External runはstart=13 length=14、snapshot=`5f9559131b98d6b5e8ea08fb7f5c7870f6276da9b2f3af8d1f0e5fe437be6b47`。引用の境界をrootとしCauseなし、と個別判断して`seal quote`、Q1を読戻した。

`trace source bind room-note --file source.txt`後、exact Q1への`trace read --seal @Q1 --run-index 0 --view snapshot --format human`と`--view current`を実行。両方42 bytes、truncated=false/quote_truncated=false。次にsourceを`Source note: the blue chair is unavailable.\n`（44 bytes）へ変更。currentはPRESENTのまま、current Blob=`2e2f0f955ce44433b92bd9ef796bc954e0f926a156dead56d6cd1551df050a13`、snapshot Blob=`83199dd076cf65f923bc1ebf434bb2b0e47bdbec951214f98e5bd7554f06584f`と分離され、本文contextからavailable→unavailableを読めた。

`trace compare --seal @Q1 --max-graph-visits 100 --format human`はEXACT_MATCH/PRESENT、graph complete=true visits=10/100。引用bytesの存在は確認できるが、椅子を利用可能とする意味的支持はcurrentでは成立しない。この例では引用範囲にavailableを含めなかったためであり、PRESENTを意味的支持へ読み替えてはならない。

source.txtをsource-unbound-fixture.txtにrenameして同じcurrent readを行うと`INCOMPLETE`, `Unconfirmed: SOURCE_READ_FAILED`。これは期待した失敗観測で、原因を直すretryはしていない。snapshot raw-sourceはsaved-original.txtへ明示export、snapshot occurrencesはstart=13 length=14、has_more=false。D2へのtrace showはorigin_map=nullで、過去の原文根拠は再構築できない。unconfirmed/NO_TRACEを監査済みにしない。

## 困惑、援助、改善候補、未実施

- 実行を遮るblocking defectは未観測。仕様外のCLI呼出しretryもなし。後処理で`git diff`を一度試験場所で実行してGit repoではないusageが出たが、環境確認コマンドのcwd誤りで、SealGraphの困惑とは別。実repoで再確認した。
- 一番の負荷はreview recordの手書き。help review setからstorage-format §15への具体的リンクと、判断欄を明確に空欄にした非実行のschema例があると初回の探索を減らせそう。定型retain理由の生成・自動判断・自動Sealは提案しない。これは診断案で、要件採択・実装はしていない。
- human出力のreview理由は省略されるためfull JSONが必要な場合があった。operationsにはその説明があり、自力で辿れた。省略を完全reviewと誤解しないことが必要。
- fixtureは短い一枝、4 REF、1 External runで、複数leaf選択、長大文書、paging継続、graph-budget不足、binary source、深いtransitive closureの操作は未実施。
- 本Agentは一般のCLI/技術文書を理解する既経験Agentであり、人間初心者の代理指標として秒数や「迷わなかった」を一般化できない。人間試験、製品閾値、追加要件の承認、実装・merge・公開は未実施。

最終読戻し・handoff確認実測: 2026-11-04T12:53:15.279795811+09:00。開始から約280秒（報告作成・読戻し込み）。終了前worktimectlもrequired_actions=[]。この試験の計時密度は場面単位の人間試験測定に転用できない。
