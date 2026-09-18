package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/mako10k/sealgraph/internal/repository"
)

// traceCompareV2Document is the ADR 0038 output shape for comparison without
// --estimate. Dispatch remains separate until the complete CLI contract exists.
type traceCompareV2Document struct {
	Schema       string                      `json:"schema"`
	Selection    traceShowSelection          `json:"selection"`
	Observation  traceCompareObservationJSON `json:"observation"`
	CandidateOwn *traceCompareCandidateJSON  `json:"candidate_own"`
	Graph        *traceCompareGraphJSON      `json:"graph"`
	GraphReason  *string                     `json:"graph_reason"`
	Limits       traceCompareLimitsJSON      `json:"limits"`
}

type traceCompareObservationJSON struct {
	REFHeads          []traceCompareREFHeadJSON `json:"ref_heads"`
	CandidateDigest   *string                   `json:"candidate_digest"`
	BindingDigest     string                    `json:"binding_digest"`
	DeclarationDigest *string                   `json:"declaration_digest"`
	Sources           []traceCompareSourceJSON  `json:"sources"`
}

type traceCompareREFHeadJSON struct {
	REF    string `json:"ref"`
	SealID string `json:"seal_id"`
}

type traceCompareSourceJSON struct {
	SourceKey  string  `json:"source_key"`
	Path       *string `json:"path"`
	ObservedAt string  `json:"observed_at"`
	BlobID     *string `json:"blob_id"`
	ByteLength *int    `json:"byte_length"`
	Error      *string `json:"error"`
}

type traceCompareCandidateJSON struct {
	Baseline traceShowBaseline     `json:"baseline"`
	Local    traceCompareLocalJSON `json:"local"`
}

type traceCompareGraphJSON struct {
	Scope      traceCompareScopeJSON   `json:"scope"`
	Center     string                  `json:"center"`
	Locals     []traceCompareLocalJSON `json:"locals"`
	Own        traceCompareLocalJSON   `json:"own"`
	Upstream   traceCompareSummaryJSON `json:"upstream"`
	Downstream traceCompareSummaryJSON `json:"downstream"`
}

type traceCompareScopeJSON struct {
	Kind          string                    `json:"kind"`
	REFHeads      []traceCompareREFHeadJSON `json:"ref_heads"`
	ExtraSeals    []string                  `json:"extra_seals"`
	ObservedSeals []string                  `json:"observed_seals"`
	Complete      bool                      `json:"complete"`
	StopReason    *string                   `json:"stop_reason"`
}

type traceCompareSummaryJSON struct {
	SealCount           int                      `json:"seal_count"`
	ExternalRunCount    int                      `json:"external_run_count"`
	ComparedRunCount    int                      `json:"compared_run_count"`
	UnexaminedRunCount  int                      `json:"unexamined_run_count"`
	NoTraceSealCount    int                      `json:"no_trace_seal_count"`
	NoExternalSealCount int                      `json:"no_external_seal_count"`
	HasDifference       bool                     `json:"has_difference"`
	HasUnresolved       bool                     `json:"has_unresolved"`
	Complete            bool                     `json:"complete"`
	Members             []traceCompareMemberJSON `json:"members"`
}

type traceCompareMemberJSON struct {
	SealID       string   `json:"seal_id"`
	Direct       bool     `json:"direct"`
	Indirect     bool     `json:"indirect"`
	DirectPath   []string `json:"direct_path"`
	IndirectPath []string `json:"indirect_path"`
}

type traceCompareLimitsJSON struct {
	MaxGraphVisits  int `json:"max_graph_visits"`
	UsedGraphVisits int `json:"used_graph_visits"`
}

func traceCompareSourceRecords(observations []repository.TraceOwnSourceObservation) []traceCompareSourceJSON {
	items := make([]traceCompareSourceJSON, 0, len(observations))
	for _, observation := range observations {
		item := traceCompareSourceJSON{SourceKey: observation.SourceKey, Path: observation.Path, ObservedAt: observation.ObservedAt.UTC().Format(time.RFC3339), ByteLength: observation.ByteLength, Error: observation.Error}
		if observation.CurrentBlobID != nil {
			id := observation.CurrentBlobID.String()
			item.BlobID = &id
		}
		items = append(items, item)
	}
	return items
}

