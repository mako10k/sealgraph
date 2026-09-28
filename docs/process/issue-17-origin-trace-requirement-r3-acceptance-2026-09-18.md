# Issue #17 R3-c1 要件承認記録 — 2026-09-18

- 状態：Accepted requirement revision（所有者による要件レビュー Step 4 の判断）
- 対象：[R3-c1 要件差分候補](issue-17-origin-trace-requirement-r3-candidate-2026-09-18.md)全文
- 対象 SHA-256：`1179004a32f3e105823bcdd714e736def52ccb5716382fa5a5f40263b649d3c3`
- 基底：Accepted [R1](issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md) と [R2-c2](issue-17-origin-trace-requirement-r2-acceptance-2026-09-17.md)
- Step 2：[初回所有者レビュー](issue-17-origin-trace-requirement-r3-first-review-route-2026-09-18.md)で `REVIEW` を選択
- Step 3：[独立レビュー](issue-17-origin-trace-requirement-r3-independent-review-2026-09-18.md)は `completed`。確定的矛盾なし。公開ページングの具体方法と AC16 の表現に未決・解釈上の注意を残す

## 所有者の判断

R3-c1 の日本語全文、独立レビュー報告、R2 との差分、残る設計事項を提示した。Step 4 の質問では、AC16 の「最初」を「探索順で最初に発見した一件」と扱い、ページングの公開詳細を後続設計に残すことも明示した。所有者は次のように回答した。

> `ACCEPT`

これにより、上記 SHA-256 の R3-c1 全文を Issue #17 の Accepted 要件改訂とする。候補本文に残る `Proposed` と Step 1 の表記は承認前の exact snapshot の履歴として保持し、現在の採用状態は本記録で示す。受入対象の本文 bytes は変更しない。

## 有効な要件と境界

Issue #17 の有効要件は、Accepted R1 に Accepted R2-c2 §2～§3 の置換・追加を適用し、さらに本 R3-c1 の限定差分を適用したもの。R2 R7 の原文残存判定は、現在ファイル F に各 External run の exact bytes P が連続部分列として一件でもあれば残存、全域を確認して無ければ完全一致なし、確認できなければ未完了のまま維持する。一件で判定を確定でき、全位置列挙は判定の条件ではない。

記録した `source_start` は Seal 時点の元版 S に対して検証された一件の binding 位置であり、探索の初期ヒントでもある。S、安定観測した F、または両方の全一致開始位置は、その操作で必要な時に exact bytes から導出する。位置を OriginMap や Seal に全件保存しない。探索で選んだ一件は全集合でも唯一の歴史的箇所でもない。AC16 の「最初」は、採用時の質問のとおり探索順で最初に発見した一致を指し、byte offset が最小の一致を保証しない。

Seal 時点の元ファイル全文 S は、Seal 範囲外も含め、後日の対象ファイル変更・削除にかかわらず immutable データから exact bytes で復元できることを要件とする。位置一覧または位置に基づく提案には既定の返却件数上限とページングを設け、異なる固定版の位置を同一一覧へ混ぜない。R3 AC16～AC21 を追加し、R1/R2 の他の条項と AC1～AC15 は R3-c1 に明示した補足以外を維持する。

Cause Link の whole-Seal 対象、既存 STALE、自身・上流・下流の独立観測、意味論非監査、原文不存在と変更後範囲推定の分離は維持する。位置の byte 一致は実編集履歴・原意・唯一の由来を証明しない。

## 下流への影響と残る判断

Accepted [ADR 0033](adr-0033-0035-origin-trace-acceptance-2026-09-17.md) は `SourceSnapshot.content` の元ファイル全文 Blob と typed closure を既に定めている。要件採用だけで実装・dump/load による復元成功を確認したことにはならない。Accepted [ADR 0039](issue-17-adr-0039-acceptance-2026-09-18.md) は本要件と同方向の後継設計判断であり、同 ADR が残した公開契約と ADR 0036～0038・計画 P2 との整合は[下流整合案](issue-17-origin-trace-r3-downstream-alignment-proposal-2026-09-18.md)を入力として別途確定する。

位置一覧・提案の公開操作、各操作が S/F/両方を使う条件、ページ識別子・固定版の継続方法・終了や中断の表現、既定上限の数値、性能条件、実装・移行・提供時期は未決。これらを独立レビューや現行実装から要件の新しい受入条件へ昇格させない。

今回の `ACCEPT` は R3-c1 の exact 要件を採用する判断である。R3 要件の Seal、後継 ADR と計画の採用、Issue #17 runtime の実装・検証・公開、Git の push は別の作業であり、本記録には実施済みの証拠がない。利用者が今使える Issue #17 の新機能は増えていない。今回確定したのは後続設計の上位要件であり、提供時期と費用は不明である。
