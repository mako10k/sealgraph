// Package v6 defines the format-6 typed records. The aliases intentionally
// share the repository's logical view with v5 while canonical codecs keep the
// persisted generations strictly separate.
package v6

import (
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

const (
	SealSchema           = "sealgraph/seal/v6"
	MaterialSchema       = domainv5.MaterialSchema
	ProvenanceSchema     = "sealgraph/provenance/v2"
	CandidateSchema      = "sealgraph/candidate/v6"
	UpstreamChangeSchema = "sealgraph/upstream-change/v2"
)

type Seal = domainv5.Seal
type Material = domainv5.Material
type Attachment = domainv5.Attachment
type Provenance = domainv5.Provenance
type CauseLink = domainv5.CauseLink
type MetadataEntry = domainv5.MetadataEntry
type Candidate = domainv5.Candidate
type ResolvedSeal = domainv5.ResolvedSeal
type AssertionSource = domainv5.AssertionSource
type RevisionObservation = domainv5.RevisionObservation

// UpstreamChange is the assessment-free identity preimage accepted by ADR
// 0030. It intentionally contains no assessment reference.
type UpstreamChange struct {
	Schema          string           `json:"schema"`
	BeforeSeal      *domain.ObjectID `json:"before_seal"`
	AfterMaterial   domain.ObjectID  `json:"after_material"`
	AfterRoot       bool             `json:"after_root"`
	AfterDraft      bool             `json:"after_draft"`
	AfterCauseLinks []CauseLink      `json:"after_cause_links"`
}
