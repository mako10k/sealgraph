# ADR 0007: Linearized standalone publication and non-draft closure

Status: accepted on 2026-08-14.

## Context

REF compare-and-swap protects one HEAD update, but it does not by itself make a
complete seal operation coherent. Without repository-wide coordination, one
process can seal candidate C1 and then delete a newer candidate C2 written by
another process. Dependency REF heads can also move between closure validation
and publication, allowing a normal seal to be stale at the moment it is
published.

Draft is defined as provisional sealing. If a normal seal can depend on a draft
seal anywhere in its reachable closure, the normal head may appear `CLEAN`
while its provenance remains provisional.

## Decision

### Repository-wide writer coordination

Every standalone sealgraph mutation after repository initialization acquires
one repository-wide process-lifetime writer guard. Cooperative writers wait and
execute serially. Read-only commands do not acquire this exclusive guard.

The successful expected-old CAS update of the one target REF is the seal
publication linearization point. Immutable object writes before this point may
leave dangling objects after a failed publication; they are reported and never
deleted or repaired automatically.

Candidate clearing occurs after publication and removes the candidate only when
it is still the exact candidate version used to create the seal. If it differs,
the REF remains published, the newer candidate remains, and the command reports
the partial outcome with the new seal ID.

The guarantee covers processes using the same sealgraph writer protocol. Outer
Git operations and manual filesystem writes do not honor this guard. Sealgraph
validates expected state and fails safely where it can detect such an external
concurrent mutation; it does not claim exclusion of arbitrary writers.

### Draft closure

A normal non-draft seal requires every reachable dependency seal in its
complete closure to be non-draft as well as HEAD-consistent. A draft anywhere
in the closure rejects normal publication.

A draft candidate may depend on current or historical draft/non-draft seals.
Draft remains distinct from stale: draft records provisional intent, while
stale is derived from concrete dependency IDs and current REF heads.

No command automatically propagates a draft flag, relinks a dependency, or
reseals a downstream REF. The operator must explicitly keep the dependent
candidate draft or wait for/relink to a non-draft upstream generation.

## Consequences

- Successful concurrent candidate edits are not silently lost by a seal.
- Cooperative standalone mutations have one simple ordering and failure model.
- One seal still publishes exactly one REF; serialization does not introduce a
  batch transaction.
- Readers may observe only atomically published files, but read commands are
  not a multi-file snapshot API.
- A normal `CLEAN` head cannot hide draft ancestry that existed when it was
  sealed.
- Platform support for the process-lifetime writer guard is explicit; an
  unsupported platform refuses mutation rather than running without the guard.

