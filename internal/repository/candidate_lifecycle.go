package repository

import (
	"context"
	"errors"
	"fmt"

	canonicalv5 "github.com/mako10k/sealgraph/internal/canonical/v5"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/store"
)

type CandidateExpectedHeadState string

const (
	CandidateExpectedAbsent  CandidateExpectedHeadState = "EXPECTED_ABSENT"
	CandidateExpectedCurrent CandidateExpectedHeadState = "EXPECTED_CURRENT"
	CandidateHeadAdvanced    CandidateExpectedHeadState = "HEAD_ADVANCED"
	CandidateHeadMissing     CandidateExpectedHeadState = "HEAD_MISSING"
	CandidateUnexpectedHead  CandidateExpectedHeadState = "UNEXPECTED_HEAD"
)

type CandidateInspection struct {
	Candidate         domainv5.Candidate
	Prospective       domainv5.ResolvedSeal
	Content           []byte
	CurrentHead       *domain.ObjectID
	ExpectedHeadState CandidateExpectedHeadState
}

type CandidateDiffResult struct {
	Inspection CandidateInspection
	Baseline   *domainv5.ResolvedSeal
}

func (r *Repository) InspectCandidate(ctx context.Context, ref string) (CandidateInspection, error) {
	inspection, _, err := r.inspectCandidate(ctx, ref)
	return inspection, err
}

func (r *Repository) DiffCandidate(ctx context.Context, ref string) (CandidateDiffResult, error) {
	inspection, baseline, err := r.inspectCandidate(ctx, ref)
	if err != nil {
		return CandidateDiffResult{}, err
	}
	return CandidateDiffResult{Inspection: inspection, Baseline: baseline}, nil
}

func (r *Repository) inspectCandidate(ctx context.Context, ref string) (CandidateInspection, *domainv5.ResolvedSeal, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return CandidateInspection{}, nil, err
	}
	candidate, err := r.candidates.Load(ref)
	if err != nil {
		if errors.Is(err, ErrCandidateNotFound) {
			return CandidateInspection{}, nil, fmt.Errorf("REF %s has no working Candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
		}
		return CandidateInspection{}, nil, fmt.Errorf("Candidate %s cannot be inspected: %w; use 'sealgraph candidate discard %s' only if you intend to remove it", ref, err, ref)
	}
	content, err := r.readRepositoryBlobID(ctx, candidate.Content, fmt.Sprintf("Candidate content for %s", ref))
	if err != nil {
		return CandidateInspection{}, nil, err
	}
	for _, attachment := range candidate.Attachments {
		if _, err := r.readRepositoryBlobID(ctx, attachment.Blob, fmt.Sprintf("Candidate attachment %q for %s", attachment.Name, ref)); err != nil {
			return CandidateInspection{}, nil, err
		}
	}
	for _, link := range candidate.CauseLinks {
		if _, err := r.LoadSeal(ctx, link.TargetSeal); err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("Candidate Cause target %s is unreadable: %w", link.TargetSeal, err)
		}
		for _, previous := range link.PreviousRevisionSealOfTargetSeal {
			if _, err := r.LoadSeal(ctx, previous); err != nil {
				return CandidateInspection{}, nil, fmt.Errorf("Candidate previous revision %s for target %s is unreadable: %w", previous, link.TargetSeal, err)
			}
		}
	}
	prospective, err := prospectiveSeal(candidate)
	if err != nil {
		return CandidateInspection{}, nil, fmt.Errorf("derive prospective Candidate IDs: %w", err)
	}
	var baseline *domainv5.ResolvedSeal
	if candidate.ExpectedREFHead != nil {
		value, err := r.LoadSeal(ctx, *candidate.ExpectedREFHead)
		if err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("Candidate publication baseline %s is unreadable: %w", candidate.ExpectedREFHead, err)
		}
		baseline = &value
	}
	var currentHead *domain.ObjectID
	head, err := r.refs.Resolve(ctx, ref)
	if err == nil {
		if _, loadErr := r.LoadSeal(ctx, head); loadErr != nil {
			return CandidateInspection{}, nil, fmt.Errorf("current HEAD %s@%s is unreadable: %w", ref, head, loadErr)
		}
		headCopy := head
		currentHead = &headCopy
	} else if !errors.Is(err, store.ErrRefNotFound) {
		return CandidateInspection{}, nil, fmt.Errorf("resolve current HEAD for Candidate %s: %w", ref, err)
	}
	prospective.ContentBytes = len(content)
	return CandidateInspection{Candidate: candidate, Prospective: prospective, Content: content, CurrentHead: currentHead,
		ExpectedHeadState: candidateExpectedHeadState(candidate.ExpectedREFHead, currentHead)}, baseline, nil
}

