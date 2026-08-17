# v0.1.0-beta.2 release receipt

Status: source and artifacts validated; publication evidence is appended only
after independently reading back the one-time external writes.

## Frozen publication record

```text
release version: 0.1.0-beta.2
validated source SHA: 2f92400e93d64b12b8a371d346ca1b299b3a7fdb
artifact: sealgraph_0.1.0-beta.2_linux_amd64.tar.gz
artifact SHA-256: cff205d0760d3db467dfe0dcd6230fac5de75bf40151b03c1d1ebdef1a079c43
checksums artifact: sealgraph_0.1.0-beta.2_checksums.txt
checksums file SHA-256: 6db5c2de167111157e943fa40e73d293d5c7fda64d0a44c36087ca4e4ba94a1b
release-note SHA-256: 380e006fe56e9f99fa005e55a157ef58fb169db4286cce278b7eee43dc01e039
maximum tag writes: 1
maximum GitHub Release writes: 1
```

The release request explicitly authorized publication. The existing
`v0.1.0-beta.1` tag remains immutable; beta.2 uses a new version and tag.

## Validation

The clean source SHA passed:

- `gofmt` clean-tree check;
- `go vet ./...`, `go test ./...`, and `go test -race ./...`;
- `npm ci` and clone detection with zero clones;
- cyclomatic-complexity and whole-program dead-code checks;
- both required llmthink audits with no fatal/error/warning result;
- `perttool document check PLAN.pert` and `perttool dag analyze PLAN.pert`;
- deterministic beta artifact build and extracted-artifact smoke;
- a second independent artifact build with byte-identical archive/checksum
  output;
- archive inventory restricted to `sealgraph`, `LICENSE`, and `README.md`.

GitHub Actions run 32001221506 succeeded for the validated source SHA:

https://github.com/mako10k/sealgraph/actions/runs/32001221506

## Identity boundary

The validated source SHA contains all runtime, test, help, documentation,
version, workflow, and release-note inputs. This receipt and the finalized
`VALIDATION.json` form a later provenance-metadata-only commit. Neither file is
compiled or included in the release archive. The archive is rebuilt after that
metadata commit and must remain byte-identical before tagging.

## Publication readback

Pending one-time tag and GitHub prerelease creation.
