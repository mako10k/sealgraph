# Sealgraph agent guide

This file is a navigation map, not the full specification.

## Read first

Before changing behavior or public CLI contracts, read:

1. `docs/requirements.md` — normative product requirements.
2. `docs/architecture.md` — boundaries and package responsibilities.
3. `docs/storage-format.md` — repository/object/ref invariants.
4. `docs/cli.md` — CLI semantics.
5. Relevant ADRs under `docs/adr/`.
6. `docs/integrations.md` when touching llmthink, secdat, perttool, or Git sidecar behavior.

When documents conflict, use this precedence:

1. Explicit current task instructions.
2. Accepted ADRs.
3. `docs/requirements.md`.
4. Other design documentation.
5. Existing implementation behavior.

Do not silently resolve a material specification conflict. Record it in an ADR or a clearly marked TODO in the relevant design document before implementing behavior that depends on it.

## Core invariants

- `sealgraph init` is always standalone. It MUST NOT detect or inspect Git.
- Git sidecar is a separate Git plugin surface invoked as `git sealgraph ...` by the `git-sealgraph` executable.
- A seal operation creates exactly one new seal for exactly one logical REF.
- No batch seal, recursive repair, automatic relink, or automatic stale repair.
- Seals and content-addressed objects are immutable.
- A movable REF points to the current seal (HEAD) for that logical REF.
- A dependency link stores a concrete target seal ID. `--depend-on REF` resolves REF HEAD at command execution time; dynamic HEAD references are never persisted inside a seal.
- Explicit historical links are valid. A normal non-draft seal defaults to requiring a HEAD-consistent dependency closure; draft/historical workflows may intentionally preserve older dependencies.
- Staleness is derived from immutable seals plus current REF heads. Do not persist stale as canonical state.
- Link targets are not copied into the dependent seal; their normalized identities are committed into the dependent seal hash.
- Direct upstream seal identities are sufficient; transitive provenance is committed Merkle-DAG style.
- Root content is explicit. Root means provenance boundary, not truth.
- Attachments are immutable blobs included by identity in the seal.
- Standalone canonical storage uses `.sealgraph/` only.
- Standalone object storage should remain low-level Git-compatible where practical, but Git repository semantics are NOT part of the standalone contract.
- Avoid canonical pack files / packed refs in v1. Keep immutable loose objects and one mutable ref per file so an outer Git repository can merge `.sealgraph/` predictably.
- `merge`, `rebase`, `checkout`, `branch`, and `cherry-pick` are not sealgraph core concepts.
- Git-sidecar conflict tooling may offer three-way inspection/resolution assistance, but MUST NOT fabricate semantic approval or silently create seals.

## Development workflow

- Prefer small vertical slices with tests over speculative framework construction.
- Keep domain semantics independent from CLI parsing and storage implementation.
- New persisted fields require:
  - a storage-format change,
  - deterministic canonicalization tests,
  - compatibility consideration,
  - and normally an ADR.
- Error messages should explain the violated invariant and the next explicit action; never silently repair provenance.
- Use deterministic test fixtures. Inject clocks/hashers where required.
- Do not store secrets, credentials, tokens, private keys, or decrypted secret material in `.sealgraph/`, fixtures, logs, or snapshots.
- Do not add Git auto-detection to standalone code, including as a convenience.
- Do not add an automatic `git commit -> seal` hook.

## Validation

Run before completing a code change:

```sh
gofmt -w .
go vet ./...
go test ./...
```

If Git integration is changed, also add/run focused integration tests with temporary repositories.

Optional project tools are documented in `docs/integrations.md`. Their absence must not prevent core Go tests from running.

## Commit/PR expectations

- Keep generated artifacts out of commits unless explicitly documented as canonical.
- Explain any invariant or storage-format change in the PR/commit description.
- Do not rewrite existing history merely to make tests pass.
