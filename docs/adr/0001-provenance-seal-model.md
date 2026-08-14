# ADR 0001: Immutable provenance seal model

Status: accepted

## Context

Sealgraph must track not only content history but the exact upstream generations used as the basis for each sealed state.

## Decision

A seal is immutable and commits to its direct upstream seal identities.

Logical REFs move to new seal heads when superseded.

Staleness is derived by comparing persisted dependency targets with current logical REF heads and traversing the dependency closure.

A reseal that only updates an upstream dependency still receives a new identity.

One seal operation applies to one REF only.

## Consequences

- Provenance forms a Merkle-style DAG.
- Upstream supersession propagates review work downstream.
- Repair is intentionally explicit.
- Historical state remains inspectable.
- Automatic recursive relinking/resealing is forbidden.
