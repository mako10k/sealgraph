# ADR 0044: Read-only operator inspection, text comparison and original-source reading

Status: Accepted within the owner's 2026-11-04 direction to design and implement Accepted UX R1. This delegates implementation choices within R1; it does not claim a separate owner review of this ADR or product acceptance.

## Context

C-UX-1: users need one inspection entry that separates workfile state, immutable provenance stale, Trace presence, and historical-review claims. E-UX-1: Accepted R1 SHA-256 `2a26ef6abac792f74ec4c6508a78b30e2c323b589af92563f3fc8c2796d06e8d`, acceptance receipt and independent review under docs/process. E-UX-2: existing status/v3 intentionally omits Trace and historical-review applicability; show previews 256 bytes; trace show human output summarizes identities.

C-UX-2: inspectability must preserve exact identity, raw bytes, existing protocols and individual judgment. E-UX-3: Accepted ADRs 0019, 0022, 0036–0043 and R1 UX-R03–07. All are retained; this ADR only adds read-only presentation surfaces.

## Decision

- `inspect [REF] [--max-refs N] [--format human|json]` summarizes explicit local repository inventory, default at most 100 REFs. It opens only registered source/Trace bindings. It does not discover arbitrary files. New JSON schema is `sealgraph/inspect/v1`; existing status/Trace/review schemas are untouched. Every entry separates sealed state, source/baseline relation, local Trace Candidate/HEAD observations, and historical claims by baseline. Unsupported format, no trace, no binding, incomplete, missing baseline, and failures cannot become a combined clean/approved result.
- `inspect` assembles existing read-only repository observations, captures all REF heads before and revalidates after assembly, captures Candidate digests and explicit source comparison identities, and rejects detected context changes before output. Individual file reads are stable; no simultaneous filesystem snapshot or future reservation is promised. The REF display cap never bypasses complete graph integrity validation. Omitted count and completeness are explicit. Trace uses local one-hit presence and leaves directional investigation to explicit `trace compare`/`impact` routes.
- `compare FROM TO --text` is an explicit human-only supplement to existing immutable comparison. Existing omitted-text human/JSON behavior remains. Each input preview and the difference-line output budget are bounded at 64 KiB; computation is bounded independently and any omitted/context fallback is labelled. It always supplies exact raw-extraction commands. Binary and large inputs do not silently appear equal. Concrete deterministic diff thresholds are owned by the implementation's documented constants, covered by limit tests.
- `trace read` selects one External run with an immutable Seal or explicit REF baseline, one snapshot/current view, optional context-line and byte bounds (3 and 65536 by default). It uses LoadTraceOccurrenceInput and its revalidation contract. Output includes exact subject/source identities and safely escaped context. Current exact presence is a representative hit, not an editorial move/identity proof. Missing/unreadable source produces incomplete observation. `--raw-source` explicitly exports the complete selected source, has bytes-only stdout and fails for an unavailable source. It conflicts with explicit JSON and does not imply authoring.
- New trace-read JSON is `sealgraph/trace-read/v1`. All new arguments are discoverable in help/completion; existing JSON versions, raw protocols, persisted formats 5/6/7 and Seal identities remain unchanged.
- Historical-review presentation shows recorded judgments separately from applicability and stale; no interpretation of arbitrary Link metadata is added. No automatic Candidate, Cause, migration, review judgment, Seal, periodic polling, notification delivery or Git inspection occurs.

## Detailed responsibilities and error boundaries

`internal/cli/inspect*.go` assembles inventory, selection, revalidation, summaries and explicit next-read actions; it delegates semantic validation to repository APIs. `compare_text.go` owns bounded human text comparison only. `trace_read.go` owns selected original-source context/extraction only, delegating typed-origin validation and version revalidation to repository. CLI dispatcher/registry expose them; repository canonical codecs and writers are untouched. Every modified/new source module retains an R: responsibility comment.

Integrity/operational/concurrent-change errors produce stderr and nonzero with no misleading completed output. Available partial Trace facts remain labelled incomplete, never silently reset to absence. JSON buffers a complete new-schema document. Paths/content/control characters are escaped in human previews; exact bytes require the explicit raw mode. Source keys are opaque, not filesystem paths. Human next-action command arguments must be safely shell-quoted for exact REFs/selectors; they are inspection suggestions, never generated Cause/Seal mutation commands.

## Alternatives

Changing status/v3 or existing compare JSON would disturb consumers and mix different meanings. A new persisted review/notification state would require migration and is unnecessary. GUI/TUI, watcher and periodic messages are outside Accepted N1. Shelling out to Git/diff would add runtime assumptions, so domain reads and bounded Go rendering remain local.

## Consequences and verification

A-UX-1 → C-UX-1/E-UX-1/2: inventory/Trace/review inspection and exact routing. A-UX-2 → C-UX-2/E-UX-3: text differences and full source context/extraction, schema and help tests, binary/long/partial/multiple-hit fixtures and no-write assertions. A-UX-3: repository-local user/LLM runbooks and six scenario checks. Require repository validation commands, independent implementation review and separately report real-user testing/owner product acceptance as unperformed until evidence exists.

## Review and follow-ups

The independent R1 requirement review found no mandatory requirement revision. Its warnings about movement inference, format5→7 explicit navigation and real-user evidence are retained here. Independent code review follows implementation. Remaining questions about release/install and actual novice testing are not solved by this ADR. No independent owner approval of implementation or trusted audit attestation is claimed.
