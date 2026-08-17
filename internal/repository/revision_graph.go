package repository

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/graph"
	"github.com/mako10k/sealgraph/internal/revision"
)

type RefStatus struct {
	REF             string
	Head            *domain.ObjectID
	Unsealed        bool
	Draft           bool
	StaleSelf       bool
	StaleDirect     []domain.ObjectID
	StaleTransitive [][]domain.ObjectID
}

func (status RefStatus) Labels() []string {
	var labels []string
	if status.Unsealed {
		labels = append(labels, "UNSEALED")
	}
	if status.Draft {
		labels = append(labels, "DRAFT")
	}
	if status.StaleSelf {
		labels = append(labels, "STALE_SELF")
	}
	if len(status.StaleDirect) != 0 {
		labels = append(labels, "STALE_DIRECT")
	}
	if len(status.StaleTransitive) != 0 {
		labels = append(labels, "STALE_TRANSITIVE")
	}
	if len(labels) == 0 {
		return []string{"CLEAN"}
	}
	return labels
}

func (r *Repository) Status(ctx context.Context, onlyREF string) ([]RefStatus, error) {
	observation, revisions, analysis, err := r.analyzeCurrent(ctx, "status", false, false)
	if err != nil {
		return nil, err
	}
	candidates, err := r.candidates.List()
	if err != nil {
		return nil, err
	}
	names, err := statusNames(observation.names, candidates, onlyREF)
	if err != nil {
		return nil, err
	}
	result := make([]RefStatus, 0, len(names))
	for _, ref := range names {
		status, err := r.statusForREF(ctx, ref, observation, revisions, analysis)
		if err != nil {
			return nil, err
		}
		result = append(result, status)
	}
	if err := r.revalidateHeads(ctx, observation, "status"); err != nil {
		return nil, err
	}
	return result, nil
}

func statusNames(refs, candidates []string, onlyREF string) ([]string, error) {
	all := make(map[string]bool, len(refs)+len(candidates))
	for _, ref := range refs {
		all[ref] = true
	}
	for _, ref := range candidates {
		all[ref] = true
	}
	if onlyREF != "" {
		if err := domain.ValidateREF(onlyREF); err != nil {
			return nil, err
		}
		if !all[onlyREF] {
			return nil, fmt.Errorf("REF %s has no head or candidate", onlyREF)
		}
		return []string{onlyREF}, nil
	}
	result := make([]string, 0, len(all))
	for ref := range all {
		result = append(result, ref)
	}
	sort.Strings(result)
	return result, nil
}

func (r *Repository) statusForREF(ctx context.Context, ref string, observation headObservation, revisions *revision.Index, analysis *graph.Analysis) (RefStatus, error) {
	status := RefStatus{REF: ref}
	if candidate, err := r.candidates.Load(ref); err == nil {
		status.Unsealed, status.Draft = true, candidate.Draft
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return RefStatus{}, err
	}
	head, found := observation.heads[ref]
	if !found {
		return status, nil
	}
	headCopy := head
	status.Head = &headCopy
	node, found := revisions.Node(head)
	if !found {
		return RefStatus{}, fmt.Errorf("current REF %s head %s is absent from the active revision DAG", ref, head)
	}
	status.Draft = status.Draft || node.Payload.Draft
	facts := analysis.Facts(head)
	status.StaleSelf, status.StaleDirect, status.StaleTransitive = facts.Self, facts.Direct, facts.Transitive
	return status, nil
}

func (r *Repository) Stale(ctx context.Context, frontier, scan bool) ([]RefStatus, string, error) {
	observation, revisions, analysis, warning, err := r.analyzeCurrentWithWarning(ctx, "stale", true, scan)
	if err != nil {
		return nil, "", err
	}
	var result []RefStatus
	staleHeads := make(map[string]domain.ObjectID)
	for _, ref := range observation.names {
		status, err := r.statusForCurrentREF(ref, observation, revisions, analysis)
		if err != nil {
			return nil, "", err
		}
		if status.StaleSelf || len(status.StaleDirect) != 0 || len(status.StaleTransitive) != 0 {
			result = append(result, status)
			staleHeads[ref] = *status.Head
		}
	}
	if frontier {
		result = filterFrontier(result, analysis.Frontier(staleHeads))
	}
	if err := r.revalidateHeads(ctx, observation, "stale"); err != nil {
		return nil, "", err
	}
	return result, warning, nil
}

func (r *Repository) statusForCurrentREF(ref string, observation headObservation, revisions *revision.Index, analysis *graph.Analysis) (RefStatus, error) {
	head := observation.heads[ref]
	node, found := revisions.Node(head)
	if !found {
		return RefStatus{}, fmt.Errorf("current REF %s head %s is absent from active revisions", ref, head)
	}
	facts := analysis.Facts(head)
	headCopy := head
	return RefStatus{
		REF: ref, Head: &headCopy, Draft: node.Payload.Draft, StaleSelf: facts.Self,
		StaleDirect: facts.Direct, StaleTransitive: facts.Transitive,
	}, nil
}

