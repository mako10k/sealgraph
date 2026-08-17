package migration

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/canonical"
	"github.com/mako10k/sealgraph/internal/domain"
)

const Format3SealSchema = "sealgraph/seal/v3"

// Format3Link and Format3SealPayload exist only inside the explicit
// logical-v1 migration document. They are not runtime repository readers.
type Format3Link struct {
	TargetREF  string          `json:"target_ref"`
	TargetSeal domain.ObjectID `json:"target_seal"`
	Message    string          `json:"message"`
}

type Format3SealPayload struct {
	Schema      string              `json:"schema"`
	REF         string              `json:"ref"`
	Parent      *domain.ObjectID    `json:"parent"`
	Content     domain.ContentRef   `json:"content"`
	Attachments []domain.Attachment `json:"attachments"`
	Links       []Format3Link       `json:"links"`
	Root        bool                `json:"root"`
	Draft       bool                `json:"draft"`
}

func normalizeFormat3Seal(payload Format3SealPayload) (Format3SealPayload, error) {
	if payload.Schema != Format3SealSchema {
		return Format3SealPayload{}, fmt.Errorf("seal schema is %q; expected %q", payload.Schema, Format3SealSchema)
	}
	if err := domain.ValidateREF(payload.REF); err != nil {
		return Format3SealPayload{}, fmt.Errorf("invalid seal REF %q: %w", payload.REF, err)
	}
	if payload.Parent != nil {
		if err := payload.Parent.ValidateNative(); err != nil {
			return Format3SealPayload{}, fmt.Errorf("invalid parent: %w", err)
		}
	}
	if err := payload.Content.ValidateNativeBlob(); err != nil {
		return Format3SealPayload{}, fmt.Errorf("invalid content: %w", err)
	}
	attachments, err := domain.NormalizeAttachments(payload.Attachments)
	if err != nil {
		return Format3SealPayload{}, err
	}
	links := append([]Format3Link(nil), payload.Links...)
	for _, link := range links {
		if err := domain.ValidateREF(link.TargetREF); err != nil {
			return Format3SealPayload{}, fmt.Errorf("invalid dependency REF %q: %w", link.TargetREF, err)
		}
		if err := link.TargetSeal.ValidateNative(); err != nil {
			return Format3SealPayload{}, fmt.Errorf("invalid dependency %s seal: %w", link.TargetREF, err)
		}
		if !utf8.ValidString(link.Message) {
			return Format3SealPayload{}, fmt.Errorf("dependency %s message is not valid UTF-8", link.TargetREF)
		}
	}
	sort.Slice(links, func(i, j int) bool {
		if links[i].TargetREF != links[j].TargetREF {
			return links[i].TargetREF < links[j].TargetREF
		}
		if links[i].TargetSeal.Hex != links[j].TargetSeal.Hex {
			return links[i].TargetSeal.Hex < links[j].TargetSeal.Hex
		}
		return links[i].Message < links[j].Message
	})
	for i := 1; i < len(links); i++ {
		if links[i-1].TargetREF == links[i].TargetREF {
			return Format3SealPayload{}, fmt.Errorf("duplicate dependency REF %q", links[i].TargetREF)
		}
	}
	if payload.Root && len(links) != 0 {
		return Format3SealPayload{}, fmt.Errorf("root REF %q cannot have dependencies", payload.REF)
	}
	if !payload.Root && len(links) == 0 {
		return Format3SealPayload{}, fmt.Errorf("non-root REF %q requires at least one dependency", payload.REF)
	}
	payload.Attachments = attachments
	payload.Links = links
	return payload, nil
}

func encodeFormat3Seal(payload Format3SealPayload) ([]byte, error) {
	normalized, err := normalizeFormat3Seal(payload)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 512)
	b = append(b, `{"schema":`...)
	b, _ = canonical.AppendString(b, normalized.Schema)
	b = append(b, `,"ref":`...)
	b, _ = canonical.AppendString(b, normalized.REF)
	b = append(b, `,"parent":`...)
	if normalized.Parent == nil {
		b = append(b, "null"...)
	} else {
		b, _ = canonical.AppendString(b, normalized.Parent.String())
	}
	b = appendFormat3Content(b, `,"content":`, normalized.Content)
	b = append(b, `,"attachments":[`...)
	for i, attachment := range normalized.Attachments {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, _ = canonical.AppendString(b, attachment.Name)
		b = append(b, `,"media_type":`...)
		b, _ = canonical.AppendString(b, attachment.MediaType)
		b = appendFormat3Content(b, `,"blob":`, attachment.Blob)
		b = append(b, '}')
	}
	b = append(b, `],"links":[`...)
	for i, link := range normalized.Links {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"target_ref":`...)
		b, _ = canonical.AppendString(b, link.TargetREF)
		b = append(b, `,"target_seal":`...)
		b, _ = canonical.AppendString(b, link.TargetSeal.String())
		b = append(b, `,"message":`...)
		b, _ = canonical.AppendString(b, link.Message)
		b = append(b, '}')
	}
	b = append(b, `],"root":`...)
	b = canonical.AppendBool(b, normalized.Root)
	b = append(b, `,"draft":`...)
	b = canonical.AppendBool(b, normalized.Draft)
	b = append(b, '}')
	return b, nil
}

// EncodeFormat3Seal emits exact legacy payload bytes only for constructing or
// validating an explicit logical-v1 migration document.
func EncodeFormat3Seal(payload Format3SealPayload) ([]byte, error) {
	return encodeFormat3Seal(payload)
}

func appendFormat3Content(b []byte, member string, content domain.ContentRef) []byte {
	b = append(b, member...)
	b = append(b, `{"store":`...)
	b, _ = canonical.AppendString(b, content.Store)
	b = append(b, `,"type":`...)
	b, _ = canonical.AppendString(b, content.Type)
	b = append(b, `,"id":`...)
	b, _ = canonical.AppendString(b, content.ID.String())
	b = append(b, '}')
	return b
}

func decodeFormat3Seal(data []byte) (Format3SealPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var payload Format3SealPayload
	if err := decoder.Decode(&payload); err != nil {
		return Format3SealPayload{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return Format3SealPayload{}, fmt.Errorf("trailing JSON value")
	}
	encoded, err := encodeFormat3Seal(payload)
	if err != nil {
		return Format3SealPayload{}, err
	}
	if !bytes.Equal(data, encoded) {
		return Format3SealPayload{}, fmt.Errorf("format-3 seal payload is not canonical")
	}
	return payload, nil
}
