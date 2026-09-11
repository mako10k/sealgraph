package repository

import (
	"context"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func TestFormat5RevisionAssertionDrivesStaleLogAndImpact(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	root1 := sealRootV5(t, repo, "premise", "v1")
	if _, err := repo.Add(ctx, AddOptions{REF: "design", Content: []byte("d1"), RootSet: true, Cause: &CauseInput{Target: "premise", Messages: []string{"basis"}}}); err != nil {
		t.Fatal(err)
	}
	design1, err := repo.Seal(ctx, "design")
	if err != nil {
		t.Fatal(err)
	}
	root2 := sealRootV5(t, repo, "premise", "v2")
	stale, _, err := repo.Stale(ctx, false, false)
	if err != nil {
		t.Fatal(err)
	}
	requireDirectStale(t, stale, "design", root1.ID)
	if _, err := repo.Add(ctx, AddOptions{REF: "design", Content: []byte("d2"), Cause: &CauseInput{Target: "premise", Previous: []string{"@" + root1.ID.String()}, Messages: []string{"revised basis"}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Unlink(ctx, "design", "@"+root1.ID.String()); err != nil {
		t.Fatal(err)
	}
	design2, err := repo.Seal(ctx, "design")
	if err != nil {
		t.Fatal(err)
	}
	stale, _, err = repo.Stale(ctx, false, false)
	if err != nil {
		t.Fatal(err)
	}
	requireNoStale(t, stale)
	log, err := repo.Log(ctx, "premise", true, 10)
	if err != nil {
		t.Fatal(err)
	}
	requireLinearRevisionLog(t, log, root2.ID, root1.ID)
	impact, err := repo.Impact(ctx, "@"+root2.ID.String(), nil, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(impact.Impacts) != 1 || !impact.Impacts[0].Head.Equal(design2.ID) {
		t.Fatalf("impact = %+v", impact)
	}
	if design1.ID.Equal(design2.ID) {
		t.Fatal("design generations unexpectedly equal")
	}
}

func requireDirectStale(t *testing.T, stale []RefStatus, ref string, target domain.ObjectID) {
	t.Helper()
	if len(stale) != 1 || stale[0].REF != ref || len(stale[0].StaleDirect) != 1 || !stale[0].StaleDirect[0].Equal(target) {
		t.Fatalf("direct stale = %+v", stale)
	}
}

func requireNoStale(t *testing.T, stale []RefStatus) {
	t.Helper()
	if len(stale) != 0 {
		t.Fatalf("unexpected stale state = %+v", stale)
	}
}

func requireLinearRevisionLog(t *testing.T, log LogResult, head, previous domain.ObjectID) {
	t.Helper()
	if len(log.Entries) != 2 || !log.Entries[0].Resolved.ID.Equal(head) || !log.Entries[1].Resolved.ID.Equal(previous) {
		t.Fatalf("log = %+v", log.Entries)
	}
	if len(log.Paths) != 1 || len(log.Paths[0].Edges) != 1 {
		t.Fatalf("paths = %+v", log.Paths)
	}
}

func TestAssertionObserverFilterChangesOnlyRevisionClosure(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	old := sealRootV5(t, repo, "premise", "old")
	newer := sealRootV5(t, repo, "premise", "new")
	if _, err := repo.Add(ctx, AddOptions{REF: "observer", Content: []byte("o"), RootSet: true, Cause: &CauseInput{Target: "premise", Previous: []string{"@" + old.ID.String()}}}); err != nil {
		t.Fatal(err)
	}
	observer, err := repo.Seal(ctx, "observer")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Add(ctx, AddOptions{REF: "downstream", Content: []byte("d"), RootSet: true, Cause: &CauseInput{Target: "@" + old.ID.String()}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "downstream"); err == nil {
		t.Fatal("normal publication against non-leaf old target succeeded")
	}
	snapshot, err := repo.candidates.LoadSnapshot("downstream")
	if err != nil {
		t.Fatal(err)
	}
	candidate := snapshot.Candidate
	candidate.Draft = true
	if err := repo.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
		t.Fatal(err)
	}
	downstream, err := repo.Seal(ctx, "downstream")
	if err != nil {
		t.Fatal(err)
	}
	unfiltered, err := repo.Impact(ctx, "premise", nil, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	filtered, err := repo.Impact(ctx, "premise", []string{"@" + observer.ID.String()}, false, 100)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, impact := range filtered.Impacts {
		if impact.Head.Equal(downstream.ID) {
			found = true
		}
	}
	if len(unfiltered.Impacts) == 0 || !found {
		t.Fatalf("impact results unfiltered=%+v filtered=%+v newer=%s", unfiltered, filtered, newer.ID)
	}
}

func TestGraphRetainsMixedAssertionSources(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	old := sealRootV5(t, repo, "target-old", "old")
	newer := sealRootV5(t, repo, "target", "new")
	for _, item := range []struct {
		ref      string
		previous []string
	}{{"observer-none", nil}, {"observer-asserted", []string{"@" + old.ID.String()}}} {
		if _, err := repo.Add(ctx, AddOptions{REF: item.ref, Content: []byte(item.ref), RootSet: true, Draft: true, DraftSet: true, Cause: &CauseInput{Target: "target", Previous: item.previous}}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, item.ref); err != nil {
			t.Fatal(err)
		}
	}
	nodes, err := repo.Graph(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range nodes {
		if node.Resolved.ID.Equal(newer.ID) {
			states := node.Revision.PreviousStates
			if len(states) != 2 || states[0] != "PREVIOUS_NONE_ASSERTED" || states[1] != "PREVIOUS_ASSERTED" || len(node.Revision.Assertions) != 2 {
				t.Fatalf("revision observation = %+v", node.Revision)
			}
			return
		}
	}
	t.Fatal("target node missing")
}
