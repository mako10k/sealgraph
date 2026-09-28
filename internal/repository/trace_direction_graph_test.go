package repository

import (
	"context"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func sealDirectionCause(t *testing.T, repo *Repository, ref, target string) SealResult {
	t.Helper()
	ctx := context.Background()
	if _, err := repo.Add(ctx, AddOptions{REF: ref, Content: []byte(ref), RootSet: true, Cause: &CauseInput{Target: target}}); err != nil {
		t.Fatal(err)
	}
	result, err := repo.Seal(ctx, ref)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func memberByID(values []DirectionGraphMember, id domain.ObjectID) (DirectionGraphMember, bool) {
	for _, value := range values {
		if value.SealID.Equal(id) {
			return value, true
		}
	}
	return DirectionGraphMember{}, false
}

func TestTraceDirectionGraphSeparatesDirectAndIndirectDiamondPaths(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	a := sealRootV5(t, repo, "a", "a")
	b := sealDirectionCause(t, repo, "b", "a")
	c := sealDirectionCause(t, repo, "c", "a")
	if _, err := repo.Add(ctx, AddOptions{REF: "d", Content: []byte("d"), RootSet: true, Cause: &CauseInput{Target: "b"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Link(ctx, "d", CauseInput{Target: "c"}); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Link(ctx, "d", CauseInput{Target: "a"}); err != nil {
		t.Fatal(err)
	}
	d, err := repo.Seal(ctx, "d")
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.TraceDirectionGraph(ctx, a.ID, 100)
	if err != nil || !got.ClosureComplete || !got.DirectionComplete {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	for _, id := range []domain.ObjectID{b.ID, c.ID} {
		member, ok := memberByID(got.Downstream, id)
		if !ok || !member.Direct || member.Indirect || len(member.DirectPath) != 2 {
			t.Fatalf("downstream %s = %+v", id, member)
		}
	}
	member, ok := memberByID(got.Downstream, d.ID)
	if !ok || !member.Direct || !member.Indirect || len(member.DirectPath) != 2 || len(member.IndirectPath) != 3 || !member.IndirectPath[0].Equal(d.ID) || !member.IndirectPath[2].Equal(a.ID) {
		t.Fatalf("diamond downstream = %+v", member)
	}
}

func TestTraceDirectionGraphDoesNotUsePreviousRevisionAsDirection(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	old := sealRootV5(t, repo, "a", "old")
	newer := sealRootV5(t, repo, "a", "new")
	if _, err := repo.Add(ctx, AddOptions{REF: "observer", Content: []byte("o"), RootSet: true, Cause: &CauseInput{Target: "a", Previous: []string{"@" + old.ID.String()}}}); err != nil {
		t.Fatal(err)
	}
	observer, err := repo.Seal(ctx, "observer")
	if err != nil {
		t.Fatal(err)
	}
	got, err := repo.TraceDirectionGraph(ctx, old.ID, 100)
	if err != nil || !got.ClosureComplete || !got.DirectionComplete {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	if _, found := memberByID(got.Downstream, observer.ID); found {
		t.Fatalf("previous-only observer appeared downstream: %+v", got.Downstream)
	}
	if _, found := memberByID(got.Upstream, newer.ID); found {
		t.Fatalf("revision assertion appeared upstream: %+v", got.Upstream)
	}
}

func TestTraceDirectionGraphAddsSelectedHistoricalClosure(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	parent := sealRootV5(t, repo, "parent", "parent")
	old := sealDirectionCause(t, repo, "a", "parent")
	_ = sealRootV5(t, repo, "a", "new")
	got, err := repo.TraceDirectionGraph(ctx, old.ID, 100)
	if err != nil || len(got.ExtraSeals) != 1 || !got.ExtraSeals[0].Equal(old.ID) || len(got.ObservedSealIDs) != 3 {
		t.Fatalf("result=%+v err=%v", got, err)
	}
	if member, ok := memberByID(got.Upstream, parent.ID); !ok || !member.Direct {
		t.Fatalf("historical upstream=%+v", got.Upstream)
	}
}

func TestTraceDirectionGraphReturnsCenterWhenBudgetStopsClosure(t *testing.T) {
	repo := newV5TestRepository(t)
	ctx := context.Background()
	a := sealRootV5(t, repo, "a", "a")
	_ = sealDirectionCause(t, repo, "b", "a")
	got, err := repo.TraceDirectionGraph(ctx, a.ID, 1)
	if err != nil || got.ClosureComplete || got.DirectionComplete || got.StopReason != directionGraphVisitLimit || len(got.ObservedSealIDs) != 1 || !got.ObservedSealIDs[0].Equal(a.ID) {
		t.Fatalf("result=%+v err=%v", got, err)
	}
}

func TestTraceDirectionGraphBudgetCanStopAfterCompleteClosure(t *testing.T) {
	repo := newV5TestRepository(t)
	_ = sealRootV5(t, repo, "a", "a")
	heads, err := repo.TraceDirectionHeads(context.Background())
	if err != nil || len(heads.REFHeads) != 1 {
		t.Fatalf("heads=%+v err=%v", heads, err)
	}
	got, err := repo.TraceDirectionGraph(context.Background(), heads.REFHeads[0].SealID, 2)
	if err != nil || !got.ClosureComplete || got.DirectionComplete || got.StopReason != directionGraphVisitLimit || got.UsedVisits != 2 || len(got.ObservedSealIDs) != 1 {
		t.Fatalf("result=%+v err=%v", got, err)
	}
}
