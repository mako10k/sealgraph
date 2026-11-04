# Review record案内の独立Agent再確認（2026-11-04）

前回の操作担当Agentが同じ試験episodeで限定再確認した。改善helpにprepare→提出recordの対応、全closure対象、個別判断欄、仕様リンク、意図的に無効なblank例が明示され、手書きのunreviewed record一件を添付・全文読戻しまで実行できた。blocking指摘なし。前回経験があり初見試験でも人間試験でもない。

## 条件と参照範囲

開始実測2026-11-04T13:00:24.971330587+09:00、操作完了13:00:56.918253627+09:00（約32秒、報告作成別）。作業ツリーのhelp改善を`go build -o /tmp/sealgraph-independent-trial-cSRXXA/sealgraph-guidance ./cmd/sealgraph`でビルド。SHA256=`dbe1acc37d7b84e555ba244d0a63082c97a94f394e426211f187a22a69f00f6b`。当初Agentは前回HEAD2467d3aと未commit作業ツリーを基準として記述したが、rootが終了後に当該binaryの `go version -m` を読み戻したところ vcs.revision=34a8cb0bda6cce197d14724a005f0f567150f568、vcs.time=2026-11-04T04:00:24Z、vcs.modified=false だった。ビルド時点では改善commit済みの34a8cb0を使用している。この訂正はbinaryの保存metadataに基づく。Agentはコード/テストを読んでいない。

参照順は今回docs/review-record.md→新binaryの`help review prepare`/`help review set`。storage-formatやADR本文は追加参照していない。ただし「新helpだけを隔離して初見で操作できた」という実験ではない。文書を先に読み、前回schema経験もある。新help自体に必要なmappingが含まれていることの読取りと、案内全体に従った実行を確認した。

操作は既存専用試験repo `/tmp/sealgraph-independent-trial-cSRXXA`のみ。実repoではこの新規報告だけを作成。Cause変更・Seal発行なし。親から解法の援助なし。

## helpから理解した対応

- prepareのschemaは提出recordと異なり、提出schemaは`sealgraph/historical-review/v1`。`ref`/`subject_seal`と**全**`other_heads`を同じfresh prepareから転記する。subjectを現在HEADに置換しない。
- 全targetsについてentriesを一つずつ作る。direct/transitiveだけでなくactive-leafも対象であり、target_sealと全compared_sealsを転記する。好みの最新版へ絞らない。実fixtureのclosureはroot P1一件のみで、深いclosureでの実証は今回していない。
- targets/revision_state/draftはprepare専用で提出recordへ持ち込まない。decision/difference/impact/reasonは自分の判断。retainには実際の非空の差分・影響・保持理由が必要で、未確認はunreviewedとして残す。
- blank例はREF/ID placeholderと空decisionを含み、意図的に無効と明記。固定retain判断・stock理由を提供していないことを目視確認した。無効例をそのままreview setへ送る試験は未実施。
- docs/review-record.mdとstorage-format §15への具体的URL、原本/差分/影響を読む手順、添付≠Seal/clean/承認がhelpにある。前回の「schema mappingにstorage-format本文が必要」という探索は今回不要だった。

## 個別判断と実操作

実コマンドは新binaryの`review prepare retained --format json`、`candidate show retained --format json`/`--raw-content`、`cat retained.txt`、`show retained --raw-content`、exact P1のshow JSON/raw、P2 raw、P1→P2 `compare --text`とP1 `impact --format human`。P1/P2の全IDは下記recordのもの。原本・Candidate・HEAD・exact Cause版・差分・影響を再読した。

Candidate本文は63 bytesの`Place the blue chair near the window if the floor supports it.\n`。HEAD本文は従来の窓配置。P1はchair存在のみのrootでtransitive Causesなし、P2はdesk追加でfloor/loadの情報なし。変更したCandidateのfloor支持が未確認という対象固有の理由で、P1についてunreviewedを選んだ。前回のretain判断をこの新Candidateへ持ち越していない。

fresh prepareを`guidance-prepare.json`に保存し、以下を`guidance-unreviewed.json`として手書きした（自動変換・定型判断生成なし）。

```json
{
  "schema": "sealgraph/historical-review/v1",
  "ref": "retained",
  "subject_seal": "f3eede89a01c10aeb0d5c64915b6d74f424d56b7798c491350c66ffaf6791967",
  "other_heads": [
    {"ref":"design","seal":"c0e2f7450b88735259015f1ab5d92823e2d16f8b02275117921108bd9002ef2b"},
    {"ref":"premise","seal":"d1d19f1f6f5db8a4f518901ba6cb723995021446a08a36b7c73a055858a98091"},
    {"ref":"quote","seal":"d54113ace734d550587711285d3e30821375c61f39e64b5c22d4156ee464fbb8"}
  ],
  "entries": [{
    "target_seal":"1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8",
    "compared_seals":["d1d19f1f6f5db8a4f518901ba6cb723995021446a08a36b7c73a055858a98091"],
    "decision":"unreviewed",
    "difference":"P2 adds a desk while preserving the chair; current Candidate additionally conditions placement on floor support.",
    "impact":"P1 is a root with no transitive Causes. Neither read premise supplies floor/load evidence for the changed Candidate.",
    "reason":"Floor support for the proposed placement remains unconfirmed; this entry records uncertainty and makes no retain judgment."
  }]
}
```

この一対象の判断をjudgments.txtへ記録し、個別操作`./sealgraph-guidance review set retained --file guidance-unreviewed.json`で添付。receiptはoperation=set。続く`review show retained --candidate --format json > guidance-readback.json`を全文読んだ。

読戻しはcandidate=true、selected_seal=`664fae4b50abb2a75694c579c85c61be3274ee750fa218821da9093379a21c32`（prospective identity、発行済Sealではない）。record/currentのsubjectは両方`f3eede89...`、other_heads三件・targets/entries一件・comparison集合が一致し、decisionと理由はunreviewedのまま。`applicable=false`、`REVIEW_UNCONFIRMED: Cause 1764628a71ee689e9cda63cb895d418895f75bf47266d0bf9c64352980591ba8 remains unreviewed`を確認。

`inspect retained --format human`でもUNSEALED、WORKFILE_MATCHES_CANDIDATE、baseline=CANDIDATE、stale direct=1、candidate claim applicable=false/REVIEW_UNCONFIRMED。旧HEAD R1=`32dae0c39a9f7330e7c78f9cc5d3a2b166f7e70ffba77bf1cc32eee0df4fb3b2`はそのまま、旧claimは別表示。未確認をclean/保持承認/公開へ変えていない。Sealによる拒否確認は不要とのscopeに従い未実施。

証拠ファイルはすべて試験場所の`guidance-prepare.json`, `guidance-unreviewed.json`, `guidance-readback.json`, `judgments.txt`。一時場所の恒久保存は保証しない。追加改善の実装はしていない。

終了前worktimectlはrequired_actions=[]、No work-time action is required。報告作成完了実測13:01:43.246145848+09:00（開始から約78秒）。共有workdayの変更なし。
