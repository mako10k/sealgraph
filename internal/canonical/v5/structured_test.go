package v5

import (
	"bytes"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

func TestStructuredBlobCanonicalBytesAndFixedIDs(t *testing.T) {
	seal := domainv5.Seal{
		Schema:     domainv5.SealSchema,
		Material:   objectID('a'),
		Provenance: objectID('b'),
	}
	material := domainv5.Material{
		Schema:  domainv5.MaterialSchema,
		Content: objectID('c'),
		Attachments: []domainv5.Attachment{
			{Name: "z😀", MediaType: "application/é", Blob: objectID('d')},
			{Name: "a/\x01\"\\é", MediaType: "text/plain\n", Blob: objectID('e')},
		},
	}
	provenance := domainv5.Provenance{
		Schema: domainv5.ProvenanceSchema,
		Root:   false,
		Draft:  true,
		CauseLinks: []domainv5.CauseLink{
			{
				TargetSeal: objectID('b'),
				Messages:   []string{"slash/quote\"back\\line\n"},
			},
			{
				TargetSeal:                       objectID('a'),
				PreviousRevisionSealOfTargetSeal: []domain.ObjectID{objectID('d'), objectID('c')},
				Messages:                         []string{"😀", "", "é"},
			},
		},
	}

	assertCanonicalFixture(t, "seal", seal, EncodeSeal, DecodeSeal,
		`{"schema":"sealgraph/seal/v5","material":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","provenance":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}`,
		"6a11ce8178e0e1c2be6c4e79a86582765606caf8e278226c0aed9219cd3a47a7")
	assertCanonicalFixture(t, "material", material, EncodeMaterial, DecodeMaterial,
		`{"schema":"sealgraph/material/v1","content":"cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","attachments":[{"name":"a/\u0001\"\\é","media_type":"text/plain\n","blob":"eeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"},{"name":"z😀","media_type":"application/é","blob":"dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"}]}`,
		"dc57481bf731525cb67b985947218c381a558bb119a204e297ab7b9efbbb9faf")
	assertCanonicalFixture(t, "provenance", provenance, EncodeProvenance, DecodeProvenance,
		`{"schema":"sealgraph/provenance/v1","root":false,"draft":true,"cause_links":[{"target_seal":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","previous_revision_seal_of_target_seal":["cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc","dddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddddd"],"messages":["","é","😀"]},{"target_seal":"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb","previous_revision_seal_of_target_seal":[],"messages":["slash/quote\"back\\line\n"]}]}`,
		"b35b2e95529e19f02c8293aeabeab4b36d12f1a1d892e5f40652c91e67e95460")
}

func TestDecodeRejectsNoncanonicalOrMistypedStructuredBlobs(t *testing.T) {
	material, err := EncodeMaterial(domainv5.Material{
		Schema:  domainv5.MaterialSchema,
		Content: objectID('a'),
		Attachments: []domainv5.Attachment{
			{Name: "é/", MediaType: "text/plain", Blob: objectID('b')},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string][]byte{
		"trailing LF":        append(append([]byte(nil), material...), '\n'),
		"leading whitespace": append([]byte(" "), material...),
		"escaped slash":      bytes.Replace(material, []byte("é/"), []byte(`é\/`), 1),
		"optional unicode":   bytes.Replace(material, []byte("é/"), []byte(`\u00e9/`), 1),
		"unknown member":     bytes.Replace(material, []byte(`,"attachments":`), []byte(`,"unknown":false,"attachments":`), 1),
		"reordered members":  []byte(`{"content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","schema":"sealgraph/material/v1","attachments":[]}`),
		"null array":         []byte(`{"schema":"sealgraph/material/v1","content":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","attachments":null}`),
		"duplicate member":   bytes.Replace(material, []byte(`{"schema":`), []byte(`{"schema":"sealgraph/material/v1","schema":`), 1),
		"unpaired surrogate": bytes.Replace(material, []byte("é/"), []byte(`\ud800/`), 1),
		"invalid UTF-8":      bytes.Replace(material, []byte("é/"), []byte{0xff, '/'}, 1),
	}
	for name, data := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeMaterial(data); err == nil {
				t.Fatalf("DecodeMaterial accepted %s bytes: %s", name, data)
			}
		})
	}
	if _, err := DecodeSeal(material); err == nil {
		t.Fatal("DecodeSeal accepted a Material Blob")
	}
}

