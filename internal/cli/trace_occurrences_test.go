package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/repository"
)

func traceOccurrencesFixture(t *testing.T, source, pattern string, start int) string {
	t.Helper()
	dir := traceCLIFixture(t)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	recipe := fmt.Sprintf(`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"s","source_key":"source-A","file":"source.txt"}],"runs":[{"kind":"external","length":%d,"source":"s","source_start":%d}]}`, len(pattern), start)
	if err := os.WriteFile(filepath.Join(dir, "recipe.json"), []byte(recipe), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", pattern)
	code, _, notice := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "recipe.json")
	if code != 0 || !strings.Contains(notice, "retaining full source file") {
		t.Fatalf("trace set: code=%d notice=%q", code, notice)
	}
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSourceBind(context.Background(), "source-A", "source.txt"); err != nil {
		t.Fatal(err)
	}
	return dir
}

func occurrencePage(t *testing.T, dir string, args ...string) map[string]any {
	t.Helper()
	code, output, stderr := runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(args, "--format", "json")...)...)
	if code != 0 || stderr != "" {
		t.Fatalf("occurrences: code=%d stdout=%q stderr=%q", code, output, stderr)
	}
	return decodeCLIJSON(t, output)
}

func occurrenceEntries(t *testing.T, doc map[string]any) []any {
	t.Helper()
	if doc["schema"] != "sealgraph/trace-occurrences/v1" {
		t.Fatalf("schema=%v", doc["schema"])
	}
	return doc["page"].(map[string]any)["entries"].([]any)
}

