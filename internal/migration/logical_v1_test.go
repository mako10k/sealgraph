package migration

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func migrationID(char byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(char), 64)}
}

func logicalDumpFixture(t *testing.T) LogicalDumpV1 {
	t.Helper()
	content := []byte{'x', 0, '\r', '\n', 0xff}
	contentID := domain.ComputeNativeBlobID(content)
	root := Format3SealPayload{
		Schema: Format3SealSchema, REF: "ROOT",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Attachments: []domain.Attachment{}, Links: []Format3Link{}, Root: true,
	}
	rootBytes, err := encodeFormat3Seal(root)
	if err != nil {
		t.Fatal(err)
	}
	rootID := domain.ComputeNativeBlobID(rootBytes)
	child := Format3SealPayload{
		Schema: Format3SealSchema, REF: "CHILD",
		Content:     domain.ContentRef{Store: domain.NativeStore, Type: domain.BlobType, ID: contentID},
		Attachments: []domain.Attachment{},
		Links:       []Format3Link{{TargetREF: "ROOT", TargetSeal: rootID, Message: "basis"}},
	}
	childBytes, err := encodeFormat3Seal(child)
	if err != nil {
		t.Fatal(err)
	}
	childID := domain.ComputeNativeBlobID(childBytes)
	return LogicalDumpV1{
		Objects: []ObjectRecord{{ID: contentID, Data: content}},
		Seals:   []SealRecord{{OldSealID: rootID, Payload: root}, {OldSealID: childID, Payload: child}},
		REFs:    []RefRecord{{Name: "CHILD", Head: childID}, {Name: "ROOT", Head: rootID}},
		Tags:    []TagRecord{}, ExcludedObjects: []domain.ObjectID{migrationID('f')},
	}
}

func TestLogicalDumpV1CanonicalRoundTrip(t *testing.T) {
	dump := logicalDumpFixture(t)
	encoded, err := EncodeLogicalDumpV1(dump)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeLogicalDumpV1(encoded)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := EncodeLogicalDumpV1(decoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(encoded, reencoded) || encoded[len(encoded)-1] != '\n' {
		t.Fatal("logical dump did not round-trip exactly")
	}
	if !bytes.Equal(decoded.Objects[0].Data, dump.Objects[0].Data) {
		t.Fatal("binary material changed")
	}
}

func TestLogicalDumpV1RejectsNoncanonicalAndMissingMappingOrder(t *testing.T) {
	encoded, err := EncodeLogicalDumpV1(logicalDumpFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeLogicalDumpV1(bytes.TrimSuffix(encoded, []byte{'\n'})); err == nil {
		t.Fatal("dump without required trailing LF was accepted")
	}
	dump := logicalDumpFixture(t)
	dump.Seals[0], dump.Seals[1] = dump.Seals[1], dump.Seals[0]
	if _, err := EncodeLogicalDumpV1(dump); err == nil {
		t.Fatal("dependent-before-dependency dump was accepted")
	}
}

func TestLogicalDumpV1RejectsUnknownMember(t *testing.T) {
	encoded, _ := EncodeLogicalDumpV1(logicalDumpFixture(t))
	changed := bytes.Replace(encoded, []byte(`{"schema":"sealgraph/logical-dump/v1"`), []byte(`{"schema":"sealgraph/logical-dump/v1","unknown":true`), 1)
	if _, err := DecodeLogicalDumpV1(changed); err == nil {
		t.Fatal("unknown logical dump member was accepted")
	}
}
