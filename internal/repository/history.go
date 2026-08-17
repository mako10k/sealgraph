package repository

import (
	"context"
	"fmt"

	"github.com/mako10k/sealgraph/internal/domain"
	"github.com/mako10k/sealgraph/internal/history"
)

func (r *Repository) Log(ctx context.Context, ref string) ([]history.Entry, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return nil, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("resolve history head for %s: %w; seal that REF first or select an existing REF", ref, err)
	}
	entries, err := history.Walk(ctx, ref, head, history.LoadSealFunc(r.LoadSeal))
	if err != nil {
		return nil, fmt.Errorf("read immutable history for %s: %w", ref, err)
	}
	return entries, nil
}

func (r *Repository) LinkLog(ctx context.Context, ref, upstreamSelector string) ([]history.LinkLogEntry, string, error) {
	target := ""
	if upstreamSelector != "" {
		resolved, err := r.ResolveSelector(ctx, upstreamSelector)
		if err != nil {
			return nil, "", fmt.Errorf("resolve Link history filter %q: %w", upstreamSelector, err)
		}
		target = resolved.ID.String()
	}
	entries, err := r.Log(ctx, ref)
	if err != nil {
		return nil, "", err
	}
	result, err := history.DeriveLinkLog(ctx, entries, target, history.LoadSealFunc(r.LoadSeal))
	return result, target, err
}

func (r *Repository) DiffCurrent(ctx context.Context, ref string) (history.SealDiff, error) {
	if err := domain.ValidateREF(ref); err != nil {
		return history.SealDiff{}, err
	}
	head, err := r.refs.Resolve(ctx, ref)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("resolve current head for diff %s: %w", ref, err)
	}
	current, err := r.LoadSeal(ctx, head)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load current generation %s@%s: %w", ref, head, err)
	}
	if current.ParentRevision == nil {
		return history.SealDiff{}, fmt.Errorf("current seal %s@%s has no parent to compare; provide two explicit Seal selectors", ref, head)
	}
	return r.DiffExact(ctx, *current.ParentRevision, head)
}

func (r *Repository) DiffSelectors(ctx context.Context, fromSelector, toSelector string) (history.SealDiff, error) {
	from, err := r.ResolveSelector(ctx, fromSelector)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("resolve older diff selector: %w", err)
	}
	to, err := r.ResolveSelector(ctx, toSelector)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("resolve newer diff selector: %w", err)
	}
	return history.DiffSeals(from.ID, from.Payload, to.ID, to.Payload)
}

func (r *Repository) DiffExact(ctx context.Context, fromID, toID domain.ObjectID) (history.SealDiff, error) {
	from, err := r.LoadSeal(ctx, fromID)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load older diff generation %s: %w", fromID, err)
	}
	to, err := r.LoadSeal(ctx, toID)
	if err != nil {
		return history.SealDiff{}, fmt.Errorf("load newer diff generation %s: %w", toID, err)
	}
	return history.DiffSeals(fromID, from, toID, to)
}
