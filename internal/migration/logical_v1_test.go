package migration

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

func TestEncodeLogicalDumpV1EmptyFixture(t *testing.T) {
	encoded, err := EncodeLogicalDumpV1(LogicalDumpV1{})
	if err != nil {
		t.Fatal(err)
	}
	want := "{\"schema\":\"sealgraph/logical-dump/v1\",\"source_repository\":{\"repository_format\":3,\"object_format\":\"sha256\"},\"objects\":[],\"seals\":[],\"refs\":[],\"tags\":[],\"excluded_objects\":[]}\n"
	if string(encoded) != want {
		t.Fatalf("empty logical dump = %q, want %q", encoded, want)
	}
}

func TestEncodeLogicalDumpV1ExactBinaryAndTagFixture(t *testing.T) {
	content := []byte{0, '\r', '\n', 0xff}
	contentID := domain.ComputeNativeBlobID(content)
	payload := domain.SealPayload{
		Schema:  domain.SealSchema,
		REF:     "ROOT",
		Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Root:    true,
	}
	sealBytes, err := canonical.EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	sealID := domain.ComputeNativeBlobID(sealBytes)
	dump := LogicalDumpV1{
		Objects: []ObjectRecord{{ID: contentID, Data: content}},
		Seals:   []SealRecord{{OldSealID: sealID, Payload: payload}},
		REFs:    []RefRecord{{Name: "ROOT", Head: sealID}},
		Tags:    []TagRecord{{REF: "ROOT", Name: "v<1", Target: sealID}},
	}
	encoded, err := EncodeLogicalDumpV1(dump)
	if err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("{\"schema\":\"sealgraph/logical-dump/v1\",\"source_repository\":{\"repository_format\":3,\"object_format\":\"sha256\"},\"objects\":[{\"id\":\"%s\",\"type\":\"blob\",\"data_base64\":\"AA0K/w==\"}],\"seals\":[{\"old_seal_id\":\"%s\",\"payload\":%s}],\"refs\":[{\"name\":\"ROOT\",\"head\":\"%s\"}],\"tags\":[{\"ref\":\"ROOT\",\"name\":\"v<1\",\"target\":\"%s\"}],\"excluded_objects\":[]}\n", contentID, sealID, sealBytes, sealID, sealID)
	if string(encoded) != want {
		t.Fatalf("logical dump = %q, want %q", encoded, want)
	}
	if strings.Contains(string(encoded), `\u003c`) {
		t.Fatalf("canonical dump used HTML escaping: %s", encoded)
	}
}

func TestEncodeLogicalDumpV1RejectsNonTopologicalSeals(t *testing.T) {
	rootContent := []byte("root")
	rootContentID := domain.ComputeNativeBlobID(rootContent)
	root := domain.SealPayload{
		Schema:  domain.SealSchema,
		REF:     "ROOT",
		Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: rootContentID},
		Root:    true,
	}
	rootBytes, err := canonical.EncodeSeal(root)
	if err != nil {
		t.Fatal(err)
	}
	rootID := domain.ComputeNativeBlobID(rootBytes)
	childContent := []byte("child")
	childContentID := domain.ComputeNativeBlobID(childContent)
	child := domain.SealPayload{
		Schema:  domain.SealSchema,
		REF:     "CHILD",
		Content: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: childContentID},
		Links:   []domain.Link{{TargetREF: "ROOT", TargetSeal: rootID}},
	}
	childBytes, err := canonical.EncodeSeal(child)
	if err != nil {
		t.Fatal(err)
	}
	childID := domain.ComputeNativeBlobID(childBytes)
	objects := []ObjectRecord{
		{ID: childContentID, Data: childContent},
		{ID: rootContentID, Data: rootContent},
	}
	sort.Slice(objects, func(i, j int) bool { return objects[i].ID.String() < objects[j].ID.String() })
	_, err = EncodeLogicalDumpV1(LogicalDumpV1{
		Objects: objects,
		Seals: []SealRecord{
			{OldSealID: childID, Payload: child},
			{OldSealID: rootID, Payload: root},
		},
		REFs: []RefRecord{{Name: "CHILD", Head: childID}, {Name: "ROOT", Head: rootID}},
	})
	if err == nil || !strings.Contains(err.Error(), "does not precede") {
		t.Fatalf("non-topological error = %v", err)
	}
}
