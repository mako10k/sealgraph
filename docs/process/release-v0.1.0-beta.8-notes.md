# SealGraph v0.1.0-beta.8 — unpublished candidate

Candidate source: `f75b43ef3a38ca1eb8f73a7220251db8a487a2b4`.
Standalone Linux amd64. These proposed notes cover changes since published
`v0.1.0-beta.7`. No beta.8 tag or GitHub Release has been created.

## Changes

- Individual historical-premise review: `review prepare`, `review set`,
  `review show`, `review clear` and explicit `seal REF --review-historical`.
  Exact-version reliance can be recorded without erasing stale observations.
  Every reachable Cause requires a version-bound judgment; unreviewed or draft
  Causes block this non-draft publication path. Default `seal REF` remains strict.
- On-demand `inspect` separates registered workfile/baseline differences,
  immutable provenance stale, original-source Trace observations and historical
  review claims. It reports omissions and unconfirmed scope; it is not a watcher
  or background notification service.
- `compare FROM TO --text` adds bounded, escaped human text differences.
  `trace read` reads an explicit saved/current original-source view for one
  External run, with exact extraction available through explicit raw mode.
  A byte match does not certify semantic support, correctness or approval.
- New read-only `observe` separates exact revision evidence, active membership,
  review reasons/Cause paths, Candidate identity versus current HEAD,
  expected publication baseline, per-version draft and workfile baseline.
  Bounded path omission is explicit. No observed successor is not proof of
  nonexistence; outside-active alone does not distinguish historical/detached.
- Workflow help connects registration, changes, adoption, historical retention,
  pending work and original-source audit. `help review set` includes a complete
  blank JSON layout and the mapping from preparation facts. Its placeholders
  and empty decision deliberately fail validation; it generates no judgments.

## Compatibility and operation

New repositories still initialize as format 5. Existing formats 5/6/7 remain
supported, with explicit migration for format6/7 features. Trace requires
format7. Format4 still requires its explicit read-only extract/load transfer;
ordinary commands do not silently migrate it. This candidate introduces no new
canonical storage fields or migration.

Existing status/stale/graph/fsck output contracts, stale membership/frontier/
refs-only, bytes-only extraction and normal Seal conditions retain their
meaning. New commands own versioned JSON schemas. Candidate compare keeps its
expected_ref_head baseline; it is distinct from observe's current-HEAD identity
comparison. Historical-review records are executor claims, not approval.

Start with `sealgraph help workflows`, `sealgraph help inspect`,
`sealgraph help observe` and `sealgraph help review set`. Full guides are in the
[source snapshot](https://github.com/mako10k/sealgraph/tree/f75b43ef3a38ca1eb8f73a7220251db8a487a2b4/docs).
The archive retains the established inventory: binary, README and MIT LICENSE.
Bash completion is available in source; it is not included in this archive.

## Scope and known limits

Standalone only. Git sidecar, draft pointers/cascade/atomic multi-pointer
publication and the proposed hyperedge model are not included. No automatic
relink, stale repair, Seal issuance, Git detection or judgment generation.

Human text/context output is bounded; inspect omission/truncation fields and
use explicit extraction for complete originals. Observe scans retained objects
and may traverse many Cause paths; path limits bound witnesses, not total cost.
No large-repository performance guarantee or simultaneous filesystem snapshot
is claimed. Independent Agent trials cover bounded synthetic scenarios,
including sibling leaves and four Cause hops; actual novice-human trials,
platforms other than Linux amd64 and broader stress/concurrency scenarios have
not been established by those trials. Automated regression/race tests are
separate evidence, not a proof of all runtime behavior.

This candidate does not publish, install or upgrade any user's repository.
Publication requires the owner's later decision on the exact source, archive,
checksums and these notes recorded in the candidate manifest.
