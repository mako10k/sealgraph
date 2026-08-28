# Cause-scoped revision decision and review record — 2026-08-28

Status: review evidence and operator-direction record for Proposed ADRs 0023,
0025, 0026, and 0027. This record does not accept an ADR or authorize
implementation, migration, publication, or external writes.

## Operator directions recorded

The following directions were given during the design discussion and are the
source for the corresponding Proposed ADR evidence entries. Exact ADR-owner
acceptance remains pending until the final candidate bytes are reviewed and
confirmed.

- **D-CR-001:** Seal has no intrinsic parent/child relation. A Cause Link may
  assert zero or more previous revisions of its exact target.
- **D-CR-002:** Old global parent meaning that no Cause Link observes may be
  lost during migration, with deterministic warning and receipt evidence.
- **D-CR-003:** Selector, comparison, history, and impact behavior must be
  redefined for the new graph rather than inheriting hidden format-4 defaults.
- **D-CR-004:** Impact defaults to the all-observer structural revision union;
  repeated explicit observer filters restrict Revision assertions only, and
  output identifies the scope and supporting assertion sources.
- **D-UB-001:** The physical Blob layer should contain exact bytes and identity
  only. Seal and Provenance are typed Blob instances, and a Seal contains only
  the minimum Material and Provenance identities.
- **D-UB-002:** RefGraph remains theory-validation work; Sealgraph is the
  operational model and has no RefGraph runtime dependency.
- **D-MG-001:** Format 5 uses explicit one-way migration rather than a runtime
  compatibility layer.
- **D-MG-002:** A revision assertion whose endpoints collapse to one new SealID
  is removed as warning-backed semantic loss rather than emitted as an invalid
  self-edge.
- **D-MG-003:** Migration and receipt bytes have one exact canonical form.
- **D-MG-004:** Repository publication and stdout receipt delivery are separate
  states; a durable in-repository receipt and read-only recovery command bridge
  output failure.
- **D-REVIEW-001:** After the committed re-review, all supported findings other
  than the separately discussed impact finding are to be corrected in the same
  Proposed decision set.
- **D-IO-001:** After the subsequent exact review, the operator directed the
  proposed corrections to proceed: graph semantics remain in ADR 0023,
  storage in ADR 0025, migration in ADR 0026, and exact format-5 CLI/inspection
  contracts in new ADR 0027. This direction authorizes drafting and validation,
  not acceptance or implementation of the resulting exact bytes.

## Exact committed re-review target

The three independent read-only reviews verified this exact target before and
after review:

```text
commit  df113d7db826977a3b0b41d0fd7474f155a35e40
ADR0023 7aad31a68c1268a3a5cf77ffb5499d0a74e0eae582c10e8e44fef5d4adbfed21
ADR0025 bf82a620fd39ba43b998fba5477ed8b2c25019723eb38afaa0df19d79c1cb512
ADR0026 0ec5c97f1650fa63b6a22b8888629ac521602d2b96239b5a4a7a5518d5eb4860
```

Scope verdicts were `FAIL` for ADR-internal, related-ADR, and repository-wide
review. No reviewer edited files.

## Integrated finding inventory

Severity is the highest supported severity across the three scopes. Findings
are deduplicated only where the creating mechanism and corrective action are
the same.

