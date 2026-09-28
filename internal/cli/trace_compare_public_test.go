package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/repository"
)

func TestTraceComparePublicEstimateAndCorrespondence(t *testing.T) {
	dir, _ := traceComparePreparedFixture(t)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("abcXQZtail"), 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{"trace", "compare", "--ref", "root", "--max-graph-visits", "100", "--format", "json"}
	code, output, stderr := runCLI(t, dir, nil, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("compare: code=%d stdout=%q stderr=%q", code, output, stderr)
	}
	doc := decodeCLIJSON(t, output)
	if doc["schema"] != "sealgraph/trace-compare/v2" {
		t.Fatalf("compare schema=%s", output)
	}
	observation := doc["observation"].(map[string]any)
	if observation["declaration_digest"] != nil {
		t.Fatalf("default comparison read declarations: %s", output)
	}
	runs := doc["candidate_own"].(map[string]any)["local"].(map[string]any)["ranges"].([]any)
	first := runs[0].(map[string]any)
	if first["presence"] != "ABSENT_EXACT" || first["estimate"].(map[string]any)["state"] != "NOT_REQUESTED" {
		t.Fatalf("default first run=%v", first)
	}
	code, estimated, stderr := runCLI(t, dir, nil, append(args, "--estimate")...)
	if code != 0 || stderr != "" {
		t.Fatalf("estimate: code=%d stdout=%q stderr=%q", code, estimated, stderr)
	}
	estimateDoc := decodeCLIJSON(t, estimated)
	estimatedRun := estimateDoc["candidate_own"].(map[string]any)["local"].(map[string]any)["ranges"].([]any)[0].(map[string]any)
	if estimatedRun["presence"] != "ABSENT_EXACT" || estimatedRun["estimate"].(map[string]any)["state"] != "CANDIDATES" || estimateDoc["observation"].(map[string]any)["declaration_digest"] == nil {
		t.Fatalf("estimated comparison=%s", estimated)
	}
	assertTraceCorrespondencePublic(t, dir, args, estimatedRun)
}

func assertTraceCorrespondencePublic(t *testing.T, dir string, args []string, estimatedRun map[string]any) {
	t.Helper()
	declaration := fmt.Sprintf(`{"schema":"sealgraph/trace-correspondence/v1","source_snapshot":%q,"source_start":3,"length":3,"current_blob":%q,"current_ranges":[{"start":3,"length":3}],"deleted":false,"reason":"manual range","declared_at":"2026-09-18T08:00:00Z"}`, estimatedRun["source_snapshot_id"], estimatedRun["current_blob_id"])
	if err := os.WriteFile(filepath.Join(dir, "declaration.json"), []byte(declaration), 0o600); err != nil {
		t.Fatal(err)
	}
	code, put, stderr := runCLI(t, dir, nil, "trace", "correspondence", "put", "--file", "declaration.json", "--format", "json")
	if code != 0 || stderr != "" {
		t.Fatalf("put: code=%d stdout=%q stderr=%q", code, put, stderr)
	}
	mutation := decodeCLIJSON(t, put)
	id := mutation["id"].(string)
	if mutation["schema"] != "sealgraph/trace-correspondence-mutation/v1" || mutation["changed"] != true || len(id) != 64 {
		t.Fatalf("put receipt=%s", put)
	}
	assertTraceCorrespondenceShowList(t, dir, id)
	code, declared, stderr := runCLI(t, dir, nil, append(args, "--estimate")...)
	if code != 0 || stderr != "" {
		t.Fatalf("declared comparison: code=%d stdout=%q stderr=%q", code, declared, stderr)
	}
	declaredRun := decodeCLIJSON(t, declared)["candidate_own"].(map[string]any)["local"].(map[string]any)["ranges"].([]any)[0].(map[string]any)
	candidate := declaredRun["estimate"].(map[string]any)["candidates"].([]any)[0].(map[string]any)
	if declaredRun["presence"] != "ABSENT_EXACT" || candidate["evidence_kind"] != "declared_correspondence" || candidate["declaration_ids"].([]any)[0] != id || !strings.Contains(candidate["reason"].(string), "bytes differ from old range") {
		t.Fatalf("declared comparison=%s", declared)
	}
	code, removed, stderr := runCLI(t, dir, nil, "trace", "correspondence", "remove", id, "--format", "json")
	if code != 0 || stderr != "" || decodeCLIJSON(t, removed)["changed"] != true {
		t.Fatalf("remove: code=%d stdout=%q stderr=%q", code, removed, stderr)
	}
	code, after, stderr := runCLI(t, dir, nil, "trace", "correspondence", "list", "--format", "json")
	if code != 0 || stderr != "" || len(decodeCLIJSON(t, after)["records"].([]any)) != 0 {
		t.Fatalf("after remove: code=%d stdout=%q stderr=%q", code, after, stderr)
	}
}

