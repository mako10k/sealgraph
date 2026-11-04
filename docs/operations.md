# SealGraph operations: start here

This guide is for a person registering a few documents, an LLM operator and an auditor. It implements Accepted [UX R1](process/usability-requirements-r1-acceptance-2026-11-04.md). Notifications occur only when inspection runs. Nothing watches your files or sends messages in the background.

## Know what the result means

| Observation | What it establishes | What it does not establish |
| --- | --- | --- |
| Workfile/baseline | Explicitly bound file matches/differs from Candidate, otherwise HEAD | Approval, dependency freshness or all repository files checked |
| Provenance stale | Observed Cause/revision relation needs review | A file changed, every new HEAD supersedes an old one, or mandatory repair |
| Original-source Trace | Saved quoted bytes are present/absent/unconfirmed in an explicitly bound current source | Semantic equivalence, editing history, or approval |
| Historical review | A recorded executor claim exists and covers or no longer covers this observation | Truth, understanding, complete preflight or owner acceptance |

REF is a logical name; Seal is an exact immutable version. Root is a declared provenance boundary. Seals have no intrinsic parent: a newly published HEAD does not by itself prove a revision relation. A Cause's `--previous` asserts that relation for its target, scoped to that observer. Do not invent revision assertions simply to get a stale result.

Help does not open a repository: `sealgraph help workflows`, then `sealgraph help COMMAND`. Source binding is explicit local input configuration, not Git membership. Standalone does not inspect Git.

## S1: register the first content and its basis

Start with actual files you have read. In the following example, `premise.txt` is the external premise and `design.txt` the dependent content. Choose the logical REF names deliberately.

```sh
sealgraph init
sealgraph add premise --root --clear-cause-links --content-file premise.txt --bind-source
sealgraph candidate show premise --format human
sealgraph candidate show premise --raw-content
```

Read the complete Candidate. Decide whether its bytes and explicit root boundary are the intended claim. Only after that individual judgment:

```sh
sealgraph seal premise
sealgraph show premise --format json
sealgraph show premise --raw-content
```

Read back the exact SealID and content. For dependent content, first read that exact premise version. Use its full ID as `@PREMISE_SEAL` below; placeholders are replaced manually, never guessed.

```sh
sealgraph add design --non-root --content-file design.txt --bind-source --target @PREMISE_SEAL --no-previous -m 'Describe why this exact premise supports this content'
sealgraph candidate show design --format human
sealgraph candidate compare design --format human
sealgraph candidate show design --raw-content
```

The message is your own claim; do not copy the example as evidence of review. Inspect content and exact Cause. If the result is intended, issue one Seal and read it back with `sealgraph show design --format json` and `--raw-content`. The tool does not make that decision for you.

## S2: find changes without mixing their meanings

```sh
sealgraph inspect --format human
sealgraph inspect design --format human
```

The first command inventories registered REFs/Candidates/source bindings; it does not discover all repository files. `inventory_complete=false` and `omitted_refs` identify omitted entries (default visible limit 100). Select a REF or explicitly increase `--max-refs`. It is not an all-clear when a binding, Trace or baseline is absent. SOURCE_MISSING/UNREADABLE and incomplete observations need investigation.

A workfile difference is relative to Candidate when present, otherwise HEAD. It does not mutate either. `add design` refreshes the explicitly bound file into a Candidate and preserves omitted semantics; `seal` never rereads the workfile.

Provenance stale follows observed revision assertions, not every file/HEAD change. For structural investigation, use `stale --frontier`, `impact`, `log` and `linklog`. Read the exact target IDs shown by `status --format json`/Candidate inspection. Do not treat structural reachability as semantic impact.

Trace is format7-only. New `init` creates format5. Inspect `.sealgraph/config` and `sealgraph help migrate repository`; if needed, choose an explicit supported migration. Inspection never migrates. Historical `NO_TRACE` is unknown origin, not a reconstructed claim.

Current source-key bindings are separate from saved original-source snapshots. `sealgraph trace source bind KEY --file SOURCE_PATH` explicitly selects the current comparison source; it does not author Trace or certify its meaning. `trace compare --ref design --max-graph-visits 100` provides local and directional detail. A graph budget or source failure leaves unobserved scope explicit.

## S3: adopt a newer premise after reading originals

Use `inspect` and `stale --frontier` to choose a subject. Do not infer a preferred latest version when several revision leaves exist. Read:

```sh
sealgraph show @DEPENDENT_SEAL --raw-content
sealgraph show @OLD_PREMISE --raw-content
sealgraph show @COMPARISON_PREMISE --raw-content
sealgraph compare @OLD_PREMISE @COMPARISON_PREMISE --text
sealgraph impact @OLD_PREMISE --format human
```

`--text` is human-only, including on a pipe. It bounds input previews at 64KiB per side and difference lines at 64KiB; large/complex/binary cases and omitted data are labelled. Full repository Blobs are still read and validated. Extract exact originals separately to review beyond the displayed scope. The renderer is not a semantic reviewer.

If no Candidate exists, prepare one with `add design` for an already bound file, or `add design --content-file design.txt` otherwise. Read the resulting Candidate and exact Cause versions before changing a link.

A target change normally requires two separate judged operations. First add/replace the new complete target record using `link design --target @NEW --previous @OLD -m YOUR_REASON` if the revision assertion is justified, or `--no-previous` if not. Inspect Candidate content and Cause records. Then individually remove the old exact record using `unlink design --target @OLD`. Neither step may leave an invalid root/Cause state. A whole-record replacement does not merge old messages automatically.

