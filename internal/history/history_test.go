package history

import (
	"context"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func historyID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(char), 64)}
}

func historySeal(parent *domain.ObjectID, links ...domain.Link) domain.SealPayload {
	return domain.SealPayload{
		Schema: domain.SealSchema, ParentRevision: parent,
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: historyID('f')},
		Attachments: []domain.Attachment{}, Links: links, Root: len(links) == 0,
	}
}

func TestWalkFollowsOwnerlessParentRevision(t *testing.T) {
	a, b, c := historyID('a'), historyID('b'), historyID('c')
	seals := map[string]domain.SealPayload{
		a.String(): historySeal(nil),
		b.String(): historySeal(&a),
		c.String(): historySeal(&b),
	}
	entries, err := Walk(context.Background(), "display/ref", c, func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 3 || !entries[2].ID.Equal(a) {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestExactTargetLinkDiff(t *testing.T) {
	a, b := historyID('a'), historyID('b')
	before := []domain.Link{{TargetSeal: a, Message: "old"}}
	after := []domain.Link{{TargetSeal: a, Message: "new"}, {TargetSeal: b, Message: "basis"}}
	changes := DiffLinks(before, after)
	if len(changes) != 2 || changes[0].Kind != LinkMessage || !changes[0].TargetSeal.Equal(a) || changes[1].Kind != LinkAdd || !changes[1].TargetSeal.Equal(b) {
		t.Fatalf("changes = %+v", changes)
	}
}

func TestDiffSealsIncludesParentRevision(t *testing.T) {
	a, b := historyID('a'), historyID('b')
	from := historySeal(nil)
	to := historySeal(&a)
	diff, err := DiffSeals(a, from, b, to)
	if err != nil {
		t.Fatal(err)
	}
	if !diff.Parent.Changed || diff.Parent.After == nil || !diff.Parent.After.Equal(a) {
		t.Fatalf("parent diff = %+v", diff.Parent)
	}
}

func TestLinkLogPresentsOnlyUnambiguousAncestryRepoint(t *testing.T) {
	oldTarget, newTarget := historyID('1'), historyID('2')
	oldDependent, newDependent := historyID('3'), historyID('4')
	seals := map[string]domain.SealPayload{
		oldTarget.String(): {Schema: domain.SealSchema, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true},
		newTarget.String(): {Schema: domain.SealSchema, ParentRevision: &oldTarget, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true},
	}
	entries := []Entry{
		{ID: newDependent, Payload: historySeal(&oldDependent, domain.Link{TargetSeal: newTarget, Message: "new"})},
		{ID: oldDependent, Payload: historySeal(nil, domain.Link{TargetSeal: oldTarget, Message: "old"})},
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	log, err := DeriveLinkLog(context.Background(), entries, "", load)
	if err != nil {
		t.Fatal(err)
	}
	changes := log[0].Changes
	if len(changes) != 1 || changes[0].Kind != LinkRepoint || !changes[0].BeforeSeal.Equal(oldTarget) || !changes[0].AfterSeal.Equal(newTarget) {
		t.Fatalf("changes = %+v", changes)
	}
	reversed := []Entry{
		{ID: newDependent, Payload: historySeal(&oldDependent, domain.Link{TargetSeal: oldTarget})},
		{ID: oldDependent, Payload: historySeal(nil, domain.Link{TargetSeal: newTarget})},
	}
	log, err = DeriveLinkLog(context.Background(), reversed, "", load)
	if err != nil {
		t.Fatal(err)
	}
	if changes := log[0].Changes; len(changes) != 2 || changes[0].Kind == LinkRepoint || changes[1].Kind == LinkRepoint {
		t.Fatalf("reverse ancestry changes = %+v", changes)
	}
}
