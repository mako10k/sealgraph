package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/repository"
)

func traceComparePreparedFixture(t *testing.T) (string, *repository.Repository) {
	t.Helper()
	dir := traceCLIFixture(t)
	writeTraceFixture(t, dir)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	mustTraceSetCLI(t, dir)
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for key, path := range map[string]string{"key-A": "a.txt", "key-B": "b.txt"} {
		if _, err := repo.TraceSourceBind(ctx, key, path); err != nil {
			t.Fatal(err)
		}
	}
	return dir, repo
}

func TestPrepareTraceCompareNoEstimateCandidate(t *testing.T) {
	_, repo := traceComparePreparedFixture(t)
	ctx := context.Background()
	ref := singleString{value: "root", set: true}
	candidate, err := prepareTraceCompareNoEstimate(ctx, repo, ref, singleString{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if candidate.Graph != nil || candidate.GraphReason == nil || *candidate.GraphReason != "NO_HEAD" || candidate.CandidateOwn == nil || candidate.CandidateOwn.Local.ExternalRunCount != 2 || candidate.Limits.UsedGraphVisits != 0 || len(candidate.Observation.Sources) != 2 {
		t.Fatalf("candidate compare=%+v", candidate)
	}
	if candidate.Observation.CandidateDigest == nil || len(*candidate.Observation.CandidateDigest) != 64 || len(candidate.Observation.BindingDigest) != 64 {
		t.Fatalf("candidate identities=%+v", candidate.Observation)
	}
}

func TestPrepareTraceCompareNoEstimateHead(t *testing.T) {
	dir, repo := traceComparePreparedFixture(t)
	ctx := context.Background()
	ref := singleString{value: "root", set: true}
	mustRunCLI(t, dir, "seal", "root")
	head, err := prepareTraceCompareNoEstimate(ctx, repo, ref, singleString{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if head.Graph == nil || head.CandidateOwn != nil || head.GraphReason != nil || !head.Graph.Scope.Complete || head.Graph.Own.Baseline.REF == nil || *head.Graph.Own.Baseline.REF != "root" || head.Graph.Own.ComparedRunCount != 2 || head.Graph.Upstream.SealCount != 0 || head.Graph.Downstream.SealCount != 0 {
		t.Fatalf("HEAD compare=%+v", head)
	}
	encoded, err := json.Marshal(head)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"schema":"sealgraph/trace-compare/v2"`) || !strings.Contains(string(encoded), `"declaration_digest":null`) || !strings.Contains(string(encoded), `"estimate":{"state":"NOT_REQUESTED"`) {
		t.Fatalf("HEAD JSON=%s", encoded)
	}
	var human bytes.Buffer
	printTraceCompareV2Human(&human, head)
	if !strings.Contains(human.String(), "Upstream:") || !strings.Contains(human.String(), "Downstream:") || !strings.Contains(human.String(), "historical origin position") {
		t.Fatalf("HEAD human=%q", human.String())
	}
}

func TestPrepareTraceCompareBudgetKeepsLocalFacts(t *testing.T) {
	dir, repo := traceComparePreparedFixture(t)
	mustRunCLI(t, dir, "seal", "root")
	doc, err := prepareTraceCompareNoEstimate(context.Background(), repo, singleString{value: "root", set: true}, singleString{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Graph == nil || doc.Graph.Scope.Complete || doc.Graph.Scope.StopReason == nil || *doc.Graph.Scope.StopReason != "GRAPH_BUDGET" || doc.Graph.Upstream.Complete || doc.Graph.Downstream.Complete {
		t.Fatalf("budget scope=%+v", doc.Graph)
	}
	if !doc.Graph.Own.Complete || doc.Graph.Own.ComparedRunCount != 2 || doc.Graph.Own.HasDifference || doc.Limits.UsedGraphVisits != 1 || len(doc.Graph.Scope.ObservedSeals) != 1 {
		t.Fatalf("budget local=%+v", doc.Graph)
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"extra_seals":[],"observed_seals":[`) || !strings.Contains(string(encoded), `"stop_reason":"GRAPH_BUDGET"`) || !strings.Contains(string(encoded), `"members":[]`) {
		t.Fatalf("budget JSON=%s", encoded)
	}
}

func TestPrepareTraceCompareDownstreamUsesCauseOnly(t *testing.T) {
	dir, repo := traceComparePreparedFixture(t)
	ctx := context.Background()
	mustRunCLI(t, dir, "seal", "root")
	if _, err := repo.Add(ctx, repository.AddOptions{REF: "child", Content: []byte("child"), RootSet: true, Cause: &repository.CauseInput{Target: "root"}}); err != nil {
		t.Fatal(err)
	}
	child, err := repo.Seal(ctx, "child")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := prepareTraceCompareNoEstimate(ctx, repo, singleString{value: "root", set: true}, singleString{}, 100)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Graph == nil || doc.Graph.Downstream.SealCount != 1 || len(doc.Graph.Downstream.Members) != 1 || doc.Graph.Downstream.Members[0].SealID != child.ID.String() || !doc.Graph.Downstream.Members[0].Direct || doc.Graph.Downstream.Members[0].Indirect {
		t.Fatalf("downstream=%+v", doc.Graph)
	}
	encoded, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"indirect_path":null`) || !strings.Contains(string(encoded), `"direct_path":[`) {
		t.Fatalf("path JSON=%s", encoded)
	}
}
