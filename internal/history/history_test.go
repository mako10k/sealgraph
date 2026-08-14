package history

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func testID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(char), 64)}
}

func testLink(ref string, id domain.ObjectID) domain.Link {
	return domain.Link{TargetREF: ref, TargetSeal: id}
}

func TestWalkReturnsNewestFirstAndRejectsForeignParent(t *testing.T) {
	a, b, c := testID('a'), testID('b'), testID('c')
	seals := map[string]domain.SealPayload{
		a.String(): {REF: "design/api", Parent: &b},
		b.String(): {REF: "design/api", Parent: &c},
		c.String(): {REF: "design/api"},
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		payload, found := seals[id.String()]
		if !found {
			return domain.SealPayload{}, errors.New("missing")
		}
		return payload, nil
	}

	entries, err := Walk(context.Background(), "design/api", a, load)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || !entries[0].ID.Equal(a) || !entries[1].ID.Equal(b) || !entries[2].ID.Equal(c) {
		t.Fatalf("entries = %+v, want a, b, c", entries)
	}

	seals[b.String()] = domain.SealPayload{REF: "other", Parent: &c}
	if _, err := Walk(context.Background(), "design/api", a, load); err == nil || !strings.Contains(err.Error(), "belongs to REF other, not design/api") {
		t.Fatalf("foreign parent error = %v", err)
	}
}

func TestWalkRejectsParentCycle(t *testing.T) {
	a, b := testID('a'), testID('b')
	seals := map[string]domain.SealPayload{
		a.String(): {REF: "REF", Parent: &b},
		b.String(): {REF: "REF", Parent: &a},
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}

	_, err := Walk(context.Background(), "REF", a, load)
	var cycle *CycleError
	if !errors.As(err, &cycle) {
		t.Fatalf("error = %v, want CycleError", err)
	}
	if len(cycle.Path) != 3 || !cycle.Path[0].Seal.Equal(a) || !cycle.Path[1].Seal.Equal(b) || !cycle.Path[2].Seal.Equal(a) {
		t.Fatalf("cycle path = %+v", cycle.Path)
	}
}

func TestDiffLinksClassifiesAndSortsChanges(t *testing.T) {
	a1, a2 := testID('a'), testID('b')
	b1, c1 := testID('c'), testID('d')
	changes := DiffLinks(
		[]domain.Link{testLink("B", b1), testLink("A", a1)},
		[]domain.Link{testLink("C", c1), testLink("A", a2)},
	)
	if len(changes) != 3 {
		t.Fatalf("changes = %+v", changes)
	}
	if changes[0].TargetREF != "A" || changes[0].Kind != LinkRepoint || !changes[0].BeforeSeal.Equal(a1) || !changes[0].AfterSeal.Equal(a2) {
		t.Fatalf("A change = %+v", changes[0])
	}
	if changes[1].TargetREF != "B" || changes[1].Kind != LinkRemove || changes[2].TargetREF != "C" || changes[2].Kind != LinkAdd {
		t.Fatalf("ordered changes = %+v", changes)
	}
	if unchanged := DiffLinks(
		[]domain.Link{testLink("B", b1), testLink("A", a1)},
		[]domain.Link{testLink("A", a1), testLink("B", b1)},
	); len(unchanged) != 0 {
		t.Fatalf("input order created false link changes: %+v", unchanged)
	}
}

func TestDeriveLinkLogIncludesInitialAddAndFilteredNoChange(t *testing.T) {
	root1, root2, other := testID('1'), testID('2'), testID('3')
	oldID, newID := testID('a'), testID('b')
	entries := []Entry{
		{ID: newID, Payload: domain.SealPayload{REF: "CHILD", Parent: &oldID, Links: []domain.Link{testLink("ROOT", root2), testLink("OTHER", other)}}},
		{ID: oldID, Payload: domain.SealPayload{REF: "CHILD", Links: []domain.Link{testLink("ROOT", root1), testLink("OTHER", other)}}},
	}
	log := DeriveLinkLog(entries, "ROOT")
	if len(log) != 2 || len(log[0].Changes) != 1 || log[0].Changes[0].Kind != LinkRepoint {
		t.Fatalf("new entry changes = %+v", log)
	}
	if len(log[1].Changes) != 1 || log[1].Changes[0].Kind != LinkAdd || log[1].Changes[0].TargetREF != "ROOT" {
		t.Fatalf("initial entry changes = %+v", log[1].Changes)
	}

	filtered := DeriveLinkLog(entries, "MISSING")
	if len(filtered) != 2 || len(filtered[0].Changes) != 0 || len(filtered[1].Changes) != 0 {
		t.Fatalf("filtered log = %+v", filtered)
	}
}

func TestDiffSealsCoversCanonicalFields(t *testing.T) {
	fromID, toID := testID('a'), testID('b')
	content1, content2 := testID('1'), testID('2')
	parent1, parent2 := testID('3'), testID('4')
	root1, root2 := testID('5'), testID('6')
	blob1, blob2 := testID('7'), testID('8')
	from := domain.SealPayload{
		REF: "design/api", Parent: &parent1,
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: content1},
		Attachments: []domain.Attachment{{Name: "report", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: blob1}}},
		Links:       []domain.Link{testLink("ROOT", root1)}, Message: "before", Root: false, Draft: true, CreatedAt: "2026-08-14T00:00:00Z",
	}
	to := domain.SealPayload{
		REF: "design/api", Parent: &parent2,
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: content2},
		Attachments: []domain.Attachment{{Name: "report", MediaType: "application/octet-stream", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: blob2}}},
		Links:       []domain.Link{testLink("ROOT", root2)}, Message: "after", Root: true, Draft: false, CreatedAt: "2026-08-14T00:00:01Z",
	}

	diff, err := DiffSeals(fromID, from, toID, to)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Content.Changed || len(diff.Attachments) != 1 || diff.Attachments[0].Kind != AttachmentChange || len(diff.Links) != 1 || diff.Links[0].Kind != LinkRepoint {
		t.Fatalf("collection diff = %+v", diff)
	}
	if !diff.Message.Changed || !diff.Root.Changed || !diff.Draft.Changed || !diff.Parent.Changed || !diff.CreatedAt.Changed {
		t.Fatalf("scalar diff = %+v", diff)
	}

	to.REF = "other"
	if _, err := DiffSeals(fromID, from, toID, to); err == nil || !strings.Contains(err.Error(), "different REFs") {
		t.Fatalf("cross-REF error = %v", err)
	}
}

func TestDiffSealsIgnoresCanonicalSetInputOrder(t *testing.T) {
	sealID := testID('a')
	content := domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: testID('1')}
	a := domain.Attachment{Name: "a", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: testID('2')}}
	b := domain.Attachment{Name: "b", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: testID('3')}}
	linkA := testLink("A", testID('4'))
	linkB := testLink("B", testID('5'))
	from := domain.SealPayload{REF: "REF", Content: content, Attachments: []domain.Attachment{b, a}, Links: []domain.Link{linkB, linkA}}
	to := domain.SealPayload{REF: "REF", Content: content, Attachments: []domain.Attachment{a, b}, Links: []domain.Link{linkA, linkB}}

	diff, err := DiffSeals(sealID, from, sealID, to)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Attachments) != 0 || len(diff.Links) != 0 {
		t.Fatalf("input order created false set changes: %+v", diff)
	}
}
