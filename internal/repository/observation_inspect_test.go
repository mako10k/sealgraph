package repository

// R: verify observation dimensions and read-only compatibility with publication.

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestObserveSeparatesSuccessorActiveAndCandidate(t *testing.T) {
	ctx := context.Background()
	repo, old, next := historicalFixture(t, true)
	before, err := repo.Status(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	physical, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil {
		t.Fatal(err)
	}
	result, err := repo.Observe(ctx, ObservationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	assertObserveRevisionCandidate(t, result, old.ID.String(), next.ID.String())
	after, err := repo.Status(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("legacy status changed")
	}
	final, err := capturePhysicalRepositoryObservation(repo.dir)
	if err != nil || !equalPhysicalRepositoryObservations(physical, final) {
		t.Fatal("observation wrote canonical state")
	}
	if _, err := repo.Seal(ctx, "dependent"); err == nil {
		t.Fatal("observation altered strict historical rejection")
	}
}

func TestObserveCandidateIdentityDraftAndInventoryAcrossFormats(t *testing.T) {
	for _, format := range []int{5, 6, 7} {
		t.Run(string(rune('0'+format)), func(t *testing.T) {
			ctx := context.Background()
			repo := observationFormatFixture(t, format)
			old := sealRootV5(t, repo, "root", "old")
			head := sealRootV5(t, repo, "root", "same")
			if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("same")}); err != nil {
				t.Fatal(err)
			}
			result, err := repo.Observe(ctx, ObservationOptions{Selector: "root"})
			if err != nil {
				t.Fatal(err)
			}
			entry := result.REFs[0]
			if entry.HeadDraft == nil || *entry.HeadDraft || entry.Candidate == nil || entry.Candidate.Draft || entry.Candidate.CurrentHeadComparison != "SAME" || !entry.Candidate.ProspectiveSeal.Equal(head.ID) {
				t.Fatalf("same candidate=%+v", entry)
			}
			if _, err := repo.Add(ctx, AddOptions{REF: "root", Content: []byte("same"), Draft: true, DraftSet: true}); err != nil {
				t.Fatal(err)
			}
			result, err = repo.Observe(ctx, ObservationOptions{Selector: "root"})
			if err != nil {
				t.Fatal(err)
			}
			if !result.REFs[0].Candidate.Draft || *result.REFs[0].HeadDraft || result.REFs[0].Candidate.CurrentHeadComparison != "DIFFERENT" {
				t.Fatal("draft dimensions merged")
			}
			assertObserveInventory(t, repo, old.ID.String())
		})
	}
}

