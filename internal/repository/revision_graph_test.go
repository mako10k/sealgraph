package repository

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/store"
)

func sealDependent(t *testing.T, repo *Repository, ref, content, upstream string) SealResult {
	t.Helper()
	if _, err := repo.Add(context.Background(), AddOptions{
		REF: ref, Content: []byte(content), Dependencies: []Dependency{{Selector: upstream}},
	}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Seal(context.Background(), ref)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func statusesByREF(statuses []RefStatus) map[string]RefStatus {
	result := make(map[string]RefStatus, len(statuses))
	for _, status := range statuses {
		result[status.REF] = status
	}
	return result
}

type staleFixture struct {
	repo           *Repository
	root1, root2   SealResult
	middle1, leaf1 SealResult
}

func newStaleFixture(t *testing.T) staleFixture {
	t.Helper()
	_, repo := newFormat4Repository(t)
	root1 := sealRoot(t, repo, "root", []byte("root-v1"))
	middle1 := sealDependent(t, repo, "middle", "middle-v1", "root")
	leaf1 := sealDependent(t, repo, "leaf", "leaf-v1", "middle")
	root2 := sealRoot(t, repo, "root", []byte("root-v2"))
	return staleFixture{repo: repo, root1: root1, root2: root2, middle1: middle1, leaf1: leaf1}
}

func TestActiveRevisionStaleFrontierImpactAndAdmission(t *testing.T) {
	fixture := newStaleFixture(t)
	verifyStaleStatuses(t, fixture)
	verifyStaleSetAndFrontier(t, fixture)
	verifyRevisionAwareImpact(t, fixture)
	verifyActiveLeafAdmission(t, fixture)
}

func verifyStaleStatuses(t *testing.T, fixture staleFixture) {
	t.Helper()
	statuses, err := fixture.repo.Status(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	byREF := statusesByREF(statuses)
	if len(byREF["middle"].StaleDirect) != 1 || !byREF["middle"].StaleDirect[0].Equal(fixture.root1.ID) {
		t.Fatalf("middle status = %+v", byREF["middle"])
	}
	if len(byREF["leaf"].StaleTransitive) != 1 || !byREF["leaf"].StaleTransitive[0][1].Equal(fixture.root1.ID) {
		t.Fatalf("leaf status = %+v", byREF["leaf"])
	}
	if labels := strings.Join(byREF["root"].Labels(), ","); labels != "CLEAN" {
		t.Fatalf("root labels = %s", labels)
	}
}

func verifyStaleSetAndFrontier(t *testing.T, fixture staleFixture) {
	t.Helper()
	stale, warning, err := fixture.repo.Stale(context.Background(), false, true)
	if err != nil || warning != "" || len(stale) != 2 || stale[0].REF != "leaf" || stale[1].REF != "middle" {
		t.Fatalf("stale=%+v warning=%q err=%v", stale, warning, err)
	}
	frontier, _, err := fixture.repo.Stale(context.Background(), true, false)
	if err != nil || len(frontier) != 1 || frontier[0].REF != "middle" {
		t.Fatalf("frontier=%+v err=%v", frontier, err)
	}
}

func verifyRevisionAwareImpact(t *testing.T, fixture staleFixture) {
	t.Helper()
	source, impacts, err := fixture.repo.Impact(context.Background(), "@"+fixture.root2.ID.String(), false, 100)
	if err != nil || !source.Equal(fixture.root2.ID) || len(impacts) != 2 {
		t.Fatalf("source=%s impacts=%+v err=%v", source, impacts, err)
	}
	impactByREF := make(map[string]struct {
		head domain.ObjectID
		path []domain.ObjectID
	})
	for _, impact := range impacts {
		impactByREF[impact.REFs[0]] = struct {
			head domain.ObjectID
			path []domain.ObjectID
		}{impact.Head, impact.Paths[0]}
	}
	if got := impactByREF["middle"]; !got.head.Equal(fixture.middle1.ID) || len(got.path) != 2 || !got.path[1].Equal(fixture.root1.ID) {
		t.Fatalf("middle impact = %+v", got)
	}
	if got := impactByREF["leaf"]; !got.head.Equal(fixture.leaf1.ID) || len(got.path) != 3 || !got.path[2].Equal(fixture.root1.ID) {
		t.Fatalf("leaf impact = %+v", got)
	}
}

func verifyActiveLeafAdmission(t *testing.T, fixture staleFixture) {
	t.Helper()
	if _, err := fixture.repo.Add(context.Background(), AddOptions{REF: "middle", Content: []byte("blocked")}); err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.repo.Seal(context.Background(), "middle"); err == nil || !strings.Contains(err.Error(), "STALE_REVISION") {
		t.Fatalf("stale Cause admission error = %v", err)
	}
	if _, err := fixture.repo.candidates.Load("middle"); err != nil {
		t.Fatalf("rejected publication did not retain candidate: %v", err)
	}
}

func TestDeriveAndAddParentCreateExplicitSiblingRevisions(t *testing.T) {
	_, repo := newFormat4Repository(t)
	root1 := sealRoot(t, repo, "root", []byte("v1"))
	root2 := sealRoot(t, repo, "root", []byte("v2"))

	derived, err := repo.Derive(context.Background(), "preserved", "@"+root1.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	if derived.ParentRevision == nil || !derived.ParentRevision.Equal(root1.ID) || !derived.Content.ID.Equal(root1.Payload.Content.ID) || derived.ExpectedREFHead != nil {
		t.Fatalf("derived candidate = %+v", derived)
	}
	preserved, err := repo.Seal(context.Background(), "preserved")
	if err != nil {
		t.Fatal(err)
	}
	if preserved.Payload.ParentRevision == nil || !preserved.Payload.ParentRevision.Equal(root1.ID) {
		t.Fatalf("preserved seal = %+v", preserved.Payload)
	}

	custom, err := repo.Add(context.Background(), AddOptions{REF: "custom", Parent: "@" + root1.ID.String(), Content: []byte("custom"), Root: true})
	if err != nil {
		t.Fatal(err)
	}
	if custom.ParentRevision == nil || !custom.ParentRevision.Equal(root1.ID) || custom.ExpectedREFHead != nil {
		t.Fatalf("custom candidate = %+v", custom)
	}
	customSeal, err := repo.Seal(context.Background(), "custom")
	if err != nil {
		t.Fatal(err)
	}
	if customSeal.ID.Equal(preserved.ID) || customSeal.ID.Equal(root2.ID) {
		t.Fatal("distinct sibling material collapsed unexpectedly")
	}
	if _, err := repo.Derive(context.Background(), "custom", "@"+root1.ID.String()); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("existing destination derive error = %v", err)
	}
}

func TestCurrentAliasToActiveNonLeafIsSelfStaleWhileSiblingsStayLeaves(t *testing.T) {
	_, repo := newFormat4Repository(t)
	root1 := sealRoot(t, repo, "root", []byte("v1"))
	sealRoot(t, repo, "root", []byte("v2"))
	if _, err := repo.Derive(context.Background(), "sibling", "@"+root1.ID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(context.Background(), "sibling"); err != nil {
		t.Fatal(err)
	}
	if err := repo.refs.Update(context.Background(), "old-alias", nil, &root1.ID); err != nil {
		t.Fatal(err)
	}
	statuses, err := repo.Status(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	byREF := statusesByREF(statuses)
	if !byREF["old-alias"].StaleSelf || strings.Join(byREF["root"].Labels(), ",") != "CLEAN" || strings.Join(byREF["sibling"].Labels(), ",") != "CLEAN" {
		t.Fatalf("statuses = %+v", byREF)
	}
}

func TestStaleCacheRebuildsAfterInvalidCache(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealRoot(t, repo, "root", []byte("v1"))
	sealDependent(t, repo, "child", "child", "root")
	sealRoot(t, repo, "root", []byte("v2"))

	first, warning, err := repo.Stale(context.Background(), false, true)
	if err != nil || warning != "" {
		t.Fatalf("initial scan warning=%q err=%v", warning, err)
	}
	if err := os.WriteFile(repo.revisionCachePath(), []byte("not-json\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, warning, err := repo.Stale(context.Background(), false, false)
	if err != nil || warning == "" || len(first) != len(second) || first[0].REF != second[0].REF {
		t.Fatalf("rebuilt stale=%+v warning=%q err=%v", second, warning, err)
	}
	if data, err := os.ReadFile(repo.revisionCachePath()); err != nil || !strings.Contains(string(data), revisionCacheSchema) {
		t.Fatalf("rebuilt cache data=%q err=%v", data, err)
	}
}

func TestStaleTreatsUnsafeCachePathAsWarningWithoutFollowingIt(t *testing.T) {
	dir, repo := newFormat4Repository(t)
	sealRoot(t, repo, "root", []byte("root"))
	external := t.TempDir()
	cacheDir := filepath.Join(dir, ".sealgraph", "cache")
	if err := os.Symlink(external, cacheDir); err != nil {
		t.Fatal(err)
	}
	statuses, warning, err := repo.Stale(context.Background(), false, true)
	if err != nil || len(statuses) != 0 || !strings.Contains(warning, "not a real directory") {
		t.Fatalf("statuses=%+v warning=%q err=%v", statuses, warning, err)
	}
	entries, err := os.ReadDir(external)
	if err != nil || len(entries) != 0 {
		t.Fatalf("external cache target entries=%v err=%v", entries, err)
	}
}

func TestImpactBoundsAllPathsWithoutHidingMembershipOrAliases(t *testing.T) {
	_, repo := newFormat4Repository(t)
	root := sealRoot(t, repo, "root", []byte("root"))
	left := sealDependent(t, repo, "left", "left", "root")
	right := sealDependent(t, repo, "right", "right", "root")
	if _, err := repo.Add(context.Background(), AddOptions{
		REF: "downstream", Content: []byte("downstream"),
		Dependencies: []Dependency{{Selector: "left"}, {Selector: "right"}},
	}); err != nil {
		t.Fatal(err)
	}
	downstream, err := repo.Seal(context.Background(), "downstream")
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.refs.Update(context.Background(), "downstream-alias", nil, &downstream.ID); err != nil {
		t.Fatal(err)
	}
	_, impacts, err := repo.Impact(context.Background(), "@"+root.ID.String(), true, 1)
	if err != nil {
		t.Fatal(err)
	}
	var found *graph.Impact
	for i := range impacts {
		if impacts[i].Head.Equal(downstream.ID) {
			found = &impacts[i]
		}
	}
	if found == nil || !found.Truncated || len(found.Paths) != 1 || len(found.REFs) != 2 || found.REFs[0] != "downstream" || found.REFs[1] != "downstream-alias" {
		t.Fatalf("downstream impact = %+v", found)
	}
	wantedFirst := left.ID
	if right.ID.String() < left.ID.String() {
		wantedFirst = right.ID
	}
	if len(found.Paths[0]) != 3 || !found.Paths[0][1].Equal(wantedFirst) {
		t.Fatalf("deterministic first path = %v", found.Paths[0])
	}
}

type changingRefStore struct {
	store.RefStore
	lists int
}

func (store *changingRefStore) List(ctx context.Context) ([]string, error) {
	refs, err := store.RefStore.List(ctx)
	store.lists++
	if err == nil && store.lists > 1 {
		refs = append(refs, "concurrent")
	}
	return refs, err
}

func TestStatusRejectsChangedCompleteHeadObservation(t *testing.T) {
	_, repo := newFormat4Repository(t)
	sealRoot(t, repo, "root", []byte("root"))
	repo.refs = &changingRefStore{RefStore: repo.refs}
	if _, err := repo.Status(context.Background(), ""); err == nil || !strings.Contains(err.Error(), "REF heads changed") {
		t.Fatalf("status observation error = %v", err)
	}
}
