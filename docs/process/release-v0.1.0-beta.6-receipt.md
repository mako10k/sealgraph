# v0.1.0-beta.6 release receipt

Recorded: 2026-08-24.

## Identity

```text
release version: 0.1.0-beta.6
validated source SHA: b1166444172e0f257cdd09bd14216ca67b505c15
tag and publication SHA: 13d94fa92c89cd2d29890307e11051a2e813120b
exact-source GitHub Actions run: 32711251818
publication GitHub Actions run: 32711506731
artifact: sealgraph_0.1.0-beta.6_linux_amd64.tar.gz
artifact size: 1489037
artifact SHA-256: 1cf4c8e71e16b3783c7a45275eee25c21dede94211f299ccc2485b9ae404189d
checksums artifact: sealgraph_0.1.0-beta.6_checksums.txt
checksums size: 108
checksums file SHA-256: 89bc89ca453c7ea8454f25b43483a6a1e22fc4088faf6d50e00cdf71d1667766
release-note SHA-256: fa21c89e8b022c35fab53e50e39bced9335b3b3e6cf5a80bcd9340ce28f10115
installed binary SHA-256: 78f81cd6453da6abade03aa406db56cdc981d50c17d46a3f49a8df0ad99e4ff1
```

## Publication and readback

- The lightweight `v0.1.0-beta.6` tag was created once and its local and remote
  values both resolved to the publication SHA above.
- Exact source run `32711251818` and publication run `32711506731` completed
  successfully, including race, static-analysis, completion, and release
  artifact smoke gates.
- One non-draft prerelease was created at
  <https://github.com/mako10k/sealgraph/releases/tag/v0.1.0-beta.6>.
- The prerelease contained exactly the approved tar archive and checksum file.
  GitHub-reported digests and independently downloaded SHA-256 values matched
  the frozen publication record and the locally reproduced artifacts.
- The downloaded archive passed its checksum file, inventory check, and
  extracted-artifact smoke test.
- The downloaded binary was installed at
  `/home/katsumata-m/.local/bin/sealgraph`; it reports
  `sealgraph 0.1.0-beta.6`, and its installed hash matches the extracted binary.
- Bash completion at
  `/home/katsumata-m/.local/share/bash-completion/completions/sealgraph`
  matches the release source wrapper.

No tag was moved, no GitHub Release write was retried, and no Git-sidecar
artifact was published.