| ID | Severity | Creating mechanism | Required correction | Candidate disposition |
| --- | --- | --- | --- | --- |
| RV-01 | P1 | ADR 0023 removed parent ancestry without an `impact` successor. | Define source closure, membership, path, filtering, evidence, and precedence. | Addressed by C-CR-011 and A-CR-007; re-review pending. |
| RV-02 | P1 | ADR 0026 named a stable export observation without an operational transaction. | Double-capture exact config, REF/tag manifests, object inventory/bytes, and Candidate namespace around buffered output. | Addressed by C-MG-002 and A-MG-001; re-review pending. |
| RV-03 | P1 | The constant exclusion taxonomy omitted recovery journal state. | Name recovery journal explicitly in document and receipt exclusions. | Addressed by C-MG-003 and A-MG-001/A-MG-003; re-review pending. |
| RV-04 | P1 | Canonical JSON escaping and dependency-first topological ordering allowed multiple byte forms. | Define exact scalar escaping and one ready-set ordering algorithm with fixed vectors. | Addressed by C-UB-006, C-MG-003, A-UB-002, and A-MG-001/A-MG-003; re-review pending. |
| RV-05 | P1 | New selector wording did not explicitly prevent migration of an unreachable lower-hex tag name. | Bind the existing `[0-9a-f]{4,64}` TAGNAME reservation and reject violations. | Addressed in ADRs 0023 and 0026; re-review pending. |
| RV-06 | P2 | Projection totality relied on the format-4 non-root Cause invariant without stating it. | Validate root has no Cause and every non-root, including draft, has at least one. | Addressed by C-MG-002/C-MG-004; re-review pending. |
| RV-07 | P2 | `source_repository.object_format` was an unconstrained string. | Require exactly `sha256`. | Addressed by C-MG-003; re-review pending. |
| RV-08 | P2 | Importer canonical validation, recomputation, and warning parity were missing from action traceability. | Map C-MG-003 through C-MG-006 to the importer and require exact importer evidence. | Addressed by A-MG-003 and the trace table; re-review pending. |
| RV-09 | P2 | “No format-4 reader” contradicted strict verification of embedded old payloads. | Separate ordinary format-5 repository decoding from a migration-document-only format-4 verifier. | Addressed by C-MG-001/A-MG-003; re-review pending. |
| RV-10 | P2 | Operator directions and prior review claims had no repository-resolvable evidence locator. | Commit the reviewed target, findings, directions, and acceptance boundary. | This record is the locator; exact ADR acceptance remains pending. |
| RV-11 | P1 | Atomic rename was conflated with nested-tree and destination-entry durability. | Define final modes, file sync, bottom-up directory sync, no-replace rename, parent sync, and a post-rename durability-uncertain state. | Addressed by C-MG-007 and A-MG-003/A-MG-004; re-review pending. |
| RV-12 | P2 | PLAN called a format-5 Candidate schema accepted while ADR 0025 deferred it. | Define exact Candidate fields and reader/writer bytes or add another gate. | Addressed by C-UB-011/A-UB-003 and PLAN synchronization; re-review pending. |
| RV-13 | P2 | Branching `linklog` did not define comparison pairs, ordering, shared-edge behavior, or sources. | Compare every unique reachable revision edge, without preferred parent, and retain assertion sources. | Addressed by C-CR-012/A-CR-008; re-review pending. |

The repository-wide review also asked whether attachment names remain
non-empty. The Proposed format-5 contract now explicitly preserves the
format-4 non-empty valid-UTF-8 and unique-name invariant.

Original line evidence in commit `df113d7`:

- **RV-01:** ADR 0023:148-150, 183-200, 286-302; ADR 0011:359-391.
- **RV-02:** ADR 0026:64-82, 133-174.
- **RV-03:** ADR 0026:138-174, 478-488, 527-530.
- **RV-04:** ADR 0025:117-132; ADR 0026:86-93, 149-167, 348-357.
- **RV-05:** ADR 0023:190-200; ADR 0026:133-136, 228-230.
- **RV-06:** ADR 0025:172-175; ADR 0026:64-82, 178-184.
- **RV-07:** ADR 0026:121-131.
- **RV-08:** ADR 0026:86-93, 163-165, 275-287, 610-616, 655-666.
- **RV-09:** ADR 0026:32-53, 178-184, 301-303.
- **RV-10:** ADR 0023:402-410; ADR 0025:363-370; ADR 0026:645-651.
- **RV-11:** ADR 0026:306-346; ADR 0012:95-99; ADR 0013:103-109.
- **RV-12:** ADR 0025:182-193; `PLAN.pert`:251-256.
- **RV-13:** ADR 0023:45-48, 164-171; `docs/cli.md`:411-413.

## Exact subsequent working-candidate re-review target

Three independent read-only reviewers verified the following working-candidate
bytes before and after review. HEAD remained at `df113d7`; the modified ADRs,
review record, and PLAN were reviewed by digest rather than represented as a
commit.

```text
HEAD      df113d7db826977a3b0b41d0fd7474f155a35e40
ADR0023   2afbe7c5a62550b310772748916bcd5265313813024f03e0bb131ed030609a0e
ADR0025   911281ba66ef1a5ce0dcd476eddd65cef8d20bd39c5f2964d8452192203d1357
ADR0026   6d6331267464e53438a6b445001acbd222254bf9491ffd8533de3809a7b10dda
REVIEW    e3fb1df72dbc77626f1b2d08726f51a88cf333df0dfbbeb17cca079d3e9e3dfb
PLAN      dae88a044a281072716b73125bc18c68b141913521ac4412e78da67211c5d1de
```

Scope verdicts were again `FAIL` for ADR-internal, related-ADR, and
repository-wide review. No reviewer edited files. The integrated findings are:

