package repository

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newFormat4Repository(t *testing.T) (string, *Repository) {
	t.Helper()
	dir := t.TempDir()
	if result, err := InitStandalone(dir); err != nil || result.Outcome != InitInitialized {
		t.Fatalf("init result=%+v err=%v", result, err)
	}
	repo, err := OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, repo
}

func sealRoot(t *testing.T, repo *Repository, ref string, content []byte) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: ref, Content: content, Root: true}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Seal(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func TestFormat4RootIdentityIsIndependentOfREF(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	first := sealRoot(t, repo, "alpha/root", []byte("same material"))
	second := sealRoot(t, repo, "beta/root", []byte("same material"))
	if !first.ID.Equal(second.ID) {
		t.Fatalf("same parentless material produced %s and %s", first.ID, second.ID)
	}
	if first.Payload.ParentRevision != nil || len(first.Payload.Links) != 0 {
		t.Fatalf("unexpected root payload: %+v", first.Payload)
	}
	config, err := os.ReadFile(filepath.Join(dir, ".sealgraph", "config"))
	if err != nil {
		t.Fatal(err)
	}
	if string(config) != "repository_format = 4\nobject_format = sha256\nref_format = manifest-v1\n" {
		t.Fatalf("config = %q", config)
	}
	show, err := repo.Show(context.Background(), "@"+first.ID.String()[:12])
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(show.REFNames, ",") != "alpha/root,beta/root" {
		t.Fatalf("aliases = %v", show.REFNames)
	}
}

func TestOrdinaryCandidateSeparatesParentAndPublicationExpectation(t *testing.T) {
	_, repo := newFormat4Repository(t)
	first := sealRoot(t, repo, "root", []byte("v1"))
	candidate, err := repo.Add(context.Background(), AddOptions{REF: "root", Content: []byte("v2"), Root: true})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ParentRevision == nil || !candidate.ParentRevision.Equal(first.ID) || candidate.ExpectedREFHead == nil || !candidate.ExpectedREFHead.Equal(first.ID) {
		t.Fatalf("candidate topology/publication state = %+v", candidate)
	}
	inspection, err := repo.InspectCandidate(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}
	if inspection.ExpectedHeadState != CandidateExpectedCurrent {
		t.Fatalf("expected-head state = %s", inspection.ExpectedHeadState)
	}
	second, err := repo.Seal(context.Background(), "root")
	if err != nil {
		t.Fatal(err)
	}
	if second.Payload.ParentRevision == nil || !second.Payload.ParentRevision.Equal(first.ID) {
		t.Fatalf("published parent revision = %v", second.Payload.ParentRevision)
	}
}

func TestExistingREFRejectsAlternateParentOverride(t *testing.T) {
	_, repo := newFormat4Repository(t)
	first := sealRoot(t, repo, "line", []byte("v1"))
	other := sealRoot(t, repo, "other", []byte("other"))
	candidate, err := repo.Add(context.Background(), AddOptions{REF: "line", Content: []byte("v2"), Root: true})
	if err != nil {
		t.Fatal(err)
	}
	candidate.ParentRevision = &other.ID
	if err := repo.candidates.Save(candidate); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(context.Background(), "line"); err == nil || !strings.Contains(err.Error(), "alternate-parent override") {
		t.Fatalf("alternate parent error = %v", err)
	}
	resolved, err := repo.ResolveSelector(context.Background(), "line")
	if err != nil || !resolved.ID.Equal(first.ID) {
		t.Fatalf("failed override moved REF: %s err=%v", resolved.ID, err)
	}
}

func TestDraftCauseLinkStoresOnlyExactSeal(t *testing.T) {
	_, repo := newFormat4Repository(t)
	root := sealRoot(t, repo, "root", []byte("root"))
	candidate, err := repo.Add(context.Background(), AddOptions{
		REF: "child", Content: []byte("child"), Draft: true,
		Dependencies: []Dependency{{Selector: "root", Message: "basis"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.Links) != 1 || !candidate.Links[0].TargetSeal.Equal(root.ID) {
		t.Fatalf("candidate links = %+v", candidate.Links)
	}
	sealed, err := repo.Seal(context.Background(), "child")
	if err != nil {
		t.Fatal(err)
	}
	if !sealed.Payload.Draft || len(sealed.Payload.Links) != 1 || !sealed.Payload.Links[0].TargetSeal.Equal(root.ID) {
		t.Fatalf("sealed child = %+v", sealed.Payload)
	}
}

func TestNormalNonRootPublicationUsesActiveLeafAdmission(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealRoot(t, repo, "root", []byte("root"))
	if _, err := repo.Add(context.Background(), AddOptions{
		REF: "child", Content: []byte("child"), Dependencies: []Dependency{{Selector: "root"}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(context.Background(), "child"); err != nil {
		t.Fatalf("normal child seal error = %v", err)
	}
	if _, err := repo.candidates.Load("child"); !errors.Is(err, ErrCandidateNotFound) {
		t.Fatalf("published candidate was not cleared: %v", err)
	}
}

func TestFormat4SelectorsEnforceScopeAncestry(t *testing.T) {
	_, repo := newFormat4Repository(t)
	first := sealRoot(t, repo, "line", []byte("v1"))
	if _, err := repo.Add(context.Background(), AddOptions{REF: "line", Content: []byte("v2"), Root: true}); err != nil {
		t.Fatal(err)
	}
	second, err := repo.Seal(context.Background(), "line")
	if err != nil {
		t.Fatal(err)
	}
	other := sealRoot(t, repo, "other", []byte("other"))
	for _, selector := range []string{"line", "@" + first.ID.String()[:12], "line@" + first.ID.String()[:12]} {
		if _, err := repo.ResolveSelector(context.Background(), selector); err != nil {
			t.Fatalf("selector %s: %v", selector, err)
		}
	}
	if resolved, err := repo.ResolveSelector(context.Background(), "line"); err != nil || !resolved.ID.Equal(second.ID) {
		t.Fatalf("current selector = %v err=%v", resolved.ID, err)
	}
	if _, err := repo.ResolveSelector(context.Background(), "line@"+other.ID.String()[:12]); err == nil || !strings.Contains(err.Error(), "outside the current parent ancestry") {
		t.Fatalf("scoped sibling selector error = %v", err)
	}
	if _, err := repo.CreateTag(context.Background(), "line@"+first.ID.String()[:12], "release"); err != nil {
		t.Fatalf("create historical tag: %v", err)
	}
	if resolved, err := repo.ResolveSelector(context.Background(), "line@release"); err != nil || !resolved.ID.Equal(first.ID) {
		t.Fatalf("tag selector = %v err=%v", resolved.ID, err)
	}
}

func TestFormat4RuntimeRejectsFormat3Repository(t *testing.T) {
	dir, _ := newFormat4Repository(t)
	config := filepath.Join(dir, ".sealgraph", "config")
	if err := os.WriteFile(config, []byte("repository_format = 3\nobject_format = sha256\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := OpenStandalone(dir); err == nil || !strings.Contains(err.Error(), "unsupported or malformed config") {
		t.Fatalf("format-3 open error = %v", err)
	}
}

func TestPrefixREFsAndCandidatesCoexistAndMoveRejectsCandidate(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealRoot(t, repo, "design", []byte("design"))
	sealRoot(t, repo, "design/api", []byte("api"))
	if refs, err := repo.refs.List(context.Background()); err != nil || len(refs) != 2 || refs[0] != "design" || refs[1] != "design/api" {
		t.Fatalf("prefix REFs = %v err=%v", refs, err)
	}
	if _, err := repo.Add(context.Background(), AddOptions{REF: "design", Content: []byte("next design"), Root: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(context.Background(), AddOptions{REF: "design/api", Content: []byte("next api"), Root: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.MoveREF(context.Background(), "design", "renamed"); err == nil || !strings.Contains(err.Error(), "candidate design blocks") {
		t.Fatalf("move with candidate error = %v", err)
	}
	if _, err := repo.ResolveSelector(context.Background(), "design"); err != nil {
		t.Fatalf("blocked move changed source: %v", err)
	}
}

func TestParseSelectorGrammar(t *testing.T) {
	tests := map[string]SelectorKind{"design/api": SelectorCurrentREF, "@abcd": SelectorGlobalSeal, "design/api@abcd": SelectorScopedSeal, "design/api@tag": SelectorScopedTag}
	for input, want := range tests {
		selector, err := ParseSelector(input)
		if err != nil || selector.Kind != want {
			t.Fatalf("ParseSelector(%q) = %+v, %v", input, selector, err)
		}
	}
	for _, input := range []string{"abcd@", "@tag", "design@@abcd", ""} {
		if _, err := ParseSelector(input); err == nil {
			t.Fatalf("invalid selector %q was accepted", input)
		}
	}
}
