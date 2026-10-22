package historicalreview

// R: verify canonical review evidence identity and reject ambiguous claim input.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func fixtureRecord() Record {
	a := domain.ObjectID{Hex: strings.Repeat("a", 64)}
	b := domain.ObjectID{Hex: strings.Repeat("b", 64)}
	return Record{Schema: Schema, REF: "design/api", SubjectSeal: a, OtherHeads: []Head{{REF: "requirements/api", Seal: b}}, Entries: []Entry{{TargetSeal: a, ComparedSeals: []domain.ObjectID{b}, Decision: "retain", Difference: "The API changed.", Impact: "The dependent implements the old API.", Reason: "The old API is the selected delivery scope."}}}
}

func TestRecordCanonicalIdentityAndStrictInput(t *testing.T) {
	record := fixtureRecord()
	data, err := Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	formatted, _ := json.MarshalIndent(record, "", "  ")
	_, again, err := Decode(formatted)
	if err != nil || !bytes.Equal(data, again) {
		t.Fatalf("normalization=%s err=%v", again, err)
	}
	digest := fmt.Sprintf("%x", sha256.Sum256(data))
	const want = "bcbf0f1da8ca87482daf885dc1bad0aabaedb8a935320d697cbbcc7f2d314107"
	if digest != want {
		t.Fatalf("canonical review digest=%s", digest)
	}
}

func TestRecordRejectsAmbiguousOrIncompleteClaims(t *testing.T) {
	data, _ := Encode(fixtureRecord())
	cases := map[string][]byte{
		"null-text":      bytes.Replace(bytes.Replace(data, []byte(`"decision":"retain"`), []byte(`"decision":"unreviewed"`), 1), []byte(`"impact":"The dependent implements the old API."`), []byte(`"impact":null`), 1),
		"case-alias":     bytes.Replace(data, []byte(`"decision":"retain"`), []byte(`"Decision":"retain"`), 1),
		"missing-field":  bytes.Replace(data, []byte(`"impact":"The dependent implements the old API.",`), nil, 1),
		"duplicate-key":  bytes.Replace(data, []byte(`"decision":"retain"`), []byte(`"decision":"unreviewed","decision":"retain"`), 1),
		"unknown-member": bytes.Replace(data, []byte(`"decision":"retain"`), []byte(`"decision":"retain","approved":true`), 1),
		"blank-reason":   bytes.Replace(data, []byte(`"reason":"The old API is the selected delivery scope."`), []byte(`"reason":"  "`), 1),
		"prefix-id":      bytes.Replace(data, []byte(strings.Repeat("a", 64)), []byte("aaaa"), 1),
		"trailing":       append(append([]byte{}, data...), []byte(`{}`)...),
		"oversized":      bytes.Repeat([]byte(" "), MaxBytes+1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, _, err := Decode(input); err == nil {
				t.Fatal("invalid record accepted")
			}
		})
	}
	record := fixtureRecord()
	record.Entries = append(record.Entries, record.Entries[0])
	if _, err := Encode(record); err == nil {
		t.Fatal("duplicate target accepted")
	}
	record = fixtureRecord()
	record.OtherHeads = append(record.OtherHeads, record.OtherHeads[0])
	if _, err := Encode(record); err == nil {
		t.Fatal("duplicate head accepted")
	}
	record = fixtureRecord()
	record.Entries[0].ComparedSeals = append(record.Entries[0].ComparedSeals, record.Entries[0].ComparedSeals[0])
	if _, err := Encode(record); err == nil {
		t.Fatal("duplicate comparison accepted")
	}
}

func TestRecordUnreviewedIsPersistableWithoutInventedReason(t *testing.T) {
	record := fixtureRecord()
	record.Entries[0].Decision = "unreviewed"
	record.Entries[0].Difference = ""
	record.Entries[0].Impact = ""
	record.Entries[0].Reason = ""
	data, err := Encode(record)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := Decode(data)
	if err != nil || decoded.Entries[0].Decision != "unreviewed" {
		t.Fatalf("record=%+v err=%v", decoded, err)
	}
}
