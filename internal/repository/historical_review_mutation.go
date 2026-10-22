package repository

// R: attach or remove one explicit historical-review claim on one Candidate
// using expected-old mutation, without changing Cause records or publishing.

import (
	"context"
	"fmt"

	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/historicalreview"
)

func (r *Repository) SetHistoricalReview(ctx context.Context, ref string, input []byte) (domainv5.Candidate, error) {
	record, data, err := historicalreview.Decode(input)
	if err != nil {
		return domainv5.Candidate{}, err
	}
	return withMutation(ctx, r.writer, "record historical review", func() (domainv5.Candidate, error) {
		snapshot, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		observation, err := r.observeHeads(ctx, "record historical review")
		if err != nil {
			return domainv5.Candidate{}, err
		}
		facts, _, err := r.historicalReviewFacts(ctx, snapshot.Candidate, observation)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		if err := validateHistoricalReviewBinding(record, facts); err != nil {
			return domainv5.Candidate{}, err
		}
		candidate, err := checkedReviewCandidate(snapshot.Candidate)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		id, err := r.objects.WriteBlob(ctx, data)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate.Attachments = append(candidate.Attachments, domainv5.Attachment{Name: historicalreview.AttachmentName, MediaType: historicalreview.MediaType, Blob: id})
		if err := r.validateCandidateMutation(ctx, candidate, observation); err != nil {
			return domainv5.Candidate{}, err
		}
		if err := r.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
			return domainv5.Candidate{}, err
		}
		return r.candidates.Load(ref)
	})
}

func checkedReviewCandidate(candidate domainv5.Candidate) (domainv5.Candidate, error) {
	for _, attachment := range candidate.Attachments {
		if attachment.Name == historicalreview.AttachmentName && attachment.MediaType != historicalreview.MediaType {
			return domainv5.Candidate{}, fmt.Errorf("review attachment name is occupied by another media type; preserve it and select a separate workflow")
		}
	}
	return withoutHistoricalReview(candidate), nil
}

func (r *Repository) ClearHistoricalReview(ctx context.Context, ref string) (domainv5.Candidate, error) {
	return withMutation(ctx, r.writer, "clear historical review", func() (domainv5.Candidate, error) {
		snapshot, err := r.candidates.LoadSnapshot(ref)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		candidate, err := checkedReviewCandidate(snapshot.Candidate)
		if err != nil {
			return domainv5.Candidate{}, err
		}
		if len(candidate.Attachments) == len(snapshot.Candidate.Attachments) {
			return domainv5.Candidate{}, fmt.Errorf("Candidate %s has no historical review", ref)
		}
		if err := r.candidates.SaveIfUnchanged(candidate, snapshot.Bytes, true); err != nil {
			return domainv5.Candidate{}, err
		}
		return r.candidates.Load(ref)
	})
}
