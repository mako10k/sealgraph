package repository

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/mako10k/sealgraph/internal/domain"
)

var benchmarkStaleStatuses []RefStatus

func TestStaleFactsMemoizeSharedCauseSubgraphWithoutChangingPaths(t *testing.T) {
	graph, heads := sharedCauseBenchmarkGraph(2, 3)
	_, direct1, paths1 := graph.staleFacts(heads[0])
	_, direct2, paths2 := graph.staleFacts(heads[1])
	if len(direct1) != 0 || len(direct2) != 0 {
		t.Fatalf("direct stale: first=%v second=%v", direct1, direct2)
	}
	if !reflect.DeepEqual(paths1, paths2) || len(paths1) != 1 || len(paths1[0]) != 4 {
		t.Fatalf("transitive paths: first=%v second=%v", paths1, paths2)
	}
	if len(graph.stalePaths) != 4 {
		t.Fatalf("memoized nodes=%d, want 4", len(graph.stalePaths))
	}
}

func TestFrontierMemoizesSharedCauseClosure(t *testing.T) {
	graph, heads := sharedCauseBenchmarkGraph(2, 3)
	stale := benchmarkObjectID(4)
	staleHeads := map[string]bool{stale.String(): true}
	blocked := make(map[string]bool)
	for _, head := range heads {
		if graph.isFrontier(head, staleHeads, blocked) {
			t.Fatalf("head %s unexpectedly selected for frontier", head)
		}
	}
	if !graph.isFrontier(stale, staleHeads, blocked) {
		t.Fatal("stale leaf head was not selected for frontier")
	}
}

func BenchmarkStaleFactsSharedCauseSubgraph(b *testing.B) {
	const (
		headCount = 2_000
		depth     = 200
	)
	graph, heads := sharedCauseBenchmarkGraph(headCount, depth)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		graph.stalePaths = make(map[string][][]domain.ObjectID)
		statuses := make([]RefStatus, 0, len(heads))
		for index, head := range heads {
			self, direct, transitive := graph.staleFacts(head)
			statuses = append(statuses, RefStatus{
				REF: fmt.Sprintf("head-%d", index), StaleSelf: self,
				StaleDirect: direct, StaleTransitive: transitive,
			})
		}
		benchmarkStaleStatuses = statuses
	}
}

func sharedCauseBenchmarkGraph(headCount, depth int) (*observedGraph, []domain.ObjectID) {
	graph := &observedGraph{
		causes:     make(map[string][]domain.ObjectID),
		children:   make(map[string][]domain.ObjectID),
		active:     make(map[string]bool),
		stalePaths: make(map[string][][]domain.ObjectID),
	}
	shared := benchmarkObjectID(1)
	graph.active[shared.String()] = true
	current := shared
	for index := 1; index < depth; index++ {
		next := benchmarkObjectID(index + 1)
		graph.active[next.String()] = true
		graph.causes[current.String()] = []domain.ObjectID{next}
		current = next
	}
	stale := benchmarkObjectID(depth + 1)
	graph.causes[current.String()] = []domain.ObjectID{stale}
	heads := make([]domain.ObjectID, headCount)
	for index := range heads {
		head := benchmarkObjectID(depth + index + 2)
		heads[index] = head
		graph.active[head.String()] = true
		graph.causes[head.String()] = []domain.ObjectID{shared}
	}
	return graph, heads
}

func benchmarkObjectID(value int) domain.ObjectID {
	return domain.ObjectID{Hex: fmt.Sprintf("%064x", value)}
}
