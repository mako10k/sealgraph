# Issue18 観測区分の要件候補 R1

Status: Proposed。2026-11-04。UX R1の成果受入後の次候補。Issue18全体の採択ではなく、観測表示の意味に限定した案。現行Accepted規範は変更していない。

## 目的と採択提案

利用者・LLM・監査者が「観測されたRevision関係」「active revision集合への所属」「依存の見直し状態」「Candidateの存在/差分」「各版のdraft」「workfile/baseline」を別々の事実として説明できること。

提案する選択は、既存STALE集合、frontier、refs-only、通常seal admissionを維持し、理由を読める観測を追加する方針。Issue18の別案であるSTALE membership変更や軽量statusへの置換は採用しない。この方針自体が今回の所有者判断対象であり、既定事実として実装しない。

## 範囲内の観測要件

1. 選択Sealについて、現在の観測集合に含まれるsuccessor/revision assertionの有無と、そのexact observer/target/previousを読めること。「successorを観測していない」を「successorは存在しない」へ変えない。
2. current headsを起点にRevision edgesだけで定まるactive集合への所属を、1とは別に示すこと。Causeから到達できることをactive所属と混同しない。historical/detachedのどちらかを現行証拠から確定できない場合は、その限界を示すこと。
3. 各見直し対象のexact Causeと経路に対し、active non-leafが根拠なのか、active集合外が根拠なのかを区別できること。direct/transitiveと理由を混ぜず、複数理由/複数経路があり得ることを保持すること。
4. Candidateの存在、比較対象として明示したHEAD版とCandidateの一致/差分、publication expectationの関係を別に示すこと。Candidateが存在するだけで内容/根拠がHEADと異なるとは説明しない。
5. Candidate draftとHEAD draftは各baselineに紐付けて読めること。片方が存在しない状態はfalseへ潰さず、存在なしとして区別すること。
6. workfile比較の対象はCandidate/HEAD/NONEを明示し、stale/revision/draft/承認を推論しないこと。R1のsource/Trace/historical reviewの分離を維持すること。
7. 対象はhuman status/stale/graph/fsckと対応する機械出力の観測意味。全既存公開スキーマ/exit codes/bytes-only protocol/セレクタを保護し、変更が必要な新しい出力契約は版と利用方法を明示して設計すること。新しいJSON fieldや具体的綴りをこの要件段階で固定しない。
8. 保存されたSeals/Causesのimmutable性と、担当者の個別判断/個別更新/1 REFずつの発行を維持すること。自動repair/relink/resealを提供しない。

## 受入シナリオ案

| ケース | 区別して確認する結果 |
| --- | --- |
| exact old Causeがあり、別observerがnew target previous oldを主張 | 観測されたsuccessorの証拠、active non-leaf由来の見直し理由、exact Cause経路 |
| HEADを移動し、旧targetへのsuccessor assertionが観測されない | 観測なしという限界とactive集合外を分け、successor-based staleだったとは言わない |
| branching sibling leaves | 一方をpreferred/latestへ選ばず、観測関係と集合所属を読める |
| CandidateがHEADと同じ/異なる/HEADなし | 存在と比較を別に表示し、比較baselineを明示 |
| HEADだけdraft/Candidateだけdraft/両方/片方なし | 各版のdraftと存在なしを分離 |
| workfileだけ変化/欠落/未束縛、歴史版review applicable/失効 | provenanceの集合/Revision事実と別軸に保持 |
| 同じfixtureで現行stable出力を比較 | stale集合・frontier・refs-only・通常admissionの意味が変わらない |

具体的な短縮ラベル（約6文字等）、画面/列配置、内部API、アルゴリズム、追加スキーマ/モードの選択は基本・詳細設計へ渡す。速度改善はこの候補の目的ではなく、速くなったという受入主張はしない。

## 明示的な対象外

separate draft pointer、再帰的draft生成、multi-pointer atomic publication、draft-inclusive prospective graph、promotion、軽量statusへの置換、Revision/hyperedge model変更、Issue17の追加fragment semanticsは含めない。これらはIssue本文の探索案で、現行要件や個別Seal規則を置き換える権限ではない。実装するには別の所有者判断が必要。

## 権威・後続判断

根拠はIssue18の公開本文と現行requirements §3–4、Accepted ADR0023/0027/0043/0044、UX R1の範囲。既存権威は維持する。この候補の採否が決まるまで計画終点・既存実装・公開出力・Sealを変更しない。

未知: 観測追加をどのpublic outputへどう配分するかは設計事項。既存STALE membershipを維持する提案が所有者の目的に合うかは要件判断。初見実利用者への表示検証と性能の一般保証は未実施である。
