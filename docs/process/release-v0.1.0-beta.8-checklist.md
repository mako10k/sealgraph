# v0.1.0-beta.8 candidate preparation — unpublished

Status: prepared locally for owner review, 2026-11-04. This is not publication,
release acceptance or installation. User instruction: prepare the candidate
bundle and verification record; do not publish yet.

## Frozen identities and scope

- Candidate label: `0.1.0-beta.8`, proposed next standalone beta after the
  independently read-back beta.7 prerelease (published 2026-09-29).
- Code/artifact source: `f75b43ef3a38ca1eb8f73a7220251db8a487a2b4`.
- Source tree: `f7cf2647fd5892421bf8aaf4ddfbdd864e11c562`.
- Base release tag: `v0.1.0-beta.7`, commit
  `21195bc273c97c751a0f91085c5b6c7482e7ac9a`.
- Clean `git archive` export used for build and local checks; the export digest
  is in evidence/source-export.json. Preparation records and PERT updates are
  on local branch `codex/release-beta8-prep`, outside the frozen artifact source.
- Standalone Linux amd64 only; static stripped binary, README.md, MIT LICENSE.
  Archive inventory is unchanged from beta.7. Completion/full guides remain in
  source; critical workflow/observe/review layout help is embedded in the binary.
- New init format5; formats5/6/7 supported. No new persisted format or implicit
  migration. No Git sidecar, automatic repair/relink/sealing or notification daemon.

## Source validation

All following checks succeeded on the frozen export, not merely the preparation
branch's document tree. Full stdout/stderr logs are in the local evidence bundle.

- [x] `gofmt -l .`: empty.
- [x] `go vet ./...`.
- [x] `go test ./...`.
- [x] `go test -race ./...`.
- [x] `npm ci` (0 vulnerabilities).
- [x] `npm run clone-check`.
- [x] `make complexity-check deadcode-check completion-check`.
- [x] Exact main-source CI successful:
  [41787911019](https://github.com/mako10k/sealgraph/actions/runs/41787911019),
  head f75b43e; retrieved completed/success. CI uses Go1.26.x/Node24;
  local frozen-source checks used Go1.27.3/Node25.1.0 on WSL2 Linux x86_64.
  Cross-toolchain byte reproducibility is not claimed.
- [x] PERT document check, both schedules and next-task inspection for preparation.

## Artifact validation

- [x] Existing build-release.sh run twice from the same frozen export into
  separate output directories. Archive and checksum-file bytes match exactly.
- [x] SHA-256 checksum file verifies the archive.
- [x] Exact expected tar members only; regular files/directory, no symlinks or
  Git-sidecar executable. README/LICENSE match the frozen source byte-for-byte.
- [x] Extracted executable is static Linux x86_64 and reports
  `sealgraph 0.1.0-beta.8`.
- [x] Embedded workflows/inspect/observe/review-set help can be read without
  opening a repository. Review example supplies no judgment and remains invalid
  until manually completed.
- [x] Manual artifact smoke: fresh format5 init; individually read original,
  Candidate, exact Cause/differences/impact, record judgment, issue exactly one
  REF and read back. Three separate Seal operations: premise P1, dependent D1,
  updated premise P2. No issuance/link-update script or loop used.
- [x] After enabled→disabled premise update, dependent's exact old Cause is kept
  pending review. Frontier/refs-only returns dependent; observe shows
  OUTSIDE_ACTIVE, not invented successor evidence. Inspect/compare-text work.
- [x] Artifact reads historical-review state and saved Trace from the earlier
  synthetic format7 fixture, without new review judgments or issuance.
- [x] Intact smoke fsck succeeds; in a separate copied fixture, deliberate
  corruption of exact P1 object produces fsck exit1. Original fixture untouched.

The existing `artifact-smoke.sh` was NOT invoked locally because it scripts
Seal issuance. The above individual operations exercise its version/init/
dependency-update/frontier/fsck/corruption outcomes while preserving the owner's
individual judgment rule. This is recorded substitution, not a claim that the
script was executed. Standard Go regression tests are separate test-suite evidence.

## Recorded artifact identities

```text
archive: sealgraph_0.1.0-beta.8_linux_amd64.tar.gz
size: 2205801 bytes
SHA-256: 7ef439d354081a6b327b57defcff3664cf475be8c8c53a4de5704c26cf4a214c
checksum file: sealgraph_0.1.0-beta.8_checksums.txt
size: 108 bytes
SHA-256: dc5383731034a21d20e22f9b79222bac2d0d402d41c1c7e085ef312dd4fc9ead
```

Candidate bundle: `dist/release-candidate-0.1.0-beta.8-20261104/` (ignored build
output, local only). It includes archive, checksum file, proposed notes, this
verification record, manifest and evidence logs. Canonical preparation records
are this checklist, release-v0.1.0-beta.8-notes.md and the candidate manifest
under docs/process. The source and notes are distinct fixed artifacts.

## Limits and next owner decision

Local smoke is a bounded synthetic Linux/WSL2 case. Automated/race tests and
prior independent Agent trials do not prove real-user understanding, production
scale, every concurrency case or all platforms. Performance and human novice
claims remain unestablished. Candidate preparation changes no consumer's install
or real repository. No signing/provenance attestation beyond the recorded
source/bytes/check results is claimed.

Publication remains pending: the owner must select the exact source, candidate
version, archive/checksums and notes before any tag/Release/asset upload. Final
remote readback still showed main=f75b43e, no beta.8 tag and latest published
prerelease beta.7. No tag, draft Release, asset upload, install, branch push or
other remote write was performed during this preparation. No artifact signature
or product acceptance was inferred from hashes, tests or Seals.
