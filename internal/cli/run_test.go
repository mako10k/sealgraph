package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStandaloneHelpDoesNotMentionGitDetection(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := RunStandalone([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if strings.Contains(strings.ToLower(out.String()), "detected") {
		t.Fatalf("standalone help unexpectedly implies environment detection: %q", out.String())
	}
	if errOut.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", errOut.String())
	}
}

func TestSealCommandRejectsMultipleREFs(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	if code := runStandaloneAt(dir, []string{"init"}, &out, &errOut); code != 0 {
		t.Fatalf("init code=%d stderr=%q", code, errOut.String())
	}
	out.Reset()
	errOut.Reset()
	code := runStandaloneAt(dir, []string{"seal", "REF-A", "REF-B", "-m", "message"}, &out, &errOut)
	if code != 2 {
		t.Fatalf("code = %d, want usage error 2; stderr=%q", code, errOut.String())
	}
	if !strings.Contains(errOut.String(), "exactly one REF") {
		t.Fatalf("stderr = %q, want one-REF explanation", errOut.String())
	}
}

func TestStandaloneEndToEndCommands(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	run := func(args ...string) string {
		t.Helper()
		out.Reset()
		errOut.Reset()
		if code := runStandaloneAt(dir, args, &out, &errOut); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		return out.String()
	}
	run("init")
	run("add", "requirements/ROOT", "--root", "--content", "requirement")
	sealed := run("seal", "requirements/ROOT", "-m", "initial")
	if !strings.Contains(sealed, "SEALED requirements/ROOT sha256:") {
		t.Fatalf("seal output = %q", sealed)
	}
	run("add", "design/api", "--content", "design", "--depend-on", "requirements/ROOT")
	run("seal", "design/api", "-m", "reviewed")
	shown := run("show", "design/api")
	if !strings.Contains(shown, "CONTENT ") || !strings.Contains(shown, "\ndesign\n") || !strings.Contains(shown, "depend-on requirements/ROOT@sha256:") {
		t.Fatalf("show output = %q", shown)
	}
	status := run("status")
	if !strings.Contains(status, "design/api CLEAN") {
		t.Fatalf("status output = %q", status)
	}
	if _, err := os.Stat(filepath.Join(dir, ".sealgraph", "refs", "seals", "design", "api")); err != nil {
		t.Fatalf("hierarchical REF file: %v", err)
	}
}

func TestExplicitInitBootstrapsRuntimeAfterCanonicalCheckout(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	run := func(args ...string) int {
		out.Reset()
		errOut.Reset()
		return runStandaloneAt(dir, args, &out, &errOut)
	}
	if code := run("init"); code != 0 {
		t.Fatalf("initial init code=%d stderr=%q", code, errOut.String())
	}
	if code := run("add", "ROOT", "--root", "--content", "root"); code != 0 {
		t.Fatalf("add code=%d stderr=%q", code, errOut.String())
	}
	if code := run("seal", "ROOT", "-m", "root"); code != 0 {
		t.Fatalf("seal code=%d stderr=%q", code, errOut.String())
	}
	repositoryDir := filepath.Join(dir, ".sealgraph")
	for _, relative := range []string{"index", "locks"} {
		if err := os.RemoveAll(filepath.Join(repositoryDir, relative)); err != nil {
			t.Fatal(err)
		}
	}
	if code := run("status"); code == 0 || !strings.Contains(errOut.String(), "run 'sealgraph init'") {
		t.Fatalf("status before bootstrap code=%d stderr=%q", code, errOut.String())
	}
	if code := run("init"); code != 0 {
		t.Fatalf("bootstrap init code=%d stderr=%q", code, errOut.String())
	}
	if code := run("status"); code != 0 || !strings.Contains(out.String(), "ROOT CLEAN") {
		t.Fatalf("status after bootstrap code=%d stdout=%q stderr=%q", code, out.String(), errOut.String())
	}
}

func TestLinkCommandAcceptsExplicitHistoricalSeal(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	run := func(args ...string) string {
		t.Helper()
		out.Reset()
		errOut.Reset()
		if code := runStandaloneAt(dir, args, &out, &errOut); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		return out.String()
	}
	run("init")
	run("add", "ROOT", "--root", "--content", "v1")
	v1Output := run("seal", "ROOT", "-m", "v1")
	v1ID := strings.TrimSpace(strings.TrimPrefix(v1Output, "SEALED ROOT "))
	run("add", "ROOT", "--root", "--content", "v2")
	run("seal", "ROOT", "-m", "v2")
	run("add", "REVIEW", "--draft", "--content", "review", "--depend-on", "ROOT")
	run("link", "REVIEW", "--depend-on", "ROOT@"+v1ID)
	run("seal", "REVIEW", "-m", "historical")
	shown := run("show", "REVIEW")
	if !strings.Contains(shown, "depend-on ROOT@"+v1ID) {
		t.Fatalf("show output = %q, want historical target %s", shown, v1ID)
	}
	status := run("status", "REVIEW")
	if !strings.Contains(status, "DRAFT,STALE_DIRECT") {
		t.Fatalf("status output = %q", status)
	}
}

func TestGraphStaleAndImpactCommands(t *testing.T) {
	dir := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer
	run := func(args ...string) string {
		t.Helper()
		out.Reset()
		errOut.Reset()
		if code := runStandaloneAt(dir, args, &out, &errOut); code != 0 {
			t.Fatalf("%v code=%d stderr=%q", args, code, errOut.String())
		}
		return out.String()
	}
	run("init")
	run("add", "ROOT", "--root", "--content", "root v1")
	run("seal", "ROOT", "-m", "root v1")
	run("add", "MIDDLE", "--content", "middle", "--depend-on", "ROOT")
	run("seal", "MIDDLE", "-m", "middle v1")
	run("add", "LEAF", "--content", "leaf", "--depend-on", "MIDDLE")
	run("seal", "LEAF", "-m", "leaf v1")
	run("add", "ROOT", "--root", "--content", "root v2")
	run("seal", "ROOT", "-m", "root v2")

	status := run("status", "LEAF")
	if !strings.Contains(status, "LEAF STALE_TRANSITIVE") || !strings.Contains(status, "transitive path=LEAF@sha256:") || !strings.Contains(status, " -> MIDDLE@sha256:") || !strings.Contains(status, " -> ROOT@sha256:") {
		t.Fatalf("transitive status output = %q", status)
	}
	stale := run("stale")
	if !strings.Contains(stale, "MIDDLE STALE_DIRECT") || !strings.Contains(stale, "LEAF STALE_TRANSITIVE") || strings.Contains(stale, "ROOT CLEAN") {
		t.Fatalf("stale output = %q", stale)
	}
	impact := run("impact", "ROOT")
	if !strings.Contains(impact, "SOURCE ROOT@sha256:") || !strings.Contains(impact, "DIRECT MIDDLE@sha256:") || !strings.Contains(impact, "TRANSITIVE LEAF@sha256:") {
		t.Fatalf("impact output = %q", impact)
	}
	graph := run("graph")
	if !strings.Contains(graph, "REF MIDDLE@sha256:") || !strings.Contains(graph, "STALE_DIRECT") || !strings.Contains(graph, "depend-on ROOT@sha256:") || !strings.Contains(graph, "HISTORICAL head=sha256:") {
		t.Fatalf("graph output = %q", graph)
	}

	run("add", "team/@name", "--root", "--content", "independent root")
	run("seal", "team/@name", "-m", "valid at-sign REF")
	atImpact := run("impact", "team/@name")
	if !strings.Contains(atImpact, "SOURCE team/@name@sha256:") || !strings.Contains(atImpact, "NO_IMPACT") {
		t.Fatalf("at-sign REF impact output = %q", atImpact)
	}
}

func TestGitPluginHasSeparateHelpSurface(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	code := RunGitPlugin([]string{"--help"}, &out, &errOut)
	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(out.String(), "Git sidecar integration") {
		t.Fatalf("help = %q, want Git sidecar description", out.String())
	}
}
