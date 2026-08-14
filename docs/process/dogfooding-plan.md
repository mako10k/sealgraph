# Sealgraph dogfooding plan

Status: R0, R1, native-v2, candidate-lifecycle, and native-v3 material-identity
rounds passed on 2026-08-14 and remain historical evidence. ADR 0011 now
accepts format 4; the next tracked round must use explicit logical dump/load,
must not rewrite format-3 dogfood in place, and must not claim migration before
the new runtime passes. The tracked project repository currently contains only
approved canonical format-3 standalone `.sealgraph` state; runtime-only paths
remain ignored. See
[`dogfooding-receipts/2026-08-14-r0.md`](dogfooding-receipts/2026-08-14-r0.md)
and [`dogfooding-receipts/2026-08-14-r1.md`](dogfooding-receipts/2026-08-14-r1.md).
The breaking format-2 regeneration and ADR dogfood are recorded in
[`dogfooding-receipts/2026-08-14-native-v2.md`](dogfooding-receipts/2026-08-14-native-v2.md).
The breaking format-3 material-identity regeneration is recorded in
[`dogfooding-receipts/2026-08-14-native-v3-material-identity.md`](dogfooding-receipts/2026-08-14-native-v3-material-identity.md).

## 1. Objective

Exercise sealgraph's own provenance semantics before expanding the product
surface. Dogfooding must test the explicit review workflow, not merely prove
that commands return exit code zero.

The plan has two separately gated rounds:

- R0: hermetic Phase 1 exercise in a temporary repository;
- R1: tracked provenance for this repository after Phase 2 graph semantics.

R0 does not authorize R1. R1 requires an explicit operator decision because it
adds canonical `.sealgraph/config`, objects, and refs to this repository.

## 2. Non-goals and safety boundaries

- Do not implement or invoke Git sidecar.
- Do not add a Git commit hook or automatic `commit -> seal` behavior.
- Do not use `seal --all`, recursive repair, or automatic relinking/resealing.
- Do not put credentials, environment dumps, source archives, or secret
  plaintext into dogfood content or link messages.
- Do not edit or delete immutable objects to make an assertion pass.
- Do not treat root as trusted or true; root only declares the scenario's
  provenance boundary.
- Do not persist derived stale state.

Dogfood commands use one REF per seal invocation and record concrete upstream
seal IDs.

## 3. Frozen inputs and receipts

Before each round, freeze and record:

1. source Git commit SHA;
2. `go version`;
3. built `sealgraph` binary SHA-256;
4. relevant ADR/storage-format digest;
5. exact command sequence;
6. expected status transition;
7. resulting REF heads.

Write a short receipt under `docs/process/dogfooding-receipts/`. A receipt is a
human-readable test record, not canonical stale state. It must distinguish
expected results, observed results, and unresolved differences. Do not commit
temporary candidates, locks, caches, logs, or decrypted data.

## 4. R0: hermetic Phase 1 dogfood

### Gate R0.0 — reproducible baseline

- Start from a committed source SHA and clean tracked worktree.
- Run `gofmt -w .`, `go vet ./...`, `go test ./...`, and
  `go test -race ./...`.
- Build `sealgraph` from that exact SHA and record the binary digest.
- Create the dogfood repository with `mktemp -d`; never reuse the project
  root's `.sealgraph/` for R0.

### Scenario R0.1 — initial Merkle provenance chain

Create and seal exactly these logical REFs in order:

```text
dogfood/spec/storage-contract             root
dogfood/implementation/native-slice       depends on storage-contract HEAD
dogfood/validation/core-tests             depends on native-slice HEAD
```

Use compact, non-secret content that includes the frozen source SHA and the
identity of the artifact being asserted. If an event attestation is required,
seal it as separate content and link it to the exact subject generation.

Acceptance:

- `show` exposes exact content, flags, parent, and concrete links;
- all three current heads are `CLEAN`;
- each dependent payload stores only its direct upstream seal identity;
- repeating inspection does not mutate any object or REF.

### Scenario R0.2 — upstream supersession and explicit repair

1. Change and reseal `dogfood/spec/storage-contract`.
2. Confirm the prior implementation seal/link is unchanged.
3. Confirm `dogfood/implementation/native-slice` is `STALE_DIRECT`.
4. In Phase 1, record that `dogfood/validation/core-tests` remains `CLEAN`
   because `STALE_TRANSITIVE` is intentionally not implemented yet.
5. Relink and reseal only `dogfood/implementation/native-slice`.
6. Confirm `dogfood/validation/core-tests` is now `STALE_DIRECT`.
7. Relink and reseal only `dogfood/validation/core-tests` and confirm `CLEAN`.

Acceptance:

- every repair is an explicit one-REF `link` plus `seal` operation;
- unchanged content with a changed direct upstream receives a new seal ID;
- parent history remains readable;
- no command offers or performs downstream recursive repair.

### Scenario R0.3 — historical draft

Create a draft review REF linked explicitly to the first historical
storage-contract seal:

```text
dogfood/review/historical-storage
  -> dogfood/spec/storage-contract@<first-seal-id>
```

Acceptance:

- the historical ID is preserved exactly;
- status reports both `DRAFT` and `STALE_DIRECT`;
- the same candidate is rejected as a normal non-draft seal;
- relinking to HEAD is explicit.

### Scenario R0.4 — integrity failure

Copy the completed temporary repository, corrupt one loose object in the copy,
and attempt to read the affected seal.

Acceptance:

- the read fails with corruption/hash mismatch context;
- the object is not overwritten or repaired;
- the original temporary repository remains readable;
- the receipt identifies the damaged object ID without embedding its content.

