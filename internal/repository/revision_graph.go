package repository

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
	"github.com/mako10k/sealgraph/internal/store/native"
	"github.com/mako10k/sealgraph/internal/workfile"
)

type RefStatus struct {
	REF             string
	Head            *domain.ObjectID
	Unsealed        bool
	Draft           bool
	StaleSelf       bool
	StaleDirect     []domain.ObjectID
	StaleTransitive [][]domain.ObjectID
	Source          *SourceStatus
}

type SourceStatus struct{ Path, Baseline, Relation string }

func (status RefStatus) Labels() []string {
	labels := []string{}
	if status.Unsealed {
		labels = append(labels, "UNSEALED")
	}
	if status.Draft {
		labels = append(labels, "DRAFT")
	}
	if status.StaleSelf {
		labels = append(labels, "STALE_SELF")
	}
	if len(status.StaleDirect) > 0 {
		labels = append(labels, "STALE_DIRECT")
	}
	if len(status.StaleTransitive) > 0 {
		labels = append(labels, "STALE_TRANSITIVE")
	}
	if len(labels) == 0 {
		return []string{"CLEAN"}
	}
	return labels
}

func (r *Repository) Status(ctx context.Context, onlyREF string) ([]RefStatus, error) {
	observation, graph, err := r.buildObservation(ctx, "status")
	if err != nil {
		return nil, err
	}
	candidates, err := r.candidates.List()
	if err != nil {
		return nil, err
	}
	bindings, err := r.sources.list()
	if err != nil {
		return nil, err
	}
	sourceREFs := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		sourceREFs = append(sourceREFs, binding.REF)
	}
	names, err := statusNames(observation.names, candidates, sourceREFs, onlyREF)
	if err != nil {
		return nil, err
	}
	result := make([]RefStatus, 0, len(names))
	for _, ref := range names {
		status, err := r.statusForREF(ref, observation, graph)
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

func statusNames(refs, candidates, sources []string, onlyREF string) ([]string, error) {
	all := make(map[string]bool, len(refs)+len(candidates)+len(sources))
	for _, values := range [][]string{refs, candidates, sources} {
		for _, ref := range values {
			all[ref] = true
		}
	}
	if onlyREF != "" {
		if err := domain.ValidateREF(onlyREF); err != nil {
			return nil, err
		}
		if !all[onlyREF] {
			return nil, fmt.Errorf("REF %s has no head, Candidate, or local source binding", onlyREF)
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

func (r *Repository) statusForREF(ref string, observation headObservation, graph *observedGraph) (RefStatus, error) {
	status := RefStatus{REF: ref}
	var baseline *domain.ObjectID
	if candidate, err := r.candidates.Load(ref); err == nil {
		status.Unsealed, status.Draft = true, candidate.Draft
		content := candidate.Content
		baseline = &content
	} else if !errors.Is(err, ErrCandidateNotFound) {
		return RefStatus{}, err
	}
	head, found := observation.heads[ref]
	if found {
		value, ok := graph.nodes[head.String()]
		if !ok {
			return RefStatus{}, fmt.Errorf("current REF %s head %s is absent from observation", ref, head)
		}
		headCopy := head
		status.Head = &headCopy
		status.Draft = status.Draft || value.Provenance.Draft
		if baseline == nil {
			content := value.Material.Content
			baseline = &content
		}
		status.StaleSelf, status.StaleDirect, status.StaleTransitive = graph.staleFacts(head)
	}
	return r.addSourceStatus(status, baseline)
}

func (r *Repository) addSourceStatus(status RefStatus, baseline *domain.ObjectID) (RefStatus, error) {
	binding, _, err := r.sources.load(status.REF)
	if errors.Is(err, ErrSourceNotFound) {
		return status, nil
	}
	if err != nil {
		return RefStatus{}, err
	}
	source := &SourceStatus{Path: binding.Path, Baseline: "NONE", Relation: "WORKFILE_DIFFERS_FROM_NONE"}
	if baseline != nil {
		if status.Unsealed {
			source.Baseline = "CANDIDATE"
		} else {
			source.Baseline = "HEAD"
		}
	}
	data, err := workfile.ReadStable(r.workDir, binding.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			source.Relation = "SOURCE_MISSING"
		} else {
			source.Relation = "SOURCE_UNREADABLE"
		}
		status.Source = source
		return status, nil
	}
	if baseline != nil && native.ObjectID(data).Equal(*baseline) {
		source.Relation = "WORKFILE_MATCHES_" + source.Baseline
	} else if baseline != nil {
		source.Relation = "WORKFILE_DIFFERS_FROM_" + source.Baseline
	}
	status.Source = source
	return status, nil
}

func (graph *observedGraph) staleFacts(head domain.ObjectID) (bool, []domain.ObjectID, [][]domain.ObjectID) {
	self := graph.active[head.String()] && !graph.isActiveLeaf(head)
	direct := []domain.ObjectID{}
	for _, target := range graph.causes[head.String()] {
		if !graph.isActiveLeaf(target) {
			direct = append(direct, target)
		}
	}
	if len(direct) > 0 {
		return self, direct, [][]domain.ObjectID{}
	}
	paths := [][]domain.ObjectID{}
	for _, target := range graph.causes[head.String()] {
		graph.firstStalePaths(target, []domain.ObjectID{target}, &paths)
	}
	sortPaths(paths)
	return self, direct, paths
}

func (graph *observedGraph) firstStalePaths(current domain.ObjectID, path []domain.ObjectID, result *[][]domain.ObjectID) {
	if !graph.isActiveLeaf(current) {
		*result = append(*result, append([]domain.ObjectID(nil), path...))
		return
	}
	for _, next := range graph.causes[current.String()] {
		graph.firstStalePaths(next, append(path, next), result)
	}
}

func sortPaths(paths [][]domain.ObjectID) {
	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) < len(paths[j])
		}
		return idPathKey(paths[i]) < idPathKey(paths[j])
	})
}
func idPathKey(path []domain.ObjectID) string {
	value := ""
	for _, id := range path {
		value += id.String() + "\x00"
	}
	return value
}

