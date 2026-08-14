// Package canonical implements the native v2 canonical seal byte contract.
package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/domain"
)

func EncodeSeal(payload domain.SealPayload) ([]byte, error) {
	normalized, err := domain.NormalizeSeal(payload)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 512)
	b = append(b, `{"schema":`...)
	b, err = appendString(b, normalized.Schema)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"ref":`...)
	b, err = appendString(b, normalized.REF)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"parent":`...)
	if normalized.Parent == nil {
		b = append(b, "null"...)
	} else {
		b, err = appendObjectID(b, *normalized.Parent)
		if err != nil {
			return nil, err
		}
	}
	b = append(b, `,"content":`...)
	b, err = appendContent(b, normalized.Content)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"attachments":[`...)
	for i, attachment := range normalized.Attachments {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"name":`...)
		b, err = appendString(b, attachment.Name)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"media_type":`...)
		b, err = appendString(b, attachment.MediaType)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"blob":`...)
		b, err = appendContent(b, attachment.Blob)
		if err != nil {
			return nil, err
		}
		b = append(b, '}')
	}
	b = append(b, `],"links":[`...)
	for i, link := range normalized.Links {
		if i > 0 {
			b = append(b, ',')
		}
		b = append(b, `{"target_ref":`...)
		b, err = appendString(b, link.TargetREF)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"target_seal":`...)
		b, err = appendObjectID(b, link.TargetSeal)
		if err != nil {
			return nil, err
		}
		b = append(b, `,"message":`...)
		b, err = appendString(b, link.Message)
		if err != nil {
			return nil, err
		}
		b = append(b, '}')
	}
	b = append(b, `],"message":`...)
	b, err = appendString(b, normalized.Message)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"root":`...)
	if normalized.Root {
		b = append(b, "true"...)
	} else {
		b = append(b, "false"...)
	}
	b = append(b, `,"draft":`...)
	if normalized.Draft {
		b = append(b, "true"...)
	} else {
		b = append(b, "false"...)
	}
	b = append(b, `,"created_at":`...)
	b, err = appendString(b, normalized.CreatedAt)
	if err != nil {
		return nil, err
	}
	b = append(b, '}')
	return b, nil
}

func DecodeSeal(data []byte) (domain.SealPayload, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var payload domain.SealPayload
	if err := decoder.Decode(&payload); err != nil {
		return domain.SealPayload{}, fmt.Errorf("decode seal payload: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.SealPayload{}, fmt.Errorf("decode seal payload: trailing JSON value")
	}
	canonical, err := EncodeSeal(payload)
	if err != nil {
		return domain.SealPayload{}, fmt.Errorf("validate seal payload: %w", err)
	}
	if !bytes.Equal(data, canonical) {
		return domain.SealPayload{}, fmt.Errorf("seal payload is not canonical; inspect the object and restore the exact canonical bytes explicitly")
	}
	return payload, nil
}

func appendObjectID(b []byte, id domain.ObjectID) ([]byte, error) {
	if err := id.ValidateNative(); err != nil {
		return nil, err
	}
	return appendString(b, id.Hex)
}

func appendContent(b []byte, content domain.ContentRef) ([]byte, error) {
	b = append(b, `{"store":`...)
	var err error
	b, err = appendString(b, content.Store)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"type":`...)
	b, err = appendString(b, content.Type)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"id":`...)
	b, err = appendObjectID(b, content.ID)
	if err != nil {
		return nil, err
	}
	b = append(b, '}')
	return b, nil
}

func appendString(b []byte, value string) ([]byte, error) {
	if !utf8.ValidString(value) {
		return nil, fmt.Errorf("canonical string is not valid UTF-8")
	}
	b = append(b, '"')
	for _, r := range value {
		switch r {
		case '"', '\\':
			b = append(b, '\\', byte(r))
		case '\b':
			b = append(b, `\b`...)
		case '\t':
			b = append(b, `\t`...)
		case '\n':
			b = append(b, `\n`...)
		case '\f':
			b = append(b, `\f`...)
		case '\r':
			b = append(b, `\r`...)
		default:
			if r < 0x20 {
				const hex = "0123456789abcdef"
				b = append(b, '\\', 'u', '0', '0', hex[byte(r)>>4], hex[byte(r)&0xf])
			} else {
				b = utf8.AppendRune(b, r)
			}
		}
	}
	b = append(b, '"')
	return b, nil
}
