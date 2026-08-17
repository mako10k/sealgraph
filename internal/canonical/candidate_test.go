package canonical

import (
	"bytes"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func TestFormat4CandidateDeterministicBytes(t *testing.T) {
	parent, expected := fixtureID('e'), fixtureID('e')
	candidate := domain.Candidate{
		Schema: domain.CandidateSchema, REF: "design/api",
		ParentRevision: &parent, ExpectedREFHead: &expected,
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('a')},
		Attachments: []domain.Attachment{},
		Links:       []domain.Link{{TargetSeal: fixtureID('c'), Message: "z"}, {TargetSeal: fixtureID('b'), Message: "a"}},
	}
	encoded, err := EncodeCandidate(candidate)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasSuffix(encoded, []byte{'\n'}) || bytes.Contains(encoded, []byte("target_ref")) || bytes.Contains(encoded, []byte(`"base"`)) {
		t.Fatalf("candidate bytes violate format-4 boundary: %s", encoded)
	}
	const expectedID = "2400bb778e6275421fe1c4651cddd500dfb7ec86c3d3dc4552b6ad0b34149fb9"
	if id := domain.ComputeNativeBlobID(encoded); id.String() != expectedID {
		t.Fatalf("candidate fixture byte hash = %s, want %s\n%s", id, expectedID, encoded)
	}
	decoded, err := DecodeCandidate(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Links[0].TargetSeal.Equal(fixtureID('b')) {
		t.Fatalf("candidate links were not persisted in deterministic order: %+v", decoded.Links)
	}
}

func TestCandidateDecoderRejectsLegacyAndUnknownMembers(t *testing.T) {
	legacy := []byte(`{"schema":"sealgraph/candidate/v3","ref":"x","base":null,"content":{"store":"native","type":"blob","id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},"attachments":[],"links":[],"root":true,"draft":false}`)
	if _, err := DecodeCandidate(legacy); err == nil {
		t.Fatal("legacy candidate was accepted")
	}
}
