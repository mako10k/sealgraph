package v5

import (
	"fmt"
	"sort"
	"unicode/utf8"

	"github.com/mako10k/sealgraph/internal/domain"
)

// ValidateSeal validates the complete format-5 Seal record. Referenced Blob
// schemas are validated by the typed repository reader, not by this value-only
// boundary.
func ValidateSeal(seal Seal) error {
	if seal.Schema != SealSchema {
		return fmt.Errorf("seal schema is %q; expected %q", seal.Schema, SealSchema)
	}
	if err := seal.Material.ValidateNative(); err != nil {
		return fmt.Errorf("invalid material ID: %w", err)
	}
	if err := seal.Provenance.ValidateNative(); err != nil {
		return fmt.Errorf("invalid provenance ID: %w", err)
	}
	return nil
}

// NormalizeMaterial validates and canonically orders one Material value.
func NormalizeMaterial(material Material) (Material, error) {
	if material.Schema != MaterialSchema {
		return Material{}, fmt.Errorf("material schema is %q; expected %q", material.Schema, MaterialSchema)
	}
	if err := material.Content.ValidateNative(); err != nil {
		return Material{}, fmt.Errorf("invalid content Blob ID: %w", err)
	}
	attachments, err := normalizeAttachments(material.Attachments)
	if err != nil {
		return Material{}, err
	}
	material.Attachments = attachments
	return material, nil
}

// NormalizeProvenance validates and canonically orders one Provenance value.
func NormalizeProvenance(provenance Provenance) (Provenance, error) {
	if provenance.Schema != ProvenanceSchema {
		return Provenance{}, fmt.Errorf("provenance schema is %q; expected %q", provenance.Schema, ProvenanceSchema)
	}
	links, err := normalizeCauseLinks(provenance.CauseLinks)
	if err != nil {
		return Provenance{}, err
	}
	if provenance.Root && len(links) != 0 {
		return Provenance{}, fmt.Errorf("root provenance cannot have Cause Links")
	}
	if !provenance.Root && len(links) == 0 {
		return Provenance{}, fmt.Errorf("non-root provenance requires at least one Cause Link")
	}
	provenance.CauseLinks = links
	return provenance, nil
}

// NormalizeCandidate validates and canonically orders one parentless
// format-5 Candidate value.
func NormalizeCandidate(candidate Candidate) (Candidate, error) {
	if candidate.Schema != CandidateSchema {
		return Candidate{}, fmt.Errorf("candidate schema is %q; expected %q", candidate.Schema, CandidateSchema)
	}
	if err := domain.ValidateREF(candidate.REF); err != nil {
		return Candidate{}, fmt.Errorf("invalid candidate REF %q: %w", candidate.REF, err)
	}
	if candidate.ExpectedREFHead != nil {
		if err := candidate.ExpectedREFHead.ValidateNative(); err != nil {
			return Candidate{}, fmt.Errorf("invalid candidate expected REF head: %w", err)
		}
	}
	if err := candidate.Content.ValidateNative(); err != nil {
		return Candidate{}, fmt.Errorf("invalid candidate content Blob ID: %w", err)
	}
	attachments, err := normalizeAttachments(candidate.Attachments)
	if err != nil {
		return Candidate{}, err
	}
	links, err := normalizeCauseLinks(candidate.CauseLinks)
	if err != nil {
		return Candidate{}, err
	}
	if candidate.Root && len(links) != 0 {
		return Candidate{}, fmt.Errorf("root candidate cannot have Cause Links")
	}
	if !candidate.Root && len(links) == 0 {
		return Candidate{}, fmt.Errorf("non-root candidate requires at least one Cause Link")
	}
	candidate.Attachments = attachments
	candidate.CauseLinks = links
	return candidate, nil
}

