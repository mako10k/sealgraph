package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/repository"
)

const traceFormat7Config = "repository_format = 7\nobject_format = sha256\nref_format = manifest-v1\n"

func traceCLIFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, ".sealgraph", "config"), []byte(traceFormat7Config), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func writeTraceFixture(t *testing.T, dir string) (string, string) {
	t.Helper()
	a, b := "abcXYZtail", "headUVtail"
	for name, content := range map[string]string{"a.txt": a, "b.txt": b} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	recipe := `{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"key-A","file":"a.txt"},{"name":"b","source_key":"key-B","file":"b.txt"}],"runs":[{"kind":"external","length":3,"source":"a","source_start":3},{"kind":"untraced","length":1},{"kind":"external","length":2,"source":"b","source_start":4}]}`
	if err := os.WriteFile(filepath.Join(dir, "recipe.json"), []byte(recipe), 0o644); err != nil {
		t.Fatal(err)
	}
	return a, b
}

func TestTraceCLIAuthorAndRecoverFullSources(t *testing.T) {
	dir := traceCLIFixture(t)
	a, b := writeTraceFixture(t, dir)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	code, output, notice := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "recipe.json", "--format", "json")
	if code != 0 || !strings.Contains(notice, `"a.txt" (10 bytes)`) || !strings.Contains(notice, `"b.txt" (10 bytes)`) {
		t.Fatalf("trace set: code=%d output=%q notice=%q", code, output, notice)
	}
	receipt := decodeCLIJSON(t, output)
	if receipt["schema"] != "sealgraph/trace-mutation/v1" || receipt["operation"] != "set" || receipt["changed"] != true || len(receipt["stored_sources"].([]any)) != 2 {
		t.Fatalf("trace set receipt=%s", output)
	}
	assertTraceShow(t, dir, true, false)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, "b.txt")); err != nil {
		t.Fatal(err)
	}
	sealOutput := mustRunCLI(t, dir, "seal", "root")
	fields := strings.Fields(sealOutput)
	if len(fields) < 3 {
		t.Fatalf("seal output=%q", sealOutput)
	}
	assertTraceShow(t, dir, false, true)
	assertRestoredTraceSources(t, dir, fields[2], a, b)
}

func assertTraceShow(t *testing.T, dir string, candidate, seal bool) {
	t.Helper()
	output := mustRunCLI(t, dir, "trace", "show", "--ref", "root", "--format", "json")
	view := decodeCLIJSON(t, output)
	baselines := view["baselines"].([]any)
	if view["schema"] != "sealgraph/trace-show/v1" || len(baselines) != 1 {
		t.Fatalf("trace show=%s", output)
	}
	item := baselines[0].(map[string]any)
	base := item["baseline"].(map[string]any)
	if candidate && base["kind"] != "candidate" || seal && base["kind"] != "seal" || item["origin_map"] == nil || len(item["source_snapshots"].([]any)) != 2 {
		t.Fatalf("trace show baseline=%s", output)
	}
}

func assertRestoredTraceSources(t *testing.T, dir, sealText, a, b string) {
	t.Helper()
	id, err := domain.ParseObjectID(sealText)
	if err != nil {
		t.Fatal(err)
	}
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := repo.LoadSeal(context.Background(), id)
	if err != nil || resolved.Provenance.Origin == nil {
		t.Fatalf("LoadSeal=%+v err=%v", resolved, err)
	}
	origin, err := repo.LoadOrigin(context.Background(), *resolved.Provenance.Origin, resolved.Material.Content)
	if err != nil {
		t.Fatal(err)
	}
	restored := make(map[string]string)
	for _, source := range origin.Sources {
		restored[source.Snapshot.SourceKey] = string(source.Content)
	}
	if restored["key-A"] != a || restored["key-B"] != b || len(restored) != 2 {
		t.Fatalf("restored full source bytes=%v", restored)
	}
}

func TestTraceCLIClearAndCandidateHEADSeparation(t *testing.T) {
	dir := traceCLIFixture(t)
	writeTraceFixture(t, dir)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	mustTraceSetCLI(t, dir)
	mustRunCLI(t, dir, "seal", "root")
	mustRunCLI(t, dir, "add", "root", "--content", "XYZ-UV")
	clear := decodeCLIJSON(t, mustRunCLI(t, dir, "trace", "clear", "root", "--format", "json"))
	if clear["changed"] != true {
		t.Fatalf("trace clear=%v", clear)
	}
	second := decodeCLIJSON(t, mustRunCLI(t, dir, "trace", "clear", "root", "--format", "json"))
	if second["changed"] != false {
		t.Fatalf("second clear=%v", second)
	}
	view := decodeCLIJSON(t, mustRunCLI(t, dir, "trace", "show", "--ref", "root", "--format", "json"))
	baselines := view["baselines"].([]any)
	if len(baselines) != 2 || baselines[0].(map[string]any)["origin_map"] != nil || baselines[1].(map[string]any)["origin_map"] == nil {
		t.Fatalf("candidate/HEAD trace show=%v", view)
	}
}

