package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/migration"
)

func decodeCLIJSON(t *testing.T, output string) map[string]any {
	t.Helper()
	var value map[string]any
	if err := json.Unmarshal([]byte(output), &value); err != nil {
		t.Fatalf("decode JSON %q: %v", output, err)
	}
	return value
}

func TestCLIInitReportsAllThreeOutcomesWithoutPaths(t *testing.T) {
	dir := t.TempDir()
	if output := mustRunCLI(t, dir, "init"); output != "INITIALIZED standalone repository runtime=index,locks\n" || strings.Contains(output, dir) {
		t.Fatalf("initial output=%q", output)
	}
	if output := mustRunCLI(t, dir, "init"); output != "ALREADY_COMPLETE\n" {
		t.Fatalf("complete output=%q", output)
	}
	if err := os.Remove(filepath.Join(dir, ".sealgraph", "locks")); err != nil {
		t.Fatal(err)
	}
	if output := mustRunCLI(t, dir, "init"); output != "BOOTSTRAPPED_RUNTIME locks\n" || strings.Contains(output, dir) {
		t.Fatalf("bootstrap output=%q", output)
	}
}

func TestCLIHelpDefinesOperatorSemanticsWithoutGitDiscovery(t *testing.T) {
	code, stdout, stderr := runCLI(t, t.TempDir(), nil, "help")
	if code != 0 || stderr != "" {
		t.Fatalf("help code=%d stderr=%q", code, stderr)
	}
	for _, phrase := range []string{"status separates CANDIDATE_TO_HEAD", "local source binding is not Git tracked membership", "REF is a movable logical identity", "STRUCTURAL_IMPACT", "root marks a provenance boundary", "does not discover or inspect Git"} {
		if !strings.Contains(stdout, phrase) {
			t.Fatalf("help lacks %q: %s", phrase, stdout)
		}
	}
}

func TestCLIHelpHierarchyIsRepositoryIndependent(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dir, ".git"), 0o700) })
	tests := []struct {
		args    []string
		markers []string
	}{
		{[]string{"--help"}, []string{"Commands:", "Topics:", "sealgraph help candidate show"}},
		{[]string{"help"}, []string{"Commands:", "Navigation explains explicit next actions"}},
		{[]string{"help", "add"}, []string{"sealgraph add REF", "--depend-on SELECTOR", "repeatable", "mutually exclusive"}},
		{[]string{"add", "--help"}, []string{"sealgraph add REF", "--content-file PATH|-"}},
		{[]string{"help", "candidate"}, []string{"Subcommands:", "show", "compare", "discard"}},
		{[]string{"help", "candidate", "show"}, []string{"sealgraph candidate show REF", "--raw-content"}},
		{[]string{"candidate", "show", "--help"}, []string{"expected REF-head relations", "sealgraph candidate show REF"}},
		{[]string{"help", "source"}, []string{"Subcommands:", "bind", "rebind", "unbind", "show", "list", "not Git tracked state"}},
		{[]string{"source", "show", "--help"}, []string{"without opening its source file", "sealgraph source show REF"}},
		{[]string{"help", "source", "rebind"}, []string{"--from OLD_PATH", "--file NEW_PATH", "Atomically replace"}},
		{[]string{"help", "ref", "drop"}, []string{"sealgraph ref drop REF", "complete tag namespace", "Immutable objects remain valid"}},
		{[]string{"help", "selectors"}, []string{"@SEAL_TOKEN", "4 through 64 lower-case hex", "There is no @latest", "full 64-character SealID"}},
		{[]string{"help", "concepts"}, []string{"parent_revision", "CLEAN means", "never searches for Git"}},
		{[]string{"help", "concepts", "root"}, []string{"not mean true, trusted, or approved"}},
		{[]string{"help", "usecases"}, []string{"Create the first root", "Review stale provenance upstream-first", "not automatic repair procedures"}},
		{[]string{"help", "impact"}, []string{"--max-paths N", "requires --all-paths", "default 100", "never removes impact membership"}},
	}
	for _, test := range tests {
		code, stdout, stderr := runCLI(t, dir, nil, test.args...)
		if code != 0 || stderr != "" {
			t.Fatalf("%v code=%d stderr=%q", test.args, code, stderr)
		}
		for _, marker := range test.markers {
			if !strings.Contains(stdout, marker) {
				t.Fatalf("%v lacks %q:\n%s", test.args, marker, stdout)
			}
		}
	}
	if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("help created repository state: %v", err)
	}
}

