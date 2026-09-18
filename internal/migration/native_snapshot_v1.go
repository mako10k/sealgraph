package migration

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/mako10k/sealgraph/internal/domain"
)

const NativeSnapshotV1Schema = "sealgraph/native-snapshot/v1"

type NativeBlobRecord struct {
	ID   domain.ObjectID
	Data []byte
}

type NativeRefRecord struct {
	REF      string
	Manifest []byte
}

type NativeCandidateRecord struct {
	REF       string
	Candidate []byte
}

type NativeSnapshotV1 struct {
	Blobs      []NativeBlobRecord
	REFs       []NativeRefRecord
	Candidates []NativeCandidateRecord
}

type nativeSnapshotBlobWire struct {
	ID         string `json:"id"`
	DataBase64 string `json:"data_base64"`
}

type nativeSnapshotRefWire struct {
	REF            string `json:"ref"`
	ManifestBase64 string `json:"manifest_base64"`
}

type nativeSnapshotCandidateWire struct {
	REF             string `json:"ref"`
	CandidateBase64 string `json:"candidate_base64"`
}

type nativeSnapshotWire struct {
	Schema           string                        `json:"schema"`
	RepositoryFormat int                           `json:"repository_format"`
	ObjectFormat     string                        `json:"object_format"`
	REFFormat        string                        `json:"ref_format"`
	Blobs            []nativeSnapshotBlobWire      `json:"blobs"`
	REFs             []nativeSnapshotRefWire       `json:"refs"`
	Candidates       []nativeSnapshotCandidateWire `json:"candidates"`
}

func EncodeNativeSnapshotV1(value NativeSnapshotV1) ([]byte, error) {
	if err := validateNativeSnapshotV1(value); err != nil {
		return nil, err
	}
	wire := nativeSnapshotWire{
		Schema: NativeSnapshotV1Schema, RepositoryFormat: 7,
		ObjectFormat: "sha256", REFFormat: "manifest-v1",
		Blobs:      make([]nativeSnapshotBlobWire, len(value.Blobs)),
		REFs:       make([]nativeSnapshotRefWire, len(value.REFs)),
		Candidates: make([]nativeSnapshotCandidateWire, len(value.Candidates)),
	}
	for i, blob := range value.Blobs {
		wire.Blobs[i] = nativeSnapshotBlobWire{ID: blob.ID.String(), DataBase64: base64.StdEncoding.EncodeToString(blob.Data)}
	}
	for i, ref := range value.REFs {
		wire.REFs[i] = nativeSnapshotRefWire{REF: ref.REF, ManifestBase64: base64.StdEncoding.EncodeToString(ref.Manifest)}
	}
	for i, candidate := range value.Candidates {
		wire.Candidates[i] = nativeSnapshotCandidateWire{REF: candidate.REF, CandidateBase64: base64.StdEncoding.EncodeToString(candidate.Candidate)}
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("encode native-snapshot-v1: %w", err)
	}
	return append(data, '\n'), nil
}

