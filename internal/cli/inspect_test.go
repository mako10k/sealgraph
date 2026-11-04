package cli

// R: verify that daily inspection separates observed axes and remains read-only.

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/historicalreview"
	"github.com/mako10k/sealgraph/internal/repository"
)

func inspectionDocument(t *testing.T, dir string, args ...string) inspectDocument {
	t.Helper()
	all := append([]string{"inspect"}, args...)
	all = append(all, "--format", "json")
	code, out, stderr := runCLI(t, dir, nil, all...)
	if code != 0 || stderr != "" {
		t.Fatalf("inspect %v: %d %s", args, code, stderr)
	}
	var doc inspectDocument
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func TestInspectSeparatesWorkfileFromProvenanceAndPreservesBytes(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	file := filepath.Join(dir, "spec.txt")
	if err := os.WriteFile(file, []byte("saved"), 0600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "add", "spec", "--root", "--clear-cause-links", "--content-file", "spec.txt", "--bind-source")
	mustRunCLI(t, dir, "seal", "spec")
	before := mustRunCLI(t, dir, "show", "spec", "--format", "json")
	if err := os.WriteFile(file, []byte("changed"), 0600); err != nil {
		t.Fatal(err)
	}
	doc := inspectionDocument(t, dir, "spec")
	if len(doc.Entries) != 1 {
		t.Fatal(doc)
	}
	entry := doc.Entries[0]
	if entry.Workfile.State != "WORKFILE_DIFFERS_FROM_HEAD" || entry.Status.Stale.Self || len(entry.Status.Stale.Direct) != 0 || entry.TraceState != "UNSUPPORTED_FORMAT" {
		t.Fatalf("axes mixed: %+v", entry)
	}
	if entry.Reviews[0].State != "NONE" || entry.Workfile.BlobID == nil {
		t.Fatal(entry)
	}
	if after := mustRunCLI(t, dir, "show", "spec", "--format", "json"); after != before {
		t.Fatal("inspection changed sealed state")
	}
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	if inspectionDocument(t, dir, "spec").Entries[0].Workfile.State != "SOURCE_MISSING" {
		t.Fatal("missing was hidden")
	}
}

func TestInspectTraceKeepsCandidateHeadAndMissingSourceDistinct(t *testing.T) {
	dir := traceCLIFixture(t)
	writeTraceFixture(t, dir)
	mustRunCLI(t, dir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	if code, _, stderr := runCLI(t, dir, nil, "trace", "set", "root", "--recipe", "recipe.json"); code != 0 {
		t.Fatal(stderr)
	}
	mustRunCLI(t, dir, "seal", "root")
	mustRunCLI(t, dir, "add", "root", "--content", "XYZ-UV")
	doc := inspectionDocument(t, dir, "root")
	if len(doc.Entries[0].Trace) != 2 || doc.Entries[0].Trace[1].Baseline.Kind != "head" {
		t.Fatal(doc)
	}
	for _, local := range doc.Entries[0].Trace {
		if !local.HasUnresolved {
			t.Fatal("unbound source was an all-clear")
		}
	}
	mustRunCLI(t, dir, "trace", "source", "bind", "key-A", "--file", "a.txt")
	mustRunCLI(t, dir, "trace", "source", "bind", "key-B", "--file", "b.txt")
	doc = inspectionDocument(t, dir, "root")
	if doc.Entries[0].Trace[0].HasUnresolved || doc.Entries[0].Trace[1].HasDifference {
		t.Fatal(doc)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("no matching quote"), 0600); err != nil {
		t.Fatal(err)
	}
	doc = inspectionDocument(t, dir, "root")
	if !doc.Entries[0].Trace[0].HasDifference || len(doc.Entries[0].Status.Stale.Direct) > 0 {
		t.Fatal("Trace difference changed stale")
	}
}

func TestInspectHistoricalClaimIsSeparateFromApplicability(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "p", "--root", "--clear-cause-links", "--content", "premise")
	mustRunCLI(t, dir, "seal", "p")
	mustRunCLI(t, dir, "add", "d", "--non-root", "--content", "dependent", "--target", "p", "--no-previous")
	var facts repository.HistoricalReviewPreparation
	if err := json.Unmarshal([]byte(mustRunCLI(t, dir, "review", "prepare", "d", "--format", "json")), &facts); err != nil {
		t.Fatal(err)
	}
	record := historicalreview.Record{Schema: historicalreview.Schema, REF: facts.REF, SubjectSeal: facts.SubjectSeal, OtherHeads: facts.OtherHeads, Entries: []historicalreview.Entry{}}
	for _, target := range facts.Targets {
		record.Entries = append(record.Entries, historicalreview.Entry{TargetSeal: target.TargetSeal, ComparedSeals: target.ComparedSeals, Decision: "retain", Difference: "same selected bytes", Impact: "retained scope", Reason: "read the original"})
	}
	data, err := historicalreview.Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	code, _, stderr := runCLI(t, dir, data, "review", "set", "d", "--file", "-", "--format", "json")
	if code != 0 {
		t.Fatal(stderr)
	}
	repo, openErr := repository.OpenStandalone(dir)
	if openErr != nil {
		t.Fatal(openErr)
	}
	selected, selectErr := repo.InspectCandidate(context.Background(), "d")
	if selectErr != nil {
		t.Fatal(selectErr)
	}
	if _, mixedErr := inspectReviewFor(context.Background(), repo, "d", true, domain.ComputeNativeBlobID([]byte("different subject")), selected.Candidate.Attachments); mixedErr == nil {
		t.Fatal("mixed review subject accepted")
	}
	doc := inspectionDocument(t, dir, "d")
	review := doc.Entries[0].Reviews[0]
	if review.State != "RECORDED" || review.Applicable == nil || !*review.Applicable {
		t.Fatal(review)
	}
	mustRunCLI(t, dir, "seal", "d")
	mustRunCLI(t, dir, "add", "unrelated", "--root", "--clear-cause-links", "--content", "other")
	mustRunCLI(t, dir, "seal", "unrelated")
	review = inspectionDocument(t, dir, "d").Entries[0].Reviews[0]
	if review.State != "RECORDED" || review.Applicable == nil || *review.Applicable {
		t.Fatal("record disappearance or stale applicability", review)
	}
}

func TestInspectOmissionsAndNoBindingAreExplicit(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	mustRunCLI(t, dir, "add", "a", "--root", "--clear-cause-links", "--content", "a")
	mustRunCLI(t, dir, "add", "b", "--root", "--clear-cause-links", "--content", "b")
	doc := inspectionDocument(t, dir, "--max-refs", "1")
	if doc.Complete || doc.Omitted != 1 || doc.Entries[0].Workfile.State != "NOT_BOUND" {
		t.Fatal(doc)
	}
	code, human, stderr := runCLI(t, dir, nil, "inspect", "--format", "human")
	if code != 0 || stderr != "" || !strings.Contains(human, "On-demand") && !strings.Contains(human, "on demand") {
		t.Fatal(human, stderr)
	}
	for _, args := range [][]string{{"inspect", "--max-refs", "0"}, {"inspect", "--max-refs", "10001"}, {"inspect", "a", "b"}} {
		code, _, _ := runCLI(t, dir, nil, args...)
		if code != 2 {
			t.Fatal(args, code)
		}
	}
}

func TestInspectRevalidationRejectsSameRelationButDifferentFile(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	if err := os.WriteFile(filepath.Join(dir, "p.txt"), []byte("first"), 0600); err != nil {
		t.Fatal(err)
	}
	mustRunCLI(t, dir, "add", "p", "--root", "--clear-cause-links", "--content-file", "p.txt", "--bind-source")
	repo, err := repository.OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	before, err := repo.Status(ctx, "p")
	if err != nil {
		t.Fatal(err)
	}
	entry, err := inspectOne(ctx, repo, before[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "p.txt"), []byte("second"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := revalidateInspect(ctx, repo, "p", before, []inspectEntry{entry}); err == nil {
		t.Fatal("changed workfile accepted")
	}
}

func TestWorkflowHelpIsRepositoryIndependentAndQuotesREF(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "help", "workflows")
	if _, err := os.Stat(filepath.Join(dir, ".sealgraph")); !os.IsNotExist(err) {
		t.Fatal("help created state")
	}
	want := "'a'\"'\"'b'"
	if got := shellArgument("a'b"); got != want {
		t.Fatal(got)
	}
	for _, args := range [][]string{{"help", "inspect"}, {"help", "trace", "read"}, {"help", "compare"}} {
		if code, _, _ := runCLI(t, dir, nil, args...); code != 0 {
			t.Fatal(args)
		}
	}
}

func TestOperatorHumanOutputDeliveryFailure(t *testing.T) {
	dir := t.TempDir()
	mustRunCLI(t, dir, "init")
	var stderr bytes.Buffer
	if code := runStandaloneAtWithInput(dir, []string{"inspect", "--format", "human"}, nil, errorWriter{}, &stderr); code != 1 || !strings.Contains(stderr.String(), "write inspection output") {
		t.Fatal(code, stderr.String())
	}
	traceDir := traceCLIFixture(t)
	writeTraceFixture(t, traceDir)
	mustRunCLI(t, traceDir, "add", "root", "--root", "--clear-cause-links", "--content", "XYZ-UV")
	if code, _, err := runCLI(t, traceDir, nil, "trace", "set", "root", "--recipe", "recipe.json"); code != 0 {
		t.Fatal(err)
	}
	stderr.Reset()
	if code := runStandaloneAtWithInput(traceDir, []string{"trace", "read", "--ref", "root", "--baseline", "candidate", "--run-index", "0", "--view", "snapshot", "--format", "human"}, nil, errorWriter{}, &stderr); code != 1 {
		t.Fatal(code, stderr.String())
	}
}