type cliDiagnosticCase struct {
	args    []string
	markers []string
}

func assertCLIUsageDiagnostics(t *testing.T, tests []cliDiagnosticCase) {
	t.Helper()
	for _, test := range tests {
		code, stdout, stderr := runCLI(t, t.TempDir(), nil, test.args...)
		if code != 2 || stdout != "" {
			t.Fatalf("%v code=%d stdout=%q stderr=%q", test.args, code, stdout, stderr)
		}
		for _, marker := range test.markers {
			if !strings.Contains(stderr, marker) {
				t.Fatalf("%v lacks %q:\n%s", test.args, marker, stderr)
			}
		}
	}
}

func TestCLIUsageAndUnknownNavigation(t *testing.T) {
	tests := []cliDiagnosticCase{
		{[]string{"impcat", "x"}, []string{`error: unknown command "impcat"`, "possible command: impact", "sealgraph help impact"}},
		{[]string{"candidate", "foo"}, []string{`unknown candidate operation "foo"`, "sealgraph candidate <show|compare|discard>", "sealgraph help candidate"}},
		{[]string{"show"}, []string{"show requires exactly one", "usage: sealgraph show SELECTOR", "help: sealgraph help show"}},
		{[]string{"impact", "--unknown", "root"}, []string{"flag provided but not defined", "usage: sealgraph impact", "help: sealgraph help impact"}},
		{[]string{"impact", "--max-paths", "10", "root"}, []string{"--max-paths is valid only with --all-paths", "use `sealgraph impact --all-paths --max-paths 10 root`", "help: sealgraph help impact"}},
		{[]string{"show", "root", "--raw-content", "--format", "json"}, []string{"mutually exclusive", "usage: sealgraph show", "help: sealgraph help show"}},
		{[]string{"show", "@latest"}, []string{"invalid selector", "4 to 64 lower-case hexadecimal", "help: sealgraph help show"}},
	}
	assertCLIUsageDiagnostics(t, tests)
}

func TestCLIGitShapedMisuseNavigatesToCanonicalVocabulary(t *testing.T) {
	tests := []cliDiagnosticCase{
		{[]string{"diff", "--cached", "spec"}, []string{"Git-shaped vocabulary", "index semantics", "candidate compare", "source compare"}},
		{[]string{"diff", "--draft", "spec"}, []string{"DRAFT is an identity-bearing", "candidate compare"}},
		{[]string{"candidate", "diff", "spec"}, []string{"retired Git-shaped vocabulary", "candidate compare"}},
		{[]string{"rm", "spec"}, []string{"does not delete workfiles", "candidate discard", "source unbind", "sealgraph ref drop REF"}},
		{[]string{"add", "-A"}, []string{"worktree-wide staging", "exactly one named REF", "sealgraph add REF"}},
		{[]string{"checkout", "spec"}, []string{"not a checked-out branch", "sealgraph status"}},
	}
	assertCLIUsageDiagnostics(t, tests)
}

func TestLocalSourceErrorsProvideExplicitNextActions(t *testing.T) {
	tests := []struct {
		command string
		message string
		markers []string
	}{
		{"add", "existing REF spec has no local source binding", []string{"no machine-local source binding", "source list", "source bind"}},
		{"source bind", `local source for spec is already "one.md"; use source rebind`, []string{"create-only", "source show", "source rebind"}},
		{"source rebind", `local source for spec is "one.md", not expected "old.md"`, []string{"changed after the operator's observation", "source show"}},
		{"add", `REF spec is bound to "one.md", not explicit source "two.md"`, []string{"would disagree", "source rebind", "source unbind"}},
		{"add", "CHANGED_DURING_READ: exact bytes changed", []string{"changed or was replaced", "no plausible candidate"}},
		{"mv", `local source binding spec -> "spec.md" blocks REF-only move`, []string{"never moves a working file", "unbind", "bind the new REF"}},
	}
	for _, test := range tests {
		reason, hints := domainNavigation(test.command, test.message)
		text := reason + " " + strings.Join(hints, " ")
		for _, marker := range test.markers {
			if !strings.Contains(text, marker) {
				t.Fatalf("%s/%q lacks %q: %s", test.command, test.message, marker, text)
			}
		}
	}
}