func TestTraceCLIIntermediateSealAndAtomicContentUpdate(t *testing.T) {
	dir := traceCLIFixture(t)
	writeTraceFixture(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "root-content.txt"), []byte("XYZ"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "root-recipe.json"), []byte(`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"a","source_key":"key-A","file":"a.txt"}],"runs":[{"kind":"external","length":3,"source":"a","source_start":3}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "draft")
	code, output, notice := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "root-recipe.json", "--content-file", "root-content.txt", "--format", "json")
	if code != 0 || !strings.Contains(notice, "a.txt") || decodeCLIJSON(t, output)["changed"] != true {
		t.Fatalf("atomic set: code=%d output=%q notice=%q", code, output, notice)
	}
	mustRunCLI(t, dir, "seal", "root")
	mustRunCLI(t, dir, "add", "middle", "--non-root", "--target", "root", "--no-previous", "--content", "UV")
	if err := os.WriteFile(filepath.Join(dir, "middle-recipe.json"), []byte(`{"schema":"sealgraph/trace-recipe/v1","sources":[{"name":"b","source_key":"key-B","file":"b.txt"}],"runs":[{"kind":"external","length":2,"source":"b","source_start":4}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, output, notice = runCLI(t, dir, nil, "trace", "set", "middle", "--recipe", "middle-recipe.json", "--format", "json")
	if code != 0 || !strings.Contains(notice, "b.txt") {
		t.Fatalf("middle set: code=%d output=%q notice=%q", code, output, notice)
	}
	mustRunCLI(t, dir, "seal", "middle")
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	rootID, err := repo.CurrentREFHead(context.Background(), "root")
	if err != nil || rootID == nil {
		t.Fatalf("root head=%v err=%v", rootID, err)
	}
	middleID, err := repo.CurrentREFHead(context.Background(), "middle")
	if err != nil || middleID == nil {
		t.Fatalf("middle head=%v err=%v", middleID, err)
	}
	middle, err := repo.LoadSeal(context.Background(), *middleID)
	if err != nil || middle.Provenance.Origin == nil || len(middle.Provenance.CauseLinks) != 1 || !middle.Provenance.CauseLinks[0].TargetSeal.Equal(*rootID) {
		t.Fatalf("intermediate Seal=%+v err=%v", middle, err)
	}
}

func mustTraceSetCLI(t *testing.T, dir string) {
	t.Helper()
	code, output, notice := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "recipe.json", "--format", "json")
	if code != 0 || !strings.Contains(notice, "retaining full source file") {
		t.Fatalf("trace set code=%d output=%q notice=%q", code, output, notice)
	}
}

func TestTraceCLIInvalidRecipePreservesCandidate(t *testing.T) {
	dir := traceCLIFixture(t)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "abc")
	before := mustRunCLI(t, dir, "candidate", "show", "root", "--format", "json")
	if err := os.WriteFile(filepath.Join(dir, "bad.json"), []byte(`{"schema":"sealgraph/trace-recipe/v1","sources":null,"runs":[]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, _, _ := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "bad.json", "--format", "json")
	if code == 0 {
		t.Fatal("invalid recipe accepted")
	}
	after := mustRunCLI(t, dir, "candidate", "show", "root", "--format", "json")
	if !bytes.Equal([]byte(before), []byte(after)) {
		t.Fatalf("invalid recipe changed Candidate: before=%s after=%s", before, after)
	}
}

func TestTraceShowJSONIsSingleCompleteDocument(t *testing.T) {
	dir := traceCLIFixture(t)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "root")
	output := mustRunCLI(t, dir, "trace", "show", "--ref", "root", "--format", "json")
	decoder := json.NewDecoder(strings.NewReader(output))
	var document any
	if err := decoder.Decode(&document); err != nil || !strings.HasSuffix(output, "\n") {
		t.Fatalf("trace show JSON=%q err=%v", output, err)
	}
}
