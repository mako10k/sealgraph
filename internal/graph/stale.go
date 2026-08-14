package graph

import (
	"context"
	"fmt"
	"sort"

	"github.com/mako10k/sealgraph/internal/domain"
)

// HeadResolver resolves one logical REF to its observed current seal.
// Graph derivation needs no mutation or REF-listing capability.
type HeadResolver interface {
	Resolve(context.Context, string) (domain.ObjectID, error)
}

type DirectStaleLink struct {
	Link        domain.Link
	CurrentHead domain.ObjectID
}

// DirectStale derives direct staleness exclusively from immutable links and
// current logical REF heads. It never writes derived state.
func DirectStale(ctx context.Context, payload domain.SealPayload, refs HeadResolver) ([]DirectStaleLink, error) {
	var stale []DirectStaleLink
	for _, link := range payload.Links {
		head, err := refs.Resolve(ctx, link.TargetREF)
		if err != nil {
			return nil, fmt.Errorf("resolve dependency head %s while deriving status: %w", link.TargetREF, err)
		}
		if !head.Equal(link.TargetSeal) {
			stale = append(stale, DirectStaleLink{Link: link, CurrentHead: head})
		}
	}
	return stale, nil
}

// StaleFrontier returns stale REFs whose current direct upstream REFs are not
// themselves stale. Inputs are treated as sets; output is deduplicated and
// bytewise lexically ordered.
func StaleFrontier(staleREFs []string, directUpstreams map[string][]string) []string {
	stale := make(map[string]struct{}, len(staleREFs))
	for _, ref := range staleREFs {
		stale[ref] = struct{}{}
	}
	frontier := make([]string, 0, len(stale))
	for ref := range stale {
		blocked := false
		for _, upstream := range directUpstreams[ref] {
			if _, found := stale[upstream]; found {
				blocked = true
				break
			}
		}
		if !blocked {
			frontier = append(frontier, ref)
		}
	}
	sort.Strings(frontier)
	return frontier
}
