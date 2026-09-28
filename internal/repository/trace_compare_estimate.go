package repository

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/tracecompare"
	"github.com/mako10k/sealgraph/internal/traceestimate"
)

const declaredTraceEstimateMethod = "declared-correspondence-v1"

type TraceOwnEstimateCandidate struct {
	CurrentRanges  []traceestimate.Range
	EvidenceKind   string
	Method         string
	Reason         string
	DeclarationIDs []string
}

type TraceOwnEstimateResult struct {
	State      string
	Method     string
	Candidates []TraceOwnEstimateCandidate
	Reason     string
}

// TraceCompareOwnBatchEstimated enriches exact presence without re-reading
// current files. Declaration records are an explicit fixed input; callers
// must recheck the all-declaration digest before publishing the document.
func (r *Repository) TraceCompareOwnBatchEstimated(ctx context.Context, selections []TraceOwnBaseline, declarations []TraceCorrespondenceRecord) ([]TraceCompareOwnResult, []TraceOwnSourceObservation, error) {
	enrich := func(results []TraceCompareOwnResult, _ []LoadedTraceOrigin, state traceOwnBatchSourceState) error {
		for resultIndex := range results {
			for runIndex := range results[resultIndex].Runs {
				run := &results[resultIndex].Runs[runIndex]
				estimate, err := estimateTraceOwnRun(*run, state, declarations)
				if err != nil {
					return err
				}
				run.Estimate = &estimate
			}
		}
		return nil
	}
	return r.traceCompareOwnBatch(ctx, selections, enrich)
}

func estimateTraceOwnRun(run TraceOwnRunResult, state traceOwnBatchSourceState, declarations []TraceCorrespondenceRecord) (TraceOwnEstimateResult, error) {
	if run.Presence != tracecompare.AbsentExact {
		return TraceOwnEstimateResult{State: "NOT_APPLICABLE", Candidates: []TraceOwnEstimateCandidate{}}, nil
	}
	source, ok := state.sources[run.SourceSnapshotID.String()]
	if !ok {
		return TraceOwnEstimateResult{}, fmt.Errorf("estimate run %d missing SourceSnapshot %s", run.RunIndex, run.SourceSnapshotID)
	}
	current, ok := state.current[run.SourceKey]
	if !ok || run.CurrentBlobID == nil {
		return TraceOwnEstimateResult{}, fmt.Errorf("estimate run %d has no stable current bytes after ABSENT_EXACT", run.RunIndex)
	}
	matching, err := matchingTraceDeclarations(run, current, declarations)
	if err != nil {
		return TraceOwnEstimateResult{}, err
	}
	if len(matching) != 0 {
		return estimateFromTraceDeclarations(matching, source.Content, run, current), nil
	}
	return estimateFromSingleDiff(source.Content, run, current)
}

func matchingTraceDeclarations(run TraceOwnRunResult, current []byte, declarations []TraceCorrespondenceRecord) ([]TraceCorrespondenceRecord, error) {
	matching := []TraceCorrespondenceRecord{}
	for _, item := range declarations {
		record := item.Record
		if !record.SourceSnapshot.Equal(run.SourceSnapshotID) || record.SourceStart != uint64(run.OldStart) || record.Length != uint64(run.Length) || !record.CurrentBlob.Equal(*run.CurrentBlobID) {
			continue
		}
		for _, currentRange := range record.CurrentRanges {
			if currentRange.Start > uint64(len(current)) || currentRange.Length > uint64(len(current))-currentRange.Start {
				return nil, fmt.Errorf("trace correspondence %s current range is outside stable current bytes", item.ID)
			}
		}
		matching = append(matching, item)
	}
	return matching, nil
}

func estimateFromTraceDeclarations(records []TraceCorrespondenceRecord, source []byte, run TraceOwnRunResult, current []byte) TraceOwnEstimateResult {
	candidates := []TraceOwnEstimateCandidate{}
	for _, item := range records {
		index := -1
		for i := range candidates {
			if sameTraceEstimateRanges(candidates[i].CurrentRanges, item.Record.CurrentRanges) {
				index = i
				break
			}
		}
		if index < 0 {
			ranges := make([]traceestimate.Range, 0, len(item.Record.CurrentRanges))
			for _, value := range item.Record.CurrentRanges {
				ranges = append(ranges, traceestimate.Range{Start: int(value.Start), Length: int(value.Length)})
			}
			reason := item.Record.Reason
			if item.Record.Deleted {
				reason += "; declared deletion"
			} else {
				joined := make([]byte, 0)
				for _, value := range item.Record.CurrentRanges {
					joined = append(joined, current[int(value.Start):int(value.Start+value.Length)]...)
				}
				if bytes.Equal(joined, source[run.OldStart:run.OldStart+run.Length]) {
					reason += "; declared concatenated bytes equal old range"
				} else {
					reason += "; declared concatenated bytes differ from old range"
				}
			}
			candidates = append(candidates, TraceOwnEstimateCandidate{CurrentRanges: ranges, EvidenceKind: "declared_correspondence", Method: declaredTraceEstimateMethod, Reason: reason, DeclarationIDs: []string{item.ID}})
			continue
		}
		candidates[index].DeclarationIDs = append(candidates[index].DeclarationIDs, item.ID)
	}
	for i := range candidates {
		sort.Strings(candidates[i].DeclarationIDs)
	}
	result := TraceOwnEstimateResult{State: "CANDIDATES", Candidates: candidates}
	if len(candidates) > 1 {
		result.Reason = "conflicting correspondence declarations; no candidate was selected"
	}
	return result
}

func sameTraceEstimateRanges(left []traceestimate.Range, right []TraceCorrespondenceRange) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i].Start != int(right[i].Start) || left[i].Length != int(right[i].Length) {
			return false
		}
	}
	return true
}

func estimateFromSingleDiff(source []byte, run TraceOwnRunResult, current []byte) (TraceOwnEstimateResult, error) {
	result, err := traceestimate.Estimate(source, traceestimate.Range{Start: run.OldStart, Length: run.Length}, current)
	if err != nil {
		return TraceOwnEstimateResult{}, err
	}
	output := TraceOwnEstimateResult{State: string(result.State), Method: result.Method, Reason: result.Reason, Candidates: []TraceOwnEstimateCandidate{}}
	for _, candidate := range result.Candidates {
		output.Candidates = append(output.Candidates, TraceOwnEstimateCandidate{
			CurrentRanges: append([]traceestimate.Range{}, candidate.CurrentRanges...), EvidenceKind: candidate.EvidenceKind,
			Method: candidate.Method, Reason: candidate.Reason, DeclarationIDs: []string{},
		})
	}
	return output, nil
}
