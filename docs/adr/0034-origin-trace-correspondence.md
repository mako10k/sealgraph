# ADR 0034: Origin Trace の全最適対応と対応宣言

- Status: Proposed（ADR 0032 D1 の比較 companion）
- Date: 2026-09-17
- Decision Owner: Operator
- Language: 日本語。レビュー対象は本ファイル全体。
- Requirement Authority: Accepted Issue #17 R1。R1 の R7～R10、AC4～AC5 を中心に、他の要件との整合を維持する。
- Governing design: [ADR 0032 D1](0032-content-origin-trace-and-directional-observation.md)。対象 bytes の SHA-256 は `072314438483af19d2eb9b84b36d6a5b15d1ed358068f1849da473f34a3c38d0`。
- Requirement record: [Issue #17 R1 acceptance](../process/issue-17-origin-trace-requirement-r1-acceptance-2026-09-17.md)。R1 の候補本文は本作業で変更しない。
- Execution authority: 比較契約の提案のみ。公開 CLI、公開 machine schema、canonical storage、実装、移行、commit/push は本 ADR の権限外。

## 1. Context

ADR 0032 は、Origin Trace の現在比較について、対象範囲の対応が全体の bytes と最小挿入・削除モデルに整合し、任意の一つの diff の tie-break を一意な対応とみなさないことを提案した。R1 は、前方挿入による位置移動を内容差異とせず、改変・削除と対応不能を区別することを要求する（R7～R10、AC4～AC5）。

この companion は、比較器が受け取る論理入力、最適解が合意する対応の定義、範囲と編集の正規化、曖昧性・宣言・予算終了の境界、理由と出力の最小内容を定める提案である。これは実編集履歴、意味上の同一性、削除や移動の現実世界での発生を証明しない。

## 2. Decision（提案）

### 2.1 比較モデル

二つの exact byte 列 `old` と `current` に対し、保持・挿入・削除だけを許すグローバル alignment を求める。保持は同一 byte の一対一対応、挿入と削除はそれぞれコスト 1、保持はコスト 0 とする。置換は独立した操作ではなく、削除と挿入の組合せとして表す。比較は文字、行、改行、文字コード正規化、意味解析を使用しない。

最小総コストを `d` とし、`d` を達成する全ての最適経路を列挙する必要はない。実装は最適経路 DAG または同値の動的計画表から、対象範囲に関係する全最適経路の合意を判定できなければならない。単一経路、任意の tie-break、最初に見つかった経路だけで対応を確定してはならない。

SourceSnapshot.content の BlobID と current 全文の BlobID が同じ場合は、alignment を実行せず同一 bytes を同位置で対応させる。この fast path は MATCHES の根拠を `IDENTICAL_SNAPSHOT` として記録する。source の読み取り、binding、Blob identity の取得は比較器の外側で行う。読み取り失敗や観測 identity の不整合は比較器の alignment 結果ではなく入力観測エラーとして上位へ返し、比較器が推測で埋めてはならない。

### 2.2 対象範囲の対応と正規化された範囲像

元の対象範囲を半開 byte 区間 `O=[a,b)` とする。各最適経路から、次の論理的な範囲像を作る。

```text
RangeImage = {
  surviving_pairs: [(old_offset, current_offset)],
  current_ranges: [R1, R2, ...],
  internal_edits: [insert/delete regions],
  left_boundary, right_boundary,
  ambiguity: ...
}
```

上のラベルはこの ADR 内の論理用語であり、公開 schema 名、CLI 名、保存形式名ではない。`surviving_pairs` は O 内の保持された bytes とその current 位置を、old 順・current 順の両方が保たれる形で表す。隣接する current 位置の保持列は一つの current 半開範囲にまとめる。保持列が分断される場合は複数の current 範囲を宣言できる。

各最適経路を、保持 edge（同一 byte の対応）と old/current の先頭・末尾 sentinel の間にある最大 edit block へ正規化する。block は `(old=[x,y), current=[u,v))` で表し、その間に保持 edge を含めない。純粋な insertion は `x=y`、純粋な deletion は `u=v` である。削除と挿入による置換は、同じ block の old 部分と current 部分をともに持つ一つの edit block とする。隣接する insertion/deletion の実行順だけが異なる経路は、この block 表現で同一視する。

対象 O に対する block の分類は次の通りである。

- `x=y` の pure insertion が `x=a` または `x=b` にある場合は、O 外の左／右 boundary insertion とする。`a<x<b` の pure insertion は O 内の internal block とする。
- old 部分が `a<=x<y<=b` の block は O 内の internal block とし、replacement も deletion も含めて current 範囲像へ反映する。
- `x<a<y`、`x<b<y`、または old 部分が O の一方の境界をまたぐ block は boundary-straddling とし、O の自動対応を未合意とする。O に surviving byte がない場合でも、この規則を適用する。
- O 外の deletion、O の境界に接する pure insertion、O の外側にある block は範囲結果へ含めない。
- `current_ranges` は O 内の surviving bytes と internal block の current 部分を current 順に表す。複数の disjoint block は複数範囲として保持し、単一の包絡範囲へ潰さない。

全最適経路を O の像とその左右の境界 anchor に投影し、その範囲内の保持 edge、最大 edit block（old/current 範囲と種別）、boundary-straddling の有無、current 範囲列が同一なら対応は合意している。O の像や境界に影響しない外部領域だけの競合は、O の合意を妨げない。挿入・削除の実行順だけが違い、上記の正規化後に同じ block 列になる経路は別の対応とは数えない。いずれかが異なる場合は、その範囲を自動確定しない。

対応が合意していて、`current_ranges` を宣言順ではなく byte 順に正規化した連結 bytes が元範囲の bytes と等しければ MATCHES とする。位置が変わっただけの場合は MATCHES に位置変化を付随させる。合意した internal 編集により連結 bytes が異なれば DIFFERS とする。合意した全削除は、左右境界が合意し、O に対応する current bytes が存在しないときに限り DIFFERS とする。

### 2.3 繰返し・移動・複製・分割結合の抑制

同じ byte 列が current の複数位置にあることは、由来の移動・複製の証拠ではない。最適経路間で surviving pairs または current 範囲像が競合する場合、検索一致の数にかかわらず UNRESOLVED とする。削除と挿入で表現できる移動、複製、分割・結合が同じ最小コストで競合する場合も UNRESOLVED とする。

全削除は検索失敗では確定しない。対象範囲の両側にある保持 bytes（または old/current の先頭・末尾 sentinel）が全最適経路で同じ位置関係を持ち、対象範囲に対応する current 範囲がない場合だけ、合意した full-deletion block として DIFFERS を返す。置換 block の current 部分が存在する場合は削除ではなく internal edit として扱う。対象範囲と同じ完全な old bytes が current 全文の別位置に存在する場合は、最小経路が一つに見えても自動削除を抑制し、REPEATED_TARGET として UNRESOLVED を返す。境界が競合する、反復列が別の対応を許す、内部編集か境界編集かが変わる場合も UNRESOLVED とする。

### 2.4 明示的な対応宣言

自動対応が UNRESOLVED の場合、利用者は固定した old snapshot と current 観測 bytes に対し、元範囲、current の一つ以上の範囲、または削除を明示的に宣言できる。宣言の論理フィールドは `schema`, `source_snapshot`, `source_start`, `length`, `current_blob`, `current_ranges`, `deleted`, `reason`, `declared_at` とする（公開 wire member の採否と型は CLI companion の責務）。`source_start` と `length` は元 snapshot 内の正の長さの範囲を示し、`current_ranges` は `{start,length}` の宣言順配列で、各長さは正、同一宣言内で重複しない。隣接範囲は宣言上統合しない。`deleted=true` は `current_ranges` が空の場合に限り、空の `current_ranges` は `deleted=true` の場合に限る。

current 観測 identity が一致する宣言だけを候補にし、不一致の `current_blob` の宣言はその観測では無視する。matching な宣言が一つだけなら、自動対応より優先して採用する。同一の固定入力・同一の元範囲・同一の current 範囲群（または同一削除）を持つ宣言は、reason や時刻が違っても同じ対応を補強する。対応範囲群が異なる宣言だけを相反宣言とする。

宣言は、snapshot identity、元範囲、current 観測 identity、current 範囲群または削除、宣言理由、宣言時点を含む論理入力とする。ID と範囲の包含、各範囲の bytes の存在、宣言された連結 bytes の比較は検証する。宣言は利用者の対応についての入力であり、意味論、実編集履歴、現実の削除を自動認定しない。

同一の固定入力に相反する matching 宣言が存在する場合、後勝ちや自動マージを行わず `UNRESOLVED` / `CONTRADICTORY_DECLARATIONS` として選択を要求する。宣言の current 観測 identity が後から変わった場合は、その宣言を新しい観測へ再利用せず、matching identity の宣言を新たに入力する。宣言は canonical provenance、Cause、Assessment、STALE を変更しない。

### 2.5 論理入力・出力と理由

比較器の論理入力は次の集合である。

| 入力 | 内容 |
|---|---|
| 比較版 | old snapshot の exact bytes と identity、current 観測の exact bytes と identity |
| 対象 | old の一つの半開 byte 範囲。Origin Trace の External run から得る |
| 方針 | 最小挿入・削除、全最適解合意、範囲境界、budget の明示値。具体的な wire 名は未決 |
| 宣言 | 任意。固定した二版に対する明示対応と理由 |

出力は少なくとも、比較版 identity、対象元範囲、正規化された current 範囲像、内部編集・境界編集、判定（MATCHES / DIFFERS / UNRESOLVED）、対応根拠、理由、位置変化、調査完了状態を論理的に持つ。これは将来の public schema の member 名や順序を決めない。

理由は次の論理 enum のいずれか一つとする。公開 wire 名への変換は別 companion の責務だが、意味を変えずこの閉じた語彙を対応させる。

`IDENTICAL_SNAPSHOT`, `ALIGNMENT_AGREES`, `INTERNAL_EDIT`, `FULL_DELETION`, `ALIGNMENT_AMBIGUOUS`, `REPEATED_TARGET`, `MOVE_OR_DUPLICATION_AMBIGUOUS`, `BOUNDARY_STRADDLES`, `DECLARED_CORRESPONDENCE`, `CONTRADICTORY_DECLARATIONS`, `INPUT_INVALID`, `SOURCE_READ_FAILED`, `BUDGET_EXCEEDED`。

範囲結果の論理 field list は `baseline_old_identity`, `baseline_current_identity`, `old_range`, `current_ranges`, `surviving_pairs`, `edit_blocks`, `position_changed`, `result`, `reason`, `evidence_kind`, `examined` とする。`edit_blocks` は各 block の old/current 半開範囲と pure insertion/deletion/replacement の種別を含む。`result` は調査済みなら MATCHES / DIFFERS / UNRESOLVED、未調査なら null、`examined` は調査完了または NOT_EXAMINED を表す別軸であり、NOT_EXAMINED は result の値ではない。source I/O や invalid input はこの範囲結果を捏造せず、対応する reason と上位エラーを返す。

### 2.6 予算終了と計算可能性

alignment の計算は有限の byte 列と有限の最適経路 DAG に対して定義できるが、入力サイズに対する時間・メモリ、最適経路数、圧縮やストリーミングの実装特性をこの ADR で保証しない。バイナリ列にも定義できる一方、許容対象サイズ・プラットフォーム・資源上限・測定条件は別の platform/budget decision が必要である。

合意判定または必要な境界分析を設定された予算内に完了できない場合、近似経路や単一経路を確定結果に昇格せず、調査状態を NOT_EXAMINED とし、budget 終了と未完了の対象を理由に出力する。入力読み取り失敗や構造不整合は別理由として UNRESOLVED またはエラーにし、予算終了へ置換しない。

比較全体に適用する alignment-cell 予算は、old の byte prefix 長 `i` と current の byte prefix 長 `j` の組 `(i,j)` を一つの DP 格子 cell と数える。境界を含む `(len(old)+1) × (len(current)+1)` が上限を超える場合、または実装が必要な cell をその上限まで検査できない場合は、最適経路の合意を返さず NOT_EXAMINED とする。最適経路 DAG の同値辺を調べる追加計算も、その cell の検査に含める。alignment-wide に共有した表を複数 run へ投影しても、各 run の再計算を理由に合意済み結果を未調査へ戻してはならない。これは計算量の acceptance threshold や性能保証ではなく、呼出しごとに利用者が選ぶ安全な上限である。CLI の引数名、正整数の wire 表現、既定値を持つかどうか、graph traversal の別予算、停止時の transport は CLI/output/storage companion の決定対象である。

## 3. Deterministic examples and acceptance mapping

以下は比較契約の実装・検証に要求する小さな決定的例である。バイトは説明上 ASCII で書くが、契約は任意の byte 値に適用する。

| 例 | old / current / 対象 | 期待 | AC |
|---|---|---|---|
| 前方挿入 | `abcXdef` / `QQabcXdef` / `[0,7)` | MATCHES、current 範囲 `[2,9)`、位置変化のみ | AC4 |
| 内部改変 | `abcXdef` / `abcYdef` / `[0,7)` | DIFFERS、保持列と internal 差異を示す | AC5 |
| 一意な削除 | `abcXdef` / `abcdef` / `[3,4)` | DIFFERS、左右境界が合意した全削除 | AC5 |
| 反復列 | `aa` / `a` / `[0,2)` | 二つの同コスト対応があり、単一の `a` の由来を選べないため UNRESOLVED / ALIGNMENT_AMBIGUOUS | AC5 |
| 一意な内部挿入 | `ab` / `aZb` / `[0,2)` | DIFFERS / INTERNAL_EDIT、current `[0,3)`。挿入を含めて比較する | AC5 |
| 範囲外の競合 | `aaZ` / `aZ` / `[2,3)` | MATCHES / ALIGNMENT_AGREES、current `[1,2)`。範囲外の a の対応競合は持ち込まない | AC4/AC5 |
| 置換（対象に surviving byte なし） | `abcXdef` / `abcYdef` / `[3,4)` | old `[3,4)` と current `[3,4)` の replacement block は O 内なので DIFFERS / INTERNAL_EDIT。削除と扱わない | AC5 |
| 境界をまたぐ edit block | `abc` / `aX` / `[0,2)` | replacement block old `[1,3)` が O の右境界をまたぐため UNRESOLVED / BOUNDARY_STRADDLES | AC5 |
| 全削除 | `abcXdef` / `abcdef` / `[3,4)` | old `[3,4)` と current 空の全削除 block、左右境界が合意するため DIFFERS / FULL_DELETION | AC5 |
| 明示的分割宣言 | old `[0,6)`、current `[0,2)` と `[5,9)` | 宣言順の二範囲を保持し、連結 bytes と理由を検証。意味の正しさは判定しない | AC5 |

予算を極小にした同一例では、全最適解の合意前に停止し NOT_EXAMINED と終了理由を返す。読み取り不能の例では UNRESOLVED または構造エラーとし、NOT_EXAMINED と混同しない。これらの例は AC4～AC5 の expected match/diff/unresolved を固定するための候補であり、公開テスト名・schema 名を決めるものではない。

## 4. Claims, Evidence, and Actions

### Claims

- C1: 全体 byte 列に対する最小挿入・削除と、全最適解の正規化後の合意だけが自動対応の確定根拠になる。
- C2: 対象範囲の current image は surviving bytes、内部編集、左右境界を含む正規化像であり、単一 diff の経路選択ではない。
- C3: 反復、移動、複製、分割・結合、境界所属の競合は保守的に UNRESOLVED とする。
- C4: 明示対応宣言は複数 current 範囲、理由、固定した二版を保持し、相反宣言を自動解決しない。
- C5: budget で合意検査を完了できない場合は NOT_EXAMINED とし、性能・資源上限を未決のまま保持する。

### Evidence

- E1: Accepted Issue #17 R1 の R7～R10 と AC4～AC5 は、位置移動、差異、対応不能、検索だけでは削除を断定しない境界を要求する。
- E2: ADR 0032 §5.1～5.3 は、全文最小挿入・削除、全最適解の対応合意、内部／境界 insertion、保守的な削除、明示宣言、budget 未完了を設計候補としている。
- E3: ADR 0032 §6、§8、§10 は、UNRESOLVED・NOT_EXAMINED・構造エラー・STALE を混同せず、Trace 比較を Cause と意味論から分離する。
- E4: 本 ADR の §3 の決定的例は、AC4～AC5 の expected outcome を byte 列で再現できる最小検証材料である。

### Actions

| Action | 根拠 | 後続成果 |
|---|---|---|
| A1 | C1～C3、E1～E2 | 比較器の内部設計で最適経路 DAG の合意判定、範囲像、競合分類を実装可能な形へ落とす |
| A2 | C4、E1～E3 | 保存・CLI companion が宣言の identity、理由、複数範囲、矛盾時の選択境界を定める |
| A3 | C5、E2～E4 | platform/budget decision が対象規模、時間・メモリ予算、測定条件、停止時の公開契約を定める |
| A4 | E4 | AC4～AC5 の deterministic tests を実装・検証へ対応付ける。テスト合格は設計の意味や製品提供完了を証明しない |

## 5. Alternatives and risks

| 選択 | 本提案 | 代案とリスク |
|---|---|---|
| 対応計算 | 全体最小挿入・削除と全最適解の正規化合意 | 単一 diff は実装が簡単だが、tie-break 依存の誤確定を許す。検索だけは反復・移動・削除を区別できない |
| 曖昧性 | UNRESOLVED として候補を保持 | 常に利用者宣言へ送ると自動利用価値が下がる。曖昧なまま DIFFERS にすると削除や移動を誤る |
| budget | 未完了は NOT_EXAMINED | 近似結果を MATCHES/DIFFERS に昇格すると、計算不足を事実認定へ変える。無期限計算は運用上の停止不能を招く |
| 元範囲の保持 | 複数 current 範囲と編集領域を正規化して保持 | 一つの包絡範囲へ潰すと内部の未対応部分と分割を失う。全経路の生データ保存は容量と可搬性の決定を増やす |

主要なリスクは、入力が大きい場合の計算量、反復 bytes による競合の増加、予算値未決のまま実装を進めること、宣言理由を意味論の証拠と誤解することである。これらは本 ADR で性能保証や意味監査を発明せず、所有する後続決定へ残す。

## 6. Non-decisions and unresolved interactions

- 公開 CLI のコマンド、flag、終了コード、JSON schema 名・member 順、保存 record 名、format version は決めない。これらは CLI/output と storage companion の所有範囲である。
- `MATCHES`、`DIFFERS`、`UNRESOLVED`、`NOT_EXAMINED` は本 ADR の論理状態の説明であり、公開名前空間の確定ではない。
- どの範囲を一つの比較呼出しで処理するか、全体予算と範囲予算の優先順位、並列化、ストリーミング、最大サイズ、性能しきい値、サポート platform は未決である。
- 重複範囲の拒否、同一 bytes の別 snapshot identity、宣言の取消・改訂・ローカル保持、観測中の bytes 変更の公開操作は、固定入力と矛盾しない範囲で CLI companion が定める。決定まで自動的に後勝ち・自動修復しない。
- 本 ADR の方式が想定する全最適解合意を実装上満たせない場合、それを単一 diff や近似解で置換して ADR 0032 の設計を緩和してはならない。具体的な計算不能条件と代替案を owner に返し、採否を得る。

## 7. Review

レビュー対象は本ファイル全体である。確認点は、Accepted R1 と ADR 0032 D1 の byte-level 境界、全最適解合意の実現可能性、範囲像の境界挿入規則、反復・移動・削除の保守性、明示宣言の矛盾、NOT_EXAMINED と UNRESOLVED の分離、AC4～AC5 の決定的例、後続 CLI/storage companion との非重複である。

本 ADR は Proposed であり、owner の確認前に public contract や implementation decision を確定しない。独立レビューが行われる場合は、レビュー対象の path とこのファイルの SHA-256 を固定し、所見・処置を別の review record に記録する。
