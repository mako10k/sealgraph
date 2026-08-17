package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/history"
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
	Candidate         domain.Candidate
	Content           []byte
	CurrentHead       *domain.ObjectID
	ExpectedHeadState CandidateExpectedHeadState
}

type CandidateDiffResult struct {
	Inspection CandidateInspection
	Diff       history.CandidateDiff
}

func (r *Repository) InspectCandidate(ctx context.Context, ref string) (CandidateInspection, error) {
	inspection, _, err := r.inspectCandidate(ctx, ref)
	return inspection, err
}

func (r *Repository) DiffCandidate(ctx context.Context, ref string) (CandidateDiffResult, error) {
	inspection, base, err := r.inspectCandidate(ctx, ref)
	if err != nil {
		return CandidateDiffResult{}, err
	}
	diff, err := history.DiffCandidate(base, inspection.Candidate)
	if err != nil {
		return CandidateDiffResult{}, err
	}
	return CandidateDiffResult{Inspection: inspection, Diff: diff}, nil
}

func (r *Repository) inspectCandidate(ctx context.Context, ref string) (CandidateInspection, *domain.SealPayload, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return CandidateInspection{}, nil, err
	}
	candidate, err := r.candidates.Load(ref)
	if err != nil {
		if errors.Is(err, ErrCandidateNotFound) {
			return CandidateInspection{}, nil, fmt.Errorf("REF %s has no working candidate; run 'sealgraph add' or 'sealgraph link' first", ref)
		}
		return CandidateInspection{}, nil, fmt.Errorf("candidate %s cannot be inspected: %w; use 'sealgraph candidate discard %s' only if you intend to remove that unsealed state", ref, err, ref)
	}
	content, err := r.readRepositoryBlob(ctx, candidate.Content, fmt.Sprintf("candidate content for %s", ref))
	if err != nil {
		return CandidateInspection{}, nil, err
	}
	for _, attachment := range candidate.Attachments {
		if _, err := r.readRepositoryBlob(ctx, attachment.Blob, fmt.Sprintf("candidate attachment %q for %s", attachment.Name, ref)); err != nil {
			return CandidateInspection{}, nil, err
		}
	}
	for _, link := range candidate.Links {
		if _, err := r.LoadSeal(ctx, link.TargetSeal); err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("candidate Cause target %s is unreadable: %w", link.TargetSeal, err)
		}
	}

	var base *domain.SealPayload
	if candidate.ParentRevision != nil {
		payload, err := r.LoadSeal(ctx, *candidate.ParentRevision)
		if err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("candidate parent revision %s is unreadable: %w", candidate.ParentRevision, err)
		}
		base = &payload
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
		return CandidateInspection{}, nil, fmt.Errorf("resolve current HEAD for candidate %s: %w", ref, err)
	}

	return CandidateInspection{
		Candidate: candidate, Content: content, CurrentHead: currentHead,
		ExpectedHeadState: candidateExpectedHeadState(candidate.ExpectedREFHead, currentHead),
	}, base, nil
}

func (r *Repository) readRepositoryBlob(ctx context.Context, ref domain.ContentRef, description string) ([]byte, error) {
	object, err := r.objects.ReadObject(ctx, ref.ID)
	if err != nil {
		return nil, fmt.Errorf("%s is unreadable: %w", description, err)
	}
	if object.Type != ref.Type {
		return nil, fmt.Errorf("%s has object type %s, expected %s", description, object.Type, ref.Type)
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

func (r *Repository) Unlink(ctx context.Context, ref, upstreamSelector string) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "unlink candidate", func() (domain.Candidate, error) {
		return r.unlink(ctx, ref, upstreamSelector)
	})
}

func (r *Repository) unlink(ctx context.Context, ref, upstreamSelector string) (domain.Candidate, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return domain.Candidate{}, err
	}
	resolved, err := r.ResolveSelector(ctx, upstreamSelector)
	if err != nil {
		return domain.Candidate{}, fmt.Errorf("resolve unlink target %q: %w", upstreamSelector, err)
	}
	candidate, err := r.candidateForEdit(ctx, ref)
	if err != nil {
		return domain.Candidate{}, err
	}
	index := -1
	for i, link := range candidate.Links {
		if link.TargetSeal.Equal(resolved.ID) {
			index = i
			break
		}
	}
	if index < 0 {
		return domain.Candidate{}, fmt.Errorf("candidate %s has no Cause link to %s; inspect it and use the exact displayed @SealID selector", ref, resolved.ID)
	}
	candidate.Links = append(candidate.Links[:index], candidate.Links[index+1:]...)
	if err := r.candidates.Save(candidate); err != nil {
		return domain.Candidate{}, fmt.Errorf("save candidate %s after unlink: %w", ref, err)
	}
	return candidate, nil
}

func (r *Repository) DiscardCandidate(ctx context.Context, ref string) error {
	_, err := withMutation(ctx, r.writer, "discard candidate", func() (struct{}, error) {
		if err := r.candidates.Discard(ref); err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				return struct{}{}, fmt.Errorf("REF %s has no working candidate; nothing was discarded", ref)
			}
			return struct{}{}, err
		}
		return struct{}{}, nil
	})
	return err
}