func TestCLIDomainInvariantFailureNavigatesWithoutMutation(t *testing.T) {
	fixture := newCLIRevisionFixture(t)
	mustRunCLI(t, fixture.dir, "add", "middle", "--content", "review", "--depend-on", "@"+fixture.root1, "--depend-on", "root")
	candidatePath := filepath.Join(fixture.dir, ".sealgraph", "index", "middle", ".candidate")
	before, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := runCLI(t, fixture.dir, nil, "seal", "middle")
	if code != 1 || stdout != "" {
		t.Fatalf("seal code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	for _, marker := range []string{"reason: normal non-draft publication requires", "sealgraph stale --frontier", "keep the candidate draft", "help: sealgraph help seal"} {
		if !strings.Contains(stderr, marker) {
			t.Fatalf("missing %q:\n%s", marker, stderr)
		}
	}
	after, err := os.ReadFile(candidatePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("failed seal changed candidate state")
	}
}

func TestCLIHelpUseCaseInvocationsAreAcceptedByTheRuntime(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "design.md"), []byte("API design"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "premise", "--root", "--content", "External premise")
	mustSealCLI(t, dir, "premise")
	mustRunCLI(t, dir, "add", "requirements/api", "--root", "--content", "API requirement")
	requirementID := mustSealCLI(t, dir, "requirements/api")
	mustRunCLI(t, dir, "tag", "requirements/api", "reviewed/1.0")
	mustRunCLI(t, dir, "add", "design/api", "--content-file", "design.md", "--depend-on", "requirements/api")
	mustRunCLI(t, dir, "candidate", "show", "design/api")
	mustRunCLI(t, dir, "candidate", "compare", "design/api")
	mustSealCLI(t, dir, "design/api")
	mustRunCLI(t, dir, "show", "requirements/api@reviewed/1.0")
	mustRunCLI(t, dir, "show", "@"+requirementID[:12])
	mustRunCLI(t, dir, "impact", "requirements/api")
	mustRunCLI(t, dir, "impact", "--all-paths", "--max-paths", "20", "requirements/api")
	mustRunCLI(t, dir, "add", "requirements/api", "--root", "--content", "API requirement v2")
	mustSealCLI(t, dir, "requirements/api")
	mustRunCLI(t, dir, "stale", "--frontier")
	mustRunCLI(t, dir, "status", "design/api")
}

