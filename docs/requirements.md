# Sealgraph requirements

Status: initial normative draft.

## 1. Purpose

Sealgraph is a provenance-sealing system for logical content.

It MUST make it possible to answer:

1. What content was sealed?
2. Which exact upstream seal generations were used as its basis?
3. What attachments were part of that sealed state?
4. What seal did it supersede?
5. Why was it sealed?
6. Which current REF heads became stale after an upstream supersession?
7. Through which dependency paths did that impact propagate?

Sealgraph is not a general-purpose VCS.

## 2. Core concepts

### 2.1 REF

A REF is a logical content identity, not a filename.

Each REF has at most one current HEAD seal.

### 2.2 Blob

Content and attachments are stored/read as immutable content-addressed blobs.

Standalone operation MUST NOT require a working file corresponding to a REF.

### 2.3 Seal

A seal is an immutable snapshot of one REF's candidate state.

A seal MUST commit to:

- schema/format version,
- logical REF identity,
- previous seal identity when present,
- content identity,
- attachment identities plus stable attachment metadata,
- dependency links,
- root/draft state,
- seal message,
- normalized seal event metadata required by the storage format.

A seal belongs to exactly one logical REF. Its owner REF is identity-bearing
canonical state. A seal MUST NOT be reused as the HEAD, parent, or tag target of
another REF. Parent history stays within one REF.

A dependency link MAY cross a REF boundary because it names both the target REF
and one concrete seal owned by that REF. The edge does not transfer or alias the
target seal into the dependent REF.

One `seal` invocation MUST create at most one new seal for exactly one REF.

There MUST NOT be a `seal --all` or equivalent batch-approval operation in the core product.

### 2.4 Link

A link is a provenance edge from one seal to an exact upstream seal generation.

Links form an N:M directed acyclic graph across seals.

A persisted link MUST contain a concrete target seal identity.

Native v2 has one domain-independent dependency edge and no persisted link
kind. A link MAY carry an edge-specific message explaining why that exact
upstream generation is a dependency. The link message is part of the seal
identity and is distinct from the seal event message.

`--depend-on UPSTREAM` is command shorthand that resolves the current HEAD at operation time. The persisted seal MUST NOT contain a dynamic HEAD pointer.

The CLI MUST also support explicit historical generation selection.

Native v2 accepts an exact full seal ID, a repository-wide unique hexadecimal
prefix of at least four characters, or an immutable REF-scoped tag wherever an
explicit seal generation is selected. Resolution MUST produce a concrete full
seal ID before candidate or seal persistence.

A tag is an immutable alias for one seal owned by one REF. It MUST NOT become a
dynamic link, a movable branch, or a separate approval claim.

### 2.5 Root

A root seal generation explicitly declares a provenance boundary for that
immutable generation. Root is an identity-bearing seal attribute, not a
permanent type of the logical REF.

Root MUST NOT be inferred merely from an empty dependency list.

Root does not mean “true”, “trusted”, or “approved by an external authority”.

Successive generations of the same REF MAY explicitly change between root and
non-root. Such a change creates a new seal identity and MUST remain visible in
history/diff; it never changes the root state of an older seal. Changing the
root attribute MUST NOT add or remove dependency links automatically.

A non-root sealed candidate normally requires at least one upstream dependency.

### 2.6 Draft

Draft is an explicit semantic state for provisional sealing.

A draft may intentionally depend on a non-HEAD upstream seal.

Draft MUST remain visible in status/show/log output.

## 3. Supersession and stale propagation

Seals are immutable. Superseding a REF creates a new seal and moves that REF's HEAD.

Existing downstream links remain unchanged.

A current seal is directly stale when one of its persisted dependency target seal identities differs from the current HEAD seal identity of that dependency's logical REF.

A current seal is transitively stale when its dependency closure contains stale provenance.

Staleness MUST be derived from canonical seals and current REF heads. It MUST NOT be authoritative persisted state.

Resealing an upstream dependent changes its seal identity because the seal commits to upstream identities, even when its own textual content is unchanged.

This naturally propagates the need for explicit downstream review/reseal.

## 4. Seal admissibility

A normal non-draft seal MUST reject a candidate whose complete reachable
dependency closure is not HEAD-consistent or contains a draft seal.

This rule exists to force unresolved upstream review to progress explicitly from upstream to downstream.

