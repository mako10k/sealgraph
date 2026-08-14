package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/history"
	"github.com/mako10k/sealgraph/internal/store"
)

type CandidateBaseState string

const (
	CandidateBaseInitial        CandidateBaseState = "INITIAL"
	CandidateBaseCurrent        CandidateBaseState = "CURRENT"
	CandidateBaseHeadAdvanced   CandidateBaseState = "HEAD_ADVANCED"
	CandidateBaseHeadMissing    CandidateBaseState = "HEAD_MISSING"
	CandidateBaseUnexpectedHead CandidateBaseState = "UNEXPECTED_HEAD"
)

type CandidateInspection struct {
	Candidate   domain.Candidate
	Content     []byte
	CurrentHead *domain.ObjectID
	BaseState   CandidateBaseState
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
		seal, err := r.LoadSeal(ctx, link.TargetSeal)
		if err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("candidate dependency %s@%s is unreadable: %w", link.TargetREF, link.TargetSeal, err)
		}
		if seal.REF != link.TargetREF {
			return CandidateInspection{}, nil, fmt.Errorf("candidate dependency %s@%s is owned by %s; discard or relink the candidate explicitly", link.TargetREF, link.TargetSeal, seal.REF)
		}
	}

	var base *domain.SealPayload
	if candidate.Base != nil {
		payload, err := r.LoadSeal(ctx, *candidate.Base)
		if err != nil {
			return CandidateInspection{}, nil, fmt.Errorf("candidate base %s@%s is unreadable: %w", ref, candidate.Base, err)
		}
		if payload.REF != ref {
			return CandidateInspection{}, nil, fmt.Errorf("candidate base %s belongs to %s, not %s; discard the candidate explicitly", candidate.Base, payload.REF, ref)
		}
		base = &payload
	}

	var currentHead *domain.ObjectID
	head, err := r.refs.Resolve(ctx, ref)
	if err == nil {
		payload, loadErr := r.LoadSeal(ctx, head)
		if loadErr != nil {
			return CandidateInspection{}, nil, fmt.Errorf("current HEAD %s@%s is unreadable: %w", ref, head, loadErr)
		}
		if payload.REF != ref {
			return CandidateInspection{}, nil, fmt.Errorf("current HEAD %s points to a seal owned by %s; repair the REF explicitly", ref, payload.REF)
		}
		headCopy := head
		currentHead = &headCopy
	} else if !errors.Is(err, store.ErrRefNotFound) {
		return CandidateInspection{}, nil, fmt.Errorf("resolve current HEAD for candidate %s: %w", ref, err)
	}

	return CandidateInspection{
		Candidate: candidate, Content: content, CurrentHead: currentHead,
		BaseState: candidateBaseState(candidate.Base, currentHead),
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

func candidateBaseState(base, current *domain.ObjectID) CandidateBaseState {
	switch {
	case base == nil && current == nil:
		return CandidateBaseInitial
	case base == nil:
		return CandidateBaseUnexpectedHead
	case current == nil:
		return CandidateBaseHeadMissing
	case base.Equal(*current):
		return CandidateBaseCurrent
	default:
		return CandidateBaseHeadAdvanced
	}
}

func (r *Repository) Unlink(ctx context.Context, ref, upstreamREF, revision string) (domain.Candidate, error) {
	return withMutation(ctx, r.writer, "unlink candidate", func() (domain.Candidate, error) {
		return r.unlink(ctx, ref, upstreamREF, revision)
	})
}

func (r *Repository) unlink(ctx context.Context, ref, upstreamREF, revision string) (domain.Candidate, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return domain.Candidate{}, err
	}
	if err := domain.ValidateREF(upstreamREF); err != nil {
		return domain.Candidate{}, fmt.Errorf("invalid upstream REF %q: %w", upstreamREF, err)
	}
	var expected *domain.ObjectID
	if revision != "" {
		id, err := r.ResolveSealID(ctx, upstreamREF, revision)
		if err != nil {
			return domain.Candidate{}, fmt.Errorf("resolve unlink precondition %s@%s: %w", upstreamREF, revision, err)
		}
		expected = &id
	}
	candidate, err := r.candidateForEdit(ctx, ref)
	if err != nil {
		return domain.Candidate{}, err
	}
	index := -1
	for i, link := range candidate.Links {
		if link.TargetREF == upstreamREF {
			index = i
			if expected != nil && !link.TargetSeal.Equal(*expected) {
				return domain.Candidate{}, fmt.Errorf("candidate %s dependency %s targets %s, not required generation %s; inspect the candidate and retry with its exact target", ref, upstreamREF, link.TargetSeal, expected)
			}
			break
		}
	}
	if index < 0 {
		return domain.Candidate{}, fmt.Errorf("candidate %s has no dependency on %s; inspect it before retrying", ref, upstreamREF)
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
