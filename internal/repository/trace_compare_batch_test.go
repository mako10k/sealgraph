package repository

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestTraceCompareOwnBatchSharesSourceObservation(t *testing.T) {
	repo, root := traceOwnFixture(t, "XYZ", true, "source.txt")
	ctx := context.Background()
	sealed, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	results, sources, err := repo.TraceCompareOwnBatch(ctx, []TraceOwnBaseline{
		{Kind: TraceOwnSeal, SealID: &sealed.ID},
		{Kind: TraceOwnSeal, SealID: &sealed.ID},
	})
	if err != nil || len(results) != 2 || len(sources) != 1 {
		t.Fatalf("results=%+v sources=%+v err=%v", results, sources, err)
	}
	if sources[0].SourceKey != "source-A" || sources[0].CurrentBlobID == nil || sources[0].ByteLength == nil || *sources[0].ByteLength != 3 {
		t.Fatalf("source=%+v", sources[0])
	}
	if results[0].Baseline.Kind != TraceOwnSeal || results[1].Baseline.Kind != TraceOwnSeal || len(results[0].Runs) != 1 || len(results[1].Runs) != 1 {
		t.Fatalf("results=%+v", results)
	}
	if results[0].Runs[0].CurrentBlobID == nil || !results[0].Runs[0].CurrentBlobID.Equal(*sources[0].CurrentBlobID) {
		t.Fatalf("run=%+v source=%+v", results[0].Runs[0], sources[0])
	}
	_ = root
}

func TestTraceCompareOwnBatchPreservesCandidateAndHeadKinds(t *testing.T) {
	repo, _ := traceOwnFixture(t, "XYZ", true, "source.txt")
	ctx := context.Background()
	sealed, err := repo.Seal(ctx, "root")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("uXYZ"), Root: true, RootSet: true, ClearCauseLinks: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSet(ctx, TraceSetOptions{REF: "root", Runs: []TraceRunInput{{Kind: "untraced", Length: 1}, {Kind: "external", Length: 3, SourceName: "source", SourceStart: 3}}, Sources: []TraceSourceInput{{Name: "source", SourceKey: "source-A", Content: []byte("abcXYZtail")}}}); err != nil {
		t.Fatal(err)
	}
	results, _, err := repo.TraceCompareOwnBatch(ctx, []TraceOwnBaseline{
		{Kind: TraceOwnCandidate, REF: "root"},
		{Kind: TraceOwnHead, REF: "root"},
		{Kind: TraceOwnSeal, SealID: &sealed.ID},
	})
	if err != nil || len(results) != 3 {
		t.Fatalf("results=%+v err=%v", results, err)
	}
	if results[0].Baseline.Kind != TraceOwnCandidate || results[1].Baseline.Kind != TraceOwnHead || results[2].Baseline.Kind != TraceOwnSeal {
		t.Fatalf("baseline kinds=%+v", results)
	}
}

func TestTraceCompareOwnBatchMissingBindingIsUndetermined(t *testing.T) {
	repo, root := traceOwnFixture(t, "unused", false, "source.txt")
	if err := os.WriteFile(filepath.Join(root, "other.txt"), []byte("unused"), 0o600); err != nil {
		t.Fatal(err)
	}
	results, sources, err := repo.TraceCompareOwnBatch(context.Background(), []TraceOwnBaseline{{Kind: TraceOwnCandidate, REF: "root"}})
	if err != nil || len(results) != 1 || len(sources) != 1 {
		t.Fatalf("results=%+v sources=%+v err=%v", results, sources, err)
	}
	if sources[0].Bound || sources[0].ReadFailed || sources[0].Error == nil || results[0].Runs[0].Presence != "UNDETERMINED" {
		t.Fatalf("result=%+v source=%+v", results[0], sources[0])
	}
}
