package repository

import (
	"context"
	"fmt"
	"reflect"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
	domainv5 "github.com/mako10k/sealgraph/internal/domain/v5"
)

type LogEntry struct {
	MinimumDepth  int
	Resolved      domainv5.ResolvedSeal
	Revision      domainv5.RevisionObservation
	OutgoingEdges []RevisionEdge
}
type LogPath struct {
	SealIDs []domain.ObjectID
	Edges   []RevisionEdge
}
type LogResult struct {
	REF       string
	Head      domain.ObjectID
	AllPaths  bool
	MaxPaths  int
	Entries   []LogEntry
	Paths     []LogPath
	Truncated bool
}

func (r *Repository) Log(ctx context.Context, ref string, allPaths bool, maxPaths int) (LogResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return LogResult{}, err
	}
	observation, graph, err := r.buildObservation(ctx, "log")
	if err != nil {
		return LogResult{}, err
	}
	result, err := logFromObservation(ref, allPaths, maxPaths, observation, graph)
	if err != nil {
		return LogResult{}, err
	}
	if err := r.revalidateHeads(ctx, observation, "log"); err != nil {
		return LogResult{}, err
	}
	return result, nil
}

func logFromObservation(ref string, allPaths bool, maxPaths int, observation headObservation, graph *observedGraph) (LogResult, error) {
	head, ok := observation.heads[ref]
	if !ok {
		return LogResult{}, fmt.Errorf("resolve history head for %s: REF not found", ref)
	}
	depths := map[string]int{head.String(): 0}
	queue := []domain.ObjectID{head}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		for _, previous := range graph.revisions[id.String()] {
			if old, ok := depths[previous.String()]; !ok || depths[id.String()]+1 < old {
				depths[previous.String()] = depths[id.String()] + 1
				queue = append(queue, previous)
			}
		}
	}
	ids := make([]string, 0, len(depths))
	for id := range depths {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		if depths[ids[i]] != depths[ids[j]] {
			return depths[ids[i]] < depths[ids[j]]
		}
		return ids[i] < ids[j]
	})
	result := LogResult{REF: ref, Head: head, AllPaths: allPaths, MaxPaths: maxPaths}
	for _, text := range ids {
		id := domain.ObjectID{Hex: text}
		entry := LogEntry{MinimumDepth: depths[text], Resolved: graph.nodes[text], Revision: graph.revisionObservation(id)}
		for _, previous := range graph.revisions[text] {
			entry.OutgoingEdges = append(entry.OutgoingEdges, graph.revisionEdge(id, previous, nil))
		}
		result.Entries = append(result.Entries, entry)
	}
	if allPaths {
		limit := maxPaths
		if limit <= 0 {
			limit = 100
		}
		paths := graph.maximalRevisionPaths(head)
		result.Truncated = len(paths) > limit
		if len(paths) > limit {
			paths = paths[:limit]
		}
		for _, path := range paths {
			item := LogPath{SealIDs: path}
			for i := 0; i+1 < len(path); i++ {
				item.Edges = append(item.Edges, graph.revisionEdge(path[i], path[i+1], nil))
			}
			result.Paths = append(result.Paths, item)
		}
	}
	return result, nil
}

func (graph *observedGraph) revisionEdge(target, previous domain.ObjectID, filter map[string]bool) RevisionEdge {
	sources := []domainv5.AssertionSource{}
	for _, source := range graph.assertions[target.String()] {
		if len(filter) > 0 && !filter[source.ObserverSeal.String()] {
			continue
		}
		for _, candidate := range source.CauseLink.PreviousRevisionSealOfTargetSeal {
			if candidate.Equal(previous) {
				sources = append(sources, source)
				break
			}
		}
	}
	return RevisionEdge{Target: target, Previous: previous, Sources: sources}
}
func (graph *observedGraph) maximalRevisionPaths(head domain.ObjectID) [][]domain.ObjectID {
	paths := [][]domain.ObjectID{}
	var walk func(domain.ObjectID, []domain.ObjectID)
	walk = func(id domain.ObjectID, path []domain.ObjectID) {
		previous := graph.revisions[id.String()]
		if len(previous) == 0 {
			paths = append(paths, append([]domain.ObjectID(nil), path...))
			return
		}
		for _, next := range previous {
			walk(next, append(path, next))
		}
	}
	walk(head, []domain.ObjectID{head})
	sortPaths(paths)
	return paths
}

type CauseLinkChange struct {
	Target        domain.ObjectID
	Before, After *domainv5.CauseLink
}
type LinkLogEntry struct {
	MinimumNewerDepth                                                  int
	Newer, Previous                                                    domain.ObjectID
	NewerSealSchema, NewerProvenanceID, NewerProvenanceSchema          string
	PreviousSealSchema, PreviousProvenanceID, PreviousProvenanceSchema string
	SupportingAssertions                                               []domainv5.AssertionSource
	TargetRevision                                                     domainv5.RevisionObservation
	Changes                                                            []CauseLinkChange
}
type LinkLogResult struct {
	REF      string
	Head     domain.ObjectID
	Upstream *domain.ObjectID
	Entries  []LinkLogEntry
}

