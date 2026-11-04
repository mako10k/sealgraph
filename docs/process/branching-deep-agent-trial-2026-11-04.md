# 複数leaf・深いCause依存の独立操作試験（2026-11-04）

## 結果と限界

独立操作担当 `/root/branching_deep_operator_trial` が、利用仕様と公開CLI helpだけから専用fixtureを作り、同じ旧targetに対する2 sibling active leaves、4 Cause hopsの深いACTIVE_NON_LEAF理由、全closureのprepare、2経路の表示省略、未確認reviewの記入・登録・読戻しを確認した。限定fixture内で要求された操作は実行できた。normal/historicalの拒否を各1回観測し、HEADは変更されずCandidateとstaleを保持した。

性能・規模一般・完全性・初心者人間の使いやすさは保証しない。OUTSIDE_ACTIVEの深いfixtureは未実施（今回は指定の選択肢のうちACTIVE_NON_LEAFを選択）。全retainでのhistorical公開、format6/7、Trace、Git sidecar、競合、破損、複数observerが同一edgeを重複主張する場合も未実施。製品修正、main repositoryの.sealgraph更新、Git操作/remote操作は行っていない。

## 独立性・環境

- 事前読取: AGENTS.md、docs/operations.md、docs/review-record.md、requirements §3/4、cli §10/12と関連索引、ADR0043/0045、help workflows/add/seal/review set。help observeも終了時に保存。
- 実装code、既存test、過去Agent結果は読んでいない。ADR中の実装file名は文書として見たのみ。
- rootからはscope、試験条件、受入HEAD、個別判断ルール、進捗通知のみ。実装解法の助言は受けていない。途中にrootのdocs/PERT更新通知があったが、binaryは再buildせず固定。
- 開始時HEAD: `45d5528bad61cc868cb26696608fdbc736429ac9`。buildは `go build -o /tmp/sg-branchdeep-E6Qm8n/sealgraph ./cmd/sealgraph`。
- binary SHA-256: `a220a0ecbbed8f0384822c837c56c4858d86c17426b635618ada5bd39699d4e8`。
- 2026-11-04 14:20頃JST開始、build後時刻14:21:49+09:00、主要操作終了14:24:54+09:00。報告保存はその後。
- `worktime-coordination` を読み、開始時・終了時のworktimectlはともに `No work-time action is required.` / required_actions=[]、planned_end=20:00+09:00。共有workday変更なし。
- 全fixture/binary/原本/logは `/tmp/sg-branchdeep-E6Qm8n`。initは同directoryで実行。作成した空のfixture subdirectoryは未使用。main repositoryの書込は本報告だけ。

## 原本・個別判断と操作

無害な短文を自分で作成。各add後に原本とCandidate全文/JSONを読み、各sealは判断を記録して個別のexec_commandで1対象だけ実行し、show JSON/raw-contentを読戻した。Cause Linkを持つaddも各対象別の個別操作。唯一の追加linkはl4→l1で、操作前にl4 Candidate全文とexact l1を読んで判断を記録し、操作後Candidate全文/JSONを確認した。Seal発行・Cause更新のloop/script/自動再試行なし。読み取り原本exportだけにloopを使用。以下は実行時judgments.txtの記録で、省略IDの完全対応は次表に示す。