func TestCLIInspectionJSONSchemasAndStructuredPaths(t *testing.T) {
	fixture := newCLIRevisionFixture(t)
	commands := []struct {
		name, schema string
		args         []string
	}{
		{"show", "sealgraph/show/v1", []string{"show", "root", "--format", "json"}},
		{"status", "sealgraph/status/v2", []string{"status", "--format=json"}},
		{"stale", "sealgraph/stale/v1", []string{"stale", "--format", "json", "--frontier"}},
		{"graph", "sealgraph/graph/v1", []string{"graph", "--format", "json"}},
		{"impact", "sealgraph/impact/v1", []string{"impact", "@" + fixture.root2, "--format", "json"}},
		{"log", "sealgraph/log/v1", []string{"log", "root", "--format", "json"}},
		{"linklog", "sealgraph/linklog/v1", []string{"linklog", "middle", "--format", "json"}},
		{"compare", "sealgraph/compare/v1", []string{"compare", "root", "--format", "json"}},
	}
	for _, test := range commands {
		t.Run(test.name, func(t *testing.T) {
			output := mustRunCLI(t, fixture.dir, test.args...)
			value := decodeCLIJSON(t, output)
			if value["schema"] != test.schema {
				t.Fatalf("schema=%v output=%s", value["schema"], output)
			}
		})
	}
	impact := decodeCLIJSON(t, mustRunCLI(t, fixture.dir, "impact", "@"+fixture.root2, "--format", "json"))
	items := impact["impacts"].([]any)
	if len(items) == 0 {
		t.Fatal("missing impact")
	}
	paths := items[0].(map[string]any)["paths"].([]any)
	if _, ok := paths[0].([]any); !ok {
		t.Fatalf("path is not a structured array: %#v", paths[0])
	}
	if code, stdout, stderr := runCLI(t, fixture.dir, nil, "stale", "--refs-only", "--format", "json"); code != 2 || stdout != "" || !strings.Contains(stderr, "mutually exclusive") {
		t.Fatalf("mixed format code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestCLIFsckHumanAndJSON(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root")
	mustSealCLI(t, dir, "root")
	if output := mustRunCLI(t, dir, "fsck"); !strings.HasPrefix(output, "FSCK_OK objects=2 seals=1 material_objects=1 refs=1") {
		t.Fatalf("fsck output=%q", output)
	}
	value := decodeCLIJSON(t, mustRunCLI(t, dir, "fsck", "--format", "json"))
	if value["schema"] != "sealgraph/fsck/v1" || value["result"] != "ok" {
		t.Fatalf("fsck JSON=%#v", value)
	}
}

func TestCLIExactContentFileAndStdinRoundTripWithoutSeal(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	content := []byte{'A', 0, '\r', '\n', 0xff, 'Z'}
	path := filepath.Join(dir, "content.bin")
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	expectedID := domain.ComputeNativeBlobID(content).String()
	code, stdout, stderr := runCLI(t, dir, nil, "add", "root", "--root", "--content-file", "content.bin")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "content="+expectedID) {
		t.Fatalf("file add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "candidate", "show", "root", "--raw-content")
	if code != 0 || stderr != "" || !bytes.Equal([]byte(stdout), content) {
		t.Fatalf("file raw code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if code, stdout, stderr = runCLI(t, dir, nil, "show", "root"); code != 1 || stdout != "" || !strings.Contains(stderr, "REF not found") {
		t.Fatalf("add sealed unexpectedly code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	mustRunCLI(t, dir, "candidate", "discard", "root")
	code, stdout, stderr = runCLI(t, dir, content, "add", "root", "--root", "--content-file", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "content="+expectedID) {
		t.Fatalf("stdin add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestCLIUnsafeContentFilesFailBeforeCandidateMutation(t *testing.T) {
	tests := map[string]func(*testing.T, string) string{
		"missing": func(_ *testing.T, _ string) string { return "missing" },
		"directory": func(t *testing.T, dir string) string {
			t.Helper()
			if err := os.Mkdir(filepath.Join(dir, "directory"), 0o755); err != nil {
				t.Fatal(err)
			}
			return "directory"
		},
		"symlink": func(t *testing.T, dir string) string {
			t.Helper()
			if err := os.Symlink("target", filepath.Join(dir, "symlink")); err != nil {
				t.Fatal(err)
			}
			return "symlink"
		},
		"fifo": func(t *testing.T, dir string) string {
			t.Helper()
			if err := syscall.Mkfifo(filepath.Join(dir, "fifo"), 0o600); err != nil {
				t.Fatal(err)
			}
			return "fifo"
		},
		"device": func(t *testing.T, _ string) string {
			t.Helper()
			info, err := os.Lstat(os.DevNull)
			if err != nil || info.Mode().IsRegular() {
				t.Skip("platform has no non-regular os.DevNull")
			}
			return os.DevNull
		},
	}
	for name, prepare := range tests {
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			mustRunCLI(t, dir, "init")
			mustRunCLI(t, dir, "add", "root", "--root", "--content", "stable")
			candidatePath := filepath.Join(dir, ".sealgraph", "index", "root", ".candidate")
			before, err := os.ReadFile(candidatePath)
			if err != nil {
				t.Fatal(err)
			}
			source := prepare(t, dir)
			code, stdout, stderr := runCLI(t, dir, nil, "add", "root", "--root", "--content-file", source)
			if code != 1 || stdout != "" || stderr == "" {
				t.Fatalf("unsafe add code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			after, err := os.ReadFile(candidatePath)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Fatal("unsafe content input changed the existing candidate")
			}
		})
	}
}

func TestCLIContentSourceConflictFailsBeforeCandidateMutation(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	code, stdout, stderr := runCLI(t, dir, nil, "add", "root", "--root", "--content", "a", "--content-file", "missing")
	if code != 2 || stdout != "" || !strings.Contains(stderr, "at most one") {
		t.Fatalf("conflict code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".sealgraph", "index", "root", ".candidate")); !os.IsNotExist(err) {
		t.Fatalf("conflicting content flags created candidate: %v", err)
	}
}

func TestCLILocalSourceLifecycleAndContentlessRefresh(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte("v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	output := mustRunCLI(t, dir, "add", "spec.md", "--root", "--bind-source")
	if !strings.Contains(output, "source_mode=initial-ref-path") || !strings.Contains(output, "source_binding=BOUND") {
		t.Fatalf("initial output=%q", output)
	}
	show := mustRunCLI(t, dir, "source", "show", "spec.md")
	if !strings.Contains(show, `path="spec.md"`) {
		t.Fatalf("source show=%q", show)
	}
	jsonOutput := decodeCLIJSON(t, mustRunCLI(t, dir, "source", "list", "--format=json"))
	if jsonOutput["schema"] != "sealgraph/source/v1" {
		t.Fatalf("source JSON=%v", jsonOutput)
	}
	mustRunCLI(t, dir, "seal", "spec.md")
	if err := os.WriteFile(filepath.Join(dir, "spec.md"), []byte("v2"), 0o600); err != nil {
		t.Fatal(err)
	}
	output = mustRunCLI(t, dir, "add", "spec.md")
	if !strings.Contains(output, "root=true") || !strings.Contains(output, "source_mode=bound-source") {
		t.Fatalf("refresh output=%q", output)
	}
	status := mustRunCLI(t, dir, "status", "spec.md")
	if !strings.Contains(status, "WORKFILE_MATCHES_CANDIDATE") {
		t.Fatalf("status=%q", status)
	}
	comparison := mustRunCLI(t, dir, "source", "compare", "spec.md")
	if !strings.Contains(comparison, "baseline=CANDIDATE") || !strings.Contains(comparison, "WORKFILE_MATCHES_CANDIDATE") {
		t.Fatalf("source compare=%q", comparison)
	}
	comparisonJSON := decodeCLIJSON(t, mustRunCLI(t, dir, "source", "compare", "spec.md", "--format=json"))
	if comparisonJSON["schema"] != "sealgraph/source-compare/v1" || comparisonJSON["relation"] != "WORKFILE_MATCHES_CANDIDATE" {
		t.Fatalf("source compare JSON=%v", comparisonJSON)
	}
	mustRunCLI(t, dir, "source", "unbind", "spec.md", "--from", "spec.md")
	code, stdout, stderr := runCLI(t, dir, nil, "add", "spec.md")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "has no local source binding") {
		t.Fatalf("fallback code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestCLIBashCompletionUsesCanonicalVocabularyAndMetadataOnly(t *testing.T) {
	dir := t.TempDir()
	mode := mustRunCLI(t, dir, "__completion", "--bash", "")
	for _, value := range []string{"__sealgraph_completion_mode=plain", "compare", "candidate", "source"} {
		if !strings.Contains(mode, value) {
			t.Fatalf("top-level completion lacks %q: %s", value, mode)
		}
	}
	if strings.Contains(mode, "\ndiff\n") || strings.Contains(mode, "\nrm\n") {
		t.Fatalf("completion advertised Git-shaped names: %s", mode)
	}
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root")
	if err := os.WriteFile(filepath.Join(dir, "source.md"), []byte("source"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "source", "bind", "bound", "--file", "source.md")
	candidates := mustRunCLI(t, dir, "__completion", "--bash", "candidate", "compare", "")
	if !strings.Contains(candidates, "\nroot\n") || strings.Contains(candidates, "\nbound\n") {
		t.Fatalf("candidate completion=%q", candidates)
	}
	sources := mustRunCLI(t, dir, "__completion", "--bash", "source", "compare", "")
	if !strings.Contains(sources, "\nbound\n") || strings.Contains(sources, "\nroot\n") {
		t.Fatalf("source completion=%q", sources)
	}
	if output := mustRunCLI(t, dir, "__completion", "--bash", "source", "bind", "x", "--file", ""); output != "__sealgraph_completion_mode=file\n" {
		t.Fatalf("file completion=%q", output)
	}
}

func TestCLIRecoveryRequiresExactOperationAndRestoresREF(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "v1")
	sealOutput := mustRunCLI(t, dir, "seal", "root")
	fields := strings.Fields(sealOutput)
	if len(fields) != 4 || !strings.HasPrefix(fields[3], "operation=") {
		t.Fatalf("seal output=%q", sealOutput)
	}
	id := strings.TrimPrefix(fields[3], "operation=")
	show := mustRunCLI(t, dir, "recover", "show", id)
	if !strings.Contains(show, "status=RECOVERABLE") || !strings.Contains(show, "current=AFTER") {
		t.Fatalf("recover show=%q", show)
	}
	jsonOutput := decodeCLIJSON(t, mustRunCLI(t, dir, "recover", "show", id, "--format=json"))
	if jsonOutput["schema"] != "sealgraph/recover/v1" {
		t.Fatalf("recover JSON=%v", jsonOutput)
	}
	completion := mustRunCLI(t, dir, "__completion", "--bash", "recover", "")
	if !strings.Contains(completion, "\nshow\n") || !strings.Contains(completion, "\n"+id+"\n") {
		t.Fatalf("recover completion=%q", completion)
	}
	code, stdout, stderr := runCLI(t, dir, nil, "recover", id[:12])
	if code != 1 || stdout != "" || !strings.Contains(stderr, "exactly 32 lower-case hexadecimal") {
		t.Fatalf("prefix recovery code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	recovered := decodeCLIJSON(t, mustRunCLI(t, dir, "recover", id, "--format", "json"))
	if recovered["result"] != "RECOVERED" || recovered["operation_id"] != id {
		t.Fatalf("recover result=%v", recovered)
	}
	show = mustRunCLI(t, dir, "recover", "show", id)
	if !strings.Contains(show, "status=ALREADY_RECOVERED") || !strings.Contains(show, "current=BEFORE") {
		t.Fatalf("recovered show=%q", show)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "recover", id)
	if code != 1 || stdout != "" || !strings.Contains(stderr, "ALREADY_RECOVERED") {
		t.Fatalf("repeat recovery code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestCLIRefDropEmitsRecoverableReceipt(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "v1")
	mustRunCLI(t, dir, "seal", "root")
	if err := os.WriteFile(filepath.Join(dir, "root.txt"), []byte("v1"), 0o600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "source", "bind", "root", "--file", "root.txt")
	code, stdout, stderr := runCLI(t, dir, nil, "ref", "drop", "root")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "source binding") || !strings.Contains(stderr, "unbind explicitly") {
		t.Fatalf("source-blocked drop code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	mustRunCLI(t, dir, "source", "unbind", "root", "--from", "root.txt")
	output := mustRunCLI(t, dir, "ref", "drop", "root")
	fields := strings.Fields(output)
	if len(fields) != 5 || fields[0] != "REF_DROPPED" || !strings.HasPrefix(fields[4], "operation=") {
		t.Fatalf("drop output=%q", output)
	}
	id := strings.TrimPrefix(fields[4], "operation=")
	if show := mustRunCLI(t, dir, "recover", "show", id); !strings.Contains(show, "kind=ref-drop") || !strings.Contains(show, "status=RECOVERABLE") {
		t.Fatalf("drop recovery=%q", show)
	}
	mustRunCLI(t, dir, "recover", id)
	if show := mustRunCLI(t, dir, "show", "root"); !strings.Contains(show, "CURRENT_REFS root") {
		t.Fatalf("restored REF show=%q", show)
	}
}

func TestCLIManifestFeedsAddWithoutRepositoryMutation(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "b.txt"), []byte("b"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o600); err != nil {
		t.Fatal(err)
	}
	code, first, stderr := runCLI(t, dir, nil, "manifest", "--source", "git:abc", "--file", "b.txt", "--file", "a.txt")
	if code != 0 || stderr != "" {
		t.Fatalf("manifest code=%d stdout=%q stderr=%q", code, first, stderr)
	}
	code, second, stderr := runCLI(t, dir, nil, "manifest", "--file", "a.txt", "--source", "git:abc", "--file", "b.txt")
	if code != 0 || stderr != "" || first != second {
		t.Fatalf("reordered manifest code=%d equal=%t stderr=%q", code, first == second, stderr)
	}
	if _, err := os.Lstat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatalf("manifest created repository state: %v", err)
	}
	mustRunCLI(t, dir, "init")
	expectedID := domain.ComputeNativeBlobID([]byte(first)).String()
	code, stdout, stderr := runCLI(t, dir, []byte(first), "add", "manifest", "--root", "--content-file", "-")
	if code != 0 || stderr != "" || !strings.Contains(stdout, "content="+expectedID) {
		t.Fatalf("manifest add code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if code, stdout, stderr = runCLI(t, dir, nil, "show", "manifest"); code != 1 || stdout != "" || !strings.Contains(stderr, "REF not found") {
		t.Fatalf("manifest add sealed unexpectedly code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func runCLI(t *testing.T, dir string, stdin []byte, args ...string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	code := runStandaloneAtWithInput(dir, args, bytes.NewReader(stdin), &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

func TestCLIFormat4RootAndSelectorShow(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatalf("init code=%d stderr=%s", code, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "add", "root", "--root", "--content", "material"); code != 0 || !strings.Contains(stdout, "CANDIDATE root") {
		t.Fatalf("add code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, stdout, stderr := runCLI(t, dir, nil, "candidate", "show", "root")
	if code != 0 || !strings.Contains(stdout, "PARENT_REVISION -") || !strings.Contains(stdout, "EXPECTED_REF_HEAD -") || !strings.Contains(stdout, "EXPECTED_HEAD_STATE EXPECTED_ABSENT") {
		t.Fatalf("candidate show code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	code, stdout, stderr = runCLI(t, dir, nil, "seal", "root")
	if code != 0 || !strings.HasPrefix(stdout, "SEALED root ") {
		t.Fatalf("seal code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	id := strings.Fields(stdout)[2]
	code, stdout, stderr = runCLI(t, dir, nil, "show", "@"+id[:12])
	if code != 0 || !strings.Contains(stdout, "CURRENT_REFS root") || !strings.Contains(stdout, "PARENT_REVISION -") || strings.Contains(stdout, "target_ref") {
		t.Fatalf("show code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func cliLogicalDump(t *testing.T) []byte {
	t.Helper()
	data := []byte("migrated")
	contentID := domain.ComputeNativeBlobID(data)
	payload := migration.Format3SealPayload{
		Schema: migration.Format3SealSchema, REF: "ROOT",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Attachments: []domain.Attachment{}, Links: []migration.Format3Link{}, Root: true,
	}
	sealBytes, err := migration.EncodeFormat3Seal(payload)
	if err != nil {
		t.Fatal(err)
	}
	oldID := domain.ComputeNativeBlobID(sealBytes)
	dump, err := migration.EncodeLogicalDumpV1(migration.LogicalDumpV1{
		Objects: []migration.ObjectRecord{{ID: contentID, Data: data}},
		Seals:   []migration.SealRecord{{OldSealID: oldID, Payload: payload}},
		REFs:    []migration.RefRecord{{Name: "ROOT", Head: oldID}},
		Tags:    []migration.TagRecord{}, ExcludedObjects: []domain.ObjectID{},
	})
	if err != nil {
		t.Fatal(err)
	}
	return dump
}

func TestCLILoadPublishesOnlyAfterCanonicalInput(t *testing.T) {
	dir := t.TempDir()
	input := cliLogicalDump(t)
	code, stdout, stderr := runCLI(t, dir, input, "load", "--format", "logical-v1")
	if code != 0 || !strings.Contains(stdout, `"schema":"sealgraph/logical-load-receipt/v1"`) || stderr != "" {
		t.Fatalf("load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "ROOT"); code != 0 || !strings.Contains(stdout, `CONTENT_PREVIEW "migrated"`) {
		t.Fatalf("show loaded code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	other := t.TempDir()
	code, stdout, stderr = runCLI(t, other, bytes.TrimSuffix(input, []byte{'\n'}), "load", "--format", "logical-v1")
	if code != 1 || stdout != "" || !strings.Contains(stderr, "not canonical") {
		t.Fatalf("noncanonical load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestCLIFormat3DumpIsAbsentAndTagsMoveWithREF(t *testing.T) {
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "dump", "--format", "logical-v1"); code != 2 || stdout != "" || !strings.Contains(stderr, `unknown command "dump"`) {
		t.Fatalf("dump code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	for _, args := range [][]string{{"graph"}, {"stale", "--scan"}} {
		if code, _, stderr := runCLI(t, dir, nil, args...); code != 0 || stderr != "" {
			t.Fatalf("%v code=%d stderr=%s", args, code, stderr)
		}
	}
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root")
	rootID := mustSealCLI(t, dir, "root")
	if stdout := mustRunCLI(t, dir, "tag", "root", "reviewed/1.0"); !strings.Contains(stdout, rootID) {
		t.Fatalf("tag create stdout=%s", stdout)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "tag", "@"+rootID[:12], "global"); code != 2 || stdout != "" || !strings.Contains(stderr, "no REF scope") {
		t.Fatalf("unscoped tag code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if stdout := mustRunCLI(t, dir, "tag", "root"); !strings.Contains(stdout, `TAG root "reviewed/1.0" `+rootID) {
		t.Fatalf("tag list stdout=%s", stdout)
	}
	if stdout := mustRunCLI(t, dir, "mv", "root", "archive/root"); !strings.Contains(stdout, "tags=1") {
		t.Fatalf("mv stdout=%s", stdout)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "root@reviewed/1.0"); code != 1 || stdout != "" || !strings.Contains(stderr, "REF not found") {
		t.Fatalf("old tag scope code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, nil, "show", "archive/root@reviewed/1.0"); code != 0 || !strings.Contains(stdout, rootID) || stderr != "" {
		t.Fatalf("moved tag scope code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func TestCLILoadRequiresSingleKnownFormatAndAbsentTarget(t *testing.T) {
	input := cliLogicalDump(t)
	for _, args := range [][]string{{"load"}, {"load", "--format", "other"}, {"load", "--format", "logical-v1", "extra"}, {"load", "--format", "logical-v1", "--format", "logical-v1"}} {
		code, stdout, _ := runCLI(t, t.TempDir(), input, args...)
		if code != 2 || stdout != "" {
			t.Fatalf("%v code=%d stdout=%s", args, code, stdout)
		}
	}
	dir := t.TempDir()
	if code, _, stderr := runCLI(t, dir, nil, "init"); code != 0 {
		t.Fatal(stderr)
	}
	if code, stdout, stderr := runCLI(t, dir, input, "load", "--format", "logical-v1"); code != 1 || stdout != "" || !strings.Contains(stderr, "already exists") {
		t.Fatalf("existing load code=%d stdout=%s stderr=%s", code, stdout, stderr)
	}
}

func mustRunCLI(t *testing.T, dir string, args ...string) string {
	t.Helper()
	code, stdout, stderr := runCLI(t, dir, nil, args...)
	if code != 0 || stderr != "" {
		t.Fatalf("%v code=%d stdout=%q stderr=%q", args, code, stdout, stderr)
	}
	return stdout
}

func mustSealCLI(t *testing.T, dir, ref string) string {
	t.Helper()
	stdout := mustRunCLI(t, dir, "seal", ref)
	fields := strings.Fields(stdout)
	if len(fields) < 4 || fields[0] != "SEALED" || fields[1] != ref || !strings.HasPrefix(fields[3], "operation=") {
		t.Fatalf("unexpected seal receipt: %q", stdout)
	}
	return fields[2]
}

type cliRevisionFixture struct {
	dir          string
	root1, root2 string
}

func newCLIRevisionFixture(t *testing.T) cliRevisionFixture {
	t.Helper()
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root-v1")
	root1 := mustSealCLI(t, dir, "root")
	mustRunCLI(t, dir, "add", "middle", "--content", "middle", "--depend-on", "root")
	mustSealCLI(t, dir, "middle")
	mustRunCLI(t, dir, "add", "leaf", "--content", "leaf", "--depend-on", "middle")
	mustSealCLI(t, dir, "leaf")
	mustRunCLI(t, dir, "add", "root", "--root", "--content", "root-v2")
	root2 := mustSealCLI(t, dir, "root")
	return cliRevisionFixture{dir: dir, root1: root1, root2: root2}
}

func TestCLIRevisionGraphDeriveStaleHistoryAndImpact(t *testing.T) {
	fixture := newCLIRevisionFixture(t)
	verifyCLIStaleAndImpact(t, fixture)
	verifyCLIDeriveAndHistory(t, fixture)
}

func verifyCLIStaleAndImpact(t *testing.T, fixture cliRevisionFixture) {
	t.Helper()
	if stdout := mustRunCLI(t, fixture.dir, "stale", "--frontier", "--refs-only", "--scan"); stdout != "middle\n" {
		t.Fatalf("stale stdout=%q", stdout)
	}
	stdout := mustRunCLI(t, fixture.dir, "impact", "--all-paths", "--max-paths", "1", "@"+fixture.root2)
	if !strings.Contains(stdout, "SOURCE "+fixture.root2) || !strings.Contains(stdout, "refs=leaf") {
		t.Fatalf("impact stdout=%q", stdout)
	}
	if code, stdout, _ := runCLI(t, fixture.dir, nil, "impact", "--max-paths", "1", "@"+fixture.root2); code != 2 || stdout != "" {
		t.Fatalf("invalid impact code=%d stdout=%q", code, stdout)
	}
}

func verifyCLIDeriveAndHistory(t *testing.T, fixture cliRevisionFixture) {
	t.Helper()
	stdout := mustRunCLI(t, fixture.dir, "derive", "preserved", "--from", "@"+fixture.root1)
	if !strings.Contains(stdout, "parent="+fixture.root1) {
		t.Fatalf("derive stdout=%q", stdout)
	}
	mustSealCLI(t, fixture.dir, "preserved")
	stdout = mustRunCLI(t, fixture.dir, "compare", "root")
	if !strings.Contains(stdout, "FROM "+fixture.root1) || !strings.Contains(stdout, "TO "+fixture.root2) {
		t.Fatalf("diff stdout=%q", stdout)
	}
	stdout = mustRunCLI(t, fixture.dir, "log", "root")
	if strings.Count(stdout, "SEAL ") != 2 {
		t.Fatalf("log stdout=%q", stdout)
	}
}
