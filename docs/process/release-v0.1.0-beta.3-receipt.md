# v0.1.0-beta.3 release receipt

Status: released and independently read back on 2026-08-21.

## Frozen publication record

```text
release version: 0.1.0-beta.3
validated source SHA: 9c80756490b63baa4641f3b5680cfd6e9b065816
release metadata/tag SHA: 1c12f6eb7fc00dfb43da5f5bfa076db8264d98c8
artifact: sealgraph_0.1.0-beta.3_linux_amd64.tar.gz
artifact SHA-256: 4421d8627b892a829b4dc4011ab0f9faabcc0d53494544cd537a0f962e5921c8
checksums artifact: sealgraph_0.1.0-beta.3_checksums.txt
checksums file SHA-256: c768b430581b8492265b38f78972d98380319483d870f0214d02bfaeac551b50
release-note SHA-256: 2c1a84f4f6228986b389a02d3e2cee0e68da3ef8823e273839420ade114610bf
maximum tag writes: 1
maximum GitHub Release writes: 1
```

The operator explicitly approved this exact record. The immutable tag and
GitHub prerelease were each created once through the configured secdat
boundary; there was no retry, replacement, or force move.

## Validation

The clean validated source passed format, vet, unit/integration, race, clone,
complexity, dead-code, llmthink, and PERT checks. Three independent artifact
builds were byte-identical. The archive inventory was restricted to
`sealgraph`, `LICENSE`, and `README.md`, and extracted-artifact smoke passed.

GitHub Actions succeeded for both the validated source and the later
validation-metadata-only commit:

- source run: https://github.com/mako10k/sealgraph/actions/runs/32451570635
- metadata run: https://github.com/mako10k/sealgraph/actions/runs/32451703032

## Publication readback

```text
remote tag v0.1.0-beta.3: 1c12f6eb7fc00dfb43da5f5bfa076db8264d98c8
prerelease: true
draft: false
published_at: 2026-08-21T05:47:29Z
archive size: 1405224
checksums size: 108
```

Release URL:

https://github.com/mako10k/sealgraph/releases/tag/v0.1.0-beta.3

The GitHub asset API reported the frozen SHA-256 digest for each asset. Both
assets were downloaded into a fresh temporary directory and independently
hashed. The downloaded archive passed `scripts/artifact-smoke.sh` for version
`0.1.0-beta.3`.

This receipt-only commit does not alter released inputs and does not move the
tag. Local source bindings remain non-canonical; the release adds no watcher,
automatic add/seal, binding-history restore, Git discovery, or Git sidecar.
