# ADR 0006: Experimental native v2 IDs and REF-scoped tags

Status: accepted on 2026-08-14.

The seal-level event-message behavior described below is superseded by ADR 0009.
ADR 0009 retains edge-specific link messages but removes seal-level `message`
and `created_at` from native v3.
The `refs/tags/<REF>/<ENCODED_TAGNAME>` storage layout is superseded for format
4 by ADR 0013's tags-inside-REF-manifest contract. TAGNAME grammar, immutable
binding semantics, and selector forms remain applicable.

## Context

Sealgraph is still experimental. The current native v1 contract repeats the
fixed `sha256` algorithm in human output, REF files, candidates, and each
native identity inside a canonical seal. It accepts only a full tagged seal ID
in historical selectors and has no stable human alias for a seal generation.

The current direction is intentionally changed before 1.0:

1. result and implementation simplicity outrank backward compatibility;
2. seal IDs should look and resolve like Git object names;
3. a REF-scoped tag may be used wherever a seal ID is accepted;
4. native loose objects should conform to Git ODB rules wherever this does not
   import Git commit/branch/merge semantics.

This decision supersedes the native v1 textual and canonical ID representation
in ADR 0005. It does not change the provenance model in ADR 0001.

## Decision

### Experimental compatibility policy

Before 1.0, a storage simplification may make an experimental repository
unreadable. This change introduces `repository_format = 2` and
`sealgraph/seal/v2`. The implementation will reject format 1 rather than carry
a dual reader, automatic migration, legacy parser, or compatibility switch.

Tracked dogfood state will be regenerated explicitly after the implementation
passes from an empty format-2 repository. No command silently rewrites a
format-1 repository.

After 1.0, persisted and public CLI compatibility becomes a required design
consideration. This decision does not pre-decide that future policy.

### Native object and seal IDs

The repository config fixes native object format to SHA-256. Therefore a native
object ID and seal ID are represented as exactly 64 lower-case hexadecimal
characters. `sha256:` is not emitted, accepted, or persisted in native v2.

Native canonical v2 stores IDs as JSON strings containing full hex rather than
objects containing redundant `algorithm` and `hex` members. This applies to:

- `parent`,
- content and attachment blob IDs,
- direct link target seal IDs,
- candidate base and concrete dependency IDs.

If a future backend needs more than one object format, that backend receives an
explicit schema decision then. Native v2 does not preserve algorithm tags only
for a speculative future reader.

REF heads, tag files, candidates, and canonical seals always store full IDs.
Human-readable command output also prints full IDs so receipts are stable and
do not depend on the current object population.

### Unique-prefix references

A user-supplied seal token may be a lower-case hexadecimal prefix of 4 through
64 characters. Resolution searches every valid object name in the native ODB,
not only current seals. This follows Git object-name abbreviation: the prefix
must name exactly one object in the repository.

After prefix resolution, the object must decode as a canonical seal owned by
the selected logical REF. Zero matches, multiple matches, a non-seal object, or
a seal owned by another REF is an explicit error. Prefixes are never persisted.

### REF-scoped tags

A tag is an immutable lightweight name for one exact existing seal. It is not
a seal, approval event, Git tag object, dynamic dependency, or source of stale
state. Resolving a tag for `--depend-on` stores the concrete full target seal ID
in the candidate and eventual seal.

Tags are scoped by logical REF. The canonical loose path is:

```text
.sealgraph/refs/tags/<REF>/<ENCODED_TAGNAME>
```

The tag file contains exactly one full 64-character lower-case seal ID followed
by LF. The same raw tag name may be used under different REFs. A tag target must
decode as a canonical seal owned by the path's REF.

Initial CLI surface:

```text
sealgraph tag REF                         # list tags for REF
sealgraph tag REF TAGNAME                 # tag the current head
sealgraph tag REF@SEAL_OR_TAG TAGNAME     # tag an exact generation

sealgraph show REF@TAGNAME
sealgraph link CHILD --depend-on REF@TAGNAME
sealgraph diff REF@OLD_TAG REF@NEW_TAG
```

Creating an absent tag is atomic. Repeating the same name and target is
idempotent; assigning an existing name to a different seal is rejected. The
first slice has no force-move, delete, or automatic tag creation.

### Dependency links and messages

Native v2 has one domain-independent link semantic: an exact upstream
dependency. It therefore removes the redundant persisted `relation`/`kind`
field and retains the `--depend-on` CLI vocabulary. Domain-specific kinds and
free-form kind taxonomies are deferred until a domain-independent semantic need
exists.

