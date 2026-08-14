package canonical

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store/native"
)

func fixtureID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(char), 64)}
}

func fixturePayload() domain.SealPayload {
	return domain.SealPayload{
		Schema:      domain.SealSchema,
		REF:         "design/api/DESIGN-001",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('a')},
		Attachments: []domain.Attachment{},
		Links: []domain.Link{
			{TargetREF: "requirements/REQ-B", TargetSeal: fixtureID('c'), Message: "later input"},
			{TargetREF: "requirements/REQ-A", TargetSeal: fixtureID('b'), Message: "review basis"},
		},
		Message:   "Reviewed <exactly>\nline two",
		CreatedAt: "2026-08-14T00:00:00Z",
	}
}

func TestCanonicalSealFixtureHash(t *testing.T) {
	encoded, err := EncodeSeal(fixturePayload())
	if err != nil {
		t.Fatal(err)
	}
	id := native.ObjectID(encoded)
	const expected = "5405d157bdc6ec7a1a23f7f575c48c6ccbdfbb582a313395b5a4e7eee65d4b42"
	if id.Hex != expected {
		t.Fatalf("fixture hash = %s, want %s\npayload=%s", id.Hex, expected, encoded)
	}
	decoded, err := DecodeSeal(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Links[0].TargetREF != "requirements/REQ-A" {
		t.Fatalf("links were not canonicalized: %+v", decoded.Links)
	}
}

func TestDependencyInputOrderHasSameCanonicalRepresentation(t *testing.T) {
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
		t.Fatalf("canonical payload differs by link input order:\n%s\n%s", a, b)
	}
}

func TestDirectUpstreamAndMessageAffectSealIdentity(t *testing.T) {
	base := fixturePayload()
	encoded, err := EncodeSeal(base)
	if err != nil {
		t.Fatal(err)
	}
	baseID := native.ObjectID(encoded)

	upstream := fixturePayload()
	upstream.Links[0].TargetSeal = fixtureID('d')
	upstreamBytes, err := EncodeSeal(upstream)
	if err != nil {
		t.Fatal(err)
	}
	if baseID.Equal(native.ObjectID(upstreamBytes)) {
		t.Fatal("changing only direct upstream seal identity did not change seal identity")
	}

	message := fixturePayload()
	message.Message = "Different review message"
	messageBytes, err := EncodeSeal(message)
	if err != nil {
		t.Fatal(err)
	}
	if baseID.Equal(native.ObjectID(messageBytes)) {
		t.Fatal("changing only message did not change seal identity")
	}

	linkMessage := fixturePayload()
	linkMessage.Links[0].Message = "different edge rationale"
	linkMessageBytes, err := EncodeSeal(linkMessage)
	if err != nil {
		t.Fatal(err)
	}
	if baseID.Equal(native.ObjectID(linkMessageBytes)) {
		t.Fatal("changing only a link message did not change seal identity")
	}

	createdAt := fixturePayload()
	createdAt.CreatedAt = "2026-08-14T00:00:01Z"
	createdAtBytes, err := EncodeSeal(createdAt)
	if err != nil {
		t.Fatal(err)
	}
	if baseID.Equal(native.ObjectID(createdAtBytes)) {
		t.Fatal("changing only created_at did not change seal identity")
	}
}

func TestCanonicalSealDoesNotPersistStaleState(t *testing.T) {
	encoded, err := EncodeSeal(fixturePayload())
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(bytes.ToLower(encoded), []byte("stale")) {
		t.Fatalf("canonical seal contains derived stale state: %s", encoded)
	}
}

func TestDuplicateDependencyREFIsRejected(t *testing.T) {
	payload := fixturePayload()
	payload.Links[1].TargetREF = payload.Links[0].TargetREF
	if _, err := EncodeSeal(payload); err == nil {
		t.Fatal("duplicate dependency REF was accepted")
	}
}

func TestAttachmentOrderAndDuplicateNamePolicy(t *testing.T) {
	payload := fixturePayload()
	payload.Attachments = []domain.Attachment{
		{Name: "z.txt", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('e')}},
		{Name: "a.txt", MediaType: "text/plain", Blob: domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: fixtureID('f')}},
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
		t.Fatal("attachment input order changed canonical bytes")
	}
	payload.Attachments[1].Name = payload.Attachments[0].Name
	if _, err := EncodeSeal(payload); err == nil {
		t.Fatal("duplicate attachment name was accepted")
	}
}

func TestDecoderRejectsNonCanonicalMemberOrder(t *testing.T) {
	encoded, err := EncodeSeal(fixturePayload())
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(encoded, []byte(`{"schema":"sealgraph/seal/v2","ref":`), []byte(`{"ref":`), 1)
	if bytes.Equal(changed, encoded) {
		t.Fatal("test setup did not alter bytes")
	}
	if _, err := DecodeSeal(changed); err == nil {
		t.Fatal("non-canonical payload was accepted")
	}
}