func DecodeNativeSnapshotV1(data []byte) (NativeSnapshotV1, error) {
	if err := scanJSONForDuplicateFields(data); err != nil {
		return NativeSnapshotV1{}, fmt.Errorf("decode native-snapshot-v1: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var wire nativeSnapshotWire
	if err := decoder.Decode(&wire); err != nil {
		return NativeSnapshotV1{}, fmt.Errorf("decode native-snapshot-v1: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return NativeSnapshotV1{}, fmt.Errorf("decode native-snapshot-v1: trailing JSON value")
	}
	value, err := nativeSnapshotFromWire(wire)
	if err != nil {
		return NativeSnapshotV1{}, err
	}
	encoded, err := EncodeNativeSnapshotV1(value)
	if err != nil {
		return NativeSnapshotV1{}, err
	}
	if !bytes.Equal(data, encoded) {
		return NativeSnapshotV1{}, fmt.Errorf("native-snapshot-v1 document is not canonical")
	}
	return value, nil
}

func nativeSnapshotFromWire(wire nativeSnapshotWire) (NativeSnapshotV1, error) {
	if wire.Schema != NativeSnapshotV1Schema || wire.RepositoryFormat != 7 || wire.ObjectFormat != "sha256" || wire.REFFormat != "manifest-v1" {
		return NativeSnapshotV1{}, fmt.Errorf("unsupported native-snapshot-v1 configuration")
	}
	value := NativeSnapshotV1{
		Blobs:      make([]NativeBlobRecord, len(wire.Blobs)),
		REFs:       make([]NativeRefRecord, len(wire.REFs)),
		Candidates: make([]NativeCandidateRecord, len(wire.Candidates)),
	}
	for i, item := range wire.Blobs {
		id, err := domain.ParseObjectID(item.ID)
		if err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("blob %d: %w", i, err)
		}
		decoded, err := decodeNativeSnapshotBase64(item.DataBase64)
		if err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("blob %s data_base64: %w", id, err)
		}
		value.Blobs[i] = NativeBlobRecord{ID: id, Data: decoded}
	}
	for i, item := range wire.REFs {
		if err := domain.ValidateREF(item.REF); err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("ref %d: %w", i, err)
		}
		decoded, err := decodeNativeSnapshotBase64(item.ManifestBase64)
		if err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("ref %s manifest_base64: %w", item.REF, err)
		}
		value.REFs[i] = NativeRefRecord{REF: item.REF, Manifest: decoded}
	}
	for i, item := range wire.Candidates {
		if err := domain.ValidateREF(item.REF); err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("candidate %d: %w", i, err)
		}
		decoded, err := decodeNativeSnapshotBase64(item.CandidateBase64)
		if err != nil {
			return NativeSnapshotV1{}, fmt.Errorf("candidate %s candidate_base64: %w", item.REF, err)
		}
		value.Candidates[i] = NativeCandidateRecord{REF: item.REF, Candidate: decoded}
	}
	return value, validateNativeSnapshotV1(value)
}

func validateNativeSnapshotV1(value NativeSnapshotV1) error {
	for i, blob := range value.Blobs {
		if err := blob.ID.ValidateNative(); err != nil {
			return fmt.Errorf("blob %d: %w", i, err)
		}
		if expected := domain.ComputeNativeBlobID(blob.Data); !blob.ID.Equal(expected) {
			return fmt.Errorf("blob %s: ID does not match data", blob.ID)
		}
		if i > 0 && value.Blobs[i-1].ID.String() >= blob.ID.String() {
			return fmt.Errorf("blobs are not strictly sorted and unique by ID")
		}
	}
	for i, ref := range value.REFs {
		if err := domain.ValidateREF(ref.REF); err != nil {
			return fmt.Errorf("ref %d: %w", i, err)
		}
		if i > 0 && value.REFs[i-1].REF >= ref.REF {
			return fmt.Errorf("refs are not strictly sorted and unique by REF")
		}
	}
	for i, candidate := range value.Candidates {
		if err := domain.ValidateREF(candidate.REF); err != nil {
			return fmt.Errorf("candidate %d: %w", i, err)
		}
		if i > 0 && value.Candidates[i-1].REF >= candidate.REF {
			return fmt.Errorf("candidates are not strictly sorted and unique by REF")
		}
	}
	return nil
}

func decodeNativeSnapshotBase64(text string) ([]byte, error) {
	decoded, err := base64.StdEncoding.Strict().DecodeString(text)
	if err != nil {
		return nil, err
	}
	if base64.StdEncoding.EncodeToString(decoded) != text {
		return nil, fmt.Errorf("base64 is not canonical padded RFC 4648")
	}
	return decoded, nil
}

func scanJSONForDuplicateFields(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := scanNativeJSONValue(decoder); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return fmt.Errorf("trailing JSON value")
	}
	return nil
}

func scanNativeJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	switch delimiter := token.(type) {
	case json.Delim:
		switch delimiter {
		case '{':
			seen := map[string]struct{}{}
			for decoder.More() {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok {
					return fmt.Errorf("object key is not a string")
				}
				if _, exists := seen[name]; exists {
					return fmt.Errorf("duplicate object member %q", name)
				}
				seen[name] = struct{}{}
				if err := scanNativeJSONValue(decoder); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		case '[':
			for decoder.More() {
				if err := scanNativeJSONValue(decoder); err != nil {
					return err
				}
			}
			_, err = decoder.Token()
			return err
		}
	}
	return nil
}