func (r *Repository) Stale(ctx context.Context, frontier, scan bool) ([]RefStatus, string, error) {
	observation, graph, err := r.buildObservation(ctx, "stale")
	if err != nil {
		return nil, "", err
	}
	result := []RefStatus{}
	staleHeads := make(map[string]domain.ObjectID)
	for _, ref := range observation.names {
		head := observation.heads[ref]
		value := graph.nodes[head.String()]
		self, direct, transitive := graph.staleFacts(head)
		if self || len(direct) > 0 || len(transitive) > 0 {
			copy := head
			result = append(result, RefStatus{REF: ref, Head: &copy, Draft: value.Provenance.Draft, StaleSelf: self, StaleDirect: direct, StaleTransitive: transitive})
			staleHeads[ref] = head
		}
	}
	if frontier {
		filtered := result[:0]
		for _, status := range result {
			if graph.isFrontier(*status.Head, staleHeads) {
				filtered = append(filtered, status)
			}
		}
		result = filtered
	}
	if err := r.revalidateHeads(ctx, observation, "stale"); err != nil {
		return nil, "", err
	}
	_ = scan // Cache bypass is semantically identical; format-5 cache is not yet persisted.
	return result, "", nil
}

func (graph *observedGraph) isFrontier(head domain.ObjectID, stale map[string]domain.ObjectID) bool {
	targets := make(map[string]bool)
	for _, id := range stale {
		targets[id.String()] = true
	}
	stack := append([]domain.ObjectID(nil), graph.causes[head.String()]...)
	seen := make(map[string]bool)
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		if targets[id.String()] {
			return false
		}
		stack = append(stack, graph.causes[id.String()]...)
	}
	return true
}

type RevisionState string

const (
	ActiveLeaf           RevisionState = "ACTIVE_LEAF"
	ActiveNonLeaf        RevisionState = "ACTIVE_NON_LEAF"
	HistoricalOrDetached RevisionState = "HISTORICAL_OR_DETACHED"
)

func (graph *observedGraph) state(id domain.ObjectID) RevisionState {
	if !graph.active[id.String()] {
		return HistoricalOrDetached
	}
	if graph.isActiveLeaf(id) {
		return ActiveLeaf
	}
	return ActiveNonLeaf
}

type GraphCause struct {
	Target domain.ObjectID
	State  RevisionState
}
type GraphNode struct {
	Resolved domainv5.ResolvedSeal
	REFs     []string
	State    RevisionState
	Revision domainv5.RevisionObservation
	Causes   []GraphCause
}

