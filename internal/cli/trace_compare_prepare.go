package cli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/repository"
)

type traceComparePreparedSelection struct {
	selection traceShowSelection
	baselines []repository.TraceOwnBaseline
	graph     *repository.DirectionGraphResult
	heads     *repository.DirectionGraphHeads
	candidate bool
}

func prepareTraceCompareNoEstimate(ctx context.Context, repo *repository.Repository, ref, seal singleString, maxGraphVisits int) (traceCompareV2Document, error) {
	return prepareTraceCompare(ctx, repo, ref, seal, maxGraphVisits, false)
}

func prepareTraceCompareWithEstimate(ctx context.Context, repo *repository.Repository, ref, seal singleString, maxGraphVisits int) (traceCompareV2Document, error) {
	return prepareTraceCompare(ctx, repo, ref, seal, maxGraphVisits, true)
}

func prepareTraceCompare(ctx context.Context, repo *repository.Repository, ref, seal singleString, maxGraphVisits int, estimate bool) (traceCompareV2Document, error) {
	if maxGraphVisits <= 0 {
		return traceCompareV2Document{}, fmt.Errorf("max graph visits must be positive")
	}
	beforeBinding, err := traceCompareBindingDigest(repo)
	if err != nil {
		return traceCompareV2Document{}, err
	}
	var declarations []repository.TraceCorrespondenceRecord
	var declarationDigest string
	if estimate {
		declarations, err = repo.TraceCorrespondenceList()
		if err != nil {
			return traceCompareV2Document{}, err
		}
		declarationDigest, err = traceCorrespondenceDigest(declarations)
		if err != nil {
			return traceCompareV2Document{}, err
		}
	}
	prepared, err := selectTraceCompareBaselines(ctx, repo, ref, seal, maxGraphVisits)
	if err != nil {
		return traceCompareV2Document{}, err
	}
	var results []repository.TraceCompareOwnResult
	var sources []repository.TraceOwnSourceObservation
	if estimate {
		results, sources, err = repo.TraceCompareOwnBatchEstimated(ctx, prepared.baselines, declarations)
	} else {
		results, sources, err = repo.TraceCompareOwnBatch(ctx, prepared.baselines)
	}
	if err != nil {
		return traceCompareV2Document{}, err
	}
	doc, err := assembleTraceCompare(prepared, results, sources, beforeBinding, maxGraphVisits)
	if err != nil {
		return traceCompareV2Document{}, err
	}
	if estimate {
		doc.Observation.DeclarationDigest = &declarationDigest
	}
	if err := revalidateTraceCompare(ctx, repo, prepared, ref, beforeBinding, declarationDigest, estimate); err != nil {
		return traceCompareV2Document{}, err
	}
	return doc, nil
}

func revalidateTraceCompare(ctx context.Context, repo *repository.Repository, prepared traceComparePreparedSelection, ref singleString, beforeBinding, declarationDigest string, estimate bool) error {
	var err error
	if prepared.graph != nil {
		err = repo.RevalidateTraceDirectionGraph(ctx, *prepared.graph)
	} else {
		err = repo.RevalidateTraceDirectionHeads(ctx, *prepared.heads)
	}
	if err != nil {
		return err
	}
	afterBinding, err := traceCompareBindingDigest(repo)
	if err != nil || afterBinding != beforeBinding {
		return fmt.Errorf("Trace source bindings changed during comparison: %v", err)
	}
	if estimate {
		currentDeclarations, err := repo.TraceCorrespondenceList()
		if err != nil {
			return err
		}
		afterDigest, err := traceCorrespondenceDigest(currentDeclarations)
		if err != nil || afterDigest != declarationDigest {
			return fmt.Errorf("Trace correspondence declarations changed during comparison: %v", err)
		}
	}
	if ref.set && !prepared.candidate {
		_, candidateErr := repo.CandidateExactDigest(ctx, ref.value)
		if !errors.Is(candidateErr, repository.ErrCandidateNotFound) {
			return fmt.Errorf("Candidate %s appeared or became unreadable during comparison: %v", ref.value, candidateErr)
		}
	}
	return nil
}

