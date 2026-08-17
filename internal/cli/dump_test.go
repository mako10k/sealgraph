package cli

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestDumpCommandRequiresExactVersionBeforeRepositoryTraversal(t *testing.T) {
	dir := t.TempDir()
	for _, args := range [][]string{
		{"dump"},
		{"dump", "--format", "json"},
		{"dump", "--format", "logical-v1", "extra"},
		{"dump", "--format", "logical-v1", "--format", "logical-v1"},
	} {
		var stdout, stderr bytes.Buffer
		code := runStandaloneAt(dir, args, &stdout, &stderr)
		if code != 2 || stdout.Len() != 0 {
			t.Fatalf("%v code=%d stdout=%q stderr=%q", args, code, stdout.String(), stderr.String())
		}
	}
}

func TestDumpCommandEmitsCanonicalDocumentAndNoDiagnostics(t *testing.T) {
	harness := newStandaloneHarness(t)
	harness.run("init")
	harness.run("add", "ROOT", "--root", "--content", "root")
	harness.run("seal", "ROOT")
	output := harness.run("dump", "--format", "logical-v1")
	if harness.stderr.Len() != 0 {
		t.Fatalf("stderr = %q", harness.stderr.String())
	}
	if !strings.HasSuffix(output, "\n") || strings.HasSuffix(output, "\n\n") {
		t.Fatalf("dump trailing LF = %q", output)
	}
	var document struct {
		Schema string `json:"schema"`
		Seals  []any  `json:"seals"`
		REFs   []any  `json:"refs"`
	}
	if err := json.Unmarshal([]byte(output), &document); err != nil {
		t.Fatal(err)
	}
	if document.Schema != "sealgraph/logical-dump/v1" || len(document.Seals) != 1 || len(document.REFs) != 1 {
		t.Fatalf("dump document = %+v", document)
	}
}

func TestDumpCommandCandidateFailureHasZeroStdout(t *testing.T) {
	harness := newStandaloneHarness(t)
	harness.run("init")
	harness.run("add", "ROOT", "--root", "--content", "unsealed")
	harness.stdout.Reset()
	harness.stderr.Reset()
	code := runStandaloneAt(harness.dir, []string{"dump", "--format", "logical-v1"}, &harness.stdout, &harness.stderr)
	if code != 1 || harness.stdout.Len() != 0 || !strings.Contains(harness.stderr.String(), "working candidate ROOT blocks logical dump") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, harness.stdout.String(), harness.stderr.String())
	}
}