func traceCompareREFHeads(heads []repository.DirectionGraphREFHead) []traceCompareREFHeadJSON {
	items := make([]traceCompareREFHeadJSON, 0, len(heads))
	for _, head := range heads {
		items = append(items, traceCompareREFHeadJSON{REF: head.REF, SealID: head.SealID.String()})
	}
	return items
}

func traceCompareMemberRecords(members []repository.DirectionGraphMember) []traceCompareMemberJSON {
	items := make([]traceCompareMemberJSON, 0, len(members))
	for _, member := range members {
		item := traceCompareMemberJSON{SealID: member.SealID.String(), Direct: member.Direct, Indirect: member.Indirect}
		for _, id := range member.DirectPath {
			item.DirectPath = append(item.DirectPath, id.String())
		}
		for _, id := range member.IndirectPath {
			item.IndirectPath = append(item.IndirectPath, id.String())
		}
		items = append(items, item)
	}
	return items
}

func traceCompareSummary(members []repository.DirectionGraphMember, locals map[string]traceCompareLocalJSON, scopeComplete bool) (traceCompareSummaryJSON, error) {
	summary := traceCompareSummaryJSON{Complete: scopeComplete, Members: traceCompareMemberRecords(members)}
	for _, member := range members {
		local, ok := locals[member.SealID.String()]
		if !ok {
			return traceCompareSummaryJSON{}, fmt.Errorf("observed direction member %s has no local observation", member.SealID)
		}
		summary.SealCount++
		summary.ExternalRunCount += local.ExternalRunCount
		summary.ComparedRunCount += local.ComparedRunCount
		summary.UnexaminedRunCount += local.UnexaminedRunCount
		if local.TracePresence == "NO_TRACE" {
			summary.NoTraceSealCount++
		}
		if local.TracePresence == "NO_EXTERNAL_RANGES" {
			summary.NoExternalSealCount++
		}
		summary.HasDifference = summary.HasDifference || local.HasDifference
		summary.HasUnresolved = summary.HasUnresolved || local.HasUnresolved
		summary.Complete = summary.Complete && local.Complete
	}
	return summary, nil
}

func buildTraceCompareGraph(graph repository.DirectionGraphResult, results []repository.TraceCompareOwnResult) (*traceCompareGraphJSON, error) {
	locals := make(map[string]traceCompareLocalJSON, len(results))
	for _, result := range results {
		if result.SealID != nil {
			locals[result.SealID.String()] = buildTraceCompareLocal(result)
		}
	}
	center, ok := locals[graph.Center.String()]
	if !ok {
		return nil, fmt.Errorf("graph center %s has no local observation", graph.Center)
	}
	complete := graph.ClosureComplete && graph.DirectionComplete
	scope := traceCompareScopeJSON{Kind: "observed-head-closure", REFHeads: traceCompareREFHeads(graph.REFHeads), ExtraSeals: []string{}, ObservedSeals: []string{}, Complete: complete}
	if len(graph.ExtraSeals) != 0 {
		scope.Kind = "observed-head-closure-plus-selected-seal"
		for _, id := range graph.ExtraSeals {
			scope.ExtraSeals = append(scope.ExtraSeals, id.String())
		}
	}
	for _, id := range graph.ObservedSealIDs {
		scope.ObservedSeals = append(scope.ObservedSeals, id.String())
	}
	if !complete {
		reason := graph.StopReason
		scope.StopReason = &reason
	}
	upstream, err := traceCompareSummary(graph.Upstream, locals, complete)
	if err != nil {
		return nil, err
	}
	downstream, err := traceCompareSummary(graph.Downstream, locals, complete)
	if err != nil {
		return nil, err
	}
	items := make([]traceCompareLocalJSON, 0, len(locals))
	for _, local := range locals {
		items = append(items, local)
	}
	sort.Slice(items, func(i, j int) bool { return *items[i].Baseline.SealID < *items[j].Baseline.SealID })
	return &traceCompareGraphJSON{Scope: scope, Center: graph.Center.String(), Locals: items, Own: center, Upstream: upstream, Downstream: downstream}, nil
}

