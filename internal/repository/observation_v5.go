package repository

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

type observedGraph struct {
	nodes      map[string]domainv5.ResolvedSeal
	heads      map[string]domain.ObjectID
	refs       map[string][]string
	causes     map[string][]domain.ObjectID
	revisions  map[string][]domain.ObjectID
	children   map[string][]domain.ObjectID
	assertions map[string][]domainv5.AssertionSource
	active     map[string]bool
	stalePaths map[string][][]domain.ObjectID
}

func (r *Repository) buildObservation(ctx context.Context, operation string) (headObservation, *observedGraph, error) {
	observation, err := r.observeHeads(ctx, operation)
	if err != nil {
		return headObservation{}, nil, err
	}
	graph, err := r.buildObservedGraph(ctx, observation.heads, nil)
	if err != nil {
		return headObservation{}, nil, fmt.Errorf("derive format-5 observation for %s: %w", operation, err)
	}
	return observation, graph, nil
}

func (r *Repository) buildObservedGraph(ctx context.Context, heads map[string]domain.ObjectID, overrides map[string]domainv5.ResolvedSeal) (*observedGraph, error) {
	graph := &observedGraph{
		nodes: make(map[string]domainv5.ResolvedSeal), heads: cloneHeads(heads), refs: make(map[string][]string),
		causes: make(map[string][]domain.ObjectID), revisions: make(map[string][]domain.ObjectID),
		children: make(map[string][]domain.ObjectID), assertions: make(map[string][]domainv5.AssertionSource), active: make(map[string]bool),
		stalePaths: make(map[string][][]domain.ObjectID),
	}
	queue := make([]domain.ObjectID, 0, len(heads))
	for ref, head := range heads {
		queue = append(queue, head)
		graph.refs[head.String()] = append(graph.refs[head.String()], ref)
	}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if _, ok := graph.nodes[id.String()]; ok {
			continue
		}
		resolved, ok := overrides[id.String()]
		if !ok {
			var err error
			resolved, err = r.LoadSeal(ctx, id)
			if err != nil {
				return nil, fmt.Errorf("load observed Seal %s: %w", id, err)
			}
		}
		graph.nodes[id.String()] = resolved
		for _, link := range resolved.Provenance.CauseLinks {
			graph.causes[id.String()] = appendUniqueID(graph.causes[id.String()], link.TargetSeal)
			queue = append(queue, link.TargetSeal)
			source := domainv5.AssertionSource{ObserverSeal: id, ObserverProvenance: resolved.Seal.Provenance, CauseLink: link}
			graph.assertions[link.TargetSeal.String()] = append(graph.assertions[link.TargetSeal.String()], source)
			for _, previous := range link.PreviousRevisionSealOfTargetSeal {
				graph.revisions[link.TargetSeal.String()] = appendUniqueID(graph.revisions[link.TargetSeal.String()], previous)
				graph.children[previous.String()] = appendUniqueID(graph.children[previous.String()], link.TargetSeal)
				queue = append(queue, previous)
			}
		}
	}
	graph.normalize()
	if err := graph.validateAcyclic(); err != nil {
		return nil, err
	}
	graph.deriveActive()
	return graph, nil
}

func cloneHeads(heads map[string]domain.ObjectID) map[string]domain.ObjectID {
	result := make(map[string]domain.ObjectID, len(heads))
	for ref, head := range heads {
		result[ref] = head
	}
	return result
}

func appendUniqueID(ids []domain.ObjectID, id domain.ObjectID) []domain.ObjectID {
	for _, existing := range ids {
		if existing.Equal(id) {
			return ids
		}
	}
	return append(ids, id)
}

func (graph *observedGraph) normalize() {
	for _, refs := range graph.refs {
		sort.Strings(refs)
	}
	for _, collection := range []map[string][]domain.ObjectID{graph.causes, graph.revisions, graph.children} {
		for key := range collection {
			sort.Slice(collection[key], func(i, j int) bool { return collection[key][i].String() < collection[key][j].String() })
		}
	}
	for target := range graph.assertions {
		sort.Slice(graph.assertions[target], func(i, j int) bool { return assertionLess(graph.assertions[target][i], graph.assertions[target][j]) })
	}
}

func assertionLess(left, right domainv5.AssertionSource) bool {
	if left.ObserverSeal.String() != right.ObserverSeal.String() {
		return left.ObserverSeal.String() < right.ObserverSeal.String()
	}
	if left.ObserverProvenance.String() != right.ObserverProvenance.String() {
		return left.ObserverProvenance.String() < right.ObserverProvenance.String()
	}
	if left.CauseLink.TargetSeal.String() != right.CauseLink.TargetSeal.String() {
		return left.CauseLink.TargetSeal.String() < right.CauseLink.TargetSeal.String()
	}
	for i := 0; i < len(left.CauseLink.PreviousRevisionSealOfTargetSeal) && i < len(right.CauseLink.PreviousRevisionSealOfTargetSeal); i++ {
		if left.CauseLink.PreviousRevisionSealOfTargetSeal[i].String() != right.CauseLink.PreviousRevisionSealOfTargetSeal[i].String() {
			return left.CauseLink.PreviousRevisionSealOfTargetSeal[i].String() < right.CauseLink.PreviousRevisionSealOfTargetSeal[i].String()
		}
	}
	if len(left.CauseLink.PreviousRevisionSealOfTargetSeal) != len(right.CauseLink.PreviousRevisionSealOfTargetSeal) {
		return len(left.CauseLink.PreviousRevisionSealOfTargetSeal) < len(right.CauseLink.PreviousRevisionSealOfTargetSeal)
	}
	for i := 0; i < len(left.CauseLink.Messages) && i < len(right.CauseLink.Messages); i++ {
		if left.CauseLink.Messages[i] != right.CauseLink.Messages[i] {
			return left.CauseLink.Messages[i] < right.CauseLink.Messages[i]
		}
	}
	return len(left.CauseLink.Messages) < len(right.CauseLink.Messages)
}