func prospectiveSeal(candidate domainv5.Candidate) (domainv5.ResolvedSeal, error) {
	material := domainv5.Material{Schema: domainv5.MaterialSchema, Content: candidate.Content, Attachments: candidate.Attachments}
	materialBytes, err := canonicalv5.EncodeMaterial(material)
	if err != nil {
		return domainv5.ResolvedSeal{}, err
	}
	materialID := domain.ComputeNativeBlobID(materialBytes)
	provenance := domainv5.Provenance{Schema: domainv5.ProvenanceSchema, Root: candidate.Root, Draft: candidate.Draft, CauseLinks: candidate.CauseLinks}
	provenanceBytes, err := canonicalv5.EncodeProvenance(provenance)
	if err != nil {
		return domainv5.ResolvedSeal{}, err
	}
	provenanceID := domain.ComputeNativeBlobID(provenanceBytes)
	seal := domainv5.Seal{Schema: domainv5.SealSchema, Material: materialID, Provenance: provenanceID}
	sealBytes, err := canonicalv5.EncodeSeal(seal)
	if err != nil {
		return domainv5.ResolvedSeal{}, err
	}
	return domainv5.ResolvedSeal{ID: domain.ComputeNativeBlobID(sealBytes), Seal: seal, Material: material, Provenance: provenance}, nil
}

func (r *Repository) readRepositoryBlobID(ctx context.Context, id domain.ObjectID, description string) ([]byte, error) {
	object, err := r.objects.ReadObject(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%s is unreadable: %w", description, err)
	}
	if object.Type != domain.BlobType {
		return nil, fmt.Errorf("%s has object type %s, expected blob", description, object.Type)
	}
	return object.Data, nil
}

func candidateExpectedHeadState(expected, current *domain.ObjectID) CandidateExpectedHeadState {
	switch {
	case expected == nil && current == nil:
		return CandidateExpectedAbsent
	case expected == nil:
		return CandidateUnexpectedHead
	case current == nil:
		return CandidateHeadMissing
	case expected.Equal(*current):
		return CandidateExpectedCurrent
	default:
		return CandidateHeadAdvanced
	}
}

func (r *Repository) Unlink(ctx context.Context, ref, targetSelector string) (domainv5.Candidate, error) {
	return withMutation(ctx, r.writer, "unlink Candidate", func() (domainv5.Candidate, error) {
		if err := domain.ValidateREF(ref); err != nil {
			return domainv5.Candidate{}, err
		}
		observation, graph, err := r.buildObservation(ctx, "unlink Candidate")
		if err != nil {
			return domainv5.Candidate{}, err
		}
		resolved, err := r.resolveSelectorObserved(ctx, targetSelector, &observation, graph)
		if err != nil {
			return domainv5.Candidate{}, fmt.Errorf("resolve unlink target %q: %w", targetSelector, err)
		}
		edit, err := r.candidateForObservedEdit(ctx, ref, observation, graph)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate := edit.Candidate
		index := -1
		for i, link := range candidate.CauseLinks {
			if link.TargetSeal.Equal(resolved.ID) {
				index = i
				break
			}
		}
		if index < 0 {
			return domainv5.Candidate{}, fmt.Errorf("Candidate %s has no Cause Link to %s", ref, resolved.ID)
		}
		if !candidate.Root && len(candidate.CauseLinks) == 1 {
			return domainv5.Candidate{}, fmt.Errorf("unlink would leave non-root Candidate %s without a Cause Link; use add --root --clear-cause-links for an explicit root transition", ref)
		}
		candidate.CauseLinks = append(candidate.CauseLinks[:index], candidate.CauseLinks[index+1:]...)
		if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
			return domainv5.Candidate{}, err
		}
		if err := r.candidates.SaveIfUnchanged(candidate, edit.Bytes, edit.Exists); err != nil {
			return domainv5.Candidate{}, fmt.Errorf("save Candidate %s after unlink: %w", ref, err)
		}
		return r.candidates.Load(ref)
	})
}

func (r *Repository) DiscardCandidate(ctx context.Context, ref string) error {
	_, err := withMutation(ctx, r.writer, "discard Candidate", func() (struct{}, error) {
		if err := r.candidates.Discard(ref); err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				return struct{}{}, fmt.Errorf("REF %s has no working Candidate; nothing was discarded", ref)
			}
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}
