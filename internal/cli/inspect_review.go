package cli

// R: expose stored historical judgments separately for Candidate and HEAD.

import (
	"context"
	"fmt"
	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/historicalreview"
	"github.com/mako10k/sealgraph/internal/repository"
)

func inspectReviews(ctx context.Context, repo *repository.Repository, status repository.RefStatus) ([]inspectReview, error) {
	result := []inspectReview{}
	if status.Unsealed {
		selected, err := repo.InspectCandidate(ctx, status.REF)
		if err != nil {
			return nil, err
		}
		item, err := inspectReviewFor(ctx, repo, status.REF, true, selected.Prospective.ID, selected.Candidate.Attachments)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	if status.Head != nil {
		selected, err := repo.LoadSeal(ctx, *status.Head)
		if err != nil {
			return nil, err
		}
		item, err := inspectReviewFor(ctx, repo, "@"+status.Head.String(), false, *status.Head, selected.Material.Attachments)
		if err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, nil
}

func inspectReviewFor(ctx context.Context, repo *repository.Repository, selector string, candidate bool, id domain.ObjectID, attachments []domainv5.Attachment) (inspectReview, error) {
	baseline := "head"
	if candidate {
		baseline = "candidate"
	}
	result := inspectReview{Baseline: baseline, SealID: id.String(), State: "NONE"}
	for _, attachment := range attachments {
		if attachment.Name != historicalreview.AttachmentName {
			continue
		}
		review, err := repo.InspectHistoricalReview(ctx, selector, candidate)
		if err != nil {
			return result, err
		}
		if !review.SelectedSeal.Equal(id) {
			return result, fmt.Errorf("historical review subject changed during inspection")
		}
		result.State = "RECORDED"
		result.Applicable = &review.Applicable
		result.Record = &review
		break
	}
	return result, nil
}
