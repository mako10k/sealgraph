package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

type standaloneHarness struct {
	t      *testing.T
	dir    string
	stdout bytes.Buffer
	stderr bytes.Buffer
}

func newStandaloneHarness(t *testing.T) *standaloneHarness {
	t.Helper()
	return &standaloneHarness{t: t, dir: t.TempDir()}
}

func (h *standaloneHarness) run(args ...string) string {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	if code := runStandaloneAt(h.dir, args, &h.stdout, &h.stderr); code != 0 {
		h.t.Fatalf("%v code=%d stderr=%q", args, code, h.stderr.String())
	}
	return h.stdout.String()
}

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
	harness := newStandaloneHarness(t)
	dir, run := harness.dir, harness.run
	run("init")
	run("add", "requirements/ROOT", "--root", "--content", "requirement")
	sealed := run("seal", "requirements/ROOT", "-m", "initial")
	if fields := strings.Fields(sealed); len(fields) != 3 || len(fields[2]) != 64 || strings.Contains(fields[2], ":") {
		t.Fatalf("seal output = %q", sealed)
	}
	run("add", "design/api", "--content", "design", "--depend-on", "requirements/ROOT")
	run("seal", "design/api", "-m", "reviewed")
	shown := run("show", "design/api")
	if !strings.Contains(shown, "CONTENT ") || !strings.Contains(shown, "\ndesign\n") || !strings.Contains(shown, "depend-on requirements/ROOT@") {
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
	harness := newStandaloneHarness(t)
	run := harness.run
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

func TestTagPrefixAndLinkMessageCommands(t *testing.T) {
	harness := newStandaloneHarness(t)
	run := harness.run
	sealID := func(output, ref string) string {
		return strings.TrimSpace(strings.TrimPrefix(output, "SEALED "+ref+" "))
	}

	run("init")
	run("add", "ROOT", "--root", "--content", "v1")
	rootV1 := sealID(run("seal", "ROOT", "-m", "v1"), "ROOT")
	run("tag", "ROOT@"+rootV1[:12], "baseline/1.0")
	run("add", "ROOT", "--root", "--content", "v2")
	run("seal", "ROOT", "-m", "v2")
	if shown := run("show", "ROOT@baseline/1.0"); !strings.Contains(shown, "SEAL "+rootV1) {
		t.Fatalf("tag show = %q", shown)
	}
	if shown := run("show", "ROOT@"+rootV1[:12]); !strings.Contains(shown, "SEAL "+rootV1) {
		t.Fatalf("prefix show = %q", shown)
	}
	if listed := run("tag", "ROOT"); !strings.Contains(listed, "baseline/1.0 "+rootV1) {
		t.Fatalf("tag list = %q", listed)
	}
	run("add", "DESIGN", "--draft", "--content", "design", "--depend-on", "ROOT")
	run("link", "DESIGN", "--depend-on", "ROOT@baseline/1.0", "-m", "reviewed baseline")
	run("seal", "DESIGN", "-m", "historical design")
	shown := run("show", "DESIGN")
	if !strings.Contains(shown, "depend-on ROOT@"+rootV1) || !strings.Contains(shown, `message="reviewed baseline"`) {
		t.Fatalf("link message show = %q", shown)
	}
}

func TestAddContentFileAndStdinPreserveExactBytes(t *testing.T) {
	fileDir := t.TempDir()
	stdinDir := t.TempDir()
	content := []byte{'a', 0, 'b', '\r', '\n'}
	if err := os.WriteFile(filepath.Join(fileDir, "content.bin"), content, 0o644); err != nil {
		t.Fatal(err)
	}
	var fileOut, fileErr bytes.Buffer
	if code := runStandaloneAt(fileDir, []string{"init"}, &fileOut, &fileErr); code != 0 {
		t.Fatal(fileErr.String())
	}
	fileOut.Reset()
	if code := runStandaloneAt(fileDir, []string{"add", "ROOT", "--root", "--content-file", "content.bin"}, &fileOut, &fileErr); code != 0 {
		t.Fatalf("file add code=%d stderr=%q", code, fileErr.String())
	}
	fileID := strings.TrimPrefix(strings.Fields(fileOut.String())[2], "content=")

	var stdinOut, stdinErr bytes.Buffer
	if code := runStandaloneAt(stdinDir, []string{"init"}, &stdinOut, &stdinErr); code != 0 {
		t.Fatal(stdinErr.String())
	}
	stdinOut.Reset()
	if code := runStandaloneAtWithInput(stdinDir, []string{"add", "ROOT", "--root", "--content-file", "-"}, bytes.NewReader(content), &stdinOut, &stdinErr); code != 0 {
		t.Fatalf("stdin add code=%d stderr=%q", code, stdinErr.String())
	}
	stdinID := strings.TrimPrefix(strings.Fields(stdinOut.String())[2], "content=")
	if fileID != stdinID {
		t.Fatalf("file content ID = %s, stdin content ID = %s", fileID, stdinID)
	}

	stdinOut.Reset()
	stdinErr.Reset()
	if code := runStandaloneAtWithInput(stdinDir, []string{"add", "BAD", "--content", "x", "--content-file", "-"}, strings.NewReader("x"), &stdinOut, &stdinErr); code != 2 {
		t.Fatalf("conflicting content flags code=%d stderr=%q", code, stdinErr.String())
	}
	if _, err := os.Stat(filepath.Join(stdinDir, ".sealgraph", "index", "BAD")); !os.IsNotExist(err) {
		t.Fatalf("conflicting flags created candidate: %v", err)
	}
}

func TestGraphStaleAndImpactCommands(t *testing.T) {
	harness := newStandaloneHarness(t)
	run := harness.run
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
	if !strings.Contains(status, "LEAF STALE_TRANSITIVE") || !strings.Contains(status, "transitive path=LEAF@") || !strings.Contains(status, " -> MIDDLE@") || !strings.Contains(status, " -> ROOT@") {
		t.Fatalf("transitive status output = %q", status)
	}
	stale := run("stale")
	if !strings.Contains(stale, "MIDDLE STALE_DIRECT") || !strings.Contains(stale, "LEAF STALE_TRANSITIVE") || strings.Contains(stale, "ROOT CLEAN") {
		t.Fatalf("stale output = %q", stale)
	}
	impact := run("impact", "ROOT")
	if !strings.Contains(impact, "SOURCE ROOT@") || !strings.Contains(impact, "DIRECT MIDDLE@") || !strings.Contains(impact, "TRANSITIVE LEAF@") {
		t.Fatalf("impact output = %q", impact)
	}
	graph := run("graph")
	if !strings.Contains(graph, "REF MIDDLE@") || !strings.Contains(graph, "STALE_DIRECT") || !strings.Contains(graph, "depend-on ROOT@") || !strings.Contains(graph, "HISTORICAL head=") {
		t.Fatalf("graph output = %q", graph)
	}

	run("add", "team/name", "--root", "--content", "independent root")
	run("seal", "team/name", "-m", "hierarchical REF")
	teamImpact := run("impact", "team/name")
	if !strings.Contains(teamImpact, "SOURCE team/name@") || !strings.Contains(teamImpact, "NO_IMPACT") {
		t.Fatalf("hierarchical REF impact output = %q", teamImpact)
	}
}

func TestLogLinkLogAndSemanticDiffCommandsAreReadOnly(t *testing.T) {
	harness := newStandaloneHarness(t)
	dir, run := harness.dir, harness.run
	sealID := func(output, ref string) string {
		t.Helper()
		return strings.TrimSpace(strings.TrimPrefix(output, "SEALED "+ref+" "))
	}

	run("init")
	run("add", "ROOT", "--root", "--content", "root v1")
	rootV1 := sealID(run("seal", "ROOT", "-m", "root v1"), "ROOT")
	run("add", "design/api", "--content", "unchanged design", "--depend-on", "ROOT")
	designV1 := sealID(run("seal", "design/api", "-m", "reviewed root v1"), "design/api")
	run("add", "ROOT", "--root", "--content", "root v2")
	rootV2 := sealID(run("seal", "ROOT", "-m", "root v2"), "ROOT")
	run("link", "design/api", "--depend-on", "ROOT")
	designV2 := sealID(run("seal", "design/api", "-m", "reviewed root v2"), "design/api")

	before := snapshotRegularFiles(t, filepath.Join(dir, ".sealgraph"))
	logOutput := run("log", "design/api")
	if first, second := strings.Index(logOutput, "SEAL "+designV2), strings.Index(logOutput, "SEAL "+designV1); first < 0 || second <= first {
		t.Fatalf("log output is not newest first: %q", logOutput)
	}
	if !strings.Contains(logOutput, `MESSAGE "reviewed root v2"`) || !strings.Contains(logOutput, "depend-on ROOT@"+rootV1) {
		t.Fatalf("log output = %q", logOutput)
	}

	linkOutput := run("linklog", "design/api", "--upstream", "ROOT")
	if !strings.Contains(linkOutput, "UPSTREAM ROOT") || !strings.Contains(linkOutput, "LINK_REPOINT ROOT old="+rootV1+" new="+rootV2) || !strings.Contains(linkOutput, "LINK_ADD ROOT new="+rootV1) {
		t.Fatalf("linklog output = %q", linkOutput)
	}

	diffOutput := run("diff", "design/api")
	if !strings.Contains(diffOutput, "FROM "+designV1) || !strings.Contains(diffOutput, "TO "+designV2) || !strings.Contains(diffOutput, "CONTENT UNCHANGED") || !strings.Contains(diffOutput, "LINK_REPOINT ROOT old="+rootV1+" new="+rootV2) || !strings.Contains(diffOutput, `MESSAGE CHANGED old="reviewed root v1" new="reviewed root v2"`) {
		t.Fatalf("diff output = %q", diffOutput)
	}
	explicitOutput := run("diff", "design/api@"+designV1, "design/api@"+designV2)
	if explicitOutput != diffOutput {
		t.Fatalf("explicit diff differs from HEAD-parent diff:\nexplicit=%q\ncurrent=%q", explicitOutput, diffOutput)
	}
	rootDiff := run("diff", "ROOT")
	if !strings.Contains(rootDiff, "CONTENT CHANGED") || !strings.Contains(rootDiff, "LINKS UNCHANGED") {
		t.Fatalf("root diff output = %q", rootDiff)
	}

	after := snapshotRegularFiles(t, filepath.Join(dir, ".sealgraph"))
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("read-only inspection changed repository files:\nbefore=%v\nafter=%v", before, after)
	}

	harness.stdout.Reset()
	harness.stderr.Reset()
	if code := runStandaloneAt(dir, []string{"diff", "ROOT@" + rootV1, "design/api@" + designV2}, &harness.stdout, &harness.stderr); code != 2 || !strings.Contains(harness.stderr.String(), "one logical REF") {
		t.Fatalf("cross-REF diff code=%d stderr=%q", code, harness.stderr.String())
	}
}

func snapshotRegularFiles(t *testing.T, root string) map[string]string {
	t.Helper()
	result := make(map[string]string)
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		result[relative] = string(data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return result
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
