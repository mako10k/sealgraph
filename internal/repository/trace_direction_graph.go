package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

// DirectionGraphREFHead records one REF head captured for an observation.
type DirectionGraphREFHead struct {
	REF    string
	SealID domain.ObjectID
}

// DirectionGraphMember records independent direct and indirect Cause routes.
// Paths always follow the Cause arrow from dependent to target.
type DirectionGraphMember struct {
	SealID       domain.ObjectID
	Direct       bool
	Indirect     bool
	DirectPath   []domain.ObjectID
	IndirectPath []domain.ObjectID
}

// DirectionGraphResult is a bounded, structural observation.  Incomplete
// observations do not establish that a member is absent from either direction.
type DirectionGraphResult struct {
	Center            domain.ObjectID
	REFHeads          []DirectionGraphREFHead
	ExtraSeals        []domain.ObjectID
	ObservedSealIDs   []domain.ObjectID
	ClosureComplete   bool
	DirectionComplete bool
	StopReason        string
	UsedVisits        int
	Upstream          []DirectionGraphMember
	Downstream        []DirectionGraphMember
	observation       headObservation
}

// DirectionGraphHeads is the stable REF-head scope used when a Candidate has
// no published HEAD and therefore has no graph center.
type DirectionGraphHeads struct {
	REFHeads    []DirectionGraphREFHead
	observation headObservation
}

const directionGraphVisitLimit = "GRAPH_BUDGET"

// TraceDirectionHeads captures the all-REF scope without performing graph
// traversal.  It deliberately consumes no graph visits.
func (r *Repository) TraceDirectionHeads(ctx context.Context) (DirectionGraphHeads, error) {
	observation, err := r.observeHeads(ctx, "trace direction heads")
	if err != nil {
		return DirectionGraphHeads{}, err
	}
	return DirectionGraphHeads{REFHeads: directionREFHeads(observation.heads), observation: observation}, nil
}

// RevalidateTraceDirectionHeads verifies a candidate-only compare scope just
// before the caller emits its document.
func (r *Repository) RevalidateTraceDirectionHeads(ctx context.Context, result DirectionGraphHeads) error {
	return r.revalidateHeads(ctx, result.observation, "trace direction heads")
}

// TraceDirectionGraph observes the REF-head closure and a selected exact Seal.
// Cause targets and revision assertions define the closure; only Cause targets
// define upstream and downstream paths.
func (r *Repository) TraceDirectionGraph(ctx context.Context, center domain.ObjectID, maxVisits int) (DirectionGraphResult, error) {
	if maxVisits <= 0 {
		return DirectionGraphResult{}, fmt.Errorf("max graph visits must be positive")
	}
	observation, err := r.observeHeads(ctx, "trace direction graph")
	if err != nil {
		return DirectionGraphResult{}, err
	}
	result := DirectionGraphResult{Center: center, ClosureComplete: true, DirectionComplete: true}
	result.REFHeads = directionREFHeads(observation.heads)
	result.observation = observation
	graph, complete, err := r.directionClosure(ctx, observation.heads, center, maxVisits, &result)
	if err != nil {
		return DirectionGraphResult{}, err
	}
	result.ClosureComplete = complete
	if !complete {
		result.DirectionComplete = false
	}
	if complete {
		if err := graph.validateAcyclic(); err != nil {
			return DirectionGraphResult{}, err
		}
		result.ExtraSeals = graph.directionExtras(observation.heads, center)
	}
	if graph.nodes[center.String()].ID.Hex == "" {
		return DirectionGraphResult{}, fmt.Errorf("selected Seal %s was not observed before the graph visit limit; rerun with a larger limit", center)
	}
	upstream, complete := graph.directionMembers(center, false, maxVisits, &result)
	result.Upstream = upstream
	result.DirectionComplete = result.DirectionComplete && complete
	downstream, complete := graph.directionMembers(center, true, maxVisits, &result)
	result.Downstream = downstream
	result.DirectionComplete = result.DirectionComplete && complete
	result.ObservedSealIDs = sortedDirectionIDs(graph.nodes)
	if !result.ClosureComplete || !result.DirectionComplete {
		result.StopReason = directionGraphVisitLimit
	}
	if err := r.revalidateHeads(ctx, observation, "trace direction graph"); err != nil {
		return DirectionGraphResult{}, err
	}
	return result, nil
}

// RevalidateTraceDirectionGraph verifies that the REF snapshot captured by
// TraceDirectionGraph remains current before a caller publishes its document.
func (r *Repository) RevalidateTraceDirectionGraph(ctx context.Context, result DirectionGraphResult) error {
	return r.revalidateHeads(ctx, result.observation, "trace direction graph")
}

func directionREFHeads(heads map[string]domain.ObjectID) []DirectionGraphREFHead {
	refs := make([]string, 0, len(heads))
	for ref := range heads {
		refs = append(refs, ref)
	}
	sort.Strings(refs)
	result := make([]DirectionGraphREFHead, 0, len(refs))
	for _, ref := range refs {
		result = append(result, DirectionGraphREFHead{REF: ref, SealID: heads[ref]})
	}
	return result
}