```text
premise: read old.txt and complete Candidate; new explicit external premise root, no Causes or prior diff; retain exact bytes as fixture premise.
l1: read full l1 source/Candidate and exact f45c7d... premise; initial content restates blue permission; no prior version; retain exact Cause because its permission directly supports the sentence.
l2: full source/Candidate and e697... read; initial selection narrows l1 permission to blue; no old diff; retain exact l1 because it explicitly permits the selected color, with f45 premise already read.
l3: read full source/Candidate and ded19...; initial rendering instruction uses selected blue without changing permission; retain l2 and its already-read exact closure, no revision assertion.
l4: read source/Candidate and 94d18... original; initial display instruction consumes l3 blue rendering; retain l3 closure (l2/l1/premise read above), four Cause hops, no prior diff.
premise A: read old root and a.txt/Candidate; adds cyan while preserving blue. Update this root bytes; no Causes. Publication alone is not revision evidence; separate observer will assert A supersedes old.
sibling-b: read b.txt/Candidate; new explicit root adds navy to the old blue premise rather than cyan; no Cause. Keep as independent root leaf so later observer can assert alternative successor, never preferred latest.
observer-a: old and A originals, oa source/full Candidate read; A preserves blue and adds cyan, so explicitly accept target A previous old revision assertion scoped to this observer. Retain that complete record; initial content accurately names extension.
observer-b: read old/B originals and ob Candidate; B preserves blue, adds navy rather than cyan; assert B previous old separately, not B after A. Retain record and issue only this observer.
l4 link judgment: Candidate text unchanged and l1 exact original read; add l1 as direct supporting blue permission alongside l3 rendering, no revision relation. This creates a justified shared-target path test; do not issue changed Candidate while old premise impact remains pending.
Negative admission probe l4: complete attached Candidate and all exact source/targets/comparisons read; old premise deliberately unreviewed. Attempt normal seal once solely to verify required rejection of reachable nonleaf; no decided publication is claimed. If it unexpectedly publishes, stop and report.
Negative historical probe l4: normal attempt readback confirmed no HEAD/Candidate change; attached record remains unreviewed for f45 old premise. One separate historical attempt tests refusal, not approval or automatic retry.
```

## Exact IDsと原本

| REF / version | exact SealID | original content |
| --- | --- | --- |
| observer-b | `8be534e83c7fbb79886ed1765c51c84d396c18a82a2a3c34c6b378e2146b86a5` | Observer B records the navy extension of the old blue premise. |
| sibling-b | `8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5` | Premise B: blue and navy are permitted. |
| l3 | `94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a` | Layer three renders the blue selected by layer two. |
| premise | `a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a` | Premise A: blue and cyan are permitted. |
| observer-a | `b6f6838ed09515e5db027f8803f652c753a364b60c24b0d85e3764f2fb0257c3` | Observer A records the cyan extension of the old blue premise. |
| l2 | `ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e` | Layer two selects blue allowed by layer one. |
| l1 | `e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf` | Layer one permits blue under the premise. |
| l4 | `e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9` | Layer four displays the layer three blue rendering. |
| premise old | `f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77` | Premise: blue is permitted. |

旧premiseをrootとして発行し、l1→旧premise、l2→l1、l3→l2、l4→l3を順に作った。この時点では全Causeがactive leafなので通常発行した。その後premiseのroot内容をAへ変更し、Bを別REF rootとして発行した。単なるHEAD進行をrevisionと読み替えず、observer-aでA previous old、observer-bでB previous oldを各々明示した。A/BはそれぞれHEADなのでactive、両edgeが同旧版へ向くため旧版はactive nonleafとなる。A/B間の前後関係は主張していない。

## 観測したrevision証拠

以下は旧exact targetを選んだhuman出力。exact observer/provenance/target/previousと2 successorが表示され、preferred/latestという選別はない。observeのJSONでもA/Bそれぞれactive=true/active_leaf=true、旧版active=true/active_leaf=falseを確認した。

