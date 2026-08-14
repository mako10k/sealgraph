# ADR 0009: Separate seal event metadata from sealed material identity

Status: accepted on 2026-08-14 by explicit operator direction.

## Context

Native v2 made `created_at` and a seal-level event `message` required members of
every canonical seal. Consequently, every dependency on that seal also depended
on unauthenticated facts about when and why the seal operation happened. An
`actor` field did not yet exist, but adding one would have the same effect.

Sealgraph should make dependency on actor, time, or approval claims explicit and
optional. A hash proves byte identity; it does not by itself prove an actor's
identity, authority, or a trusted time.

## Decision

### Material/provenance seal state

Native v3 canonical seals contain only:

- schema and owner REF,
- parent seal identity,
- content identity,
- attachment identities and stable attachment metadata,
- concrete direct dependency links, including an optional edge-specific link
  message,
- root and draft state.

Seal-level `message` and `created_at` are removed. `actor` and other seal event
metadata are not added. `sealgraph seal` accepts exactly one REF and no `-m`
event-message option.

The parent remains canonical history structure. REF, root, draft, attachment
metadata, direct links, and an edge-specific link message remain semantic
material/provenance state rather than seal-operation metadata. In particular,
the link message explains the concrete dependency relation; it does not claim
who created or approved it.

### Explicit attestations

When a domain needs actor, time, approval rationale, or similar event evidence,
it represents that claim as ordinary content under a separate logical REF,
seals it, and links it to the exact subject seal generation. The claim's own
schema defines its semantics. Sealgraph core does not imply authentication,
authority, trusted time, or truth.

Signatures and trusted timestamps remain future features requiring an explicit
ADR. They must not be approximated by reintroducing unauthenticated scalar
fields into every seal.

### Experimental format boundary

This changes the canonical member set. The repository format becomes 3 and the
schemas become `sealgraph/seal/v3` and `sealgraph/candidate/v3`. Formats 1 and 2
are rejected. There is no dual reader, automatic migration, ignored legacy
field, compatibility option, or mutable event log added by this decision.

## Consequences

- Repeating the same initial material/provenance state under the same REF has
  the same seal ID regardless of wall clock, process identity, or operator.
- A dependency does not inherit compulsory actor/time/event-rationale claims
  through its upstream seal ID.
- Parent, content, attachment, link, root, or draft changes still change seal
  identity. Relinking to a different upstream generation still changes it.
- Event evidence is opt-in, graph-visible, independently inspectable content.
- Existing experimental format-2 repositories must be regenerated explicitly.
- Historical v2 receipts remain evidence about the implementation that produced
  them; they are not readable as v3 repositories.

## Superseded decisions

This ADR supersedes the seal-level event-message and `created_at` parts of ADRs
0005 and 0006. It does not change ADR 0006's edge-specific link message decision.
