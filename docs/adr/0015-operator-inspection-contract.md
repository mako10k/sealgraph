# ADR 0015: Operator semantics and versioned inspection output

Status: accepted for implementation on 2026-08-17 by explicit operator approval.

ADR 0019 supersedes only this ADR's working-file-related `CLEAN` presentation:
status v2 separates sealed/candidate facts from an explicit local
workfile/baseline relation. The remaining inspection terminology stays in
force.

## Context

Human inspection text was being scraped by dogfood receipts, familiar Git words
invited incorrect interpretations, and explicit `init` concealed the local
creation of missing runtime directories. Dependency links also allow one shared
message per command, so distinct rationales require distinct candidate edits.

## Decision

- `CLEAN` means no candidate and no derived stale relation. It does not compare
  working files with sealed content or a path manifest.
- REF is a movable logical identity, not a branch or checkout target.
- impact is structural Cause reachability; stale is a current-head review fact.
- root is an explicit provenance boundary, not truth or trust.
- log and linklog are Seal revision and Cause histories, not Git history.
- standalone commands inspect only explicit inputs and `.sealgraph`; `init`
  never discovers or inspects Git.

Human `status`, `impact`, and `graph` begin with `SEALED_STATE`,
`STRUCTURAL_IMPACT`, and `REVISION_CAUSE_GRAPH`. Existing status labels remain
unchanged. ADR 0010's `stale --refs-only` bytes remain unchanged.

`show`, `status`, `stale`, `graph`, `impact`, `log`, `linklog`, and `diff`
accept `--format human|json`, with human as the default. JSON documents carry a
command-specific `sealgraph/<command>/v1` schema identifier. IDs are complete
strings, paths are arrays of IDs, and direct stale targets remain distinct from
transitive stale paths. JSON contains canonical/derived repository facts only;
it excludes content bytes, environment values, filesystem paths, and implicit
Git observations. Result state exits zero; usage errors exit two and
integrity/operational failures exit one. `show --raw-content` and
`stale --refs-only` remain separate formats and cannot be combined with JSON.

Explicit `init` returns one of `INITIALIZED`, `BOOTSTRAPPED_RUNTIME`, or
`ALREADY_COMPLETE`. Bootstrap names only the created `index` and/or `locks`
directory labels, never their absolute paths. It does not rewrite canonical
objects, manifests, or config.

The repeated `link` command remains the distinct-message contract. This avoids
adding delimiter and escaping rules to the public CLI: one invocation applies
one rationale to all of its dependencies, while different rationales are
separate visible candidate edits. Every selector in an invocation is resolved
before the candidate is saved, exact target Seal IDs are normalized, and no
dependency kind taxonomy is introduced.

## Consequences

Automation can consume explicitly versioned structures without depending on
human field order. Human headings and the legend make result domains explicit.
Adding or changing a JSON field requires a compatible v1 addition or a new
schema version. No persisted object, REF, or storage-format field changes.
