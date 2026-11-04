package cli

// R: connect provenance principles to complete, explicit operator workflows.

import (
	"fmt"
	"io"
)

func printWorkflowsHelp(out io.Writer) {
	fmt.Fprint(out, `workflows — start here for people, LLM operators and auditors

1. First registration
   sealgraph init
   sealgraph help add
   Choose a logical REF and explicit root boundary, or one exact Cause.
   Read the content and exact premise, prepare one Candidate, inspect it,
   decide individually and publish one REF only. Root is not truth/approval.

2. Daily check (notifications happen only when this command runs)
   sealgraph inspect --format human
   sealgraph inspect REF --format human
   Workfile/baseline, stale provenance, original-source presence and recorded
   historical review are different facts. Unsupported format, missing binding,
   NO_TRACE and INCOMPLETE mean unobserved scope, never an all-clear.
   Inspect omitted_refs; select one omitted REF or raise --max-refs explicitly.

3. Review changed premises upstream-first
   sealgraph stale --frontier
   sealgraph show REF --raw-content
   sealgraph show @OLD_CAUSE --raw-content
   sealgraph show @COMPARISON --raw-content
   sealgraph compare @OLD_CAUSE @COMPARISON --text
   sealgraph impact @OLD_CAUSE --format human
   Read differences and impact before choosing:
   UPDATE: if no Candidate exists, prepare one with add REF --content-file PATH
           (or add REF using its already explicit source binding).
           Inspect candidate show/compare. Each new Cause and removal of the
           exact old target is a separate judged operation: help link/unlink.
   RETAIN: keep exact historical Causes; help review prepare, review set and
           seal --review-historical describe the existing individual review.
           The prepared facts are not judgments. STALE remains observable.
   PENDING: leave uncertainty visible. Do not update a Cause or issue a Seal.
   Every chosen mutation and one-REF publication is individual; read back the
   exact immutable result. Do not run these decisions in loops or scripts.

4. Audit originals, not just hashes
   sealgraph trace show --ref REF --format json
   sealgraph trace read --ref REF --baseline head --run-index 0 --view snapshot
   sealgraph trace read --ref REF --baseline head --run-index 0 --view snapshot --raw-source
   sealgraph review show REF --format human
   Candidate, HEAD, snapshot and current are explicit different selections.
   Use trace occurrences for all matching positions; a non-final page is partial.
   A representative current match or offset difference does not prove a move.
   Structural validity, copied-byte equality and semantic support are separate.
   Trace needs format 7: help migrate repository. Never migrate implicitly;
   historical NO_TRACE is not reconstructed or certified automatically.

5. LLM operator
   Follow the same read -> exact versions -> difference/impact -> individual
   judgment -> one operation -> readback path. Automate read-only preparation
   only. A Seal, claim, clean state or passing check is not truth, understanding
   or owner approval. Keep unknowns pending and report the observed scope.

Repository guide: docs/operations.md. Need syntax? sealgraph help COMMAND.
This guide offers navigation, never automatic repair or approval.
`)
}
