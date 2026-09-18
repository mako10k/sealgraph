// Package v7 defines the format-7 typed records. Legacy semantic records are
// aliases so shared graph/material behavior remains consistent while the v7
// codec keeps schema pairing and Origin identity generation-specific.
package v7

import (
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

const (
	SealSchema           = "sealgraph/seal/v7"
	MaterialSchema       = domainv5.MaterialSchema
	ProvenanceSchema     = "sealgraph/provenance/v3"
	CandidateSchema      = "sealgraph/candidate/v7"
	OriginMapSchema      = "sealgraph/origin-map/v1"
	SourceSnapshotSchema = "sealgraph/source-snapshot/v1"
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

// SourceSnapshot records one immutable source file representation.
type SourceSnapshot struct {
	Schema    string
	SourceKey string
	Content   domain.ObjectID
}

// OriginRun describes one contiguous range of candidate content bytes.
// External runs use Snapshot and SourceStart; untraced runs leave both zero.
type OriginRun struct {
	Kind        string
	Length      uint64
	Snapshot    domain.ObjectID
	SourceStart uint64
}

// OriginMap is an immutable, content-ordered correspondence map.
type OriginMap struct {
	Schema  string
	Content domain.ObjectID
	Runs    []OriginRun
}
