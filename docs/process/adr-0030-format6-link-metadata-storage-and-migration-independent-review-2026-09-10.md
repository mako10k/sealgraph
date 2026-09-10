# Independent ADR review: ADR 0030 format-6 Link metadata storage and migration — 2026-09-10

## Review identity

- Subject: `docs/adr/0030-format6-link-metadata-storage-and-migration.md`
- Stage: `Proposed`
- Expected SHA-256:
  `b879af866be94106c8a7675a1477e9ccfabfdc823058c4b3bb7157aa757f198e`
- Digest before review: matched
- Digest after review: matched
- Reviewer role: one independent, read-only repository-wide ADR reviewer
- Verdict: `PASS`

This review does not accept ADR 0030, change its status, authorize a companion
decision, or authorize normative conversion, implementation, migration, commit,
push, release, deployment, or another external write.

## Findings

No P0, P1, P2, or P3 findings.

Source-cause phase is not applicable because no design defect was established.

## Review questions

### 1. Historical v5 reading inside format 6

**Result: Pass.** ADR 0026 identifies the format-4 incompatibility as replacing
global `parent_revision` with Cause-scoped assertions and therefore carrying two
different graph authorities (ADR 0026 lines 26–49). Accepted ADR 0029 instead
keeps graph derivation exclusively in the existing target and previous-revision
fields and makes metadata structurally opaque (ADR 0029 lines 125–136).

ADR 0030 accepts only exact v5/v1 and v6/v2 typed pairings, keeps the v5 reader
schema-exact and read-only, and applies one ADR 0023 graph meaning to both (ADR
0030 lines 182–203). The two generations have different bytes but no competing
Cause or revision semantics. This is a bounded compatibility reader, not a
return to format-4 mixed graph authority.

### 2. Typed-schema transitions

**Result: Pass.** ADR 0030 introduces Seal v6, Provenance v2, and Candidate v6,
requires v6-only writes, and forbids shape-based schema guessing (lines 53–67).
It fixes successor records and makes `metadata` required on every successor Link
(lines 68–118), prohibits Seal/Provenance cross-pairing (lines 184–193), and
assigns the changed assessment-free preimage to `upstream-change/v2` (lines
274–293).

Accepted ADR 0025 requires exact typed-schema validation, canonical re-encoding,
exact byte equality, and unknown-member rejection (ADR 0025 lines 82–89 and
93–128). Every persisted type whose meaning changes receives a new identifier.
Material v1, Blob identity, and REF manifest v1 retain their meaning and do not
need artificial version changes.

### 3. Config-only migration

**Result: Pass.** Accepted ADR 0025 establishes exact Candidate bytes as mutable
optimistic-version state (ADR 0025 lines 216–244). ADR 0030 reads Candidate v5
without mutation and converts only during a separately authorized Candidate
mutation or publication projection (lines 205–222).

Migration requires the writer guard, exact format-5 config, complete fsck,
double-captured REF, Candidate, and object inventories, atomic replacement of
only config, file and directory synchronization, and a complete format-6 reopen
and readback (lines 224–252). It expressly forbids Blob, REF, tag, Candidate,
metadata, or Seal mutation and treats ambiguous durability or readback as
uncertain without automatic retry.

Interruption therefore leaves exact format-5 or exact format-6 config rather
than a multi-file semantic rewrite. Both sides can read the untouched v5 objects
and Candidate bytes applicable to that side of the transition.

### 4. Canonical values and resource dimensions

**Result: Pass.** Accepted ADR 0029 delegates integer representation, encoding,
ordering, depth, node, string/key, entry, and byte bounds to this companion (ADR
0029 lines 99–123). ADR 0030 fixes entry order, namespace order and uniqueness,
signed 64-bit integer range and spelling, scalar encoding, array significance,
object-key ordering and uniqueness, and exact byte revalidation (lines 99–155).

It fixes entry, namespace, schema, nesting, node, key/string, and complete
metadata-array byte limits and requires bounded validation without truncation or
partial persistence (lines 157–180). It explicitly chooses not to impose a new
containing-object limit that could retroactively invalidate valid historical
message or Link counts. Each new independently variable resource dimension is
therefore bounded or explicitly retained as an existing compatibility dimension.

### 5. ADR 0028 change identity

**Result: Pass.** Accepted ADR 0028 defines the assessment-free change preimage,
makes every Cause Link identity-bearing in it, and excludes Assessment references
to avoid self-reference (ADR 0028 lines 174–197). It also prohibits retroactive
invalidity or fabricated assessments for historical Seals (lines 352–358).

ADR 0030 preserves the preimage member order, uses complete format-6 Cause Links,
and moves the changed meaning to `upstream-change/v2` (lines 274–287). It
preserves existing immutable v1 change and Assessment Blobs and leaves
Assessment-reference representation to the separately gated ADR 0028 storage
successor (lines 289–293). Metadata changes `change_id`, but no AssessmentID,
adoption path, disposition, or fabricated migration state enters this slice.

## Uncertainty and limitation

The repository process writer guard does not continuously monitor direct
filesystem or outer-VCS mutation outside the supported protocol. ADR 0023 lines
158–171 classify such mutation as unsupported, and lines 207–221 limit the
observation claim rather than promising continuous monitoring. Supported writers
remain serialized by the guard and existing Blobs remain immutable. This
limitation does not change the verdict.

## Unresolved questions

The following remain separately gated and are not ADR 0030 defects:

- exact migration CLI spelling;
- public human and versioned machine output schemas;
- persisted Assessment references, closure, and historical Assessment handling;
- implementation fixtures; and
- normative-document conversion.

## Coverage

The review covered:

- ADR 0030 internal definitions, alternatives, consequences, migration steps,
  review questions, and Claim/Evidence/Action traceability;
- accepted ADRs 0023, 0025, 0026, 0027, 0028, and 0029;
- the accepted revision-2 requirement and acceptance record;
- current requirements, architecture, storage-format, and CLI documents; and
- relevant format-5 config, fsck, Candidate, and writer-guard implementation for
  current-fit facts.

Expected CLI/output work and separately gated Assessment-reference storage were
excluded from defect scope as required.

## Reviewer conclusion

The exact candidate receives `PASS` with no P0 through P3 findings. ADR 0030
remains `Proposed` pending the Operator's separate `ACCEPT` or `REVISE`
decision.
