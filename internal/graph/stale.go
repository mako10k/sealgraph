package graph

import (
	"context"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/store"
)

type DirectStaleLink struct {
	Link        domain.Link
	CurrentHead domain.ObjectID
}

// DirectStale derives direct staleness exclusively from immutable links and
// current logical REF heads. It never writes derived state.
func DirectStale(ctx context.Context, payload domain.SealPayload, refs store.RefStore) ([]DirectStaleLink, error) {
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
