# ADR 0022: Terminal-first human output and non-terminal machine output

Status: Accepted

Date: 2026-08-24

Accepted: 2026-08-24 by explicit operator request.

## Context

The versioned inspection JSON accepted by ADR 0015 already gives automation a
stable contract with complete object identities and structured graph paths.
The default human text nevertheless retained the narrow line protocols from
early dogfooding. In ordinary terminal use it exposed complete 64-character
hashes, mixed labels and values on long lines, and represented graph structure
mostly as repeated flat records. Long REFs, paths, and Cause paths routinely
exceeded an 80-to-100-column terminal.

This made a presentation-only format behave like an unversioned machine
protocol even though ADR 0015 explicitly says automation should consume the
versioned JSON instead of scraping human field order.

## Decision

### Output selection

For commands that support versioned `--format human|json` output, omission of
`--format` selects:

- human output when stdout is a terminal;
- the existing command-specific versioned JSON when stdout is a regular file,
  pipe, or other known non-terminal destination.

`--format human` and `--format json` always override automatic selection.
Unknown in-process writers retain human output so embedding and unit tests do
not silently become machine contracts.

Explicit bytes-only and narrow protocols take precedence over destination
detection. `show --raw-content`, `candidate show --raw-content`, and
`stale --refs-only` retain their exact existing bytes. Combining an explicitly
requested JSON format with one of those protocols remains an error.

Candidate inspection gains versioned `sealgraph/candidate-show/v1` and
`sealgraph/candidate-compare/v1` JSON documents so redirected candidate
inspection is structured as well.

### Human presentation

Human output uses these presentation rules:

- related scalar values use aligned labels;
- collections use titled, aligned columns;
- Seal and object hashes use 12-character presentation prefixes;
- JSON retains every complete 64-character identity;
- Cause children and impact path hops use indentation to show hierarchy;
- internal enum spellings are rendered as readable lower-case phrases where
  the versioned JSON retains the exact machine value;
- safely escaped previews remain bounded, and may be clipped further to the
  available display width;
- table columns shrink and long display values are rune-safe clipped to the
  detected terminal width, with East Asian wide characters counted as two
  columns.

Mutation commands without a versioned JSON schema use the same human label and
short-hash presentation on a terminal. Their existing full-identity narrow
receipts remain unchanged for non-terminal stdout, so automation and captured
operation receipts do not lose exact identities.

The headings required by ADR 0015 remain `SEALED_STATE`,
`STRUCTURAL_IMPACT`, and `REVISION_CAUSE_GRAPH`. The stable
`stale --refs-only` stream and every JSON schema predating this ADR remain
byte-contract concerns independent from the human renderer.

### Compatibility boundary

This decision changes only presentation and destination-based default format
selection. It does not change canonical objects, candidate or REF storage,
Seal identity, selector resolution, graph derivation, exit status, or an
existing JSON field.

One-line mutation receipts that do not yet have a versioned JSON schema remain
their existing exact non-terminal output and gain a presentation-only terminal
view. Source-binding and recovery mutations already having a versioned JSON
document follow automatic destination selection; automatic selection otherwise
applies only where versioned structured output exists.

## Claims and evidence

- **C-0022-1:** Full identities and flat key/value records are unnecessary in
  human inspection output. **Evidence E-0022-1:** ADR 0015 already assigns full
  identities and structured paths to versioned JSON and declares human field
  order presentation-only.
- **C-0022-2:** Terminal width can be honored without weakening repository
  validation. **Evidence E-0022-2:** Presentation occurs in `internal/cli`
  after repository commands have buffered and revalidated their factual
  results.
- **C-0022-3:** Non-terminal auto-JSON does not remove exact extraction modes.
  **Evidence E-0022-3:** `--raw-content` and `--refs-only` remain separate
  parser branches and are tested independently from destination detection.

## Alternatives

### Keep human output as the default everywhere

Rejected. Scripts would still be tempted to scrape a presentation format, and
redirected output would retain shortened identities by default.

### Print full identities below every shortened identity

Rejected. It would defeat the terminal readability goal. Operators can request
JSON for complete identities, and displayed Seal prefixes remain valid input
only when repository-wide selector resolution confirms uniqueness.

### Add color and interactive paging

Deferred. Color and paging introduce environment, accessibility, and process
control decisions that are not required for aligned width-aware text.

## Consequences

- Ordinary terminal inspection is compact and scan-friendly.
- Terminal mutation receipts are readable and abbreviated, while redirected
  receipts retain exact full identities.
- Redirected inspection is structured without requiring an extra option.
- A script that previously relied on omitted `--format` while redirecting
  human inspection text must add `--format human`; scripts should normally use
  the now-default JSON instead.
- Human output is explicitly not a stable parsing protocol and may change
  presentation without a JSON schema revision.
- Candidate inspection now has a versioned machine contract.

## Implementation notes

- **A-0022-1:** Route automatic format selection through the stdout
  destination and preserve explicit format precedence.
- **A-0022-2:** Centralize display-width calculation, clipping, aligned tables,
  and shortened identity rendering in `internal/cli`.
- **A-0022-3:** Cover short hashes, hierarchy indentation, aligned columns,
  terminal width, redirected JSON, and explicit-human override with CLI tests.
- **A-0022-4:** Update CLI help and normative documentation in the same change.

Actions A-0022-1 through A-0022-4 implement accepted ADR 0022 and are justified
by claims C-0022-1 through C-0022-3 and evidence E-0022-1 through E-0022-3.

## Review

The review boundary is presentation-only. Full repository reads and coherent
snapshot revalidation still complete before either human or JSON emission.
No human abbreviation is persisted or fed back into graph computation.

## Follow-ups

- Add versioned JSON to mutation commands only when a concrete consumer needs
  it; do not treat a human receipt as a schema.
- Consider optional color only under a separate accessibility and non-terminal
  behavior decision.
