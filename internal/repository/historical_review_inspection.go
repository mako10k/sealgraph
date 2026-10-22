package repository

// R: inspect recorded historical judgments separately from current applicability
// and enforce explicit historical admission without changing stale facts.

import (
	"context"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/historicalreview"
)

type HistoricalReviewInspection struct {
	Schema       string                      `json:"schema"`
	SelectedSeal domain.ObjectID             `json:"selected_seal"`
	Candidate    bool                        `json:"candidate"`
	Record       historicalreview.Record     `json:"record"`
	Facts        HistoricalReviewPreparation `json:"current"`
	Applicable   bool                        `json:"applicable"`
	Limitation   string                      `json:"limitation"`
}

func (r *Repository) loadHistoricalReview(ctx context.Context, candidate domainv5.Candidate) (historicalreview.Record, error) {
	for _, attachment := range candidate.Attachments {
		if attachment.Name != historicalreview.AttachmentName {
			continue
		}
		if attachment.MediaType != historicalreview.MediaType {
			return historicalreview.Record{}, fmt.Errorf("historical review has unexpected media type")
		}
		data, err := r.readRepositoryBlobID(ctx, attachment.Blob, "historical review")
		if err != nil {
			return historicalreview.Record{}, err
		}
		record, _, err := historicalreview.Decode(data)
		return record, err
	}
	return historicalreview.Record{}, fmt.Errorf("REVIEW_REQUIRED: record one Candidate review with 'sealgraph review set REF --file PATH' first")
}

func (r *Repository) validateRecordedHistoricalReview(ctx context.Context, candidate domainv5.Candidate, observation headObservation) error {
	record, err := r.loadHistoricalReview(ctx, candidate)
	if err != nil {
		return err
	}
	facts, _, err := r.historicalReviewFacts(ctx, candidate, observation)
	if err != nil {
		return err
	}
	return historicalReviewAdmission(record, facts)
}

func historicalReviewAdmission(record historicalreview.Record, facts HistoricalReviewPreparation) error {
	if err := validateHistoricalReviewBinding(record, facts); err != nil {
		return err
	}
	for i, entry := range record.Entries {
		if entry.Decision != "retain" {
			return fmt.Errorf("REVIEW_UNCONFIRMED: Cause %s remains unreviewed", entry.TargetSeal)
		}
		if facts.Targets[i].Draft {
			return fmt.Errorf("review cannot admit draft Cause %s; keep provisional provenance draft", entry.TargetSeal)
		}
	}
	return nil
}

// InspectHistoricalReview uses an explicit Candidate REF or immutable selector.
// candidate=true never falls back to a HEAD when the Candidate is absent.
func (r *Repository) InspectHistoricalReview(ctx context.Context, selector string, candidate bool) (HistoricalReviewInspection, error) {
	observation, graph, err := r.buildObservation(ctx, "inspect historical review")
	if err != nil {
		return HistoricalReviewInspection{}, err
	}
	var selectedSeal domain.ObjectID
	var subject domainv5.Candidate
	var snapshot candidateSnapshot
	if candidate {
		snapshot, err = r.candidates.LoadSnapshot(selector)
		subject = snapshot.Candidate
	} else {
		var selected ResolvedSelector
		selected, err = r.resolveSelectorObserved(ctx, selector, &observation, graph)
		if err == nil {
			resolved := selected.Resolved
			selectedSeal = selected.ID
			subject = domainv5.Candidate{REF: "review-subject", Content: resolved.Material.Content, Attachments: resolved.Material.Attachments, Root: resolved.Provenance.Root, Draft: resolved.Provenance.Draft, CauseLinks: resolved.Provenance.CauseLinks, Origin: resolved.Provenance.Origin}
		}
	}
	if err != nil {
		return HistoricalReviewInspection{}, err
	}
	if candidate {
		prospective, err := r.prospectiveSeal(subject)
		if err != nil {
			return HistoricalReviewInspection{}, err
		}
		selectedSeal = prospective.ID
	}
	record, err := r.loadHistoricalReview(ctx, subject)
	if err != nil {
		return HistoricalReviewInspection{}, err
	}
	if !candidate {
		subject.REF = record.REF
	}
	facts, _, err := r.historicalReviewFacts(ctx, subject, observation)
	if err != nil {
		return HistoricalReviewInspection{}, err
	}
	limitation := "executor claim only; not proof of understanding, correctness or approval; STALE remains independently derived"
	admission := historicalReviewAdmission(record, facts)
	if admission != nil {
		limitation = admission.Error() + "; " + limitation
	}
	result := HistoricalReviewInspection{Schema: "sealgraph/historical-review-show/v1", SelectedSeal: selectedSeal, Candidate: candidate, Record: record, Facts: facts, Applicable: admission == nil && !subject.Draft && !subject.Root, Limitation: limitation}
	if err := r.revalidateHeads(ctx, observation, "inspect historical review"); err != nil {
		return HistoricalReviewInspection{}, err
	}
	if candidate {
		current, err := r.candidates.LoadSnapshot(selector)
		if err != nil || string(current.Bytes) != string(snapshot.Bytes) {
			return HistoricalReviewInspection{}, ErrCandidateChanged
		}
	}
	return result, nil
}
