# UX R1 basic and detailed design

Authority: Accepted exact R1 and owner direction to design/implement. Design decisions: ADR0044. This design allocates requirements without changing their acceptance meaning.

| Requirement / Work | Component and behavior | Evidence |
| --- | --- | --- |
| R01/R02 DAILY_INSPECTION | inspect explicit REF or local inventory; source, stale and local Trace axes; bounded visible inventory, no background activity | CLI status/Trace/format/no-write/concurrency tests |
| R03/R04 CONTENT_REVIEW | exact compare plus bounded human --text and raw extraction routes; Cause diff remains existing compare | later-than256-byte change, long/binary/computation/output-limit tests |
| R05 ORIGINAL_AUDIT | trace read explicit source version/context/raw full source and identity | multiple hits, movement fixture without move claim, unbound, interrupted/version-changed and raw-byte tests |
| R06 HISTORICAL_STATUS | inspect Candidate/HEAD recorded review and applicability independently from stale | none/retain/unreviewed/invalidated other-head tests |
| R01/R07 RUNBOOK/LLM | README/help/usecases to repository-local operations guide; six complete paths | navigation tests and independent role review; actual human novice test remains separate |

No new persisted fields. Existing JSON consumers and raw protocols retain their versions/meaning. New JSON belongs only to new commands. New command output is evidence to be read, not an authorization/judgment or mutation plan.

Before emitting inspect: capture heads; existing Status validates observed graph; bound selected REF count; capture selected Candidate exact digest; collect per-baseline local Trace via repository stable comparisons; collect review claims only when reserved attachment exists; compare status/Candidate/source identities and global heads again. Changed context fails rather than selecting a new target silently. This is not a simultaneous cross-file snapshot.

Trace read reuses the existing explicit occurrence selection and baseline/source revalidation. Snapshot/current views stay separate; raw full-source selection bypasses display truncation only on explicit request. Compare --text reuses immutable comparison plus stable exact content reads; deterministic line rendering must cap compute/output and identify omissions while retaining full extraction.

Completion and diagnostics are registry-based. Shell command suggestions quote untrusted REF/source-key arguments; user text is escaped. Tests cover missing .sealgraph help without creation, format5 unsupported Trace with explicit migration guidance, and unknown/invalid flags. Full Go/tooling validation follows integration; implementation feedback returns to this design if a contract conflict is found.

Implementation review must check real behavior, error propagation, no hidden mutation, limits/escaping and old JSON compatibility. Owner-requested design/implementation progress does not mark UX_PERSONA_REVIEW or UX_OWNER_ACCEPTANCE complete. That remains downstream evidence and owner judgment.
