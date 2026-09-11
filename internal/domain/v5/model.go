// Package v5 defines the domain model selected by the accepted format-5 ADRs.
// The explicit version boundary prevents migration-only format-4 payloads from
// being treated as live format-5 repository state.
package v5

import (
	"encoding/json"

	"github.com/mako10k/sealgraph/internal/domain"
)

const (
	SealSchema       = "sealgraph/seal/v5"
	MaterialSchema   = "sealgraph/material/v1"
	ProvenanceSchema = "sealgraph/provenance/v1"
	CandidateSchema  = "sealgraph/candidate/v5"
)

// Seal is the minimal immutable join of one Material and one Provenance Blob.
type Seal struct {
	Schema     string          `json:"schema"`
	Material   domain.ObjectID `json:"material"`
	Provenance domain.ObjectID `json:"provenance"`
}

// Material contains the exact content and named attachment Blob identities.
type Material struct {
	Schema      string          `json:"schema"`
	Content     domain.ObjectID `json:"content"`
	Attachments []Attachment    `json:"attachments"`
}

// Attachment is one immutable named Blob included in Material identity.
type Attachment struct {
	Name      string          `json:"name"`
	MediaType string          `json:"media_type"`
	Blob      domain.ObjectID `json:"blob"`
}

// Provenance contains graph claims and no mutable repository observation.
type Provenance struct {
	Schema     string      `json:"schema"`
	Root       bool        `json:"root"`
	Draft      bool        `json:"draft"`
	CauseLinks []CauseLink `json:"cause_links"`
}

// CauseLink scopes revision assertions and messages to one exact target Seal.
type CauseLink struct {
	TargetSeal                       domain.ObjectID   `json:"target_seal"`
	PreviousRevisionSealOfTargetSeal []domain.ObjectID `json:"previous_revision_seal_of_target_seal"`
	Messages                         []string          `json:"messages"`
	// Metadata is part of the shared in-memory Link view. Exact format-5
	// codecs require it to be empty and never encode it; format 6 codecs
	// require and encode the member. This keeps historical v5 bytes exact.
	Metadata []MetadataEntry `json:"metadata"`
}

// MetadataEntry is an opaque, namespaced, identity-bearing Link annotation.
// Value retains its exact JSON input until the format-6 codec validates and
// canonicalizes the closed metadata value grammar.
type MetadataEntry struct {
	Namespace string          `json:"namespace"`
	Schema    *string         `json:"schema"`
	Value     json.RawMessage `json:"value"`
}

// Candidate is mutable parentless publication intent. Exact persisted bytes,
// rather than a BlobID, are its optimistic mutation version.
type Candidate struct {
	Schema          string           `json:"schema"`
	REF             string           `json:"ref"`
	ExpectedREFHead *domain.ObjectID `json:"expected_ref_head"`
	Content         domain.ObjectID  `json:"content"`
	Attachments     []Attachment     `json:"attachments"`
	Root            bool             `json:"root"`
	Draft           bool             `json:"draft"`
	CauseLinks      []CauseLink      `json:"cause_links"`
}

// ResolvedSeal is the repository reader's typed join of a Seal Blob and the
// exact Material and Provenance Blobs it names. ID is the Seal BlobID; the
// embedded records retain the two typed child IDs and their validated values.
// It is an in-memory view and is never encoded as a canonical object.
type ResolvedSeal struct {
	ID           domain.ObjectID
	Seal         Seal
	Material     Material
	Provenance   Provenance
	ContentBytes int
}

// AssertionSource retains the exact immutable observer that contributed one
// Cause-Link-scoped revision assertion to an observation.
type AssertionSource struct {
	ObserverSeal             domain.ObjectID
	ObserverSealSchema       string
	ObserverProvenance       domain.ObjectID
	ObserverProvenanceSchema string
	CauseLink                CauseLink
}

// RevisionObservation exposes every observed assertion for one target and the
// sorted structural union of the asserted previous SealIDs.
type RevisionObservation struct {
	TargetSeal         domain.ObjectID
	PreviousStates     []string
	Assertions         []AssertionSource
	StructuralPrevious []domain.ObjectID
}
