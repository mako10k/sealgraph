# SealGraph v0.1.0-beta.7

Standalone Linux amd64 prerelease. These notes describe the changes since
v0.1.0-beta.6.

## What's new

- Format 5 makes Cause Link revision assertions observer-scoped and adds an
  isolated, read-only format-4 extractor. New repositories initialize as
  format 5.
- Format 6 adds bounded, namespaced metadata on exact Cause Links. Upgrade a
  validated format-5 repository explicitly with
  `sealgraph migrate repository --from 5 --to 6`.
- Format 7 adds Origin Trace: immutable full-source snapshots, byte-range
  origin maps, exact presence comparison, optional changed-range estimates,
  paged occurrence positions, local source bindings, and native snapshot
  dump/load. Upgrade a validated format-5 or format-6 repository explicitly
  with `sealgraph migrate repository --from 5 --to 7` or `--from 6 --to 7`.
- `trace set` accepts a source file and selected substring directly through
  `--source-file PATH --source-key KEY --content STRING` or
  `--content-file PATH|-`. The existing `--recipe PATH` route remains available.

## Upgrading from beta.6

Beta.6 format-4 repositories need an explicit transfer into a new format-5
repository. With the beta.7 binary in the format-4 source repository, run
`sealgraph migrate extract --source-format 4 --format universal-blob-v1`
and save its output. In a different directory without `.sealgraph`, run
`sealgraph load --format universal-blob-v1` with that document on stdin.
Keep the original repository until the new repository and the old-to-new
identity receipt have been inspected. Ordinary beta.7 operations do not open
format 4 as live state.

Format-6 and format-7 migrations change the repository config after validation;
they do not rewrite retained records or create historical Origin Trace claims.

## Package

The Linux amd64 archive contains `sealgraph`, `LICENSE`, and `README.md`.
The release also provides a SHA-256 checksum file. Git sidecar and attachment
mutation are outside this standalone release.