Explicit draft/historical workflows MAY seal against older generations, but the resulting non-HEAD relation MUST remain observable and MUST NOT be reported as fresh.

A draft candidate MAY depend on current or historical draft/non-draft seals.
Draft is distinct from stale and MUST NOT be propagated, relinked, or resealed
automatically. To depend on provisional provenance, the operator explicitly
keeps the dependent candidate draft.

The exact CLI override model is to be finalized before v1; do not introduce a generic “ignore validation” escape hatch.

## 5. Attachments

Content may include zero or more named attachments.

Attachment bytes are immutable blobs.

A seal MUST commit to each attachment's blob identity and stable semantic metadata such as name and media type.

Renaming an attachment changes the seal state even if the attachment bytes are unchanged.

An attachment is contained evidence/artifact data. A link is an external provenance relation. The two MUST remain semantically distinct.

## 6. Working candidate

`add`, `link`, `unlink`, `attach`, and `detach` edit the next candidate state for one REF.

`add` MAY specify dependencies atomically with content creation/update:

```sh
sealgraph add DESIGN-001 \
  --content '...' \
  --depend-on REQ-001 \
  --depend-on POLICY-001@<seal-id>
```

`link` remains necessary for relinking without content changes.

Working candidate state is not a seal and is not authoritative history.

Standalone mutations MUST use repository-wide writer coordination. Cooperative
writers execute serially. A seal publishes at the successful expected-old CAS
update of its one target REF and MUST NOT clear a candidate version other than
the one it sealed.

## 7. Required inspection commands

The product is expected to provide:

- `show`
- `diff`
- `status`
- `log`
- `linklog`
- `impact`
- `graph`
- `stale`
- `fsck`

`diff` MUST be capable of representing content, attachment, link, and material metadata differences.

`status` MUST distinguish at least candidate modifications/unsealed state from direct/transitive staleness.

## 8. Intentionally absent VCS semantics

Core sealgraph MUST NOT implement Git-like:

- merge
- rebase
- branch
- checkout
- cherry-pick

Multiple upstream bases are expressed through provenance links, not merge commits.

## 9. Standalone initialization

`sealgraph init` MUST always initialize standalone mode.

It MUST NOT:

- detect `.git`,
- change behavior because it runs inside a Git working tree,
- suggest or activate Git sidecar automatically.

Standalone canonical reads MUST use `.sealgraph` only.

### 9.1 Standalone Git low-level compatibility

Standalone Git compatibility is limited to object identity/envelope
conformance and safe read-only low-level forensic interoperability. Native
objects MUST retain the documented Git SHA-256 loose-blob envelope and identity
so an explicitly configured Git SHA-256 low-level object API can read them
without identity disagreement, silent translation, or mutation.

This compatibility MUST NOT make `.sealgraph` a Git repository or import Git
commit, branch, merge, checkout, reflog, garbage-collection, maintenance, or
porcelain semantics. A sealgraph adapter or conformance tool used in an
incompatible object-format context MUST reject it rather than guess or
translate. In particular, the native SHA-256 object directory is not an
alternate object directory for a SHA-1 repository.

Standalone product code continues to avoid `.git`. Explicit temporary Git
conformance tests do not change that lifecycle boundary.

## 10. Git sidecar

Git sidecar is a separate product surface exposed as `git sealgraph ...` through a `git-sealgraph` executable.

Sidecar MAY use Git blobs, trees, commits, history, index stages, and merge state as read sources.

Sealgraph provenance semantics remain independent from Git commit semantics.

Git commits MUST NOT automatically create seals.

Git merge MUST NOT automatically repair stale provenance.

Git-sidecar MAY provide three-way conflict inspection/resolution assistance for sealgraph REF conflicts.

Automatic semantic merging or fabricated approval is forbidden.

## 11. Merge-friendly metadata

A `.sealgraph` directory tracked by an outer Git repository SHOULD merge predictably:

- immutable objects should be additive,
- one logical REF should use one small mutable ref file,
- canonical native storage should avoid pack/repack churn,
- canonical native refs should avoid packed-refs-like aggregation.

When the same logical REF advances differently on two Git branches, an outer Git merge conflict on that REF file is desirable.

When different REFs advance independently, Git should normally merge them without conflict.

## 12. Security

Sealgraph MUST NOT treat secret plaintext as a normal metadata field.

Repository docs/tests MUST NOT include real credentials.

Integration with secdat is optional and explicit; core operation does not depend on secdat.