func selectTraceCompareBaselines(ctx context.Context, repo *repository.Repository, ref, seal singleString, maxGraphVisits int) (traceComparePreparedSelection, error) {
	prepared := traceComparePreparedSelection{}
	var center *domain.ObjectID
	if seal.set {
		id, err := resolveTraceCompareSeal(ctx, repo, seal.value)
		if err != nil {
			return prepared, err
		}
		center = &id
		idText := id.String()
		prepared.selection = traceShowSelection{Kind: "seal", Requested: seal.value, ResolvedSealID: &idText}
	} else {
		prepared.selection = traceShowSelection{Kind: "ref", Requested: ref.value}
		_, err := repo.CandidateExactDigest(ctx, ref.value)
		if err == nil {
			prepared.candidate = true
			prepared.baselines = append(prepared.baselines, repository.TraceOwnBaseline{Kind: repository.TraceOwnCandidate, REF: ref.value})
		} else if !errors.Is(err, repository.ErrCandidateNotFound) {
			return prepared, err
		}
		head, err := repo.CurrentREFHead(ctx, ref.value)
		if err != nil {
			return prepared, err
		}
		center = head
		if head != nil {
			idText := head.String()
			prepared.selection.ResolvedSealID = &idText
		} else if !prepared.candidate {
			return prepared, fmt.Errorf("REF %s has no Candidate or HEAD", ref.value)
		}
	}
	if center == nil {
		heads, err := repo.TraceDirectionHeads(ctx)
		if err != nil {
			return prepared, err
		}
		prepared.heads = &heads
		return prepared, nil
	}
	graph, err := repo.TraceDirectionGraph(ctx, *center, maxGraphVisits)
	if err != nil {
		return prepared, err
	}
	prepared.graph = &graph
	if ref.set {
		prepared.baselines = append(prepared.baselines, repository.TraceOwnBaseline{Kind: repository.TraceOwnHead, REF: ref.value})
	} else {
		prepared.baselines = append(prepared.baselines, repository.TraceOwnBaseline{Kind: repository.TraceOwnSeal, SealID: center})
	}
	for _, id := range graph.ObservedSealIDs {
		if !id.Equal(*center) {
			copyID := id
			prepared.baselines = append(prepared.baselines, repository.TraceOwnBaseline{Kind: repository.TraceOwnSeal, SealID: &copyID})
		}
	}
	return prepared, nil
}

func resolveTraceCompareSeal(ctx context.Context, repo *repository.Repository, text string) (domain.ObjectID, error) {
	selector, err := repository.ParseSelector(text)
	if err != nil {
		return domain.ObjectID{}, err
	}
	if selector.Kind == repository.SelectorCurrentREF {
		return domain.ObjectID{}, fmt.Errorf("trace compare --seal requires an immutable Seal selector")
	}
	if selector.Kind == repository.SelectorGlobalSeal && len(selector.Token) == 64 {
		id, err := domain.ParseObjectID(selector.Token)
		if err != nil {
			return domain.ObjectID{}, err
		}
		if _, err := repo.LoadSeal(ctx, id); err != nil {
			return domain.ObjectID{}, err
		}
		return id, nil
	}
	selected, err := repo.ResolveSelector(ctx, text)
	if err != nil {
		return domain.ObjectID{}, err
	}
	return selected.ID, nil
}

func traceCompareBindingDigest(repo *repository.Repository) (string, error) {
	bindings, err := repo.TraceSourceList()
	if err != nil {
		return "", err
	}
	records := make([]traceSourceBindingJSON, 0, len(bindings))
	for _, binding := range bindings {
		records = append(records, *traceSourceBindingJSONValue(binding))
	}
	data, err := json.Marshal(records)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func assembleTraceCompare(prepared traceComparePreparedSelection, results []repository.TraceCompareOwnResult, sources []repository.TraceOwnSourceObservation, bindingDigest string, maxVisits int) (traceCompareV2Document, error) {
	doc := traceCompareV2Document{Schema: "sealgraph/trace-compare/v2", Selection: prepared.selection, Observation: traceCompareObservationJSON{REFHeads: []traceCompareREFHeadJSON{}, BindingDigest: bindingDigest, Sources: traceCompareSourceRecords(sources)}, Limits: traceCompareLimitsJSON{MaxGraphVisits: maxVisits}}
	if prepared.graph != nil {
		doc.Observation.REFHeads = traceCompareREFHeads(prepared.graph.REFHeads)
		doc.Limits.UsedGraphVisits = prepared.graph.UsedVisits
	} else {
		doc.Observation.REFHeads = traceCompareREFHeads(prepared.heads.REFHeads)
		reason := "NO_HEAD"
		doc.GraphReason = &reason
	}
	for _, result := range results {
		if result.Baseline.Kind == repository.TraceOwnCandidate {
			local := buildTraceCompareLocal(result)
			doc.CandidateOwn = &traceCompareCandidateJSON{Baseline: local.Baseline, Local: local}
			doc.Observation.CandidateDigest = local.Baseline.CandidateDigest
			break
		}
	}
	if prepared.graph != nil {
		graph, err := buildTraceCompareGraph(*prepared.graph, results)
		if err != nil {
			return traceCompareV2Document{}, err
		}
		doc.Graph = graph
	}
	return doc, nil
}
