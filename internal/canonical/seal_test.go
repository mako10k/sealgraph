package canonical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func fixtureID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(char), 64)}
}

func fixturePayload() domain.SealPayload {
	parent := fixtureID('e')
	return domain.SealPayload{
		Schema:         domain.SealSchema,
		ParentRevision: &parent,
		Content:        domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('a')},
		Attachments:    []domain.Attachment{},
		Links: []domain.Link{
			{TargetSeal: fixtureID('c'), Message: "later input"},
			{TargetSeal: fixtureID('b'), Message: "review basis"},
		},
	}
}

func TestCanonicalFormat4SealFixture(t *testing.T) {
	encoded, err := EncodeSeal(fixturePayload())
	if err != nil {
		t.Fatal(err)
	}
	const expected = `{"schema":"sealgraph/seal/v4","parent_revision":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","content":{"store":"native","type":"blob","id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"attachments":[],"links":[{"target_seal":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","message":"review basis"},{"target_seal":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","message":"later input"}],"root":false,"draft":false}`
	if string(encoded) != expected {
		t.Fatalf("canonical format-4 bytes differ:\n%s", encoded)
	}
	const expectedID = "d73988845debf3a426e92b33b3269f6dcca41f5dce265f8630c80f88911364ec"
	if id := domain.ComputeNativeBlobID(encoded); id.String() != expectedID {
		t.Fatalf("fixture hash = %s, want %s", id, expectedID)
	}
	decoded, err := DecodeSeal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Links[0].TargetSeal.Equal(fixtureID('b')) {
		t.Fatalf("links were not sorted by exact target: %+v", decoded.Links)
	}
}

func TestFormat4SealInputOrderAndREFIndependence(t *testing.T) {
	first := fixturePayload()
	second := fixturePayload()
	second.Links[0], second.Links[1] = second.Links[1], second.Links[0]
	a, err := EncodeSeal(first)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EncodeSeal(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Fatal("link input order changed canonical identity")
	}
	if bytes.Contains(a, []byte(`"ref"`)) || bytes.Contains(a, []byte("target_ref")) {
		t.Fatalf("format-4 Seal retained REF ownership: %s", a)
	}
}

func TestFormat4IdentityCommitsParentTargetAndMessage(t *testing.T) {
	base := fixturePayload()
	baseBytes, _ := EncodeSeal(base)
	baseID := domain.ComputeNativeBlobID(baseBytes)
	variants := []domain.SealPayload{fixturePayload(), fixturePayload(), fixturePayload()}
	variants[0].ParentRevision = nil
	variants[1].Links[0].TargetSeal = fixtureID('d')
	variants[2].Links[0].Message = "different"
	for i, variant := range variants {
		encoded, err := EncodeSeal(variant)
		if err != nil {
			t.Fatal(err)
		}
		if domain.ComputeNativeBlobID(encoded).Equal(baseID) {
			t.Fatalf("identity-bearing variant %d did not change SealID", i)
		}
	}
}

func TestFormat4SealRejectsDuplicateTargetAndNoncanonicalBytes(t *testing.T) {
	payload := fixturePayload()
	payload.Links[1].TargetSeal = payload.Links[0].TargetSeal
	if _, err := EncodeSeal(payload); err == nil {
		t.Fatal("duplicate exact Cause target was accepted")
	}
	encoded, _ := EncodeSeal(fixturePayload())
	changed := bytes.Replace(encoded, []byte(`{"schema":"sealgraph/seal/v4","parent_revision":`), []byte(`{"parent_revision":`), 1)
	if _, err := DecodeSeal(changed); err == nil {
		t.Fatal("noncanonical member set/order was accepted")
	}
}

func TestFormat4AttachmentOrderAndDuplicateName(t *testing.T) {
	payload := fixturePayload()
	payload.Attachments = []domain.Attachment{
		{Name: "z", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('f')}},
		{Name: "a", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('d')}},
	}
	first, err := EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	payload.Attachments[0], payload.Attachments[1] = payload.Attachments[1], payload.Attachments[0]
	second, err := EncodeSeal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("attachment order changed canonical bytes")
	}
	payload.Attachments[1].Name = payload.Attachments[0].Name
	if _, err := EncodeSeal(payload); err == nil {
		t.Fatal("duplicate attachment name was accepted")
	}
}
