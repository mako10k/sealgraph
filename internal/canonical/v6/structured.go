package v6

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/mako10k/sealgraph/internal/canonical"
	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	domainv6 "github.com/mako10k/sealgraph/internal/domain/v6"
)

func EncodeSeal(seal domainv6.Seal) ([]byte, error) {
	legacy := seal
	legacy.Schema = domainv5.SealSchema
	if _, err := canonicalv5.EncodeSeal(legacy); err != nil {
		return nil, err
	}
	if seal.Schema != domainv6.SealSchema {
		return nil, fmt.Errorf("seal schema is %q; expected %q", seal.Schema, domainv6.SealSchema)
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, seal.Schema)
	b = append(b, `,"material":`...)
	b = appendObjectID(b, seal.Material)
	b = append(b, `,"provenance":`...)
	b = appendObjectID(b, seal.Provenance)
	return append(b, '}'), nil
}

func DecodeSeal(data []byte) (domainv6.Seal, error) { return decodeCanonical(data, "seal", EncodeSeal) }

func EncodeMaterial(material domainv6.Material) ([]byte, error) {
	return canonicalv5.EncodeMaterial(material)
}
func DecodeMaterial(data []byte) (domainv6.Material, error) { return canonicalv5.DecodeMaterial(data) }

func EncodeProvenance(provenance domainv6.Provenance) ([]byte, error) {
	normalized, err := normalizeProvenance(provenance)
	if err != nil {
		return nil, err
	}
	b, err := canonical.AppendProvenanceStart(nil, normalized.Schema, normalized.Root, normalized.Draft)
	if err != nil {
		return nil, err
	}
	b, err = appendCauseLinks(b, normalized.CauseLinks)
	if err != nil {
		return nil, err
	}
	return append(b, '}'), nil
}

func DecodeProvenance(data []byte) (domainv6.Provenance, error) {
	return decodeCanonical(data, "provenance", EncodeProvenance)
}

func EncodeCandidate(candidate domainv6.Candidate) ([]byte, error) {
	normalized, err := normalizeCandidate(candidate)
	if err != nil {
		return nil, err
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, normalized.Schema)
	b = append(b, `,"ref":`...)
	b, _ = canonical.AppendString(b, normalized.REF)
	b = append(b, `,"expected_ref_head":`...)
	if normalized.ExpectedREFHead == nil {
		b = append(b, "null"...)
	} else {
		b = appendObjectID(b, *normalized.ExpectedREFHead)
	}
	b = append(b, `,"content":`...)
	b = appendObjectID(b, normalized.Content)
	b = append(b, `,"attachments":`...)
	b = appendAttachments(b, normalized.Attachments)
	b = append(b, `,"root":`...)
	b = canonical.AppendBool(b, normalized.Root)
	b = append(b, `,"draft":`...)
	b = canonical.AppendBool(b, normalized.Draft)
	b = append(b, `,"cause_links":`...)
	b, err = appendCauseLinks(b, normalized.CauseLinks)
	if err != nil {
		return nil, err
	}
	return append(b, '}', '\n'), nil
}

func DecodeCandidate(data []byte) (domainv6.Candidate, error) {
	return decodeCanonical(data, "candidate", EncodeCandidate)
}

func EncodeUpstreamChange(change domainv6.UpstreamChange) ([]byte, error) {
	if change.Schema != domainv6.UpstreamChangeSchema {
		return nil, fmt.Errorf("upstream change schema is %q; expected %q", change.Schema, domainv6.UpstreamChangeSchema)
	}
	if change.BeforeSeal != nil {
		if err := change.BeforeSeal.ValidateNative(); err != nil {
			return nil, fmt.Errorf("invalid before SealID: %w", err)
		}
	}
	if err := change.AfterMaterial.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid after MaterialID: %w", err)
	}
	links, err := normalizeCauseLinks(change.AfterCauseLinks)
	if err != nil {
		return nil, fmt.Errorf("invalid after Cause Links: %w", err)
	}
	b := []byte(`{"schema":`)
	b, _ = canonical.AppendString(b, change.Schema)
	b = append(b, `,"before_seal":`...)
	if change.BeforeSeal == nil {
		b = append(b, "null"...)
	} else {
		b = appendObjectID(b, *change.BeforeSeal)
	}
	b = append(b, `,"after_material":`...)
	b = appendObjectID(b, change.AfterMaterial)
	b = append(b, `,"after_root":`...)
	b = canonical.AppendBool(b, change.AfterRoot)
	b = append(b, `,"after_draft":`...)
	b = canonical.AppendBool(b, change.AfterDraft)
	b = append(b, `,"after_cause_links":`...)
	b, err = appendCauseLinks(b, links)
	if err != nil {
		return nil, err
	}
	return append(b, '}'), nil
}

func DecodeUpstreamChange(data []byte) (domainv6.UpstreamChange, error) {
	return decodeCanonical(data, "upstream change", EncodeUpstreamChange)
}

