package graph

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/revision"
)

type LoadSealFunc func(context.Context, domain.ObjectID) (domain.SealPayload, error)

type StaleFacts struct {
	Self       bool
	Direct     []domain.ObjectID
	Transitive [][]domain.ObjectID
}

type CauseCycleError struct {
	Path []domain.ObjectID
}

func (err *CauseCycleError) Error() string {
	parts := make([]string, len(err.Path))
	for i, id := range err.Path {
		parts[i] = id.String()
	}
	return "cycle detected in immutable Cause DAG: " + strings.Join(parts, " -> ")
}

type Analysis struct {
	revisions *revision.Index
	nodes     map[string]domain.SealPayload
}

func (analysis *Analysis) Payload(id domain.ObjectID) (domain.SealPayload, bool) {
	payload, found := analysis.nodes[id.String()]
	return payload, found
}

func Build(ctx context.Context, roots []domain.ObjectID, revisions *revision.Index, load LoadSealFunc) (*Analysis, error) {
	builder := causeBuilder{
		ctx:               ctx,
		load:              load,
		nodes:             make(map[string]domain.SealPayload),
		state:             make(map[string]uint8),
		position:          make(map[string]int),
		validatedRevision: make(map[string]bool),
	}
	ordered := append([]domain.ObjectID(nil), roots...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	for _, root := range ordered {
		if err := builder.visit(root); err != nil {
			return nil, fmt.Errorf("validate Cause graph from %s: %w", root, err)
		}
	}
	return &Analysis{revisions: revisions, nodes: builder.nodes}, nil
}

type causeBuilder struct {
	ctx               context.Context
	load              LoadSealFunc
	nodes             map[string]domain.SealPayload
	state             map[string]uint8
	path              []domain.ObjectID
	position          map[string]int
	validatedRevision map[string]bool
}

func (builder *causeBuilder) visit(id domain.ObjectID) error {
	if err := builder.ctx.Err(); err != nil {
		return err
	}
	key := id.String()
	switch builder.state[key] {
	case 1:
		start := builder.position[key]
		cycle := append([]domain.ObjectID(nil), builder.path[start:]...)
		return &CauseCycleError{Path: append(cycle, id)}
	case 2:
		return nil
	}
	payload, err := builder.load(builder.ctx, id)
	if err != nil {
		return fmt.Errorf("load Cause seal %s: %w", id, err)
	}
	if err := builder.validateParentChain(id); err != nil {
		return err
	}
	builder.nodes[key] = payload
	builder.state[key] = 1
	builder.position[key] = len(builder.path)
	builder.path = append(builder.path, id)
	for _, link := range payload.Links {
		if err := builder.visit(link.TargetSeal); err != nil {
			return err
		}
	}
	builder.path = builder.path[:len(builder.path)-1]
	delete(builder.position, key)
	builder.state[key] = 2
	return nil
}

func (builder *causeBuilder) validateParentChain(id domain.ObjectID) error {
	if builder.validatedRevision[id.String()] {
		return nil
	}
	ancestors, err := revision.WalkAncestors(builder.ctx, id, revision.LoadSealFunc(builder.load))
	if err != nil {
		return fmt.Errorf("validate revision ancestry for Cause seal %s: %w", id, err)
	}
	for _, ancestor := range ancestors {
		builder.validatedRevision[ancestor.String()] = true
	}
	return nil
}

func (analysis *Analysis) Facts(head domain.ObjectID) StaleFacts {
	facts := StaleFacts{Self: analysis.revisions.State(head) == revision.StaleRevision}
	payload, found := analysis.nodes[head.String()]
	if !found {
		return facts
	}
	for _, link := range payload.Links {
		if !analysis.revisions.IsCurrentLeaf(link.TargetSeal) {
			facts.Direct = append(facts.Direct, link.TargetSeal)
		}
	}
	if len(facts.Direct) == 0 {
		facts.Transitive = analysis.transitiveStalePaths(payload.Links)
	}
	return facts
}

func (analysis *Analysis) transitiveStalePaths(links []domain.Link) [][]domain.ObjectID {
	var result [][]domain.ObjectID
	for _, link := range links {
		analysis.collectFirstStale(link.TargetSeal, []domain.ObjectID{link.TargetSeal}, &result)
	}
	sortPaths(result)
	return result
}

func (analysis *Analysis) collectFirstStale(id domain.ObjectID, path []domain.ObjectID, result *[][]domain.ObjectID) {
	payload := analysis.nodes[id.String()]
	for _, link := range payload.Links {
		nextPath := append(append([]domain.ObjectID(nil), path...), link.TargetSeal)
		if !analysis.revisions.IsCurrentLeaf(link.TargetSeal) {
			*result = append(*result, nextPath)
			continue
		}
		analysis.collectFirstStale(link.TargetSeal, nextPath, result)
	}
}

func (analysis *Analysis) Frontier(staleHeads map[string]domain.ObjectID) map[string]bool {
	staleIDs := make(map[string]bool, len(staleHeads))
	for _, head := range staleHeads {
		staleIDs[head.String()] = true
	}
	result := make(map[string]bool, len(staleHeads))
	for ref, head := range staleHeads {
		result[ref] = !analysis.strictClosureContains(head, staleIDs)
	}
	return result
}

func (analysis *Analysis) strictClosureContains(head domain.ObjectID, wanted map[string]bool) bool {
	seen := make(map[string]bool)
	stack := directTargets(analysis.nodes[head.String()])
	for len(stack) != 0 {
		last := len(stack) - 1
		id := stack[last]
		stack = stack[:last]
		if seen[id.String()] {
			continue
		}
		seen[id.String()] = true
		if wanted[id.String()] {
			return true
		}
		stack = append(stack, directTargets(analysis.nodes[id.String()])...)
	}
	return false
}

func directTargets(payload domain.SealPayload) []domain.ObjectID {
	result := make([]domain.ObjectID, len(payload.Links))
	for i, link := range payload.Links {
		result[i] = link.TargetSeal
	}
	return result
}

type Impact struct {
	Head      domain.ObjectID
	REFs      []string
	Paths     [][]domain.ObjectID
	Truncated bool
}

func (analysis *Analysis) Impact(ctx context.Context, source domain.ObjectID, heads []revision.Head, load LoadSealFunc, allPaths bool, maxPaths int) ([]Impact, error) {
	ancestors, err := revision.WalkAncestors(ctx, source, revision.LoadSealFunc(load))
	if err != nil {
		return nil, fmt.Errorf("validate impact source ancestry: %w", err)
	}
	wanted := make(map[string]bool, len(ancestors))
	for _, id := range ancestors {
		wanted[id.String()] = true
	}
	aliases := groupHeadAliases(heads)
	var result []Impact
	for _, head := range sortedHeadIDs(aliases) {
		if head.Equal(source) || !analysis.reaches(head, wanted, make(map[string]bool)) {
			continue
		}
		limit := 1
		if allPaths {
			limit = maxPaths
		}
		paths, truncated := analysis.orderedPaths(head, wanted, limit)
		result = append(result, Impact{Head: head, REFs: aliases[head.String()], Paths: paths, Truncated: allPaths && truncated})
	}
	return result, nil
}

func (analysis *Analysis) reaches(id domain.ObjectID, wanted map[string]bool, memo map[string]bool) bool {
	if value, found := memo[id.String()]; found {
		return value
	}
	for _, target := range directTargets(analysis.nodes[id.String()]) {
		if wanted[target.String()] || analysis.reaches(target, wanted, memo) {
			memo[id.String()] = true
			return true
		}
	}
	memo[id.String()] = false
	return false
}

func (analysis *Analysis) orderedPaths(head domain.ObjectID, wanted map[string]bool, limit int) ([][]domain.ObjectID, bool) {
	queue := [][]domain.ObjectID{{head}}
	var result [][]domain.ObjectID
	for len(queue) != 0 && len(result) <= limit {
		path := queue[0]
		queue = queue[1:]
		last := path[len(path)-1]
		for _, target := range directTargets(analysis.nodes[last.String()]) {
			next := append(append([]domain.ObjectID(nil), path...), target)
			if wanted[target.String()] {
				result = append(result, next)
				continue
			}
			queue = append(queue, next)
		}
		sortPaths(queue)
	}
	if len(result) > limit {
		return result[:limit], true
	}
	return result, false
}

func groupHeadAliases(heads []revision.Head) map[string][]string {
	result := make(map[string][]string)
	for _, head := range heads {
		result[head.Seal.String()] = append(result[head.Seal.String()], head.REF)
	}
	for key := range result {
		sort.Strings(result[key])
	}
	return result
}

func sortedHeadIDs(aliases map[string][]string) []domain.ObjectID {
	result := make([]domain.ObjectID, 0, len(aliases))
	for key := range aliases {
		id, _ := domain.ParseObjectID(key)
		result = append(result, id)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}

func sortPaths(paths [][]domain.ObjectID) {
	sort.Slice(paths, func(i, j int) bool {
		if len(paths[i]) != len(paths[j]) {
			return len(paths[i]) < len(paths[j])
		}
		for index := range paths[i] {
			left, right := paths[i][index].String(), paths[j][index].String()
			if left != right {
				return left < right
			}
		}
		return false
	})
}
