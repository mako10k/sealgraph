package repository

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func newV5TestRepository(t *testing.T) *Repository {
	t.Helper()
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func newTestRepositoryWithDir(t *testing.T) (string, *Repository) {
	t.Helper()
	dir := t.TempDir()
	if _, err := InitStandalone(dir); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir, repo
}

func sealRootV5(t *testing.T, repo *Repository, ref, content string) SealResult {
	return addAndSealRoot(t, repo, ref, content)
}

func cause(selector string, previous ...string) *CauseInput {
	return &CauseInput{Target: selector, Previous: previous}
}

func TestScopedSealPrefixIgnoresMatchingObjectsOutsideRevisionClosure(t *testing.T) {
	head := domain.ObjectID{Hex: "abcd" + strings.Repeat("1", 60)}
	unrelated := domain.ObjectID{Hex: "abcd" + strings.Repeat("2", 60)}
	graph := &observedGraph{nodes: map[string]domainv5.ResolvedSeal{head.String(): {}, unrelated.String(): {}}, revisions: map[string][]domain.ObjectID{}}
	resolved, err := resolveScopedSealPrefix(graph, head, "abcd")
	if err != nil || !resolved.Equal(head) {
		t.Fatalf("scoped resolution=%s err=%v", resolved, err)
	}
	graph.revisions[head.String()] = []domain.ObjectID{unrelated}
	if _, err := resolveScopedSealPrefix(graph, head, "abcd"); err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("closure ambiguity error=%v", err)
	}
}

func TestScopedTagSelectorUsesCapturedManifest(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	first := sealRootV5(t, repo, "premise", "v1")
	if _, err := repo.CreateTag(ctx, "premise", "snapshot"); err != nil {
		t.Fatal(err)
	}
	second := sealRootV5(t, repo, "premise", "v2")
	observation, graph, err := repo.buildObservation(ctx, "tag observation test")
	if err != nil {
		t.Fatal(err)
	}
	captured := observation.manifests["premise"]
	live := bytes.Replace(captured, []byte(first.ID.String()), []byte(second.ID.String()), 1)
	if bytes.Equal(captured, live) {
		t.Fatal("test did not change captured tag target")
	}
	refs, err := repo.recoveryRefs()
	if err != nil {
		t.Fatal(err)
	}
	if err := refs.ReplaceExact(ctx, "premise", captured, live); err != nil {
		t.Fatal(err)
	}
	resolved, err := repo.resolveSelectorObserved(ctx, "premise@snapshot", &observation, graph)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.ID.Equal(first.ID) {
		t.Fatalf("scoped tag resolved live target %s instead of captured target %s", resolved.ID, first.ID)
	}
}