func (r *Repository) Graph(ctx context.Context) ([]GraphNode, error) {
	observation, graph, err := r.buildObservation(ctx, "graph")
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(graph.nodes))
	for id := range graph.nodes {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]GraphNode, 0, len(ids))
	for _, text := range ids {
		id := domain.ObjectID{Hex: text}
		node := GraphNode{Resolved: graph.nodes[text], REFs: graph.refsFor(id), State: graph.state(id), Revision: graph.revisionObservation(id)}
		for _, target := range graph.causes[text] {
			node.Causes = append(node.Causes, GraphCause{Target: target, State: graph.state(target)})
		}
		result = append(result, node)
	}
	if err := r.revalidateHeads(ctx, observation, "graph"); err != nil {
		return nil, err
	}
	return result, nil
}

type RevisionEdge struct {
	Target, Previous domain.ObjectID
	Sources          []domainv5.AssertionSource
}
type ScopedRevisionObservation struct {
	TargetSeal         domain.ObjectID
	PreviousStates     []string
	Assertions         []domainv5.AssertionSource
	StructuralPrevious []domain.ObjectID
}
type RevisionProof struct {
	SealIDs      []domain.ObjectID
	Edges        []RevisionEdge
	Observations []ScopedRevisionObservation
}
type ImpactPath struct {
	CauseSealIDs    []domain.ObjectID
	MatchedRevision domain.ObjectID
	Proof           RevisionProof
}
type ImpactRecord struct {
	Head      domain.ObjectID
	REFs      []string
	Paths     []ImpactPath
	Truncated bool
}
type ImpactResult struct {
	Source          domain.ObjectID
	AssertionScope  string
	ObserverSealIDs []domain.ObjectID
	AllPaths        bool
	MaxPaths        int
	Impacts         []ImpactRecord
}

