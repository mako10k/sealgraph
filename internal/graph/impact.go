package graph

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

// Impact is one current downstream REF whose exact immutable dependency
// closure names the selected upstream REF. Path runs downstream-to-upstream.
type Impact struct {
	REF    string
	Head   domain.ObjectID
	Direct bool
	Path   []SealIdentity
}

// ReverseImpact derives downstream impact from current REF heads while walking
// the exact historical seal identities stored in each closure.
func ReverseImpact(ctx context.Context, sourceREF string, current []SealIdentity, load LoadSealFunc) ([]Impact, error) {
	ordered := append([]SealIdentity(nil), current...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].REF < ordered[j].REF })
	var result []Impact
	for _, downstream := range ordered {
		if downstream.REF == sourceREF {
			continue
		}
		payload, err := load(ctx, downstream.Seal)
		if err != nil {
			return nil, fmt.Errorf("load current seal %s: %w", downstream, err)
		}
		paths, err := findImpactPaths(ctx, sourceREF, downstream, payload, load, nil, make(map[string]int))
		if err != nil {
			return nil, err
		}
		for _, path := range paths {
			result = append(result, Impact{REF: downstream.REF, Head: downstream.Seal, Direct: len(path) == 2, Path: path})
		}
	}
	return result, nil
}

func findImpactPaths(
	ctx context.Context,
	sourceREF string,
	identity SealIdentity,
	payload domain.SealPayload,
	load LoadSealFunc,
	path []SealIdentity,
	active map[string]int,
) ([][]SealIdentity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if payload.REF != identity.REF {
		return nil, fmt.Errorf("seal %s belongs to REF %s, not %s", identity.Seal, payload.REF, identity.REF)
	}
	path = append(path, identity)
	key := identity.Seal.String()
	active[key] = len(path) - 1
	defer delete(active, key)

	var result [][]SealIdentity
	for _, link := range payload.Links {
		target := SealIdentity{REF: link.TargetREF, Seal: link.TargetSeal}
		if start, found := active[target.Seal.String()]; found {
			cycle := append([]SealIdentity(nil), path[start:]...)
			cycle = append(cycle, target)
			return nil, &CycleError{Path: cycle}
		}
		targetPayload, err := load(ctx, target.Seal)
		if err != nil {
			return nil, fmt.Errorf("load dependency %s: %w", target, err)
		}
		if targetPayload.REF != target.REF {
			return nil, fmt.Errorf("dependency seal %s belongs to REF %s, not %s", target.Seal, targetPayload.REF, target.REF)
		}
		if link.TargetREF == sourceREF {
			result = append(result, append(append([]SealIdentity(nil), path...), target))
			continue
		}
		foundPaths, err := findImpactPaths(ctx, sourceREF, target, targetPayload, load, path, active)
		if err != nil {
			return nil, err
		}
		result = append(result, foundPaths...)
	}
	return result, nil
}