func TestStructuredBlobSemanticValidation(t *testing.T) {
	duplicateAttachment := domainv5.Material{
		Schema:  domainv5.MaterialSchema,
		Content: objectID('a'),
		Attachments: []domainv5.Attachment{
			{Name: "same", Blob: objectID('b')},
			{Name: "same", Blob: objectID('c')},
		},
	}
	if _, err := EncodeMaterial(duplicateAttachment); err == nil || !strings.Contains(err.Error(), "duplicate attachment") {
		t.Fatalf("duplicate attachment error = %v", err)
	}

	link := domainv5.CauseLink{
		TargetSeal:                       objectID('a'),
		PreviousRevisionSealOfTargetSeal: []domain.ObjectID{objectID('b')},
		Messages:                         []string{"basis"},
	}
	for name, provenance := range map[string]domainv5.Provenance{
		"root with Cause Link": {
			Schema: domainv5.ProvenanceSchema, Root: true, CauseLinks: []domainv5.CauseLink{link},
		},
		"non-root without Cause Link": {
			Schema: domainv5.ProvenanceSchema, Root: false,
		},
		"duplicate target": {
			Schema: domainv5.ProvenanceSchema, CauseLinks: []domainv5.CauseLink{link, link},
		},
		"duplicate previous": {
			Schema: domainv5.ProvenanceSchema, CauseLinks: []domainv5.CauseLink{{
				TargetSeal: objectID('a'), PreviousRevisionSealOfTargetSeal: []domain.ObjectID{objectID('b'), objectID('b')},
			}},
		},
		"duplicate message": {
			Schema: domainv5.ProvenanceSchema, CauseLinks: []domainv5.CauseLink{{
				TargetSeal: objectID('a'), Messages: []string{"same", "same"},
			}},
		},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := EncodeProvenance(provenance); err == nil {
				t.Fatalf("EncodeProvenance accepted %s", name)
			}
		})
	}

	if _, err := EncodeSeal(domainv5.Seal{Schema: domainv5.SealSchema}); err == nil {
		t.Fatal("EncodeSeal accepted empty typed IDs")
	}
}

func TestSealIdentityCommitsToMaterialAndProvenanceIdentities(t *testing.T) {
	materialA, err := EncodeMaterial(domainv5.Material{
		Schema: domainv5.MaterialSchema, Content: objectID('a'),
	})
	if err != nil {
		t.Fatal(err)
	}
	materialB, err := EncodeMaterial(domainv5.Material{
		Schema: domainv5.MaterialSchema, Content: objectID('b'),
	})
	if err != nil {
		t.Fatal(err)
	}
	provenance, err := EncodeProvenance(domainv5.Provenance{
		Schema: domainv5.ProvenanceSchema, Root: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	provenanceID := domain.ComputeNativeBlobID(provenance)
	sealA, err := EncodeSeal(domainv5.Seal{
		Schema: domainv5.SealSchema, Material: domain.ComputeNativeBlobID(materialA), Provenance: provenanceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	sealB, err := EncodeSeal(domainv5.Seal{
		Schema: domainv5.SealSchema, Material: domain.ComputeNativeBlobID(materialB), Provenance: provenanceID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if domain.ComputeNativeBlobID(sealA).Equal(domain.ComputeNativeBlobID(sealB)) {
		t.Fatal("different Material identities produced the same Seal identity")
	}
}

func assertCanonicalFixture[T any](
	t *testing.T,
	name string,
	value T,
	encode func(T) ([]byte, error),
	decode func([]byte) (T, error),
	wantBytes string,
	wantID string,
) {
	t.Helper()
	encoded, err := encode(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != wantBytes {
		t.Fatalf("%s bytes differ:\n got: %s\nwant: %s", name, encoded, wantBytes)
	}
	if _, err := decode(encoded); err != nil {
		t.Fatalf("decode canonical %s: %v", name, err)
	}
	if got := domain.ComputeNativeBlobID(encoded).String(); got != wantID {
		t.Fatalf("%s BlobID = %s, want %s", name, got, wantID)
	}
}

func objectID(fill byte) domain.ObjectID {
	return domain.ObjectID{Hex: strings.Repeat(string(fill), 64)}
}
