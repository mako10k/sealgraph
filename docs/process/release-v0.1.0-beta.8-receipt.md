# v0.1.0-beta.8 publication and readback receipt

Published at **2026-11-04T07:24:45Z** (16:24:45 JST), following the owner's
explicit acceptance of the candidate bundle and verification and direction to
publish. Release: https://github.com/mako10k/sealgraph/releases/tag/v0.1.0-beta.8
Release ID: `433620465`. Readback: draft=false, prerelease=true. Creation and
publication requested latest=false.

## Exact source and payload

- Lightweight tag `v0.1.0-beta.8` points to
  `f75b43ef3a38ca1eb8f73a7220251db8a487a2b4`; both `git ls-remote` and the GitHub
  tag-ref API independently returned that commit (not the preparation HEAD).
- Source tree: `f7cf2647fd5892421bf8aaf4ddfbdd864e11c562`.
- Accepted candidate manifest SHA-256:
  `501e4c92ec5283ef2a8633a5bf226f1062120027933beb162f0cae7a194f4ea5`.
- Published notes are docs/releases/v0.1.0-beta.8.md, SHA-256
  `b1965525d84952fe4f168b9677b8f6bb8908afc023c17fa07e73368f062d3121`.
  The Release body was read back byte-for-byte. Changes from candidate notes
  concern publication/status/owner-approval wording only; features and limits
  are unchanged.

| Asset | ID | Bytes | SHA-256 |
| --- | --- | --- | --- |
| sealgraph_0.1.0-beta.8_linux_amd64.tar.gz | 708634359 | 2205801 | 7ef439d354081a6b327b57defcff3664cf475be8c8c53a4de5704c26cf4a214c |
| sealgraph_0.1.0-beta.8_checksums.txt | 708634361 | 108 | dc5383731034a21d20e22f9b79222bac2d0d402d41c1c7e085ef312dd4fc9ead |

## Execution and verification

Root was the sole writer. Local tag was created once at the explicit frozen
commit, pushed once and read back. One draft Release was created, two assets
uploaded once without clobber, and draft metadata/body/assets were downloaded
and verified before one publish transition. No ambiguous result or retry,
force update, asset replacement or release deletion occurred.

After publication, Release metadata, tag-ref API and ls-remote were read again.
The two assets were downloaded into a fresh public-download directory, both
matched the accepted candidate bytes and GitHub API asset digests, and the
checksum file verified the archive. Asset state is uploaded. The downloaded
archive was extracted, its executable reported `sealgraph 0.1.0-beta.8`, and
embedded observe help exposed observation/v1. No application installation,
new Seal/Cause operation, main merge or branch push was performed.

Before publication, an independent read-only Agent verified source/artifact
identities, safe package inventory, executable version, public docs, notes diff
and the bounded absence of common credential signatures, reporting no blocker.
This review, source CI and tests do not certify semantic truth or all-platform
behavior. Prior source/race/manual-smoke verification remains bound to the same
uploaded bytes; it was not claimed as a new post-publication full test run.

Machine-readable receipt: release-v0.1.0-beta.8-receipt.json. Local raw evidence:
`dist/release-v0.1.0-beta.8-publication-20261104/` (draft/public metadata, downloaded
assets, checksum results and version/help reads). Candidate manifest/checklist
retain their earlier prepared_unpublished/pending snapshot unchanged; this
receipt records the later accepted publication and readback state.