func normalizeProvenance(value domainv6.Provenance) (domainv6.Provenance, error) {
	if value.Schema != domainv6.ProvenanceSchema {
		return domainv6.Provenance{}, fmt.Errorf("provenance schema is %q; expected %q", value.Schema, domainv6.ProvenanceSchema)
	}
	links, err := normalizeCauseLinks(value.CauseLinks)
	if err != nil {
		return domainv6.Provenance{}, err
	}
	if value.Root && len(links) != 0 {
		return domainv6.Provenance{}, fmt.Errorf("root provenance cannot have Cause Links")
	}
	if !value.Root && len(links) == 0 {
		return domainv6.Provenance{}, fmt.Errorf("non-root provenance requires at least one Cause Link")
	}
	value.CauseLinks = links
	return value, nil
}

func normalizeCandidate(value domainv6.Candidate) (domainv6.Candidate, error) {
	if value.Schema != domainv6.CandidateSchema {
		return domainv6.Candidate{}, fmt.Errorf("candidate schema is %q; expected %q", value.Schema, domainv6.CandidateSchema)
	}
	legacy := value
	legacy.Schema = domainv5.CandidateSchema
	legacy.CauseLinks = stripMetadata(value.CauseLinks)
	normalized, err := domainv5.NormalizeCandidate(legacy)
	if err != nil {
		return domainv6.Candidate{}, err
	}
	links, err := normalizeCauseLinks(value.CauseLinks)
	if err != nil {
		return domainv6.Candidate{}, err
	}
	value.Attachments = normalized.Attachments
	value.CauseLinks = links
	return value, nil
}

func normalizeCauseLinks(input []domainv6.CauseLink) ([]domainv6.CauseLink, error) {
	links := make([]domainv6.CauseLink, len(input))
	for i, link := range input {
		legacy := domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: false, CauseLinks: stripMetadata([]domainv6.CauseLink{link})}
		normalized, err := domainv5.NormalizeProvenance(legacy)
		if err != nil {
			return nil, err
		}
		metadata, _, err := normalizeMetadata(link.Metadata)
		if err != nil {
			return nil, fmt.Errorf("Cause Link target %s: %w", link.TargetSeal, err)
		}
		links[i] = normalized.CauseLinks[0]
		links[i].Metadata = metadata
	}
	sort.Slice(links, func(i, j int) bool { return links[i].TargetSeal.String() < links[j].TargetSeal.String() })
	for i := 1; i < len(links); i++ {
		if links[i-1].TargetSeal.Equal(links[i].TargetSeal) {
			return nil, fmt.Errorf("duplicate Cause Link target %s", links[i].TargetSeal)
		}
	}
	if links == nil {
		links = []domainv6.CauseLink{}
	}
	return links, nil
}

func stripMetadata(input []domainv6.CauseLink) []domainv5.CauseLink {
	result := make([]domainv5.CauseLink, len(input))
	for i, link := range input {
		result[i] = domainv5.CauseLink{TargetSeal: link.TargetSeal, PreviousRevisionSealOfTargetSeal: append([]domain.ObjectID(nil), link.PreviousRevisionSealOfTargetSeal...), Messages: append([]string(nil), link.Messages...), Metadata: []domainv5.MetadataEntry{}}
	}
	return result
}

func appendCauseLinks(b []byte, links []domainv6.CauseLink) ([]byte, error) {
	b = append(b, '[')
	for i, link := range links {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"target_seal":`...)
		var err error
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
		b = append(b, `,"metadata":`...)
		metadata, err := appendMetadata(nil, link.Metadata)
		if err != nil {
			return nil, err
		}
		b = append(b, metadata...)
		b = append(b, '}')
	}
	return append(b, ']'), nil
}

func appendAttachments(b []byte, attachments []domainv6.Attachment) []byte {
	b = append(b, '[')
	for i, attachment := range attachments {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, _ = canonical.AppendString(b, attachment.Name)
		b = append(b, `,"media_type":`...)
		b, _ = canonical.AppendString(b, attachment.MediaType)
		b = append(b, `,"blob":`...)
		b = appendObjectID(b, attachment.Blob)
		b = append(b, '}')
	}
	return append(b, ']')
}

func appendObjectID(b []byte, id domain.ObjectID) []byte {
	return append(append(append(b, '"'), id.String()...), '"')
}

func decodeCanonical[T any](data []byte, label string, encode func(T) ([]byte, error)) (T, error) {
	var zero T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value T
	if err := decoder.Decode(&value); err != nil {
		return zero, fmt.Errorf("decode %s: %w", label, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return zero, fmt.Errorf("decode %s: trailing JSON value", label)
	}
	encoded, err := encode(value)
	if err != nil {
		return zero, fmt.Errorf("validate %s: %w", label, err)
	}
	if !bytes.Equal(data, encoded) {
		return zero, fmt.Errorf("%s is not canonical", label)
	}
	return value, nil
}
