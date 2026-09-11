# Format 6 normative synchronization independent review r2 — 2026-09-11

Status: completed

This is the fresh Step 3 independent review of the revised snapshot routed by
the Operator as `REVIEW` without additional questions. It reviews only the
exact candidate and review-input bytes identified below.

## Reviewed snapshot

- `docs/requirements.md`
  - SHA-256: `828bf756da52243d7c5ac5c8a87917820a471574ab16e62353543851a956c6e4`
- `docs/architecture.md`
  - SHA-256: `18352065ba6874691516c9fbe5d0cf51a618618078d0fe188f0b4fe0ee3cdab1`
- `docs/storage-format.md`
  - SHA-256: `c01fd58c8d655dfc248c20fb92d41078b5faf1bc7a799d7d5b441dea1a6d6f88`
- `docs/cli.md`
  - SHA-256: `95225e234c0bca875cc8b34c2a9d4c748888fa0e6df219c46249c81c840cd3ad`
- Review input: `docs/process/format6-normative-sync-review-ja-2026-09-11.md`
  - SHA-256: `1d212d69cb37f0ae4770a78192264777cd5c5f61bda8e797124e006e3c187e33`

All five pre-review digests matched the owner-reviewed identities. They were
recomputed after review and remained unchanged.

## Authority and method

The authoritative sources were Accepted ADR 0029, Accepted ADR 0030, Accepted
ADR 0031, and ADR 0029's accepted revision-2 requirement authority. Accepted
format-5 ADRs 0023, 0025, and 0027 were used to distinguish preserved format-5
contracts from their format-6 successors.

The review independently checked:

- every stated acceptance criterion in the review input;
- the disposition of prior findings F-01 through F-04 against the accepted
  sources and revised candidate bytes;
- generic metadata semantics, canonical bounds and identity, graph
  independence, exact typed-schema pairing, historical format-5 readability,
  Candidate-v5 projection/publication, config-only migration, human/machine
  inspection, and `upstream-change/v2`;
- format-5 compatibility and old-reader failure boundaries;
- optional/future and out-of-scope containment;
- metadata namespace/schema ownership; and
- the eight concrete parent/leaf help-navigation routes.

Existing implementation and tests were used only as non-normative comparison
evidence. A temporary binary was invoked outside a repository for all eight
documented help routes; all succeeded and no `.sealgraph` directory was
created. Focused tests for `internal/cli`, `internal/repository`, and
`internal/canonical/v6` passed.

Before relying on the disposition, the reasoning was audited through the
command-line `llmthink dsl audit` as
`format6-normative-sync-step3-r2-2026-09-11`. The relied-upon result was
`fatal=0`, `error=0`, `warning=0`, `hint=0`.

## Disposition of prior findings

### F-01 — resolved

The selector contract now requires `@SEAL_TOKEN` to decode as a canonical Seal
generation permitted by the current repository format. `docs/requirements.md`
also states the exact boundary: format 5 accepts Seal v5 only, while format 6
accepts valid Seal v5 and Seal v6 objects with their required Provenance
pairing. This matches Accepted ADR 0030's mixed-generation contract and no
longer excludes format-6 Seals.

### F-02 — resolved

The candidate now records the Assessment-free `sealgraph/upstream-change/v2`
schema in requirements, architecture, storage-format, and CLI documentation.
It preserves the accepted member order, complete metadata-bearing
`after_cause_links`, metadata-sensitive `change_id`, immutable v1/Assessment
Blobs, and the separately gated Assessment-reference boundary.

### F-03 — resolved

The candidate now distinguishes side-effect-free Candidate-v5 reading,
Candidate-file upgrade only through an authorized mutation, and direct sealing
through an exact in-memory Candidate-v6 projection. It explicitly forbids a
hidden preliminary Candidate-file rewrite and creates Provenance v2/Seal v6
before the existing coherent-observation and one-REF CAS workflow.

### F-04 — resolved

The review input now explicitly enumerates seven acceptance criteria. They cover
source-authority consistency, format-5 compatibility, metadata and graph
semantics, selector generations, `upstream-change/v2`, Candidate-v5
publication, help navigation, and excluded scope.

## Classified findings

### Contradiction — none — P0 through P3

No material contradiction with the reviewed requirement or Accepted ADRs 0029,
0030, and 0031 was found.

### Evidence gap or unresolved unknown — none — P0 through P3

No material mandatory-contract or review-evidence gap remains in the exact
revised snapshot. Acceptance itself remains the explicit Step 4 owner decision;
that lifecycle state is not a candidate defect.

### Optional or future candidate — none promoted — P0 through P3

Metadata query/filter/AST/index work, schema registries and validators,
core-owned revision-context namespaces, and Assessment-reference commands were
not promoted into acceptance criteria. Existing future Git-sidecar constraints
do not promise implementation in this candidate.

### Out of scope — contained — P0 through P3

Release, actual repository migration, commit, push, deployment, Git-sidecar
implementation, format 7, existing format-4 migration changes, and ADR 0028
Assessment adoption/storage remain outside this review. No evidence from these
areas was used as requirement authority.

## Acceptance-criteria results and passed areas

1. The four normative documents are consistent with Accepted ADRs 0029–0031
   and preserve the accepted format-5 contract.
2. Generic metadata, canonical limits, identity commitment, graph independence,
   v5/v6 exact pairing, and explicit config-only migration are synchronized.
3. `@SEAL_TOKEN` resolves every canonical Seal generation permitted by the
   current repository format and does not permit Seal v6 in format 5.
4. Assessment-free `upstream-change/v2` commits complete format-6 Cause Links,
   reflects metadata changes in `change_id`, and preserves existing v1 and
   Assessment Blobs.
5. Candidate-v5 reading in format 6 is side-effect free; only mutation upgrades
   its file, while publication uses an in-memory v6 projection without a
   preliminary file rewrite.
6. All eight documented format-6 help-navigation routes succeed without
   repository bootstrap or mutation.
7. Query, registry, validator, Assessment-reference, release, actual migration,
   and Git-sidecar implementation remain optional/future or out of scope.

Additional passed checks:

- Metadata remains embedded in the containing Cause Link and Provenance; no
  CauseLinkID or independent Link publication lifecycle is introduced.
- Legacy `messages` remain independently usable and are neither parsed nor
  migrated into structured metadata.
- Unknown namespaces remain inspectable but do not receive truth, trust,
  approval, validation, or graph meaning from core.
- Historical format-5 object bytes and IDs remain unchanged, and projected
  empty metadata is distinguishable from stored v2 emptiness.
- Format-6 legacy `add` and `link` preserve unmentioned metadata, while explicit
  whole-Link deletion continues to remove it.
- Changed machine records use successor schemas while unchanged `status/v3`
  and `stale/v2` retain their accepted meanings.
- No Sealgraph-owned metadata namespace or external schema ownership was
  assigned.

## Step 3 disposition

The independent review is `completed`. No material finding prevents the
unchanged revised snapshot from proceeding under the selected `REVIEW` route to
Step 4. Only the Operator may choose `REVISE`, `REREVIEW`, or `ACCEPT` there.

This review does not accept the requirement and does not authorize a candidate
edit, ADR change, implementation change, commit, push, PR, merge, release,
deployment, repository migration, or runtime mutation.