func (r *Repository) LinkLog(ctx context.Context, ref, upstreamSelector string) (LinkLogResult, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return LinkLogResult{}, err
	}
	observation, graph, err := r.buildObservation(ctx, "linklog")
	if err != nil {
		return LinkLogResult{}, err
	}
	log, err := logFromObservation(ref, false, 0, observation, graph)
	if err != nil {
		return LinkLogResult{}, err
	}
	result := LinkLogResult{REF: ref, Head: log.Head}
	if upstreamSelector != "" {
		resolved, err := r.resolveSelectorObserved(ctx, upstreamSelector, &observation, graph)
		if err != nil {
			return LinkLogResult{}, fmt.Errorf("resolve Link history filter %q: %w", upstreamSelector, err)
		}
		id := resolved.ID
		result.Upstream = &id
	}
	for _, entry := range log.Entries {
		for _, edge := range entry.OutgoingEdges {
			newer := graph.nodes[edge.Target.String()]
			previous := graph.nodes[edge.Previous.String()]
			changes := diffCauseLinks(previous.Provenance.CauseLinks, newer.Provenance.CauseLinks)
			if result.Upstream != nil {
				filtered := changes[:0]
				for _, change := range changes {
					if change.Target.Equal(*result.Upstream) {
						filtered = append(filtered, change)
					}
				}
				changes = filtered
				if len(changes) == 0 {
					continue
				}
			}
			result.Entries = append(result.Entries, LinkLogEntry{
				MinimumNewerDepth: entry.MinimumDepth, Newer: edge.Target,
				NewerSealSchema: newer.Seal.Schema, NewerProvenanceID: newer.Seal.Provenance.String(), NewerProvenanceSchema: newer.Provenance.Schema,
				Previous: edge.Previous, PreviousSealSchema: previous.Seal.Schema,
				PreviousProvenanceID: previous.Seal.Provenance.String(), PreviousProvenanceSchema: previous.Provenance.Schema,
				SupportingAssertions: edge.Sources, TargetRevision: graph.revisionObservation(edge.Target), Changes: changes,
			})
		}
	}
	sort.Slice(result.Entries, func(i, j int) bool {
		a, b := result.Entries[i], result.Entries[j]
		if a.MinimumNewerDepth != b.MinimumNewerDepth {
			return a.MinimumNewerDepth < b.MinimumNewerDepth
		}
		if !a.Newer.Equal(b.Newer) {
			return a.Newer.String() < b.Newer.String()
		}
		return a.Previous.String() < b.Previous.String()
	})
	if err := r.revalidateHeads(ctx, observation, "linklog"); err != nil {
		return LinkLogResult{}, err
	}
	return result, nil
}

func diffCauseLinks(before, after []domainv5.CauseLink) []CauseLinkChange {
	beforeMap := make(map[string]domainv5.CauseLink, len(before))
	afterMap := make(map[string]domainv5.CauseLink, len(after))
	keys := make(map[string]bool)
	for _, link := range before {
		beforeMap[link.TargetSeal.String()] = link
		keys[link.TargetSeal.String()] = true
	}
	for _, link := range after {
		afterMap[link.TargetSeal.String()] = link
		keys[link.TargetSeal.String()] = true
	}
	ids := make([]string, 0, len(keys))
	for id := range keys {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := []CauseLinkChange{}
	for _, id := range ids {
		left, leftOK := beforeMap[id]
		right, rightOK := afterMap[id]
		if leftOK && rightOK && reflect.DeepEqual(left, right) {
			continue
		}
		change := CauseLinkChange{Target: domain.ObjectID{Hex: id}}
		if leftOK {
			copy := left
			change.Before = &copy
		}
		if rightOK {
			copy := right
			change.After = &copy
		}
		result = append(result, change)
	}
	return result
}

type SealComparison struct{ From, To domainv5.ResolvedSeal }

func (r *Repository) DiffSelectors(ctx context.Context, fromSelector, toSelector string) (SealComparison, error) {
	observation, graph, err := r.buildObservation(ctx, "compare")
	if err != nil {
		return SealComparison{}, err
	}
	from, err := r.resolveSelectorObserved(ctx, fromSelector, &observation, graph)
	if err != nil {
		return SealComparison{}, fmt.Errorf("resolve left comparison selector: %w", err)
	}
	to, err := r.resolveSelectorObserved(ctx, toSelector, &observation, graph)
	if err != nil {
		return SealComparison{}, fmt.Errorf("resolve right comparison selector: %w", err)
	}
	if err := r.revalidateHeads(ctx, observation, "compare"); err != nil {
		return SealComparison{}, err
	}
	return SealComparison{From: from.Resolved, To: to.Resolved}, nil
}
