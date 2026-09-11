// Package v5 implements the exact canonical structured Blob bytes selected by
// the accepted format-5 ADRs.
package v5

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

// EncodeSeal returns canonical format-5 Seal Blob bytes without a trailing LF.
func EncodeSeal(seal domainv5.Seal) ([]byte, error) {
	if err := domainv5.ValidateSeal(seal); err != nil {
		return nil, err
	}
	b := make([]byte, 0, 192)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, seal.Schema)
	b = append(b, `,"material":`...)
	b = appendObjectID(b, seal.Material)
	b = append(b, `,"provenance":`...)
	b = appendObjectID(b, seal.Provenance)
	b = append(b, '}')
	return b, nil
}

// DecodeSeal requires semantically valid bytes and exact canonical re-encoding.
func DecodeSeal(data []byte) (domainv5.Seal, error) {
	return decodeCanonical(data, "seal", EncodeSeal)
}

// EncodeMaterial returns canonical Material Blob bytes without a trailing LF.
func EncodeMaterial(material domainv5.Material) ([]byte, error) {
	normalized, err := domainv5.NormalizeMaterial(material)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 384)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, normalized.Schema)
	b = append(b, `,"content":`...)
	b = appendObjectID(b, normalized.Content)
	b = append(b, `,"attachments":`...)
	b, err = appendAttachments(b, normalized.Attachments)
	if err != nil {
		return nil, err
	}
	b = append(b, '}')
	return b, nil
}

func appendAttachments(b []byte, attachments []domainv5.Attachment) ([]byte, error) {
	var err error
	b = append(b, '[')
	for i, attachment := range attachments {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, err = canonical.AppendString(b, attachment.Name)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"media_type":`...)
		b, err = canonical.AppendString(b, attachment.MediaType)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"blob":`...)
		b = appendObjectID(b, attachment.Blob)
		b = append(b, '}')
	}
	return append(b, ']'), nil
}

// DecodeMaterial requires semantically valid bytes and exact canonical re-encoding.
func DecodeMaterial(data []byte) (domainv5.Material, error) {
	return decodeCanonical(data, "material", EncodeMaterial)
}

// EncodeProvenance returns canonical Provenance Blob bytes without a trailing LF.
func EncodeProvenance(provenance domainv5.Provenance) ([]byte, error) {
	normalized, err := domainv5.NormalizeProvenance(provenance)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 512)
	b, err = canonical.AppendProvenanceStart(b, normalized.Schema, normalized.Root, normalized.Draft)
	if err != nil {
		return nil, err
	}
	b, err = appendCauseLinks(b, normalized.CauseLinks)
	if err != nil {
		return nil, err
	}
	b = append(b, '}')
	return b, nil
}

func appendCauseLinks(b []byte, links []domainv5.CauseLink) ([]byte, error) {
	var err error
	b = append(b, '[')
	for i, link := range links {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"target_seal":`...)
		b, err = canonical.AppendNativeObjectID(b, link.TargetSeal)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"previous_revision_seal_of_target_seal":`...)
		b, err = canonical.AppendNativeObjectIDs(b, link.PreviousRevisionSealOfTargetSeal)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"messages":`...)
		b, err = canonical.AppendStrings(b, link.Messages)
		if err != nil {
			return nil, err
		}
		b = append(b, '}')
	}
	return append(b, ']'), nil
}

// DecodeProvenance requires semantically valid bytes and exact canonical re-encoding.
func DecodeProvenance(data []byte) (domainv5.Provenance, error) {
	return decodeCanonical(data, "provenance", EncodeProvenance)
}

func appendObjectID(b []byte, id domain.ObjectID) []byte {
	b = append(b, '"')
	b = append(b, id.Hex...)
	return append(b, '"')
}

func decodeCanonical[T any](data []byte, label string, encode func(T) ([]byte, error)) (T, error) {
	var zero T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s Blob: %w", label, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return zero, fmt.Errorf("decode %s Blob: trailing JSON value", label)
	}
	canonicalBytes, err := encode(value)
	if err != nil {
		return zero, fmt.Errorf("validate %s Blob: %w", label, err)
	}
	if !bytes.Equal(data, canonicalBytes) {
		return zero, fmt.Errorf("%s Blob is not canonical; restore the exact canonical bytes explicitly", label)
	}
	return value, nil
}
