# 旧版保持レビューJSONの記入

`sealgraph review prepare REF --format json` は記入に必要な事実を返す読み取り専用操作です。提出用のレビューJSONではありません。まず `sealgraph help review set` の記入例を確認してください。仕様の正本は [storage-format §15](storage-format.md#15-contained-historical-review-evidence)、判断・公開境界は [ADR0043](adr/0043-reviewed-historical-cause-publication.md)です。

## 事実を転記する

同じ時点のprepare結果から、次の対応で手元のJSONファイルへ転記します。IDは省略しない64桁の正確な値です。

| 提出recordの欄 | 転記元・記入方法 |
| --- | --- |
| schema | `sealgraph/historical-review/v1`。prepareのschemaとは異なる |
| ref / subject_seal | prepareの同名欄。subjectはreview attachmentを除いたprospective identityで、現在HEADのIDへ置換しない |
| other_heads | prepareの全要素をそのまま転記。対象REFは含まれない |
| entries | prepareのtargets全要素につき1件。direct/transitive・active leafも省略しない |
| target_seal / compared_seals | 対応するtargets要素の同名欄と全比較ID。比較集合を「好みの最新版」一つに絞らない |
| decision / difference / impact / reason | 下記の個別判断を自分で記入する |

prepareだけの`targets`・`revision_state`・`draft`は提出recordへ追加しません。unknown/duplicate fieldsは拒否されます。空の配列はprepareの完全な観測が実際に空の場合だけ使い、記入省略を表さないでください。

以下は**未記入の構造例**です。ID/REFのプレースホルダーと空decisionは意図的に無効なので、そのまま登録できません。対象数に合わせて全要素を記入してください。

```json
{
  "schema": "sealgraph/historical-review/v1",
  "ref": "<prepare.ref>",
  "subject_seal": "<prepare.subject_seal>",
  "other_heads": [
    {"ref": "<prepare.other_heads[i].ref>", "seal": "<prepare.other_heads[i].seal>"}
  ],
  "entries": [
    {
      "target_seal": "<prepare.targets[i].target_seal>",
      "compared_seals": ["<prepare.targets[i].compared_seals の全ID>"],
      "decision": "",
      "difference": "",
      "impact": "",
      "reason": ""
    }
  ]
}
```

## 判断を記入して読み戻す

対象原本・Candidate全文・各exact Cause・比較版・差分と影響を自分で読みます。`retain`は、その特定版を維持すると個別に判断した場合だけ選び、difference（何が違うか）、impact（この内容への影響）、reason（維持理由）を実際の根拠からそれぞれ空白でない文章にします。定型の理由をコピーして確認済みにしません。

未確認なら`unreviewed`を選べます。この場合は文章を空にできますが、`seal --review-historical`での公開は拒否されます。未確認事項は未確認のまま残してください。draft Causeがある場合も、このレビューでnon-draft公開を通せません。

作成したファイルを `review set REF --file YOUR_FILE` で対象Candidateへ添付し、`review show REF --candidate --format json`で全文と適用可能性を読み戻します。添付はSeal発行ではなく、staleも消しません。その後のSeal発行は独立した個別判断と操作です。本文/Cause/他HEAD等が変わった場合はprepareを読み直し、影響を受ける判断を再確認してください。空欄を埋めただけで正しさ・理解・承認を証明するものではありません。
