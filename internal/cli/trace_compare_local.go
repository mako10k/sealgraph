package cli

import (
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/mako10k/sealgraph/internal/repository"
	"github.com/mako10k/sealgraph/internal/tracecompare"
)

// These records are the local part of ADR 0038's compare/v2 document. The
// complete public document is assembled when graph observation is available.
type traceCompareRangeJSON struct {
	Start  int `json:"start"`
	Length int `json:"length"`
}

type traceCompareEstimateJSON struct {
	State      string                              `json:"state"`
	Method     *string                             `json:"method"`
	Candidates []traceCompareEstimateCandidateJSON `json:"candidates"`
	Reason     *string                             `json:"reason"`
}

type traceCompareEstimateCandidateJSON struct {
	CurrentRanges  []traceCompareRangeJSON `json:"current_ranges"`
	EvidenceKind   string                  `json:"evidence_kind"`
	Method         string                  `json:"method"`
	Reason         string                  `json:"reason"`
	DeclarationIDs []string                `json:"declaration_ids"`
}

type traceCompareRangeResultJSON struct {
	RunIndex           int                      `json:"run_index"`
	SourceSnapshotID   string                   `json:"source_snapshot_id"`
	SourceKey          string                   `json:"source_key"`
	OldRange           traceCompareRangeJSON    `json:"old_range"`
	CurrentBlobID      *string                  `json:"current_blob_id"`
	Presence           string                   `json:"presence"`
	PresenceReason     string                   `json:"presence_reason"`
	Examined           string                   `json:"examined"`
	SelectedMatchStart *int                     `json:"selected_match_start"`
	PositionChanged    *bool                    `json:"position_changed"`
	Estimate           traceCompareEstimateJSON `json:"estimate"`
}

type traceCompareLocalJSON struct {
	Baseline           traceShowBaseline             `json:"baseline"`
	TracePresence      string                        `json:"trace_presence"`
	ExternalRunCount   int                           `json:"external_run_count"`
	ComparedRunCount   int                           `json:"compared_run_count"`
	UnexaminedRunCount int                           `json:"unexamined_run_count"`
	HasDifference      bool                          `json:"has_difference"`
	HasUnresolved      bool                          `json:"has_unresolved"`
	Complete           bool                          `json:"complete"`
	Ranges             []traceCompareRangeResultJSON `json:"ranges"`
}

func buildTraceCompareLocal(result repository.TraceCompareOwnResult) traceCompareLocalJSON {
	baseline := traceShowBaseline{Kind: "seal", ContentBlobID: result.Content.String()}
	if result.REF != "" {
		ref := result.REF
		baseline.REF = &ref
	}
	if result.Baseline.Kind == repository.TraceOwnCandidate {
		baseline.Kind = "candidate"
		digest := hex.EncodeToString(result.CandidateDigest[:])
		baseline.CandidateDigest = &digest
	} else if result.SealID != nil {
		id := result.SealID.String()
		baseline.SealID = &id
	}
	if result.Origin != nil {
		id := result.Origin.String()
		baseline.OriginMapID = &id
	}
	local := traceCompareLocalJSON{Baseline: baseline, TracePresence: "EXTERNAL_RANGES", Complete: true, Ranges: []traceCompareRangeResultJSON{}}
	if result.Origin == nil {
		local.TracePresence = "NO_TRACE"
	} else if len(result.Runs) == 0 {
		local.TracePresence = "NO_EXTERNAL_RANGES"
	}
	for _, run := range result.Runs {
		item := traceCompareRangeResultJSON{
			RunIndex: run.RunIndex, SourceSnapshotID: run.SourceSnapshotID.String(), SourceKey: run.SourceKey,
			OldRange: traceCompareRangeJSON{Start: run.OldStart, Length: run.Length}, Presence: string(run.Presence), PresenceReason: run.PresenceReason,
			Examined: "EXAMINED", Estimate: traceCompareEstimateRecord(run.Estimate),
		}
		if run.CurrentBlobID != nil {
			id := run.CurrentBlobID.String()
			item.CurrentBlobID = &id
		}
		if run.SelectedMatchStart != nil {
			item.SelectedMatchStart = run.SelectedMatchStart
			changed := *run.SelectedMatchStart != run.OldStart
			item.PositionChanged = &changed
		}
		if !run.Examined {
			item.Examined = "NOT_EXAMINED"
			local.UnexaminedRunCount++
			local.Complete = false
		}
		switch run.Presence {
		case tracecompare.Present, tracecompare.AbsentExact:
			local.ComparedRunCount++
		}
		if run.Presence == tracecompare.AbsentExact {
			local.HasDifference = true
		}
		if run.Presence == tracecompare.Undetermined {
			local.HasUnresolved = true
		}
		local.Ranges = append(local.Ranges, item)
	}
	local.ExternalRunCount = len(local.Ranges)
	return local
}

func traceCompareEstimateRecord(value *repository.TraceOwnEstimateResult) traceCompareEstimateJSON {
	result := traceCompareEstimateJSON{State: "NOT_REQUESTED", Candidates: []traceCompareEstimateCandidateJSON{}}
	if value == nil {
		return result
	}
	result.State = value.State
	if value.Method != "" {
		method := value.Method
		result.Method = &method
	}
	if value.Reason != "" {
		reason := value.Reason
		result.Reason = &reason
	}
	for _, candidate := range value.Candidates {
		item := traceCompareEstimateCandidateJSON{CurrentRanges: []traceCompareRangeJSON{}, EvidenceKind: candidate.EvidenceKind, Method: candidate.Method, Reason: candidate.Reason, DeclarationIDs: append([]string{}, candidate.DeclarationIDs...)}
		for _, current := range candidate.CurrentRanges {
			item.CurrentRanges = append(item.CurrentRanges, traceCompareRangeJSON{Start: current.Start, Length: current.Length})
		}
		result.Candidates = append(result.Candidates, item)
	}
	return result
}

func printTraceCompareLocalHuman(out io.Writer, local traceCompareLocalJSON) {
	fmt.Fprintf(out, "%s content=%s trace=%s complete=%t\n", strings.ToUpper(local.Baseline.Kind), local.Baseline.ContentBlobID, local.TracePresence, local.Complete)
	for _, run := range local.Ranges {
		fmt.Fprintf(out, "  run %d source=%s old=[%d,%d) %s reason=%s", run.RunIndex, quoteHumanString(run.SourceKey), run.OldRange.Start, run.OldRange.Start+run.OldRange.Length, run.Presence, run.PresenceReason)
		if run.SelectedMatchStart != nil {
			fmt.Fprintf(out, " selected_start=%d (selected match; historical origin is not established)", *run.SelectedMatchStart)
		}
		if run.Estimate.State != "NOT_REQUESTED" {
			fmt.Fprintf(out, " estimate=%s", run.Estimate.State)
			if run.Estimate.Method != nil {
				fmt.Fprintf(out, " method=%s", *run.Estimate.Method)
			}
			if run.Estimate.Reason != nil {
				fmt.Fprintf(out, " reason=%s", quoteHumanString(*run.Estimate.Reason))
			}
		}
		fmt.Fprintln(out)
		for _, candidate := range run.Estimate.Candidates {
			fmt.Fprintf(out, "    candidate kind=%s method=%s ranges=%v declarations=%v reason=%s\n", candidate.EvidenceKind, candidate.Method, candidate.CurrentRanges, candidate.DeclarationIDs, quoteHumanString(candidate.Reason))
		}
	}
}