```text
OBSERVATION sealgraph/observation/v1 scope=SELECTED_SEAL
Membership limitation: Outside-active membership does not distinguish historical from detached with current observation evidence; no observed successor is not proof of nonexistence.
Recorded evidence only: no observed successor does not prove nonexistence. Cause reachability does not confer active membership.
Reasons explain review targets; they do not change legacy stale membership or publication conditions.

Seal f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77 active=true active_leaf=false
  Observed successors=[8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5 a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a]
  Previous assertions:
  {
    "TargetSeal": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
    "PreviousStates": [
      "PREVIOUS_NONE_ASSERTED"
    ],
    "Assertions": [
      {
        "ObserverSeal": "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
        "ObserverSealSchema": "sealgraph/seal/v5",
        "ObserverProvenance": "30365eb60b9aa621de1fb450eb3bf47b2987e216519d316a874be6c79e416732",
        "ObserverProvenanceSchema": "sealgraph/provenance/v1",
        "CauseLink": {
          "target_seal": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
          "previous_revision_seal_of_target_seal": [],
          "messages": [
            "Layer one uses the explicit blue permission, not a revision assertion."
          ],
          "metadata": []
        }
      }
    ],
    "StructuralPrevious": []
  }
  Successor assertions (exact observer / target / previous):
  [
    {
      "ObserverSeal": "8be534e83c7fbb79886ed1765c51c84d396c18a82a2a3c34c6b378e2146b86a5",
      "ObserverSealSchema": "sealgraph/seal/v5",
      "ObserverProvenance": "e0947bb24cd80b0368349ff0f0fe63f63d016dce1607a3af7d1f01aa1e8cd862",
      "ObserverProvenanceSchema": "sealgraph/provenance/v1",
      "CauseLink": {
        "target_seal": "8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5",
        "previous_revision_seal_of_target_seal": [
          "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"
        ],
        "messages": [
          "B extends old blue permission with navy independently of A."
        ],
        "metadata": []
      }
    },
    {
      "ObserverSeal": "b6f6838ed09515e5db027f8803f652c753a364b60c24b0d85e3764f2fb0257c3",
      "ObserverSealSchema": "sealgraph/seal/v5",
      "ObserverProvenance": "8881c42fcb6b75fa2f5192a79177c945d32194e5a6972cdfb7f8aa522e37a2c4",
      "ObserverProvenanceSchema": "sealgraph/provenance/v1",
      "CauseLink": {
        "target_seal": "a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a",
        "previous_revision_seal_of_target_seal": [
          "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"
        ],
        "messages": [
          "A extends old blue permission with cyan; observer A asserts this revision only."
        ],
        "metadata": []
      }
    }
  ]
  Review reasons=NONE in this observation scope
  Read immutable original: sealgraph show @f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77

Next read: sealgraph inspect (workfiles/Trace/review); sealgraph help observe; sealgraph help selectors
```

## 4 hopsと共有targetの複数経路

l4 HEADからoldへの4 Cause hopsをhumanで確認。revision edgeはこのCause pathに混ぜない。

```text
OBSERVATION sealgraph/observation/v1 scope=SELECTED_REF
Membership limitation: Outside-active membership does not distinguish historical from detached with current observation evidence; no observed successor is not proof of nonexistence.
Recorded evidence only: no observed successor does not prove nonexistence. Cause reachability does not confer active membership.
Reasons explain review targets; they do not change legacy stale membership or publication conditions.

REF "l4"
  HEAD=e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9 draft=false
  Candidate=ABSENT
  Workfile binding=ABSENT
  Legacy stale self=false direct=[] transitive=[[94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77]]

Seal e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9 active=true active_leaf=true
  Observed successors=[]
  Previous assertions:
  {
    "TargetSeal": "e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9",
    "PreviousStates": [
      "PREVIOUS_UNOBSERVED"
    ],
    "Assertions": null,
    "StructuralPrevious": []
  }
  Successor assertions (exact observer / target / previous):
  []
  Reason ACTIVE_NON_LEAF subject=e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9 target=f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77 direct=false paths_complete=true
    Cause path: e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9 -> 94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a -> ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e -> e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf -> f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77
  Read immutable original: sealgraph show @e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9

Next read: sealgraph inspect (workfiles/Trace/review); sealgraph help observe; sealgraph help selectors
```

l4に同文Candidateを作成してl1を直接Causeとして追加した後、旧premiseに2経路がある。以下はcandidate診断の実出力抽出。--max-paths=1は1経路表示とpaths_complete=false、100は4hopと2hopの両経路とtrue。両方に同一targetの理由が残り、表示制限をtarget集合の省略へ読み替えていない。

### limited

```json
{
  "draft": false,
  "prospective_seal": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
  "exact_digest": "a19cf8e050ac16b669cfd3aeae2a83723092c72eb9cd658db0f10b699bcb929d",
  "expected_ref_head": "e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9",
  "expected_head_state": "EXPECTED_CURRENT",
  "current_head_comparison": "DIFFERENT",
  "diagnostic_basis": "CURRENT_OBSERVATION_NOT_ADMISSION",
  "reasons": [
    {
      "subject": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
      "target": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
      "kind": "ACTIVE_NON_LEAF",
      "direct": false,
      "paths": [
        [
          "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
          "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a",
          "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e",
          "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
          "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"
        ]
      ],
      "paths_complete": false
    }
  ]
}
```

### expanded

