package repository

import (
	"context"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func TestTraceCompareOwnBatchEstimatedKeepsAbsentPresence(t *testing.T) {
	repo, _ := traceOwnFixture(t, "abcXQZtail", true, "source.txt")
	results, sources, err := repo.TraceCompareOwnBatchEstimated(context.Background(), []TraceOwnBaseline{{Kind: TraceOwnCandidate, REF: "root"}}, nil)
	if err != nil || len(results) != 1 || len(sources) != 1 || len(results[0].Runs) != 1 {
		t.Fatalf("results=%+v sources=%+v err=%v", results, sources, err)
	}
	run := results[0].Runs[0]
	if run.Presence != "ABSENT_EXACT" || run.PresenceReason != "NO_EXACT_MATCH" || run.Estimate == nil || run.Estimate.State != "CANDIDATES" || run.Estimate.Method == "" || len(run.Estimate.Candidates) != 1 || run.Estimate.Candidates[0].EvidenceKind != "single_diff" {
		t.Fatalf("estimated run=%+v", run)
	}
}

func TestTraceCompareOwnBatchEstimatedUsesDeclaredCorrespondence(t *testing.T) {
	repo, _ := traceOwnFixture(t, "abcXQZtail", true, "source.txt")
	ctx := context.Background()
	baseline := []TraceOwnBaseline{{Kind: TraceOwnCandidate, REF: "root"}}
	before, _, err := repo.TraceCompareOwnBatch(ctx, baseline)
	if err != nil {
		t.Fatal(err)
	}
	run := before[0].Runs[0]
	first := putEstimateDeclaration(t, repo, run, []TraceCorrespondenceRange{{Start: 4, Length: 1}}, "declared replacement")
	second := putEstimateDeclaration(t, repo, run, []TraceCorrespondenceRange{{Start: 3, Length: 2}}, "different declaration")
	declarations, err := repo.TraceCorrespondenceList()
	if err != nil {
		t.Fatal(err)
	}
	results, _, err := repo.TraceCompareOwnBatchEstimated(ctx, baseline, declarations)
	if err != nil {
		t.Fatal(err)
	}
	estimate := results[0].Runs[0].Estimate
	if results[0].Runs[0].Presence != "ABSENT_EXACT" || estimate == nil || estimate.State != "CANDIDATES" || estimate.Method != "" || len(estimate.Candidates) != 2 || estimate.Reason == "" {
		t.Fatalf("declaration estimate=%+v", estimate)
	}
	ids := map[string]bool{}
	for _, candidate := range estimate.Candidates {
		if candidate.EvidenceKind != "declared_correspondence" || len(candidate.DeclarationIDs) != 1 {
			t.Fatalf("candidate=%+v", candidate)
		}
		ids[candidate.DeclarationIDs[0]] = true
	}
	if !ids[first] || !ids[second] {
		t.Fatalf("candidate IDs=%v, want %s and %s", ids, first, second)
	}
}

func putEstimateDeclaration(t *testing.T, repo *Repository, run TraceOwnRunResult, ranges []TraceCorrespondenceRange, reason string) string {
	t.Helper()
	if run.CurrentBlobID == nil {
		t.Fatal("run has no current BlobID")
	}
	result, err := repo.TraceCorrespondencePut(context.Background(), TraceCorrespondencePutOptions{
		SourceSnapshot: run.SourceSnapshotID, SourceStart: uint64(run.OldStart), Length: uint64(run.Length),
		CurrentBlob: domain.ObjectID{Hex: run.CurrentBlobID.String()}, CurrentRanges: ranges,
		Reason: reason, DeclaredAt: "2026-09-18T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.ID
}
