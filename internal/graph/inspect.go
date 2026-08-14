package graph

import (
	"context"
	"fmt"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

// LoadSealFunc loads and validates one immutable canonical seal.
type LoadSealFunc func(context.Context, domain.ObjectID) (domain.SealPayload, error)

// SealIdentity identifies one exact node in the immutable seal DAG.
type SealIdentity struct {
	REF  string
	Seal domain.ObjectID
}

func (identity SealIdentity) String() string {
	return identity.REF + "@" + identity.Seal.String()
}

// StalePath records one stale relation below the inspected seal. Nodes starts
// at the inspected seal and ends at the seal containing Link.
type StalePath struct {
	Nodes       []SealIdentity
	Link        domain.Link
	CurrentHead domain.ObjectID
}

// Inspection is derived from immutable seals and current REF heads. Direct and
// Transitive are intentionally not persisted.
type Inspection struct {
	Direct     []DirectStaleLink
	Transitive []StalePath
}

// CycleError identifies an invalid cycle in the immutable seal-ID graph.
type CycleError struct {
	Path []SealIdentity
}

func (err *CycleError) Error() string {
	parts := make([]string, len(err.Path))
	for i, identity := range err.Path {
		parts[i] = identity.String()
	}
	return "cycle detected in immutable seal DAG: " + strings.Join(parts, " -> ")
}

// Inspect derives direct and transitive staleness for one exact seal and also
// validates that its reachable immutable seal graph is acyclic and that each
// target seal belongs to the REF named by its link.
func Inspect(ctx context.Context, root SealIdentity, payload domain.SealPayload, refs HeadResolver, load LoadSealFunc) (Inspection, error) {
	if payload.REF != root.REF {
		return Inspection{}, fmt.Errorf("seal %s belongs to REF %s, not %s", root.Seal, payload.REF, root.REF)
	}
	walker := inspectionWalker{
		ctx:      ctx,
		refs:     refs,
		load:     load,
		active:   make(map[string]int),
		complete: make(map[string]bool),
	}
	if err := walker.visit(root, payload, []SealIdentity{root}); err != nil {
		return Inspection{}, err
	}
	return walker.result, nil
}

type inspectionWalker struct {
	ctx      context.Context
	refs     HeadResolver
	load     LoadSealFunc
	active   map[string]int
	complete map[string]bool
	result   Inspection
}

func (walker *inspectionWalker) visit(identity SealIdentity, payload domain.SealPayload, path []SealIdentity) error {
	if err := walker.ctx.Err(); err != nil {
		return err
	}
	key := identity.Seal.String()
	walker.active[key] = len(path) - 1
	defer delete(walker.active, key)

	for _, link := range payload.Links {
		head, err := walker.refs.Resolve(walker.ctx, link.TargetREF)
		if err != nil {
			return fmt.Errorf("resolve dependency head %s while deriving graph: %w", link.TargetREF, err)
		}
		if !head.Equal(link.TargetSeal) {
			if len(path) == 1 {
				walker.result.Direct = append(walker.result.Direct, DirectStaleLink{Link: link, CurrentHead: head})
			} else {
				walker.result.Transitive = append(walker.result.Transitive, StalePath{
					Nodes: append([]SealIdentity(nil), path...), Link: link, CurrentHead: head,
				})
			}
		}

		target := SealIdentity{REF: link.TargetREF, Seal: link.TargetSeal}
		if start, found := walker.active[target.Seal.String()]; found {
			cycle := append([]SealIdentity(nil), path[start:]...)
			cycle = append(cycle, target)
			return &CycleError{Path: cycle}
		}
		targetPayload, err := walker.load(walker.ctx, target.Seal)
		if err != nil {
			return fmt.Errorf("load dependency %s: %w", target, err)
		}
		if targetPayload.REF != target.REF {
			return fmt.Errorf("dependency seal %s belongs to REF %s, not %s", target.Seal, targetPayload.REF, target.REF)
		}
		if walker.complete[target.Seal.String()] {
			continue
		}
		if err := walker.visit(target, targetPayload, append(path, target)); err != nil {
			return err
		}
	}
	walker.complete[key] = true
	return nil
}