```json
{
  "draft": false,
  "prospective_seal": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
  "exact_digest": "a19cf8e050ac16b669cfd3aeae2a83723092c72eb9cd658db0f10b699bcb929d",
  "expected_ref_head": "e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9",
  "expected_head_state": "EXPECTED_CURRENT",
  "current_head_comparison": "DIFFERENT",
  "diagnostic_basis": "CURRENT_OBSERVATION_NOT_ADMISSION",
  "reasons": [
    {
      "subject": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
      "target": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
      "kind": "ACTIVE_NON_LEAF",
      "direct": false,
      "paths": [
        [
          "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
          "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a",
          "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e",
          "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
          "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"
        ],
        [
          "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
          "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
          "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"
        ]
      ],
      "paths_complete": true
    }
  ]
}
```

## prepareと手書きreview

prepareは4 distinct targets（l3/l2/l1/旧premise）を保持。direct targetのl3/l1だけに絞らず、transitive l2/旧premiseも転記。旧premiseのcompared_sealsはB/Aの両方、各active leafは自身との比較である。subjectはHEADでもreview attachment込Candidateでもない。

```json
{
  "schema": "sealgraph/historical-review-prepare/v1",
  "ref": "l4",
  "subject_seal": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
  "other_heads": [
    {
      "ref": "l1",
      "seal": "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf"
    },
    {
      "ref": "l2",
      "seal": "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e"
    },
    {
      "ref": "l3",
      "seal": "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a"
    },
    {
      "ref": "observer-a",
      "seal": "b6f6838ed09515e5db027f8803f652c753a364b60c24b0d85e3764f2fb0257c3"
    },
    {
      "ref": "observer-b",
      "seal": "8be534e83c7fbb79886ed1765c51c84d396c18a82a2a3c34c6b378e2146b86a5"
    },
    {
      "ref": "premise",
      "seal": "a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a"
    },
    {
      "ref": "sibling-b",
      "seal": "8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5"
    }
  ],
  "targets": [
    {
      "target_seal": "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a",
      "compared_seals": [
        "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a"
      ],
      "revision_state": "ACTIVE_LEAF",
      "draft": false
    },
    {
      "target_seal": "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e",
      "compared_seals": [
        "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e"
      ],
      "revision_state": "ACTIVE_LEAF",
      "draft": false
    },
    {
      "target_seal": "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
      "compared_seals": [
        "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf"
      ],
      "revision_state": "ACTIVE_LEAF",
      "draft": false
    },
    {
      "target_seal": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
      "compared_seals": [
        "8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5",
        "a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a"
      ],
      "revision_state": "ACTIVE_NON_LEAF",
      "draft": false
    }
  ]
}
```

記入はdocs/review-record.mdとhelpの対応表で完了できた。ref/subject/全other_headsはprepareの事実を転記。recordをPythonで整形したが、entriesの4 decision/difference/impact/reasonは対象別に自分で直接記述したliteralであり、targetsのloopや定型retain生成ではない。l3/l2/l1は各本文の役割を区別してretain、旧premiseはA/Bの具体的差分を読んだが、採用/旧版保持の判断を意図的に未確定としてunreviewedを残した。これは確認不足を装うのではなく、未決定状態を保存するための明示fixture条件。本文上blueが保存されるという観測と、historical relianceを決定したという主張を区別した。

