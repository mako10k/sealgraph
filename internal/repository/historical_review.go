package repository

// R: derive version-bound historical-review subjects and complete Cause facts
// from a coherent repository observation, without producing review judgments.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/historicalreview"
)

type HistoricalReviewTarget struct {
	TargetSeal    domain.ObjectID   `json:"target_seal"`
	ComparedSeals []domain.ObjectID `json:"compared_seals"`
	RevisionState RevisionState     `json:"revision_state"`
	Draft         bool              `json:"draft"`
}
type HistoricalReviewPreparation struct {
	Schema      string                   `json:"schema"`
	REF         string                   `json:"ref"`
	SubjectSeal domain.ObjectID          `json:"subject_seal"`
	OtherHeads  []historicalreview.Head  `json:"other_heads"`
	Targets     []HistoricalReviewTarget `json:"targets"`
}

func withoutHistoricalReview(candidate domainv5.Candidate) domainv5.Candidate {
	attachments := make([]domainv5.Attachment, 0, len(candidate.Attachments))
	for _, attachment := range candidate.Attachments {
		if attachment.Name != historicalreview.AttachmentName {
			attachments = append(attachments, attachment)
		}
	}
	candidate.Attachments = attachments
	return candidate
}

func (r *Repository) historicalReviewFacts(ctx context.Context, candidate domainv5.Candidate, observation headObservation) (HistoricalReviewPreparation, *observedGraph, error) {
	prospective, err := r.prospectiveSeal(withoutHistoricalReview(candidate))
	if err != nil {
		return HistoricalReviewPreparation{}, nil, err
	}
	heads := cloneHeads(observation.heads)
	heads[candidate.REF] = prospective.ID
	graph, err := r.buildObservedGraph(ctx, heads, map[string]domainv5.ResolvedSeal{prospective.ID.String(): prospective})
	if err != nil {
		return HistoricalReviewPreparation{}, nil, err
	}
	result := HistoricalReviewPreparation{Schema: "sealgraph/historical-review-prepare/v1", REF: candidate.REF, SubjectSeal: prospective.ID, OtherHeads: []historicalreview.Head{}, Targets: []HistoricalReviewTarget{}}
	for ref, id := range observation.heads {
		if ref != candidate.REF {
			result.OtherHeads = append(result.OtherHeads, historicalreview.Head{REF: ref, Seal: id})
		}
	}
	sort.Slice(result.OtherHeads, func(i, j int) bool { return result.OtherHeads[i].REF < result.OtherHeads[j].REF })
	for _, id := range historicalCauseTargets(candidate, graph) {
		result.Targets = append(result.Targets, HistoricalReviewTarget{TargetSeal: id, ComparedSeals: reviewComparisonLeaves(id, graph), RevisionState: graph.state(id), Draft: graph.nodes[id.String()].Provenance.Draft})
	}
	return result, graph, nil
}

func historicalCauseTargets(candidate domainv5.Candidate, graph *observedGraph) []domain.ObjectID {
	stack := []domain.ObjectID{}
	for _, link := range candidate.CauseLinks {
		stack = append(stack, link.TargetSeal)
	}
	seen := map[string]bool{}
	result := []domain.ObjectID{}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		result = append(result, id)
		stack = append(stack, graph.causes[id.String()]...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func reviewComparisonLeaves(target domain.ObjectID, graph *observedGraph) []domain.ObjectID {
	result := []domain.ObjectID{}
	stack := []domain.ObjectID{target}
	seen := map[string]bool{}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		if graph.isActiveLeaf(id) {
			result = append(result, id)
		}
		stack = append(stack, graph.children[id.String()]...)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func (r *Repository) PrepareHistoricalReview(ctx context.Context, ref string) (HistoricalReviewPreparation, error) {
	snapshot, err := r.candidates.LoadSnapshot(ref)
	if err != nil {
		return HistoricalReviewPreparation{}, err
	}
	observation, err := r.observeHeads(ctx, "prepare historical review")
	if err != nil {
		return HistoricalReviewPreparation{}, err
	}
	result, _, err := r.historicalReviewFacts(ctx, snapshot.Candidate, observation)
	if err != nil {
		return HistoricalReviewPreparation{}, err
	}
	if err := r.revalidateHeads(ctx, observation, "prepare historical review"); err != nil {
		return HistoricalReviewPreparation{}, err
	}
	current, err := r.candidates.LoadSnapshot(ref)
	if err != nil || string(current.Bytes) != string(snapshot.Bytes) {
		return HistoricalReviewPreparation{}, fmt.Errorf("Candidate changed while preparing review: %w", ErrCandidateChanged)
	}
	return result, nil
}

func validateHistoricalReviewBinding(record historicalreview.Record, facts HistoricalReviewPreparation) error {
	if record.REF != facts.REF || !record.SubjectSeal.Equal(facts.SubjectSeal) {
		return fmt.Errorf("REVIEW_SUBJECT_CHANGED: review must match the exact REF, content, origin, attachments and Causes")
	}
	recordedHeads, _ := json.Marshal(record.OtherHeads)
	currentHeads, _ := json.Marshal(facts.OtherHeads)
	if string(recordedHeads) != string(currentHeads) {
		return fmt.Errorf("REVIEW_OBSERVATION_CHANGED: other REF heads changed; inspect differences and record individual judgments again")
	}
	if len(record.Entries) != len(facts.Targets) {
		return fmt.Errorf("REVIEW_COVERAGE: exactly one entry is required for every direct and transitive Cause target")
	}
	for i, target := range facts.Targets {
		entry := record.Entries[i]
		left, _ := json.Marshal(entry.ComparedSeals)
		right, _ := json.Marshal(target.ComparedSeals)
		if !entry.TargetSeal.Equal(target.TargetSeal) || string(left) != string(right) {
			return fmt.Errorf("REVIEW_COVERAGE: target/comparison versions differ for %s", target.TargetSeal)
		}
	}
	return nil
}
