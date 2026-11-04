package cli

// R: explain manual historical-review record authoring without supplying judgments.

const reviewRecordGuidance = "Writing guide: docs/review-record.md. Exact schema: docs/storage-format.md section 15 (https://github.com/mako10k/sealgraph/blob/main/docs/storage-format.md#15-contained-historical-review-evidence). Copy ref, subject_seal and the COMPLETE other_heads from one fresh review prepare result. For EVERY targets item, including transitive/active-leaf Causes, copy target_seal and its complete compared_seals into entries. Do not copy preparation-only schema/targets/revision_state/draft into the record. Preparation is facts, not a ready-to-submit review."

const reviewRecordExample = `Manual record layout ONLY — not a runnable command or a valid submitted review.
Replace ID/REF placeholders with exact facts; repeat all heads/targets (arrays may be empty only when preparation says so).
{
  "schema": "sealgraph/historical-review/v1",
  "ref": "<REF from prepare>",
  "subject_seal": "<subject_seal from prepare>",
  "other_heads": [
    {"ref": "<other_heads[i].ref>", "seal": "<other_heads[i].seal>"}
  ],
  "entries": [
    {
      "target_seal": "<targets[i].target_seal>",
      "compared_seals": ["<each targets[i].compared_seals ID>"],
      "decision": "",
      "difference": "",
      "impact": "",
      "reason": ""
    }
  ]
}
The blank decision and placeholders deliberately fail validation; do not submit unchanged.
Read each exact original, Candidate, comparison version, difference and impact yourself.
Choose retain only after that individual judgment and write nonblank difference/impact/reason.
Choose unreviewed to record uncertainty; text may remain empty, but historical publication is blocked.
Copy no stock retain reasons. If facts change, prepare again and reassess affected judgments.
Only after authoring: review set REF --file YOUR_FILE; read back review show REF --candidate.
Attaching a record does not publish a Seal, clear stale, prove truth or confer approval.`
