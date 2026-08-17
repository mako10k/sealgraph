# Standalone alpha release checklist

Status: release-blocked by the accepted ADR 0011 format-4 implementation and
explicit dogfood conversion. Completing this document does not authorize a tag,
GitHub Release, package publication, or Git-sidecar release. Publication
requires a separate explicit operator approval after an exact commit SHA and
artifacts are frozen.

## 1. Release identity and scope

- Target: `v0.1.0-alpha.1`.
- Product surface: standalone `sealgraph` only.
- Repository format: native format 4. The tracked project dogfood remains
  format 3 and is not runtime-compatible until explicit conversion.
- Compatibility policy: formats 1 through 3 remain unsupported by the
  format-4 runtime; pre-1.0 breaking
  regeneration is allowed and preferred over compatibility scaffolding.
- Initial binary artifact scope: Linux amd64 only unless another platform is
  separately built and tested before the freeze.
- `git-sealgraph` is a source-tree placeholder and MUST NOT be included in the
  standalone alpha artifacts or described as implemented.
- The release does not imply Git integration, truth, trust, signatures,
  automatic repair, migration, remote storage, or server support.

## 2. Product blockers before freeze

- [x] Add the deterministic read-only format-3 logical dump.
- [x] Implement format-4 canonical Seal/candidate bytes, fixtures, and
      empty-repository load with complete ID mapping.
- [ ] Implement and validate active revision DAG, same-material sibling,
      active-leaf admission, stale cache/`--scan`, exact-Cause frontier, and
      bounded impact.
- [ ] Resolve SG-BL-010's REF-scoped tag loose-path collision with an explicit
      pre-1.0 storage decision and tests.
- [ ] Explicitly convert tracked dogfood through dump/load after the tag
      contract and verify there is no tag loss, mixed-format state, or partial
      owner-check behavior.
- [ ] Complete SG-BL-002's deterministic explicit-path manifest builder.
- [ ] Complete remaining SG-BL-001 acceptance coverage for unsafe file kinds
      and prove failure before candidate mutation.
- [ ] Complete SG-BL-005's compact operator semantic legend.
- [ ] Complete SG-BL-007's distinct initialized/runtime-bootstrap/already-ready
      result reporting.
- [ ] Implement the read-only full-inventory `fsck` slice from SG-BL-009.
- [ ] Run a controlled recurring standalone dogfood from a fresh checkout,
      including manifest construction, one-REF sealing, stale frontier,
      sequential downstream repair, history/diff inspection, and `fsck`.
- [ ] Reconcile implemented backlog Issues with executable acceptance evidence;
      do not close partially implemented Issues merely to make the release look
      complete.

The cross-command versioned JSON contract (SG-BL-006), multi-edge link-message
ergonomics (SG-BL-011), attachments, and Git sidecar may remain open for this
alpha. Recurring alpha evidence may be collected manually using bounded human
output plus the stable `stale --refs-only` stream; automation must wait for a
versioned structured format.

## 3. Release engineering blockers

- [ ] Select and commit an explicit LICENSE. Do not infer a license from public
      repository visibility.
- [ ] Replace the hard-coded `0.1.0-dev` release value with deterministic build
      metadata injection while keeping an honest development fallback.
- [ ] Document source build and Linux amd64 installation/uninstallation.
- [ ] Add concise release notes covering format 4, explicit dump/load,
      breaking-regeneration policy,
      standalone-only scope, supported platform, and known omissions.
- [ ] Add a deterministic artifact build that produces:

  ```text
  sealgraph_0.1.0-alpha.1_linux_amd64.tar.gz
  sealgraph_0.1.0-alpha.1_checksums.txt
  ```

- [ ] Ensure the archive contains only the standalone binary plus approved
      documentation/license files; it must not contain `.sealgraph`, candidates,
      secrets, local paths, `git-sealgraph`, or generated development state.
- [ ] Refresh `project.json` and `VALIDATION.json` so they describe the actual
      alpha scope and exact validation performed.
- [ ] Add an artifact smoke test that checks `--version`, standalone init,
      root/dependent seal, stale frontier, and corruption refusal in a temporary
      directory.

## 4. Exact-SHA validation gate

Before building artifacts, freeze one clean commit SHA. Every check below must
run on that exact SHA:

```sh
test -z "$(git status --short)"
gofmt -w .
test -z "$(git status --short)"
go vet ./...
go test ./...
go test -race ./...
npm ci
npm run clone-check
llmthink dsl audit docs/decisions/sealgraph-design.think --pretty
llmthink dsl audit docs/decisions/2026-08-14-seal-revision-dag.think --pretty
perttool document check PLAN.pert
perttool dag analyze PLAN.pert
```

- [ ] The remote CI run for the frozen SHA succeeds.
- [ ] The dogfood receipt names the frozen source SHA and its separate
      provenance-metadata commit without claiming self-sealing.
- [ ] `sealgraph stale --refs-only` and `--frontier --refs-only` have the
      expected exact bytes after the dogfood sequence.
- [ ] `fsck` succeeds from a fresh canonical checkout after explicit runtime
      bootstrap.
- [ ] Artifact SHA-256 checksums are computed after the final archive bytes are
      fixed and independently rechecked.

## 5. Publication gate

Publication is a separate, non-idempotent operation. Before acting, record:

```text
release version:
exact commit SHA:
artifact names:
artifact SHA-256 values:
release-note digest:
maximum tag writes: 1
maximum GitHub Release writes: 1
```

- [ ] Obtain explicit operator approval for the frozen record above.
- [ ] Create one immutable `v0.1.0-alpha.1` tag at the approved SHA. Never
      force-move or reuse a published version tag.
- [ ] Push the tag through the configured `secdat exec` boundary.
- [ ] Create the GitHub prerelease once, through `secdat exec`, with only the
      approved artifacts and notes.
- [ ] Read back the remote tag SHA, prerelease flag, asset names/sizes, and
      downloaded asset checksums independently.
- [ ] Confirm the source tree and release assets contain no secret material.

If publication is incomplete or ambiguous, stop and inspect remote state. Do
not blindly resend. If a published alpha is defective, keep its tag immutable,
document the defect, and release a new alpha version after a new gate.

## 6. Post-release evidence

- [ ] Record the final SHA, tag, Release URL, CI URL, asset checksums, and smoke
      result in a release receipt.
- [ ] Verify installation from the downloaded artifact in a clean temporary
      environment.
- [ ] Keep Git-sidecar, structured JSON, attachments, signatures, and other
      deferred work explicitly open; alpha publication does not complete the
      `READY` milestone in `PLAN.pert`.
