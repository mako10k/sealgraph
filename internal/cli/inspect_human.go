package cli

// R: render daily observations and safely quoted read-only navigation.

import (
	"fmt"
	"github.com/mako10k/sealgraph/internal/repository"
	"io"
	"strings"
)

func shellArgument(value string) string { return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'" }

func inspectReadActions(status repository.RefStatus) []string {
	ref := shellArgument(status.REF)
	result := []string{"sealgraph status " + ref + " --format human"}
	if status.Unsealed {
		result = append(result, "sealgraph candidate show "+ref+" --format human", "sealgraph candidate compare "+ref+" --format human")
	}
	if status.Head != nil {
		result = append(result, "sealgraph show @"+status.Head.String()+" --raw-content")
	}
	for _, target := range status.StaleDirect {
		result = append(result, "sealgraph show @"+target.String()+" --raw-content", "sealgraph impact @"+target.String()+" --format human")
	}
	result = append(result, "sealgraph help workflows", "sealgraph help review prepare")
	return result
}

func printInspectHuman(out io.Writer, doc inspectDocument) {
	fmt.Fprintf(out, "DAILY INSPECTION (on demand)\nRepository format: %d; inventory complete: %t; omitted REFs: %d\n", doc.RepositoryFormat, doc.Complete, doc.Omitted)
	for _, entry := range doc.Entries {
		fmt.Fprintf(out, "\nREF %s\n", quoteHumanString(entry.Status.REF))
		fmt.Fprintf(out, "  Candidate/HEAD: %s; draft: %t\n", entry.Status.Candidate, entry.Status.Draft)
		fmt.Fprintf(out, "  Workfile: %s; baseline: %s\n", entry.Workfile.State, entry.Workfile.Baseline)
		fmt.Fprintf(out, "  Provenance stale: self=%t direct=%d transitive=%d (independent of files and review claims)\n", entry.Status.Stale.Self, len(entry.Status.Stale.Direct), len(entry.Status.Stale.Transitive))
		printInspectTrace(out, entry)
		for _, review := range entry.Reviews {
			fmt.Fprintf(out, "  Historical review [%s @%s]: %s", review.Baseline, review.SealID, review.State)
			if review.Record != nil {
				fmt.Fprintf(out, "; applicable=%t; %s", review.Record.Applicable, quoteHumanString(review.Record.Limitation))
			}
			fmt.Fprintln(out)
			if review.Record != nil {
				candidate := ""
				selector := "@" + review.SealID
				if review.Baseline == "candidate" {
					selector = shellArgument(entry.Status.REF)
					candidate = " --candidate"
				}
				fmt.Fprintf(out, "    Read recorded differences, impact and reasons: sealgraph review show %s%s --format human\n", selector, candidate)
			}
		}
		fmt.Fprintln(out, "  Next read actions:")
		for _, action := range entry.Next {
			fmt.Fprintln(out, "    "+action)
		}
	}
	fmt.Fprintln(out, "\n"+doc.Limitation)
}

func printInspectTrace(out io.Writer, entry inspectEntry) {
	fmt.Fprintln(out, "  Original-source Trace: "+entry.TraceState)
	if entry.TraceState == "UNSUPPORTED_FORMAT" {
		fmt.Fprintln(out, "    Trace requires format 7; inspect `sealgraph help migrate repository`; migration is explicit.")
	}
	for _, local := range entry.Trace {
		fmt.Fprintf(out, "    %s: %s; difference=%t unresolved=%t complete=%t\n", local.Baseline.Kind, local.TracePresence, local.HasDifference, local.HasUnresolved, local.Complete)
		for _, run := range local.Ranges {
			fmt.Fprintf(out, "      run %d source=%s: %s (%s); snapshot=%s\n", run.RunIndex, quoteHumanString(run.SourceKey), run.Presence, quoteHumanString(run.PresenceReason), run.SourceSnapshotID)
		}
	}
}