func TestFormat5RootPublicationUsesTypedBlobs(t *testing.T) {
	repo := newV5TestRepository(t)
	result := sealRootV5(t, repo, "premise", "root bytes")
	loaded, err := repo.LoadSeal(context.Background(), result.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Seal.Schema != domainv5.SealSchema || loaded.Material.Schema != domainv5.MaterialSchema || loaded.Provenance.Schema != domainv5.ProvenanceSchema {
		t.Fatalf("typed schemas = %+v", loaded)
	}
	if !loaded.Provenance.Root || len(loaded.Provenance.CauseLinks) != 0 {
		t.Fatalf("root provenance = %+v", loaded.Provenance)
	}
	if loaded.ContentBytes != len("root bytes") {
		t.Fatalf("content bytes = %d", loaded.ContentBytes)
	}
	if loaded.Seal.Material.Equal(loaded.Seal.Provenance) || loaded.ID.Equal(loaded.Seal.Material) {
		t.Fatal("typed IDs unexpectedly collapsed")
	}
}

func TestCandidatePublicationBaselineIsNotRevisionParent(t *testing.T) {
	repo := newV5TestRepository(t)
	first := sealRootV5(t, repo, "premise", "v1")
	candidate, err := repo.Add(context.Background(), AddOptions{REF: "premise", Content: []byte("v2"), Root: true, RootSet: true, ClearCauseLinks: true})
	if err != nil {
		t.Fatal(err)
	}
	if candidate.ExpectedREFHead == nil || !candidate.ExpectedREFHead.Equal(first.ID) {
		t.Fatalf("expected head = %v", candidate.ExpectedREFHead)
	}
	second, err := repo.Seal(context.Background(), "premise")
	if err != nil {
		t.Fatal(err)
	}
	observation, graph, err := repo.buildObservation(context.Background(), "test")
	if err != nil {
		t.Fatal(err)
	}
	if graph.revisionReachable(second.ID, first.ID) {
		t.Fatal("publication baseline became an intrinsic revision edge")
	}
	if err := repo.revalidateHeads(context.Background(), observation, "test"); err != nil {
		t.Fatal(err)
	}
}

func TestUnchangedProspectiveSealIsRejectedBeforePublication(t *testing.T) {
	repo := newV5TestRepository(t)
	head := sealRootV5(t, repo, "premise", "same")
	if _, err := repo.Add(context.Background(), AddOptions{REF: "premise", Content: []byte("same")}); err != nil {
		t.Fatal(err)
	}
	beforeObjects, err := repo.objects.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	beforeRecovery, err := repo.recovery.List()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(context.Background(), "premise"); err == nil || !strings.Contains(err.Error(), "SEAL_ID_UNCHANGED") {
		t.Fatalf("seal error=%v", err)
	}
	current, err := repo.refs.Resolve(context.Background(), "premise")
	if err != nil || !current.Equal(head.ID) {
		t.Fatalf("current=%s err=%v", current, err)
	}
	afterObjects, _ := repo.objects.List(context.Background())
	afterRecovery, _ := repo.recovery.List()
	if len(afterObjects) != len(beforeObjects) || len(afterRecovery) != len(beforeRecovery) {
		t.Fatalf("unchanged seal mutated objects/recovery: objects %d->%d recovery %d->%d", len(beforeObjects), len(afterObjects), len(beforeRecovery), len(afterRecovery))
	}
	if _, err := repo.candidates.Load("premise"); err != nil {
		t.Fatalf("unchanged Candidate was not retained: %v", err)
	}
}

func TestCauseLinkWholeRecordAndUnlinkInvariant(t *testing.T) {
	repo := newV5TestRepository(t)
	previous := sealRootV5(t, repo, "premise", "v1")
	sealRootV5(t, repo, "premise", "v2")
	candidate, err := repo.Add(context.Background(), AddOptions{REF: "design", Content: []byte("d1"), RootSet: true, Cause: &CauseInput{Target: "premise", Messages: []string{"b", "a"}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.CauseLinks) != 1 || len(candidate.CauseLinks[0].Messages) != 2 || candidate.CauseLinks[0].Messages[0] != "a" {
		t.Fatalf("Cause Link = %+v", candidate.CauseLinks)
	}
	candidate, err = repo.Link(context.Background(), "design", CauseInput{Target: "premise", Previous: []string{"@" + previous.ID.String()}, Messages: []string{"replacement"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(candidate.CauseLinks) != 1 || len(candidate.CauseLinks[0].Messages) != 1 || candidate.CauseLinks[0].Messages[0] != "replacement" {
		t.Fatalf("whole-record replacement = %+v", candidate.CauseLinks)
	}
	if _, err = repo.Unlink(context.Background(), "design", "premise"); err == nil || !strings.Contains(err.Error(), "without a Cause Link") {
		t.Fatalf("unlink error = %v", err)
	}
}

func TestFormat4RepositoryFailsWithMigrationGuideBeforeMutation(t *testing.T) {
	dir := t.TempDir()
	root := filepath.Join(dir, ".sealgraph")
	for _, relative := range []string{"objects", filepath.Join("refs", "seals"), "index", "locks"} {
		if err := os.MkdirAll(filepath.Join(root, relative), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "config"), []byte(format4ConfigBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(root, "config"))
	if err != nil {
		t.Fatal(err)
	}
	_, err = OpenStandalone(dir)
	if err == nil || !strings.Contains(err.Error(), "FORMAT4_REQUIRES_MIGRATION") || !strings.Contains(err.Error(), "migrate extract --source-format 4 --format universal-blob-v1") || !strings.Contains(err.Error(), "load --format universal-blob-v1") {
		t.Fatalf("open error = %v", err)
	}
	after, err := os.ReadFile(filepath.Join(root, "config"))
	if err != nil {
		t.Fatal(err)
	}
	if !errors.Is(nil, nil) || string(before) != string(after) {
		t.Fatal("format-4 config changed")
	}
}

func TestProspectiveSelfRevisionCycleIsRejectedBeforeCandidateMutation(t *testing.T) {
	repo := newV5TestRepository(t)
	target := sealRootV5(t, repo, "premise", "v1")
	if _, err := repo.Add(context.Background(), AddOptions{REF: "design", Content: []byte("d"), RootSet: true, Draft: true, DraftSet: true, Cause: cause("premise")}); err != nil {
		t.Fatal(err)
	}
	before, err := repo.candidates.LoadSnapshot("design")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Link(context.Background(), "design", *cause("premise", "premise")); err == nil || !strings.Contains(err.Error(), "self-edge") {
		t.Fatalf("link error = %v target=%s", err, target.ID)
	}
	after, err := repo.candidates.LoadSnapshot("design")
	if err != nil {
		t.Fatal(err)
	}
	if string(before.Bytes) != string(after.Bytes) {
		t.Fatal("rejected revision self-edge changed Candidate bytes")
	}
}
