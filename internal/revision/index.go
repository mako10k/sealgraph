// Package revision derives the active format-4 revision DAG from current REF
// heads and immutable parent_revision edges.
package revision

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

type LoadSealFunc func(context.Context, domain.ObjectID) (domain.SealPayload, error)

type Head struct {
	REF  string
	Seal domain.ObjectID
}

type State string

const (
	CurrentLeaf        State = "CURRENT_LEAF"
	StaleRevision      State = "STALE_REVISION"
	HistoricalDetached State = "HISTORICAL_OR_DETACHED"
)

type Node struct {
	ID       domain.ObjectID
	Payload  domain.SealPayload
	Children []domain.ObjectID
	REFs     []string
}

type Index struct {
	nodes map[string]Node
}

type Record struct {
	ID     domain.ObjectID
	Parent *domain.ObjectID
}

type CycleError struct {
	Path []domain.ObjectID
}

func (err *CycleError) Error() string {
	parts := make([]string, len(err.Path))
	for i, id := range err.Path {
		parts[i] = id.String()
	}
	return "cycle detected in immutable parent_revision DAG: " + strings.Join(parts, " -> ")
}

func Build(ctx context.Context, heads []Head, load LoadSealFunc) (*Index, error) {
	ordered := append([]Head(nil), heads...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].REF < ordered[j].REF })
	builder := indexBuilder{
		ctx:      ctx,
		load:     load,
		nodes:    make(map[string]Node),
		state:    make(map[string]uint8),
		position: make(map[string]int),
	}
	for _, head := range ordered {
		if err := domain.ValidateREF(head.REF); err != nil {
			return nil, fmt.Errorf("invalid current REF %q: %w", head.REF, err)
		}
		if err := builder.visit(head.Seal); err != nil {
			return nil, fmt.Errorf("build active revision DAG from %s@%s: %w", head.REF, head.Seal, err)
		}
		node := builder.nodes[head.Seal.String()]
		node.REFs = append(node.REFs, head.REF)
		builder.nodes[head.Seal.String()] = node
	}
	for key, node := range builder.nodes {
		sort.Slice(node.Children, func(i, j int) bool { return node.Children[i].String() < node.Children[j].String() })
		sort.Strings(node.REFs)
		builder.nodes[key] = node
	}
	return &Index{nodes: builder.nodes}, nil
}

type indexBuilder struct {
	ctx      context.Context
	load     LoadSealFunc
	nodes    map[string]Node
	state    map[string]uint8
	path     []domain.ObjectID
	position map[string]int
}

func (builder *indexBuilder) visit(id domain.ObjectID) error {
	if err := builder.ctx.Err(); err != nil {
		return err
	}
	key := id.String()
	switch builder.state[key] {
	case 1:
		start := builder.position[key]
		cycle := append([]domain.ObjectID(nil), builder.path[start:]...)
		return &CycleError{Path: append(cycle, id)}
	case 2:
		return nil
	}
	payload, err := builder.load(builder.ctx, id)
	if err != nil {
		return fmt.Errorf("load revision %s: %w", id, err)
	}
	builder.state[key] = 1
	builder.position[key] = len(builder.path)
	builder.path = append(builder.path, id)
	builder.nodes[key] = Node{ID: id, Payload: payload}
	if payload.ParentRevision != nil {
		parent := *payload.ParentRevision
		if err := builder.visit(parent); err != nil {
			return err
		}
		parentNode := builder.nodes[parent.String()]
		if !containsID(parentNode.Children, id) {
			parentNode.Children = append(parentNode.Children, id)
			builder.nodes[parent.String()] = parentNode
		}
	}
	builder.path = builder.path[:len(builder.path)-1]
	delete(builder.position, key)
	builder.state[key] = 2
	return nil
}

func containsID(ids []domain.ObjectID, wanted domain.ObjectID) bool {
	for _, id := range ids {
		if id.Equal(wanted) {
			return true
		}
	}
	return false
}

func (index *Index) State(id domain.ObjectID) State {
	node, found := index.nodes[id.String()]
	if !found {
		return HistoricalDetached
	}
	if len(node.Children) != 0 {
		return StaleRevision
	}
	return CurrentLeaf
}

func (index *Index) IsCurrentLeaf(id domain.ObjectID) bool {
	return index.State(id) == CurrentLeaf
}