func (r *Repository) directionClosure(ctx context.Context, heads map[string]domain.ObjectID, center domain.ObjectID, max int, result *DirectionGraphResult) (*observedGraph, bool, error) {
	graph := &observedGraph{nodes: map[string]domainv5.ResolvedSeal{}, causes: map[string][]domain.ObjectID{}, revisions: map[string][]domain.ObjectID{}, children: map[string][]domain.ObjectID{}}
	queue := append([]domain.ObjectID{center}, directionHeadIDs(heads)...)
	for len(queue) > 0 {
		if result.UsedVisits >= max {
			graph.normalize()
			return graph, false, nil
		}
		id := queue[0]
		queue = queue[1:]
		result.UsedVisits++
		if _, exists := graph.nodes[id.String()]; exists {
			continue
		}
		resolved, err := r.LoadSeal(ctx, id)
		if err != nil {
			return nil, false, fmt.Errorf("load observed Seal %s: %w", id, err)
		}
		graph.nodes[id.String()] = resolved
		for _, link := range resolved.Provenance.CauseLinks {
			graph.causes[id.String()] = appendUniqueID(graph.causes[id.String()], link.TargetSeal)
			queue = append(queue, link.TargetSeal)
			for _, previous := range link.PreviousRevisionSealOfTargetSeal {
				graph.revisions[link.TargetSeal.String()] = appendUniqueID(graph.revisions[link.TargetSeal.String()], previous)
				graph.children[previous.String()] = appendUniqueID(graph.children[previous.String()], link.TargetSeal)
				queue = append(queue, previous)
			}
		}
	}
	graph.normalize()
	return graph, true, nil
}

func directionHeadIDs(heads map[string]domain.ObjectID) []domain.ObjectID {
	ids := make([]domain.ObjectID, 0, len(heads))
	for _, id := range heads {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func sortedDirectionIDs(nodes map[string]domainv5.ResolvedSeal) []domain.ObjectID {
	ids := make([]domain.ObjectID, 0, len(nodes))
	for id := range nodes {
		ids = append(ids, domain.ObjectID{Hex: id})
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	return ids
}

func (graph *observedGraph) directionMembers(center domain.ObjectID, reverse bool, max int, result *DirectionGraphResult) ([]DirectionGraphMember, bool) {
	edges := graph.causes
	if reverse {
		edges = directionReverseCauses(graph.causes)
	}
	queue := [][]domain.ObjectID{{center}}
	members := map[string]DirectionGraphMember{}
	for len(queue) > 0 {
		if result.UsedVisits >= max {
			return sortedDirectionMembers(members), false
		}
		path := queue[0]
		queue = queue[1:]
		result.UsedVisits++
		current := path[len(path)-1]
		if reverse {
			current = path[0]
		}
		for _, next := range edges[current.String()] {
			if _, observed := graph.nodes[next.String()]; !observed {
				continue
			}
			candidate := appendDirectionPath(path, next, reverse)
			if directionPathContains(path, next) {
				continue
			}
			member := members[next.String()]
			member.SealID = next
			if len(candidate) == 2 {
				member.Direct, member.DirectPath = true, betterDirectionPath(member.DirectPath, candidate)
			} else {
				member.Indirect, member.IndirectPath = true, betterDirectionPath(member.IndirectPath, candidate)
			}
			members[next.String()] = member
			queue = append(queue, candidate)
		}
	}
	return sortedDirectionMembers(members), true
}

func (graph *observedGraph) directionExtras(heads map[string]domain.ObjectID, center domain.ObjectID) []domain.ObjectID {
	defaultSet := graph.directionClosureSet(directionHeadIDs(heads))
	if defaultSet[center.String()] {
		return nil
	}
	return []domain.ObjectID{center}
}

func (graph *observedGraph) directionClosureSet(seeds []domain.ObjectID) map[string]bool {
	seen := map[string]bool{}
	queue := append([]domain.ObjectID{}, seeds...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		queue = append(queue, graph.causes[id.String()]...)
		queue = append(queue, graph.revisions[id.String()]...)
	}
	return seen
}

func directionReverseCauses(causes map[string][]domain.ObjectID) map[string][]domain.ObjectID {
	reverse := map[string][]domain.ObjectID{}
	for dependent, targets := range causes {
		for _, target := range targets {
			reverse[target.String()] = appendUniqueID(reverse[target.String()], domain.ObjectID{Hex: dependent})
		}
	}
	for id := range reverse {
		sort.Slice(reverse[id], func(i, j int) bool { return reverse[id][i].String() < reverse[id][j].String() })
	}
	return reverse
}

func appendDirectionPath(path []domain.ObjectID, next domain.ObjectID, reverse bool) []domain.ObjectID {
	if !reverse {
		return append(append([]domain.ObjectID{}, path...), next)
	}
	return append([]domain.ObjectID{next}, path...)
}

func directionPathContains(path []domain.ObjectID, next domain.ObjectID) bool {
	for _, id := range path {
		if id.Equal(next) {
			return true
		}
	}
	return false
}

func betterDirectionPath(current, candidate []domain.ObjectID) []domain.ObjectID {
	if len(current) == 0 || len(candidate) < len(current) || len(candidate) == len(current) && directionPathLess(candidate, current) {
		return append([]domain.ObjectID{}, candidate...)
	}
	return current
}

func directionPathLess(left, right []domain.ObjectID) bool {
	for i := range left {
		if left[i].String() != right[i].String() {
			return left[i].String() < right[i].String()
		}
	}
	return false
}

func sortedDirectionMembers(values map[string]DirectionGraphMember) []DirectionGraphMember {
	ids := make([]string, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]DirectionGraphMember, 0, len(ids))
	for _, id := range ids {
		result = append(result, values[id])
	}
	return result
}
