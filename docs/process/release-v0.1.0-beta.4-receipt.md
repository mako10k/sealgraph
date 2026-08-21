# v0.1.0-beta.4 release receipt

Recorded: 2026-08-21.

## Identity

```text
release version: 0.1.0-beta.4
validated source SHA: 8e7ca4bc9a20261cd1f09447849b9bd6ba9f796d
tag and publication SHA: 3284c3259576ed281e5af412f5a84dd2a3e93b94
GitHub Actions run: 32462204170
artifact: sealgraph_0.1.0-beta.4_linux_amd64.tar.gz
artifact SHA-256: eff9136499c24e71ec909448591f0c4ae268afd157cc4aaf58844407b253e1b0
checksums artifact: sealgraph_0.1.0-beta.4_checksums.txt
checksums file SHA-256: 481aa74c040eedd89d31bbd86a5015da80d37b153f741db87a51757c70c81c71
release-note SHA-256: cae36b1cc9187effca5cac97e8bd578d5f31748fb0602ec86bfc832a7a0e65ab
installed binary SHA-256: e1769fe2758093c47f30b3be12306dd00eff56cb4adb7bbaf2aecce56aa35dd5
```

## Publication and readback

- The lightweight `v0.1.0-beta.4` tag was created once and its local and remote
  values both resolved to the publication SHA above.
- GitHub Actions run `32462204170` completed successfully for that SHA,
  including race, static-analysis, completion, and release-artifact smoke gates.
- One non-draft prerelease was created at
  <https://github.com/mako10k/sealgraph/releases/tag/v0.1.0-beta.4>.
- The prerelease contained exactly the approved tar archive and checksum file.
  GitHub-reported digests and independently downloaded SHA-256 values matched
  the frozen publication record.
- The downloaded archive passed its checksum file and extracted-artifact smoke.
- The downloaded binary was installed at
  `/home/katsumata-m/.local/bin/sealgraph`; it reports
  `sealgraph 0.1.0-beta.4`, and its installed hash matches the extracted binary.
- Bash completion remains installed at
  `/home/katsumata-m/.local/share/bash-completion/completions/sealgraph`.

No tag was moved, no release write was retried, and no Git-sidecar artifact was
published.