func filterFrontier(statuses []RefStatus, selected map[string]bool) []RefStatus {
	result := make([]RefStatus, 0, len(statuses))
	for _, status := range statuses {
		if selected[status.REF] {
			result = append(result, status)
		}
	}
	return result
}

type GraphLink struct {
	Target domain.ObjectID
	State  revision.State
}

type GraphNode struct {
	ID     domain.ObjectID
	REFs   []string
	Parent *domain.ObjectID
	State  revision.State
	Links  []GraphLink
}

func (r *Repository) Graph(ctx context.Context) ([]GraphNode, error) {
	observation, revisions, _, err := r.analyzeCurrent(ctx, "graph", false, false)
	if err != nil {
		return nil, err
	}
	nodes := revisions.Nodes()
	result := make([]GraphNode, 0, len(nodes))
	for _, node := range nodes {
		graphNode := GraphNode{ID: node.ID, REFs: node.REFs, Parent: node.Payload.ParentRevision, State: revisions.State(node.ID)}
		for _, link := range node.Payload.Links {
			graphNode.Links = append(graphNode.Links, GraphLink{Target: link.TargetSeal, State: revisions.State(link.TargetSeal)})
		}
		result = append(result, graphNode)
	}
	if err := r.revalidateHeads(ctx, observation, "graph"); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repository) Impact(ctx context.Context, selector string, allPaths bool, maxPaths int) (domain.ObjectID, []graph.Impact, error) {
	resolved, err := r.ResolveSelector(ctx, selector)
	if err != nil {
		return domain.ObjectID{}, nil, err
	}
	observation, _, analysis, err := r.analyzeCurrent(ctx, "impact", false, false)
	if err != nil {
		return domain.ObjectID{}, nil, err
	}
	result, err := analysis.Impact(ctx, resolved.ID, observation.revisionHeads(), graph.LoadSealFunc(r.LoadSeal), allPaths, maxPaths)
	if err != nil {
		return domain.ObjectID{}, nil, err
	}
	if err := r.revalidateHeads(ctx, observation, "impact"); err != nil {
		return domain.ObjectID{}, nil, err
	}
	return resolved.ID, result, nil
}

func (r *Repository) analyzeCurrent(ctx context.Context, operation string, cache, scan bool) (headObservation, *revision.Index, *graph.Analysis, error) {
	observation, revisions, analysis, _, err := r.analyzeCurrentWithWarning(ctx, operation, cache, scan)
	return observation, revisions, analysis, err
}

func (r *Repository) analyzeCurrentWithWarning(ctx context.Context, operation string, cache, scan bool) (headObservation, *revision.Index, *graph.Analysis, string, error) {
	observation, err := r.observeHeads(ctx, operation)
	if err != nil {
		return headObservation{}, nil, nil, "", err
	}
	var revisions *revision.Index
	var warning string
	if cache {
		revisions, warning, err = r.revisionIndex(ctx, observation, scan)
	} else {
		revisions, err = revision.Build(ctx, observation.revisionHeads(), revision.LoadSealFunc(r.LoadSeal))
	}
	if err != nil {
		return headObservation{}, nil, nil, "", fmt.Errorf("derive active revisions for %s: %w", operation, err)
	}
	roots := make([]domain.ObjectID, 0, len(revisions.Nodes()))
	for _, node := range revisions.Nodes() {
		roots = append(roots, node.ID)
	}
	analysis, err := graph.Build(ctx, roots, revisions, graph.LoadSealFunc(r.LoadSeal))
	if err != nil {
		return headObservation{}, nil, nil, "", fmt.Errorf("derive Cause graph for %s: %w", operation, err)
	}
	return observation, revisions, analysis, warning, nil
}

func (r *Repository) requireActiveLeafClosure(ctx context.Context, links []domain.Link) (headObservation, error) {
	observation, revisions, analysis, err := r.analyzeCurrent(ctx, "seal admission", false, false)
	if err != nil {
		return headObservation{}, err
	}
	for _, link := range links {
		if err := requireLinkClosureActive(link.TargetSeal, revisions, analysis); err != nil {
			return headObservation{}, err
		}
	}
	return observation, nil
}

func requireLinkClosureActive(root domain.ObjectID, revisions *revision.Index, analysis *graph.Analysis) error {
	stack := []domain.ObjectID{root}
	seen := make(map[string]bool)
	for len(stack) != 0 {
		last := len(stack) - 1
		id := stack[last]
		stack = stack[:last]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		if !revisions.IsCurrentLeaf(id) {
			return fmt.Errorf("normal seal requires Cause %s to be an active current revision leaf, but it is %s; keep the candidate draft or relink explicitly", id, revisions.State(id))
		}
		payload, found := analysisPayload(analysis, id)
		if !found {
			return fmt.Errorf("Cause %s is absent from the validated graph", id)
		}
		if payload.Draft {
			return fmt.Errorf("normal seal requires non-draft Cause %s; keep the candidate draft or select a non-draft Cause explicitly", id)
		}
		for _, link := range payload.Links {
			stack = append(stack, link.TargetSeal)
		}
	}
	return nil
}

func analysisPayload(analysis *graph.Analysis, id domain.ObjectID) (domain.SealPayload, bool) {
	return analysis.Payload(id)
}