func TestObserveReasonsKeepAllTargetsWhenPathsBounded(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	old := sealRootV5(t, repo, "target", "old")
	for _, name := range []string{"left", "right"} {
		if _, err := repo.Add(ctx, AddOptions{REF: name, Content: []byte(name), RootSet: true, Cause: cause("target")}); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Seal(ctx, name); err != nil {
			t.Fatal(err)
		}
	}
	sealRootV5(t, repo, "target", "new")
	if _, err := repo.Add(ctx, AddOptions{REF: "subject", Content: []byte("subject"), RootSet: true, Cause: cause("left")}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Link(ctx, "subject", CauseInput{Target: "right"}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Observe(ctx, ObservationOptions{Selector: "subject", MaxPaths: 1})
	if err != nil {
		t.Fatal(err)
	}
	reasons := result.REFs[0].Candidate.Reasons
	if len(reasons) != 1 || !reasons[0].Target.Equal(old.ID) || len(reasons[0].Paths) != 1 || reasons[0].PathsComplete || reasons[0].Kind != "OUTSIDE_ACTIVE" {
		t.Fatalf("bounded reasons=%+v", reasons)
	}
}

func TestObserveReasonDoesNotImportCandidateRevisionAssertions(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	old := sealRootV5(t, repo, "root", "old")
	next := sealRootV5(t, repo, "root", "new")
	if _, err := repo.Add(ctx, AddOptions{REF: "observer", Content: []byte("pending"), RootSet: true, Cause: cause("root", "@"+old.ID.String())}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Observe(ctx, ObservationOptions{Inventory: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, seal := range result.Seals {
		if seal.SealID.Equal(old.ID) && (seal.Active || len(seal.Successors) > 0) {
			t.Fatal("Candidate assertions changed canonical observation")
		}
		if seal.SealID.Equal(next.ID) && len(seal.Previous.StructuralPrevious) > 0 {
			t.Fatal("Candidate previous assertion imported")
		}
	}
}

func assertObserveRevisionCandidate(t *testing.T, result ObservationResult, old, next string) {
	t.Helper()
	var found bool
	for _, seal := range result.Seals {
		if seal.SealID.String() == old {
			found = true
			if !seal.Active || seal.ActiveLeaf || len(seal.Successors) != 1 || !(seal.Successors[0].String() == next) || len(seal.SuccessorAssertions) != 1 {
				t.Fatalf("old observation=%+v", seal)
			}
		}
	}
	if !found {
		t.Fatal("old target omitted")
	}
	for _, ref := range result.REFs {
		if ref.REF == "dependent" {
			if ref.HeadDraft != nil || ref.Candidate == nil || ref.Candidate.CurrentHeadComparison != "NO_HEAD" || len(ref.Candidate.Reasons) != 1 || ref.Candidate.Reasons[0].Kind != "ACTIVE_NON_LEAF" || ref.Candidate.Reasons[0].Direct {
				t.Fatalf("candidate=%+v", ref)
			}
		}
	}
}

func observationFormatFixture(t *testing.T, format int) *Repository {
	t.Helper()
	root := t.TempDir()
	if _, err := InitStandalone(root); err != nil {
		t.Fatal(err)
	}
	config := configBytes
	if format == 6 {
		config = format6ConfigBytes
	}
	if format == 7 {
		config = format7ConfigBytes
	}
	if err := os.WriteFile(filepath.Join(root, ".sealgraph", "config"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenStandalone(root)
	if err != nil {
		t.Fatal(err)
	}
	return repo
}

func assertObserveInventory(t *testing.T, repo *Repository, old string) {
	t.Helper()
	ctx := context.Background()
	selected, err := repo.Observe(ctx, ObservationOptions{Selector: "@" + old})
	if err != nil {
		t.Fatal(err)
	}
	if len(selected.Seals) != 1 || selected.Seals[0].Active || len(selected.Seals[0].Successors) != 0 {
		t.Fatalf("detached=%+v", selected)
	}
	inventory, err := repo.Observe(ctx, ObservationOptions{Inventory: true})
	if err != nil {
		t.Fatal(err)
	}
	seen := false
	for _, seal := range inventory.Seals {
		seen = seen || seal.SealID.String() == old
	}
	if !seen {
		t.Fatal("inventory omitted orphan history")
	}
}

func TestObserveDraftMatrix(t *testing.T) {
	for _, fixture := range []struct {
		name      string
		candidate bool
		draft     bool
	}{
		{"head-only", false, false}, {"head-draft-candidate-final", true, false}, {"both-draft", true, true},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newV5TestRepository(t)
			if _, err := repo.Add(ctx, AddOptions{REF: "draft", Content: []byte("draft"), Root: true, RootSet: true, ClearCauseLinks: true, Draft: true, DraftSet: true}); err != nil {
				t.Fatal(err)
			}
			if _, err := repo.Seal(ctx, "draft"); err != nil {
				t.Fatal(err)
			}
			if fixture.candidate {
				if _, err := repo.Add(ctx, AddOptions{REF: "draft", Content: []byte("candidate"), Draft: fixture.draft, DraftSet: true}); err != nil {
					t.Fatal(err)
				}
			}
			result, err := repo.Observe(ctx, ObservationOptions{Selector: "draft"})
			if err != nil {
				t.Fatal(err)
			}
			entry := result.REFs[0]
			if entry.HeadDraft == nil || !*entry.HeadDraft {
				t.Fatalf("head draft absent or false: %+v", entry)
			}
			if fixture.candidate {
				if entry.Candidate == nil || entry.Candidate.Draft != fixture.draft {
					t.Fatalf("Candidate draft merged: %+v", entry)
				}
			} else if entry.Candidate != nil {
				t.Fatal("absent Candidate represented as present")
			}
		})
	}
}

func TestObserveBranchSuccessorsKeepSiblingLeaves(t *testing.T) {
	ctx := context.Background()
	repo := newV5TestRepository(t)
	old := sealRootV5(t, repo, "old", "old")
	left := sealRootV5(t, repo, "left", "left")
	right := sealRootV5(t, repo, "right", "right")
	if _, err := repo.Add(ctx, AddOptions{REF: "observer", Content: []byte("branch evidence"), RootSet: true, Cause: cause("left", "@"+old.ID.String())}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Link(ctx, "observer", CauseInput{Target: "right", Previous: []string{"@" + old.ID.String()}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Seal(ctx, "observer"); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Observe(ctx, ObservationOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, seal := range result.Seals {
		if seal.SealID.Equal(old.ID) {
			if !seal.Active || seal.ActiveLeaf || len(seal.Successors) != 2 || len(seal.SuccessorAssertions) != 2 {
				t.Fatalf("branch old=%+v", seal)
			}
		}
		if seal.SealID.Equal(left.ID) || seal.SealID.Equal(right.ID) {
			if !seal.ActiveLeaf || len(seal.Successors) != 0 {
				t.Fatalf("sibling=%+v", seal)
			}
		}
	}
}