```json
{
  "schema": "sealgraph/historical-review/v1",
  "ref": "l4",
  "subject_seal": "e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c",
  "other_heads": [
    {
      "ref": "l1",
      "seal": "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf"
    },
    {
      "ref": "l2",
      "seal": "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e"
    },
    {
      "ref": "l3",
      "seal": "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a"
    },
    {
      "ref": "observer-a",
      "seal": "b6f6838ed09515e5db027f8803f652c753a364b60c24b0d85e3764f2fb0257c3"
    },
    {
      "ref": "observer-b",
      "seal": "8be534e83c7fbb79886ed1765c51c84d396c18a82a2a3c34c6b378e2146b86a5"
    },
    {
      "ref": "premise",
      "seal": "a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a"
    },
    {
      "ref": "sibling-b",
      "seal": "8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5"
    }
  ],
  "entries": [
    {
      "target_seal": "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a",
      "compared_seals": [
        "94d18ab3014214e24b9be7c2cac3471caf2a55ef7de02d41ec1691c9752bfe0a"
      ],
      "decision": "retain",
      "difference": "The comparison is the same immutable layer-three rendering sentence.",
      "impact": "Layer four displays precisely that blue rendering; no change in this rendering instruction.",
      "reason": "Keep this exact rendering source rather than invent another rendering version."
    },
    {
      "target_seal": "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e",
      "compared_seals": [
        "ded19d13cb623f8044e7d94441f1cb640006a8ec570be3f166729d5e5dea5b5e"
      ],
      "decision": "retain",
      "difference": "Layer two compares only to itself and selects blue.",
      "impact": "Its blue choice remains the choice consumed by layer three.",
      "reason": "Retain the exact selection instruction; this judgment does not settle the upstream premise comparison."
    },
    {
      "target_seal": "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf",
      "compared_seals": [
        "e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf"
      ],
      "decision": "retain",
      "difference": "Layer one has no different compared leaf; its text permits blue under the premise.",
      "impact": "It supplies the direct blue permission and also supports layer two.",
      "reason": "Retain that precise permission statement, while the old root reliance is separately pending."
    },
    {
      "target_seal": "f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77",
      "compared_seals": [
        "8f39054f1a7d0c505a035efd4c68b8199821cc52bccc6280e4c85821232036b5",
        "a58a9bc204d5637782eb69e84043620a8422ebccc3b818b7f53c2308c340f99a"
      ],
      "decision": "unreviewed",
      "difference": "A adds cyan; B adds navy; both preserve blue in their short fixture originals.",
      "impact": "The displayed blue is textually permitted by all three, but this trial intentionally leaves the choice to adopt or retain a premise unresolved.",
      "reason": "Pending: no decided historical reliance is claimed for this premise. Keep both sibling comparisons visible."
    }
  ]
}
```

`review set l4 --file review.json`は成功、`review show l4 --candidate --format json`で全文章と4 decision、全comparison/headsを読戻し。selected_seal=`68f8955eb5277c72721cfa6950eca9a72268f13b59c61418bb12585d8faa534b`、record subject=`e31d4b6806d64850b82b0b76987ef7df5950ed23cd78be1e7e8e3308dea4f91c`。`applicable=false`、limitationは`REVIEW_UNCONFIRMED: Cause f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77 remains unreviewed`とexecutor claimの限界を表示した。添付によってstaleは消えなかった。

## 拒否と非変更の確認

完全Candidate/record/Causeと比較版を読んだ後、拒否を確かめるnegative probeとして各1回だけ個別に実行。normalの拒否後にHEAD/Candidateを先に読戻してから、別判断でhistorical probeを実行した。いずれもexit=1。新しいSeal発行を得るためのretryではない。

normal:

```text
error: sealgraph seal: normal Seal requires Cause f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77 to be an active revision leaf; inspect individual historical reliance with review prepare, or leave publication pending
reason: normal non-draft publication requires every reachable Cause to be a non-draft active revision leaf
hint: inspect the named target with `sealgraph show @SEAL_ID` and current review state with `sealgraph stale --frontier`
hint: for decided historical reliance, use `sealgraph review prepare REF`, record individual version-bound judgments with `review set`, then explicitly select `seal REF --review-historical`; leave unreviewed impact pending
hint: draft represents provisional work; historical reliance alone does not require draft, and no automatic relink or re-Seal is performed
help: sealgraph help seal
```

historical:

```text
error: sealgraph seal: REVIEW_UNCONFIRMED: Cause f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77 remains unreviewed
reason: recorded historical reliance must cover the exact dependent subject, all Cause versions and the current observation
hint: inspect `sealgraph review show REF --candidate` and `sealgraph review prepare REF`; review each difference and impact individually before recording a new claim
hint: leave unreviewed dependencies pending; no automatic relink, re-Seal or force is available
help: sealgraph help seal
```

終了時HEADは元のl4 `e79b4590070f9b5441fc61ad9907d45e3315ba98bdd975f1a91f6b8b63d500e9`、Candidateはreview添付済み `68f8955eb5277c72721cfa6950eca9a72268f13b59c61418bb12585d8faa534b`、reviewはapplicable=falseのまま。Candidate未発行・旧premise未確認を保留した。

## 旧診断との区別

status/staleのsealed HEADに対する結果はl1 STALE_DIRECT、l2/l3/l4 STALE_TRANSITIVE。refs-onlyはl1/l2/l3/l4、frontierはl1のみ。observeのCandidateは同じold targetについて複数経路/表示省略を説明するが、旧診断の意味・membership・admissionを変更しない。l4 Candidateありを公開成功やcleanに読み替えていない。以下は取得したfrontierとrefs-only。