### Gate R0.5 — closeout

R0 passes only when every expected transition matches and the receipt contains
no unresolved integrity or semantic difference. Preserve the receipt, then
remove the temporary repository. A failure blocks R1 and Phase 2 expansion at
the earliest failed scenario.

## 5. R1: tracked repository provenance

R1 starts only after:

1. R0 passes;
2. Phase 2 `STALE_TRANSITIVE` and impact traversal pass automated tests;
3. the operator explicitly approves creating this repository's `.sealgraph/`;
4. the exact initial REF set and content manifest are reviewed.

If a canonical-only outer checkout cannot be opened because ignored runtime
directories are absent, complete `RUNTIME_BOOTSTRAP` first. The accepted
behavior is an explicit idempotent `sealgraph init` that recreates only missing
empty runtime directories and never repairs objects or REFs.

Approved initial REF hierarchy for R1:

```text
sealgraph/spec/requirements                 root
sealgraph/design/architecture               depends on requirements
sealgraph/spec/storage-v1                   depends on requirements and architecture
sealgraph/implementation/native-phase1      depends on storage-v1
sealgraph/validation/go-core                depends on native-phase1
```

Each content value is a small UTF-8 manifest containing the frozen predecessor
Git commit SHA plus every included path and its SHA-256 digest. The
architecture manifest covers `docs/architecture.md`, `docs/cli.md`,
`docs/integrations.md`, all current ADRs, and the llmthink design document. The
storage manifest covers `docs/storage-format.md` and ADR 0005. The
implementation manifest commits to a sorted aggregate digest of the Go source
files. The validation manifest records the exact Go validation commands and
their successful result. The R1 receipt records the exact bytes supplied to
each `add` command.

This is a manifest claim: Phase 2 still has no file-import command, so a path
name alone is never presented as though sealgraph read or stored that file's
bytes.

Run standalone `sealgraph init` in the project root. Its behavior must be the
same with or without `.git`; do not invoke `git sealgraph`. Track only canonical
repository state:

- `.sealgraph/config`;
- `.sealgraph/objects/`;
- `.sealgraph/refs/seals/`.

The existing `.gitignore` keeps candidate/index, cache, locks, and logs out of
commits. Inspect the staged paths before committing and reject any secret or
runtime-only material.

R1 acceptance:

- initial heads are clean and graph/impact output matches the declared chain;
- a controlled storage-contract supersession produces the expected direct and
  transitive impact without stored stale fields;
- repairs remain one REF at a time;
- the resulting canonical metadata is committed in its own reviewable Git
  commit;
- checking out the parent commit and the dogfood commit reproduces the two
  exact canonical states; an explicit `sealgraph init` may bootstrap ignored
  runtime directories, but no automatic object/REF repair is invoked.

If R1 is rejected, use an explicit outer-Git revert of the isolated dogfood
commit. Do not rewrite Git history or selectively delete immutable sealgraph
objects in place.

## 6. Promotion criteria

Dogfooding is evidence, not release authority. A release decision remains a
separate gate. Promote the workflow from R0 to R1 only after reviewing the R0
receipt; promote from R1 to recurring use only after at least one controlled
upstream supersession and sequential downstream repair has been independently
read back.

## 7. Native v2 decision-document round

ADR 0006 acceptance authorizes a breaking regeneration from an empty format-2
repository. This round seals exact document bytes for requirements,
architecture, ADR 0006, and the native-v2 storage contract. It must exercise
REF-scoped tags, one unique-prefix selector, and per-edge messages while still
persisting only full concrete IDs. It completes `NATIVE_V2_SLICE`; it does not
claim the later recurring `DOGFOOD_R2` milestone or authorize Git sidecar work.

## 8. Future decision-document causality

The native-v2 round is retrospective bootstrap evidence: the sealed
requirements and architecture generations already contain the ADR 0006
conclusion. It proves exact document storage, concrete links, tags, and
inspection behavior; it does not prove the causal order in which the decision
was formed.

Future ADR dogfood should preserve the decision sequence explicitly:

1. seal pre-decision premises and evidence;
2. seal the ADR against those exact upstream generations;
3. update and supersede normative documents with a concrete link to the ADR
   generation;
4. inspect the resulting stale and explicit downstream repair path.

Do not rewrite an old seal or receipt to make a retrospective graph look
causal.

## 9. Candidate lifecycle focused round

After ADR 0008 implementation, use a temporary standalone repository to
exercise candidate show/diff, guarded historical unlink, explicit relink,
discard, bounded binary-safe preview, and exact candidate/sealed raw content.
The round must prove that unlink/discard affect one candidate only and that no
raw control byte is mixed with human metadata. It is focused implementation
evidence and does not complete the recurring `DOGFOOD_R2` workflow.

## 10. Native v3 material-identity regeneration

ADR 0009 authorizes a breaking regeneration from an empty format-3 repository.
After code and canonical fixture validation, preserve the old format-2 state in
outer Git history and replace the tracked `.sealgraph/` state explicitly; do
not rewrite old objects in place or add a v2 reader.

The focused round must:

1. initialize format 3 without inspecting `.git`;
2. seal exact ADR 0009 and normative-document bytes without a seal event
   message/time/actor;
3. link normative documents to the exact ADR generation where applicable;
4. prove `show`, `log`, and canonical payload bytes contain none of those event
   fields;
5. preserve full concrete IDs and edge-specific link messages;
6. record source/binary/document hashes and resulting heads in a new receipt.

This is a format-regeneration decision, not an automatic migration or authority
claim. Any actor/time approval assertion must be modeled as separately sealed
content and an explicit link.
