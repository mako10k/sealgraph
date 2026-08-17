package domain

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

func NormalizeLinks(links []Link) ([]Link, error) {
	normalized := make([]Link, len(links))
	copy(normalized, links)
	for _, link := range normalized {
		if err := link.TargetSeal.ValidateNative(); err != nil {
			return nil, fmt.Errorf("invalid dependency seal: %w", err)
		}
		if !utf8.ValidString(link.Message) {
			return nil, fmt.Errorf("dependency %s message is not valid UTF-8", link.TargetSeal)
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		a, b := normalized[i], normalized[j]
		if a.TargetSeal.Hex != b.TargetSeal.Hex {
			return a.TargetSeal.Hex < b.TargetSeal.Hex
		}
		return a.Message < b.Message
	})
	for i := 1; i < len(normalized); i++ {
		if normalized[i-1].TargetSeal.Equal(normalized[i].TargetSeal) {
			return nil, fmt.Errorf("duplicate dependency seal %s; one exact Cause target may appear only once", normalized[i].TargetSeal)
		}
	}
	return normalized, nil
}

func NormalizeAttachments(attachments []Attachment) ([]Attachment, error) {
	normalized := make([]Attachment, len(attachments))
	copy(normalized, attachments)
	for _, attachment := range normalized {
		if attachment.Name == "" {
			return nil, fmt.Errorf("attachment name is empty")
		}
		if !utf8.ValidString(attachment.Name) || !utf8.ValidString(attachment.MediaType) {
			return nil, fmt.Errorf("attachment %q metadata is not valid UTF-8", attachment.Name)
		}
		if err := attachment.Blob.ValidateNativeBlob(); err != nil {
			return nil, fmt.Errorf("attachment %q: %w", attachment.Name, err)
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		a, b := normalized[i], normalized[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.MediaType != b.MediaType {
			return a.MediaType < b.MediaType
		}
		if a.Blob.Store != b.Blob.Store {
			return a.Blob.Store < b.Blob.Store
		}
		if a.Blob.Type != b.Blob.Type {
			return a.Blob.Type < b.Blob.Type
		}
		return a.Blob.ID.Hex < b.Blob.ID.Hex
	})
	for i := 1; i < len(normalized); i++ {
		if normalized[i-1].Name == normalized[i].Name {
			return nil, fmt.Errorf("duplicate attachment name %q", normalized[i].Name)
		}
	}
	return normalized, nil
}

func NormalizeSeal(payload SealPayload) (SealPayload, error) {
	if payload.Schema != SealSchema {
		return SealPayload{}, fmt.Errorf("seal schema is %q; expected %q", payload.Schema, SealSchema)
	}
	if payload.ParentRevision != nil {
		if err := payload.ParentRevision.ValidateNative(); err != nil {
			return SealPayload{}, fmt.Errorf("invalid parent revision: %w", err)
		}
	}
	if err := payload.Content.ValidateNativeBlob(); err != nil {
		return SealPayload{}, fmt.Errorf("invalid content: %w", err)
	}
	attachments, err := NormalizeAttachments(payload.Attachments)
	if err != nil {
		return SealPayload{}, err
	}
	links, err := NormalizeLinks(payload.Links)
	if err != nil {
		return SealPayload{}, err
	}
	if payload.Root && len(links) != 0 {
		return SealPayload{}, fmt.Errorf("root seal cannot have Cause links; unlink them before sealing")
	}
	if !payload.Root && len(links) == 0 {
		return SealPayload{}, fmt.Errorf("non-root seal requires at least one Cause link; link an exact upstream seal or declare it root")
	}
	payload.Attachments = attachments
	payload.Links = links
	return payload, nil
}

func ValidateCandidate(candidate Candidate) error {
	if candidate.Schema != CandidateSchema {
		return fmt.Errorf("candidate schema is %q; expected %q", candidate.Schema, CandidateSchema)
	}
	if err := ValidateREF(candidate.REF); err != nil {
		return fmt.Errorf("invalid candidate REF %q: %w", candidate.REF, err)
	}
	if candidate.ParentRevision != nil {
		if err := candidate.ParentRevision.ValidateNative(); err != nil {
			return fmt.Errorf("invalid candidate parent revision: %w", err)
		}
	}
	if candidate.ExpectedREFHead != nil {
		if err := candidate.ExpectedREFHead.ValidateNative(); err != nil {
			return fmt.Errorf("invalid candidate expected REF head: %w", err)
		}
	}
	if err := candidate.Content.ValidateNativeBlob(); err != nil {
		return fmt.Errorf("invalid candidate content: %w", err)
	}
	if candidate.Attachments == nil || candidate.Links == nil {
		return fmt.Errorf("candidate attachments and links must be JSON arrays, not null")
	}
	if _, err := NormalizeAttachments(candidate.Attachments); err != nil {
		return err
	}
	if _, err := NormalizeLinks(candidate.Links); err != nil {
		return err
	}
	return nil
}
