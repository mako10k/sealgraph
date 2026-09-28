package migration

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

func TestNativeSnapshotV1EncodeDecodeCanonical(t *testing.T) {
	blobData := []byte("blob bytes")
	value := NativeSnapshotV1{
		Blobs:      []NativeBlobRecord{{ID: domain.ComputeNativeBlobID(blobData), Data: blobData}},
		REFs:       []NativeRefRecord{{REF: "refs/main", Manifest: []byte("manifest")}},
		Candidates: []NativeCandidateRecord{{REF: "refs/main", Candidate: []byte("candidate")}},
	}
	want := fmt.Sprintf(`{"schema":"sealgraph/native-snapshot/v1","repository_format":7,"object_format":"sha256","ref_format":"manifest-v1","blobs":[{"id":"%s","data_base64":"YmxvYiBieXRlcw=="}],"refs":[{"ref":"refs/main","manifest_base64":"bWFuaWZlc3Q="}],"candidates":[{"ref":"refs/main","candidate_base64":"Y2FuZGlkYXRl"}]}`+"\n", value.Blobs[0].ID)
	encoded, err := EncodeNativeSnapshotV1(value)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != want {
		t.Fatalf("encoded bytes mismatch:\n got %q\nwant %q", encoded, want)
	}
	decoded, err := DecodeNativeSnapshotV1(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(decoded.Blobs[0].Data, blobData) || decoded.REFs[0].REF != "refs/main" || !bytes.Equal(decoded.Candidates[0].Candidate, []byte("candidate")) {
		t.Fatalf("decoded value mismatch: %#v", decoded)
	}
}

func TestNativeSnapshotV1RejectsInvalidInput(t *testing.T) {
	id := domain.ComputeNativeBlobID([]byte("x"))
	valid := fmt.Sprintf(`{"schema":"sealgraph/native-snapshot/v1","repository_format":7,"object_format":"sha256","ref_format":"manifest-v1","blobs":[{"id":"%s","data_base64":"eA=="}],"refs":[],"candidates":[]}`+"\n", id)
	cases := map[string]string{
		"hash mismatch":           strings.Replace(valid, "eA==", "eQ==", 1),
		"duplicate field":         strings.Replace(valid, `"refs":[]`, `"refs":[],"refs":[]`, 1),
		"unknown field":           strings.Replace(valid, `"refs":[]`, `"extra":0,"refs":[]`, 1),
		"noncanonical base64":     strings.Replace(valid, "eA==", "eA", 1),
		"missing field":           strings.Replace(valid, `,"candidates":[]`, "", 1),
		"noncanonical whitespace": " " + valid,
		"invalid REF":             strings.Replace(valid, `"refs":[]`, `"refs":[{"ref":"bad@ref","manifest_base64":""}]`, 1),
	}
	for name, input := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeNativeSnapshotV1([]byte(input)); err == nil {
				t.Fatal("DecodeNativeSnapshotV1 accepted invalid input")
			}
		})
	}
}

func TestNativeSnapshotV1EncodeRequiresSortedUniqueRecords(t *testing.T) {
	dataA, dataB := []byte("a"), []byte("b")
	idA, idB := domain.ComputeNativeBlobID(dataA), domain.ComputeNativeBlobID(dataB)
	if idA.String() > idB.String() {
		idA, idB, dataA, dataB = idB, idA, dataB, dataA
	}
	for name, value := range map[string]NativeSnapshotV1{
		"unsorted blobs":      {Blobs: []NativeBlobRecord{{ID: idB, Data: dataB}, {ID: idA, Data: dataA}}},
		"duplicate refs":      {REFs: []NativeRefRecord{{REF: "refs/a"}, {REF: "refs/a"}}},
		"unsorted candidates": {Candidates: []NativeCandidateRecord{{REF: "refs/b"}, {REF: "refs/a"}}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := EncodeNativeSnapshotV1(value); err == nil {
				t.Fatal("EncodeNativeSnapshotV1 accepted noncanonical ordering")
			}
		})
	}
}
