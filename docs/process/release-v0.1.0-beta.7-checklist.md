# v0.1.0-beta.7 release checklist

Status: preparation candidate; tag and GitHub Release are pending a separate
publication decision. Release notes: [`release-v0.1.0-beta.7-notes.md`](release-v0.1.0-beta.7-notes.md).

## Scope

- Version: `0.1.0-beta.7`; standalone `sealgraph`, Linux amd64, MIT License.
- Changes since beta.6: format-5 observer-scoped Cause Link revisions and
  format-4 extract/load, format-6 Cause Link metadata, and format-7 Origin
  Trace through Accepted Issue #17 R4-c1, including both direct substring
  input and the existing recipe route.
- New repositories default to format 5. Format 5→6 and 5/6→7 migrations are
  explicit. The beta.6 format-4 upgrade route is read-only extraction followed
  by load into an absent format-5 target; no in-place format-4 upgrade.
- Package inventory: standalone binary, `LICENSE`, and `README.md` only;
  publish the archive and its SHA-256 checksum file.
- Original Issue #17 broader fragment STALE behavior, Git sidecar, attachment
  mutation, and format-7 migration of this tracked repository are separate work.

## Candidate source gate

- [ ] Commit runtime, tests, docs, CI beta.7 smoke setting, these notes, and
      this checklist; record the clean source SHA without changing its tree.
- [ ] On that SHA, run gofmt clean-tree check, `go vet ./...`, `go test ./...`,
      `go test -race ./...`, `npm ci`, completion check, clone check, complexity
      check, and dead-code check.
- [ ] Check the applicable Accepted requirement and ADR lineage against the
      notes and verify the version/format upgrade examples.
- [ ] Check `PLAN.pert` and the Issue #17 plan with document check, both
      schedules, and next-task selection; record any release-task gap.
- [ ] Build twice from the exact SHA into separate absent directories; compare
      archive and checksum bytes, verify the checksum, archive inventory,
      version string, and extracted-artifact smoke.
- [ ] Push the validated source SHA to its designated review ref and read it
      back; require successful GitHub Actions on that exact SHA before release.

Validated source SHA: pending.
Exact-source GitHub Actions run: pending.
Archive and checksum SHA-256: pending.

## Publication gate

Before publication, freeze the exact tag target, release-note text, two asset
names and digests, and one-write limits. After the owner authorizes that
record, create and read back the tag and prerelease once, then independently
verify the downloaded assets, installed binary, and final receipt.
