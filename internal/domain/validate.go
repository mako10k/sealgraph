package domain

import (
	"fmt"
	"sort"
	"unicode/utf8"
)

func NormalizeLinks(links []Link) ([]Link, error) {
	normalized := append([]Link(nil), links...)
	for _, link := range normalized {
		if err := ValidateREF(link.TargetREF); err != nil {
			return nil, fmt.Errorf("invalid dependency REF %q: %w", link.TargetREF, err)
		}
		if err := link.TargetSeal.ValidateNative(); err != nil {
			return nil, fmt.Errorf("invalid dependency %s seal: %w", link.TargetREF, err)
		}
		if !utf8.ValidString(link.Message) {
			return nil, fmt.Errorf("dependency %s message is not valid UTF-8", link.TargetREF)
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		a, b := normalized[i], normalized[j]
		if a.TargetREF != b.TargetREF {
			return a.TargetREF < b.TargetREF
		}
		if a.TargetSeal.Hex != b.TargetSeal.Hex {
			return a.TargetSeal.Hex < b.TargetSeal.Hex
		}
		return a.Message < b.Message
	})
	for i := 1; i < len(normalized); i++ {
		if normalized[i-1].TargetREF == normalized[i].TargetREF {
			return nil, fmt.Errorf("duplicate dependency REF %q; use link to replace its target explicitly", normalized[i].TargetREF)
		}
	}
	return normalized, nil
}

func NormalizeAttachments(attachments []Attachment) ([]Attachment, error) {
	normalized := append([]Attachment(nil), attachments...)
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
	if err := ValidateREF(payload.REF); err != nil {
		return SealPayload{}, fmt.Errorf("invalid seal REF %q: %w", payload.REF, err)
	}
	if payload.Parent != nil {
		if err := payload.Parent.ValidateNative(); err != nil {
			return SealPayload{}, fmt.Errorf("invalid parent: %w", err)
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
		return SealPayload{}, fmt.Errorf("root REF %q cannot have dependencies; unlink them before sealing", payload.REF)
	}
	if !payload.Root && len(links) == 0 {
		return SealPayload{}, fmt.Errorf("non-root REF %q requires at least one dependency; use link or declare it --root", payload.REF)
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
	if candidate.Base != nil {
		if err := candidate.Base.ValidateNative(); err != nil {
			return fmt.Errorf("invalid candidate base: %w", err)
		}
	}
	if err := candidate.Content.ValidateNativeBlob(); err != nil {
		return fmt.Errorf("invalid candidate content: %w", err)
	}
	if _, err := NormalizeAttachments(candidate.Attachments); err != nil {
		return err
	}
	if _, err := NormalizeLinks(candidate.Links); err != nil {
		return err
	}
	return nil
}
