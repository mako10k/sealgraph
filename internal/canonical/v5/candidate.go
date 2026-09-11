package v5

import (
	"fmt"

	"github.com/mako10k/sealgraph/internal/canonical"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

// EncodeCandidate emits the exact persisted format-5 Candidate bytes,
// including the required final LF.
func EncodeCandidate(candidate domainv5.Candidate) ([]byte, error) {
	normalized, err := domainv5.NormalizeCandidate(candidate)
	if err != nil {
		return nil, err
	}
	b := make([]byte, 0, 768)
	b = append(b, `{"schema":`...)
	b, err = canonical.AppendString(b, normalized.Schema)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"ref":`...)
	b, err = canonical.AppendString(b, normalized.REF)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"expected_ref_head":`...)
	if normalized.ExpectedREFHead == nil {
		b = append(b, "null"...)
	} else {
		b = appendObjectID(b, *normalized.ExpectedREFHead)
	}
	b = append(b, `,"content":`...)
	b = appendObjectID(b, normalized.Content)
	b = append(b, `,"attachments":`...)
	b, err = appendAttachments(b, normalized.Attachments)
	if err != nil {
		return nil, err
	}
	b = append(b, `,"root":`...)
	b = canonical.AppendBool(b, normalized.Root)
	b = append(b, `,"draft":`...)
	b = canonical.AppendBool(b, normalized.Draft)
	b = append(b, `,"cause_links":`...)
	b, err = appendCauseLinks(b, normalized.CauseLinks)
	if err != nil {
		return nil, err
	}
	b = append(b, '}', '\n')
	return b, nil
}

// DecodeCandidate requires the exact canonical writer bytes, including the
// final LF. It does not accept semantically equivalent manual formatting.
func DecodeCandidate(data []byte) (domainv5.Candidate, error) {
	candidate, err := decodeCanonical(data, "candidate", func(value domainv5.Candidate) ([]byte, error) {
		return EncodeCandidate(value)
	})
	if err != nil {
		return domainv5.Candidate{}, fmt.Errorf("decode persisted Candidate: %w", err)
	}
	return candidate, nil
}