func assertTraceCorrespondenceShowList(t *testing.T, dir, id string) {
	t.Helper()
	for _, operation := range [][]string{{"show", id}, {"list"}} {
		code, listing, stderr := runCLI(t, dir, nil, append([]string{"trace", "correspondence"}, append(operation, "--format", "json")...)...)
		if code != 0 || stderr != "" {
			t.Fatalf("%s: code=%d stdout=%q stderr=%q", operation[0], code, listing, stderr)
		}
		result := decodeCLIJSON(t, listing)
		if result["schema"] != "sealgraph/trace-correspondence-list/v1" || len(result["records"].([]any)) != 1 {
			t.Fatalf("%s result=%s", operation[0], listing)
		}
	}
}

func TestTraceComparePublicRejectsLegacyBudgetWithoutSuccessJSON(t *testing.T) {
	dir, _ := traceComparePreparedFixture(t)
	code, stdout, stderr := runCLI(t, dir, nil, "trace", "compare", "--ref", "root", "--max-graph-visits", "1", "--max-alignment-cells", "5", "--format", "json")
	if code == 0 || stdout != "" || !strings.Contains(stderr, "max-alignment-cells") {
		t.Fatalf("legacy budget: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	var value any
	if err := json.Unmarshal([]byte(stdout), &value); err == nil {
		t.Fatalf("unexpected success JSON: %s", stdout)
	}
}

func TestTraceComparePublicDirectionGraphAndBudget(t *testing.T) {
	dir, repo := traceComparePreparedFixture(t)
	mustRunCLI(t, dir, "seal", "root")
	if _, err := repo.Add(context.Background(), repository.AddOptions{REF: "child", Content: []byte("child"), RootSet: true, Cause: &repository.CauseInput{Target: "root"}}); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "seal", "child")
	code, output, stderr := runCLI(t, dir, nil, "trace", "compare", "--ref", "root", "--max-graph-visits", "100", "--format", "json")
	if code != 0 || stderr != "" {
		t.Fatalf("direction compare: code=%d stdout=%q stderr=%q", code, output, stderr)
	}
	graph := decodeCLIJSON(t, output)["graph"].(map[string]any)
	if graph["own"].(map[string]any)["compared_run_count"] != float64(2) || graph["downstream"].(map[string]any)["seal_count"] != float64(1) || graph["upstream"].(map[string]any)["seal_count"] != float64(0) || graph["scope"].(map[string]any)["complete"] != true {
		t.Fatalf("direction graph=%s", output)
	}
	code, limited, stderr := runCLI(t, dir, nil, "trace", "compare", "--ref", "root", "--max-graph-visits", "1", "--format", "json")
	if code != 0 || stderr != "" {
		t.Fatalf("limited compare: code=%d stdout=%q stderr=%q", code, limited, stderr)
	}
	limitedGraph := decodeCLIJSON(t, limited)["graph"].(map[string]any)
	if limitedGraph["scope"].(map[string]any)["complete"] != false || limitedGraph["own"].(map[string]any)["compared_run_count"] != float64(2) {
		t.Fatalf("budget erased own facts=%s", limited)
	}
}