func printTraceCompareV2Human(out io.Writer, doc traceCompareV2Document) {
	fmt.Fprintf(out, "TRACE COMPARE %s %s\n", strings.ToUpper(doc.Selection.Kind), quoteHumanString(doc.Selection.Requested))
	fmt.Fprintf(out, "Observation: %d REF heads; %d source files; graph visits %d/%d\n", len(doc.Observation.REFHeads), len(doc.Observation.Sources), doc.Limits.UsedGraphVisits, doc.Limits.MaxGraphVisits)
	printTraceCompareObservationHuman(out, doc.Observation)
	if doc.CandidateOwn != nil {
		fmt.Fprintln(out, "Candidate own:")
		printTraceCompareLocalHuman(out, doc.CandidateOwn.Local)
	}
	if doc.Graph == nil {
		fmt.Fprintf(out, "Graph: %s\n", valueOrNone(doc.GraphReason))
		return
	}
	fmt.Fprintf(out, "Graph: center=%s scope=%s complete=%t observed=%d", doc.Graph.Center, doc.Graph.Scope.Kind, doc.Graph.Scope.Complete, len(doc.Graph.Scope.ObservedSeals))
	if doc.Graph.Scope.StopReason != nil {
		fmt.Fprintf(out, " stop=%s", *doc.Graph.Scope.StopReason)
	}
	fmt.Fprintln(out)
	fmt.Fprintln(out, "Own:")
	printTraceCompareLocalHuman(out, doc.Graph.Own)
	printTraceCompareDirectionHuman(out, "Upstream", doc.Graph.Upstream, doc.Graph.Locals)
	printTraceCompareDirectionHuman(out, "Downstream", doc.Graph.Downstream, doc.Graph.Locals)
	fmt.Fprintln(out, "Paths shown are one shortest direct and one shortest indirect Cause path per member, not all paths. A selected exact match does not establish the historical origin position.")
}

func printTraceCompareObservationHuman(out io.Writer, observation traceCompareObservationJSON) {
	fmt.Fprintf(out, "Bindings: digest=%s\n", observation.BindingDigest)
	for _, head := range observation.REFHeads {
		fmt.Fprintf(out, "  REF %s HEAD %s\n", head.REF, head.SealID)
	}
	if observation.CandidateDigest != nil {
		fmt.Fprintf(out, "  Candidate digest=%s\n", *observation.CandidateDigest)
	}
	for _, source := range observation.Sources {
		fmt.Fprintf(out, "  Source %s path=%s observed_at=%s blob_id=%s", source.SourceKey, valueOrNone(source.Path), source.ObservedAt, valueOrNone(source.BlobID))
		if source.ByteLength != nil {
			fmt.Fprintf(out, " byte_length=%d", *source.ByteLength)
		}
		if source.Error != nil {
			fmt.Fprintf(out, " error=%s", *source.Error)
		}
		fmt.Fprintln(out)
	}
}

func printTraceCompareDirectionHuman(out io.Writer, label string, summary traceCompareSummaryJSON, locals []traceCompareLocalJSON) {
	fmt.Fprintf(out, "%s: seals=%d compared_runs=%d unexamined_runs=%d difference=%t unresolved=%t complete=%t\n", label, summary.SealCount, summary.ComparedRunCount, summary.UnexaminedRunCount, summary.HasDifference, summary.HasUnresolved, summary.Complete)
	byID := make(map[string]traceCompareLocalJSON, len(locals))
	for _, local := range locals {
		if local.Baseline.SealID != nil {
			byID[*local.Baseline.SealID] = local
		}
	}
	for _, member := range summary.Members {
		fmt.Fprintf(out, "  Seal %s direct=%t indirect=%t", member.SealID, member.Direct, member.Indirect)
		if member.Direct {
			fmt.Fprintf(out, " direct_path=%s", strings.Join(member.DirectPath, " -> "))
		}
		if member.Indirect {
			fmt.Fprintf(out, " indirect_path=%s", strings.Join(member.IndirectPath, " -> "))
		}
		fmt.Fprintln(out)
		if local, ok := byID[member.SealID]; ok {
			printTraceCompareLocalHuman(out, local)
		}
	}
}

func valueOrNone(value *string) string {
	if value == nil {
		return "none"
	}
	return *value
}