func normalizeAttachments(input []Attachment) ([]Attachment, error) {
	attachments := append([]Attachment(nil), input...)
	for _, attachment := range attachments {
		if attachment.Name == "" {
			return nil, fmt.Errorf("attachment name is empty")
		}
		if !utf8.ValidString(attachment.Name) || !utf8.ValidString(attachment.MediaType) {
			return nil, fmt.Errorf("attachment %q metadata is not valid UTF-8", attachment.Name)
		}
		if err := attachment.Blob.ValidateNative(); err != nil {
			return nil, fmt.Errorf("attachment %q has invalid Blob ID: %w", attachment.Name, err)
		}
	}
	sort.Slice(attachments, func(i, j int) bool {
		a, b := attachments[i], attachments[j]
		if a.Name != b.Name {
			return a.Name < b.Name
		}
		if a.MediaType != b.MediaType {
			return a.MediaType < b.MediaType
		}
		return a.Blob.Hex < b.Blob.Hex
	})
	for i := 1; i < len(attachments); i++ {
		if attachments[i-1].Name == attachments[i].Name {
			return nil, fmt.Errorf("duplicate attachment name %q", attachments[i].Name)
		}
	}
	if attachments == nil {
		attachments = []Attachment{}
	}
	return attachments, nil
}

func normalizeCauseLinks(input []CauseLink) ([]CauseLink, error) {
	links := make([]CauseLink, len(input))
	for i, link := range input {
		if len(link.Metadata) != 0 {
			return nil, fmt.Errorf("format-5 Cause Link target %s cannot contain metadata", link.TargetSeal)
		}
		if err := link.TargetSeal.ValidateNative(); err != nil {
			return nil, fmt.Errorf("Cause Link has invalid target SealID: %w", err)
		}
		previous, err := normalizeObjectIDs(link.PreviousRevisionSealOfTargetSeal, "previous-revision SealID")
		if err != nil {
			return nil, fmt.Errorf("Cause Link target %s: %w", link.TargetSeal, err)
		}
		messages, err := normalizeMessages(link.Messages)
		if err != nil {
			return nil, fmt.Errorf("Cause Link target %s: %w", link.TargetSeal, err)
		}
		links[i] = CauseLink{
			TargetSeal:                       link.TargetSeal,
			PreviousRevisionSealOfTargetSeal: previous,
			Messages:                         messages,
			Metadata:                         []MetadataEntry{},
		}
	}
	sort.Slice(links, func(i, j int) bool { return causeLinkLess(links[i], links[j]) })
	for i := 1; i < len(links); i++ {
		if links[i-1].TargetSeal.Equal(links[i].TargetSeal) {
			return nil, fmt.Errorf("duplicate Cause Link target %s", links[i].TargetSeal)
		}
	}
	if links == nil {
		links = []CauseLink{}
	}
	return links, nil
}

func normalizeObjectIDs(input []domain.ObjectID, label string) ([]domain.ObjectID, error) {
	ids := append([]domain.ObjectID(nil), input...)
	for _, id := range ids {
		if err := id.ValidateNative(); err != nil {
			return nil, fmt.Errorf("invalid %s: %w", label, err)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].Hex < ids[j].Hex })
	for i := 1; i < len(ids); i++ {
		if ids[i-1].Equal(ids[i]) {
			return nil, fmt.Errorf("duplicate %s %s", label, ids[i])
		}
	}
	if ids == nil {
		ids = []domain.ObjectID{}
	}
	return ids, nil
}

func normalizeMessages(input []string) ([]string, error) {
	messages := append([]string(nil), input...)
	for _, message := range messages {
		if !utf8.ValidString(message) {
			return nil, fmt.Errorf("message is not valid UTF-8")
		}
	}
	sort.Strings(messages)
	for i := 1; i < len(messages); i++ {
		if messages[i-1] == messages[i] {
			return nil, fmt.Errorf("duplicate message %q", messages[i])
		}
	}
	if messages == nil {
		messages = []string{}
	}
	return messages, nil
}

func causeLinkLess(a, b CauseLink) bool {
	if a.TargetSeal.Hex != b.TargetSeal.Hex {
		return a.TargetSeal.Hex < b.TargetSeal.Hex
	}
	if comparison := compareObjectIDs(a.PreviousRevisionSealOfTargetSeal, b.PreviousRevisionSealOfTargetSeal); comparison != 0 {
		return comparison < 0
	}
	return compareStrings(a.Messages, b.Messages) < 0
}

func compareObjectIDs(a, b []domain.ObjectID) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i].Hex < b[i].Hex {
			return -1
		}
		if a[i].Hex > b[i].Hex {
			return 1
		}
	}
	return len(a) - len(b)
}

func compareStrings(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if a[i] < b[i] {
			return -1
		}
		if a[i] > b[i] {
			return 1
		}
	}
	return len(a) - len(b)
}
