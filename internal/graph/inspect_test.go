package graph

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

type memoryRefs map[string]domain.ObjectID

func (refs memoryRefs) Resolve(_ context.Context, ref string) (domain.ObjectID, error) {
	id, found := refs[ref]
	if !found {
		return domain.ObjectID{}, store.ErrRefNotFound
	}
	return id, nil
}
func (memoryRefs) Update(context.Context, string, *domain.ObjectID, *domain.ObjectID) error {
	return errors.New("not implemented")
}
func (refs memoryRefs) List(context.Context) ([]string, error) {
	result := make([]string, 0, len(refs))
	for ref := range refs {
		result = append(result, ref)
	}
	return result, nil
}

func graphID(char byte) domain.ObjectID {
	return domain.ObjectID{Algorithm: domain.NativeAlgorithm, Hex: strings.Repeat(string(char), 64)}
}

func graphSeal(ref string, links ...domain.Link) domain.SealPayload {
	return domain.SealPayload{REF: ref, Links: links}
}

func graphLink(ref string, id domain.ObjectID) domain.Link {
	return domain.Link{Relation: domain.DependOn, TargetREF: ref, TargetSeal: id}
}

func TestInspectDerivesDirectAndTransitiveStale(t *testing.T) {
	a1, a2, b1, c1 := graphID('a'), graphID('d'), graphID('b'), graphID('c')
	seals := map[string]domain.SealPayload{
		a1.String(): graphSeal("A"),
		b1.String(): graphSeal("B", graphLink("A", a1)),
		c1.String(): graphSeal("C", graphLink("B", b1)),
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	refs := memoryRefs{"A": a2, "B": b1, "C": c1}

	b, err := Inspect(context.Background(), SealIdentity{REF: "B", Seal: b1}, seals[b1.String()], refs, load)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Direct) != 1 || len(b.Transitive) != 0 {
		t.Fatalf("B inspection = %+v, want one direct stale relation", b)
	}

	c, err := Inspect(context.Background(), SealIdentity{REF: "C", Seal: c1}, seals[c1.String()], refs, load)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Direct) != 0 || len(c.Transitive) != 1 {
		t.Fatalf("C inspection = %+v, want one transitive stale relation", c)
	}
	if got := c.Transitive[0].Nodes; len(got) != 2 || got[0].REF != "C" || got[1].REF != "B" {
		t.Fatalf("transitive path = %+v, want C -> B", got)
	}

	impacts, err := ReverseImpact(context.Background(), "A", []SealIdentity{{REF: "A", Seal: a2}, {REF: "B", Seal: b1}, {REF: "C", Seal: c1}}, load)
	if err != nil {
		t.Fatal(err)
	}
	if len(impacts) != 2 || !impacts[0].Direct || impacts[1].Direct || impacts[0].REF != "B" || impacts[1].REF != "C" {
		t.Fatalf("impacts = %+v, want direct B and transitive C", impacts)
	}
}

func TestInspectRejectsCycle(t *testing.T) {
	x, y := graphID('e'), graphID('f')
	seals := map[string]domain.SealPayload{
		x.String(): graphSeal("X", graphLink("Y", y)),
		y.String(): graphSeal("Y", graphLink("X", x)),
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	refs := memoryRefs{"X": x, "Y": y}
	_, err := Inspect(context.Background(), SealIdentity{REF: "X", Seal: x}, seals[x.String()], refs, load)
	var cycle *CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("error = %v, want CycleError", err)
	}
	if len(cycle.Path) != 3 || cycle.Path[0].REF != "X" || cycle.Path[2].REF != "X" {
		t.Fatalf("cycle path = %+v, want X -> Y -> X", cycle.Path)
	}
}

func TestReverseImpactReportsEveryDistinctPath(t *testing.T) {
	a, b, c, d := graphID('a'), graphID('b'), graphID('c'), graphID('d')
	seals := map[string]domain.SealPayload{
		a.String(): graphSeal("A"),
		b.String(): graphSeal("B", graphLink("A", a)),
		c.String(): graphSeal("C", graphLink("A", a)),
		d.String(): graphSeal("D", graphLink("B", b), graphLink("C", c)),
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	impacts, err := ReverseImpact(context.Background(), "A", []SealIdentity{{REF: "A", Seal: a}, {REF: "D", Seal: d}}, load)
	if err != nil {
		t.Fatal(err)
	}
	if len(impacts) != 2 || impacts[0].REF != "D" || impacts[1].REF != "D" {
		t.Fatalf("impacts = %+v, want two paths for D", impacts)
	}
	if got := impacts[0].Path; len(got) != 3 || got[1].REF != "B" {
		t.Fatalf("first path = %+v, want D -> B -> A", got)
	}
	if got := impacts[1].Path; len(got) != 3 || got[1].REF != "C" {
		t.Fatalf("second path = %+v, want D -> C -> A", got)
	}
}
