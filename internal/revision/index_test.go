package revision

import (
	"context"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func revisionID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: string(make([]byte, 0)) + string(repeatByte(char, 64))}
}

func repeatByte(value byte, count int) []byte {
	result := make([]byte, count)
	for i := range result {
		result[i] = value
	}
	return result
}

func TestBuildClassifiesBranchingActiveLeavesAndAliases(t *testing.T) {
	root, left, right := revisionID('1'), revisionID('2'), revisionID('3')
	seals := map[string]domain.SealPayload{
		root.String():  {Schema: domain.SealSchema, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true},
		left.String():  {Schema: domain.SealSchema, ParentRevision: &root, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true},
		right.String(): {Schema: domain.SealSchema, ParentRevision: &root, Attachments: []domain.Attachment{}, Links: []domain.Link{}, Root: true},
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	index, err := Build(context.Background(), []Head{{REF: "right", Seal: right}, {REF: "left", Seal: left}, {REF: "alias", Seal: left}}, load)
	if err != nil {
		t.Fatal(err)
	}
	if index.State(root) != StaleRevision || index.State(left) != CurrentLeaf || index.State(right) != CurrentLeaf {
		t.Fatalf("states root=%s left=%s right=%s", index.State(root), index.State(left), index.State(right))
	}
	node, _ := index.Node(left)
	if len(node.REFs) != 2 || node.REFs[0] != "alias" || node.REFs[1] != "left" {
		t.Fatalf("aliases=%v", node.REFs)
	}
}

func TestBuildRejectsParentCycle(t *testing.T) {
	first, second := revisionID('a'), revisionID('b')
	seals := map[string]domain.SealPayload{
		first.String():  {Schema: domain.SealSchema, ParentRevision: &second},
		second.String(): {Schema: domain.SealSchema, ParentRevision: &first},
	}
	load := func(_ context.Context, id domain.ObjectID) (domain.SealPayload, error) {
		return seals[id.String()], nil
	}
	if _, err := Build(context.Background(), []Head{{REF: "cycle", Seal: first}}, load); err == nil {
		t.Fatal("expected parent cycle rejection")
	}
}