func (graph *observedGraph) validateAcyclic() error {
	state := make(map[string]uint8)
	var visit func(domain.ObjectID) error
	visit = func(id domain.ObjectID) error {
		switch state[id.String()] {
		case 1:
			return fmt.Errorf("combined Cause/revision cycle reaches Seal %s", id)
		case 2:
			return nil
		}
		if _, ok := graph.nodes[id.String()]; !ok {
			return fmt.Errorf("graph references missing Seal %s", id)
		}
		state[id.String()] = 1
		for _, next := range graph.causes[id.String()] {
			if next.Equal(id) {
				return fmt.Errorf("Cause self-edge at Seal %s", id)
			}
			if err := visit(next); err != nil {
				return err
			}
		}
		for _, next := range graph.revisions[id.String()] {
			if next.Equal(id) {
				return fmt.Errorf("revision self-edge at Seal %s", id)
			}
			if err := visit(next); err != nil {
				return err
			}
		}
		state[id.String()] = 2
		return nil
	}
	ids := make([]string, 0, len(graph.nodes))
	for id := range graph.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, text := range ids {
		if err := visit(domain.ObjectID{Hex: text}); err != nil {
			return err
		}
	}
	return nil
}

func (graph *observedGraph) deriveActive() {
	stack := make([]domain.ObjectID, 0, len(graph.heads))
	for _, head := range graph.heads {
		stack = append(stack, head)
	}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if graph.active[id.String()] {
			continue
		}
		graph.active[id.String()] = true
		stack = append(stack, graph.revisions[id.String()]...)
	}
}

func (graph *observedGraph) isActiveLeaf(id domain.ObjectID) bool {
	if !graph.active[id.String()] {
		return false
	}
	for _, child := range graph.children[id.String()] {
		if graph.active[child.String()] {
			return false
		}
	}
	return true
}

func (graph *observedGraph) revisionReachable(head, selected domain.ObjectID) bool {
	stack := []domain.ObjectID{head}
	seen := make(map[string]bool)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if id.Equal(selected) {
			return true
		}
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		stack = append(stack, graph.revisions[id.String()]...)
	}
	return false
}

func (graph *observedGraph) refsFor(id domain.ObjectID) []string {
	return append([]string{}, graph.refs[id.String()]...)
}

func (graph *observedGraph) revisionObservation(id domain.ObjectID) domainv5.RevisionObservation {
	assertions := append([]domainv5.AssertionSource(nil), graph.assertions[id.String()]...)
	states := []string{}
	if len(assertions) == 0 {
		states = append(states, "PREVIOUS_UNOBSERVED")
	} else {
		none, asserted := false, false
		for _, assertion := range assertions {
			if len(assertion.CauseLink.PreviousRevisionSealOfTargetSeal) == 0 {
				none = true
			} else {
				asserted = true
			}
		}
		if none {
			states = append(states, "PREVIOUS_NONE_ASSERTED")
		}
		if asserted {
			states = append(states, "PREVIOUS_ASSERTED")
		}
	}
	return domainv5.RevisionObservation{TargetSeal: id, PreviousStates: states, Assertions: assertions, StructuralPrevious: append([]domain.ObjectID{}, graph.revisions[id.String()]...)}
}

func (r *Repository) validateProspectiveCandidate(ctx context.Context, candidate domainv5.Candidate) (headObservation, error) {
	observation, err := r.observeHeads(ctx, "seal admission")
	if err != nil {
		return headObservation{}, err
	}
	prospective, err := prospectiveSeal(candidate)
	if err != nil {
		return headObservation{}, err
	}
	heads := cloneHeads(observation.heads)
	heads[candidate.REF] = prospective.ID
	graph, err := r.buildObservedGraph(ctx, heads, map[string]domainv5.ResolvedSeal{prospective.ID.String(): prospective})
	if err != nil {
		return headObservation{}, fmt.Errorf("prospective Candidate graph is invalid: %w", err)
	}
	if candidate.Draft || candidate.Root {
		return observation, nil
	}
	stack := make([]domain.ObjectID, 0, len(candidate.CauseLinks))
	for _, link := range candidate.CauseLinks {
		stack = append(stack, link.TargetSeal)
	}
	seen := make(map[string]bool)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		resolved, ok := graph.nodes[id.String()]
		if !ok {
			return headObservation{}, fmt.Errorf("Cause %s is absent from prospective graph", id)
		}
		if resolved.Provenance.Draft {
			return headObservation{}, fmt.Errorf("normal Seal requires non-draft Cause %s; keep the Candidate draft or select another Cause", id)
		}
		if !graph.isActiveLeaf(id) {
			return headObservation{}, fmt.Errorf("normal Seal requires Cause %s to be an active revision leaf; keep the Candidate draft or revise the assertion explicitly", id)
		}
		stack = append(stack, graph.causes[id.String()]...)
	}
	return observation, nil
}
