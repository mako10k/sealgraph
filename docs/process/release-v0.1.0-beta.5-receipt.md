# v0.1.0-beta.5 release receipt

Recorded: 2026-08-21.

## Identity

```text
release version: 0.1.0-beta.5
validated source SHA: 6cfd6cfde869366e65fffb1e8c3d48cd22d0fafb
tag and publication SHA: 360408970d815db8da4bebb90abdaae0686b56bd
GitHub Actions run: 32468221350
artifact: sealgraph_0.1.0-beta.5_linux_amd64.tar.gz
artifact size: 1453367
artifact SHA-256: 45e2832b67a62d2302c062f9a54826b0df3805ab49e34a3992d97712838f97f1
checksums artifact: sealgraph_0.1.0-beta.5_checksums.txt
checksums size: 108
checksums file SHA-256: 8540ea3ae680b81b070f28e004d266263128ee28faff29b9c000c6c9a3b9fa5f
release-note SHA-256: 4957b6c7c1f797afa38275b43dd97d2740d57ec5570c501ef9340972466a7747
```

## Publication and readback

- The lightweight `v0.1.0-beta.5` tag was created once and its local and remote
  values both resolved to the publication SHA above.
- GitHub Actions run `32468221350` completed successfully for that SHA,
  including race, static-analysis, and release-artifact smoke gates.
- One non-draft prerelease was created at
  <https://github.com/mako10k/sealgraph/releases/tag/v0.1.0-beta.5>.
- The prerelease contained exactly the approved tar archive and checksum file.
  GitHub-reported digests and independently downloaded SHA-256 values matched
  the frozen publication record.
- The downloaded archive passed its checksum file and extracted-artifact smoke.
- Local installation was not requested and was not performed.

No tag was moved, no release write was retried, and no Git-sidecar artifact was
published.
