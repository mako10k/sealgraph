package cli

// R: verify the explicit historical-review CLI, its diagnostics and bounded inputs.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/historicalreview"
	"github.com/mako10k/sealgraph/internal/repository"
)

func TestCLIHistoricalReviewExplicitWorkflow(t *testing.T) {
	dir := t.TempDir()
	run := func(input string, args ...string) (int, string, string) {
		t.Helper()
		var out, err bytes.Buffer
		code := runStandaloneAtWithInput(dir, args, strings.NewReader(input), &out, &err)
		return code, out.String(), err.String()
	}
	success := func(args ...string) string {
		t.Helper()
		code, out, err := run("", args...)
		if code != 0 {
			t.Fatalf("%v code=%d err=%s", args, code, err)
		}
		return out
	}
	success("init")
	success("add", "premise", "--root", "--clear-cause-links", "--content", "old")
	old := strings.Fields(success("seal", "premise"))[2]
	success("add", "premise", "--content", "new")
	success("seal", "premise")
	success("add", "observer", "--non-root", "--content", "revision evidence", "--target", "premise", "--previous", "@"+old)
	success("seal", "observer")
	success("add", "dependent", "--non-root", "--content", "old scope", "--target", "@"+old, "--no-previous")
	var facts repository.HistoricalReviewPreparation
	if err := json.Unmarshal([]byte(success("review", "prepare", "dependent", "--format", "json")), &facts); err != nil {
		t.Fatal(err)
	}
	record := historicalreview.Record{Schema: historicalreview.Schema, REF: facts.REF, SubjectSeal: facts.SubjectSeal, OtherHeads: facts.OtherHeads, Entries: []historicalreview.Entry{}}
	for _, target := range facts.Targets {
		record.Entries = append(record.Entries, historicalreview.Entry{TargetSeal: target.TargetSeal, ComparedSeals: target.ComparedSeals, Decision: "retain", Difference: "Changed requirement version.", Impact: "This content remains within the old scope.", Reason: "Old contract is deliberately retained."})
	}
	data, err := historicalreview.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	if code, _, stderr := run(string(data), "review", "set", "dependent", "--file", "-", "--format", "json"); code != 0 {
		t.Fatal(stderr)
	}
	if code, _, _ := run("", "seal", "dependent"); code == 0 {
		t.Fatal("default admission accepted stale")
	}
	show := decodeCLIJSON(t, success("review", "show", "dependent", "--candidate", "--format", "json"))
	if show["applicable"] != true {
		t.Fatalf("inspection=%+v", show)
	}
	success("seal", "dependent", "--review-historical")
	human := success("review", "show", "dependent", "--format", "human")
	for _, marker := range []string{"Claim currently applicable", "executor claim only", "Old contract is deliberately retained.", old} {
		if !strings.Contains(human, marker) {
			t.Fatalf("human output lacks %q: %s", marker, human)
		}
	}
	show = decodeCLIJSON(t, success("review", "show", "dependent", "--format", "json"))
	if show["applicable"] != true {
		t.Fatalf("published=%+v", show)
	}
	stale := success("stale", "--refs-only")
	if !strings.Contains(stale, "dependent\n") {
		t.Fatalf("published historical dependency disappeared from stale: %q", stale)
	}
	success("fsck", "--format", "json")
	success("add", "dependent", "--content", "changed subject")
	show = decodeCLIJSON(t, success("review", "show", "dependent", "--candidate", "--format", "json"))
	if show["applicable"] != false {
		t.Fatal("changed subject showed applicable review")
	}
	success("review", "clear", "dependent", "--format", "json")
	if code, _, _ := run("", "review", "show", "dependent", "--candidate"); code == 0 {
		t.Fatal("clear left review")
	}
}

func TestCLIHistoricalReviewUsageAndBoundedInput(t *testing.T) {
	dir := t.TempDir()
	if _, err := repository.InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"seal", "a", "b"}, {"seal", "a", "--force"}, {"seal", "a", "--review-historical", "b"}, {"review", "prepare", "a", "b"}, {"review", "set", "a"}, {"review", "show", "a", "--candidate", "--candidate=false"}, {"review", "set", "a", "--file", "a", "--file", "b"}} {
		var out, err bytes.Buffer
		if code := runStandaloneAtWithInput(dir, args, strings.NewReader(""), &out, &err); code == 0 {
			t.Fatalf("invalid usage accepted: %v", args)
		}
	}
	if _, err := readHistoricalReviewInput(dir, "-", strings.NewReader(strings.Repeat(" ", historicalreview.MaxBytes+1))); err == nil {
		t.Fatal("oversized input accepted")
	}
	path := filepath.Join(dir, "review.json")
	if err := os.WriteFile(path, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(path, filepath.Join(dir, "link.json")); err != nil {
		t.Fatal(err)
	}
	if _, err := readHistoricalReviewInput(dir, "link.json", nil); err == nil {
		t.Fatal("symlink accepted")
	}
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.CompletionNames(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestHistoricalReviewHelpAndCompletion(t *testing.T) {
	for _, path := range []string{"review", "review prepare", "review set", "review clear", "review show", "seal"} {
		var out bytes.Buffer
		if !printHelpTopic(&out, strings.Fields(path)) {
			t.Fatalf("missing help %s", path)
		}
	}
	mode, names := bashCompletion(t.TempDir(), []string{"review", ""})
	if mode != "plain" || !strings.Contains(strings.Join(names, " "), "prepare") {
		t.Fatalf("review completion=%v", names)
	}
	_, names = bashCompletion(t.TempDir(), []string{"seal", "--"})
	if !strings.Contains(strings.Join(names, " "), "--review-historical") {
		t.Fatalf("seal option completion=%v", names)
	}
	names = reviewSubjectCompletion([]string{"existing"}, []string{"existing", "unsealed"})
	if strings.Join(names, " ") != "existing unsealed" {
		t.Fatalf("subject completion=%v", names)
	}
}
