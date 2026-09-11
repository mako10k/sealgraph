package v6

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv6 "github.com/mako10k/sealgraph/internal/domain/v6"
)

func oid(character byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(character), 64)}
}
func raw(value string) json.RawMessage { return json.RawMessage(value) }

func TestFormat6CanonicalMetadataFixture(t *testing.T) {
	schema := "urn:test:v1"
	value := domainv6.Provenance{Schema: domainv6.ProvenanceSchema, Root: false, CauseLinks: []domainv6.CauseLink{{
		TargetSeal: oid('a'), PreviousRevisionSealOfTargetSeal: []domain.ObjectID{oid('c')}, Messages: []string{"z", "a"},
		Metadata: []domainv6.MetadataEntry{{Namespace: "z", Value: raw(`[true,null,-2]`)}, {Namespace: "a", Schema: &schema, Value: raw(`{"z":1,"a":"x"}`)}},
	}}}
	got, err := EncodeProvenance(value)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"sealgraph/provenance/v2","root":false,"draft":false,"cause_links":[{"target_seal":"` + oid('a').String() + `","previous_revision_seal_of_target_seal":["` + oid('c').String() + `"],"messages":["a","z"],"metadata":[{"namespace":"a","schema":"urn:test:v1","value":{"a":"x","z":1}},{"namespace":"z","schema":null,"value":[true,null,-2]}]}]}`
	if string(got) != want {
		t.Fatalf("canonical bytes\n got %s\nwant %s", got, want)
	}
	decoded, err := DecodeProvenance(got)
	if err != nil {
		t.Fatal(err)
	}
	reencoded, err := EncodeProvenance(decoded)
	if err != nil || !bytes.Equal(got, reencoded) {
		t.Fatalf("round trip err=%v", err)
	}
}

func TestFormat6MetadataRejectsNoncanonicalAndBoundaries(t *testing.T) {
	base := func(value string) domainv6.Provenance {
		return domainv6.Provenance{Schema: domainv6.ProvenanceSchema, CauseLinks: []domainv6.CauseLink{{TargetSeal: oid('a'), Metadata: []domainv6.MetadataEntry{{Namespace: "n", Value: raw(value)}}}}}
	}
	for _, invalid := range []string{`1.0`, `1e0`, `-0`, `9223372036854775808`, `{"a":1,"\u0061":2}`, `"` + strings.Repeat("x", 4097) + `"`} {
		if _, err := EncodeProvenance(base(invalid)); err == nil {
			t.Fatalf("accepted invalid value %s", invalid)
		}
	}
	depth := "0"
	for i := 0; i < 16; i++ {
		depth = "[" + depth + "]"
	}
	if _, err := EncodeProvenance(base(depth)); err == nil || !strings.Contains(err.Error(), "depth") {
		t.Fatalf("depth err=%v", err)
	}
	entries := make([]domainv5.MetadataEntry, 65)
	for i := range entries {
		entries[i] = domainv5.MetadataEntry{Namespace: string(rune('A' + i)), Value: raw("null")}
	}
	value := base("null")
	value.CauseLinks[0].Metadata = entries
	if _, err := EncodeProvenance(value); err == nil || !strings.Contains(err.Error(), "maximum") {
		t.Fatalf("entry limit err=%v", err)
	}
}

func TestFormat6MetadataAcceptsExactResourceBoundaries(t *testing.T) {
	base := func(entries []domainv5.MetadataEntry) domainv6.Provenance {
		return domainv6.Provenance{Schema: domainv6.ProvenanceSchema, CauseLinks: []domainv6.CauseLink{{TargetSeal: oid('a'), Metadata: entries}}}
	}
	entries := metadataEntries(64, 1)
	if _, err := EncodeProvenance(base(entries)); err != nil {
		t.Fatalf("64 entries: %v", err)
	}
	namespace := strings.Repeat("n", 255)
	schema := strings.Repeat("s", 1024)
	if _, err := EncodeProvenance(base([]domainv5.MetadataEntry{{Namespace: namespace, Schema: &schema, Value: raw(`"` + strings.Repeat("x", 4096) + `"`)}})); err != nil {
		t.Fatalf("string boundaries: %v", err)
	}
	depth := "null"
	for i := 1; i < 16; i++ {
		depth = "[" + depth + "]"
	}
	if _, err := EncodeProvenance(base([]domainv5.MetadataEntry{{Namespace: "depth", Value: raw(depth)}})); err != nil {
		t.Fatalf("depth 16: %v", err)
	}
	nodes := "[" + strings.TrimSuffix(strings.Repeat("null,", 4095), ",") + "]"
	if _, err := EncodeProvenance(base([]domainv5.MetadataEntry{{Namespace: "nodes", Value: raw(nodes)}})); err != nil {
		t.Fatalf("4096 nodes: %v", err)
	}
	metadataAtLimit := metadataEntriesAtByteLimit(t)
	if _, encoded, err := normalizeMetadata(metadataAtLimit); err != nil || len(encoded) != maxMetadataBytes {
		t.Fatalf("metadata bytes=%d err=%v", len(encoded), err)
	}
	if _, err := EncodeProvenance(base(metadataAtLimit)); err != nil {
		t.Fatalf("65536 metadata bytes: %v", err)
	}
}

func TestFormat6MetadataRejectsOnePastResourceBoundaries(t *testing.T) {
	encode := func(entries []domainv5.MetadataEntry) error {
		_, err := EncodeProvenance(domainv6.Provenance{Schema: domainv6.ProvenanceSchema, CauseLinks: []domainv6.CauseLink{{TargetSeal: oid('a'), Metadata: entries}}})
		return err
	}
	tooLongNamespace := []domainv5.MetadataEntry{{Namespace: strings.Repeat("n", 256), Value: raw("null")}}
	if err := encode(tooLongNamespace); err == nil {
		t.Fatal("accepted 256-byte namespace")
	}
	schema := strings.Repeat("s", 1025)
	if err := encode([]domainv5.MetadataEntry{{Namespace: "n", Schema: &schema, Value: raw("null")}}); err == nil {
		t.Fatal("accepted 1025-byte schema")
	}
	nodes := "[" + strings.TrimSuffix(strings.Repeat("null,", 4096), ",") + "]"
	if err := encode([]domainv5.MetadataEntry{{Namespace: "nodes", Value: raw(nodes)}}); err == nil {
		t.Fatal("accepted 4097 nodes")
	}
	limit := metadataEntriesAtByteLimit(t)
	last := len(limit) - 1
	limit[last].Value = raw(`"` + strings.Repeat("x", len(limit[last].Value)-1) + `"`)
	if err := encode(limit); err == nil {
		t.Fatal("accepted metadata array larger than 65536 bytes")
	}
}

func metadataEntries(count, valueLength int) []domainv5.MetadataEntry {
	entries := make([]domainv5.MetadataEntry, count)
	for i := range entries {
		entries[i] = domainv5.MetadataEntry{Namespace: fmt.Sprintf("n%02d", i), Value: raw(`"` + strings.Repeat("x", valueLength) + `"`)}
	}
	return entries
}

func metadataEntriesAtByteLimit(t *testing.T) []domainv5.MetadataEntry {
	t.Helper()
	entries := metadataEntries(16, 4096)
	entries[len(entries)-1].Value = raw(`""`)
	_, encoded, err := normalizeMetadata(entries)
	if err != nil {
		t.Fatal(err)
	}
	lastLength := maxMetadataBytes - len(encoded)
	if lastLength < 0 || lastLength > 4096 {
		t.Fatalf("cannot construct byte boundary: %d", lastLength)
	}
	entries[len(entries)-1].Value = raw(`"` + strings.Repeat("x", lastLength) + `"`)
	return entries
}

func TestFormat6DecoderRequiresStoredMetadataAndCanonicalValue(t *testing.T) {
	missing := `{"schema":"sealgraph/provenance/v2","root":false,"draft":false,"cause_links":[{"target_seal":"` + oid('a').String() + `","previous_revision_seal_of_target_seal":[],"messages":[]}]}`
	if _, err := DecodeProvenance([]byte(missing)); err == nil {
		t.Fatal("accepted missing metadata")
	}
	noncanonical := strings.Replace(missing, `"messages":[]}`, `"messages":[],"metadata":[{"namespace":"n","schema":null,"value":{"z":1,"a":2}}]}`, 1)
	if _, err := DecodeProvenance([]byte(noncanonical)); err == nil {
		t.Fatal("accepted noncanonical object key order")
	}
}

func TestUpstreamChangeV2IdentityIncludesMetadata(t *testing.T) {
	base := domainv6.UpstreamChange{Schema: domainv6.UpstreamChangeSchema, AfterMaterial: oid('b'), AfterCauseLinks: []domainv6.CauseLink{{TargetSeal: oid('a'), Metadata: []domainv6.MetadataEntry{}}}}
	empty, err := EncodeUpstreamChange(base)
	if err != nil {
		t.Fatal(err)
	}
	base.AfterCauseLinks[0].Metadata = []domainv6.MetadataEntry{{Namespace: "n", Value: raw("true")}}
	withMetadata, err := EncodeUpstreamChange(base)
	if err != nil {
		t.Fatal(err)
	}
	if domain.ComputeNativeBlobID(empty).Equal(domain.ComputeNativeBlobID(withMetadata)) {
		t.Fatal("metadata did not change upstream-change/v2 identity")
	}
	if _, err := DecodeUpstreamChange(withMetadata); err != nil {
		t.Fatal(err)
	}
}

func TestMaterialV1SharedCodecRoundTrip(t *testing.T) {
	value := domainv6.Material{Schema: domainv6.MaterialSchema, Content: oid('a'), Attachments: []domainv6.Attachment{}}
	encoded, err := EncodeMaterial(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeMaterial(encoded); err != nil {
		t.Fatal(err)
	}
}
