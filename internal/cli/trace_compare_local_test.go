package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/repository"
	"github.com/mako10k/sealgraph/internal/tracecompare"
)

func TestBuildTraceCompareLocalPreservesPresenceAndCompleteness(t *testing.T) {
	content := domain.ComputeNativeBlobID([]byte("abc"))
	origin := domain.ComputeNativeBlobID([]byte("origin"))
	snapshot := domain.ComputeNativeBlobID([]byte("snapshot"))
	current := domain.ComputeNativeBlobID([]byte("current"))
	start := 9
	result := repository.TraceCompareOwnResult{
		REF: "root", Baseline: repository.TraceOwnBaseline{Kind: repository.TraceOwnCandidate, REF: "root"}, Content: content, Origin: &origin,
		Runs: []repository.TraceOwnRunResult{
			{RunIndex: 0, SourceSnapshotID: snapshot, SourceKey: "source-A", OldStart: 2, Length: 3, CurrentBlobID: &current, Presence: tracecompare.Present, PresenceReason: "EXACT_MATCH", SelectedMatchStart: &start, Examined: true},
			{RunIndex: 2, SourceSnapshotID: snapshot, SourceKey: "source-B", OldStart: 6, Length: 1, Presence: tracecompare.Undetermined, PresenceReason: "SOURCE_READ_FAILED", Examined: true},
			{RunIndex: 3, SourceSnapshotID: snapshot, SourceKey: "source-C", OldStart: 7, Length: 1, Presence: tracecompare.Undetermined, PresenceReason: "SEARCH_INTERRUPTED", Examined: false},
		},
	}
	local := buildTraceCompareLocal(result)
	if local.TracePresence != "EXTERNAL_RANGES" || local.ExternalRunCount != 3 || local.ComparedRunCount != 1 || local.UnexaminedRunCount != 1 || !local.HasUnresolved || local.HasDifference || local.Complete {
		t.Fatalf("local summary=%+v", local)
	}
	if local.Ranges[0].PositionChanged == nil || !*local.Ranges[0].PositionChanged || local.Ranges[0].SelectedMatchStart == nil || local.Ranges[1].CurrentBlobID != nil || local.Ranges[2].Examined != "NOT_EXAMINED" {
		t.Fatalf("range records=%+v", local.Ranges)
	}
	data, err := json.Marshal(local)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte(`"estimate":{"state":"NOT_REQUESTED","method":null,"candidates":[],"reason":null}`)) || !bytes.Contains(data, []byte(`"run_index":2`)) {
		t.Fatalf("local JSON=%s", data)
	}
	var human bytes.Buffer
	printTraceCompareLocalHuman(&human, local)
	if !strings.Contains(human.String(), "selected match; historical origin is not established") || !strings.Contains(human.String(), "SOURCE_READ_FAILED") {
		t.Fatalf("local human=%q", human.String())
	}
}

func TestBuildTraceCompareLocalKeepsNoTraceSeparate(t *testing.T) {
	content := domain.ComputeNativeBlobID([]byte("abc"))
	local := buildTraceCompareLocal(repository.TraceCompareOwnResult{Baseline: repository.TraceOwnBaseline{Kind: repository.TraceOwnSeal}, Content: content})
	if local.TracePresence != "NO_TRACE" || !local.Complete || local.ExternalRunCount != 0 || local.Ranges == nil {
		t.Fatalf("no trace local=%+v", local)
	}
}
