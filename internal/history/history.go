// Package history derives read-only seal generation history and semantic
// changes from immutable canonical seal payloads.
package history

import (
	"context"
	"fmt"
	"strings"

	"github.com/mako10k/sealgraph/internal/domain"
)

// LoadSealFunc loads and validates one immutable canonical seal.
type LoadSealFunc func(context.Context, domain.ObjectID) (domain.SealPayload, error)

// Identity identifies one exact immutable generation. REF is optional display
// scope and is never validated as Seal ownership.
type Identity struct {
	REF  string
	Seal domain.ObjectID
}

func (identity Identity) String() string {
	return identity.REF + "@" + identity.Seal.String()
}

// Entry is one validated immutable generation. Walk returns entries newest
// first, following only the payload parent chain.
type Entry struct {
	ID      domain.ObjectID
	Payload domain.SealPayload
}

// CycleError identifies a repeated seal ID in a parent chain.
type CycleError struct {
	Path []Identity
}

func (err *CycleError) Error() string {
	parts := make([]string, len(err.Path))
	for i, identity := range err.Path {
		parts[i] = identity.String()
	}
	return "cycle detected in immutable seal parent history: " + strings.Join(parts, " -> ")
}

// Walk loads the complete parent chain from one REF head. REF is presentation
// context only; format-4 Seals do not contain an owner.
func Walk(ctx context.Context, ref string, head domain.ObjectID, load LoadSealFunc) ([]Entry, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	if err := head.ValidateNative(); err != nil {
		return nil, fmt.Errorf("invalid history head for %s: %w", ref, err)
	}

	var entries []Entry
	seen := make(map[string]int)
	current := head
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		key := current.String()
		if start, found := seen[key]; found {
			path := make([]Identity, 0, len(entries)-start+1)
			for _, entry := range entries[start:] {
				path = append(path, Identity{REF: ref, Seal: entry.ID})
			}
			path = append(path, Identity{REF: ref, Seal: current})
			return nil, &CycleError{Path: path}
		}
		seen[key] = len(entries)

		payload, err := load(ctx, current)
		if err != nil {
			return nil, fmt.Errorf("load history generation %s@%s: %w", ref, current, err)
		}
		entries = append(entries, Entry{ID: current, Payload: payload})
		if payload.ParentRevision == nil {
			return entries, nil
		}
		current = *payload.ParentRevision
	}
}