func TestTraceOccurrencesOverlappingPagesAndBothViews(t *testing.T) {
	dir := traceOccurrencesFixture(t, "aaaa", "aa", 0)
	args := []string{"--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "both", "--limit", "1"}
	seen := []string{}
	for pageIndex := 0; pageIndex < 6; pageIndex++ {
		doc := occurrencePage(t, dir, args...)
		entries := occurrenceEntries(t, doc)
		if len(entries) != 1 {
			t.Fatalf("page %d entries=%v", pageIndex, entries)
		}
		entry := entries[0].(map[string]any)
		seen = append(seen, fmt.Sprintf("%s:%v", entry["view"], entry["start"]))
		page := doc["page"].(map[string]any)
		if pageIndex < 5 {
			if page["has_more"] != true || page["next_cursor"] == nil {
				t.Fatalf("page %d=%v", pageIndex, page)
			}
			args = append(args[:10:10], "--cursor", page["next_cursor"].(string))
		} else if page["has_more"] != false || page["next_cursor"] != nil {
			t.Fatalf("last page=%v", page)
		}
	}
	want := []string{"snapshot:0", "snapshot:1", "snapshot:2", "current:0", "current:1", "current:2"}
	if strings.Join(seen, ",") != strings.Join(want, ",") {
		t.Fatalf("entries=%v want=%v", seen, want)
	}
}

func TestTraceOccurrencesCursorDetectsCurrentChangeAndInvalidToken(t *testing.T) {
	dir := traceOccurrencesFixture(t, "aaaa", "aa", 0)
	args := []string{"--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "current", "--limit", "1"}
	first := occurrencePage(t, dir, args...)
	cursor := first["page"].(map[string]any)["next_cursor"].(string)
	tampered := cursor[:len(cursor)-1] + "!"
	code, stdout, stderr := runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(args, "--cursor", tampered, "--format", "json")...)...)
	if code == 0 || stdout != "" || !strings.Contains(stderr, "PAGE_TOKEN_INVALID") {
		t.Fatalf("tampered: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	parsed, err := decodeTraceOccurrencesCursor(cursor)
	if err != nil {
		t.Fatal(err)
	}
	parsed.Limit = 0
	malformed, err := encodeTraceOccurrencesCursor(parsed)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(args, "--cursor", malformed, "--format", "json")...)...)
	if code == 0 || stdout != "" || !strings.Contains(stderr, "PAGE_TOKEN_INVALID") {
		t.Fatalf("malformed: %d %q %q", code, stdout, stderr)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("baaaa"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr = runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(args, "--cursor", cursor, "--format", "json")...)...)
	if code == 0 || stdout != "" || !strings.Contains(stderr, "PAGE_CONTEXT_CHANGED") {
		t.Fatalf("changed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("aaaa"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "trace", "clear", "root")
	code, stdout, stderr = runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(args, "--cursor", cursor, "--format", "json")...)...)
	if code == 0 || stdout != "" || !strings.Contains(stderr, "PAGE_CONTEXT_CHANGED") {
		t.Fatalf("Candidate changed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestTraceOccurrencesSnapshotSurvivesFileRemovalAndMissingCurrentIsIncomplete(t *testing.T) {
	dir := traceOccurrencesFixture(t, "aaaa", "aa", 0)
	if err := os.Remove(filepath.Join(dir, "source.txt")); err != nil {
		t.Fatal(err)
	}
	base := []string{"--ref", "root", "--baseline", "candidate", "--run-index", "0"}
	snapshot := occurrencePage(t, dir, append(base, "--view", "snapshot")...)
	if len(occurrenceEntries(t, snapshot)) != 3 || snapshot["observation"].(map[string]any)["current_blob_id"] != nil {
		t.Fatalf("snapshot=%v", snapshot)
	}
	current := occurrencePage(t, dir, append(base, "--view", "current")...)
	page := current["page"].(map[string]any)
	if page["state"] != "INCOMPLETE" || page["reason"] != "SOURCE_READ_FAILED" || page["has_more"] != nil || len(occurrenceEntries(t, current)) != 0 {
		t.Fatalf("current=%v", current)
	}
	both := occurrencePage(t, dir, append(base, "--view", "both", "--limit", "1")...)
	if both["page"].(map[string]any)["state"] != "INCOMPLETE" || len(occurrenceEntries(t, both)) != 1 || occurrenceEntries(t, both)[0].(map[string]any)["view"] != "snapshot" {
		t.Fatalf("both with missing current=%v", both)
	}
}

func TestTraceOccurrencesExplicitBaselineAndInvalidRun(t *testing.T) {
	dir := traceOccurrencesFixture(t, "aaaa", "aa", 0)
	code, stdout, stderr := runCLI(t, dir, nil, "trace", "occurrences", "--ref", "root", "--run-index", "0", "--view", "snapshot", "--format", "json")
	if code == 0 || stdout != "" || !strings.Contains(stderr, "baseline") {
		t.Fatalf("missing baseline: %d %q %q", code, stdout, stderr)
	}
	mustRunCLI(t, dir, "seal", "root")
	head := occurrencePage(t, dir, "--ref", "root", "--baseline", "head", "--run-index", "0", "--view", "snapshot")
	if head["selection"].(map[string]any)["baseline"].(map[string]any)["kind"] != "seal" {
		t.Fatalf("HEAD=%v", head)
	}
	id := head["selection"].(map[string]any)["baseline"].(map[string]any)["seal_id"].(string)
	exact := occurrencePage(t, dir, "--seal", "@"+id, "--run-index", "0", "--view", "snapshot")
	if exact["selection"].(map[string]any)["baseline"].(map[string]any)["seal_id"] != id {
		t.Fatalf("exact Seal=%v", exact)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "trace", "occurrences", "--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "snapshot", "--format", "json")
	if code == 0 || stdout != "" {
		t.Fatalf("absent candidate fallback: %d %q %q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "trace", "occurrences", "--ref", "root", "--baseline", "head", "--run-index", "1", "--view", "snapshot", "--format", "json")
	if code == 0 || stdout != "" || !strings.Contains(stderr, "External run") {
		t.Fatalf("invalid run: %d %q %q", code, stdout, stderr)
	}
}

func TestTraceOccurrencesDefaultLimitAndCurrentBindingChange(t *testing.T) {
	dir := traceOccurrencesFixture(t, strings.Repeat("a", 102), "a", 0)
	base := []string{"--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "snapshot"}
	first := occurrencePage(t, dir, base...)
	if len(occurrenceEntries(t, first)) != 100 || first["page"].(map[string]any)["has_more"] != true {
		t.Fatalf("default first page=%v", first["page"])
	}
	cursor := first["page"].(map[string]any)["next_cursor"].(string)
	second := occurrencePage(t, dir, append(base, "--cursor", cursor)...)
	if len(occurrenceEntries(t, second)) != 2 || second["page"].(map[string]any)["has_more"] != false {
		t.Fatalf("default last page=%v", second["page"])
	}
	currentBase := []string{"--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "current", "--limit", "1"}
	current := occurrencePage(t, dir, currentBase...)
	currentCursor := current["page"].(map[string]any)["next_cursor"].(string)
	if err := os.WriteFile(filepath.Join(dir, "other.txt"), []byte(strings.Repeat("a", 102)), 0o600); err != nil {
		t.Fatal(err)
	}
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.TraceSourceRebind(context.Background(), "source-A", "source.txt", "other.txt"); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, dir, nil, append([]string{"trace", "occurrences"}, append(currentBase, "--cursor", currentCursor, "--format", "json")...)...)
	if code == 0 || stdout != "" || !strings.Contains(stderr, "PAGE_CONTEXT_CHANGED") {
		t.Fatalf("rebound: %d %q %q", code, stdout, stderr)
	}
}

func TestTraceOccurrencesRevalidationClassifiesSelectedBaselineChange(t *testing.T) {
	dir := traceOccurrencesFixture(t, "aaaa", "aa", 0)
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	input, err := repo.LoadTraceOccurrenceInput(context.Background(), repository.TraceOwnBaseline{Kind: repository.TraceOwnCandidate, REF: "root"}, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "trace", "clear", "root")
	if err := repo.RevalidateTraceOccurrenceInput(context.Background(), input); !errors.Is(err, repository.ErrTraceOccurrenceBaselineChanged) {
		t.Fatalf("baseline revalidation error=%v", err)
	}
}
