package v5

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func TestFormat5CandidateExactBytesAndFixedFixtureHash(t *testing.T) {
	expected := objectID('e')
	candidate := domainv5.Candidate{
		Schema:          domainv5.CandidateSchema,
		REF:             "design/api",
		ExpectedREFHead: &expected,
		Content:         objectID('a'),
		Attachments: []domainv5.Attachment{
			{Name: "z", MediaType: "application/octet-stream", Blob: objectID('d')},
			{Name: "a", MediaType: "text/plain", Blob: objectID('c')},
		},
		Root:  false,
		Draft: true,
		CauseLinks: []domainv5.CauseLink{
			{
				TargetSeal:                       objectID('b'),
				PreviousRevisionSealOfTargetSeal: []domain.ObjectID{objectID('d'), objectID('c')},
				Messages:                         []string{"z", "a"},
			},
		},
	}
	encoded, err := EncodeCandidate(candidate)
	if err != nil {
		t.Fatal(err)
	}
	const want = `{"schema":"sealgraph/candidate/v5","ref":"design/api","expected_ref_head":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee","content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","attachments":[{"name":"a","media_type":"text/plain","blob":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"},{"name":"z","media_type":"application/octet-stream","blob":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}],"root":false,"draft":true,"cause_links":[{"target_seal":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","previous_revision_seal_of_target_seal":["cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"],"messages":["a","z"]}]}
`
	if string(encoded) != want {
		t.Fatalf("candidate bytes differ:\n got: %s\nwant: %s", encoded, want)
	}
	const wantHash = "10e9e48cf9353bc24cf3c1e21a5f28ff2ff4713db9cc4304fd977e1391201559"
	if got := domain.ComputeNativeBlobID(encoded).String(); got != wantHash {
		t.Fatalf("candidate fixture byte hash = %s, want %s", got, wantHash)
	}
	decoded, err := DecodeCandidate(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !decoded.Attachments[0].Blob.Equal(objectID('c')) || decoded.CauseLinks[0].Messages[0] != "a" {
		t.Fatalf("candidate arrays were not canonical: %+v", decoded)
	}
}

func TestFormat5CandidateRequiresExactWriterBytes(t *testing.T) {
	candidate := domainv5.Candidate{
		Schema:  domainv5.CandidateSchema,
		REF:     "root",
		Content: objectID('a'),
		Root:    true,
	}
	canonicalBytes, err := EncodeCandidate(candidate)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"missing LF":         bytes.TrimSuffix(canonicalBytes, []byte{'\n'}),
		"extra LF":           append(append([]byte(nil), canonicalBytes...), '\n'),
		"leading space":      append([]byte(" "), canonicalBytes...),
		"unknown parent":     bytes.Replace(canonicalBytes, []byte(`,"expected_ref_head":`), []byte(`,"parent_revision":null,"expected_ref_head":`), 1),
		"null attachments":   bytes.Replace(canonicalBytes, []byte(`"attachments":[]`), []byte(`"attachments":null`), 1),
		"null Cause Links":   bytes.Replace(canonicalBytes, []byte(`"cause_links":[]`), []byte(`"cause_links":null`), 1),
		"noncanonical REF":   bytes.Replace(canonicalBytes, []byte(`"ref":"root"`), []byte(`"ref":"bad ref"`), 1),
		"missing root field": bytes.Replace(canonicalBytes, []byte(`,"root":true`), nil, 1),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeCandidate(data); err == nil {
				t.Fatalf("DecodeCandidate accepted %s bytes: %s", name, data)
			}
		})
	}
}

func TestFormat5CandidateSemanticInvariants(t *testing.T) {
	link := domainv5.CauseLink{TargetSeal: objectID('b')}
	cases := map[string]domainv5.Candidate{
		"root with Cause Link": {
			Schema: domainv5.CandidateSchema, REF: "root", Content: objectID('a'), Root: true, CauseLinks: []domainv5.CauseLink{link},
		},
		"non-root without Cause Link": {
			Schema: domainv5.CandidateSchema, REF: "child", Content: objectID('a'), Root: false,
		},
		"invalid expected head": {
			Schema: domainv5.CandidateSchema, REF: "root", Content: objectID('a'), Root: true, ExpectedREFHead: &domain.ObjectID{Hex: "short"},
		},
	}
	for name, candidate := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := EncodeCandidate(candidate); err == nil {
				t.Fatalf("EncodeCandidate accepted %s", name)
			}
		})
	}

	encoded, err := EncodeCandidate(domainv5.Candidate{
		Schema:  domainv5.CandidateSchema,
		REF:     "root",
		Content: objectID('a'),
		Root:    true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"expected_ref_head":null`) {
		t.Fatalf("expected-absent Candidate did not encode null: %s", encoded)
	}
}
