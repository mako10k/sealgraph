package cli

// R: explain observation dimensions and exact evidence without implying approval.

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/repository"
)

func printObserveHuman(out io.Writer, doc repository.ObservationResult) {
	fmt.Fprintf(out, "OBSERVATION %s scope=%s\n", doc.Schema, doc.Scope)
	fmt.Fprintln(out, "Recorded evidence only: no observed successor does not prove nonexistence. Cause reachability does not confer active membership.")
	fmt.Fprintln(out, "Reasons explain review targets; they do not change legacy stale membership or publication conditions.")
	for _, ref := range doc.REFs {
		printObserveREF(out, ref)
	}
	for _, seal := range doc.Seals {
		printObserveSeal(out, seal)
	}
	fmt.Fprintln(out, "\nNext read: sealgraph inspect (workfiles/Trace/review); sealgraph help observe; sealgraph help selectors")
}
func printObserveREF(out io.Writer, ref repository.ObservationREF) {
	fmt.Fprintf(out, "\nREF %s\n  HEAD=%s draft=%s\n", quoteHumanString(ref.REF), observationID(ref.Head), observationBool(ref.HeadDraft))
	if ref.Candidate == nil {
		fmt.Fprintln(out, "  Candidate=ABSENT")
	} else {
		c := ref.Candidate
		fmt.Fprintf(out, "  Candidate=PRESENT draft=%t prospective=%s digest=%s\n", c.Draft, c.ProspectiveSeal, c.ExactDigest)
		fmt.Fprintf(out, "    current HEAD comparison=%s\n    publication baseline expected_ref_head=%s state=%s\n", c.CurrentHeadComparison, observationID(c.ExpectedREFHead), c.ExpectedHeadState)
		fmt.Fprintf(out, "    diagnostics=%s (not admission simulation)\n", c.DiagnosticBasis)
		printObserveReasons(out, c.Reasons)
		fmt.Fprintf(out, "    Read expected baseline: sealgraph candidate compare %s\n", shellArgument(ref.REF))
	}
	if ref.Source == nil {
		fmt.Fprintln(out, "  Workfile binding=ABSENT")
	} else {
		fmt.Fprintf(out, "  Workfile path=%s baseline=%s relation=%s\n", quoteHumanString(ref.Source.Path), ref.Source.Baseline, ref.Source.Relation)
	}
	fmt.Fprintf(out, "  Legacy stale self=%t direct=%v transitive=%v\n", ref.Legacy.Self, ref.Legacy.Direct, ref.Legacy.Transitive)
}
func printObserveSeal(out io.Writer, seal repository.ObservationSeal) {
	fmt.Fprintf(out, "\nSeal %s active=%t active_leaf=%t\n", seal.SealID, seal.Active, seal.ActiveLeaf)
	fmt.Fprintf(out, "  Observed successors=%v\n", seal.Successors)
	printObservationEvidence(out, "Previous assertions", seal.Previous)
	printObservationEvidence(out, "Successor assertions (exact observer / target / previous)", seal.SuccessorAssertions)
	printObserveReasons(out, seal.Reasons)
	fmt.Fprintf(out, "  Read immutable original: sealgraph show @%s\n", seal.SealID)
}
func printObserveReasons(out io.Writer, reasons []repository.ObservationReason) {
	if len(reasons) == 0 {
		fmt.Fprintln(out, "  Review reasons=NONE in this observation scope")
		return
	}
	for _, reason := range reasons {
		fmt.Fprintf(out, "  Reason %s subject=%s target=%s direct=%t paths_complete=%t\n", reason.Kind, reason.Subject, reason.Target, reason.Direct, reason.PathsComplete)
		for _, path := range reason.Paths {
			values := make([]string, len(path))
			for i, id := range path {
				values[i] = id.String()
			}
			fmt.Fprintf(out, "    Cause path: %s\n", strings.Join(values, " -> "))
		}
		if !reason.PathsComplete {
			fmt.Fprintln(out, "    Witness paths omitted; reason membership remains complete. Increase --max-paths to inspect more.")
		}
	}
}
func printObservationEvidence(out io.Writer, label string, value any) {
	// JSON encoding escapes all untrusted text, including assertion messages.
	data, _ := json.MarshalIndent(value, "  ", "  ")
	fmt.Fprintf(out, "  %s:\n  %s\n", label, data)
}
func observationID(id *domain.ObjectID) string {
	if id == nil {
		return "ABSENT"
	}
	return id.String()
}
func observationBool(value *bool) string {
	if value == nil {
		return "ABSENT"
	}
	return fmt.Sprint(*value)
}