```json
{"schema":"sealgraph/stale/v2","frontier":true,"scan":false,"statuses":[{"ref":"l1","head_seal_id":"e6978445159d1442f891067756e88d699c9d816d734983c3f584cd0750073bdf","draft":false,"stale":{"self":false,"direct_target_seal_ids":["f45c7d034663bd14702585887f7610676f364e83451b3646206146796b7d3b77"],"transitive_paths":[]},"sealed_state_labels":["STALE_DIRECT"]}]}
```

```text
l1
l2
l3
l4
```

## 困惑・改善候補・未実施

操作を止めるほどの不明点・想定外挙動・仕様矛盾はなかった。初回にreview setのblank exampleが提出可能なrecordではないこと、prepare subjectと添付後Candidate identityが別であることを注意して読んだ。いずれも今回の記入案内に説明があり、解法の援助は必要なかった。新しい必須改善候補はこの限定試験からは提起しない。日本語案内と英語helpの往復、および全closure手記入の負担はあるが、短い4 targetsの成功から大規模・初心者にも容易とは推論しない。

未実施の範囲は冒頭のとおり。完全性、実データでの判断妥当性、一般性能の結論は出さない。fsckは構造okだけを示し、review理解・承認・cleanの証拠とはしない。

## 補足証拠と保存先

一時directoryにobservation.json、limited.json、expanded.json、prepare.json、review.json、review-readback.json、final-review.json、status/stale/frontier/refs-only、admission失敗出力、final-head/candidate、全9 immutable originals JSON/txt、judgments.txt、build-metadata.txtを保存。tmpは恒久保管ではないため、主要結果とexact IDs/recordは本報告にも収録した。

build metadata:

```text
./sealgraph: go1.27.3
	path	github.com/mako10k/sealgraph/cmd/sealgraph
	mod	github.com/mako10k/sealgraph	v0.1.0-beta.7.0.20261104052043-45d5528bad61	
	build	-buildmode=exe
	build	-compiler=gc
	build	DefaultGODEBUG=tracebacklabels=0,x509sslcertoverrideplatform=0
	build	CGO_ENABLED=1
	build	CGO_CFLAGS=
	build	CGO_CPPFLAGS=
	build	CGO_CXXFLAGS=
	build	CGO_LDFLAGS=
	build	GOARCH=amd64
	build	GOOS=linux
	build	GOAMD64=v1
	build	vcs=git
	build	vcs.revision=45d5528bad61cc868cb26696608fdbc736429ac9
	build	vcs.time=2026-11-04T05:20:43Z
	build	vcs.modified=false
Linux AXPC-7020N 6.18.33.2-microsoft-standard-WSL2 #1 SMP PREEMPT_DYNAMIC Thu Jun 18 21:54:43 UTC 2026 x86_64 x86_64 x86_64 GNU/Linux
repository_format = 5
object_format = sha256
ref_format = manifest-v1
```

fsck:

```json
{"schema":"sealgraph/fsck/v2","result":"ok","blobs":35,"seals":9,"materials":9,"provenances":7,"refs":8,"tags":0,"active_seals":9,"historical_or_detached_seal_ids":[],"unreferenced_blob_ids":["1cfe1661dcdf7707b194c12be14c552d816fa314c68c68ba312586eda268aefa"]}
```

報告・最終読戻し完了: 2026-11-04T14:26:30+09:00。機械的な事実検証として、最終recordと手書きreview.jsonの完全一致、current prepareと保存prepareの完全一致、applicable=false、l4 HEAD不変、A/B両active leaf・旧版active nonleafを確認した。最終worktimectlもrequired_actions=[]であった。

## rootによる保存証拠の照合

担当rootは報告受領後、保存binaryのSHA-256とbuild metadata、limited/expanded観測、prepare、final-review、final-head、normal/historical拒否ログとfsckを読み戻した。手書きrecordとfinal-review.recordは一致し、4 entriesのうち旧premiseはunreviewed、applicable=false。max-paths=1/100間でsubject/target/kindの集合は同一で、省略表示と2経路を照合した。これらは独立実行の保存証拠の検証であり、試験操作全体の再実行や新たな人間試験ではない。製品Go sourceへの差分はない。