func (r *Repository) Impact(ctx context.Context, selector string, assertedBy []string, allPaths bool, maxPaths int) (ImpactResult, error) {
	observation, graph, err := r.buildObservation(ctx, "impact")
	if err != nil {
		return ImpactResult{}, err
	}
	selected, err := r.resolveSelectorObserved(ctx, selector, &observation, graph)
	if err != nil {
		return ImpactResult{}, err
	}
	filter := make(map[string]bool)
	observers := []domain.ObjectID{}
	for _, raw := range assertedBy {
		resolved, err := r.resolveSelectorObserved(ctx, raw, &observation, graph)
		if err != nil {
			return ImpactResult{}, fmt.Errorf("resolve assertion observer %q: %w", raw, err)
		}
		if _, ok := graph.nodes[resolved.ID.String()]; !ok {
			return ImpactResult{}, fmt.Errorf("ASSERTION_OBSERVER_NOT_OBSERVED: %s", resolved.ID)
		}
		if !filter[resolved.ID.String()] {
			filter[resolved.ID.String()] = true
			observers = append(observers, resolved.ID)
		}
	}
	sort.Slice(observers, func(i, j int) bool { return observers[i].String() < observers[j].String() })
	revisions := graph.filteredRevisions(filter)
	sourceSet := revisionClosure(selected.ID, revisions)
	result := ImpactResult{Source: selected.ID, AssertionScope: "ALL_OBSERVED", ObserverSealIDs: observers, AllPaths: allPaths, MaxPaths: maxPaths}
	if len(filter) > 0 {
		result.AssertionScope = "FILTERED"
	}
	uniqueHeads := make(map[string]domain.ObjectID)
	for _, head := range observation.heads {
		uniqueHeads[head.String()] = head
	}
	ids := make([]string, 0, len(uniqueHeads))
	for id := range uniqueHeads {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, text := range ids {
		head := uniqueHeads[text]
		if head.Equal(selected.ID) {
			continue
		}
		paths := graph.impactPaths(head, sourceSet)
		if len(paths) == 0 {
			continue
		}
		sortPaths(paths)
		limit := 1
		if allPaths {
			limit = maxPaths
			if limit <= 0 {
				limit = 100
			}
		}
		shown := paths
		if len(shown) > limit {
			shown = shown[:limit]
		}
		record := ImpactRecord{Head: head, REFs: graph.refsFor(head), Truncated: len(paths) > len(shown)}
		for _, path := range shown {
			matched := path[len(path)-1]
			proof, err := graph.revisionProof(selected.ID, matched, revisions, filter)
			if err != nil {
				return ImpactResult{}, err
			}
			record.Paths = append(record.Paths, ImpactPath{CauseSealIDs: path, MatchedRevision: matched, Proof: proof})
		}
		result.Impacts = append(result.Impacts, record)
	}
	if err := r.revalidateHeads(ctx, observation, "impact"); err != nil {
		return ImpactResult{}, err
	}
	return result, nil
}

func (graph *observedGraph) filteredRevisions(filter map[string]bool) map[string][]domain.ObjectID {
	if len(filter) == 0 {
		return graph.revisions
	}
	result := make(map[string][]domain.ObjectID)
	for target, sources := range graph.assertions {
		for _, source := range sources {
			if filter[source.ObserverSeal.String()] {
				for _, previous := range source.CauseLink.PreviousRevisionSealOfTargetSeal {
					result[target] = appendUniqueID(result[target], previous)
				}
			}
		}
		sort.Slice(result[target], func(i, j int) bool { return result[target][i].String() < result[target][j].String() })
	}
	return result
}
func revisionClosure(source domain.ObjectID, revisions map[string][]domain.ObjectID) map[string]bool {
	result := make(map[string]bool)
	stack := []domain.ObjectID{source}
	for len(stack) > 0 {
		id := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if result[id.String()] {
			continue
		}
		result[id.String()] = true
		stack = append(stack, revisions[id.String()]...)
	}
	return result
}
func (graph *observedGraph) impactPaths(head domain.ObjectID, sources map[string]bool) [][]domain.ObjectID {
	result := [][]domain.ObjectID{}
	var visit func(domain.ObjectID, []domain.ObjectID)
	visit = func(id domain.ObjectID, path []domain.ObjectID) {
		if len(path) > 1 && sources[id.String()] {
			result = append(result, append([]domain.ObjectID(nil), path...))
			return
		}
		for _, next := range graph.causes[id.String()] {
			visit(next, append(path, next))
		}
	}
	visit(head, []domain.ObjectID{head})
	return result
}

func (graph *observedGraph) revisionProof(source, target domain.ObjectID, revisions map[string][]domain.ObjectID, filter map[string]bool) (RevisionProof, error) {
	queue := [][]domain.ObjectID{{source}}
	seen := map[string]bool{source.String(): true}
	var path []domain.ObjectID
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		last := current[len(current)-1]
		if last.Equal(target) {
			path = current
			break
		}
		for _, next := range revisions[last.String()] {
			if !seen[next.String()] {
				seen[next.String()] = true
				queue = append(queue, append(append([]domain.ObjectID(nil), current...), next))
			}
		}
	}
	if path == nil {
		return RevisionProof{}, fmt.Errorf("no in-scope revision proof from %s to %s", source, target)
	}
	proof := RevisionProof{SealIDs: path}
	for _, id := range path {
		proof.Observations = append(proof.Observations, graph.scopedRevisionObservation(id, filter))
	}
	for i := 0; i+1 < len(path); i++ {
		sources := []domainv5.AssertionSource{}
		for _, assertion := range graph.assertions[path[i].String()] {
			if len(filter) > 0 && !filter[assertion.ObserverSeal.String()] {
				continue
			}
			for _, previous := range assertion.CauseLink.PreviousRevisionSealOfTargetSeal {
				if previous.Equal(path[i+1]) {
					sources = append(sources, assertion)
				}
			}
		}
		proof.Edges = append(proof.Edges, RevisionEdge{Target: path[i], Previous: path[i+1], Sources: sources})
	}
	return proof, nil
}

func (graph *observedGraph) scopedRevisionObservation(id domain.ObjectID, filter map[string]bool) ScopedRevisionObservation {
	assertions := make([]domainv5.AssertionSource, 0, len(graph.assertions[id.String()]))
	for _, assertion := range graph.assertions[id.String()] {
		if len(filter) == 0 || filter[assertion.ObserverSeal.String()] {
			assertions = append(assertions, assertion)
		}
	}
	states := []string{}
	previous := []domain.ObjectID{}
	if len(assertions) == 0 {
		states = append(states, "ASSERTION_SCOPE_UNOBSERVED")
	} else {
		none, asserted := false, false
		for _, assertion := range assertions {
			if len(assertion.CauseLink.PreviousRevisionSealOfTargetSeal) == 0 {
				none = true
			} else {
				asserted = true
			}
			for _, candidate := range assertion.CauseLink.PreviousRevisionSealOfTargetSeal {
				previous = appendUniqueID(previous, candidate)
			}
		}
		if none {
			states = append(states, "ASSERTION_SCOPE_NONE_ASSERTED")
		}
		if asserted {
			states = append(states, "ASSERTION_SCOPE_ASSERTED")
		}
	}
	sort.Slice(previous, func(i, j int) bool { return previous[i].String() < previous[j].String() })
	return ScopedRevisionObservation{TargetSeal: id, PreviousStates: states, Assertions: assertions, StructuralPrevious: previous}
}
