# ADR 0021: Do not expand the attachment model

Status: Accepted

Date: 2026-08-21

Accepted: 2026-08-21 by explicit operator approval.

## Context

Format 4 includes `attachments` in every canonical Seal and candidate. Each
attachment gives one immutable blob a name and media type inside its containing
Seal. The field is identity-bearing, is preserved by dump/load and derive, and
is validated by inspection and `fsck`. The current CLI deliberately does not
expose the planned `attach` or `detach` mutations.

The implemented product now has a simpler set of composition mechanisms:

- root content stores the primary immutable result;
- a local source binding associates one REF with one machine-local workfile
  without entering canonical identity;
- a Cause Link relates a dependent Seal to one exact upstream Seal generation;
- an explicit path manifest can make a reviewed multi-file path/digest claim
  ordinary content without importing the named files implicitly.

A secondary artifact can therefore be modeled as the content of its own Seal
and related through a Link when it needs independent identity, review,
revision, staleness, or reuse. A collection that does not need independent
artifact identity can be represented explicitly in primary content, for
example by a canonical manifest.

Attachments retain one narrow capability: embedding a named, media-typed blob
inside another Seal without giving the blob its own logical lifecycle. No
concrete workflow has justified the additional mutation, canonicalization,
comparison, migration, and operator-choice surface that this capability would
require.

The separately explored future Content/Link/Context direction further weakens
that role. In that direction, a Link can commit the sealing information needed
to relate Content in Context, so a secondary Content need not be wrapped in a
second attachment mechanism. That future model is not the format-4 contract
and must not be partially introduced into the current runtime.

## Decision

### No attachment mutation surface

Sealgraph will not implement `sealgraph attach`, `sealgraph detach`, or an
equivalent attachment mutation API for format 4.

New user workflows should use:

- primary Seal content for the principal result;
- local source binding for machine-local workfile selection;
- a separately sealed Content plus an exact Link when the related artifact has
  independent identity or provenance;
- explicit manifest content when the result is a reviewed claim about a set of
  files rather than storage of those files.

This decision does not make a local source binding canonical and does not make
a format-4 Link target a raw blob. A format-4 Link continues to target an exact
Seal ID.

### Minimize canonical sealing information

Information used only to select or locate machine-local source material should
remain in explicit local binding state wherever practical. Examples include a
workfile path and other acquisition details that are not needed to verify the
portable result or its provenance.

Canonical Seal, Content, Link, and future Context state should contain only the
minimum identities and semantic claims that another repository or machine must
be able to verify. Local binding is not a portable provenance claim, approval,
trusted-time assertion, or substitute for an exact canonical dependency.

The current format-4 binding schema stores one path only. This principle does
not authorize adding binding fields, a generic `--bind` option, or a new
binding schema. Any such extension requires its own concrete use case and
contract. The existing public surfaces remain `source bind` and
`add --bind-source`.

### Preserve format-4 compatibility

Format-4 `attachments` remain a required canonical array. The runtime must
continue to:

- decode and validate attachment-bearing Seals and candidates;
- preserve attachment identities and metadata through supported candidate,
  derive, dump/load, inspection, comparison, and repository validation paths;
- include attachments in canonical Seal identity and deterministic ordering;
- reject malformed, duplicate, missing, or corrupt attachment material;
- emit the required empty array when a new Seal has no preserved attachments.

The runtime must not silently delete, flatten, convert, or reinterpret an
existing attachment as primary content or a Cause Link. Existing
attachment-bearing repositories remain valid format-4 repositories.

### Future removal requires a semantic-format decision

Removing the field is deferred to a separately accepted semantic-format
change. That decision must define Content, Link, Context, sealing identity,
projection of representative format-4 cases, dump/load or migration behavior,
and compatibility policy together.

Before such a change, representative cases must demonstrate how at least the
following project without information loss or fabricated semantics:

1. one primary document with a secondary binary artifact;
2. one generated result with independently revised evidence;
3. one named media object reused by multiple results;
4. one attachment-bearing historical format-4 Seal;
5. one multi-file manifest whose named files are not themselves imported.

There is no automatic rule that every attachment becomes a Cause Link. A
projection may choose primary Content, linked Content, manifest Content, or an
explicit unsupported disposition according to the reviewed meaning of the
case.

## Consequences

- The planned attachment CLI and its tests are removed from the implementation
  roadmap.
- The ordinary model has fewer ways to express the same related material.
- Related artifacts that require an independent lifecycle become visible
  graph nodes rather than hidden members of another Seal.
- Format-4 readers and migration logic retain attachment complexity until a
  later format decision removes it explicitly.
- No current Seal ID, candidate schema, repository format, CLI output schema,
  or attachment-bearing repository changes merely by accepting this ADR.
- The future Content/Link/Context direction remains a design input, not an
  authorization to implement format 5 or rewrite existing provenance.

## Accepted documentation effects

Align the normative and planning documents in one documentation change:

1. update `docs/requirements.md` to retain format-4 attachment compatibility
   while removing attachment mutation as a required product capability;
2. change the planned `attach`/`detach` section in `docs/cli.md` to an
   intentionally absent surface;
3. remove the attachment implementation phase from
   `docs/process/implementation-plan.md`;
4. retain storage-format and canonical compatibility requirements unchanged;
5. add no runtime code solely to claim attachment feature completion.
