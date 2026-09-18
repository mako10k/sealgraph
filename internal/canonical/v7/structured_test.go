package v7

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv7 "github.com/mako10k/sealgraph/internal/domain/v7"
)

func testID(c byte) domain.ObjectID { return domain.ObjectID{Hex: strings.Repeat(string(c), 64)} }

func TestOriginMapCanonicalBytesAndRoundTrip(t *testing.T) {
	v := domainv7.OriginMap{
		Schema: domainv7.OriginMapSchema, Content: testID('c'),
		Runs: []domainv7.OriginRun{
			{Kind: "external", Length: 2, Snapshot: testID('a'), SourceStart: 7},
			{Kind: "untraced", Length: 3},
		},
	}
	got, err := EncodeOriginMap(v)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"sealgraph/origin-map/v1","content":"` + testID('c').String() + `","runs":[{"kind":"external","length":2,"snapshot":"` + testID('a').String() + `","source_start":7},{"kind":"untraced","length":3}]}`
	if string(got) != want {
		t.Fatalf("bytes\n got %s\nwant %s", got, want)
	}
	decoded, err := DecodeOriginMap(got)
	if err != nil {
		t.Fatal(err)
	}
	if id := domain.ComputeNativeBlobID(got).String(); id != "077cd333f977f421dab6d36867d1df2764fd2cf7741e23942d481f1f81d2eac0" {
		t.Fatalf("OriginMap canonical ID=%s", id)
	}
	reencoded, err := EncodeOriginMap(decoded)
	if err != nil || !bytes.Equal(got, reencoded) {
		t.Fatalf("round trip err=%v", err)
	}
}

func TestSourceSnapshotCanonicalID(t *testing.T) {
	encoded, err := EncodeSourceSnapshot(domainv7.SourceSnapshot{Schema: domainv7.SourceSnapshotSchema, SourceKey: "source-A", Content: testID('c')})
	if err != nil {
		t.Fatal(err)
	}
	if id := domain.ComputeNativeBlobID(encoded).String(); id != "12b732a69208f7a9d7c4a7f43f6c83f14561ceb0fa75a963ec7577dc3802fb60" {
		t.Fatalf("SourceSnapshot canonical ID=%s", id)
	}
}

func TestFormat7ProvenanceAndCandidateCanonicalBytes(t *testing.T) {
	origin := testID('d')
	p := domainv7.Provenance{Schema: domainv7.ProvenanceSchema, Root: true, Origin: &origin}
	got, err := EncodeProvenance(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"schema":"sealgraph/provenance/v3","root":true,"draft":false,"cause_links":[],"origin":"` + origin.String() + `"}`
	if string(got) != want {
		t.Fatalf("provenance bytes\n got %s\nwant %s", got, want)
	}
	c := domainv7.Candidate{Schema: domainv7.CandidateSchema, REF: "refs/main", Content: testID('c'), Root: true, Origin: &origin}
	got, err = EncodeCandidate(c)
	if err != nil {
		t.Fatal(err)
	}
	want = `{"schema":"sealgraph/candidate/v7","ref":"refs/main","expected_ref_head":null,"content":"` + testID('c').String() + `","attachments":[],"root":true,"draft":false,"cause_links":[],"origin":"` + origin.String() + `"}`
	if string(got) != want {
		t.Fatalf("candidate bytes\n got %s\nwant %s", got, want)
	}
	if got[len(got)-1] == '\n' {
		t.Fatal("v7 candidate has trailing LF")
	}
	if _, err := DecodeCandidate(got); err != nil {
		t.Fatal(err)
	}
}

func TestFormat7RejectsMalformedRecords(t *testing.T) {
	valid := `{"schema":"sealgraph/origin-map/v1","content":"` + testID('c').String() + `","runs":[]}`
	for name, input := range map[string]string{
		"unknown":           strings.Replace(valid, `,"runs"`, `,"extra":1,"runs"`, 1),
		"duplicate":         strings.Replace(valid, `,"runs"`, `,"content":"`+testID('c').String()+`","runs"`, 1),
		"leading zero":      strings.Replace(valid, `[]}`, `[{"kind":"untraced","length":01}]}`, 1),
		"negative":          strings.Replace(valid, `[]}`, `[{"kind":"untraced","length":-1}]}`, 1),
		"exponent":          strings.Replace(valid, `[]}`, `[{"kind":"untraced","length":1e1}]}`, 1),
		"overflow":          strings.Replace(valid, `[]}`, `[{"kind":"untraced","length":18446744073709551616}]}`, 1),
		"adjacent untraced": strings.Replace(valid, `[]}`, `[{"kind":"untraced","length":1},{"kind":"untraced","length":1}]}`, 1),
		"external overflow": strings.Replace(valid, `[]}`, `[{"kind":"external","length":2,"snapshot":"`+testID('a').String()+`","source_start":18446744073709551615}]}`, 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeOriginMap([]byte(input)); err == nil {
				t.Fatalf("accepted malformed %s", name)
			}
		})
	}
	if _, err := DecodeOriginMap([]byte(" {" + strings.TrimPrefix(valid, "{"))); err == nil {
		t.Fatal("accepted noncanonical whitespace")
	}
}

func TestFormat7RejectsUnknownAndDuplicateMembersNested(t *testing.T) {
	base := `{"schema":"sealgraph/origin-map/v1","content":"` + testID('c').String() + `","runs":[{"kind":"untraced","length":1}]}`
	for _, bad := range []string{
		strings.Replace(base, `,"length":1`, `,"length":1,"length":1`, 1),
		strings.Replace(base, `,"length":1`, `,"bogus":0,"length":1`, 1),
	} {
		if _, err := DecodeOriginMap([]byte(bad)); err == nil {
			t.Fatalf("accepted nested malformed: %s", bad)
		}
	}
}

func TestFormat7PreservesCauseLinkMetadata(t *testing.T) {
	link := domainv7.CauseLink{
		TargetSeal:                       testID('a'),
		PreviousRevisionSealOfTargetSeal: []domain.ObjectID{},
		Messages:                         []string{"reason"},
		Metadata:                         []domainv7.MetadataEntry{{Namespace: "example.test/context", Value: json.RawMessage(`{"role":"source"}`)}},
	}
	p := domainv7.Provenance{Schema: domainv7.ProvenanceSchema, CauseLinks: []domainv7.CauseLink{link}}
	encoded, err := EncodeProvenance(p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(`"metadata":[{"namespace":"example.test/context"`)) {
		t.Fatalf("metadata missing from Provenance v3: %s", encoded)
	}
	decoded, err := DecodeProvenance(encoded)
	if err != nil || len(decoded.CauseLinks[0].Metadata) != 1 {
		t.Fatalf("metadata round trip: %+v, %v", decoded, err)
	}
	withoutMetadata := bytes.Replace(encoded, []byte(`,"metadata":[{"namespace":"example.test/context","schema":null,"value":{"role":"source"}}]`), nil, 1)
	if bytes.Equal(encoded, withoutMetadata) {
		t.Fatal("test did not remove metadata")
	}
	if _, err := DecodeProvenance(withoutMetadata); err == nil {
		t.Fatal("accepted v7 Cause Link without required metadata member")
	}
}