func (index *Index) Node(id domain.ObjectID) (Node, bool) {
	node, found := index.nodes[id.String()]
	return node, found
}

func (index *Index) Nodes() []Node {
	result := make([]Node, 0, len(index.nodes))
	for _, node := range index.nodes {
		result = append(result, node)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result
}

func (index *Index) Records() []Record {
	nodes := index.Nodes()
	result := make([]Record, len(nodes))
	for i, node := range nodes {
		result[i] = Record{ID: node.ID, Parent: copyID(node.Payload.ParentRevision)}
	}
	return result
}

func Restore(ctx context.Context, heads []Head, records []Record, load LoadSealFunc) (*Index, error) {
	recordByID, err := validateRecords(records)
	if err != nil {
		return nil, err
	}
	if err := validateRecordedReachability(heads, recordByID); err != nil {
		return nil, err
	}
	nodes := make(map[string]Node, len(records))
	for _, record := range records {
		payload, err := load(ctx, record.ID)
		if err != nil {
			return nil, fmt.Errorf("load cached active revision %s: %w", record.ID, err)
		}
		if !optionalIDsEqual(payload.ParentRevision, record.Parent) {
			return nil, fmt.Errorf("cached parent for %s does not match canonical Seal", record.ID)
		}
		nodes[record.ID.String()] = Node{ID: record.ID, Payload: payload}
	}
	for _, record := range records {
		if record.Parent == nil {
			continue
		}
		parent := nodes[record.Parent.String()]
		parent.Children = append(parent.Children, record.ID)
		nodes[record.Parent.String()] = parent
	}
	for _, head := range heads {
		node := nodes[head.Seal.String()]
		node.REFs = append(node.REFs, head.REF)
		nodes[head.Seal.String()] = node
	}
	for key, node := range nodes {
		sort.Slice(node.Children, func(i, j int) bool { return node.Children[i].String() < node.Children[j].String() })
		sort.Strings(node.REFs)
		nodes[key] = node
	}
	return &Index{nodes: nodes}, nil
}

func validateRecords(records []Record) (map[string]Record, error) {
	result := make(map[string]Record, len(records))
	previous := ""
	for _, record := range records {
		if err := record.ID.ValidateNative(); err != nil {
			return nil, fmt.Errorf("invalid cached revision ID: %w", err)
		}
		key := record.ID.String()
		if previous != "" && key <= previous {
			return nil, errors.New("cached revision records are not strictly sorted and unique")
		}
		if record.Parent != nil {
			if err := record.Parent.ValidateNative(); err != nil {
				return nil, fmt.Errorf("invalid cached parent for %s: %w", record.ID, err)
			}
		}
		result[key] = record
		previous = key
	}
	return result, nil
}

func validateRecordedReachability(heads []Head, records map[string]Record) error {
	reached := make(map[string]bool, len(records))
	for _, head := range heads {
		active := make(map[string]bool)
		current := head.Seal
		for {
			key := current.String()
			if active[key] {
				return fmt.Errorf("cached parent cycle reaches %s", current)
			}
			record, found := records[key]
			if !found {
				return fmt.Errorf("cached active revisions omit %s reached from REF %s", current, head.REF)
			}
			active[key], reached[key] = true, true
			if record.Parent == nil {
				break
			}
			current = *record.Parent
		}
	}
	if len(reached) != len(records) {
		return errors.New("cached active revisions contain a record unreachable from current REF heads")
	}
	return nil
}

func copyID(id *domain.ObjectID) *domain.ObjectID {
	if id == nil {
		return nil
	}
	copy := *id
	return &copy
}

func optionalIDsEqual(left, right *domain.ObjectID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return left.Equal(*right)
}

func WalkAncestors(ctx context.Context, start domain.ObjectID, load LoadSealFunc) ([]domain.ObjectID, error) {
	index, err := Build(ctx, []Head{{REF: "validation", Seal: start}}, load)
	if err != nil {
		return nil, err
	}
	var result []domain.ObjectID
	current := start
	for {
		result = append(result, current)
		node, found := index.Node(current)
		if !found || node.Payload.ParentRevision == nil {
			return result, nil
		}
		current = *node.Payload.ParentRevision
	}
}