Read the complete resulting Candidate, compare against its fixed publication baseline and inspect the closure. Publish only the chosen one REF after your judgment; read the exact new Seal and its Cause back. No loop, script or retry substitutes for those judgments. Stop and reconcile any failure rather than automatically retrying publication.

## S4: deliberately retain a historical premise

Keep the exact historical target. Draft means provisional work; decided historical reliance uses ADR0043's explicit path.

```sh
sealgraph review prepare design --format json
sealgraph help review set
```

`review prepare` returns exact subject, other heads, complete targets and comparison leaves. It does not write a review or invent reasons. Read the dependent original/Candidate, each exact Cause, comparison versions and differences/impact, including transitive Causes. Author the bounded [historical-review/v1 record](storage-format.md#15-contained-historical-review-evidence) yourself, with retain or unreviewed and actual reasons.

Only then individually attach it with `review set design --file REVIEW_FILE`, read it back with `review show design --candidate --format human` and decide whether to issue `seal design --review-historical`. Unreviewed or draft Causes cannot be admitted that way. The normal strict publication path remains strict.

`inspect design` shows Candidate and HEAD claims separately. A recorded claim can remain visible while `applicable=false` after another REF changes. It is not deleted and stale is not erased. Use the displayed `review show` command to read differences, impact and reasons. Applicable is not complete preflight or approval. Human output may clip long reasons to the terminal width; use `review show design --candidate --format json` (or `review show @SUBJECT --format json`) for the complete recorded difference/impact/reason fields.

## S5: leave uncertainty pending

Do not change a Cause or issue a Seal when the original, relevant version, difference or impact is unconfirmed. Keep the existing Candidate/stale state and describe the exact missing evidence. There is no requirement to make the tool clean. `candidate discard` is only for an explicit decision to remove one unsealed Candidate; it does not recover semantic understanding.

## S6: independently audit the source

Start from an exact immutable Seal. Read its complete content and Cause records, then follow each selected exact Cause. If Trace is present:

```sh
sealgraph trace show --seal @SUBJECT --format json
sealgraph trace read --seal @SUBJECT --run-index 0 --view snapshot --format human
sealgraph trace read --seal @SUBJECT --run-index 0 --view current --format human
```

`run-index` selects one External run from the OriginMap. Snapshot means saved immutable full original; current means the explicitly bound current source. They are separate facts. Human context is escaped and bounded (default 3 context lines, 64KiB; configurable maximum 1MiB). JSON `excerpt_bytes` is base64, preserving arbitrary bytes, not plain text. Read `truncated`/`quote_truncated` before claiming complete review. These are display limits; underlying validated source reads still load the full Blob/file.

For a complete exact original, explicitly use `trace read ... --view snapshot --raw-source > original.bin`, choosing your own output destination. It exports the full source, not just the quoted substring. Current-source extraction is also explicit and may expose all current bytes. Raw mode has no JSON wrapper; it fails if current source observation is unavailable. Do not confuse the whole saved source with the dependent content or automatically publish extracted bytes.

A current match is one representative occurrence. To inspect all positions, use `trace occurrences` with the same exact baseline/run and `--view snapshot|current|both`; continue pages while `has_more=true`. Context changes invalidate a cursor. A changed offset does not prove editorial movement/history; missing exact bytes does not prove deletion or semantic contradiction.

Read `review show @SUBJECT` if a historical claim attachment exists. Use fsck only for structural integrity. Report separately: structural facts verified, byte correspondence observed, and your own semantic conclusion supported by the original. If Trace/claim is absent, say so. Do not manufacture provenance for history.

## LLM operating agreement

Use the same six paths and explain the beneficiary, exact subject, observed scope and uncertainty. Read-only inventory, hashing, extraction and review-material preparation may be automated. Cause updates, review judgments and individual Seal issuance require the executor's own target-specific judgment and separate operation. Never generate a script that pretends all reasons are reviewed, updates all Causes or issues Seals until stale=0.

Ask the owner only for unresolved material choices or missing authority. Use existing instructions and already accepted decisions without repetitive confirmation. A help topic, plan task, claim, tag, passing test or Seal does not itself authorize a later effect. Local guidance is `AGENTS.md`; no user-private Skill is needed to follow this guide.

This implementation's automated tests and agent role review do not establish how a first-time human user will behave. Actual novice trials and owner product acceptance remain separately reported evidence.

### 観測理由と比較基準を分けて読む

日常の入口は引き続き `sealgraph inspect`。理由を追う場合は `sealgraph observe REF --format human` を使う。`Observed successors` は登録済みの観測証拠であり、空でも「後継が実在しない」とは断定しない。`active` はHEADからRevision辺で到達する所属で、Causeで参照されるだけではactiveにならない。ACTIVE_NON_LEAFとOUTSIDE_ACTIVEの理由から、対象の特定版を `sealgraph show @ID` で読む。`paths_complete=false` なら表示経路の一部が省略されている。上限を上げても、診断を自動修復の指示に読み替えない。

Candidate=PRESENTは変更ありとは限らない。prospective/current HEADのSAME/DIFFERENT/NO_HEAD、公開時のexpected_ref_head、workfileのCandidate/HEAD content baselineを別々に読む。`candidate compare REF` は従来どおりexpected_ref_headとの比較である。HEADとCandidateのdraftも別の版の属性で、欠落をfalseにしない。Trace原文や履歴レビューは `inspect` / `trace read` / `review show` へ進む。`observe --inventory` は保管中の履歴・切離しSealも見る明示操作で、その存在だけでは現在の観測証拠にならない。
