package cli

// R: verify additive observation navigation, baselines and output boundaries.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/repository"
)

func observeCLI(t *testing.T, dir string, args ...string) repository.ObservationResult {
	t.Helper()
	all := append([]string{"observe"}, args...)
	all = append(all, "--format", "json")
	code, out, stderr := runCLI(t, dir, nil, all...)
	if code != 0 || stderr != "" {
		t.Fatalf("observe %v: %d %s", args, code, stderr)
	}
	var result repository.ObservationResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	return result
}
func TestObserveCLISeparatesBaselinesAndPreservesOldOutputs(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	file := filepath.Join(dir, "s.txt")
	if err := os.WriteFile(file, []byte("one"), 0600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "add", "s", "--root", "--clear-cause-links", "--content-file", "s.txt", "--bind-source")
	mustRunCLI(t, dir, "seal", "s")
	mustRunCLI(t, dir, "add", "s", "--content", "one")
	commands := [][]string{{"status", "--format", "json"}, {"stale", "--format", "json"}, {"stale", "--refs-only"}, {"stale", "--frontier", "--format", "json"}, {"graph", "--format", "json"}, {"fsck", "--format", "json"}, {"candidate", "compare", "s", "--format", "json"}}
	before := make([]string, len(commands))
	for i, args := range commands {
		before[i] = mustRunCLI(t, dir, args...)
	}
	doc := observeCLI(t, dir, "s")
	ref := doc.REFs[0]
	assertObserveBaselineAxes(t, doc)
	for i, args := range commands {
		if got := mustRunCLI(t, dir, args...); got != before[i] {
			t.Fatalf("%v changed\nbefore %s\nafter %s", args, before[i], got)
		}
	}
	mustRunCLI(t, dir, "add", "s", "--content", "two", "--draft")
	doc = observeCLI(t, dir, "s")
	ref = doc.REFs[0]
	if ref.Candidate.CurrentHeadComparison != "DIFFERENT" || !ref.Candidate.Draft || *ref.HeadDraft {
		t.Fatal(ref)
	}
	human := mustRunCLI(t, dir, "observe", "s", "--format", "human")
	for _, want := range []string{"current HEAD comparison=DIFFERENT", "publication baseline expected_ref_head=", "Workfile path=", "baseline=CANDIDATE", "not admission simulation", "Candidate=PRESENT draft=true", "HEAD="} {
		if !strings.Contains(human, want) {
			t.Fatalf("missing %q in %s", want, human)
		}
	}
	id := ref.Head.String()
	selected := observeCLI(t, dir, "@"+id)
	if selected.Scope != "SELECTED_SEAL" || len(selected.REFs) != 0 || len(selected.Seals) != 1 || selected.Seals[0].SealID.String() != id {
		t.Fatal(selected)
	}
	inventory := observeCLI(t, dir, "--inventory")
	if inventory.Scope != "RETAINED_INVENTORY" {
		t.Fatal(inventory)
	}
}
func TestObserveCLIRejectsInvalidOptionsAndBootstrap(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{{"observe"}, {"observe", "x", "--inventory"}, {"observe", "--max-paths", "0"}, {"observe", "--max-paths", "10001"}, {"observe", "--max-paths", "1", "--max-paths", "2"}, {"observe", "x", "y"}, {"observe", "--format", "raw"}} {
		code, out, _ := runCLI(t, dir, nil, args...)
		if code == 0 || out != "" {
			t.Fatalf("accepted invalid %v: %d %s", args, code, out)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("observe bootstrapped: %v", err)
	}
}
func TestObserveCLIHelpCompletionAndOutputFailure(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "new", "--root", "--clear-cause-links", "--content", "a")
	if doc := observeCLI(t, dir); len(doc.REFs) != 1 || doc.REFs[0].Head != nil || doc.REFs[0].HeadDraft != nil || doc.REFs[0].Candidate.CurrentHeadComparison != "NO_HEAD" {
		t.Fatal(doc)
	}
	for _, args := range [][]string{{"help", "observe"}, {"observe", "--help"}} {
		if out := mustRunCLI(t, dir, args...); !strings.Contains(out, "--inventory") {
			t.Fatal(out)
		}
	}
	_, values := bashCompletion(dir, []string{"observe", ""})
	if !strings.Contains(strings.Join(values, " "), "new") {
		t.Fatal(values)
	}
	_, values = bashCompletion(dir, []string{"observe", "--"})
	if !strings.Contains(strings.Join(values, " "), "--max-paths") {
		t.Fatal(values)
	}
	for _, format := range []string{"json", "human"} {
		var stderr bytes.Buffer
		if code := runObserve(t.Context(), dir, []string{"--format", format}, errorWriter{}, &stderr); code == 0 || stderr.Len() == 0 {
			t.Fatal("output failure hidden")
		}
	}
}

func assertObserveBaselineAxes(t *testing.T, doc repository.ObservationResult) {
	t.Helper()
	ref := doc.REFs[0]
	if !strings.Contains(doc.MembershipLimitation, "historical from detached") {
		t.Fatal("missing membership limitation")
	}
	if doc.Schema != "sealgraph/observation/v1" || doc.Scope != "SELECTED_REF" || ref.Candidate == nil || ref.Candidate.CurrentHeadComparison != "SAME" || ref.HeadDraft == nil || *ref.HeadDraft || ref.Candidate.Draft || ref.Source == nil || ref.Source.Baseline != "CANDIDATE" {
		t.Fatalf("mixed baselines: %+v", ref)
	}
}
