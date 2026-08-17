package canonical

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/mako10k/sealgraph/internal/domain"
)

// EncodeCandidate emits the deterministic persisted format-4 candidate bytes.
// Candidates are mutable and have no ObjectID, but exact bytes are used to
// prove unchanged cleanup after publication.
func EncodeCandidate(candidate domain.Candidate) ([]byte, error) {
	links, err := domain.NormalizeLinks(candidate.Links)
	if err != nil {
		return nil, err
	}
	attachments, err := domain.NormalizeAttachments(candidate.Attachments)
	if err != nil {
		return nil, err
	}
	candidate.Links = links
	candidate.Attachments = attachments
	if candidate.Links == nil {
		candidate.Links = []domain.Link{}
	}
	if candidate.Attachments == nil {
		candidate.Attachments = []domain.Attachment{}
	}
	if err := domain.ValidateCandidate(candidate); err != nil {
		return nil, err
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		return nil, fmt.Errorf("encode candidate %s: %w", candidate.REF, err)
	}
	return append(data, '\n'), nil
}

// DecodeCandidate rejects unknown/trailing state and validates the complete
// candidate. Writer formatting is deterministic, while semantically valid
// manually formatted JSON remains readable mutable state.
func DecodeCandidate(data []byte) (domain.Candidate, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var candidate domain.Candidate
	if err := decoder.Decode(&candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("decode candidate: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return domain.Candidate{}, fmt.Errorf("decode candidate: trailing JSON value")
	}
	if err := domain.ValidateCandidate(candidate); err != nil {
		return domain.Candidate{}, err
	}
	return candidate, nil
}