Each dependency link has an optional UTF-8 `message`. `link -m MESSAGE` stores
the rationale for that exact dependency edge in candidate state and the next
seal. It is part of canonical bytes and seal identity. `seal -m MESSAGE`
remains separate event metadata explaining why the complete candidate was
sealed. Changing only a link message is a material candidate change and creates
a new seal when sealed.

Native v2 permits one dependency link per target REF. Canonical link order is
`(target_ref, target_seal, message)` using the existing bytewise string rules.
`linklog` distinguishes target repoints from link-message changes. Every link
participates in HEAD consistency, stale propagation, impact, and the Merkle
DAG; v2 has no informational/non-staling link class.

### TAGNAME encoding and selector grammar

The user-facing TAGNAME is valid non-empty UTF-8. ASCII control bytes, DEL, and
`@` are forbidden. `/` is allowed in the raw name.

Encoding operates on UTF-8 bytes. ASCII letters, digits, `-`, and `_` remain
literal; every other byte is `%` followed by two upper-case hexadecimal digits.
The mapping is injective and reversible. For example:

```text
release-1    -> release-1
release/1    -> release%2F1
v1.0         -> v1%2E0
```

The encoded result is one filesystem/refname component, so the only unencoded
`/` in the internal tag path is the boundary between `<REF>` and
`<ENCODED_TAGNAME>`.

The simple `<REF>/<ENCODED_TAGNAME>` layout can have an explicit loose-path
conflict when a tag on a prefix REF equals the next component of a child REF;
for example, `design` tag `api` versus tags scoped to `design/api`. V2 rejects
the conflicting tag namespace in either order. It does not add a hidden path
marker or a compatibility mapping not present in the approved layout.

`@` becomes the unambiguous selector delimiter in native v2 and is forbidden
inside both REF and TAGNAME. `REF@TOKEN` interprets TOKEN as:

1. a TAGNAME when it is not a 4-to-64-character lower-case hex token;
2. an ODB object-name prefix when it is such a hex token.

TAGNAMEs matching `[0-9a-f]{4,64}` are reserved so a tag can never hide a seal
ID prefix. This deliberately narrows the Git-compatible REF grammar to keep the
sealgraph selector grammar deterministic and small.

### Git ODB conformance boundary

Native v2 keeps the Git SHA-256 loose-object contract:

```text
object bytes: zlib("blob " + decimal-size + NUL + payload)
object ID:    sha256(uncompressed object bytes), rendered as lower-case hex
path:         objects/<first 2 hex>/<remaining 62 hex>
```

Seal payloads remain Git `blob` objects. Modeling them as Git commit or tag
objects would import the wrong semantics and is not required for ODB
compatibility.

Conformance tests will:

- compare native blob IDs with `git hash-object` in a temporary SHA-256 Git
  repository when Git is available;
- prove Git can read a native loose object with `git cat-file`;
- prove sealgraph can read a Git-produced SHA-256 loose blob;
- retain deterministic tests that do not require Git, so core tests do not
  gain a runtime Git dependency.

Standalone product code still never discovers or reads `.git`. Git conformance
fixtures and explicit temporary integration tests do not change that boundary.

Native v2 remains loose-only. Packfiles, alternates, compatibility object maps,
Git commits, branches, reflogs, and annotated Git tag objects are not added by
this change.

## Consequences

- Human IDs and REF/tag files become simpler full hex.
- Any unique prefix of at least four hex characters may select an object.
- Tags improve repeatable human workflows without weakening concrete-link
  provenance.
- Existing native v1 repositories and tracked dogfood objects are deliberately
  not readable by the format-2 implementation.
- Every current canonical seal ID changes because the v2 canonical payload
  changes.
- REF and TAGNAME can no longer contain `@`.
- Numeric/hex-like tags such as `2026` are reserved; a non-hex name such as
  `release-2026` remains available.
- Dependency edges gain hash-committed rationale without a domain-specific
  kind taxonomy.

## Approved scope

Acceptance explicitly approves the material choices below:

1. breaking format 1 with no compatibility reader or automatic migration;
2. removing algorithm members from native canonical v2, not only CLI output;
3. reserving `@` in REF/TAGNAME and reserving 4-to-64-character hex TAGNAMEs;
4. immutable REF-scoped tags with the percent encoding and CLI forms above.
5. deferring link kinds while adding an optional hash-committed link message.

## References

- Git loose object format: <https://git-scm.com/docs/gitformat-loose>
- Git revision/object-name syntax: <https://git-scm.com/docs/gitrevisions>
- Git abbreviated object names: <https://git-scm.com/docs/git-rev-parse>
- Git refname rules: <https://git-scm.com/docs/git-check-ref-format>
- Git SHA-256 transition: <https://git-scm.com/docs/hash-function-transition>