| ID | Severity | Creating mechanism | Required correction | Candidate disposition |
| --- | --- | --- | --- | --- |
| RV-14 | P1 | The formal impact first-match quantifier included `v_0`, contradicting the stated strict-path case for a source-set current head. | Exclude `v_0` from the intermediate first-match test and explain zero-edge versus strict-path behavior. | Addressed in ADR 0023 C-CR-011; re-review pending. |
| RV-15 | P1 | Graph/selector snapshot captured REF heads but not scoped tags or the repository-wide prefix domain used by selectors. | Capture exact manifest bytes and conditionally capture/revalidate the complete canonical SealID inventory. | Addressed in ADR 0023 C-CR-003/C-CR-006; re-review pending. |
| RV-16 | P1 | Format-5 semantics removed parent fields without a version transition for all affected public JSON. | Define a deterministic support matrix and exact successor schemas without reusing incompatible v1 identifiers. | Addressed by ADR 0027 C-IO-001 through C-IO-003; re-review pending. |
| RV-17 | P1 | Branching `log` specified sibling order but not membership, shared-node handling, path order/bounds, or a machine successor. | Define unique-node default order, complete edge evidence, bounded all-path enumeration, and `log/v2`. | Addressed by ADR 0023 C-CR-013 and ADR 0027 C-IO-004; re-review pending. |
| RV-18 | P2 | `impact/v2` and `linklog/v2` named semantic contents without exact JSON records, types, nullability, and ordering. | Define shared records and every exact top-level/nested shape. | Addressed by ADR 0027 C-IO-002/C-IO-004; re-review pending. |
| RV-19 | P2 | “non-hex” was not the exact complement of `[0-9a-f]{4,64}`, leaving short and long all-hex tokens undefined. | Make regex match the revision branch and every other valid TAGNAME the tag branch; reject the remaining domain. | Addressed in ADR 0023 selector grammar; re-review pending. |
| RV-20 | P2 | C-MG-009 assigned source-retention responsibility only to receipt implementation. | Assign read-only source and destination-only path obligations to exporter/importer actions as well as receipt recovery. | Addressed by ADR 0026 A-MG-001/A-MG-003/A-MG-004; re-review pending. |
| RV-21 | P2 | E-CR-006 and E-UB-007 attributed decisions/findings beyond what the review record could prove. | Bind E-CR-006 to accepted inspection evidence and narrow E-UB-007 so only D-MG-002 is operator-directed; leave other Proposed clauses pending acceptance. | Addressed in ADRs 0023 and 0025; re-review pending. |
| RV-22 | P2 | Cause Link CLI options did not map previous revisions/messages to one target or define replace/union behavior. | Use one-target whole-record operations with mutually exclusive previous/no-previous input. | Addressed by ADR 0027 C-IO-005; re-review pending. |
| RV-23 | P2 | PLAN could reach READY without synchronizing format-4 normative documentation to format 5. | Add an explicit normative synchronization milestone and gate. | Addressed in PLAN; re-review pending. |
| RV-24 | P3 | ADR 0023 rejected ancestry repoint inference without superseding ADR 0006's accepted repoint wording. | Preserve ADR 0006 ID/tag/message rules and explicitly supersede only ancestry-based repoint inference. | Addressed in ADR 0023 precedence and metadata; re-review pending. |

Original line evidence in the reviewed working candidate:

- **RV-14:** ADR 0023:319-340.
- **RV-15:** ADR 0023:94-95, 140-147, 236-245; ADR 0026:89-92.
- **RV-16:** ADR 0015:36-44, 60-63; ADR 0022:43-45, 69-79; ADR
  0023:201-205, 253-263, 366-377; ADR 0025:239-245;
  `internal/cli/inspection_json.go`:90-107, 236-238, 266-271, 312.
- **RV-17:** ADR 0023:165-168, 616-623; `docs/cli.md`:392-394.
- **RV-18:** ADR 0023:201-205, 366-377.
- **RV-19:** ADR 0023:236-245.
- **RV-20:** ADR 0026:589-602, 734-737, 800.
- **RV-21:** ADR 0023:628-640, 659-662; ADR 0025:411-419, 441-444;
  this record:30-76.
- **RV-22:** ADR 0023:453-472; ADR 0025:215-229;
  `internal/cli/help.go`:72-76.
- **RV-23:** ADR 0023:698; ADR 0025:470; ADR 0026:807;
  `PLAN.pert`:251-263.
- **RV-24:** ADR 0006:127-131; ADR 0023:9-11, 192-199.

## Review and authority boundary

`Candidate disposition` means that the current working candidate contains a
proposed correction. It is not a finding closure. Closure requires a fresh
three-scope review over exact new ADR bytes, followed by explicit owner
acceptance. Any material edit changes the digests and invalidates a prior PASS
or owner-decision request.
