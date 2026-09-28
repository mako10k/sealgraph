package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/mako10k/sealgraph/internal/repository"
)

func newTraceSourceCLIRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if _, err := repository.InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestTraceSourceCompletionUsesSourceKeys(t *testing.T) {
	dir := newTraceSourceCLIRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "trace", "source", "bind", "manual-A", "--file", "source.txt")
	for _, action := range []string{"show", "rebind", "unbind"} {
		got := mustRunCLI(t, dir, "__completion", "--bash", "trace", "source", action, "")
		if got != "__sealgraph_completion_mode=plain\nmanual-A\n" {
			t.Fatalf("%s completion=%q", action, got)
		}
	}
	for _, action := range []string{"rebind", "unbind"} {
		got := mustRunCLI(t, dir, "__completion", "--bash", "trace", "source", action, "manual-A", "--from", "")
		if got != "__sealgraph_completion_mode=file\n" {
			t.Fatalf("%s --from completion=%q", action, got)
		}
	}
}

func TestRunTraceSourceBindingMutationAndInspectionJSON(t *testing.T) {
	dir := newTraceSourceCLIRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := runTraceSource(context.Background(), dir, []string{"bind", "manual-A", "--file", "source.txt", "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("bind code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	var bind struct {
		Schema    string `json:"schema"`
		Operation string `json:"operation"`
		SourceKey string `json:"source_key"`
		Before    any    `json:"before"`
		After     any    `json:"after"`
		Changed   bool   `json:"changed"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &bind); err != nil {
		t.Fatal(err)
	}
	if bind.Schema != "sealgraph/trace-source-mutation/v1" || bind.Operation != "bind" || bind.SourceKey != "manual-A" || bind.Before != nil || !bind.Changed {
		t.Fatalf("bind receipt=%s", stdout.String())
	}

	stdout.Reset()
	code = runTraceSource(context.Background(), dir, []string{"bind", "manual-A", "--file", "source.txt", "--format", "json"}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("idempotent bind code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if !bytes.Contains(stdout.Bytes(), []byte(`"changed":false`)) {
		t.Fatalf("idempotent bind receipt=%s", stdout.String())
	}

	stdout.Reset()
	code = runTraceSource(context.Background(), dir, []string{"show", "manual-A", "--format", "json"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"schema":"sealgraph/trace-source-list/v1"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"source_key":"manual-A"`)) {
		t.Fatalf("show code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	code = runTraceSource(context.Background(), dir, []string{"unbind", "manual-A", "--from", "source.txt", "--format", "json"}, &stdout, &stderr)
	if code != 0 || !bytes.Contains(stdout.Bytes(), []byte(`"operation":"unbind"`)) || !bytes.Contains(stdout.Bytes(), []byte(`"after":null`)) {
		t.Fatalf("unbind code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunTraceSourceListSortsBySourceKey(t *testing.T) {
	dir := newTraceSourceCLIRepo(t)
	for _, name := range []string{"a.txt", "b.txt"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(name), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, args := range [][]string{
		{"bind", "z-key", "--file", "a.txt", "--format", "json"},
		{"bind", "a-key", "--file", "b.txt", "--format", "json"},
	} {
		var stdout, stderr bytes.Buffer
		if code := runTraceSource(context.Background(), dir, args, &stdout, &stderr); code != 0 {
			t.Fatalf("bind args=%v code=%d stderr=%q", args, code, stderr.String())
		}
	}
	var stdout, stderr bytes.Buffer
	if code := runTraceSource(context.Background(), dir, []string{"list", "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("list code=%d stderr=%q", code, stderr.String())
	}
	first := bytes.Index(stdout.Bytes(), []byte(`"source_key":"a-key"`))
	second := bytes.Index(stdout.Bytes(), []byte(`"source_key":"z-key"`))
	if first < 0 || second < 0 || first > second {
		t.Fatalf("list ordering=%s", stdout.String())
	}
}

func TestRunTraceSourceBindReportsCorruptExistingBinding(t *testing.T) {
	dir := newTraceSourceCLIRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "source.txt"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	args := []string{"bind", "manual-A", "--file", "source.txt", "--format", "json"}
	if code := runTraceSource(context.Background(), dir, args, &stdout, &stderr); code != 0 {
		t.Fatalf("initial bind code=%d stderr=%q", code, stderr.String())
	}
	bindings, err := filepath.Glob(filepath.Join(dir, ".sealgraph", "local", "trace-sources", "*.json"))
	if err != nil || len(bindings) != 1 {
		t.Fatalf("binding files=%v err=%v", bindings, err)
	}
	if err := os.WriteFile(bindings[0], []byte("corrupt"), 0o600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	if code := runTraceSource(context.Background(), dir, args, &stdout, &stderr); code == 0 || stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("corrupt")) {
		t.Fatalf("corrupt bind code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}
