# ADR 0043: Reviewed historical Cause publication

Status: Accepted within the explicitly delegated Issue #22 implementation scope.
Date: 2026-10-22
Authority: https://github.com/mako10k/sealgraph/issues/22 and the owner-directed implementation request of 2026-10-22. This does not authorize release, installation, push, or kn publication.

## Context and evidence

C-HR-001: exact-version reliance and current revision consistency are different claims. E-HR-001: Issue #22 requires a reviewed historical non-draft path while retaining stale facts. E-HR-002: requirements §4 and ADR 0023 candidate admission currently require every reachable Cause to be a non-draft active leaf. E-HR-003: the kn KI0 review and historical-cause publication-boundary records illustrate a reviewed but unpublishable dependency closure; they remain external read-only evidence.

C-HR-002: a review must be bound to the complete dependent subject and observation, without representing understanding or truth. E-HR-004: Issue #22 requires exact IDs, content, differences, impact, and retention reasons, including transitive dependencies and HEAD changes.

## Decision

Keep default `seal REF` admission unchanged. Add explicit `seal REF --review-historical`. It requires a review record previously attached to that exact Candidate by `review set REF --file PATH|-`. This is one selected Candidate mutation, not Seal issuance. `review prepare REF` and `review show SELECTOR` are read-only. No command creates review judgments or publishes multiple Seals.

Use existing immutable Material attachment storage, not a new Provenance field or interpreted Link metadata namespace. The dedicated attachment is `sealgraph-historical-review.json`, media type `application/vnd.sealgraph.historical-review+json`, with content schema `sealgraph/historical-review/v1`. This narrowly supersedes ADR 0021's absence of attachment authoring only for this dedicated record. Arbitrary attachment mutation remains absent. The native formats 5/6/7, canonical object codecs, and existing inspection schemas do not change. Older binaries can inspect/transport the bytes and retain their strict admission; they do not interpret this record.

The subject is the prospective SealID after removing only the dedicated review attachment. Thus it commits content, other attachments, full Cause records, root/draft, format generation, and OriginMap without self-reference. The record also binds its destination REF and every other observed REF/head pair. Its destination head is omitted from this recorded observation so publication does not immediately invalidate its own record; publication still validates expected_ref_head and the entire captured observation independently.

Each distinct reachable Cause target has one record, sorted by exact SealID. Fields are `target_seal`, `compared_seals`, `decision` (`retain` or `unreviewed`), `difference`, `impact`, and `reason`. `compared_seals` is the complete sorted active revision leaf descendant set under the prospective observation (or empty when detached with no observed lineage). This set does not assert a unique latest version. All targets, including active leaves, require an explicit entry so unreviewed transitive impact cannot disappear at a traversal shortcut. Retain requires non-empty difference/impact/reason. Unreviewed may be saved but blocks explicit historical publication. A draft Cause anywhere in the closure always blocks non-draft publication. Records and arrays reject duplicate keys/members and unknown members and are bounded to 1 MiB.

Prepare reports facts and the subject/observation, never a completed review. Set validates binding and target/comparison coverage, canonicalizes the record, stores its immutable Blob and replaces only the expected Candidate version. It never changes Causes or REF heads. Explicit historical publication revalidates the record against the prospective graph, then uses the existing guarded publication transaction. Candidate or other-head changes invalidate applicability; ordinary inspection remains possible. Stale, draft, structural graph, and frontier semantics are unchanged. A record is an executor claim, not evidence of human understanding, correctness, approval, or trusted time.

## Alternatives

- Removing leaf validation globally would admit unreviewed dependencies and was rejected.
- Draft-only historical publication conflates provisional content with decided historical reliance and was rejected by Issue #22.
- Rate limits or read tokens do not bind an individual judgment and do not implement this requirement.
- New typed Provenance fields would require a new repository format and migration; ordinary contained evidence plus explicit admission can satisfy this slice without rewriting canonical object schemas.

## Implementation and verification

A-HR-001 -> this ADR -> C-HR-001/C-HR-002: repository review preparation, bounded record codec, dedicated Candidate attachment mutation, explicit admission, CLI/help/completion, and tests. Verify direct/transitive/detached old targets, unknown decisions, draft, subject/head changes, exact readback, legacy strict path, unchanged stale and old object bytes. Existing CAS and observation-race tests remain applicable and receive focused coverage for the new path.

## Consequences and follow-ups

The conservative whole-other-head observation can require renewed review after an unrelated REF change. This is a known usability cost, not an automatic relink or invalidation of the historical judgment itself. Read-only inspection distinguishes recorded judgment from current applicability. No recurrence-prevention claim is made about an executor fabricating text; independent approval and time-based warnings remain separate proposals.

## Review

Self-review against Issue #22: no automatic judgment, repair, generic force, draft bypass, storage migration, or kn write. Full implementation and test evidence will be recorded in the Issue #22 handoff. No independent review is claimed.
