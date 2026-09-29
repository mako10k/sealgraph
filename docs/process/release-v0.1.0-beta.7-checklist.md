# v0.1.0-beta.7 release checklist

Status: artifact and release-note identities recorded for publication review on
2026-09-28. Release notes: [`release-v0.1.0-beta.7-notes.md`](release-v0.1.0-beta.7-notes.md).

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

- [x] Freeze the clean code and artifact source commit with runtime, tests,
      docs, and CI beta.7 smoke setting; record its SHA without changing its tree.
- [x] On that SHA, run gofmt clean-tree check, `go vet ./...`, `go test ./...`,
      `go test -race ./...`, `npm ci`, completion check, clone check, complexity
      check, and dead-code check.
- [x] Check the applicable Accepted requirement and ADR lineage against the
      notes and verify the version/format upgrade examples.
- [x] Check `PLAN.pert` and the Issue #17 plan with document check, both
      schedules, and next-task selection; record any release-task gap.
- [x] Build twice from the exact SHA into separate absent directories; compare
      archive and checksum bytes, verify the checksum, archive inventory,
      version string, and extracted-artifact smoke.
- [x] Push the validated source SHA to its designated review ref and read it
      back; require successful GitHub Actions on that exact SHA before release.

Validated source SHA: `04fa95e057ee7d69dc041ede61a7ff282e65aa01`.
Merged main SHA: `f79c8af39882558548d3b66d3f557483771b950b`;
its tree and beta.7 artifact bytes match the validated source.
Exact-source GitHub Actions runs: push `36416266969`, PR `36417704496`;
post-merge main run `36418314491` also succeeded.
The two checked PERT plans have no remaining scheduled tasks; beta.7
publication is a separate, unscheduled gate.

## Publication record candidate

```text
release version: 0.1.0-beta.7
validated source SHA: 04fa95e057ee7d69dc041ede61a7ff282e65aa01
merged main SHA: f79c8af39882558548d3b66d3f557483771b950b
artifact: sealgraph_0.1.0-beta.7_linux_amd64.tar.gz
artifact size: 2133113
artifact SHA-256: a500bdc286b3a3acfd3aa6559e768be0c2fa375285ff163c3f5bf63e1c25fca5
checksums artifact: sealgraph_0.1.0-beta.7_checksums.txt
checksums size: 108
checksums file SHA-256: 3b983091c47cc325350767669a3c1ac1ce4347b656c847a3ef88c64a0627a37d
release notes: docs/process/release-v0.1.0-beta.7-notes.md
release-note SHA-256: 019632b22472e5494b52bd7fd18feb96bd3b540935dd7be6583730e98ebf7ee2
maximum tag writes: 1
maximum GitHub Release writes: 1
```

The archive contains only the standalone binary, `LICENSE`, and `README.md`.
Its checksum file verifies that archive. A build from merged main produced
identical archive and checksum bytes to both builds from the validated source.

The tag target and publication authority are the next decision. Once fixed,
create and read back the tag and prerelease, then verify downloaded assets,
installation, and the final receipt against this record.
